package fmp

import (
	"errors"
	"net/http"
	"testing"
)

func TestRateLimiterDowngradesOnRateLimitErrors(t *testing.T) {
	limiter := newAdaptiveLimiter()

	limiter.RecordError(750, &APIError{Status: http.StatusTooManyRequests, Body: "too many requests"})
	if got, want := limiter.tiers[limiter.tierIndex], 299; got != want {
		t.Fatalf("rate after 429 = %d, want %d", got, want)
	}

	limiter.RecordError(299, &APIError{Status: http.StatusForbidden, Body: "rate limit reached"})
	if got, want := limiter.tiers[limiter.tierIndex], 249; got != want {
		t.Fatalf("rate after rate-limit body = %d, want %d", got, want)
	}
}

func TestRateLimiterIgnoresNonRateLimitErrors(t *testing.T) {
	limiter := newAdaptiveLimiter()

	limiter.RecordError(750, &APIError{Status: http.StatusBadRequest, Body: "invalid symbol"})
	limiter.RecordError(750, errors.New("network reset"))

	if got, want := limiter.tiers[limiter.tierIndex], 750; got != want {
		t.Fatalf("rate after non-rate-limit errors = %d, want %d", got, want)
	}
}

func TestRateLimiterIgnoresStaleRateLimitErrors(t *testing.T) {
	limiter := newAdaptiveLimiter()

	limiter.RecordError(750, &APIError{Status: http.StatusTooManyRequests})
	limiter.RecordError(750, &APIError{Status: http.StatusTooManyRequests})

	if got, want := limiter.tiers[limiter.tierIndex], 299; got != want {
		t.Fatalf("rate after stale 750/min error = %d, want %d", got, want)
	}
}
