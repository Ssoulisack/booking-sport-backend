package servicetest

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/internal/core/services"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"testing"
)

const (
	business = 7
	mb       = 1 << 20
	maxBytes = 2 * mb // the upload limit
)

var roomyLimits = domain.PlanLimits{MaxSports: 10, MaxImages: 100, MaxStorageMB: 50, MaxMembers: 5}

// keys the server generates for business 7
var ownKeyPattern = regexp.MustCompile(`^businesses/7/sports/[0-9a-f-]{36}\.(png|jpg|webp)$`)

func newMediaService(store *fakeStorage, media *fakeMedia, limits domain.PlanLimits) input.MediaService {
	l := services.NewLimits(fakeBusinesses{limits: limits}, nil, nil, nil, nil, nil, nil, media)
	return services.NewMediaService(store, media, l)
}

// smallest valid headers; content type is detected from the leading bytes
var (
	pngBytes  = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	jpegBytes = append([]byte("\xff\xd8\xff\xe0"), make([]byte, 32)...)
	webpBytes = append([]byte("RIFF\x24\x00\x00\x00WEBPVP8 "), make([]byte, 32)...)
)

func TestMediaUploadAccepts(t *testing.T) {
	for name, data := range map[string][]byte{"png": pngBytes, "jpeg": jpegBytes, "webp": webpBytes} {
		t.Run(name, func(t *testing.T) {
			store, media := newFakeStorage(), newFakeMedia()
			res, err := newMediaService(store, media, roomyLimits).Upload(context.Background(), business, "sports", bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if !ownKeyPattern.MatchString(res.Key) {
				t.Fatalf("bad key %q", res.Key)
			}
			if _, ok := store.uploaded[res.Key]; !ok || res.URL != "https://files.test/"+res.Key {
				t.Fatalf("not stored / wrong url: %+v", res)
			}
			if rec, _ := media.Get(context.Background(), business, res.Key); rec == nil || rec.SizeBytes != int64(len(data)) {
				t.Fatalf("media record missing or wrong size: %+v", rec)
			}
		})
	}
}

func TestMediaUploadRejects(t *testing.T) {
	tooBig := append(append([]byte{}, pngBytes...), make([]byte, maxBytes)...)
	cases := map[string]struct {
		folder string
		data   []byte
		code   string
	}{
		"executable renamed to jpg": {"sports", []byte("MZ\x90\x00\x03\x00\x00\x00 not an image"), "UNSUPPORTED_FILE_TYPE"},
		"svg (script risk)":         {"sports", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), "UNSUPPORTED_FILE_TYPE"},
		"html":                      {"sports", []byte("<html><body>hi</body></html>"), "UNSUPPORTED_FILE_TYPE"},
		"empty":                     {"sports", nil, "FILE_IS_EMPTY"},
		"too large":                 {"sports", tooBig, "FILE_TOO_LARGE"},
		"unknown folder":            {"../etc", pngBytes, "INVALID_FOLDER"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			store := newFakeStorage()
			_, err := newMediaService(store, newFakeMedia(), roomyLimits).Upload(context.Background(), business, tc.folder, bytes.NewReader(tc.data))
			if err == nil || err.Error() != tc.code {
				t.Fatalf("got %v, want %s", err, tc.code)
			}
			if len(store.uploaded) != 0 {
				t.Fatal("rejected file must not reach storage")
			}
		})
	}
}

func TestMediaUploadPlanLimits(t *testing.T) {
	ctx := context.Background()

	t.Run("image count", func(t *testing.T) {
		svc := newMediaService(newFakeStorage(), newFakeMedia(), domain.PlanLimits{MaxImages: 1, MaxStorageMB: 50})
		if _, err := svc.Upload(ctx, business, "sports", bytes.NewReader(pngBytes)); err != nil {
			t.Fatal(err)
		}
		_, err := svc.Upload(ctx, business, "sports", bytes.NewReader(pngBytes))
		if err == nil || err.Error() != "PLAN_LIMIT_REACHED_IMAGES" {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("storage size", func(t *testing.T) {
		media := newFakeMedia()
		media.items["existing"] = domain.Media{BusinessID: business, Key: "existing", SizeBytes: mb} // 1 MB used
		svc := newMediaService(newFakeStorage(), media, domain.PlanLimits{MaxImages: 10, MaxStorageMB: 1})
		_, err := svc.Upload(ctx, business, "sports", bytes.NewReader(pngBytes))
		if err == nil || err.Error() != "PLAN_LIMIT_REACHED_STORAGE" {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("other businesses' files do not count", func(t *testing.T) {
		media := newFakeMedia()
		for i := range 5 {
			key := fmt.Sprintf("other-%d", i)
			media.items[key] = domain.Media{BusinessID: 99, Key: key, SizeBytes: 10}
		}
		svc := newMediaService(newFakeStorage(), media, domain.PlanLimits{MaxImages: 1, MaxStorageMB: 1})
		if _, err := svc.Upload(ctx, business, "sports", bytes.NewReader(pngBytes)); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
}

func TestMediaDeleteOnlyOwnKeys(t *testing.T) {
	ctx := context.Background()
	store, media := newFakeStorage(), newFakeMedia()
	svc := newMediaService(store, media, roomyLimits)
	own := "businesses/7/coaches/11111111-1111-1111-1111-111111111111.jpg"
	other := "businesses/8/coaches/11111111-1111-1111-1111-111111111111.jpg"
	media.items[own] = domain.Media{BusinessID: 7, Key: own}
	media.items[other] = domain.Media{BusinessID: 8, Key: other}

	for _, bad := range []string{"", "../secret.png", "businesses/7/../x.png", "businesses/7/other/11111111-1111-1111-1111-111111111111.png", "businesses/7/sports/not-a-uuid.png", other} {
		if err := svc.Delete(ctx, business, bad); err == nil {
			t.Fatalf("key %q must be rejected", bad)
		}
	}
	if len(store.deleted) != 0 {
		t.Fatal("nothing should have been deleted from storage yet")
	}
	if err := svc.Delete(ctx, business, own); err != nil || len(store.deleted) != 1 {
		t.Fatalf("own key rejected: %v", err)
	}
	if _, ok := media.items[other]; !ok {
		t.Fatal("another business's file must survive")
	}
}

// A sport may only point at an image this business uploaded (checked through the catalog service)
func TestSportImageMustBeOwnUpload(t *testing.T) {
	ctx := context.Background()
	media := newFakeMedia()
	sports := &fakeSports{owner: map[uint]uint{1: business}}
	limits := services.NewLimits(fakeBusinesses{limits: roomyLimits}, sports, nil, nil, nil, nil, nil, media)
	catalog := services.NewCatalogService(sports, nil, nil, newFakeStorage(), media, limits)

	own := "businesses/7/sports/22222222-2222-2222-2222-222222222222.png"
	media.items[own] = domain.Media{BusinessID: business, Key: own}
	notUploaded := "businesses/7/sports/33333333-3333-3333-3333-333333333333.png"
	otherBiz := "businesses/8/sports/22222222-2222-2222-2222-222222222222.png"

	for _, key := range []string{"", own} {
		if _, err := catalog.UpdateSport(ctx, business, 1, dto.SportRequest{Name: "Tennis", ImageKey: key}); err != nil {
			t.Fatalf("key %q rejected: %v", key, err)
		}
	}
	for _, bad := range []string{notUploaded, otherBiz, "https://evil.example/x.png"} {
		_, err := catalog.UpdateSport(ctx, business, 1, dto.SportRequest{Name: "Tennis", ImageKey: bad})
		if err == nil || err.Error() != "INVALID_FILE_KEY" {
			t.Fatalf("key %q: got %v, want INVALID_FILE_KEY", bad, err)
		}
	}
	if len(sports.updated) != 2 {
		t.Fatalf("only the two valid updates may be saved, got %d", len(sports.updated))
	}
}
