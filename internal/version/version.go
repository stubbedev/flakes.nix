package version

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	raw   string
	parts []uint64
}

func Parse(s string) (Version, error) {
	if s == "" {
		return Version{}, fmt.Errorf("parse version %q: empty", s)
	}
	fields := strings.Split(s, ".")
	parts := make([]uint64, len(fields))
	for i, f := range fields {
		n, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("parse version %q: component %q is not a non-negative integer: %w", s, f, err)
		}
		parts[i] = n
	}
	return Version{raw: s, parts: parts}, nil
}

func (v Version) String() string {
	return v.raw
}

func (v Version) Compare(o Version) int {
	for i := range max(len(v.parts), len(o.parts)) {
		a, b := component(v.parts, i), component(o.parts, i)
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		}
	}
	return 0
}

func component(parts []uint64, i int) uint64 {
	if i < len(parts) {
		return parts[i]
	}
	return 0
}

func (v Version) HasPrefix(prefix Version) bool {
	if len(prefix.parts) > len(v.parts) {
		return false
	}
	for i, p := range prefix.parts {
		if v.parts[i] != p {
			return false
		}
	}
	return true
}
