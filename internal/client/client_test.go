package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
)

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

	c := New(srv.URL, "test-key")
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

	c := New(srv.URL, "k")
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

func TestRetryOnce_On429(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errorMessage":"rate limited"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	_, err := c.ListSubscriptions("", 1, 20)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("calls = %d, want 2 (initial + 1 retry)", calls)
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
	c := New(srv.URL, "k")
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
	c := New(srv.URL, "k")
	_, err := c.ListArticles(123, 1, 100)
	typed, ok := errs.As(err)
	if !ok {
		t.Fatalf("not typed: %v", err)
	}
	if typed.Code != errs.CodeNotFound {
		t.Errorf("Code = %d, want CodeNotFound", typed.Code)
	}
}
