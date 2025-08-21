package middleware

import (
	"context"
	"fmt"

	"go-falcon/internal/auth/models"
	authServices "go-falcon/internal/auth/services"
	"go-falcon/internal/groups/services"
	groupsMiddleware "go-falcon/internal/groups/middleware"
)

// AuthMiddleware provides authentication and authorization for site settings
type AuthMiddleware struct {
	authService   *authServices.AuthService
	groupService  *services.Service
	characterContext *groupsMiddleware.CharacterContextMiddleware
}

// NewAuthMiddleware creates a new auth middleware instance
func NewAuthMiddleware(authService *authServices.AuthService, groupService *services.Service) *AuthMiddleware {
	characterContext := groupsMiddleware.NewCharacterContextMiddleware(authService, groupService)
	
	return &AuthMiddleware{
		authService:      authService,
		groupService:     groupService,
		characterContext: characterContext,
	}
}

// RequireAuth ensures the user is authenticated
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
		Scopes:        "publicData", // Default scope
	}, nil
}

// RequireSuperAdmin ensures the user is authenticated and has super admin privileges
func (m *AuthMiddleware) RequireSuperAdmin(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	// Resolve character context
	charContext, err := m.characterContext.ResolveCharacterContext(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	
	// Check if user has super admin privileges
	if !charContext.IsSuperAdmin {
		return nil, fmt.Errorf("super admin access required for site settings management")
	}
	
	// Convert to AuthenticatedUser for backward compatibility
	return &models.AuthenticatedUser{
		UserID:        charContext.UserID,
		CharacterID:   int(charContext.CharacterID),
		CharacterName: charContext.CharacterName,
		Scopes:        "publicData",
	}, nil
}

// GetCharacterContext returns the full character context for advanced use cases
func (m *AuthMiddleware) GetCharacterContext(ctx context.Context, authHeader, cookieHeader string) (*groupsMiddleware.CharacterContext, error) {
	return m.characterContext.ResolveCharacterContext(ctx, authHeader, cookieHeader)
}