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
  {"version": "go1.28.1", "stable": true, "files": [
    {"os": "", "arch": "", "kind": "source", "sha256": "91a44a2b8a716600ad94a4f143e156116620db8828086a6e3602291be9bd0bb4"}
  ]},
  {"version": "go1.27.4", "stable": true, "files": [
    {"os": "", "arch": "", "kind": "source", "sha256": "7226736c2339719cda28fbf3a54d71aa63bcefe376e98cef00034268dab24840"},
    {"os": "linux", "arch": "amd64", "kind": "archive", "sha256": "91a44a2b8a716600ad94a4f143e156116620db8828086a6e3602291be9bd0bb4"},
    {"os": "darwin", "arch": "arm64", "kind": "archive", "sha256": "91a44a2b"}
  ]},
  {"version": "go1.27.3", "stable": true, "files": []}
]`

const hashA = "sha256-ciZzbCM5cZzaKPvzpU1xqmO87+N26YzvAANCaNqySEA="

func serve(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func newSource(t *testing.T, url string, s Settings) *Source {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	src, err := New(raw, fetch.Client{HTTP: http.DefaultClient}, url)
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func TestLatestStaysInSeries(t *testing.T) {
	s := newSource(t, serve(t, releases), Settings{Series: "1.27", Files: map[string]Selector{"source": {Kind: "source"}}})
	got, err := s.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Version.String() != "1.27.4" {
		t.Errorf("version = %s, want 1.27.4", got.Version)
	}
	if got.Hashes["source"] != hashA || len(got.Hashes) != 1 {
		t.Errorf("hashes = %v, want only source = %s", got.Hashes, hashA)
	}
}

func TestLatestSelectsByKindOSArch(t *testing.T) {
	s := newSource(t, serve(t, releases), Settings{Series: "1.27", Files: map[string]Selector{"x86_64-linux": {Kind: "archive", OS: "linux", Arch: "amd64"}}})
	got, err := s.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Hashes["x86_64-linux"] != "sha256-kaRKK4pxZgCtlKTxQ+FWEWYg24goCGpuNgIpG+m9C7Q=" {
		t.Errorf("hashes = %v", got.Hashes)
	}
}

func TestLatestFailsOnMissingFile(t *testing.T) {
	s := newSource(t, serve(t, releases), Settings{Series: "1.27", Files: map[string]Selector{"aarch64-linux": {Kind: "archive", OS: "linux", Arch: "arm64"}}})
	if _, err := s.Latest(context.Background()); err == nil {
		t.Fatal("Latest succeeded, want error for missing file")
	}
}

func TestLatestFailsOnMalformedHash(t *testing.T) {
	s := newSource(t, serve(t, releases), Settings{Series: "1.27", Files: map[string]Selector{"aarch64-darwin": {Kind: "archive", OS: "darwin", Arch: "arm64"}}})
	if _, err := s.Latest(context.Background()); err == nil {
		t.Fatal("Latest succeeded, want error for truncated sha256")
	}
}

func TestLatestFailsWithoutReleaseInSeries(t *testing.T) {
	s := newSource(t, serve(t, releases), Settings{Series: "1.29", Files: map[string]Selector{"source": {Kind: "source"}}})
	if _, err := s.Latest(context.Background()); err == nil {
		t.Fatal("Latest succeeded, want error")
	}
}

func TestNewRejectsInvalidSettings(t *testing.T) {
	for _, raw := range []string{
		`{}`,
		`{"series":"1.27"}`,
		`{"series":"1.27","files":{}}`,
		`{"series":"1.27","files":{"source":{}}}`,
		`{"series":"latest","files":{"source":{"kind":"source"}}}`,
		`{"series":"1.27","files":{"source":{"kind":"source"}},"extra":1}`,
	} {
		if _, err := New(json.RawMessage(raw), fetch.New(), DownloadsURL); err == nil {
			t.Errorf("New(%s) succeeded, want error", raw)
		}
	}
}
