// Package config persists CLI configuration to ~/.mp2rss/config.json.
//
// The file is created with mode 0600 inside a 0700 directory so the Feed Key
// stays readable only by the current user.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DefaultAPIBaseURL is the production mp2rss Open API endpoint.
const DefaultAPIBaseURL = "https://api.mp2rss.com"

// Config is the on-disk representation of the CLI state.
type Config struct {
	FeedKey      string `json:"feed_key,omitempty"`
	APIURL       string `json:"api_url,omitempty"`
	LastLoginAt  int64  `json:"last_login_at,omitempty"`
	LastVerifyAt int64  `json:"last_verify_at,omitempty"`
}

var (
	instance *Config
	once     sync.Once
)

// Get returns the singleton config, loading from disk on first call.
//
// A missing file is not an error: callers see an empty Config they can
// populate and Save.
func Get() *Config {
	once.Do(func() {
		instance = &Config{}
		_ = instance.load()
	})
	return instance
}

// reset clears the singleton. Intended for tests only.
func reset() {
	instance = nil
	once = sync.Once{}
}

// Path returns the on-disk path of the config file.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mp2rss", "config.json"), nil
}

func (c *Config) load() error {
	path, err := Path()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, c)
}

// Save persists the config to disk with 0600 file / 0700 directory perms.
func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	// Tighten the dir in case it pre-existed with looser perms.
	_ = os.Chmod(dir, 0o700)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return os.Chmod(path, 0o600)
}

// Clear wipes the Feed Key but preserves api_url.
func (c *Config) Clear() error {
	c.FeedKey = ""
	c.LastLoginAt = 0
	c.LastVerifyAt = 0
	return c.Save()
}

// IsLoggedIn reports whether a Feed Key is currently configured (env or file).
func (c *Config) IsLoggedIn() bool {
	if v := os.Getenv("MP2RSS_FEED_KEY"); v != "" {
		return true
	}
	return c.FeedKey != ""
}

// EffectiveAPIURL applies the precedence: env > config > default.
//
// Per-invocation overrides via --api-url should be merged into the config
// (in-memory only) by the caller before invoking the HTTP client.
func (c *Config) EffectiveAPIURL() string {
	if v := os.Getenv("MP2RSS_API_URL"); v != "" {
		return v
	}
	if c.APIURL != "" {
		return c.APIURL
	}
	return DefaultAPIBaseURL
}

// EffectiveFeedKey applies the precedence: env > config.
func (c *Config) EffectiveFeedKey() string {
	if v := os.Getenv("MP2RSS_FEED_KEY"); v != "" {
		return v
	}
	return c.FeedKey
}

// MaskKey returns the first 6 visible chars followed by "***".
// An empty key returns "".
func MaskKey(k string) string {
	if k == "" {
		return ""
	}
	if len(k) <= 6 {
		return k + "***"
	}
	return k[:6] + "***"
}
