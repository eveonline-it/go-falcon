package middleware

import (
	"context"
	"fmt"
	"strings"

	"go-falcon/internal/auth/models"
	"go-falcon/pkg/config"

	"github.com/danielgtaylor/huma/v2"
)

// AuthContextKey key for storing user info in context
type AuthContextKey string

const (
	AuthContextKeyUser = AuthContextKey("authenticated_user")
)

// JWTValidator interface for JWT validation
type JWTValidator interface {
	ValidateJWT(token string) (*models.AuthenticatedUser, error)
}

// AuthMiddleware provides authentication utilities for API operations
type AuthMiddleware struct {
	jwtValidator JWTValidator
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(validator JWTValidator) *AuthMiddleware {
	return &AuthMiddleware{
		jwtValidator: validator,
	}
}

// ValidateAuthFromHeaders validates authentication from request headers
func (m *AuthMiddleware) ValidateAuthFromHeaders(authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	// Try to get token from Authorization header first
	token := m.ExtractTokenFromHeaders(authHeader)

	// If not found, try cookie
	if token == "" && cookieHeader != "" {
		token = m.ExtractTokenFromCookie(cookieHeader)
	}

	if token == "" {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	user, err := m.ValidateToken(token)
	if err != nil {
		return nil, huma.Error401Unauthorized("Invalid authentication token", err)
	}

	return user, nil
}

// ValidateOptionalAuthFromHeaders validates optional authentication from request headers
func (m *AuthMiddleware) ValidateOptionalAuthFromHeaders(authHeader, cookieHeader string) *models.AuthenticatedUser {
	user, _ := m.ValidateAuthFromHeaders(authHeader, cookieHeader)
	return user
}

// ValidateScopesFromHeaders validates authentication and required EVE scopes
func (m *AuthMiddleware) ValidateScopesFromHeaders(authHeader, cookieHeader string, requiredScopes ...string) (*models.AuthenticatedUser, error) {
	user, err := m.ValidateAuthFromHeaders(authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}

	// Check if user has required scopes
	if !hasRequiredScopes(user.Scopes, requiredScopes) {
		return nil, huma.Error403Forbidden("Insufficient EVE Online permissions")
	}

	return user, nil
}

// ExtractTokenFromHeaders extracts JWT token from Authorization header string
func (m *AuthMiddleware) ExtractTokenFromHeaders(authHeader string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

// ExtractTokenFromCookie extracts JWT token from cookie header string
func (m *AuthMiddleware) ExtractTokenFromCookie(cookieHeader string) string {
	// Parse cookie header to find falcon_auth_token
	cookies := strings.Split(cookieHeader, ";")
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		if strings.HasPrefix(cookie, "falcon_auth_token=") {
			return strings.TrimPrefix(cookie, "falcon_auth_token=")
		}
	}
	return ""
}

// ValidateToken validates a JWT token string and returns the authenticated user
func (m *AuthMiddleware) ValidateToken(token string) (*models.AuthenticatedUser, error) {
	if token == "" {
		return nil, &AuthError{message: "no authentication token provided"}
	}

	// Validate JWT using the injected validator
	user, err := m.jwtValidator.ValidateJWT(token)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetAuthenticatedUser retrieves authenticated user from standard context
func GetAuthenticatedUser(ctx context.Context) *models.AuthenticatedUser {
	if user, ok := ctx.Value(AuthContextKeyUser).(*models.AuthenticatedUser); ok {
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

// CreateAuthCookieHeader creates a Set-Cookie header string for authentication
func CreateAuthCookieHeader(token string) string {
	// Get cookie domain from config
	cookieDomain := config.GetCookieDomain()

	// Get cookie duration from config and convert to seconds for Max-Age
	cookieDuration := config.GetCookieDuration()
	maxAge := int(cookieDuration.Seconds())

	// Create cookie similar to the traditional handler
	cookie := "falcon_auth_token=" + token + "; Path=/"
	if cookieDomain != "" {
		cookie += "; Domain=" + cookieDomain
	}
	cookie += fmt.Sprintf("; Max-Age=%d; HttpOnly; Secure; SameSite=Lax", maxAge)
	return cookie
}

// CreateClearCookieHeader creates a Set-Cookie header string to clear the auth cookie
func CreateClearCookieHeader() string {
	// Get cookie domain from config
	cookieDomain := config.GetCookieDomain()

	// Clear cookie by setting it to empty with past expiration
	cookie := "falcon_auth_token=; Path=/"
	if cookieDomain != "" {
		cookie += "; Domain=" + cookieDomain
	}
	cookie += "; Max-Age=0; HttpOnly; Secure; SameSite=Lax"
	return cookie
}
