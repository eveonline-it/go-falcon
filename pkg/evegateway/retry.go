package evegateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go-falcon/internal/auth/models"
)

// AuthContextKey key for storing user info in context (matches pkg/middleware)
type AuthContextKey string

const (
	AuthContextKeyUser = AuthContextKey("authenticated_user")
)

// getAuthenticatedUser retrieves authenticated user from context (avoids import cycle with pkg/middleware)
func getAuthenticatedUser(ctx context.Context) *models.AuthenticatedUser {
	if user, ok := ctx.Value(AuthContextKeyUser).(*models.AuthenticatedUser); ok {
		return user
	}
	return nil
}

// DefaultRetryClient implements retry logic with exponential backoff
type DefaultRetryClient struct {
	httpClient  *http.Client
	errorLimits *ESIErrorLimits
	limitsMutex *sync.RWMutex
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
		// All errors except 404 (Not Found) count against the error limit
		// 403 (Forbidden) DOES count against the limit
		if resp.StatusCode != 404 {
			r.updateErrorLimitsWithContext(resp.Header, ctx, req)
		}

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

// updateErrorLimits updates the client's error limit tracking from response headers (legacy function for backward compatibility)
func (r *DefaultRetryClient) updateErrorLimits(headers http.Header) {
	r.updateErrorLimitsWithContext(headers, nil, nil)
}

// updateErrorLimitsWithContext updates the client's error limit tracking from response headers with critical error logging
func (r *DefaultRetryClient) updateErrorLimitsWithContext(headers http.Header, ctx context.Context, req *http.Request) {
	r.limitsMutex.Lock()
	defer r.limitsMutex.Unlock()

	// Parse ESI error limit headers
	if remainStr := headers.Get("X-ESI-Error-Limit-Remain"); remainStr != "" {
		if remain, err := strconv.Atoi(remainStr); err == nil {
			r.errorLimits.Remain = remain

			// Log critical error when X-ESI-Error-Limit-Remain header is present and indicates potential issues
			// This helps track ESI rate limiting events and the users/endpoints that trigger them
			if remain <= 50 { // Critical threshold - log when remaining errors are low
				var userID string = "unknown"
				var characterID int = 0
				var characterName string = "unknown"
				var endpoint string = "unknown"
				var method string = "unknown"

				// Extract user information from context if available
				if ctx != nil {
					if user := getAuthenticatedUser(ctx); user != nil {
						userID = user.UserID
						characterID = user.CharacterID
						characterName = user.CharacterName
					}
				}

				// Extract endpoint information from request if available
				if req != nil {
					endpoint = req.URL.String()
					method = req.Method
				}

				slog.ErrorContext(ctx, "ESI Error Limit Warning - X-ESI-Error-Limit-Remain triggered",
					"x_esi_error_limit_remain", remain,
					"user_id", userID,
					"character_id", characterID,
					"character_name", characterName,
					"endpoint", endpoint,
					"method", method,
					"reset_time", r.errorLimits.Reset.Format(time.RFC3339),
					"window", r.errorLimits.Window,
				)
			}
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
