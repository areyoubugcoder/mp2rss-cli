package client

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// throttle enforces a minimum interval between requests across *all* mp2rss
// processes on this machine. The typical offender is a shell loop like
// `for id in $(mp2rss mp list ...); do mp2rss mp articles $id; done` — each
// iteration is a fresh process, so an in-memory limiter would never fire.
//
// State = one file holding the last request time (unix ms). The file is held
// under an advisory lock (flock on unix; best-effort on windows) for the whole
// "read → maybe sleep → write" critical section so concurrent processes
// serialise instead of all sleeping the same amount and then firing together.
//
// Any filesystem problem disables throttling for this call: being throttled is
// a courtesy toward the server, never a reason to fail the user's command.
type throttle struct {
	path     string
	interval time.Duration
}

func newThrottle(path string, interval time.Duration) *throttle {
	if path == "" || interval <= 0 {
		return nil
	}
	return &throttle{path: path, interval: interval}
}

// wait blocks until at least `interval` has elapsed since the last request
// recorded in the state file, then records now. nil receiver = disabled.
func (t *throttle) wait(ctx context.Context) error {
	if t == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o700); err != nil {
		return nil
	}
	f, err := os.OpenFile(t.path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil
	}
	defer f.Close()
	if err := lockFile(f); err != nil {
		return nil
	}
	defer unlockFile(f)

	last := readUnixMs(f)
	now := time.Now()
	if last > 0 {
		elapsed := now.Sub(time.UnixMilli(last))
		if elapsed >= 0 && elapsed < t.interval {
			if err := sleepCtx(ctx, t.interval-elapsed); err != nil {
				return err
			}
			now = time.Now()
		}
	}
	writeUnixMs(f, now.UnixMilli())
	return nil
}

func readUnixMs(f *os.File) int64 {
	buf := make([]byte, 32)
	n, _ := f.ReadAt(buf, 0)
	s := strings.TrimSpace(string(buf[:n]))
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func writeUnixMs(f *os.File, ms int64) {
	_ = f.Truncate(0)
	_, _ = f.WriteAt([]byte(strconv.FormatInt(ms, 10)), 0)
	_ = f.Sync()
}
