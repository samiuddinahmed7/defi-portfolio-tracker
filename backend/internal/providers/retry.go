// Package providers — this file adds retry and timeout helpers to make
// external API calls more resilient. Provider implementations call these
// helpers directly rather than duplicating retry logic.
package providers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// RetryConfig holds settings for the retry helper.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultRetryConfig is a conservative configuration suitable for external
// blockchain APIs that have rate limits.
var DefaultRetryConfig = RetryConfig{
	MaxAttempts: 3,
	BaseDelay:   500 * time.Millisecond,
	MaxDelay:    5 * time.Second,
}

// IsRetryableHTTPStatus returns true for status codes where retrying makes
// sense (rate limits, server errors, gateway timeouts).
func IsRetryableHTTPStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// RetryableError wraps an error with a status code for retry decisions.
type RetryableError struct {
	Err        error
	StatusCode int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("status %d: %v", e.StatusCode, e.Err)
}

func (e *RetryableError) Unwrap() error { return e.Err }

// WithRetry calls fn up to cfg.MaxAttempts times, sleeping between attempts
// with simple linear backoff. It stops early when:
//   - fn succeeds (err == nil)
//   - the context is cancelled
//   - the error is not retryable
func WithRetry(ctx context.Context, log *slog.Logger, cfg RetryConfig, name string, fn func() error) error {
	var lastErr error
	delay := cfg.BaseDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%s: context cancelled: %w", name, err)
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Check if the error is retryable.
		var retryable *RetryableError
		if errors.As(lastErr, &retryable) && !IsRetryableHTTPStatus(retryable.StatusCode) {
			return lastErr // not worth retrying
		}

		if attempt < cfg.MaxAttempts {
			log.Warn("provider request failed, retrying",
				"provider", name,
				"attempt", attempt,
				"max", cfg.MaxAttempts,
				"delay", delay,
				"err", lastErr,
			)
			select {
			case <-ctx.Done():
				return fmt.Errorf("%s: context cancelled during retry: %w", name, ctx.Err())
			case <-time.After(delay):
			}
			delay *= 2
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
	}

	return fmt.Errorf("%s: all %d attempts failed: %w", name, cfg.MaxAttempts, lastErr)
}
