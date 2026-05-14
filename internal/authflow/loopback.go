// Package authflow implements the OAuth Loopback flow used by `mp2rss auth login`.
//
// Lifecycle:
//  1. Caller calls New() with the resolved Web origin and asks for an open port.
//  2. Caller composes the authorize URL using AuthorizeURL().
//  3. Caller invokes Wait(ctx) to block until the browser POSTs the Feed Key
//     to /cli/callback, or 120s elapses.
//  4. Server shuts itself down after the first successful callback.
//
// Security:
//   - Listens on 127.0.0.1 only.
//   - Single route: /cli/callback (POST).
//   - Validates Origin against an allow-list.
//   - Validates state nonce against CSRF.
//   - Never prints the Feed Key to the terminal.
package authflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultTimeout is how long we wait for the browser to POST back.
const DefaultTimeout = 120 * time.Second

// AllowedOrigins are the only Origin headers we'll accept.
//
// Web 端 vite dev 默认起在 :3000（apps/web vite.config 已固定），生产是
// https://mp2rss.com。同一台机器同时通过 IPv4 / IPv6 解析 localhost 时
// 浏览器可能发出 http://[::1]:3000，所以两条都列上。
var AllowedOrigins = []string{
	"https://mp2rss.com",
	"http://localhost:3000",
	"http://[::1]:3000",
}

// CallbackResult is what the browser POSTs to /cli/callback.
type CallbackResult struct {
	FeedKey string `json:"feed_key"`
	State   string `json:"state"`
}

// Flow is a single-shot loopback server.
type Flow struct {
	State          string        // 32-byte hex CSRF nonce
	Port           int           // TCP port we bound to
	WebOrigin      string        // e.g. https://mp2rss.com
	Timeout        time.Duration // override DefaultTimeout for tests

	listener net.Listener
	server   *http.Server
	mu       sync.Mutex
	done     chan struct{}
	result   *CallbackResult
	resErr   error

	// AllowedOrigins overrides the package default — useful for tests.
	AllowedOrigins []string
}

// New binds 127.0.0.1 on a random unprivileged port and generates a state.
func New(webOrigin string) (*Flow, error) {
	state, err := randomHex(32)
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("binding loopback listener: %w", err)
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return nil, fmt.Errorf("unexpected listener address: %T", ln.Addr())
	}
	if addr.Port < 1024 || addr.Port > 65535 {
		_ = ln.Close()
		return nil, fmt.Errorf("got unusable port %d", addr.Port)
	}
	return &Flow{
		State:     state,
		Port:      addr.Port,
		WebOrigin: strings.TrimRight(webOrigin, "/"),
		listener:  ln,
		done:      make(chan struct{}),
	}, nil
}

// AuthorizeURL composes the URL the user opens in their browser.
func (f *Flow) AuthorizeURL(cliVersion string) string {
	v := url.Values{}
	v.Set("port", fmt.Sprintf("%d", f.Port))
	v.Set("state", f.State)
	v.Set("v", cliVersion)
	return f.WebOrigin + "/cli/authorize?" + v.Encode()
}

// CallbackURL returns the loopback URL the browser POSTs to.
func (f *Flow) CallbackURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/cli/callback", f.Port)
}

// Wait starts the loopback HTTP server and blocks until either:
//   - the browser POSTs a valid callback (returns the result)
//   - timeout elapses
//   - ctx is canceled
func (f *Flow) Wait(ctx context.Context) (*CallbackResult, error) {
	timeout := f.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/cli/callback", f.handleCallback)

	f.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		_ = f.server.Serve(f.listener)
	}()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-f.done:
		_ = f.shutdown()
		return f.result, f.resErr
	case <-ctx.Done():
		_ = f.shutdown()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("等待浏览器授权超时（%s），请改用 --feed-key 或 --no-browser", timeout)
		}
		return nil, ctx.Err()
	}
}

// Close shuts down the underlying listener without waiting. Safe to call
// multiple times.
func (f *Flow) Close() error { return f.shutdown() }

func (f *Flow) shutdown() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.server == nil {
		// Listener may still be open if Wait was never called.
		if f.listener != nil {
			err := f.listener.Close()
			f.listener = nil
			return err
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := f.server.Shutdown(ctx)
	f.server = nil
	return err
}

func (f *Flow) handleCallback(w http.ResponseWriter, r *http.Request) {
	// CORS preflight: many browsers send OPTIONS for cross-origin fetches.
	origin := r.Header.Get("Origin")
	allowed := f.originAllowed(origin)
	if allowed {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "600")
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !allowed {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}

	var payload CallbackResult
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if payload.State == "" || payload.State != f.State {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		f.fail(fmt.Errorf("state mismatch — possible CSRF"))
		return
	}
	if payload.FeedKey == "" {
		http.Error(w, "missing feed_key", http.StatusBadRequest)
		f.fail(fmt.Errorf("回调中未带 feed_key"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(successHTML))

	f.succeed(&payload)
}

func (f *Flow) originAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	list := f.AllowedOrigins
	if len(list) == 0 {
		list = AllowedOrigins
	}
	for _, o := range list {
		if origin == o {
			return true
		}
	}
	return false
}

func (f *Flow) succeed(res *CallbackResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.result != nil || f.resErr != nil {
		return
	}
	f.result = res
	close(f.done)
}

func (f *Flow) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.result != nil || f.resErr != nil {
		return
	}
	f.resErr = err
	close(f.done)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

const successHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>授权成功 — mp2rss CLI</title>
<style>
  body { font: 16px/1.6 system-ui, -apple-system, "Segoe UI", "PingFang SC", sans-serif; margin: 0; padding: 64px 24px; background: #fafafa; color: #111; text-align: center; }
  .card { max-width: 480px; margin: 0 auto; background: #fff; padding: 40px 32px; border-radius: 12px; box-shadow: 0 1px 3px rgba(0,0,0,.04), 0 8px 24px rgba(0,0,0,.04); }
  h1 { margin: 0 0 8px; font-size: 20px; }
  p { margin: 8px 0 0; color: #555; }
  .check { width: 56px; height: 56px; line-height: 56px; border-radius: 50%; background: #10b981; color: #fff; font-size: 28px; margin: 0 auto 16px; }
</style>
</head>
<body>
  <div class="card">
    <div class="check">✓</div>
    <h1>授权成功</h1>
    <p>mp2rss CLI 已收到您的 Feed Key，您可以关闭此页面回到终端。</p>
  </div>
</body>
</html>
`
