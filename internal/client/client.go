// Package client is a thin HTTP wrapper around the mp2rss Open API.
//
// It handles:
//   - Bearer auth (Feed Key)
//   - User-Agent stamping
//   - 30s timeout with one retry on 429 / 5xx
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
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/areyoubugcoder/mp2rss-cli/internal/version"
)

// DefaultTimeout is the per-request HTTP timeout.
const DefaultTimeout = 30 * time.Second

// Client is an mp2rss Open API client.
type Client struct {
	baseURL    string
	feedKey    string
	userAgent  string
	httpClient *http.Client
}

// New builds a Client. baseURL should NOT have a trailing slash.
func New(baseURL, feedKey string) *Client {
	return NewWithHTTP(baseURL, feedKey, &http.Client{Timeout: DefaultTimeout})
}

// NewWithHTTP is the test-friendly constructor accepting a custom *http.Client.
func NewWithHTTP(baseURL, feedKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL:    baseURL,
		feedKey:    feedKey,
		userAgent:  fmt.Sprintf("mp2rss-cli/%s (%s/%s)", version.String(), runtime.GOOS, runtime.GOARCH),
		httpClient: httpClient,
	}
}

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

// doCtx executes a request with one retry on 429 / 5xx. body, if non-nil, is
// JSON-encoded; out, if non-nil, decodes the response body.
//
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
				time.Sleep(backoffFor(attempt))
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
				time.Sleep(backoffFor(attempt))
				continue
			}
			return errs.Wrap(errs.CodeGeneric, fmt.Errorf("reading response: %w", readErr))
		}

		// Retryable status?
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt < maxAttempts {
				time.Sleep(backoffFor(attempt))
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
