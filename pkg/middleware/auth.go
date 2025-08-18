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
	AuthContextKeyUser        = AuthContextKey("authenticated_user")
	AuthContextKeyAuth        = AuthContextKey("auth_context")
	AuthContextKeyExpanded    = AuthContextKey("expanded_auth_context")
)

// AuthContext represents basic authentication information
type AuthContext struct {
	UserID          string `json:"user_id"`
	PrimaryCharID   int64  `json:"primary_character_id"`
	RequestType     string `json:"request_type"` // "cookie" or "bearer"
	IsAuthenticated bool   `json:"is_authenticated"`
}

// ExpandedAuthContext represents full authentication context with all related identifiers
type ExpandedAuthContext struct {
	*AuthContext
	
	// Character Information
	CharacterIDs    []int64 `json:"character_ids"`
	CorporationIDs  []int64 `json:"corporation_ids"`
	AllianceIDs     []int64 `json:"alliance_ids,omitempty"`
	
	// Primary Character Details
	PrimaryCharacter struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		CorporationID int64  `json:"corporation_id"`
		AllianceID    int64  `json:"alliance_id,omitempty"`
	} `json:"primary_character"`
	
	// Additional Context
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

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
	fmt.Printf("[DEBUG] ValidateAuthFromHeaders: authHeader=%q cookieHeader=%q\n", authHeader, cookieHeader)
	// Try to get token from Authorization header first
	token := m.ExtractTokenFromHeaders(authHeader)
	
	// If not found, try cookie
	if token == "" && cookieHeader != "" {
		token = m.ExtractTokenFromCookie(cookieHeader)
		fmt.Printf("[DEBUG] ValidateAuthFromHeaders: extracted token from cookie: %q\n", token)
	} else if token != "" {
		fmt.Printf("[DEBUG] ValidateAuthFromHeaders: extracted token from header: %q\n", token)
	}

	if token == "" {
		fmt.Printf("[DEBUG] ValidateAuthFromHeaders: no token found\n")
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	user, err := m.ValidateToken(token)
	if err != nil {
		fmt.Printf("[DEBUG] ValidateAuthFromHeaders: token validation failed: %v\n", err)
		return nil, huma.Error401Unauthorized("Invalid authentication token", err)
	}
	fmt.Printf("[DEBUG] ValidateAuthFromHeaders: token validation successful, userID=%s\n", user.UserID)

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
	fmt.Printf("[DEBUG] ExtractTokenFromHeaders: authHeader=%q\n", authHeader)
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		fmt.Printf("[DEBUG] ExtractTokenFromHeaders: extracted bearer token (length=%d)\n", len(token))
		return token
	}
	fmt.Printf("[DEBUG] ExtractTokenFromHeaders: no bearer token found\n")
	return ""
}

// ExtractTokenFromCookie extracts JWT token from cookie header string
func (m *AuthMiddleware) ExtractTokenFromCookie(cookieHeader string) string {
	fmt.Printf("[DEBUG] ExtractTokenFromCookie: cookieHeader=%q\n", cookieHeader)
	// Parse cookie header to find falcon_auth_token
	cookies := strings.Split(cookieHeader, ";")
	fmt.Printf("[DEBUG] ExtractTokenFromCookie: found %d cookies\n", len(cookies))
	for i, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		fmt.Printf("[DEBUG] ExtractTokenFromCookie: cookie[%d]=%q\n", i, cookie)
		if strings.HasPrefix(cookie, "falcon_auth_token=") {
			token := strings.TrimPrefix(cookie, "falcon_auth_token=")
			fmt.Printf("[DEBUG] ExtractTokenFromCookie: found falcon_auth_token (length=%d)\n", len(token))
			return token
		}
	}
	fmt.Printf("[DEBUG] ExtractTokenFromCookie: falcon_auth_token not found\n")
	return ""
}

// ValidateToken validates a JWT token string and returns the authenticated user
func (m *AuthMiddleware) ValidateToken(token string) (*models.AuthenticatedUser, error) {
	fmt.Printf("[DEBUG] ValidateToken: token length=%d\n", len(token))
	if token == "" {
		fmt.Printf("[DEBUG] ValidateToken: empty token provided\n")
		return nil, &AuthError{message: "no authentication token provided"}
	}

	// Validate JWT using the injected validator
	fmt.Printf("[DEBUG] ValidateToken: calling jwtValidator.ValidateJWT\n")
	user, err := m.jwtValidator.ValidateJWT(token)
	if err != nil {
		fmt.Printf("[DEBUG] ValidateToken: JWT validation failed: %v\n", err)
		return nil, err
	}
	fmt.Printf("[DEBUG] ValidateToken: JWT validation successful, userID=%s, characterID=%d\n", user.UserID, user.CharacterID)

	return user, nil
}

// GetAuthenticatedUser retrieves authenticated user from standard context
func GetAuthenticatedUser(ctx context.Context) *models.AuthenticatedUser {
	if user, ok := ctx.Value(AuthContextKeyUser).(*models.AuthenticatedUser); ok {
		return user
	}
	return nil
}

// GetAuthContext retrieves base auth context from request context
func GetAuthContext(ctx context.Context) *AuthContext {
	if authCtx, ok := ctx.Value(AuthContextKeyAuth).(*AuthContext); ok {
		return authCtx
	}
	return nil
}

// GetExpandedAuthContext retrieves expanded auth context from request context
func GetExpandedAuthContext(ctx context.Context) *ExpandedAuthContext {
	if expandedCtx, ok := ctx.Value(AuthContextKeyExpanded).(*ExpandedAuthContext); ok {
		return expandedCtx
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