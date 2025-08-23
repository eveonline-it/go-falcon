package middleware

import (
	"context"
	"fmt"
	"strings"

	"go-falcon/internal/auth/models"
	authServices "go-falcon/internal/auth/services"
	"go-falcon/pkg/permissions"
	
	"github.com/danielgtaylor/huma/v2"
)

// AuthMiddleware provides authentication and authorization for users
type AuthMiddleware struct {
	authService       *authServices.AuthService
	permissionManager *permissions.PermissionManager
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *authServices.AuthService, permissionManager ...*permissions.PermissionManager) *AuthMiddleware {
	// Handle optional permission manager
	var pm *permissions.PermissionManager
	if len(permissionManager) > 0 {
		pm = permissionManager[0]
	}
	
	return &AuthMiddleware{
		authService:       authService,
		permissionManager: pm,
	}
}

// RequireAuth ensures the user is authenticated and returns user context
func (m *AuthMiddleware) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	if m.authService == nil {
		return nil, huma.Error500InternalServerError("Auth service not available")
	}
	
	// Extract JWT token from header or cookie
	var jwtToken string
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		jwtToken = authHeader[7:]
	} else if cookieHeader != "" {
		// Parse cookie header to find falcon_auth_token
		cookies := strings.Split(cookieHeader, ";")
		for _, cookie := range cookies {
			cookie = strings.TrimSpace(cookie)
			if strings.HasPrefix(cookie, "falcon_auth_token=") {
				jwtToken = strings.TrimPrefix(cookie, "falcon_auth_token=")
				break
			}
		}
	}
	
	if jwtToken == "" {
		return nil, huma.Error401Unauthorized("Authentication required")
	}
	
	user, err := m.authService.ValidateJWT(jwtToken)
	if err != nil {
		return nil, huma.Error401Unauthorized("Invalid authentication token", err)
	}
	
	return user, nil
}

// RequireUserManagement ensures the user has user management permissions
func (m *AuthMiddleware) RequireUserManagement(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	user, err := m.RequireAuth(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}
	
	// Check permission via permission manager if available
	if m.permissionManager != nil {
		hasPermission, err := m.permissionManager.HasPermission(ctx, int64(user.CharacterID), "users:management:full")
		if err == nil && hasPermission {
			return user, nil
		}
		// Continue to super admin check if permission failed
	}
	
	// For user management, we need to restrict access properly
	// Since we don't have direct access to groups service here, we'll deny access
	// unless specific permissions are granted through the permission manager
	return nil, huma.Error403Forbidden("Insufficient permissions: user management requires 'users:management:full' permission or super admin access")
}

// RequireUserAccess ensures the user can access user information (self or admin)
func (m *AuthMiddleware) RequireUserAccess(ctx context.Context, authHeader, cookieHeader, targetUserID string) (*models.AuthenticatedUser, error) {
	user, err := m.RequireAuth(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}
	
	// Users can always access their own data
	if user.UserID == targetUserID {
		return user, nil
	}
	
	// Check if user has admin permissions for accessing other users
	return m.RequireUserManagement(ctx, authHeader, cookieHeader)
}

// RequirePermission checks if the authenticated user has a specific permission
func (m *AuthMiddleware) RequirePermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, error) {
	user, err := m.RequireAuth(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}
	
	// Check permission via permission manager
	if m.permissionManager != nil {
		hasPermission, err := m.permissionManager.HasPermission(ctx, int64(user.CharacterID), permissionID)
		if err != nil {
			return nil, fmt.Errorf("permission check failed: %w", err)
		}
		
		if !hasPermission {
			return nil, huma.Error403Forbidden(fmt.Sprintf("Permission denied: %s required", permissionID))
		}
		
		return user, nil
	}
	
	return nil, huma.Error500InternalServerError("Permission system not available")
}