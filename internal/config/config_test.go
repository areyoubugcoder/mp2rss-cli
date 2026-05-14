package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveCreates0600File0700Dir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on Windows")
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Cleanup(reset)
	reset()

	c := Get()
	c.FeedKey = "abcdef0123456789"
	c.APIURL = "https://mp2rss.bugcode.dev/api"
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(filepath.Join(tmp, ".mp2rss"))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("dir perm = %o, want 0700", info.Mode().Perm())
	}

	info, err = os.Stat(filepath.Join(tmp, ".mp2rss", "config.json"))
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file perm = %o, want 0600", info.Mode().Perm())
	}
}

func TestClearKeepsAPIURL(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Cleanup(reset)
	reset()

	c := Get()
	c.FeedKey = "secret"
	c.APIURL = "https://api.example.com"
	c.LastVerifyAt = 1234
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	if err := c.Clear(); err != nil {
		t.Fatal(err)
	}
	if c.FeedKey != "" {
		t.Errorf("FeedKey = %q, want empty", c.FeedKey)
	}
	if c.APIURL != "https://api.example.com" {
		t.Errorf("APIURL = %q, want preserved", c.APIURL)
	}
	if c.LastVerifyAt != 0 {
		t.Errorf("LastVerifyAt = %d, want 0", c.LastVerifyAt)
	}
}

func TestEffectiveAPIURL_EnvWins(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("MP2RSS_API_URL", "https://override.example.com")
	t.Cleanup(reset)
	reset()

	c := Get()
	c.APIURL = "https://mp2rss.bugcode.dev/api"
	if got, want := c.EffectiveAPIURL(), "https://override.example.com"; got != want {
		t.Errorf("EffectiveAPIURL = %q, want %q", got, want)
	}
}

func TestEffectiveFeedKey_EnvWins(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("MP2RSS_FEED_KEY", "from-env")
	t.Cleanup(reset)
	reset()

	c := Get()
	c.FeedKey = "from-config"
	if got := c.EffectiveFeedKey(); got != "from-env" {
		t.Errorf("EffectiveFeedKey = %q, want from-env", got)
	}
}

func TestMaskKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"abc", "abc***"},
		{"abcdef", "abcdef***"},
		{"abcdef1234567890", "abcdef***"},
	}
	for _, c := range cases {
		if got := MaskKey(c.in); got != c.want {
			t.Errorf("MaskKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
