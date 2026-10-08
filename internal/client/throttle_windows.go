//go:build windows

package client

import "os"

// Windows: no flock; concurrent mp2rss processes are rare there and the server
// still enforces the real limit. Best-effort read/sleep/write without a lock.
func lockFile(_ *os.File) error   { return nil }
func unlockFile(_ *os.File) error { return nil }
