package version

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"1.0.0", "v1.0.0", 0}, // leading v optional
		{"v1.0.0", "v1.0.1", -1},
		{"v1.1.0", "v1.0.9", 1},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.0", "v1.0.0", 0}, // zero-padding
		{"v1.0", "v1.0.1", -1},
		{"v1.2.3", "v1.2.3-rc1", 1},  // pre-release loses to release
		{"v1.2.3-rc1", "v1.2.3-rc2", -1},
		{"v1.0.0+abc", "v1.0.0+def", 0}, // build metadata ignored
		{"dev", "v0.0.1", -1},
		{"dev", "dev", 0},
		{"", "v1.0.0", -1},
		{"v1.0.0", "junk", 1},
	}
	for _, c := range cases {
		got := Compare(c.a, c.b)
		if got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	cases := []struct {
		v, goos, goarch, ext, want string
	}{
		{"v0.1.0", "darwin", "arm64", ".tar.gz", "mp2rss-cli_0.1.0_darwin_arm64.tar.gz"},
		{"0.2.3", "linux", "amd64", ".tar.gz", "mp2rss-cli_0.2.3_linux_amd64.tar.gz"},
		{"v1.0.0", "windows", "amd64", ".zip", "mp2rss-cli_1.0.0_windows_amd64.zip"},
	}
	for _, c := range cases {
		if got := AssetName(c.v, c.goos, c.goarch, c.ext); got != c.want {
			t.Errorf("AssetName(%q,%q,%q,%q) = %q, want %q",
				c.v, c.goos, c.goarch, c.ext, got, c.want)
		}
	}
}

func TestString(t *testing.T) {
	old := Version
	defer func() { Version = old }()

	Version = "v1.2.3"
	if got := String(); got != "1.2.3" {
		t.Errorf("String() = %q, want 1.2.3", got)
	}
	Version = "1.2.3"
	if got := String(); got != "1.2.3" {
		t.Errorf("String() = %q, want 1.2.3", got)
	}
}
