// Package semver parses, compares and bumps semantic versions (semver.org 2.0.0),
// tolerating a leading tag prefix such as "v".
package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a parsed semantic version. Prefix keeps whatever came before the
// numbers (e.g. "v") so a bumped version is rendered the way the repo tags them.
type Version struct {
	Prefix     string
	Major      uint64
	Minor      uint64
	Patch      uint64
	Prerelease []string
	Build      string
}

// Parse reads a version like "v1.2.3", "1.2.3-rc.1" or "release-2.0.0+abc".
// prefix is stripped first when non-empty; otherwise an optional "v" is allowed.
func Parse(s, prefix string) (Version, error) {
	raw := s
	var v Version
	switch {
	case prefix != "":
		if !strings.HasPrefix(s, prefix) {
			return v, fmt.Errorf("%q does not start with prefix %q", raw, prefix)
		}
		v.Prefix, s = prefix, s[len(prefix):]
	case strings.HasPrefix(s, "v") || strings.HasPrefix(s, "V"):
		v.Prefix, s = s[:1], s[1:]
	}
	if i := strings.IndexByte(s, '+'); i >= 0 {
		v.Build, s = s[i+1:], s[:i]
		if v.Build == "" {
			return v, fmt.Errorf("%q: empty build metadata", raw)
		}
	}
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre := s[i+1:]
		s = s[:i]
		if pre == "" {
			return v, fmt.Errorf("%q: empty prerelease", raw)
		}
		v.Prerelease = strings.Split(pre, ".")
		for _, id := range v.Prerelease {
			if id == "" {
				return v, fmt.Errorf("%q: empty prerelease identifier", raw)
			}
		}
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, fmt.Errorf("%q: want MAJOR.MINOR.PATCH", raw)
	}
	nums := make([]uint64, 3)
	for i, p := range parts {
		if p == "" || (len(p) > 1 && p[0] == '0') {
			return v, fmt.Errorf("%q: invalid number %q", raw, p)
		}
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return v, fmt.Errorf("%q: invalid number %q", raw, p)
		}
		nums[i] = n
	}
	v.Major, v.Minor, v.Patch = nums[0], nums[1], nums[2]
	return v, nil
}

// String renders the version including its prefix.
func (v Version) String() string {
	s := fmt.Sprintf("%s%d.%d.%d", v.Prefix, v.Major, v.Minor, v.Patch)
	if len(v.Prerelease) > 0 {
		s += "-" + strings.Join(v.Prerelease, ".")
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// Compare returns -1, 0 or 1. Build metadata is ignored, as the spec requires.
func Compare(a, b Version) int {
	for _, d := range [][2]uint64{{a.Major, b.Major}, {a.Minor, b.Minor}, {a.Patch, b.Patch}} {
		if d[0] != d[1] {
			if d[0] < d[1] {
				return -1
			}
			return 1
		}
	}
	// A version without prerelease ranks higher than one with it.
	switch {
	case len(a.Prerelease) == 0 && len(b.Prerelease) == 0:
		return 0
	case len(a.Prerelease) == 0:
		return 1
	case len(b.Prerelease) == 0:
		return -1
	}
	for i := 0; i < len(a.Prerelease) && i < len(b.Prerelease); i++ {
		if c := compareIdent(a.Prerelease[i], b.Prerelease[i]); c != 0 {
			return c
		}
	}
	switch {
	case len(a.Prerelease) < len(b.Prerelease):
		return -1
	case len(a.Prerelease) > len(b.Prerelease):
		return 1
	}
	return 0
}

func compareIdent(a, b string) int {
	an, aErr := strconv.ParseUint(a, 10, 64)
	bn, bErr := strconv.ParseUint(b, 10, 64)
	switch {
	case aErr == nil && bErr == nil:
		if an == bn {
			return 0
		}
		if an < bn {
			return -1
		}
		return 1
	case aErr == nil: // numeric identifiers rank lower than alphanumeric
		return -1
	case bErr == nil:
		return 1
	}
	return strings.Compare(a, b)
}

// Level is how far a release moves the version.
type Level int

const (
	None Level = iota
	Patch
	Minor
	Major
)

func (l Level) String() string {
	return [...]string{"none", "patch", "minor", "major"}[l]
}

// Bump returns the next release version. Prerelease and build data are dropped.
// Before 1.0.0, a breaking change bumps minor instead of major, following the
// common convention that 0.x is unstable.
func Bump(v Version, l Level) Version {
	next := Version{Prefix: v.Prefix, Major: v.Major, Minor: v.Minor, Patch: v.Patch}
	if len(v.Prerelease) > 0 {
		// 1.2.0-rc.1 + any change releases as 1.2.0 when the core already
		// reflects at least that level of change.
		if l == Patch || (l == Minor && v.Patch == 0) || (l == Major && v.Patch == 0 && v.Minor == 0) {
			return next
		}
	}
	if l == Major && v.Major == 0 {
		l = Minor
	}
	switch l {
	case Major:
		next.Major++
		next.Minor, next.Patch = 0, 0
	case Minor:
		next.Minor++
		next.Patch = 0
	case Patch:
		next.Patch++
	}
	return next
}
