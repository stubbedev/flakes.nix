package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stubbedev/flakes.nix/internal/fetch"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadPackagesRejectsUnknownSource(t *testing.T) {
	path := writeConfig(t, `{"packages":[{"name":"x","pin":"p","source":{"type":"nope","settings":{}}}]}`)
	_, err := loadPackages(path, fetch.New())
	if err == nil || !strings.Contains(err.Error(), `unknown source type "nope"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadPackagesRejectsUnknownFields(t *testing.T) {
	path := writeConfig(t, `{"packages":[],"extra":true}`)
	if _, err := loadPackages(path, fetch.New()); err == nil {
		t.Fatal("want error")
	}
}

func TestLoadPackagesBuildsGoRelease(t *testing.T) {
	path := writeConfig(t, `{"packages":[{"name":"go","pin":"packages/go/hashes.json","source":{"type":"go-release","settings":{"platforms":{"x86_64-linux":"linux-amd64"}}}}]}`)
	pkgs, err := loadPackages(path, fetch.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].Name != "go" || pkgs[0].PinPath != "packages/go/hashes.json" {
		t.Fatalf("packages = %+v", pkgs)
	}
}

func TestRunRequiresRepositoryRoot(t *testing.T) {
	path := writeConfig(t, `{"packages":[]}`)
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"-config", path, "-root", t.TempDir()}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "not the repository root") {
		t.Fatalf("err = %v", err)
	}
}
