package authflow

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewBindsValidPort(t *testing.T) {
	f, err := New("https://mp2rss.com")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if f.Port < 1024 || f.Port > 65535 {
		t.Errorf("Port = %d outside 1024-65535", f.Port)
	}
	if len(f.State) != 64 {
		t.Errorf("State hex length = %d, want 64 (32 bytes hex)", len(f.State))
	}
}

func TestAuthorizeURL_Composition(t *testing.T) {
	f, err := New("https://mp2rss.com")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	url := f.AuthorizeURL("0.1.0")
	if !strings.HasPrefix(url, "https://mp2rss.com/cli/authorize?") {
		t.Errorf("bad prefix: %s", url)
	}
	if !strings.Contains(url, "state="+f.State) {
		t.Errorf("missing state in URL: %s", url)
	}
	if !strings.Contains(url, "v=0.1.0") {
		t.Errorf("missing version in URL: %s", url)
	}
}

func TestWait_SuccessfulCallback(t *testing.T) {
	f, err := New("http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	f.Timeout = 5 * time.Second
	f.AllowedOrigins = []string{"http://localhost:5173"}

	done := make(chan struct {
		res *CallbackResult
		err error
	}, 1)
	go func() {
		res, err := f.Wait(context.Background())
		done <- struct {
			res *CallbackResult
			err error
		}{res, err}
	}()

	// Wait a hair for the server to spin up, then POST.
	time.Sleep(50 * time.Millisecond)
	body, _ := json.Marshal(map[string]string{"feed_key": "k1", "state": f.State})
	req, _ := http.NewRequest(http.MethodPost, f.CallbackURL(), bytes.NewReader(body))
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("Wait returned err: %v", r.err)
		}
		if r.res.FeedKey != "k1" {
			t.Errorf("FeedKey = %q, want k1", r.res.FeedKey)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return")
	}
}

func TestWait_BadStateRejected(t *testing.T) {
	f, err := New("http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	f.Timeout = 2 * time.Second
	f.AllowedOrigins = []string{"http://localhost:5173"}

	done := make(chan error, 1)
	go func() {
		_, err := f.Wait(context.Background())
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)

	body, _ := json.Marshal(map[string]string{"feed_key": "k1", "state": "wrong"})
	req, _ := http.NewRequest(http.MethodPost, f.CallbackURL(), bytes.NewReader(body))
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	select {
	case e := <-done:
		if e == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(e.Error(), "state") {
			t.Errorf("error not about state: %v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Wait did not return on bad state")
	}
}

func TestWait_OriginRejected(t *testing.T) {
	f, err := New("http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	f.Timeout = 1500 * time.Millisecond
	f.AllowedOrigins = []string{"http://localhost:5173"}

	done := make(chan error, 1)
	go func() {
		_, err := f.Wait(context.Background())
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)

	body, _ := json.Marshal(map[string]string{"feed_key": "k1", "state": f.State})
	req, _ := http.NewRequest(http.MethodPost, f.CallbackURL(), bytes.NewReader(body))
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Server keeps waiting after a rejected origin → timeout path.
	select {
	case e := <-done:
		if e == nil || !strings.Contains(e.Error(), "超时") {
			t.Errorf("expected timeout error, got %v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Wait did not time out")
	}
}

func TestWait_Timeout(t *testing.T) {
	f, err := New("http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	f.Timeout = 200 * time.Millisecond

	start := time.Now()
	_, err = f.Wait(context.Background())
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed > 2*time.Second {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestOptionsPreflight(t *testing.T) {
	f, err := New("http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	f.Timeout = 500 * time.Millisecond
	f.AllowedOrigins = []string{"http://localhost:5173"}

	go func() {
		_, _ = f.Wait(context.Background())
	}()
	time.Sleep(50 * time.Millisecond)

	req, _ := http.NewRequest(http.MethodOptions, f.CallbackURL(), nil)
	req.Header.Set("Origin", "http://localhost:5173")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("CORS origin = %q", got)
	}
}
