package middleware

import (
	"net/http"
	"time"

	"go-falcon/internal/groups/services"
	"go-falcon/pkg/handlers"

	"go.opentelemetry.io/otel/attribute"
)

// Middleware combines all groups-specific middleware
type Middleware struct {
	permission *PermissionMiddleware
	validation *ValidationMiddleware
}

// New creates a new middleware instance
func New(granularService *services.GranularPermissionService, legacyService *services.PermissionService) *Middleware {
	return &Middleware{
		permission: NewPermissionMiddleware(granularService, legacyService),
		validation: NewValidationMiddleware(),
	}
}

// RequestLogging adds request logging for groups endpoints
func (m *Middleware) RequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Skip health check logging to reduce noise
		if r.URL.Path == "/groups/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Create response wrapper to capture status code
		wrapped := handlers.NewResponseWrapper(w)
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		
		// Log request details
		handlers.LogRequest(r, wrapped.StatusCode, duration, map[string]interface{}{
			"module": "groups",
			"method": r.Method,
			"path":   r.URL.Path,
		})
	})
}

// RateLimiting applies rate limiting for groups endpoints
func (m *Middleware) RateLimiting(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Implement rate limiting based on endpoint
		// Different limits for different operations:
		// - Group creation: 5/minute
		// - Group updates: 10/minute
		// - Group listing: 60/minute
		// - Permission checks: 100/minute
		// - Admin operations: 20/minute
		
		// For now, pass through - implement actual rate limiting later
		next.ServeHTTP(w, r)
	})
}

// Tracing adds OpenTelemetry tracing for groups operations
func (m *Middleware) Tracing(operationName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span, r := handlers.StartHTTPSpan(r, operationName,
				attribute.String("service", "groups"),
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

// CORS adds CORS headers for cross-origin requests
func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// GetPermissionMiddleware returns the permission middleware
func (m *Middleware) GetPermissionMiddleware() *PermissionMiddleware {
	return m.permission
}

// GetValidationMiddleware returns the validation middleware
func (m *Middleware) GetValidationMiddleware() *ValidationMiddleware {
	return m.validation
}

// Convenience methods for common middleware combinations

// RequireGranularPermission is a convenience method for granular permission checks
func (m *Middleware) RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler {
	return m.permission.RequireGranularPermission(service, resource, action)
}

// OptionalGranularPermission is a convenience method for optional granular permission checks
func (m *Middleware) OptionalGranularPermission(service, resource, action string) func(http.Handler) http.Handler {
	return m.permission.OptionalGranularPermission(service, resource, action)
}

// RequireLegacyPermission is a convenience method for legacy permission checks
func (m *Middleware) RequireLegacyPermission(resource string, actions ...string) func(http.Handler) http.Handler {
	return m.permission.RequireLegacyPermission(resource, actions...)
}

// RequireSuperAdmin is a convenience method for super admin checks
func (m *Middleware) RequireSuperAdmin() func(http.Handler) http.Handler {
	return m.permission.RequireSuperAdmin()
}

// ValidateQueryParams is a convenience method for query parameter validation
func (m *Middleware) ValidateQueryParams(next http.Handler) http.Handler {
	return m.validation.ValidateQueryParams(next)
}