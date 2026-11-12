package gorelease

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stubbedev/flakes.nix/internal/fetch"
	"github.com/stubbedev/flakes.nix/internal/pin"
	"github.com/stubbedev/flakes.nix/internal/version"
)

const DownloadsURL = "https://go.dev/dl/?mode=json"

type Selector struct {
	Kind string `json:"kind"`
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type Settings struct {
	Series string              `json:"series"`
	Files  map[string]Selector `json:"files"`
}

type Source struct {
	url    string
	client fetch.Client
	series version.Version
	files  map[string]Selector
}

func New(raw json.RawMessage, client fetch.Client, url string) (*Source, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var s Settings
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("go release settings: %w", err)
	}
	series, err := version.Parse(s.Series)
	if err != nil {
		return nil, fmt.Errorf("go release settings: series: %w", err)
	}
	if len(s.Files) == 0 {
		return nil, fmt.Errorf("go release settings: no files")
	}
	for key, sel := range s.Files {
		if sel.Kind == "" {
			return nil, fmt.Errorf("go release settings: file %s: kind is required", key)
		}
	}
	return &Source{url: url, client: client, series: series, files: s.Files}, nil
}

type release struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Files   []file `json:"files"`
}

type file struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
}

func (s *Source) Latest(ctx context.Context) (pin.Pin, error) {
	var releases []release
	if err := s.client.JSON(ctx, s.url, &releases); err != nil {
		return pin.Pin{}, fmt.Errorf("list go releases: %w", err)
	}
	newest, v, err := s.newestStable(releases)
	if err != nil {
		return pin.Pin{}, err
	}
	hashes, err := s.hashes(newest)
	if err != nil {
		return pin.Pin{}, err
	}
	return pin.Pin{Version: v, Hashes: hashes}, nil
}

func (s *Source) newestStable(releases []release) (release, version.Version, error) {
	var (
		best    release
		bestVer version.Version
		found   bool
	)
	for _, r := range releases {
		if !r.Stable {
			continue
		}
		v, err := version.Parse(strings.TrimPrefix(r.Version, "go"))
		if err != nil {
			return release{}, version.Version{}, fmt.Errorf("go release %q: %w", r.Version, err)
		}
		if !v.HasPrefix(s.series) {
			continue
		}
		if !found || v.Compare(bestVer) > 0 {
			best, bestVer, found = r, v, true
		}
	}
	if !found {
		return release{}, version.Version{}, fmt.Errorf("list go releases: no stable release in the %s series", s.series)
	}
	return best, bestVer, nil
}

func (s *Source) hashes(r release) (map[string]pin.SRI, error) {
	hashes := make(map[string]pin.SRI, len(s.files))
	for key, sel := range s.files {
		var matches []file
		for _, f := range r.Files {
			if f.Kind == sel.Kind && f.OS == sel.OS && f.Arch == sel.Arch {
				matches = append(matches, f)
			}
		}
		if len(matches) != 1 {
			return nil, fmt.Errorf("go release %s: want one %+v file for %s, found %d", r.Version, sel, key, len(matches))
		}
		sri, err := pin.SHA256FromHex(matches[0].SHA256)
		if err != nil {
			return nil, fmt.Errorf("go release %s: file %s: %w", r.Version, key, err)
		}
		hashes[key] = sri
	}
	return hashes, nil
}
