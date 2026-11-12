package pin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stubbedev/flakes.nix/internal/version"
)

func TestSHA256FromHex(t *testing.T) {
	got, err := SHA256FromHex("7226736c2339719cda28fbf3a54d71aa63bcefe376e98cef00034268dab24840")
	if err != nil {
		t.Fatal(err)
	}
	const want = "sha256-ciZzbCM5cZzaKPvzpU1xqmO87+N26YzvAANCaNqySEA="
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestSHA256FromHexRejectsInvalid(t *testing.T) {
	for _, s := range []string{"", "zz", "abcd"} {
		if _, err := SHA256FromHex(s); err == nil {
			t.Errorf("SHA256FromHex(%q) succeeded, want error", s)
		}
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hashes.json")
	v, err := version.Parse("1.27.4")
	if err != nil {
		t.Fatal(err)
	}
	want := Pin{Version: v, Hashes: map[string]SRI{
		"x86_64-linux":   "sha256-ciZzbCM5cZzaKPvzpU1xqmO87+N26YzvAANCaNqySEA=",
		"aarch64-darwin": "sha256-kaRKK4pxZgCtlKTxQ+FWEWYg24goCGpuNgIpG+m9C7Q=",
	}}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const wantJSON = `{
  "version": "1.27.4",
  "hashes": {
    "aarch64-darwin": "sha256-kaRKK4pxZgCtlKTxQ+FWEWYg24goCGpuNgIpG+m9C7Q=",
    "x86_64-linux": "sha256-ciZzbCM5cZzaKPvzpU1xqmO87+N26YzvAANCaNqySEA="
  }
}
`
	if string(data) != wantJSON {
		t.Errorf("saved JSON:\n%s\nwant:\n%s", data, wantJSON)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version.Compare(want.Version) != 0 || len(got.Hashes) != len(want.Hashes) {
		t.Fatalf("round trip: got %+v, want %+v", got, want)
	}
	for system, h := range want.Hashes {
		if got.Hashes[system] != h {
			t.Errorf("system %s: got %s, want %s", system, got.Hashes[system], h)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("directory has %d entries, want only hashes.json", len(entries))
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hashes.json")
	if err := os.WriteFile(path, []byte(`{"version":"1","hashes":{},"extra":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("Load succeeded, want error for unknown field")
	}
}

func TestLoadRejectsBadHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hashes.json")
	if err := os.WriteFile(path, []byte(`{"version":"1","hashes":{"x86_64-linux":"sha256-nope"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("Load succeeded, want error for malformed hash")
	}
}
