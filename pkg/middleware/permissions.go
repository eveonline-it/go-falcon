package middleware

import (
	"context"
	"fmt"
	"log/slog"

	"go-falcon/internal/auth/models"
	"go-falcon/pkg/permissions"

	"github.com/danielgtaylor/huma/v2"
)

// PermissionChecker interface for permission checking operations
type PermissionChecker interface {
	HasPermission(ctx context.Context, characterID int64, permissionID string) (bool, error)
	CheckPermission(ctx context.Context, characterID int64, permissionID string) (*permissions.PermissionCheck, error)
}

// PermissionMode defines how permissions are evaluated
type PermissionMode int

const (
	// PermissionModeAND requires all permissions (default)
	PermissionModeAND PermissionMode = iota
	// PermissionModeOR requires any one permission
	PermissionModeOR
)

// MiddlewareOptions configure the permission middleware behavior
type MiddlewareOptions struct {
	EnableDebugLogging   bool
	EnableCircuitBreaker bool
	FallbackToAuth       bool // Fallback to auth-only if permission system unavailable
}

// PermissionMiddleware provides centralized authentication and permission checking
type PermissionMiddleware struct {
	authMiddleware    *AuthMiddleware
	permissionChecker PermissionChecker
	options           MiddlewareOptions
}

// NewPermissionMiddleware creates a new centralized permission middleware
func NewPermissionMiddleware(
	jwtValidator JWTValidator,
	permissionChecker PermissionChecker,
	opts ...MiddlewareOption,
) *PermissionMiddleware {
	options := MiddlewareOptions{
		EnableDebugLogging:   false,
		EnableCircuitBreaker: false,
		FallbackToAuth:       true, // Safe default
	}

	// Apply options
	for _, opt := range opts {
		opt(&options)
	}

	return &PermissionMiddleware{
		authMiddleware:    NewAuthMiddleware(jwtValidator),
		permissionChecker: permissionChecker,
		options:           options,
	}
}

// MiddlewareOption configures permission middleware
type MiddlewareOption func(*MiddlewareOptions)

// WithDebugLogging enables detailed debug logging for permission checks
func WithDebugLogging() MiddlewareOption {
	return func(o *MiddlewareOptions) {
		o.EnableDebugLogging = true
	}
}

// WithCircuitBreaker enables circuit breaker pattern for permission checks
func WithCircuitBreaker() MiddlewareOption {
	return func(o *MiddlewareOptions) {
		o.EnableCircuitBreaker = true
	}
}

// WithoutFallback disables fallback to auth-only mode
func WithoutFallback() MiddlewareOption {
	return func(o *MiddlewareOptions) {
		o.FallbackToAuth = false
	}
}

// RequireAuth ensures the user is authenticated (permission-aware version)
func (pm *PermissionMiddleware) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	if pm.options.EnableDebugLogging {
		slog.Debug("[Permission Middleware] Checking authentication",
			"has_auth_header", authHeader != "",
			"has_cookie_header", cookieHeader != "")
	}

	user, err := pm.authMiddleware.ValidateAuthFromHeaders(authHeader, cookieHeader)
	if err != nil {
		if pm.options.EnableDebugLogging {
			slog.Debug("[Permission Middleware] Authentication failed", "error", err)
		}
		return nil, err
	}

	if pm.options.EnableDebugLogging {
		slog.Debug("[Permission Middleware] Authentication successful",
			"character_id", user.CharacterID,
			"character_name", user.CharacterName,
			"user_id", user.UserID)
	}

	return user, nil
}

// RequirePermission checks if the authenticated user has a specific permission
func (pm *PermissionMiddleware) RequirePermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, error) {
	// First, authenticate the user
	user, err := pm.RequireAuth(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}

	// Check permission if permission checker is available
	if pm.permissionChecker != nil {
		return pm.checkUserPermission(ctx, user, permissionID)
	}

	// Fallback behavior
	if pm.options.FallbackToAuth {
		slog.Warn("[Permission Middleware] Permission system not available, falling back to auth-only mode",
			"permission_id", permissionID,
			"character_id", user.CharacterID)
		return user, nil
	}

	return nil, huma.Error500InternalServerError("Permission system not available")
}

// RequireAnyPermission checks if user has any of the specified permissions (OR logic)
func (pm *PermissionMiddleware) RequireAnyPermission(ctx context.Context, authHeader, cookieHeader string, permissionIDs []string) (*models.AuthenticatedUser, error) {
	// First, authenticate the user
	user, err := pm.RequireAuth(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}

	if pm.options.EnableDebugLogging {
		slog.Debug("[Permission Middleware] Checking any permission",
			"character_id", user.CharacterID,
			"permissions", permissionIDs)
	}

	// Check permissions if permission checker is available
	if pm.permissionChecker != nil {
		return pm.checkUserAnyPermission(ctx, user, permissionIDs)
	}

	// Fallback behavior
	if pm.options.FallbackToAuth {
		slog.Warn("[Permission Middleware] Permission system not available, falling back to auth-only mode",
			"permissions", permissionIDs,
			"character_id", user.CharacterID)
		return user, nil
	}

	return nil, huma.Error500InternalServerError("Permission system not available")
}

// RequireAllPermissions checks if user has all specified permissions (AND logic)
func (pm *PermissionMiddleware) RequireAllPermissions(ctx context.Context, authHeader, cookieHeader string, permissionIDs []string) (*models.AuthenticatedUser, error) {
	// First, authenticate the user
	user, err := pm.RequireAuth(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}

	if pm.options.EnableDebugLogging {
		slog.Debug("[Permission Middleware] Checking all permissions",
			"character_id", user.CharacterID,
			"permissions", permissionIDs)
	}

	// Check permissions if permission checker is available
	if pm.permissionChecker != nil {
		return pm.checkUserAllPermissions(ctx, user, permissionIDs)
	}

	// Fallback behavior
	if pm.options.FallbackToAuth {
		slog.Warn("[Permission Middleware] Permission system not available, falling back to auth-only mode",
			"permissions", permissionIDs,
			"character_id", user.CharacterID)
		return user, nil
	}

	return nil, huma.Error500InternalServerError("Permission system not available")
}

// checkUserPermission performs the actual permission check for a single permission
func (pm *PermissionMiddleware) checkUserPermission(ctx context.Context, user *models.AuthenticatedUser, permissionID string) (*models.AuthenticatedUser, error) {
	if pm.options.EnableDebugLogging {
		slog.Debug("[Permission Middleware] Checking single permission",
			"character_id", user.CharacterID,
			"character_name", user.CharacterName,
			"user_id", user.UserID,
			"permission", permissionID)
	}

	// Use CheckPermission for detailed information
	permCheck, err := pm.permissionChecker.CheckPermission(ctx, int64(user.CharacterID), permissionID)
	if err != nil {
		slog.Error("[Permission Middleware] Permission check failed",
			"error", err,
			"character_id", user.CharacterID,
			"permission", permissionID)
		return nil, fmt.Errorf("permission check failed: %w", err)
	}

	if !permCheck.Granted {
		if pm.options.EnableDebugLogging {
			slog.Debug("[Permission Middleware] Permission denied",
				"character_id", user.CharacterID,
				"character_name", user.CharacterName,
				"permission", permissionID)
		}
		return nil, huma.Error403Forbidden(fmt.Sprintf("Permission denied: %s required", permissionID))
	}

	if pm.options.EnableDebugLogging {
		slog.Debug("[Permission Middleware] Permission granted",
			"character_id", user.CharacterID,
			"character_name", user.CharacterName,
			"permission", permissionID,
			"granted_via", permCheck.GrantedVia)
	}

	return user, nil
}

// checkUserAnyPermission checks if user has any of the specified permissions
func (pm *PermissionMiddleware) checkUserAnyPermission(ctx context.Context, user *models.AuthenticatedUser, permissionIDs []string) (*models.AuthenticatedUser, error) {
	if len(permissionIDs) == 0 {
		return user, nil
	}

	// Check each permission until we find one that's granted
	for _, permissionID := range permissionIDs {
		hasPermission, err := pm.permissionChecker.HasPermission(ctx, int64(user.CharacterID), permissionID)
		if err != nil {
			if pm.options.EnableDebugLogging {
				slog.Debug("[Permission Middleware] Permission check error",
					"error", err,
					"character_id", user.CharacterID,
					"permission", permissionID)
			}
			continue // Try next permission
		}

		if hasPermission {
			if pm.options.EnableDebugLogging {
				slog.Debug("[Permission Middleware] Permission granted (any mode)",
					"character_id", user.CharacterID,
					"character_name", user.CharacterName,
					"granted_permission", permissionID)
			}
			return user, nil
		}
	}

	// No permissions granted
	if pm.options.EnableDebugLogging {
		slog.Debug("[Permission Middleware] All permissions denied",
			"character_id", user.CharacterID,
			"character_name", user.CharacterName,
			"permissions", permissionIDs)
	}

	return nil, huma.Error403Forbidden(fmt.Sprintf("Permission denied: one of %v required", permissionIDs))
}

// checkUserAllPermissions checks if user has all specified permissions
func (pm *PermissionMiddleware) checkUserAllPermissions(ctx context.Context, user *models.AuthenticatedUser, permissionIDs []string) (*models.AuthenticatedUser, error) {
	if len(permissionIDs) == 0 {
		return user, nil
	}

	// Check all permissions - all must be granted
	for _, permissionID := range permissionIDs {
		hasPermission, err := pm.permissionChecker.HasPermission(ctx, int64(user.CharacterID), permissionID)
		if err != nil {
			slog.Error("[Permission Middleware] Permission check failed",
				"error", err,
				"character_id", user.CharacterID,
				"permission", permissionID)
			return nil, fmt.Errorf("permission check failed for %s: %w", permissionID, err)
		}

		if !hasPermission {
			if pm.options.EnableDebugLogging {
				slog.Debug("[Permission Middleware] Required permission denied",
					"character_id", user.CharacterID,
					"character_name", user.CharacterName,
					"missing_permission", permissionID,
					"all_required", permissionIDs)
			}
			return nil, huma.Error403Forbidden(fmt.Sprintf("Permission denied: %s required (all of %v must be granted)", permissionID, permissionIDs))
		}
	}

	if pm.options.EnableDebugLogging {
		slog.Debug("[Permission Middleware] All permissions granted",
			"character_id", user.CharacterID,
			"character_name", user.CharacterName,
			"permissions", permissionIDs)
	}

	return user, nil
}

// GetAuthMiddleware returns the underlying auth middleware for advanced use cases
func (pm *PermissionMiddleware) GetAuthMiddleware() *AuthMiddleware {
	return pm.authMiddleware
}

// GetPermissionChecker returns the permission checker for advanced use cases
func (pm *PermissionMiddleware) GetPermissionChecker() PermissionChecker {
	return pm.permissionChecker
}

// IsPermissionSystemAvailable checks if the permission system is available
func (pm *PermissionMiddleware) IsPermissionSystemAvailable() bool {
	return pm.permissionChecker != nil
}
