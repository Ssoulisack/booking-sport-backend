package services

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

type fakeStorage struct {
	uploaded map[string][]byte
	deleted  []string
}

func newFakeStorage() *fakeStorage { return &fakeStorage{uploaded: map[string][]byte{}} }

func (f *fakeStorage) Upload(_ context.Context, key, _ string, body io.Reader) error {
	b, _ := io.ReadAll(body)
	f.uploaded[key] = b
	return nil
}
func (f *fakeStorage) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}
func (f *fakeStorage) URL(key string) string { return "https://files.test/" + key }

// smallest valid headers; DetectContentType only reads the leading bytes
var (
	pngBytes  = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	jpegBytes = append([]byte("\xff\xd8\xff\xe0"), make([]byte, 32)...)
	webpBytes = append([]byte("RIFF\x24\x00\x00\x00WEBPVP8 "), make([]byte, 32)...)
)

func TestMediaUploadAccepts(t *testing.T) {
	for name, data := range map[string][]byte{"png": pngBytes, "jpeg": jpegBytes, "webp": webpBytes} {
		t.Run(name, func(t *testing.T) {
			store := newFakeStorage()
			res, err := NewMediaService(store).Upload(context.Background(), "sports", bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if !isValidMediaKey(res.Key) || !strings.HasPrefix(res.Key, "sports/") {
				t.Fatalf("bad key %q", res.Key)
			}
			if _, ok := store.uploaded[res.Key]; !ok || res.URL != "https://files.test/"+res.Key {
				t.Fatalf("not stored / wrong url: %+v", res)
			}
		})
	}
}

func TestMediaUploadRejects(t *testing.T) {
	tooBig := append(append([]byte{}, pngBytes...), make([]byte, maxUploadBytes)...)
	cases := map[string]struct {
		folder string
		data   []byte
	}{
		"executable renamed to jpg": {"sports", []byte("MZ\x90\x00\x03\x00\x00\x00 not an image")},
		"svg (script risk)":         {"sports", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)},
		"html":                      {"sports", []byte("<html><body>hi</body></html>")},
		"empty":                     {"sports", nil},
		"too large":                 {"sports", tooBig},
		"unknown folder":            {"../etc", pngBytes},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			store := newFakeStorage()
			if _, err := NewMediaService(store).Upload(context.Background(), tc.folder, bytes.NewReader(tc.data)); err == nil {
				t.Fatal("expected rejection")
			}
			if len(store.uploaded) != 0 {
				t.Fatal("rejected file must not reach storage")
			}
		})
	}
}

func TestMediaDeleteOnlyServerKeys(t *testing.T) {
	store := newFakeStorage()
	svc := NewMediaService(store)
	for _, bad := range []string{"", "../secret.png", "sports/../x.png", "other/11111111-1111-1111-1111-111111111111.png", "sports/not-a-uuid.png"} {
		if err := svc.Delete(context.Background(), bad); err == nil {
			t.Fatalf("key %q must be rejected", bad)
		}
	}
	good := "coaches/11111111-1111-1111-1111-111111111111.jpg"
	if err := svc.Delete(context.Background(), good); err != nil || len(store.deleted) != 1 {
		t.Fatalf("valid key rejected: %v", err)
	}
}
