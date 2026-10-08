// Package client is a thin HTTP wrapper around the mp2rss Open API.
//
// It handles:
//   - Bearer auth (Feed Key)
//   - User-Agent stamping
//   - 30s timeout with one retry on 5xx / transport errors
//   - 429: honour Retry-After (bounded by MaxRetryWait), retry once, then give up
//     with a KindRateLimited error that tells the user when the lock lifts
//   - a cross-process minimum interval between requests (file-backed) so a
//     shell loop over subscriptions stays under the server's 60 req/min
//   - decoding {"errorMessage":"..."} error bodies into typed errs.Error
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/areyoubugcoder/mp2rss-cli/internal/version"
)

// DefaultTimeout is the per-request HTTP timeout.
const DefaultTimeout = 30 * time.Second

// Server-side rate limit facts (docs/api/open-api.md): 60 req/min per feed key,
// a breach locks the key for 600s. These only feed user-facing hints; the
// authoritative numbers always come from the 429 response headers.
const (
	ServerRequestsPerMinute = 60
	ServerLockoutDuration   = 10 * time.Minute
)

const (
	// DefaultMaxRetryWait caps how long a 429 Retry-After may make us sleep
	// before retrying once. Longer waits (the 600s lockout) fail fast instead:
	// a cron job should exit and run again next tick, not hang for 10 minutes.
	DefaultMaxRetryWait = 90 * time.Second
	// DefaultMinInterval spaces consecutive requests from this machine at
	// ~50 req/min — under the 60/min server limit with headroom for other
	// clients sharing the key. Enforced across processes via a state file.
	DefaultMinInterval = 1200 * time.Millisecond
	// fallbackRetryAfter is used when a 429 carries no usable Retry-After
	// (e.g. the nginx per-IP text 429).
	fallbackRetryAfter = 2 * time.Second

	envMaxRetryWait = "MP2RSS_MAX_RETRY_WAIT"  // seconds
	envMinInterval  = "MP2RSS_MIN_INTERVAL_MS" // milliseconds
)

// Options tunes rate-limit behaviour. Zero values fall back to env / defaults
// via DefaultOptions; tests construct explicit values.
type Options struct {
	// MaxRetryWait: a 429 whose Retry-After is <= this is waited out then
	// retried once; larger → fail immediately with KindRateLimited. Negative
	// = use default. 0 = never wait.
	MaxRetryWait time.Duration
	// MinInterval between consecutive requests from this host (all processes).
	// Negative = use default. 0 = disabled.
	MinInterval time.Duration
	// ThrottleStatePath is the file holding the last-request timestamp. Empty
	// = ~/.mp2rss/.last-request. Unwritable paths silently disable throttling
	// (rate limiting is a courtesy, never a reason to fail a command).
	ThrottleStatePath string
}

var (
	defaultsMu          sync.Mutex
	defaultMaxRetryWait = -1 * time.Second // -1 = resolve from env/default
	defaultMinInterval  = -1 * time.Second
)

// SetDefaultMaxRetryWait overrides the process-wide default (set from the
// --max-retry-wait flag). It takes precedence over the env variable.
func SetDefaultMaxRetryWait(d time.Duration) {
	defaultsMu.Lock()
	defer defaultsMu.Unlock()
	defaultMaxRetryWait = d
}

// DefaultOptions resolves options from flags (SetDefault*), then env, then
// built-in defaults.
func DefaultOptions() Options {
	defaultsMu.Lock()
	maxWait, minInt := defaultMaxRetryWait, defaultMinInterval
	defaultsMu.Unlock()
	if maxWait < 0 {
		maxWait = DefaultMaxRetryWait
		if v := os.Getenv(envMaxRetryWait); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				maxWait = time.Duration(n) * time.Second
			}
		}
	}
	if minInt < 0 {
		minInt = DefaultMinInterval
		if v := os.Getenv(envMinInterval); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				minInt = time.Duration(n) * time.Millisecond
			}
		}
	}
	return Options{MaxRetryWait: maxWait, MinInterval: minInt}
}

// Client is an mp2rss Open API client.
type Client struct {
	baseURL    string
	feedKey    string
	userAgent  string
	httpClient *http.Client
	opts       Options
	throttle   *throttle
}

// New builds a Client with DefaultOptions. baseURL should NOT have a trailing slash.
func New(baseURL, feedKey string) *Client {
	return NewWithOptions(baseURL, feedKey, &http.Client{Timeout: DefaultTimeout}, DefaultOptions())
}

// NewWithHTTP is the test-friendly constructor accepting a custom *http.Client.
func NewWithHTTP(baseURL, feedKey string, httpClient *http.Client) *Client {
	return NewWithOptions(baseURL, feedKey, httpClient, DefaultOptions())
}

// NewWithOptions is the fully explicit constructor.
func NewWithOptions(baseURL, feedKey string, httpClient *http.Client, opts Options) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	if opts.MaxRetryWait < 0 {
		opts.MaxRetryWait = DefaultOptions().MaxRetryWait
	}
	if opts.MinInterval < 0 {
		opts.MinInterval = DefaultOptions().MinInterval
	}
	if opts.ThrottleStatePath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			opts.ThrottleStatePath = filepath.Join(home, ".mp2rss", ".last-request")
		}
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL:    baseURL,
		feedKey:    feedKey,
		userAgent:  fmt.Sprintf("mp2rss-cli/%s (%s/%s)", version.String(), runtime.GOOS, runtime.GOARCH),
		httpClient: httpClient,
		opts:       opts,
		throttle:   newThrottle(opts.ThrottleStatePath, opts.MinInterval),
	}
}

// Options returns the resolved options (for tests / diagnostics).
func (c *Client) Options() Options { return c.opts }

// BaseURL returns the resolved base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// ---------------------------------------------------------------------------
// DTOs
// ---------------------------------------------------------------------------

// Subscription is one row of GET /open-api/subscriptions.
//
// The endpoint returns a discriminated union keyed by `sourceType`:
//   - MP items populate the Mp* fields
//   - X items populate the X* fields
//
// X-specific fields use omitempty so an MP-filtered listing stays clean; MP
// fields likewise omitempty so X-filtered listing isn't polluted with
// `mpId:0` noise.
type Subscription struct {
	SourceType string `json:"sourceType,omitempty"`

	// MP fields
	MpID            int64  `json:"mpId,omitempty"`
	MpName          string `json:"mpName,omitempty"`
	MpAvatarURL     string `json:"mpAvatarUrl,omitempty"`
	MpLastArticleAt int64  `json:"mpLastArticleAt,omitempty"`

	// X fields
	XUserID      string `json:"xUserId,omitempty"`
	XUsername    string `json:"xUsername,omitempty"`
	XDisplayName string `json:"xDisplayName,omitempty"`
	XAvatarURL   string `json:"xAvatarUrl,omitempty"`
	XVerified    bool   `json:"xVerified,omitempty"`
	XLastItemAt  int64  `json:"xLastItemAt,omitempty"`

	CreatedAt int64 `json:"createdAt,omitempty"`
}

// SubscriptionList is the response body of GET /open-api/subscriptions.
type SubscriptionList struct {
	Items    []Subscription `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

// Article is one row of /open-api/subscriptions/{mpId}/articles.
type Article struct {
	MpID            int64  `json:"mpId"`
	ArticleID       string `json:"articleId"`
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	CoverImageURL   string `json:"coverImageUrl"`
	OriginalURL     string `json:"originalUrl"`
	ContentMarkdown string `json:"contentMarkdown"`
	PublishedAt     int64  `json:"publishedAt"`
	UpdatedAt       int64  `json:"updatedAt"`
}

// ArticleList is the response body of GET /open-api/subscriptions/{mpId}/articles.
type ArticleList struct {
	Items []Article `json:"items"`
}

// SubscribeRequest is the body of POST /open-api/subscriptions.
type SubscribeRequest struct {
	ArticleURL string `json:"articleUrl"`
}

// ---------------------------------------------------------------------------
// X DTOs
// ---------------------------------------------------------------------------
//
// X 账号搜索与订阅 / 取消订阅仅在 Web 控制台提供——Open API 与本 client
// 都不暴露这些写类端点。只保留读类 DTO（posts / articles）。

// XPostMedia is one media attachment on an X post.
type XPostMedia struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}

// XPost is one row of GET /open-api/x/:xUserId/posts.
type XPost struct {
	PostID        string         `json:"postId"`
	Content       string         `json:"content"`
	Media         []XPostMedia   `json:"media"`
	RetweetedPost map[string]any `json:"retweetedPost"`
	QuotedPost    map[string]any `json:"quotedPost"`
	ThreadPosts   []any          `json:"threadPosts"`
	PostedAt      int64          `json:"postedAt"`
}

// XPostList is the response body of GET /open-api/x/:xUserId/posts.
type XPostList struct {
	Items    []XPost `json:"items"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"pageSize"`
}

// XArticle is one row of GET /open-api/x/:xUserId/articles.
type XArticle struct {
	URL             string `json:"url"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	ContentMarkdown string `json:"contentMarkdown"`
	CoverURL        string `json:"coverUrl"`
	PublishedAt     int64  `json:"publishedAt"`
}

// XArticleList is the response body of GET /open-api/x/:xUserId/articles.
type XArticleList struct {
	Items    []XArticle `json:"items"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
}

// ---------------------------------------------------------------------------
// API methods (MP)
// ---------------------------------------------------------------------------

// ListSubscriptions calls GET /open-api/subscriptions without a sourceType
// filter (server default = all). Prefer ListSubscriptionsFiltered when you
// want to scope to a single source type.
func (c *Client) ListSubscriptions(q string, page, pageSize int) (*SubscriptionList, error) {
	return c.ListSubscriptionsFiltered(q, "", page, pageSize)
}

// ListSubscriptionsFiltered calls GET /open-api/subscriptions?sourceType=<...>.
//
// sourceType must be "", "mp", "x" or "all". Empty omits the param entirely.
func (c *Client) ListSubscriptionsFiltered(q, sourceType string, page, pageSize int) (*SubscriptionList, error) {
	v := url.Values{}
	if q != "" {
		v.Set("q", q)
	}
	if sourceType != "" {
		v.Set("sourceType", sourceType)
	}
	if page > 0 {
		v.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	var out SubscriptionList
	if err := c.do(http.MethodGet, "/open-api/subscriptions", v, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Subscribe calls POST /open-api/subscriptions.
func (c *Client) Subscribe(articleURL string) error {
	body := SubscribeRequest{ArticleURL: articleURL}
	return c.do(http.MethodPost, "/open-api/subscriptions", nil, body, nil)
}

// Unsubscribe calls DELETE /open-api/subscriptions/{mpId}.
func (c *Client) Unsubscribe(mpID int64) error {
	path := fmt.Sprintf("/open-api/subscriptions/%d", mpID)
	return c.do(http.MethodDelete, path, nil, nil, nil)
}

// ListArticles calls GET /open-api/subscriptions/{mpId}/articles.
func (c *Client) ListArticles(mpID int64, page, pageSize int) (*ArticleList, error) {
	v := url.Values{}
	if page > 0 {
		v.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	path := fmt.Sprintf("/open-api/subscriptions/%d/articles", mpID)
	var out ArticleList
	if err := c.do(http.MethodGet, path, v, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// VerifyAuth pings GET /open-api/subscriptions?pageSize=1 to confirm the
// Feed Key is valid. Returns an errs.Error with CodeAuth on 401.
func (c *Client) VerifyAuth() error {
	_, err := c.ListSubscriptions("", 1, 1)
	return err
}

// ---------------------------------------------------------------------------
// API methods (X)
// ---------------------------------------------------------------------------
//
// 只暴露读类端点（posts / articles）。X 账号搜索与订阅 / 取消订阅仅在
// Web 控制台提供，本 client 不再持有对应方法。

// XListPosts calls GET /open-api/x/:xUserId/posts.
func (c *Client) XListPosts(xUserID string, page, pageSize int) (*XPostList, error) {
	v := url.Values{}
	if page > 0 {
		v.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	path := "/open-api/x/" + url.PathEscape(xUserID) + "/posts"
	var out XPostList
	if err := c.do(http.MethodGet, path, v, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// XListArticles calls GET /open-api/x/:xUserId/articles.
func (c *Client) XListArticles(xUserID string, page, pageSize int) (*XArticleList, error) {
	v := url.Values{}
	if page > 0 {
		v.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	path := "/open-api/x/" + url.PathEscape(xUserID) + "/articles"
	var out XArticleList
	if err := c.do(http.MethodGet, path, v, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------------------------------------------------------------------------
// transport
// ---------------------------------------------------------------------------

type apiError struct {
	ErrorMessage string `json:"errorMessage"`
}

// do is the context-less variant — equivalent to doCtx(context.Background(),...).
func (c *Client) do(method, path string, query url.Values, body any, out any) error {
	return c.doCtx(context.Background(), method, path, query, body, out)
}

// doCtx executes a request. Retry policy:
//   - transport error / 5xx: one retry after a short backoff (unchanged)
//   - 429: wait for Retry-After (<= opts.MaxRetryWait) then retry once; a
//     Retry-After beyond the cap, or a second 429, fails immediately with a
//     KindRateLimited error. We never hammer a locked key.
//
// Every attempt first passes the cross-process throttle (min interval).
// body, if non-nil, is JSON-encoded; out, if non-nil, decodes the response body.
// ctx is plumbed onto every request so callers can cancel mid-flight.
func (c *Client) doCtx(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	full := c.baseURL + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return errs.Wrap(errs.CodeGeneric, fmt.Errorf("marshaling request: %w", err))
		}
	}

	const maxAttempts = 2 // initial + 1 retry
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Bail early if the caller already canceled.
		if err := ctx.Err(); err != nil {
			return errs.Wrap(errs.CodeGeneric, err)
		}

		if err := c.throttle.wait(ctx); err != nil {
			return errs.Wrap(errs.CodeGeneric, err)
		}

		req, err := http.NewRequestWithContext(ctx, method, full, bytes.NewReader(bodyBytes))
		if err != nil {
			return errs.Wrap(errs.CodeGeneric, err)
		}
		if c.feedKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.feedKey)
		}
		req.Header.Set("User-Agent", c.userAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Context cancel surfaces here — don't retry, don't mask.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return errs.Wrap(errs.CodeGeneric, ctxErr)
			}
			lastErr = err
			// transport error → retry once
			if attempt < maxAttempts {
				if err := sleepCtx(ctx, backoffFor(attempt)); err != nil {
					return errs.Wrap(errs.CodeGeneric, err)
				}
				continue
			}
			return errs.Newf(errs.CodeUpstreamDown, "网络错误：%s", err.Error())
		}

		// Buffer body for both decode and error paths.
		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < maxAttempts {
				if err := sleepCtx(ctx, backoffFor(attempt)); err != nil {
					return errs.Wrap(errs.CodeGeneric, err)
				}
				continue
			}
			return errs.Wrap(errs.CodeGeneric, fmt.Errorf("reading response: %w", readErr))
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			rl := parseRateLimit(resp.Header, time.Now())
			if attempt < maxAttempts && rl.retryAfter <= c.opts.MaxRetryWait {
				if err := sleepCtx(ctx, rl.retryAfter); err != nil {
					return errs.Wrap(errs.CodeGeneric, err)
				}
				continue
			}
			return mapRateLimitError(respBody, rl, c.opts.MaxRetryWait)
		}

		if resp.StatusCode >= 500 {
			if attempt < maxAttempts {
				if err := sleepCtx(ctx, backoffFor(attempt)); err != nil {
					return errs.Wrap(errs.CodeGeneric, err)
				}
				continue
			}
			return mapHTTPError(resp.StatusCode, respBody)
		}

		if resp.StatusCode >= 400 {
			return mapHTTPError(resp.StatusCode, respBody)
		}

		// 2xx
		if out != nil && len(respBody) > 0 && resp.StatusCode != http.StatusNoContent {
			if err := json.Unmarshal(respBody, out); err != nil {
				return errs.Wrap(errs.CodeGeneric, fmt.Errorf("decoding response: %w", err))
			}
		}
		return nil
	}
	if lastErr != nil {
		return errs.Wrap(errs.CodeUpstreamDown, lastErr)
	}
	return errs.Newf(errs.CodeGeneric, "请求失败")
}

func backoffFor(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 500 * time.Millisecond
	default:
		return time.Second
	}
}

// sleepCtx sleeps for d unless ctx is canceled first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// rateLimitInfo is what a 429 told us.
type rateLimitInfo struct {
	retryAfter time.Duration // how long the server asked us to wait
	resetAt    time.Time     // absolute unlock time (zero if unknown)
	limit      int           // X-RateLimit-Limit (0 if absent)
}

// parseRateLimit reads Retry-After (seconds or HTTP-date) and X-RateLimit-*
// from a 429 response. Missing / malformed headers fall back to
// fallbackRetryAfter so a header-less 429 still gets one gentle retry.
func parseRateLimit(h http.Header, now time.Time) rateLimitInfo {
	info := rateLimitInfo{}
	haveRetryAfter := false
	if v := strings.TrimSpace(h.Get("Retry-After")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			info.retryAfter = time.Duration(n) * time.Second
			haveRetryAfter = true
		} else if t, err := http.ParseTime(v); err == nil {
			if d := t.Sub(now); d > 0 {
				info.retryAfter = d
			}
			haveRetryAfter = true
		}
	}
	if v := strings.TrimSpace(h.Get("X-RateLimit-Reset")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			info.resetAt = time.Unix(n, 0)
		}
	}
	switch {
	case !haveRetryAfter && !info.resetAt.IsZero():
		// Retry-After unusable but an absolute reset is known → derive the wait.
		if d := info.resetAt.Sub(now); d > 0 {
			info.retryAfter = d
		}
	case !haveRetryAfter:
		info.retryAfter = fallbackRetryAfter
	}
	if info.resetAt.IsZero() && info.retryAfter > 0 {
		info.resetAt = now.Add(info.retryAfter)
	}
	if v := strings.TrimSpace(h.Get("X-RateLimit-Limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			info.limit = n
		}
	}
	return info
}

// mapRateLimitError builds the user-facing 429 error. Exit code stays 1 (the
// CLI contract has no dedicated rate-limit code); Kind = rate_limited lets
// scripts branch on `error.kind` in JSON mode.
func mapRateLimitError(body []byte, rl rateLimitInfo, maxWait time.Duration) error {
	msg := strings.TrimSpace(string(body))
	if len(body) > 0 {
		var apiErr apiError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.ErrorMessage != "" {
			msg = apiErr.ErrorMessage
		}
	}
	if msg == "" {
		msg = "Too Many Requests"
	}
	limit := rl.limit
	if limit == 0 {
		limit = ServerRequestsPerMinute
	}
	var b strings.Builder
	fmt.Fprintf(&b, "请求被限流（HTTP 429）：%s。", msg)
	if !rl.resetAt.IsZero() {
		fmt.Fprintf(&b, "预计 %s（本地时间，约 %s 后）解除，", rl.resetAt.Local().Format("15:04:05"), humanDuration(rl.retryAfter))
	}
	if rl.retryAfter > maxWait {
		fmt.Fprintf(&b, "超过 --max-retry-wait（%s）未自动等待。", humanDuration(maxWait))
	} else {
		b.WriteString("已按 Retry-After 等待并重试一次仍被拒。")
	}
	fmt.Fprintf(&b, "服务端限制每分钟 %d 次请求；请减少并发、串行调用并加大请求间隔后再试", limit)
	return &errs.Error{
		Code:       errs.CodeGeneric,
		HTTPStatus: http.StatusTooManyRequests,
		Kind:       errs.KindRateLimited,
		Message:    b.String(),
		Cause:      errors.New(msg),
	}
}

func humanDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	sec := int(d.Seconds()) % 60
	if sec == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, sec)
}

// mapHTTPError converts a non-2xx response into a typed *errs.Error.
func mapHTTPError(status int, body []byte) error {
	msg := strings.TrimSpace(string(body))
	if len(body) > 0 {
		var apiErr apiError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.ErrorMessage != "" {
			msg = apiErr.ErrorMessage
		}
	}
	code := errs.CodeGeneric
	prefix := ""
	switch {
	case status == http.StatusUnauthorized:
		code = errs.CodeAuth
		prefix = "鉴权失败"
	case status == http.StatusNotFound:
		code = errs.CodeNotFound
		prefix = "未找到"
	case status == http.StatusBadRequest:
		code = errs.CodeArgs
		prefix = "请求参数错误"
	case status == http.StatusServiceUnavailable:
		code = errs.CodeUpstreamDown
		prefix = "上游服务不可用"
	case status >= 500:
		code = errs.CodeUpstreamDown
		prefix = "服务器错误"
	}
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", status)
	}
	full := msg
	if prefix != "" {
		full = fmt.Sprintf("%s（HTTP %d）：%s", prefix, status, msg)
	}
	return &errs.Error{Code: code, HTTPStatus: status, Message: full, Cause: errors.New(msg)}
}
