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

// Subscription is one row of /open-api/subscriptions.
type Subscription struct {
	MpID            int64  `json:"mpId"`
	MpName          string `json:"mpName"`
	MpAvatarURL     string `json:"mpAvatarUrl"`
	CreatedAt       int64  `json:"createdAt"`
	MpLastArticleAt int64  `json:"mpLastArticleAt"`
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
// API methods
// ---------------------------------------------------------------------------

// ListSubscriptions calls GET /open-api/subscriptions.
func (c *Client) ListSubscriptions(q string, page, pageSize int) (*SubscriptionList, error) {
	v := url.Values{}
	if q != "" {
		v.Set("q", q)
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
// transport
// ---------------------------------------------------------------------------

type apiError struct {
	ErrorMessage string `json:"errorMessage"`
}

// do executes a request with one retry on 429 / 5xx. body, if non-nil, is
// JSON-encoded; out, if non-nil, decodes the response body.
func (c *Client) do(method, path string, query url.Values, body any, out any) error {
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
		req, err := http.NewRequest(method, full, bytes.NewReader(bodyBytes))
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
