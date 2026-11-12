package gorelease

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/stubbedev/flakes.nix/internal/fetch"
	"github.com/stubbedev/flakes.nix/internal/pin"
	"github.com/stubbedev/flakes.nix/internal/version"
)

const DownloadsURL = "https://go.dev/dl/?mode=json"

var platformPattern = regexp.MustCompile(`^[a-z0-9]+-[a-z0-9]+$`)

type Settings struct {
	Platforms map[string]string `json:"platforms"`
}

type Source struct {
	url       string
	client    fetch.Client
	platforms map[string]string
}

func New(raw json.RawMessage, client fetch.Client, url string) (*Source, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var s Settings
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("go release settings: %w", err)
	}
	if len(s.Platforms) == 0 {
		return nil, fmt.Errorf("go release settings: no platforms")
	}
	for system, platform := range s.Platforms {
		if !platformPattern.MatchString(platform) {
			return nil, fmt.Errorf("go release settings: system %s: platform %q is not <goos>-<goarch>", system, platform)
		}
	}
	return &Source{url: url, client: client, platforms: s.Platforms}, nil
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
	newest, err := newestStable(releases)
	if err != nil {
		return pin.Pin{}, err
	}
	hashes, err := s.hashes(newest)
	if err != nil {
		return pin.Pin{}, err
	}
	v, err := version.Parse(strings.TrimPrefix(newest.Version, "go"))
	if err != nil {
		return pin.Pin{}, fmt.Errorf("go release %s: %w", newest.Version, err)
	}
	return pin.Pin{Version: v, Hashes: hashes}, nil
}

func newestStable(releases []release) (release, error) {
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
			return release{}, fmt.Errorf("go release %q: %w", r.Version, err)
		}
		if !found || v.Compare(bestVer) > 0 {
			best, bestVer, found = r, v, true
		}
	}
	if !found {
		return release{}, fmt.Errorf("list go releases: no stable release")
	}
	return best, nil
}

func (s *Source) hashes(r release) (map[string]pin.SRI, error) {
	hashes := make(map[string]pin.SRI, len(s.platforms))
	for system, platform := range s.platforms {
		var matches []file
		for _, f := range r.Files {
			if f.Kind == "archive" && f.OS+"-"+f.Arch == platform {
				matches = append(matches, f)
			}
		}
		if len(matches) != 1 {
			return nil, fmt.Errorf("go release %s: want one archive for %s (%s), found %d", r.Version, platform, system, len(matches))
		}
		sri, err := pin.SHA256FromHex(matches[0].SHA256)
		if err != nil {
			return nil, fmt.Errorf("go release %s: %s archive: %w", r.Version, platform, err)
		}
		hashes[system] = sri
	}
	return hashes, nil
}
