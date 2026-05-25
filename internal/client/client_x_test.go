package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
)

// ============================================================================
// XListPosts
// ============================================================================

func TestXListPostsHappy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-api/x/44196397/posts" {
			t.Errorf("bad path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("pageSize") != "20" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [{"postId":"1","content":"hi","media":[],"retweetedPost":null,"quotedPost":null,"threadPosts":[],"postedAt":1}],
			"total": 1, "page": 2, "pageSize": 20
		}`))
	}))
	defer srv.Close()
	list, err := New(srv.URL, "k").XListPosts("44196397", 2, 20)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].PostID != "1" {
		t.Errorf("unexpected list: %+v", list)
	}
}

func TestXListPosts404NotSubscribed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorMessage":"X account is not subscribed"}`))
	}))
	defer srv.Close()
	_, err := New(srv.URL, "k").XListPosts("404404", 1, 20)
	typed, ok := errs.As(err)
	if !ok {
		t.Fatalf("not typed: %v", err)
	}
	if typed.Code != errs.CodeNotFound {
		t.Errorf("Code=%d want CodeNotFound", typed.Code)
	}
}

func TestXListPosts500RetryThenSucceed(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errorMessage":"Internal server error"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0,"page":1,"pageSize":20}`))
	}))
	defer srv.Close()
	list, err := New(srv.URL, "k").XListPosts("1", 1, 20)
	if err != nil {
		t.Fatalf("expected success after retry: %v", err)
	}
	if list.Total != 0 {
		t.Errorf("unexpected list: %+v", list)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

// ============================================================================
// XListArticles
// ============================================================================

func TestXListArticlesHappy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-api/x/44196397/articles" {
			t.Errorf("bad path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [{"url":"https://x.com/a","title":"t","description":"d","contentMarkdown":"# h","coverUrl":"","publishedAt":1}],
			"total": 1, "page": 1, "pageSize": 20
		}`))
	}))
	defer srv.Close()
	list, err := New(srv.URL, "k").XListArticles("44196397", 1, 20)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Title != "t" {
		t.Errorf("unexpected list: %+v", list)
	}
}

func TestXListArticles401Auth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errorMessage":"Feed key is invalid or revoked"}`))
	}))
	defer srv.Close()
	_, err := New(srv.URL, "k").XListArticles("1", 1, 20)
	typed, ok := errs.As(err)
	if !ok {
		t.Fatalf("not typed: %v", err)
	}
	if typed.Code != errs.CodeAuth {
		t.Errorf("Code=%d want CodeAuth", typed.Code)
	}
}

// ============================================================================
// ListSubscriptionsFiltered (sourceType plumbing)
// ============================================================================

func TestListSubscriptionsFilteredPassesSourceType(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0,"page":1,"pageSize":20}`))
	}))
	defer srv.Close()
	_, err := New(srv.URL, "k").ListSubscriptionsFiltered("", "x", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "sourceType=x") {
		t.Errorf("query missing sourceType=x: %s", gotQuery)
	}
}

func TestListSubscriptionsDecodesXItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [
				{"sourceType":"mp","mpId":2234567,"mpName":"公众号 A","createdAt":1,"mpLastArticleAt":2},
				{"sourceType":"x","xUserId":"44196397","xUsername":"elonmusk","xDisplayName":"Elon Musk","xVerified":true,"createdAt":3,"xLastItemAt":4}
			],
			"total": 2, "page": 1, "pageSize": 20
		}`))
	}))
	defer srv.Close()
	list, err := New(srv.URL, "k").ListSubscriptions("", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("want 2 items, got %d", len(list.Items))
	}
	mp, x := list.Items[0], list.Items[1]
	if mp.SourceType != "mp" || mp.MpID != 2234567 {
		t.Errorf("bad mp item: %+v", mp)
	}
	if x.SourceType != "x" || x.XUserID != "44196397" || !x.XVerified {
		t.Errorf("bad x item: %+v", x)
	}
}
