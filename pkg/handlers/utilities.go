package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// ContextKey type for context keys
type ContextKey string

const (
	// ValidatedRequestKey is the context key for validated requests
	ValidatedRequestKey ContextKey = "validated_request"
	// ValidatedQueryKey is the context key for validated query parameters
	ValidatedQueryKey ContextKey = "validated_query"
)

// WithValidatedRequest stores a validated request in the context
func WithValidatedRequest(ctx context.Context, req interface{}) context.Context {
	return context.WithValue(ctx, ValidatedRequestKey, req)
}

// GetValidatedRequest retrieves a validated request from the context
func GetValidatedRequest(ctx context.Context) interface{} {
	return ctx.Value(ValidatedRequestKey)
}

// WithValidatedQuery stores validated query parameters in the context
func WithValidatedQuery(ctx context.Context, query interface{}) context.Context {
	return context.WithValue(ctx, ValidatedQueryKey, query)
}

// GetValidatedQuery retrieves validated query parameters from the context
func GetValidatedQuery(ctx context.Context) interface{} {
	return ctx.Value(ValidatedQueryKey)
}

// ParseIntQuery parses an integer from a query string with a default value
func ParseIntQuery(value string, defaultValue int) (int, error) {
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue, err
	}
	return parsed, nil
}

// ParseCommaSeparated parses a comma-separated string into a slice
func ParseCommaSeparated(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ResponseWrapper wraps http.ResponseWriter to capture status codes
type ResponseWrapper struct {
	http.ResponseWriter
	StatusCode int
	Written    bool
}

// NewResponseWrapper creates a new response wrapper
func NewResponseWrapper(w http.ResponseWriter) *ResponseWrapper {
	return &ResponseWrapper{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
		Written:        false,
	}
}

// WriteHeader captures the status code
func (rw *ResponseWrapper) WriteHeader(statusCode int) {
	if !rw.Written {
		rw.StatusCode = statusCode
		rw.Written = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write ensures WriteHeader is called
func (rw *ResponseWrapper) Write(data []byte) (int, error) {
	if !rw.Written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}

// LogRequest logs HTTP request details
func LogRequest(r *http.Request, statusCode int, duration time.Duration, metadata map[string]interface{}) {
	// Skip health check logging to reduce noise
	if r.URL.Path == "/health" {
		return
	}

	fields := []interface{}{
		"method", r.Method,
		"path", r.URL.Path,
		"status", statusCode,
		"duration", duration.String(),
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
	}

	// Add metadata fields
	for key, value := range metadata {
		fields = append(fields, key, value)
	}

	if statusCode >= 400 {
		// Log errors as warnings
		fields = append(fields, "query", r.URL.RawQuery)
		slog.Warn("HTTP request error", fields...)
	} else {
		slog.Info("HTTP request", fields...)
	}
}

// ValidationErrorResponseFromError converts validator errors to response
func ValidationErrorResponseFromError(w http.ResponseWriter, err error) {
	var errorMessages []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, formatValidationError(fieldError))
		}
	} else {
		errorMessages = append(errorMessages, err.Error())
	}

	ValidationErrorResponse(w, errorMessages)
}

// formatValidationError formats a validation error into a human-readable message
func formatValidationError(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()
	param := err.Param()

	switch tag {
	case "required":
		return field + " is required"
	case "min":
		return field + " must be at least " + param + " characters"
	case "max":
		return field + " must be at most " + param + " characters"
	case "email":
		return field + " must be a valid email address"
	case "oneof":
		return field + " must be one of: " + param
	case "cron":
		return field + " must be a valid cron expression"
	default:
		return field + " is invalid"
	}
}

// AuthContextKey type for authentication context keys
type AuthContextKey string

const (
	// AuthContextKeyUser key for storing user info in request context
	AuthContextKeyUser = AuthContextKey("user")
	// PermissionResultKey key for storing permission results
	PermissionResultKey ContextKey = "permission_result"
)

// AuthenticatedUser represents an authenticated user (simplified interface)
type AuthenticatedUser interface {
	GetCharacterID() int
	GetCharacterName() string
	GetScopes() string
}

// GetCharacterIDFromRequest retrieves character ID from authenticated user in request context
func GetCharacterIDFromRequest(r *http.Request) (int, error) {
	// Try to get user from auth context
	if user := r.Context().Value(AuthContextKeyUser); user != nil {
		if authUser, ok := user.(AuthenticatedUser); ok {
			return authUser.GetCharacterID(), nil
		}
		// Try reflection-based approach for compatibility
		if userValue, ok := user.(interface{ CharacterID() int }); ok {
			return userValue.CharacterID(), nil
		}
	}
	return 0, NewAuthError("no authenticated user found in request context")
}

// GetValidatedRequestFromRequest retrieves a validated request from the request context
func GetValidatedRequestFromRequest(r *http.Request) interface{} {
	return r.Context().Value(ValidatedRequestKey)
}

// GetValidatedQueryFromRequest retrieves validated query parameters from the request context
func GetValidatedQueryFromRequest(r *http.Request) interface{} {
	return r.Context().Value(ValidatedQueryKey)
}

// WithPermissionResult stores a permission result in the context
func WithPermissionResult(ctx context.Context, result interface{}) context.Context {
	return context.WithValue(ctx, PermissionResultKey, result)
}

// GetPermissionResult retrieves a permission result from the context
func GetPermissionResult(ctx context.Context) interface{} {
	return ctx.Value(PermissionResultKey)
}

// AuthError represents an authentication error
type AuthError struct {
	message string
}

func (e *AuthError) Error() string {
	return e.message
}

// NewAuthError creates a new authentication error
func NewAuthError(message string) *AuthError {
	return &AuthError{message: message}
}
