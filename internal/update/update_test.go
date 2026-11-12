package update

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stubbedev/flakes.nix/internal/pin"
	"github.com/stubbedev/flakes.nix/internal/version"
)

const (
	hashA = pin.SRI("sha256-ciZzbCM5cZzaKPvzpU1xqmO87+N26YzvAANCaNqySEA=")
	hashB = pin.SRI("sha256-kaRKK4pxZgCtlKTxQ+FWEWYg24goCGpuNgIpG+m9C7Q=")
)

type fixed struct {
	pin pin.Pin
	err error
}

func (f fixed) Latest(context.Context) (pin.Pin, error) {
	return f.pin, f.err
}

func mustPin(t *testing.T, v string, hashes map[string]pin.SRI) pin.Pin {
	t.Helper()
	parsed, err := version.Parse(v)
	if err != nil {
		t.Fatal(err)
	}
	return pin.Pin{Version: parsed, Hashes: hashes}
}

func setup(t *testing.T, current *pin.Pin) (string, string) {
	t.Helper()
	root := t.TempDir()
	rel := filepath.Join("packages", "tool", "hashes.json")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	switch {
	case current == nil:
	case len(current.Hashes) == 0:
		seed := `{"version": "` + current.Version.String() + `", "hashes": {}}`
		if err := os.WriteFile(filepath.Join(root, rel), []byte(seed), 0o644); err != nil {
			t.Fatal(err)
		}
	default:
		if err := pin.Save(filepath.Join(root, rel), *current); err != nil {
			t.Fatal(err)
		}
	}
	return root, rel
}

func run(t *testing.T, current *pin.Pin, latest Source) (Result, pin.Pin, string) {
	t.Helper()
	root, rel := setup(t, current)
	res := Run(context.Background(), root, []Package{{Name: "tool", PinPath: rel, Source: latest}})
	var saved pin.Pin
	if p, err := pin.Load(filepath.Join(root, rel)); err == nil {
		saved = p
	}
	return res, saved, filepath.Join(root, rel)
}

func TestRunBumpsNewerVersion(t *testing.T) {
	cur := mustPin(t, "1.27.3", map[string]pin.SRI{"x86_64-linux": hashB})
	res, saved, _ := run(t, &cur, fixed{pin: mustPin(t, "1.27.4", map[string]pin.SRI{"x86_64-linux": hashA})})
	if res.Failures != nil {
		t.Fatal(res.Failures)
	}
	if len(res.Changes) != 1 || res.Changes[0].String() != "tool: 1.27.3 -> 1.27.4" {
		t.Fatalf("changes = %v", res.Changes)
	}
	if saved.Version.String() != "1.27.4" || saved.Hashes["x86_64-linux"] != hashA {
		t.Errorf("saved = %+v", saved)
	}
}

func TestRunLeavesCurrentPinAlone(t *testing.T) {
	cur := mustPin(t, "1.27.4", map[string]pin.SRI{"x86_64-linux": hashA})
	res, _, path := run(t, &cur, fixed{pin: cur})
	if res.Failures != nil || len(res.Changes) != 0 || len(res.Current) != 1 {
		t.Fatalf("result = %+v", res)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("pin was truncated")
	}
}

func TestRunRefusesDowngrade(t *testing.T) {
	cur := mustPin(t, "1.27.4", map[string]pin.SRI{"x86_64-linux": hashA})
	res, saved, _ := run(t, &cur, fixed{pin: mustPin(t, "1.27.3", map[string]pin.SRI{"x86_64-linux": hashB})})
	if res.Failures == nil {
		t.Fatal("want failure")
	}
	if saved.Version.String() != "1.27.4" {
		t.Errorf("pin changed to %s", saved.Version)
	}
}

func TestRunRefusesChangedHashForSameVersion(t *testing.T) {
	cur := mustPin(t, "1.27.4", map[string]pin.SRI{"x86_64-linux": hashA})
	res, saved, _ := run(t, &cur, fixed{pin: mustPin(t, "1.27.4", map[string]pin.SRI{"x86_64-linux": hashB})})
	if res.Failures == nil {
		t.Fatal("want failure")
	}
	if saved.Hashes["x86_64-linux"] != hashA {
		t.Error("pin was overwritten")
	}
}

func TestRunAddsNewFileAtSameVersion(t *testing.T) {
	cur := mustPin(t, "1.27.4", map[string]pin.SRI{"x86_64-linux": hashA})
	res, saved, _ := run(t, &cur, fixed{pin: mustPin(t, "1.27.4", map[string]pin.SRI{"x86_64-linux": hashA, "aarch64-darwin": hashB})})
	if res.Failures != nil {
		t.Fatal(res.Failures)
	}
	if len(res.Changes) != 1 || res.Changes[0].String() != "tool: 1.27.4, files [x86_64-linux] -> [aarch64-darwin x86_64-linux]" {
		t.Fatalf("changes = %v", res.Changes)
	}
	if len(saved.Hashes) != 2 {
		t.Errorf("saved = %+v", saved)
	}
}

func TestRunBootstrapsFromEmptyPin(t *testing.T) {
	cur := mustPin(t, "0", map[string]pin.SRI{})
	res, saved, _ := run(t, &cur, fixed{pin: mustPin(t, "1.27.4", map[string]pin.SRI{"x86_64-linux": hashA})})
	if res.Failures != nil {
		t.Fatal(res.Failures)
	}
	if len(res.Changes) != 1 || res.Changes[0].String() != "tool: 0 -> 1.27.4" {
		t.Fatalf("changes = %v", res.Changes)
	}
	if saved.Version.String() != "1.27.4" {
		t.Errorf("saved = %+v", saved)
	}
}

func TestRunFailsOnMissingPin(t *testing.T) {
	res, _, _ := run(t, nil, fixed{pin: mustPin(t, "1.27.4", map[string]pin.SRI{"x86_64-linux": hashA})})
	if res.Failures == nil {
		t.Fatal("want failure for missing pin")
	}
}

func TestRunContinuesAfterFailure(t *testing.T) {
	empty := mustPin(t, "0", map[string]pin.SRI{})
	root, rel := setup(t, &empty)
	good := fixed{pin: mustPin(t, "2", map[string]pin.SRI{"x86_64-linux": hashA})}
	res := Run(context.Background(), root, []Package{
		{Name: "broken", PinPath: rel, Source: fixed{err: errors.New("upstream down")}},
		{Name: "tool", PinPath: rel, Source: good},
	})
	if res.Failures == nil || len(res.Changes) != 1 || res.Changes[0].Package != "tool" {
		t.Fatalf("result = %+v", res)
	}
}
