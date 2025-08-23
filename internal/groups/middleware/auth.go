package middleware

import (
	"context"
	"fmt"
	"log/slog"

	"go-falcon/internal/auth/models"
	authServices "go-falcon/internal/auth/services"
	"go-falcon/internal/groups/services"
	"go-falcon/pkg/permissions"
)

// AuthMiddleware provides authentication and authorization for groups
type AuthMiddleware struct {
	groupService       *services.Service
	characterContext   *CharacterContextMiddleware
	permissionManager  *permissions.PermissionManager
}


// NewAuthMiddleware creates a full auth middleware with character context resolution
func NewAuthMiddleware(authService interface{}, groupService *services.Service, permissionManager ...*permissions.PermissionManager) *AuthMiddleware {
	// Type assert to auth service
	var as *authServices.AuthService
	if authService != nil {
		if typed, ok := authService.(*authServices.AuthService); ok {
			as = typed
		}
	}
	
	// Create character context middleware with real auth service
	characterContext := NewCharacterContextMiddleware(as, groupService)
	
	// Handle optional permission manager
	var pm *permissions.PermissionManager
	if len(permissionManager) > 0 {
		pm = permissionManager[0]
	}
	
	return &AuthMiddleware{
		groupService:     groupService,
		characterContext: characterContext,
		permissionManager: pm,
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
	
	// Check permission via permission manager if available
	if m.permissionManager != nil {
		// Check specific permission for group management
		hasPermission, err := m.permissionManager.HasPermission(ctx, charContext.CharacterID, "groups:management:full")
		if err != nil {
			slog.Error("[Groups Auth] Permission check failed", "error", err, "character_id", charContext.CharacterID)
			// Fall back to super admin check
		} else if hasPermission {
			slog.Debug("[Groups Auth] Access granted via permission system",
				"character_id", charContext.CharacterID,
				"permission", "groups:management:full")
			
			// Convert to AuthenticatedUser for backward compatibility
			return &models.AuthenticatedUser{
				UserID:        charContext.UserID,
				CharacterID:   int(charContext.CharacterID),
				CharacterName: charContext.CharacterName,
				Scopes:        "publicData",
			}, nil
		}
	}
	
	// Debug logging to help identify the issue
	slog.Debug("[Groups Auth] Checking group access (fallback to super admin)", 
		"character_id", charContext.CharacterID,
		"character_name", charContext.CharacterName,
		"is_super_admin", charContext.IsSuperAdmin,
		"group_memberships", charContext.GroupMemberships)
	
	// Fall back to super admin check
	if !charContext.IsSuperAdmin {
		slog.Warn("[Groups Auth] Access denied - no group management permission",
			"character_id", charContext.CharacterID,
			"character_name", charContext.CharacterName,
			"groups", charContext.GroupMemberships)
		return nil, fmt.Errorf("group management permission required")
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
	// Resolve character context
	charContext, err := m.characterContext.ResolveCharacterContext(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	
	// Check permission via permission manager if available
	if m.permissionManager != nil {
		// Check specific permission for membership management
		hasPermission, err := m.permissionManager.HasPermission(ctx, charContext.CharacterID, "groups:memberships:manage")
		if err != nil {
			slog.Error("[Groups Auth] Permission check failed", "error", err, "character_id", charContext.CharacterID)
			// Fall back to super admin check
		} else if hasPermission {
			slog.Debug("[Groups Auth] Membership access granted via permission system",
				"character_id", charContext.CharacterID,
				"permission", "groups:memberships:manage")
			
			return &models.AuthenticatedUser{
				UserID:        charContext.UserID,
				CharacterID:   int(charContext.CharacterID),
				CharacterName: charContext.CharacterName,
				Scopes:        "publicData",
			}, nil
		}
	}
	
	// Fall back to group management permission
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

// RequirePermission checks if the authenticated user has a specific permission
func (m *AuthMiddleware) RequirePermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, error) {
	// Resolve character context
	charContext, err := m.characterContext.ResolveCharacterContext(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	
	// Check permission via permission manager
	if m.permissionManager != nil {
		hasPermission, err := m.permissionManager.HasPermission(ctx, charContext.CharacterID, permissionID)
		if err != nil {
			return nil, fmt.Errorf("permission check failed: %w", err)
		}
		
		if !hasPermission {
			slog.Warn("[Groups Auth] Access denied - missing permission",
				"character_id", charContext.CharacterID,
				"character_name", charContext.CharacterName,
				"required_permission", permissionID,
				"groups", charContext.GroupMemberships)
			return nil, fmt.Errorf("permission denied: %s required", permissionID)
		}
		
		slog.Debug("[Groups Auth] Access granted via permission system",
			"character_id", charContext.CharacterID,
			"permission", permissionID)
	} else {
		return nil, fmt.Errorf("permission system not available")
	}
	
	// Convert to AuthenticatedUser for backward compatibility
	return &models.AuthenticatedUser{
		UserID:        charContext.UserID,
		CharacterID:   int(charContext.CharacterID),
		CharacterName: charContext.CharacterName,
		Scopes:        "publicData",
	}, nil
}

// HasPermission is a utility method to check permissions without authentication
func (m *AuthMiddleware) HasPermission(ctx context.Context, characterID int64, permissionID string) (bool, error) {
	if m.permissionManager == nil {
		return false, fmt.Errorf("permission system not available")
	}
	
	return m.permissionManager.HasPermission(ctx, characterID, permissionID)
}

// CheckDetailedPermission returns detailed permission check information
func (m *AuthMiddleware) CheckDetailedPermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, *permissions.PermissionCheck, error) {
	// Resolve character context
	charContext, err := m.characterContext.ResolveCharacterContext(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, nil, fmt.Errorf("authentication failed: %w", err)
	}
	
	if m.permissionManager == nil {
		return nil, nil, fmt.Errorf("permission system not available")
	}
	
	// Get detailed permission check
	permCheck, err := m.permissionManager.CheckPermission(ctx, charContext.CharacterID, permissionID)
	if err != nil {
		return nil, nil, fmt.Errorf("permission check failed: %w", err)
	}
	
	// Convert to AuthenticatedUser
	user := &models.AuthenticatedUser{
		UserID:        charContext.UserID,
		CharacterID:   int(charContext.CharacterID),
		CharacterName: charContext.CharacterName,
		Scopes:        "publicData",
	}
	
	return user, permCheck, nil
}

// GetAuthService returns the auth service for module reconfiguration
func (m *AuthMiddleware) GetAuthService() *authServices.AuthService {
	if m.characterContext != nil {
		return m.characterContext.authService
	}
	return nil
}