package middleware

import (
	"context"
	"fmt"
	"log/slog"

	"go-falcon/internal/auth/models"
	authServices "go-falcon/internal/auth/services"
	"go-falcon/internal/groups/services"
)

// AuthMiddleware provides authentication and authorization for groups
type AuthMiddleware struct {
	groupService       *services.Service
	characterContext   *CharacterContextMiddleware
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
	
	// Debug logging to help identify the issue
	slog.Debug("[Groups Auth] Checking group access", 
		"character_id", charContext.CharacterID,
		"character_name", charContext.CharacterName,
		"is_super_admin", charContext.IsSuperAdmin,
		"group_memberships", charContext.GroupMemberships)
	
	// Check if user has super admin privileges
	if !charContext.IsSuperAdmin {
		slog.Warn("[Groups Auth] Access denied - not super admin",
			"character_id", charContext.CharacterID,
			"character_name", charContext.CharacterName,
			"groups", charContext.GroupMemberships)
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