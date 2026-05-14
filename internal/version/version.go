// Package version holds the CLI version, injected at build time via ldflags.
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is set at build time:
//
//	-ldflags "-X github.com/areyoubugcoder/mp2rss-cli/internal/version.Version=v0.1.0"
var Version = "dev"

// String returns the version without a leading "v".
func String() string {
	return strings.TrimPrefix(Version, "v")
}

// Compare returns:
//
//	-1 if a < b, 0 if equal, 1 if a > b.
//
// Both inputs may have a leading "v". Non-numeric suffixes (-rc1, +build5) are
// treated as making the version *older* than the same base without a suffix —
// e.g. "1.2.0-rc1" < "1.2.0" — which matches semver pre-release ordering.
//
// Empty strings, "dev", or any value that does not parse to at least one
// numeric component is considered older than any real version.
func Compare(a, b string) int {
	an, apre, aok := parse(a)
	bn, bpre, bok := parse(b)
	switch {
	case !aok && !bok:
		return 0
	case !aok:
		return -1
	case !bok:
		return 1
	}
	// pad to equal length so "1.2" < "1.2.1"
	for len(an) < len(bn) {
		an = append(an, 0)
	}
	for len(bn) < len(an) {
		bn = append(bn, 0)
	}
	for i := range an {
		if an[i] != bn[i] {
			if an[i] < bn[i] {
				return -1
			}
			return 1
		}
	}
	// numbers equal; pre-release loses
	switch {
	case apre == "" && bpre == "":
		return 0
	case apre == "":
		return 1
	case bpre == "":
		return -1
	}
	if apre < bpre {
		return -1
	} else if apre > bpre {
		return 1
	}
	return 0
}

// parse extracts the dotted-numeric components from v plus any "-suffix".
// Returns ok=false if v has no leading numeric component.
func parse(v string) (nums []int, pre string, ok bool) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if v == "" || v == "dev" {
		return nil, "", false
	}
	// strip metadata after '+' (e.g. "1.0.0+abc")
	if i := strings.IndexByte(v, '+'); i != -1 {
		v = v[:i]
	}
	// split pre-release suffix after '-'
	if i := strings.IndexByte(v, '-'); i != -1 {
		pre = v[i+1:]
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, "", false
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return nil, "", false
	}
	return nums, pre, true
}

// AssetName returns the conventional release asset filename for the given
// version, OS and arch. ext is ".tar.gz" for unix or ".zip" for windows.
func AssetName(version, goos, goarch, ext string) string {
	return fmt.Sprintf("mp2rss-cli_%s_%s_%s%s",
		strings.TrimPrefix(version, "v"), goos, goarch, ext)
}
