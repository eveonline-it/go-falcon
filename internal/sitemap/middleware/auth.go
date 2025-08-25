package middleware

import (
	"context"
	"net/http"

	"go-falcon/pkg/handlers"

	"github.com/danielgtaylor/huma/v2"
)

// AuthMiddleware provides authentication middleware for sitemap routes
type AuthMiddleware struct{}

// NewAuthMiddleware creates a new auth middleware instance
func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{}
}

// RequireAuth middleware ensures the user is authenticated
func (m *AuthMiddleware) RequireAuth(ctx context.Context, req *http.Request) error {
	user := handlers.GetAuthenticatedUser(ctx)
	if user == nil {
		return huma.Error401Unauthorized("Authentication required")
	}
	return nil
}

// RequireAdmin middleware ensures the user is a super administrator
func (m *AuthMiddleware) RequireAdmin(ctx context.Context, req *http.Request) error {
	user := handlers.GetAuthenticatedUser(ctx)
	if user == nil {
		return huma.Error401Unauthorized("Authentication required")
	}

	if !user.IsSuperAdmin {
		return huma.Error403Forbidden("Admin access required")
	}

	return nil
}

// RequirePermission middleware ensures the user has a specific permission
func (m *AuthMiddleware) RequirePermission(permission string) func(context.Context, *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		user := handlers.GetAuthenticatedUser(ctx)
		if user == nil {
			return huma.Error401Unauthorized("Authentication required")
		}

		// Super admin bypasses permission checks
		if user.IsSuperAdmin {
			return nil
		}

		// Check if user has the required permission
		// Note: This would integrate with the permission manager
		// For now, we'll use a placeholder check
		hasPermission := false
		for _, perm := range user.Permissions {
			if perm == permission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			return huma.Error403Forbidden("Insufficient permissions")
		}

		return nil
	}
}

// RequireAnyPermission middleware ensures the user has at least one of the specified permissions
func (m *AuthMiddleware) RequireAnyPermission(permissions []string) func(context.Context, *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		user := handlers.GetAuthenticatedUser(ctx)
		if user == nil {
			return huma.Error401Unauthorized("Authentication required")
		}

		// Super admin bypasses permission checks
		if user.IsSuperAdmin {
			return nil
		}

		// Check if user has any of the required permissions
		hasAnyPermission := false
		for _, userPerm := range user.Permissions {
			for _, reqPerm := range permissions {
				if userPerm == reqPerm {
					hasAnyPermission = true
					break
				}
			}
			if hasAnyPermission {
				break
			}
		}

		if !hasAnyPermission {
			return huma.Error403Forbidden("Insufficient permissions")
		}

		return nil
	}
}

// CacheControl middleware adds cache headers for sitemap responses
func (m *AuthMiddleware) CacheControl(maxAge int) func(context.Context, *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		// This would be implemented in a response middleware
		// For now, it's a placeholder
		return nil
	}
}
