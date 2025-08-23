package middleware

import (
	"net/http"
	"time"

	"go-falcon/pkg/handlers"

	"go.opentelemetry.io/otel/attribute"
)

// Middleware combines all scheduler-specific middleware
type Middleware struct {
	validation *ValidationMiddleware
	auth       *AuthMiddleware
}

// New creates a new middleware instance
func New(authMiddleware *AuthMiddleware) *Middleware {
	return &Middleware{
		validation: NewValidationMiddleware(),
		auth:       authMiddleware,
	}
}

// GetAuthMiddleware returns the auth middleware
func (m *Middleware) GetAuthMiddleware() *AuthMiddleware {
	return m.auth
}

// RequestLogging adds request logging for scheduler endpoints
func (m *Middleware) RequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Skip health check logging to reduce noise
		if r.URL.Path == "/scheduler/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Create response wrapper to capture status code
		wrapped := handlers.NewResponseWrapper(w)
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		
		// Log request details
		handlers.LogRequest(r, wrapped.StatusCode, duration, map[string]interface{}{
			"module": "scheduler",
			"method": r.Method,
			"path":   r.URL.Path,
		})
	})
}

// RateLimiting applies rate limiting for scheduler endpoints
func (m *Middleware) RateLimiting(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Implement rate limiting based on endpoint
		// Different limits for different operations:
		// - Task creation: 10/minute
		// - Task updates: 30/minute
		// - Task listing: 100/minute
		// - Manual execution: 5/minute
		
		// For now, pass through - implement actual rate limiting later
		next.ServeHTTP(w, r)
	})
}

// Tracing adds OpenTelemetry tracing for scheduler operations
func (m *Middleware) Tracing(operationName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span, r := handlers.StartHTTPSpan(r, operationName,
				attribute.String("service", "scheduler"),
				attribute.String("operation", operationName),
				attribute.String("method", r.Method),
				attribute.String("path", r.URL.Path),
			)
			defer span.End()

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders adds security headers
func (m *Middleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		next.ServeHTTP(w, r)
	})
}

// GetValidationMiddleware returns the validation middleware
func (m *Middleware) GetValidationMiddleware() *ValidationMiddleware {
	return m.validation
}

// ValidateQueryParams provides query parameter validation
func (m *Middleware) ValidateQueryParams(next http.Handler) http.Handler {
	return m.validation.ValidateQueryParams(next)
}