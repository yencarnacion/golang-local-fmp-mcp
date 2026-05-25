package fmp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
		return nil, fmt.Errorf("http url=%s: %w", safeURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body url=%s: %w", safeURL, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{Status: resp.StatusCode, URL: safeURL, Body: string(body)}
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
