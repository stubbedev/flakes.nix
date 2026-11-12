package pin

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/stubbedev/flakes.nix/internal/version"
)

type SRI string

var sriPattern = regexp.MustCompile(`^sha256-[A-Za-z0-9+/]{43}=$`)

func SHA256FromHex(s string) (SRI, error) {
	raw, err := hex.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("decode sha256 hex %q: %w", s, err)
	}
	if len(raw) != 32 {
		return "", fmt.Errorf("decode sha256 hex %q: got %d bytes, want 32", s, len(raw))
	}
	return SRI("sha256-" + base64.StdEncoding.EncodeToString(raw)), nil
}

func ParseSRI(s string) (SRI, error) {
	if !sriPattern.MatchString(s) {
		return "", fmt.Errorf("parse SRI hash %q: want sha256-<base64 of 32 bytes>", s)
	}
	return SRI(s), nil
}

type Pin struct {
	Version version.Version
	Hashes  map[string]SRI
}

type document struct {
	Version string            `json:"version"`
	Hashes  map[string]string `json:"hashes"`
}

func Load(path string) (Pin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Pin{}, fmt.Errorf("load pin: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var doc document
	if err := dec.Decode(&doc); err != nil {
		return Pin{}, fmt.Errorf("load pin %s: %w", path, err)
	}
	v, err := version.Parse(doc.Version)
	if err != nil {
		return Pin{}, fmt.Errorf("load pin %s: %w", path, err)
	}
	hashes := make(map[string]SRI, len(doc.Hashes))
	for system, h := range doc.Hashes {
		sri, err := ParseSRI(h)
		if err != nil {
			return Pin{}, fmt.Errorf("load pin %s: system %s: %w", path, system, err)
		}
		hashes[system] = sri
	}
	return Pin{Version: v, Hashes: hashes}, nil
}

func Save(path string, p Pin) (err error) {
	if len(p.Hashes) == 0 {
		return fmt.Errorf("save pin %s: no hashes", path)
	}
	doc := document{Version: p.Version.String(), Hashes: make(map[string]string, len(p.Hashes))}
	for system, h := range p.Hashes {
		doc.Hashes[system] = string(h)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("save pin %s: %w", path, err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("save pin %s: %w", path, err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, removeIfExists(tmp.Name()))
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return errors.Join(fmt.Errorf("save pin %s: %w", path, err), tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("save pin %s: %w", path, err)
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return fmt.Errorf("save pin %s: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("save pin %s: %w", path, err)
	}
	return nil
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove temporary file: %w", err)
	}
	return nil
}
