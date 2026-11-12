package update

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/stubbedev/flakes.nix/internal/pin"
)

type Source interface {
	Latest(ctx context.Context) (pin.Pin, error)
}

type Package struct {
	Name    string
	PinPath string
	Source  Source
}

type Change struct {
	Package string
	From    pin.Pin
	To      pin.Pin
}

func (c Change) String() string {
	if c.From.Version.Compare(c.To.Version) == 0 {
		return fmt.Sprintf("%s: %s, files %v -> %v", c.Package, c.To.Version, fileKeys(c.From), fileKeys(c.To))
	}
	return fmt.Sprintf("%s: %s -> %s", c.Package, c.From.Version, c.To.Version)
}

type Result struct {
	Changes  []Change
	Current  []string
	Failures error
}

func Run(ctx context.Context, root string, packages []Package) Result {
	var (
		res  Result
		errs []error
	)
	for _, p := range packages {
		change, err := one(ctx, filepath.Join(root, p.PinPath), p.Source)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("update %s: %w", p.Name, err))
		case change == nil:
			res.Current = append(res.Current, p.Name)
		default:
			change.Package = p.Name
			res.Changes = append(res.Changes, *change)
		}
	}
	res.Failures = errors.Join(errs...)
	return res
}

func one(ctx context.Context, path string, source Source) (*Change, error) {
	current, err := pin.Load(path)
	if err != nil {
		return nil, err
	}
	latest, err := source.Latest(ctx)
	if err != nil {
		return nil, err
	}
	changed, err := compare(current, latest)
	if err != nil || !changed {
		return nil, err
	}
	if err := pin.Save(path, latest); err != nil {
		return nil, err
	}
	return &Change{From: current, To: latest}, nil
}

func compare(current, latest pin.Pin) (bool, error) {
	switch latest.Version.Compare(current.Version) {
	case -1:
		return false, fmt.Errorf("upstream latest %s is older than pinned %s", latest.Version, current.Version)
	case 1:
		return true, nil
	}
	for key, h := range latest.Hashes {
		if old, ok := current.Hashes[key]; ok && old != h {
			return false, fmt.Errorf("upstream hash for %s %s changed from %s to %s without a version change", current.Version, key, old, h)
		}
	}
	return !maps.Equal(current.Hashes, latest.Hashes), nil
}

func fileKeys(p pin.Pin) []string {
	return slices.Sorted(maps.Keys(p.Hashes))
}
