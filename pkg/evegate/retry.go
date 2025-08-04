package evegate

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// DefaultRetryClient implements retry logic with exponential backoff
type DefaultRetryClient struct {
	httpClient    *http.Client
	errorLimits   *ESIErrorLimits
	limitsMutex   *sync.RWMutex
}

// NewDefaultRetryClient creates a new default retry client
func NewDefaultRetryClient(httpClient *http.Client, errorLimits *ESIErrorLimits, limitsMutex *sync.RWMutex) *DefaultRetryClient {
	return &DefaultRetryClient{
		httpClient:  httpClient,
		errorLimits: errorLimits,
		limitsMutex: limitsMutex,
	}
}

// DoWithRetry makes an HTTP request with retry logic and proper error handling
func (r *DefaultRetryClient) DoWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Clone request for retry attempts
		reqClone := req.Clone(ctx)

		resp, err = r.httpClient.Do(reqClone)
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries+1, err)
			}

			// Wait before retry for network errors
			backoffDuration := time.Duration(1<<uint(attempt)) * time.Second
			if backoffDuration > 10*time.Second {
				backoffDuration = 10 * time.Second
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffDuration):
				continue
			}
		}

		// Update error limits from headers
		r.updateErrorLimits(resp.Header)

		// Check if we need to retry based on status code
		if resp.StatusCode >= 500 || resp.StatusCode == 420 || resp.StatusCode == 429 {
			resp.Body.Close() // Close body before retry

			if attempt == maxRetries {
				return nil, fmt.Errorf("request failed with status %d after %d attempts", resp.StatusCode, maxRetries+1)
			}

			// Apply backoff for error status codes
			if err := r.backoffForError(ctx, resp.StatusCode, attempt); err != nil {
				return nil, err
			}
			continue
		}

		// Success or non-retryable error
		break
	}

	return resp, nil
}

// updateErrorLimits updates the client's error limit tracking from response headers
func (r *DefaultRetryClient) updateErrorLimits(headers http.Header) {
	r.limitsMutex.Lock()
	defer r.limitsMutex.Unlock()

	// Parse ESI error limit headers
	if remainStr := headers.Get("X-ESI-Error-Limit-Remain"); remainStr != "" {
		if remain, err := strconv.Atoi(remainStr); err == nil {
			r.errorLimits.Remain = remain
		}
	}

	if resetStr := headers.Get("X-ESI-Error-Limit-Reset"); resetStr != "" {
		if reset, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			r.errorLimits.Reset = time.Unix(reset, 0)
		}
	}

	if windowStr := headers.Get("X-ESI-Error-Limit-Window"); windowStr != "" {
		if window, err := strconv.Atoi(windowStr); err == nil {
			r.errorLimits.Window = window
		}
	}
}

// backoffForError implements exponential backoff based on HTTP status codes
func (r *DefaultRetryClient) backoffForError(ctx context.Context, statusCode int, attempt int) error {
	var backoffDuration time.Duration

	switch {
	case statusCode == 420: // ESI-specific rate limit
		// Use longer backoff for rate limiting
		backoffDuration = time.Duration(1<<uint(attempt)) * time.Minute
		if backoffDuration > 10*time.Minute {
			backoffDuration = 10 * time.Minute
		}
	case statusCode >= 500: // Server errors
		// Exponential backoff for server errors
		backoffDuration = time.Duration(1<<uint(attempt)) * time.Second
		if backoffDuration > 30*time.Second {
			backoffDuration = 30 * time.Second
		}
	case statusCode == 429: // Too Many Requests
		// Standard rate limit backoff
		backoffDuration = time.Duration(1<<uint(attempt)) * time.Second
		if backoffDuration > 60*time.Second {
			backoffDuration = 60 * time.Second
		}
	default:
		return nil // No backoff needed
	}

	slog.WarnContext(ctx, "ESI error requires backoff",
		"status_code", statusCode,
		"attempt", attempt,
		"backoff_duration", backoffDuration.String())

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(backoffDuration):
		return nil
	}
}

