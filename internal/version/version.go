// Package version holds the CLI version, injected at build time via ldflags.
package version

import "strings"

// Version is set at build time:
//
//	-ldflags "-X github.com/areyoubugcoder/mp2rss-cli/internal/version.Version=v0.1.0"
var Version = "dev"

// String returns the version without a leading "v".
func String() string {
	return strings.TrimPrefix(Version, "v")
}
