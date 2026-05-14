// Package main implements a minimal in-memory mock of the mp2rss Open API.
//
// 用途：mp2rss-cli 的端到端测试脚手架（test/e2e.sh）。当本机没有真实
// Feed Key / dev API 可用时，e2e.sh 会自动启动本进程作为替身。
//
// 这不是生产代码，也不是 CLI 的运行时依赖。
// 详见 ../README.md。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// MockFeedKey 是 mock 唯一接受的 Bearer token。
// e2e.sh 会读取本常量同名值；如调整请同步脚本。
const MockFeedKey = "test-feed-key-0123456789abcdef"

type subscription struct {
	MPID            int64  `json:"mpId"`
	MPName          string `json:"mpName"`
	MPAvatarURL     any    `json:"mpAvatarUrl"`
	CreatedAt       int64  `json:"createdAt"`
	MPLastArticleAt int64  `json:"mpLastArticleAt"`
}

type article struct {
	MPID            int64  `json:"mpId"`
	ArticleID       string `json:"articleId"`
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	CoverImageURL   any    `json:"coverImageUrl"`
	OriginalURL     string `json:"originalUrl"`
	ContentMarkdown string `json:"contentMarkdown"`
	PublishedAt     int64  `json:"publishedAt"`
	UpdatedAt       int64  `json:"updatedAt"`
}

type store struct {
	mu   sync.Mutex
	subs map[int64]subscription
}

func newStore() *store {
	return &store{subs: map[int64]subscription{}}
}

// 预置一组示例文章，方便 articles 端点测试。
var seedArticles = map[int64][]article{
	2234567: {
		{
			MPID:            2234567,
			ArticleID:       "a1",
			Title:           "Hello RSS",
			Summary:         "测试摘要 A",
			CoverImageURL:   nil,
			OriginalURL:     "https://mp.weixin.qq.com/s/test-article-001",
			ContentMarkdown: "# Hi\n\n这是一篇测试文章。",
			PublishedAt:     1744886400000,
			UpdatedAt:       1744886500000,
		},
		{
			MPID:            2234567,
			ArticleID:       "a2",
			Title:           "RSS 进阶",
			Summary:         "测试摘要 B",
			CoverImageURL:   nil,
			OriginalURL:     "https://mp.weixin.qq.com/s/test-article-002",
			ContentMarkdown: "# 进阶\n\n第二篇。",
			PublishedAt:     1744972800000,
			UpdatedAt:       1744972900000,
		},
	},
}

var articleURLRe = regexp.MustCompile(`^https?://mp\.weixin\.qq\.com/s/[A-Za-z0-9_\-]+`)

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"errorMessage": msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// authOK 校验 Authorization: Bearer <MockFeedKey>。
func authOK(r *http.Request) bool {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	return strings.TrimPrefix(h, prefix) == MockFeedKey
}

func parseIntQuery(r *http.Request, key string, def int) (int, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *store) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	if !authOK(r) {
		writeErr(w, http.StatusUnauthorized, "Feed key is invalid or revoked")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.listSubs(w, r)
	case http.MethodPost:
		s.createSub(w, r)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *store) listSubs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	page, err := parseIntQuery(r, "page", 1)
	if err != nil || page < 1 {
		writeErr(w, http.StatusBadRequest, "Invalid page parameter")
		return
	}
	pageSize, err := parseIntQuery(r, "pageSize", 20)
	if err != nil || pageSize < 1 || pageSize > 50 {
		writeErr(w, http.StatusBadRequest, "Invalid pageSize parameter (max 50)")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	items := []subscription{}
	for _, sub := range s.subs {
		if q == "" || strings.Contains(sub.MPName, q) {
			items = append(items, sub)
		}
	}
	// 简单分页（mock 不强求顺序稳定，e2e 校验 total/items 即可）
	total := len(items)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":    items[start:end],
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (s *store) createSub(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ArticleURL string `json:"articleUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if !articleURLRe.MatchString(body.ArticleURL) {
		writeErr(w, http.StatusBadRequest, "Invalid article URL")
		return
	}
	// mock 简化：所有 mp.weixin.qq.com 文章统一解析到 mpId=2234567。
	mpID := int64(2234567)
	mpName := "测试公众号 A"
	s.mu.Lock()
	if _, ok := s.subs[mpID]; !ok {
		s.subs[mpID] = subscription{
			MPID:            mpID,
			MPName:          mpName,
			MPAvatarURL:     nil,
			CreatedAt:       1776553200000,
			MPLastArticleAt: 1776854096000,
		}
	}
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *store) handleSubByID(w http.ResponseWriter, r *http.Request) {
	if !authOK(r) {
		writeErr(w, http.StatusUnauthorized, "Feed key is invalid or revoked")
		return
	}
	// 路径形如 /open-api/subscriptions/{mpId} 或 /open-api/subscriptions/{mpId}/articles
	rest := strings.TrimPrefix(r.URL.Path, "/open-api/subscriptions/")
	parts := strings.SplitN(rest, "/", 2)
	mpID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || mpID <= 0 {
		writeErr(w, http.StatusBadRequest, "Invalid request parameters")
		return
	}
	if len(parts) == 2 && parts[1] == "articles" {
		s.listArticles(w, r, mpID)
		return
	}
	if len(parts) == 1 || parts[1] == "" {
		if r.Method != http.MethodDelete {
			writeErr(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		s.mu.Lock()
		delete(s.subs, mpID)
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeErr(w, http.StatusNotFound, "Not found")
}

func (s *store) listArticles(w http.ResponseWriter, r *http.Request, mpID int64) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	s.mu.Lock()
	_, subscribed := s.subs[mpID]
	s.mu.Unlock()
	if !subscribed {
		writeErr(w, http.StatusNotFound, "MP account is not subscribed")
		return
	}
	items := seedArticles[mpID]
	if items == nil {
		items = []article{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func main() {
	port := flag.Int("port", 0, "listen port (0 = random)")
	flag.Parse()

	// 允许环境变量覆盖（脚本调用更方便）
	if envPort := os.Getenv("MOCK_PORT"); envPort != "" && *port == 0 {
		if p, err := strconv.Atoi(envPort); err == nil {
			*port = p
		}
	}

	s := newStore()
	mux := http.NewServeMux()
	mux.HandleFunc("/open-api/subscriptions", s.handleSubscriptions)
	mux.HandleFunc("/open-api/subscriptions/", s.handleSubByID)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "ok")
	})

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s failed: %v", addr, err)
	}
	actualAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		log.Fatalf("unexpected listener address: %T", ln.Addr())
	}
	// 第一行 stdout 必须是 "listening on :NNNN"，脚本依此读取端口。
	fmt.Printf("listening on :%d\n", actualAddr.Port)

	srv := &http.Server{Handler: mux}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve failed: %v", err)
	}
}
