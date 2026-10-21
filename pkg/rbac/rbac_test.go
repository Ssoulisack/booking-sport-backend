package rbac

import (
	"backend/internal/config"
	"backend/internal/core/domain"
	"backend/pkg/jwt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func tokenFor(t *testing.T, roleID uint) string {
	t.Helper()
	pair, err := jwt.GenerateJWTToken(1, int(roleID))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Trim(string(pair.AccessToken), `"`)
}

func TestWithRoles(t *testing.T) {
	config.JWTGlobal = config.JWTConfig{Secret: "test-access-secret-0123456789abc", RefreshSecret: "test-refresh-secret-0123456789ab"}

	app := fiber.New()
	app.Get("/admin", AdminOnly, func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/staff", StaffUp, func(c *fiber.Ctx) error { return c.SendString("ok") })

	cases := []struct {
		name   string
		path   string
		header string
		want   int
	}{
		{"no header", "/admin", "", 401},
		{"short header does not panic", "/admin", "abc", 401},
		{"garbage token", "/admin", "Bearer garbage", 401},
		{"client on admin route", "/admin", "Bearer " + tokenFor(t, domain.RoleClient), 403},
		{"staff on admin route", "/admin", "Bearer " + tokenFor(t, domain.RoleStaff), 403},
		{"admin on admin route", "/admin", "Bearer " + tokenFor(t, domain.RoleAdmin), 200},
		{"staff on staff route", "/staff", "Bearer " + tokenFor(t, domain.RoleStaff), 200},
		{"client on staff route", "/staff", "Bearer " + tokenFor(t, domain.RoleClient), 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != tc.want {
				t.Fatalf("got %d, want %d", res.StatusCode, tc.want)
			}
		})
	}
}
