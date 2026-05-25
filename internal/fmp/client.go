package fmp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client is a thin wrapper around an HTTP client targeting the FMP API.
// It transparently appends the API key as ?apikey=... on every request.
type Client struct {
	BaseURL   string // e.g. https://financialmodelingprep.com
	APIPath   string // e.g. /stable
	APIKey    string
	UserAgent string
	HTTP      *http.Client
	limiter   *adaptiveLimiter
}

// New creates a configured client.
func New(baseURL, apiPath, apiKey, userAgent string, timeout time.Duration) *Client {
	if userAgent == "" {
		userAgent = "golang-local-fmp-mcp/0.1"
	}
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		APIPath:   "/" + strings.Trim(apiPath, "/"),
		APIKey:    apiKey,
		UserAgent: userAgent,
		HTTP:      &http.Client{Timeout: timeout},
		limiter:   newAdaptiveLimiter(),
	}
}

// APIError is returned when FMP responds with a non-2xx status.
type APIError struct {
	Status int
	URL    string
	Body   string
}

func (e *APIError) Error() string {
	body := e.Body
	if len(body) > 500 {
		body = body[:500] + "..."
	}
	return fmt.Sprintf("fmp api error: status=%d url=%s body=%s", e.Status, e.URL, body)
}

// Get issues a GET to <BaseURL><APIPath>/<path> with the given query parameters.
// Returns the response body parsed as JSON (any: object, array, etc.).
// The apikey query parameter is appended automatically and never logged.
func (c *Client) Get(ctx context.Context, path string, params url.Values) (any, error) {
	rate := 0
	if c.limiter != nil {
		var err error
		rate, err = c.limiter.Wait(ctx)
		if err != nil {
			return nil, err
		}
	}

	if params == nil {
		params = url.Values{}
	}
	// Always inject apikey last so it cannot be overridden by callers.
	params.Set("apikey", c.APIKey)

	endpoint := c.BaseURL + c.APIPath + "/" + strings.TrimLeft(path, "/")
	full := endpoint + "?" + params.Encode()
	safeURL := redactAPIKey(full)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return nil, fmt.Errorf("build request url=%s: %w", safeURL, err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		err = fmt.Errorf("http url=%s: %w", safeURL, err)
		if c.limiter != nil {
			c.limiter.RecordError(rate, err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body url=%s: %w", safeURL, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := &APIError{Status: resp.StatusCode, URL: safeURL, Body: string(body)}
		if c.limiter != nil {
			c.limiter.RecordError(rate, err)
		}
		return nil, err
	}

	if len(body) == 0 {
		return nil, nil
	}

	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		// Some FMP endpoints (xlsx, csv) return non-JSON. Surface as a string.
		return string(body), nil
	}
	return v, nil
}

var defaultRateTiersPerMinute = []int{750, 299, 249, 199, 149, 99, 59, 29}

type adaptiveLimiter struct {
	mu        sync.Mutex
	tiers     []int
	tierIndex int
	next      time.Time
}

func newAdaptiveLimiter() *adaptiveLimiter {
	tiers := append([]int(nil), defaultRateTiersPerMinute...)
	return &adaptiveLimiter{tiers: tiers}
}

func (l *adaptiveLimiter) Wait(ctx context.Context) (int, error) {
	l.mu.Lock()
	now := time.Now()
	waitUntil := l.next
	if waitUntil.Before(now) {
		waitUntil = now
	}
	rate := l.currentRateLocked()
	l.next = waitUntil.Add(intervalForRate(rate))
	l.mu.Unlock()

	timer := time.NewTimer(time.Until(waitUntil))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return rate, ctx.Err()
	case <-timer.C:
		return rate, nil
	}
}

func (l *adaptiveLimiter) RecordError(rate int, err error) {
	if !isRateLimitError(err) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.tierIndex >= len(l.tiers)-1 {
		return
	}
	if rate != l.currentRateLocked() {
		return
	}

	oldRate := l.tiers[l.tierIndex]
	l.tierIndex++
	newRate := l.tiers[l.tierIndex]
	log.Printf("fmp rate limiter reducing request rate from %d/min to %d/min after upstream rate-limit error", oldRate, newRate)
}

func (l *adaptiveLimiter) currentRateLocked() int {
	return l.tiers[l.tierIndex]
}

func intervalForRate(perMinute int) time.Duration {
	if perMinute <= 0 {
		perMinute = 1
	}
	return time.Minute / time.Duration(perMinute)
}

func isRateLimitError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Status == http.StatusTooManyRequests {
		return true
	}
	body := strings.ToLower(apiErr.Body)
	return strings.Contains(body, "rate limit") ||
		strings.Contains(body, "too many requests") ||
		strings.Contains(body, "limit reach") ||
		strings.Contains(body, "limit reached") ||
		strings.Contains(body, "reached your limit")
}

func redactAPIKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	if q.Get("apikey") != "" {
		q.Set("apikey", "REDACTED")
	}
	u.RawQuery = q.Encode()
	return u.String()
}
