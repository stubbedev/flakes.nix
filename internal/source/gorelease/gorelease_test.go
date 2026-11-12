package gorelease

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stubbedev/flakes.nix/internal/fetch"
)

const releases = `[
  {"version": "go1.28rc1", "stable": false, "files": []},
  {"version": "go1.27.4", "stable": true, "files": [
    {"os": "linux", "arch": "amd64", "kind": "archive", "sha256": "7226736c2339719cda28fbf3a54d71aa63bcefe376e98cef00034268dab24840"},
    {"os": "linux", "arch": "amd64", "kind": "installer", "sha256": "00"},
    {"os": "darwin", "arch": "arm64", "kind": "archive", "sha256": "91a44a2b8a716600ad94a4f143e1561166"},
    {"os": "", "arch": "", "kind": "source", "sha256": "00"}
  ]},
  {"version": "go1.26.9", "stable": true, "files": []}
]`

func serve(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func newSource(t *testing.T, url string, platforms map[string]string) *Source {
	t.Helper()
	raw, err := json.Marshal(Settings{Platforms: platforms})
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(raw, fetch.Client{HTTP: http.DefaultClient}, url)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLatestPicksNewestStableArchive(t *testing.T) {
	s := newSource(t, serve(t, releases), map[string]string{"x86_64-linux": "linux-amd64"})
	got, err := s.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Version.String() != "1.27.4" {
		t.Errorf("version = %s, want 1.27.4", got.Version)
	}
	const want = "sha256-ciZzbCM5cZzaKPvzpU1xqmO87+N26YzvAANCaNqySEA="
	if got.Hashes["x86_64-linux"] != want || len(got.Hashes) != 1 {
		t.Errorf("hashes = %v, want only x86_64-linux = %s", got.Hashes, want)
	}
}

func TestLatestFailsOnMissingPlatform(t *testing.T) {
	s := newSource(t, serve(t, releases), map[string]string{"aarch64-linux": "linux-arm64"})
	if _, err := s.Latest(context.Background()); err == nil {
		t.Fatal("Latest succeeded, want error for missing platform")
	}
}

func TestLatestFailsOnMalformedHash(t *testing.T) {
	s := newSource(t, serve(t, releases), map[string]string{"aarch64-darwin": "darwin-arm64"})
	if _, err := s.Latest(context.Background()); err == nil {
		t.Fatal("Latest succeeded, want error for truncated sha256")
	}
}

func TestLatestFailsWithoutStableRelease(t *testing.T) {
	s := newSource(t, serve(t, `[{"version": "go1.28rc1", "stable": false, "files": []}]`), map[string]string{"x86_64-linux": "linux-amd64"})
	if _, err := s.Latest(context.Background()); err == nil {
		t.Fatal("Latest succeeded, want error")
	}
}

func TestNewRejectsInvalidSettings(t *testing.T) {
	for _, raw := range []string{`{}`, `{"platforms":{}}`, `{"platforms":{"x86_64-linux":"Linux amd64"}}`, `{"platforms":{"x":"linux-amd64"},"extra":1}`} {
		if _, err := New(json.RawMessage(raw), fetch.New(), DownloadsURL); err == nil {
			t.Errorf("New(%s) succeeded, want error", raw)
		}
	}
}
