// Package e2e runs the whole API in-process against a real local Postgres and Mailpit.
//
//	docker compose up -d postgres redis mailpit
//	docker exec postgres_db psql -U booking -c "CREATE DATABASE booking_test"   # once
//	E2E=1 go test ./test/e2e -count=1 -v
//
// The test database is wiped at the start, so the test refuses any database that is not
// local and whose name does not end in "_test".
package e2e

import (
	"backend/cmd/application"
	"backend/internal/adapters/secondary/mailer"
	"backend/internal/adapters/secondary/minio"
	"backend/internal/adapters/secondary/postgres"
	"backend/internal/adapters/secondary/redis"
	"backend/internal/adapters/secondary/storage"
	"backend/internal/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var mailpitAPI = env("E2E_MAILPIT_API", "http://localhost:8025/api/v1")

// ---------- setup ----------

func newApp(t *testing.T) *fiber.App {
	t.Helper()
	if os.Getenv("E2E") == "" {
		t.Skip("set E2E=1 to run (needs local Postgres, Redis and Mailpit)")
	}
	pg := postgres.Config{
		Host: env("E2E_DB_HOST", "localhost"), Port: env("E2E_DB_PORT", "5432"),
		User: env("E2E_DB_USER", "booking"), Password: env("E2E_DB_PASSWORD", "booking"),
		DBName: env("E2E_DB_NAME", "booking_test"), SSLMode: "disable", TimeZone: "Asia/Vientiane",
	}
	if (pg.Host != "localhost" && pg.Host != "127.0.0.1") || !strings.HasSuffix(pg.DBName, "_test") {
		t.Fatalf("refusing to wipe %s/%s: only a local *_test database is allowed", pg.Host, pg.DBName)
	}
	resetDatabase(t, pg)
	resetMailpit(t)

	files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(files.Close)

	config.JWTGlobal = config.JWTConfig{Secret: "e2e-access-secret-0123456789abcd", RefreshSecret: "e2e-refresh-secret-0123456789abc"}
	cfg := &config.Config{
		App:      config.AppConfig{Name: "Sport Booking E2E", FrontendURL: "http://frontend.test"},
		Postgres: pg,
		Redis:    redis.Config{Host: env("E2E_REDIS_HOST", "localhost"), Port: "6379"},
		Mailer:   mailer.Config{Host: env("E2E_SMTP_HOST", "localhost"), Port: 1025, From: "no-reply@booking.test"},
		Minio:    minio.Config{Endpoint: "localhost:9000"}, // template adapter exits on an empty endpoint; unused
		Storage:  storage.Config{ProjectURL: files.URL, Bucket: "media", ServiceKey: "fake"},
	}
	repos := application.SetupRepositories(cfg)
	t.Cleanup(func() { application.CloseRepositories(repos) })

	app := fiber.New()
	application.SetupRoutes(app, application.SetupHandlers(application.SetupServices(repos)))
	return app
}

func resetDatabase(t *testing.T, pg postgres.Config) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		pg.Host, pg.Port, pg.User, pg.Password, pg.DBName)
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	if err := db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;").Error; err != nil {
		t.Fatalf("reset test database: %v", err)
	}
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}

// ---------- HTTP client ----------

type api struct {
	t   *testing.T
	app *fiber.App
}

type reply struct {
	status int
	body   map[string]any
}

func (r reply) msg() string { s, _ := r.body["message"].(string); return s }

// str reads a dotted path from the body, e.g. "data.user.email"; a number picks a list item ("data.0.id")
func (r reply) str(path string) string {
	var cur any = r.body
	for _, key := range strings.Split(path, ".") {
		switch c := cur.(type) {
		case map[string]any:
			cur = c[key]
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(c) {
				return ""
			}
			cur = c[i]
		default:
			return ""
		}
	}
	switch v := cur.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case nil:
		return ""
	}
	return fmt.Sprint(cur)
}

func (a api) send(req *http.Request, token string) reply {
	a.t.Helper()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := a.app.Test(req, -1) // no timeout: bcrypt is deliberately slow
	if err != nil {
		a.t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := reply{status: res.StatusCode, body: map[string]any{}}
	_ = json.Unmarshal(raw, &out.body)
	return out
}

func (a api) call(method, path, token string, body any) reply {
	a.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, "/api/v1"+path, rd)
	req.Header.Set("Content-Type", "application/json")
	return a.send(req, token)
}

// expect fails the test unless the call returns want (and wantMsg, when given)
func (a api) expect(want int, wantMsg string, method, path, token string, body any) reply {
	a.t.Helper()
	r := a.call(method, path, token, body)
	if r.status != want || (wantMsg != "" && r.msg() != wantMsg) {
		a.t.Fatalf("%s %s: got %d %q, want %d %q (body %v)", method, path, r.status, r.msg(), want, wantMsg, r.body)
	}
	return r
}

func (a api) upload(path, token, folder string, data []byte) reply {
	a.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "photo.png")
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1"+path+"?folder="+folder, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return a.send(req, token)
}

// ---------- Mailpit ----------

func resetMailpit(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, mailpitAPI+"/messages", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Mailpit not reachable at %s: %v", mailpitAPI, err)
	}
	res.Body.Close()
}

// linkFor returns the token from the newest email to `to` whose link contains pathPart
func linkFor(t *testing.T, to, pathPart string) string {
	t.Helper()
	re := regexp.MustCompile(regexp.QuoteMeta(pathPart) + `([A-Za-z0-9_-]+)`)
	for range 20 { // the email is sent synchronously, but give Mailpit a moment to index it
		var list struct {
			Messages []struct{ ID string } `json:"messages"`
		}
		getJSON(t, mailpitAPI+"/search?query=to:"+to, &list)
		for _, m := range list.Messages {
			var msg struct{ Text string }
			getJSON(t, mailpitAPI+"/message/"+m.ID, &msg)
			if found := re.FindStringSubmatch(msg.Text); found != nil {
				return found[1]
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no email to %s with a %s link", to, pathPart)
	return ""
}

func getJSON(t *testing.T, url string, out any) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)

// ---------- the scenario ----------

func TestPlatformWorkflow(t *testing.T) {
	a := api{t: t, app: newApp(t)}
	const pw = "password123"

	// Super Admin (seeded) logs in and finds the Free plan
	admin := a.expect(200, "", "POST", "/auth/login", "", map[string]any{"email": "admin@booking.local", "password": "ChangeMe123!"}).str("data.access_token")
	plans := a.expect(200, "", "GET", "/platform/plans", admin, nil)
	freePlan := plans.body["data"].([]any)[0].(map[string]any)
	if freePlan["name"] != "Free" {
		t.Fatalf("first plan should be Free, got %v", freePlan["name"])
	}
	freeID := freePlan["id"]

	// a customer is not a Super Admin
	customer := a.expect(201, "", "POST", "/auth/register", "", map[string]any{"name": "Cus Tomer", "email": "customer@test.la", "password": pw, "nationality": "TH"}).str("data.access_token")
	a.expect(403, "", "GET", "/platform/businesses", customer, nil)

	// --- create two businesses ---
	smash := a.expect(201, "", "POST", "/platform/businesses", admin, map[string]any{"name": "Smash Tennis", "plan_id": freeID})
	if smash.str("data.slug") != "smash-tennis" || smash.str("data.status") != "draft" {
		t.Fatalf("unexpected business: %v", smash.body)
	}
	smashID := smash.str("data.id")
	otherID := a.expect(201, "", "POST", "/platform/businesses", admin, map[string]any{"name": "Other Club", "plan_id": freeID}).str("data.id")
	a.expect(409, "SLUG_TAKEN", "POST", "/platform/businesses", admin, map[string]any{"name": "X", "slug": "smash-tennis", "plan_id": freeID})
	m := "/manage/businesses/" + smashID

	// --- owner invite by email, accepted as a new account ---
	a.expect(201, "", "POST", "/platform/businesses/"+smashID+"/owner-invitation", admin, map[string]any{"email": "owner@smash.test"})
	a.expect(409, "OWNER_INVITATION_ALREADY_PENDING", "POST", "/platform/businesses/"+smashID+"/owner-invitation", admin, map[string]any{"email": "x@smash.test"})
	tok := linkFor(t, "owner@smash.test", "/invite/")
	info := a.expect(200, "", "GET", "/invitations/"+tok, "", nil)
	if info.str("data.role") != "owner" || info.str("data.account_exists") != "false" || info.str("data.business_name") != "Smash Tennis" {
		t.Fatalf("unexpected invitation info: %v", info.body)
	}
	owner := a.expect(200, "", "POST", "/invitations/"+tok+"/accept", "", map[string]any{"name": "Olivia Owner", "password": pw}).str("data.access_token")
	a.expect(404, "INVITATION_NOT_FOUND_OR_EXPIRED", "GET", "/invitations/"+tok, "", nil) // single use
	a.expect(404, "INVITATION_NOT_FOUND_OR_EXPIRED", "GET", "/invitations/not-a-real-token", "", nil)
	mine := a.expect(200, "", "GET", "/me/businesses", owner, nil)
	if fmt.Sprint(mine.body["data"]) == "[]" || !strings.Contains(fmt.Sprint(mine.body["data"]), "owner") {
		t.Fatalf("owner should see the business in the switcher: %v", mine.body)
	}

	// --- team size: Free allows 2 members; Super Admin raises it to 3 for this business ---
	a.expect(201, "", "POST", m+"/invitations", owner, map[string]any{"email": "manager@smash.test", "role": "manager"})
	a.expect(409, "PLAN_LIMIT_REACHED_MEMBERS", "POST", m+"/invitations", owner, map[string]any{"email": "staff@smash.test", "role": "staff"})
	a.expect(200, "", "PUT", "/platform/businesses/"+smashID, admin, map[string]any{"name": "Smash Tennis", "plan_id": freeID, "overrides": map[string]any{"max_members": 3}})
	a.expect(201, "", "POST", m+"/invitations", owner, map[string]any{"email": "staff@smash.test", "role": "staff"})
	a.expect(409, "PLAN_LIMIT_REACHED_MEMBERS", "POST", m+"/invitations", owner, map[string]any{"email": "fourth@smash.test", "role": "staff"})
	a.expect(400, "INVALID_ROLE", "POST", m+"/invitations", owner, map[string]any{"email": "boss@smash.test", "role": "owner"})

	manager := a.expect(200, "", "POST", "/invitations/"+linkFor(t, "manager@smash.test", "/invite/")+"/accept", "", map[string]any{"name": "Max Manager", "password": pw}).str("data.access_token")
	staff := a.expect(200, "", "POST", "/invitations/"+linkFor(t, "staff@smash.test", "/invite/")+"/accept", "", map[string]any{"name": "Sam Staff", "password": pw}).str("data.access_token")

	// --- manager sets the business up; staff can read but not change ---
	sportID := a.expect(201, "", "POST", m+"/sports", manager, map[string]any{"name": "Tennis"}).str("data.id")
	courtID := a.expect(201, "", "POST", m+"/resources", manager, map[string]any{"sport_id": num(sportID), "name": "Court 1", "kind": "court"}).str("data.id")
	week := []map[string]any{}
	for d := range 7 {
		week = append(week, map[string]any{"weekday": d, "open": "08:00", "close": "22:00"})
	}
	a.expect(400, "PUBLISH_NEEDS_ACTIVE_RESOURCE_WITH_HOURS_AND_PRICE", "POST", m+"/publish", owner, nil)
	a.expect(200, "", "PUT", m+"/opening-hours/resource/"+courtID, manager, map[string]any{"hours": week})
	a.expect(200, "", "PUT", m+"/prices/resource/"+courtID, manager, map[string]any{"price_per_hour": 200, "currency": "LAK"})
	a.expect(200, "", "PUT", m+"/rules/sport/"+sportID, manager, map[string]any{"max_duration_min": 240})
	if e := a.expect(200, "", "GET", m+"/rules/effective?resource_id="+courtID, staff, nil); e.str("data.max_duration_min") != "240" {
		t.Fatalf("effective rule: %v", e.body)
	}
	a.expect(200, "", "GET", m+"/sports", staff, nil)
	a.expect(403, "ROLE_NOT_ALLOWED", "PUT", m+"/prices/resource/"+courtID, staff, map[string]any{"price_per_hour": 1, "currency": "LAK"})
	a.expect(403, "ROLE_NOT_ALLOWED", "POST", m+"/sports", staff, map[string]any{"name": "Boxing"})
	a.expect(403, "ROLE_NOT_ALLOWED", "POST", m+"/invitations", manager, map[string]any{"email": "x@smash.test", "role": "staff"})
	a.expect(403, "ROLE_NOT_ALLOWED", "POST", m+"/publish", manager, nil)

	// --- catalog limits: Free allows 3 resources ---
	a.expect(201, "", "POST", m+"/resources", manager, map[string]any{"sport_id": num(sportID), "name": "Court 2", "kind": "court"})
	a.expect(201, "", "POST", m+"/resources", manager, map[string]any{"sport_id": num(sportID), "name": "Court 3", "kind": "court"})
	a.expect(409, "PLAN_LIMIT_REACHED_RESOURCES", "POST", m+"/resources", manager, map[string]any{"sport_id": num(sportID), "name": "Court 4", "kind": "court"})
	if u := a.expect(200, "", "GET", m+"/usage", manager, nil); u.str("data.resources.used") != "3" || u.str("data.members.used") != "3" {
		t.Fatalf("usage: %v", u.body)
	}

	// --- images: own upload accepted, another business's key rejected ---
	img := a.upload(m+"/uploads", manager, "sports", pngBytes)
	if img.status != 201 || !strings.HasPrefix(img.str("data.key"), "businesses/"+smashID+"/sports/") {
		t.Fatalf("upload: %d %v", img.status, img.body)
	}
	a.expect(200, "", "PUT", m+"/sports/"+sportID, manager, map[string]any{"name": "Tennis", "image_key": img.str("data.key")})
	foreignKey := "businesses/" + otherID + "/sports/11111111-1111-1111-1111-111111111111.png"
	a.expect(400, "INVALID_FILE_KEY", "PUT", m+"/sports/"+sportID, manager, map[string]any{"name": "Tennis", "image_key": foreignKey})

	// --- publish: public page appears; drafts and unknown slugs stay hidden ---
	a.expect(404, "BUSINESS_NOT_FOUND", "GET", "/public/smash-tennis/", "", nil)
	if p := a.expect(200, "", "POST", m+"/publish", owner, nil); p.str("data.status") != "published" || p.str("data.public_path") != "/b/smash-tennis" {
		t.Fatalf("publish: %v", p.body)
	}
	if pub := a.expect(200, "", "GET", "/public/smash-tennis/", "", nil); pub.str("data.name") != "Smash Tennis" {
		t.Fatalf("public page: %v", pub.body)
	}
	a.expect(200, "", "GET", "/public/smash-tennis/resources?sport_id="+sportID, "", nil)
	a.expect(404, "BUSINESS_NOT_FOUND", "GET", "/public/other-club/", "", nil)
	a.expect(404, "BUSINESS_NOT_FOUND", "GET", "/public/nope/", "", nil)
	settings := a.expect(200, "", "GET", m+"/settings", owner, nil)
	body := map[string]any{}
	for _, k := range []string{"name", "timezone", "currency", "language", "booking_mode"} {
		body[k] = settings.str("data." + k)
	}
	body["slug"] = "new-slug"
	a.expect(409, "SLUG_LOCKED_AFTER_PUBLISH", "PUT", m+"/settings", owner, body)

	// --- cross-tenant: the other business's owner sees nothing of Smash Tennis ---
	a.expect(201, "", "POST", "/platform/businesses/"+otherID+"/owner-invitation", admin, map[string]any{"email": "owner@other.test"})
	intruder := a.expect(200, "", "POST", "/invitations/"+linkFor(t, "owner@other.test", "/invite/")+"/accept", "", map[string]any{"name": "Ivan Intruder", "password": pw}).str("data.access_token")
	for _, probe := range []struct{ method, path string }{
		{"GET", "/settings"}, {"PUT", "/settings"}, {"POST", "/publish"}, {"GET", "/usage"},
		{"GET", "/members"}, {"GET", "/invitations"}, {"POST", "/invitations"},
		{"GET", "/sports"}, {"GET", "/sports/" + sportID}, {"PUT", "/sports/" + sportID}, {"DELETE", "/sports/" + sportID},
		{"GET", "/resources/" + courtID}, {"DELETE", "/resources/" + courtID},
		{"GET", "/opening-hours/resource/" + courtID}, {"PUT", "/prices/resource/" + courtID},
		{"GET", "/rules"}, {"GET", "/rules/effective?resource_id=" + courtID}, {"POST", "/uploads"},
	} {
		a.expect(404, "BUSINESS_NOT_FOUND", probe.method, m+probe.path, intruder, map[string]any{})
	}
	// ...and using its own business id with Smash Tennis's records finds nothing either
	om := "/manage/businesses/" + otherID
	a.expect(404, "", "GET", om+"/sports/"+sportID, intruder, nil)
	a.expect(400, "OWNER_NOT_FOUND", "GET", om+"/opening-hours/resource/"+courtID, intruder, nil)
	a.expect(400, "SPORT_NOT_FOUND", "POST", om+"/resources", intruder, map[string]any{"sport_id": num(sportID), "name": "Stolen", "kind": "court"})
	a.expect(404, "BUSINESS_NOT_FOUND", "GET", m+"/sports", customer, nil)
	a.expect(401, "", "GET", m+"/sports", "", nil)

	// --- disabling takes effect on the very next request ---
	staffID := ""
	for _, raw := range a.expect(200, "", "GET", m+"/members", owner, nil).body["data"].([]any) {
		if mem := raw.(map[string]any); mem["email"] == "staff@smash.test" {
			staffID = fmt.Sprintf("%.0f", mem["user_id"])
		}
	}
	a.expect(200, "", "PATCH", m+"/members/"+staffID, owner, map[string]any{"active": false})
	a.expect(403, "MEMBERSHIP_DISABLED", "GET", m+"/sports", staff, nil)
	a.expect(200, "", "PATCH", m+"/members/"+staffID, owner, map[string]any{"active": true})
	a.expect(200, "", "GET", m+"/sports", staff, nil)
	a.expect(200, "", "PATCH", "/platform/users/"+staffID+"/status", admin, map[string]any{"status": false})
	a.expect(403, "ACCOUNT_DISABLED", "GET", m+"/sports", staff, nil)
	a.expect(200, "", "PATCH", "/platform/users/"+staffID+"/status", admin, map[string]any{"status": true})
	a.expect(400, "OWNER_CANNOT_BE_REMOVED", "DELETE", m+"/members/"+a.expect(200, "", "GET", "/auth/me", owner, nil).str("data.id"), owner, nil)

	// --- Super Admin support view: read-only and logged once per window ---
	a.expect(200, "", "GET", m+"/sports", admin, nil)
	a.expect(200, "", "GET", m+"/settings", admin, nil)
	a.expect(403, "SUPPORT_VIEW_IS_READ_ONLY", "PUT", m+"/prices/resource/"+courtID, admin, map[string]any{"price_per_hour": 1, "currency": "LAK"})
	if v := a.expect(200, "", "GET", m+"/support-visits", owner, nil); v.str("meta.total") != "1" {
		t.Fatalf("support visits: %v", v.body)
	}

	// --- suspension hides the page and locks members out, without losing data ---
	a.expect(200, "", "POST", "/platform/businesses/"+smashID+"/suspend", admin, nil)
	a.expect(404, "BUSINESS_NOT_FOUND", "GET", "/public/smash-tennis/", "", nil)
	a.expect(403, "BUSINESS_SUSPENDED", "GET", m+"/sports", owner, nil)
	a.expect(200, "", "POST", "/platform/businesses/"+smashID+"/reactivate", admin, nil)
	a.expect(200, "", "GET", "/public/smash-tennis/", "", nil)

	// --- ownership transfer ---
	managerID := a.expect(200, "", "GET", "/auth/me", manager, nil).str("data.id")
	a.expect(200, "", "POST", m+"/transfer-ownership", owner, map[string]any{"user_id": num(managerID)})
	a.expect(200, "", "POST", m+"/unpublish", manager, nil)             // the manager is now the owner
	a.expect(403, "ROLE_NOT_ALLOWED", "POST", m+"/publish", owner, nil) // the old owner is now a manager

	// --- forgot / reset password ---
	a.expect(200, "IF_THE_ACCOUNT_EXISTS_AN_EMAIL_WAS_SENT", "POST", "/auth/forgot-password", "", map[string]any{"email": "nobody@test.la"})
	a.expect(200, "IF_THE_ACCOUNT_EXISTS_AN_EMAIL_WAS_SENT", "POST", "/auth/forgot-password", "", map[string]any{"email": "Customer@Test.la"})
	resetTok := linkFor(t, "customer@test.la", "/reset-password/")
	time.Sleep(1100 * time.Millisecond) // tokens carry whole seconds; make the old token clearly older
	a.expect(400, "PASSWORD_TOO_SHORT", "POST", "/auth/reset-password", "", map[string]any{"token": resetTok, "password": "short"})
	a.expect(200, "PASSWORD_UPDATED", "POST", "/auth/reset-password", "", map[string]any{"token": resetTok, "password": "new-password-456"})
	a.expect(400, "INVALID_OR_EXPIRED_TOKEN", "POST", "/auth/reset-password", "", map[string]any{"token": resetTok, "password": "another-pass-789"})
	a.expect(401, "TOKEN_REVOKED", "GET", "/auth/me", customer, nil)
	a.expect(401, "INVALID_CREDENTIALS", "POST", "/auth/login", "", map[string]any{"email": "customer@test.la", "password": pw})
	a.expect(200, "", "POST", "/auth/login", "", map[string]any{"email": "customer@test.la", "password": "new-password-456"})

	// --- rate limit: forgot-password allows 5 per 15 minutes per client (2 used above) ---
	for range 3 {
		a.expect(200, "", "POST", "/auth/forgot-password", "", map[string]any{"email": "nobody@test.la"})
	}
	a.expect(429, "RATE_LIMIT_EXCEEDED", "POST", "/auth/forgot-password", "", map[string]any{"email": "nobody@test.la"})
}

// num turns an id read from a reply back into a JSON number for request bodies
func num(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic("not a number: " + s)
	}
	return n
}

// raw returns the response body as text (for CSV downloads)
func (a api) raw(method, path, token string) string {
	a.t.Helper()
	req := httptest.NewRequest(method, "/api/v1"+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := a.app.Test(req, -1)
	if err != nil {
		a.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return string(b)
}

// expectEmail fails unless an email to `to` contains every one of the given texts
func expectEmail(t *testing.T, to string, texts ...string) {
	t.Helper()
	for range 20 {
		var list struct {
			Messages []struct{ ID string } `json:"messages"`
		}
		getJSON(t, mailpitAPI+"/search?query=to:"+to, &list)
		for _, m := range list.Messages {
			var msg struct{ Text string }
			getJSON(t, mailpitAPI+"/message/"+m.ID, &msg)
			all := true
			for _, want := range texts {
				all = all && strings.Contains(msg.Text, want)
			}
			if all {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no email to %s containing %q", to, texts)
}
