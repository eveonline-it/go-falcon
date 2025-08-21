package middleware

import (
	"context"
	"fmt"

	"go-falcon/internal/auth/models"
	authServices "go-falcon/internal/auth/services"
	"go-falcon/internal/groups/services"
)

// AuthMiddleware provides authentication and authorization for groups
type AuthMiddleware struct {
	groupService       *services.Service
	characterContext   *CharacterContextMiddleware
}

// NewSimpleAuthMiddleware creates a simple auth middleware for Phase 1 (no real auth)
func NewSimpleAuthMiddleware(groupService *services.Service) *AuthMiddleware {
	// Create character context middleware with no auth service (dummy mode)
	characterContext := NewCharacterContextMiddleware(nil, groupService)
	
	return &AuthMiddleware{
		groupService:     groupService,
		characterContext: characterContext,
	}
}

// NewAuthMiddleware creates a full auth middleware with character context resolution
func NewAuthMiddleware(authService interface{}, groupService *services.Service) *AuthMiddleware {
	// Type assert to auth service
	var as *authServices.AuthService
	if authService != nil {
		if typed, ok := authService.(*authServices.AuthService); ok {
			as = typed
		}
	}
	
	// Create character context middleware with real auth service
	characterContext := NewCharacterContextMiddleware(as, groupService)
	
	return &AuthMiddleware{
		groupService:     groupService,
		characterContext: characterContext,
	}
}

// RequireAuth ensures the user is authenticated and returns character context
func (m *AuthMiddleware) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	// Resolve character context using the character context middleware
	charContext, err := m.characterContext.ResolveCharacterContext(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	
	// Convert character context to AuthenticatedUser for backward compatibility
	return &models.AuthenticatedUser{
		UserID:        charContext.UserID,
		CharacterID:   int(charContext.CharacterID),
		CharacterName: charContext.CharacterName,
		Scopes:        "publicData", // Default scope for Phase 1
	}, nil
}

// RequireGroupAccess ensures the user has access to group management
func (m *AuthMiddleware) RequireGroupAccess(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	// Resolve character context
	charContext, err := m.characterContext.ResolveCharacterContext(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	
	// Check if user has super admin privileges
	if !charContext.IsSuperAdmin {
		return nil, fmt.Errorf("admin access required for group management")
	}
	
	// Convert to AuthenticatedUser for backward compatibility
	return &models.AuthenticatedUser{
		UserID:        charContext.UserID,
		CharacterID:   int(charContext.CharacterID),
		CharacterName: charContext.CharacterName,
		Scopes:        "publicData",
	}, nil
}

// RequireGroupMembershipAccess ensures the user can modify group memberships
func (m *AuthMiddleware) RequireGroupMembershipAccess(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	// For Phase 1, same permission requirements as group management
	return m.RequireGroupAccess(ctx, authHeader, cookieHeader)
}

// GetCharacterContext returns the full character context for advanced use cases
func (m *AuthMiddleware) GetCharacterContext(ctx context.Context, authHeader, cookieHeader string) (*CharacterContext, error) {
	return m.characterContext.ResolveCharacterContext(ctx, authHeader, cookieHeader)
}

// GetCharacterContextWithBypass returns character context with super admin bypass for character-specific routes
func (m *AuthMiddleware) GetCharacterContextWithBypass(ctx context.Context, characterID int64, authHeader, cookieHeader string) (*CharacterContext, error) {
	return m.characterContext.ResolveCharacterContextWithBypass(ctx, characterID, authHeader, cookieHeader)
}