package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
)

// newTestClient builds a client with throttling disabled and an explicit
// MaxRetryWait so tests never touch ~/.mp2rss or sleep on real defaults.
func newTestClient(baseURL string, maxRetryWait time.Duration) *Client {
	return NewWithOptions(baseURL, "k", nil, Options{
		MaxRetryWait: maxRetryWait,
		MinInterval:  0,
	})
}

func TestListSubscriptions_Auth401Decoded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("missing/wrong Authorization header: %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "mp2rss-cli/") {
			t.Errorf("bad User-Agent: %q", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errorMessage":"Feed key is invalid or revoked"}`))
	}))
	defer srv.Close()

	c := NewWithOptions(srv.URL, "test-key", nil, Options{MaxRetryWait: 0, MinInterval: 0})
	_, err := c.ListSubscriptions("", 1, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	typed, ok := errs.As(err)
	if !ok {
		t.Fatalf("expected *errs.Error, got %T", err)
	}
	if typed.Code != errs.CodeAuth {
		t.Errorf("Code = %d, want CodeAuth (%d)", typed.Code, errs.CodeAuth)
	}
	if typed.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("HTTPStatus = %d, want 401", typed.HTTPStatus)
	}
	if !strings.Contains(typed.Message, "Feed key is invalid or revoked") {
		t.Errorf("Message missing errorMessage payload: %q", typed.Message)
	}
}

func TestRetryOnce_On5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errorMessage":"server fart"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0,"page":1,"pageSize":20}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, 0)
	list, err := c.ListSubscriptions("", 1, 20)
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if list.Total != 0 {
		t.Errorf("unexpected list: %+v", list)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("calls = %d, want exactly 2 (initial + 1 retry)", calls)
	}
}

func TestRateLimit_WaitsRetryAfterThenRetriesOnce(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.Header().Set("X-RateLimit-Limit", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"errorMessage":"rate limited"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0,"page":1,"pageSize":20}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, 5*time.Second)
	start := time.Now()
	list, err := c.ListSubscriptions("", 1, 20)
	if err != nil {
		t.Fatalf("expected success after honouring Retry-After, got %v", err)
	}
	if list == nil || atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("calls = %d, want 2 (initial + 1 retry)", calls)
	}
	if waited := time.Since(start); waited < 900*time.Millisecond {
		t.Errorf("waited %s, want >= ~1s (Retry-After: 1)", waited)
	}
}

func TestRateLimit_RetryAfterBeyondCap_FailsImmediately(t *testing.T) {
	var calls int32
	reset := time.Now().Add(600 * time.Second).Unix()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Retry-After", "600")
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errorMessage":"Rate limit exceeded; retry after the Retry-After header"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, 90*time.Second)
	start := time.Now()
	_, err := c.ListSubscriptions("", 1, 20)
	if err == nil {
		t.Fatal("expected rate-limit error")
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("should fail fast, took %s", time.Since(start))
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("calls = %d, want exactly 1 (no blind retry inside lockout)", calls)
	}
	typed, ok := errs.As(err)
	if !ok {
		t.Fatalf("not typed: %v", err)
	}
	if typed.Code != errs.CodeGeneric || typed.HTTPStatus != http.StatusTooManyRequests {
		t.Errorf("Code/HTTPStatus = %d/%d, want 1/429", typed.Code, typed.HTTPStatus)
	}
	if typed.Kind != errs.KindRateLimited {
		t.Errorf("Kind = %q, want %q", typed.Kind, errs.KindRateLimited)
	}
	want := time.Unix(reset, 0).Local().Format("15:04:05")
	for _, frag := range []string{"HTTP 429", want, "max-retry-wait", "每分钟 60 次"} {
		if !strings.Contains(typed.Message, frag) {
			t.Errorf("message missing %q: %s", frag, typed.Message)
		}
	}
}

func TestRateLimit_SecondTwentyNine_GivesUpWithKind(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errorMessage":"rate limited"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, 5*time.Second)
	_, err := c.ListSubscriptions("", 1, 20)
	typed, ok := errs.As(err)
	if !ok {
		t.Fatalf("expected typed error, got %v", err)
	}
	if typed.Kind != errs.KindRateLimited {
		t.Errorf("Kind = %q, want rate_limited", typed.Kind)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("calls = %d, want 2 (initial + 1 retry, then stop)", calls)
	}
}

func TestRateLimit_ZeroMaxWait_NeverSleeps(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, 0)
	start := time.Now()
	_, err := c.ListSubscriptions("", 1, 20)
	if err == nil {
		t.Fatal("expected error")
	}
	if time.Since(start) > time.Second || atomic.LoadInt32(&calls) != 1 {
		t.Errorf("max-retry-wait=0 must fail immediately: took %s, calls=%d", time.Since(start), calls)
	}
}

func TestRateLimit_RespectsContextCancelWhileWaiting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, 60*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := c.doCtx(ctx, http.MethodGet, "/open-api/subscriptions", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("cancel did not interrupt the Retry-After sleep: %s", time.Since(start))
	}
}

func TestParseRateLimit_HeaderFallbacks(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	h := http.Header{}
	got := parseRateLimit(h, now)
	if got.retryAfter != fallbackRetryAfter {
		t.Errorf("no headers: retryAfter = %s, want fallback %s", got.retryAfter, fallbackRetryAfter)
	}
	if !got.resetAt.Equal(now.Add(fallbackRetryAfter)) {
		t.Errorf("no headers: resetAt = %s", got.resetAt)
	}

	h = http.Header{"Retry-After": {now.Add(45 * time.Second).UTC().Format(http.TimeFormat)}}
	got = parseRateLimit(h, now)
	if got.retryAfter < 44*time.Second || got.retryAfter > 46*time.Second {
		t.Errorf("http-date Retry-After = %s, want ~45s", got.retryAfter)
	}

	// Garbage Retry-After but a usable absolute reset → wait derived from the reset.
	h = http.Header{}
	h.Set("Retry-After", "garbage")
	h.Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(90*time.Second).Unix(), 10))
	got = parseRateLimit(h, now)
	if got.retryAfter != 90*time.Second || !got.resetAt.Equal(now.Add(90*time.Second)) {
		t.Errorf("garbage Retry-After + reset: %+v", got)
	}

	// Garbage Retry-After and no reset → gentle fallback.
	h = http.Header{}
	h.Set("Retry-After", "garbage")
	got = parseRateLimit(h, now)
	if got.retryAfter != fallbackRetryAfter {
		t.Errorf("garbage Retry-After alone: %+v", got)
	}
}

func TestThrottle_MinIntervalAcrossClients(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0,"page":1,"pageSize":20}`))
	}))
	defer srv.Close()

	state := filepath.Join(t.TempDir(), "last-request")
	mk := func() *Client {
		return NewWithOptions(srv.URL, "k", nil, Options{
			MaxRetryWait:      0,
			MinInterval:       300 * time.Millisecond,
			ThrottleStatePath: state,
		})
	}

	start := time.Now()
	// Three separate clients = three separate processes sharing the state file.
	for i := 0; i < 3; i++ {
		if _, err := mk().ListSubscriptions("", 1, 20); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	if elapsed < 550*time.Millisecond {
		t.Errorf("3 calls at 300ms min interval took %s, want >= ~600ms", elapsed)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("calls = %d", calls)
	}
}

func TestThrottle_DisabledWhenIntervalZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0,"page":1,"pageSize":20}`))
	}))
	defer srv.Close()
	c := NewWithOptions(srv.URL, "k", nil, Options{MaxRetryWait: 0, MinInterval: 0, ThrottleStatePath: filepath.Join(t.TempDir(), "x")})
	start := time.Now()
	for i := 0; i < 5; i++ {
		if _, err := c.ListSubscriptions("", 1, 20); err != nil {
			t.Fatal(err)
		}
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Errorf("throttle should be off: %s", time.Since(start))
	}
}

func TestDefaultOptions_EnvAndFlagPrecedence(t *testing.T) {
	t.Setenv(envMaxRetryWait, "7")
	t.Setenv(envMinInterval, "250")
	SetDefaultMaxRetryWait(-1)
	o := DefaultOptions()
	if o.MaxRetryWait != 7*time.Second || o.MinInterval != 250*time.Millisecond {
		t.Errorf("env not honoured: %+v", o)
	}
	SetDefaultMaxRetryWait(3 * time.Second)
	defer SetDefaultMaxRetryWait(-1)
	if got := DefaultOptions().MaxRetryWait; got != 3*time.Second {
		t.Errorf("flag override not honoured: %s", got)
	}
}

func TestSubscribe204NoBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := newTestClient(srv.URL, 0)
	if err := c.Subscribe("https://mp.weixin.qq.com/s/abc"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestArticles404_NotSubscribed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorMessage":"MP account is not subscribed"}`))
	}))
	defer srv.Close()
	c := newTestClient(srv.URL, 0)
	_, err := c.ListArticles(123, 1, 100)
	typed, ok := errs.As(err)
	if !ok {
		t.Fatalf("not typed: %v", err)
	}
	if typed.Code != errs.CodeNotFound {
		t.Errorf("Code = %d, want CodeNotFound", typed.Code)
	}
}
