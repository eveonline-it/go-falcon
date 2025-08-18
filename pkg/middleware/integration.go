package middleware

import (
	"context"
	"fmt"
	"net/http"
	
	"github.com/danielgtaylor/huma/v2"
)

// HumaIntegration provides integration helpers for Huma v2 framework
type HumaIntegration struct {
	factory *MiddlewareFactory
}

// NewHumaIntegration creates a new Huma integration helper
func NewHumaIntegration(factory *MiddlewareFactory) *HumaIntegration {
	return &HumaIntegration{
		factory: factory,
	}
}

// HumaMiddlewareFunc converts HTTP middleware to Huma middleware
// Note: This is a placeholder - Huma v2 middleware works differently than HTTP middleware
func (h *HumaIntegration) HumaMiddlewareFunc(httpMiddleware func(http.Handler) http.Handler) func(context.Context, *huma.Operation, func(context.Context, *huma.Operation)) {
	return func(ctx context.Context, operation *huma.Operation, next func(context.Context, *huma.Operation)) {
		fmt.Printf("[DEBUG] HumaIntegration: Converting HTTP middleware for operation %s\n", operation.OperationID)
		// For Huma, we'll need to integrate differently
		// This is a placeholder - Huma middleware works differently than HTTP middleware
		next(ctx, operation)
	}
}

// PublicEndpoint creates metadata for public endpoints with optional auth
func (h *HumaIntegration) PublicEndpoint(operationID, summary string) map[string]interface{} {
	return map[string]interface{}{
		"operationId": operationID,
		"summary":     summary,
		"description": fmt.Sprintf("Public endpoint: %s (optional authentication)", summary),
		"tags":        []string{"public"},
	}
}

// AuthenticatedEndpoint creates metadata for authenticated endpoints
func (h *HumaIntegration) AuthenticatedEndpoint(operationID, summary string) map[string]interface{} {
	return map[string]interface{}{
		"operationId": operationID,
		"summary":     summary,
		"description": fmt.Sprintf("Authenticated endpoint: %s (requires valid JWT)", summary),
		"tags":        []string{"authenticated"},
		"security": []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}
}

// PermissionEndpoint creates metadata for permission-protected endpoints
func (h *HumaIntegration) PermissionEndpoint(operationID, summary, resource, action string) map[string]interface{} {
	return map[string]interface{}{
		"operationId": operationID,
		"summary":     summary,
		"description": fmt.Sprintf("Permission-protected endpoint: %s (requires %s:%s permission)", summary, resource, action),
		"tags":        []string{"authenticated", "permission-protected"},
		"security": []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}
}

// AdminEndpoint creates metadata for admin-only endpoints
func (h *HumaIntegration) AdminEndpoint(operationID, summary string) map[string]interface{} {
	return map[string]interface{}{
		"operationId": operationID,
		"summary":     summary,
		"description": fmt.Sprintf("Admin-only endpoint: %s (requires admin privileges)", summary),
		"tags":        []string{"admin"},
		"security": []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}
}

// ValidateAuth provides authentication validation for Huma handlers
func (h *HumaIntegration) ValidateAuth(authHeader, cookieHeader string) (*AuthInfo, error) {
	// Use the auth middleware to validate
	user, err := h.factory.authMiddleware.ValidateAuthFromHeaders(authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}
	
	// Create auth info
	authInfo := &AuthInfo{
		IsAuthenticated:     true,
		UserID:             user.UserID,
		PrimaryCharacterID: int64(user.CharacterID),
		RequestType:        "unknown", // This would need to be determined
	}
	
	return authInfo, nil
}

// ValidateOptionalAuth provides optional authentication validation for Huma handlers
func (h *HumaIntegration) ValidateOptionalAuth(authHeader, cookieHeader string) *AuthInfo {
	authInfo, err := h.ValidateAuth(authHeader, cookieHeader)
	if err != nil {
		// Return minimal auth info for unauthenticated users
		return &AuthInfo{
			IsAuthenticated: false,
		}
	}
	return authInfo
}

// ValidateScope validates that authenticated user has required EVE scopes
func (h *HumaIntegration) ValidateScope(authHeader, cookieHeader string, requiredScopes ...string) (*AuthInfo, error) {
	user, err := h.factory.authMiddleware.ValidateScopesFromHeaders(authHeader, cookieHeader, requiredScopes...)
	if err != nil {
		return nil, err
	}
	
	// Create auth info
	authInfo := &AuthInfo{
		IsAuthenticated:     true,
		UserID:             user.UserID,
		PrimaryCharacterID: int64(user.CharacterID),
		RequestType:        "unknown", // This would need to be determined
	}
	
	return authInfo, nil
}

// Example usage patterns for modules

// ExampleAuthenticatedHandler shows how to use authentication in a Huma handler
func (h *HumaIntegration) ExampleAuthenticatedHandler(ctx context.Context, input struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
}) (struct {
	Body map[string]interface{} `json:"data"`
}, error) {
	fmt.Printf("[DEBUG] HumaIntegration: ExampleAuthenticatedHandler called\n")
	
	// Validate authentication
	authInfo, err := h.ValidateAuth(input.Authorization, input.Cookie)
	if err != nil {
		return struct {
			Body map[string]interface{} `json:"data"`
		}{}, err
	}
	
	fmt.Printf("[DEBUG] HumaIntegration: Authenticated user: %s\n", authInfo.UserID)
	
	return struct {
		Body map[string]interface{} `json:"data"`
	}{
		Body: map[string]interface{}{
			"message": "authenticated successfully",
			"user_id": authInfo.UserID,
		},
	}, nil
}

// ExampleOptionalAuthHandler shows how to use optional authentication in a Huma handler
func (h *HumaIntegration) ExampleOptionalAuthHandler(ctx context.Context, input struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
}) (struct {
	Body map[string]interface{} `json:"data"`
}, error) {
	fmt.Printf("[DEBUG] HumaIntegration: ExampleOptionalAuthHandler called\n")
	
	// Validate optional authentication
	authInfo := h.ValidateOptionalAuth(input.Authorization, input.Cookie)
	
	message := "public endpoint accessed"
	if authInfo.IsAuthenticated {
		message = fmt.Sprintf("authenticated user %s accessed public endpoint", authInfo.UserID)
	}
	
	return struct {
		Body map[string]interface{} `json:"data"`
	}{
		Body: map[string]interface{}{
			"message":        message,
			"authenticated": authInfo.IsAuthenticated,
		},
	}, nil
}