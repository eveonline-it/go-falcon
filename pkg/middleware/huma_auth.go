package middleware

import (
	"context"
	"strings"

	"go-falcon/internal/auth/models"
	"go-falcon/pkg/config"

	"github.com/danielgtaylor/huma/v2"
)

// HumaAuthContextKey key for storing user info in Huma context
type HumaAuthContextKey string

const (
	HumaAuthContextKeyUser = HumaAuthContextKey("authenticated_user")
)

// JWTValidator interface for JWT validation
type JWTValidator interface {
	ValidateJWT(token string) (*models.AuthenticatedUser, error)
}

// HumaAuthMiddleware provides authentication utilities for Huma operations
type HumaAuthMiddleware struct {
	jwtValidator JWTValidator
}

// NewHumaAuthMiddleware creates a new Huma authentication middleware
func NewHumaAuthMiddleware(validator JWTValidator) *HumaAuthMiddleware {
	return &HumaAuthMiddleware{
		jwtValidator: validator,
	}
}

// ValidateAuthFromHeaders validates authentication from request headers
func (m *HumaAuthMiddleware) ValidateAuthFromHeaders(authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
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
func (m *HumaAuthMiddleware) ValidateOptionalAuthFromHeaders(authHeader, cookieHeader string) *models.AuthenticatedUser {
	user, _ := m.ValidateAuthFromHeaders(authHeader, cookieHeader)
	return user
}

// ValidateScopesFromHeaders validates authentication and required EVE scopes
func (m *HumaAuthMiddleware) ValidateScopesFromHeaders(authHeader, cookieHeader string, requiredScopes ...string) (*models.AuthenticatedUser, error) {
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
func (m *HumaAuthMiddleware) ExtractTokenFromHeaders(authHeader string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

// ExtractTokenFromCookie extracts JWT token from cookie header string
func (m *HumaAuthMiddleware) ExtractTokenFromCookie(cookieHeader string) string {
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
func (m *HumaAuthMiddleware) ValidateToken(token string) (*models.AuthenticatedUser, error) {
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

// GetHumaAuthenticatedUser retrieves authenticated user from standard context
func GetHumaAuthenticatedUser(ctx context.Context) *models.AuthenticatedUser {
	if user, ok := ctx.Value(HumaAuthContextKeyUser).(*models.AuthenticatedUser); ok {
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
	
	// Create cookie similar to the traditional handler
	cookie := "falcon_auth_token=" + token + "; Path=/"
	if cookieDomain != "" {
		cookie += "; Domain=" + cookieDomain
	}
	cookie += "; Max-Age=86400; HttpOnly; Secure; SameSite=Lax"
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