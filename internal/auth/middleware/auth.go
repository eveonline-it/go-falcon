package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"go-falcon/internal/auth/models"
	"go-falcon/pkg/handlers"
)

// AuthContextKey key for storing user info in request context
type AuthContextKey string

const (
	AuthContextKeyUser = AuthContextKey("user")
)

// AuthMiddleware provides authentication middleware functionality
type AuthMiddleware struct {
	jwtValidator JWTValidator
}

// JWTValidator interface for JWT validation
type JWTValidator interface {
	ValidateJWT(token string) (*models.AuthenticatedUser, error)
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(validator JWTValidator) *AuthMiddleware {
	return &AuthMiddleware{
		jwtValidator: validator,
	}
}

// RequireAuth ensures the user is authenticated via JWT
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := m.extractAndValidateJWT(r)
		if err != nil {
			slog.Warn("Authentication failed", "error", err.Error())
			handlers.UnauthorizedResponse(w)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), AuthContextKeyUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth adds user context if JWT is present and valid
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := m.extractAndValidateJWT(r)
		if err != nil {
			// Log but don't fail for optional auth
			slog.Debug("Optional auth failed", "error", err.Error())
		}

		// Add user to context if available
		ctx := r.Context()
		if user != nil {
			ctx = context.WithValue(ctx, AuthContextKeyUser, user)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireScopes ensures the user has specific EVE Online scopes
func (m *AuthMiddleware) RequireScopes(requiredScopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetAuthenticatedUser(r)
			if user == nil {
				handlers.UnauthorizedResponse(w)
				return
			}

			// Check if user has required scopes
			if !hasRequiredScopes(user.Scopes, requiredScopes) {
				handlers.ForbiddenResponse(w, "Insufficient EVE Online permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractAndValidateJWT extracts JWT from request and validates it
func (m *AuthMiddleware) extractAndValidateJWT(r *http.Request) (*models.AuthenticatedUser, error) {
	// Try to get JWT from cookie or Authorization header
	var jwtToken string

	// Try cookie first
	if cookie, err := r.Cookie("falcon_auth_token"); err == nil {
		jwtToken = cookie.Value
	} else {
		// Try Authorization header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			jwtToken = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if jwtToken == "" {
		return nil, ErrNoToken
	}

	// Validate JWT using the injected validator
	user, err := m.jwtValidator.ValidateJWT(jwtToken)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetAuthenticatedUser retrieves authenticated user from request context
func GetAuthenticatedUser(r *http.Request) *models.AuthenticatedUser {
	if user, ok := r.Context().Value(AuthContextKeyUser).(*models.AuthenticatedUser); ok {
		return user
	}
	return nil
}

// hasRequiredScopes checks if user has all required EVE scopes
func hasRequiredScopes(userScopes string, requiredScopes []string) bool {
	if len(requiredScopes) == 0 {
		return true
	}

	userScopeList := strings.Fields(userScopes)
	userScopeMap := make(map[string]bool)
	for _, scope := range userScopeList {
		userScopeMap[scope] = true
	}

	for _, required := range requiredScopes {
		if !userScopeMap[required] {
			return false
		}
	}

	return true
}

// Common middleware errors
var (
	ErrNoToken      = NewAuthError("no authentication token provided")
	ErrInvalidToken = NewAuthError("invalid authentication token")
)

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
