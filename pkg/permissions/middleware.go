package permissions

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"go-falcon/internal/auth/models"
)

// AuthService interface for token validation
type AuthService interface {
	ValidateToken(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error)
}

// PermissionMiddleware provides permission-based authorization
type PermissionMiddleware struct {
	permissionManager *PermissionManager
	authService       AuthService
}

// NewPermissionMiddleware creates a new permission middleware instance
func NewPermissionMiddleware(permissionManager *PermissionManager, authService AuthService) *PermissionMiddleware {
	return &PermissionMiddleware{
		permissionManager: permissionManager,
		authService:       authService,
	}
}

// RequirePermission creates a middleware that requires a specific permission
func (pm *PermissionMiddleware) RequirePermission(permissionID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Get auth headers
			authHeader := r.Header.Get("Authorization")
			cookieHeader := ""
			if cookie, err := r.Cookie("falcon_auth_token"); err == nil {
				cookieHeader = cookie.Value
			}
			
			// Validate authentication
			user, err := pm.authService.ValidateToken(ctx, authHeader, cookieHeader)
			if err != nil {
				slog.Warn("[Permissions] Authentication failed",
					"error", err,
					"permission_required", permissionID,
					"path", r.URL.Path)
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Check permission
			hasPermission, err := pm.permissionManager.HasPermission(ctx, int64(user.CharacterID), permissionID)
			if err != nil {
				slog.Error("[Permissions] Permission check failed",
					"error", err,
					"character_id", user.CharacterID,
					"permission_id", permissionID)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			
			if !hasPermission {
				slog.Warn("[Permissions] Access denied",
					"character_id", user.CharacterID,
					"character_name", user.CharacterName,
					"permission_required", permissionID,
					"path", r.URL.Path)
				http.Error(w, fmt.Sprintf("Permission denied: %s required", permissionID), http.StatusForbidden)
				return
			}
			
			// Permission granted, add user to context and continue
			ctx = context.WithValue(ctx, "authenticated_user", user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAnyPermission creates a middleware that requires at least one of the specified permissions
func (pm *PermissionMiddleware) RequireAnyPermission(permissionIDs ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Get auth headers
			authHeader := r.Header.Get("Authorization")
			cookieHeader := ""
			if cookie, err := r.Cookie("falcon_auth_token"); err == nil {
				cookieHeader = cookie.Value
			}
			
			// Validate authentication
			user, err := pm.authService.ValidateToken(ctx, authHeader, cookieHeader)
			if err != nil {
				slog.Warn("[Permissions] Authentication failed",
					"error", err,
					"permissions_required", strings.Join(permissionIDs, ", "),
					"path", r.URL.Path)
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Check if user has any of the required permissions
			var hasPermission bool
			var grantedPermission string
			
			for _, permissionID := range permissionIDs {
				granted, err := pm.permissionManager.HasPermission(ctx, int64(user.CharacterID), permissionID)
				if err != nil {
					slog.Error("[Permissions] Permission check failed",
						"error", err,
						"character_id", user.CharacterID,
						"permission_id", permissionID)
					continue
				}
				
				if granted {
					hasPermission = true
					grantedPermission = permissionID
					break
				}
			}
			
			if !hasPermission {
				slog.Warn("[Permissions] Access denied - no required permissions",
					"character_id", user.CharacterID,
					"character_name", user.CharacterName,
					"permissions_required", strings.Join(permissionIDs, ", "),
					"path", r.URL.Path)
				http.Error(w, fmt.Sprintf("Permission denied: one of [%s] required", strings.Join(permissionIDs, ", ")), http.StatusForbidden)
				return
			}
			
			slog.Debug("[Permissions] Access granted",
				"character_id", user.CharacterID,
				"character_name", user.CharacterName,
				"granted_permission", grantedPermission,
				"path", r.URL.Path)
			
			// Permission granted, add user to context and continue
			ctx = context.WithValue(ctx, "authenticated_user", user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CheckPermission is a utility function for checking permissions in handlers
func (pm *PermissionMiddleware) CheckPermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, bool, error) {
	// Validate authentication
	user, err := pm.authService.ValidateToken(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, false, fmt.Errorf("authentication failed: %w", err)
	}
	
	// Check permission
	hasPermission, err := pm.permissionManager.HasPermission(ctx, int64(user.CharacterID), permissionID)
	if err != nil {
		return user, false, fmt.Errorf("permission check failed: %w", err)
	}
	
	return user, hasPermission, nil
}

// GetDetailedPermissionCheck returns detailed information about a permission check
func (pm *PermissionMiddleware) GetDetailedPermissionCheck(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, *PermissionCheck, error) {
	// Validate authentication
	user, err := pm.authService.ValidateToken(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, nil, fmt.Errorf("authentication failed: %w", err)
	}
	
	// Get detailed permission check
	permCheck, err := pm.permissionManager.CheckPermission(ctx, int64(user.CharacterID), permissionID)
	if err != nil {
		return user, nil, fmt.Errorf("permission check failed: %w", err)
	}
	
	return user, permCheck, nil
}

// AuthenticateOnly provides basic authentication without permission checking
func (pm *PermissionMiddleware) AuthenticateOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// Get auth headers
		authHeader := r.Header.Get("Authorization")
		cookieHeader := ""
		if cookie, err := r.Cookie("falcon_auth_token"); err == nil {
			cookieHeader = cookie.Value
		}
		
		// Validate authentication
		user, err := pm.authService.ValidateToken(ctx, authHeader, cookieHeader)
		if err != nil {
			slog.Warn("[Permissions] Authentication failed",
				"error", err,
				"path", r.URL.Path)
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		
		// Add user to context and continue
		ctx = context.WithValue(ctx, "authenticated_user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}