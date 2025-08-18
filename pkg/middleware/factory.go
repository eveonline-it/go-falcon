package middleware

import (
	"fmt"
	"go-falcon/pkg/database"
	"net/http"
)

// MiddlewareFactory creates pre-configured middleware stacks for common patterns
type MiddlewareFactory struct {
	authMiddleware        *AuthMiddleware
	enhancedAuthMiddleware *EnhancedAuthMiddleware
	convenienceMiddleware *ConvenienceMiddleware
	contextHelper         *ContextHelper
}

// NewMiddlewareFactory creates a new middleware factory with all required dependencies
func NewMiddlewareFactory(jwtValidator JWTValidator, mongodb *database.MongoDB) *MiddlewareFactory {
	// Create character resolver
	characterResolver := NewUserCharacterResolver(mongodb)
	
	// Create middleware instances
	authMiddleware := NewAuthMiddleware(jwtValidator)
	enhancedAuthMiddleware := NewEnhancedAuthMiddleware(jwtValidator, characterResolver)
	convenienceMiddleware := NewConvenienceMiddleware(enhancedAuthMiddleware)
	contextHelper := NewContextHelper()
	
	return &MiddlewareFactory{
		authMiddleware:        authMiddleware,
		enhancedAuthMiddleware: enhancedAuthMiddleware,
		convenienceMiddleware: convenienceMiddleware,
		contextHelper:         contextHelper,
	}
}

// GetAuthMiddleware returns the basic auth middleware
func (f *MiddlewareFactory) GetAuthMiddleware() *AuthMiddleware {
	return f.authMiddleware
}

// GetEnhancedAuthMiddleware returns the enhanced auth middleware
func (f *MiddlewareFactory) GetEnhancedAuthMiddleware() *EnhancedAuthMiddleware {
	return f.enhancedAuthMiddleware
}

// GetConvenienceMiddleware returns the convenience middleware
func (f *MiddlewareFactory) GetConvenienceMiddleware() *ConvenienceMiddleware {
	return f.convenienceMiddleware
}

// GetContextHelper returns the context helper
func (f *MiddlewareFactory) GetContextHelper() *ContextHelper {
	return f.contextHelper
}

// Common middleware stacks for different use cases

// PublicWithOptionalAuth - for endpoints that are public but can benefit from auth context
func (f *MiddlewareFactory) PublicWithOptionalAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] MiddlewareFactory.PublicWithOptionalAuth: Starting middleware chain for %s %s\n", r.Method, r.URL.Path)
			ChainMiddleware(
				f.convenienceMiddleware.OptionalAuth(),
				DebugMiddleware(),
			)(next).ServeHTTP(w, r)
		})
	}
}

// RequireBasicAuth - for endpoints that need basic authentication only
func (f *MiddlewareFactory) RequireBasicAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] MiddlewareFactory.RequireBasicAuth: Starting middleware chain for %s %s\n", r.Method, r.URL.Path)
			ChainMiddleware(
				f.convenienceMiddleware.RequireAuth(),
				DebugMiddleware(),
			)(next).ServeHTTP(w, r)
		})
	}
}

// RequireAuthWithCharacters - for endpoints that need full character context
func (f *MiddlewareFactory) RequireAuthWithCharacters() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] MiddlewareFactory.RequireAuthWithCharacters: Starting middleware chain for %s %s\n", r.Method, r.URL.Path)
			ChainMiddleware(
				f.convenienceMiddleware.RequireAuthWithCharacters(),
				DebugMiddleware(),
			)(next).ServeHTTP(w, r)
		})
	}
}

// RequireScope - for endpoints that need specific EVE scopes
func (f *MiddlewareFactory) RequireScope(scopes ...string) func(http.Handler) http.Handler {
	return ChainMiddleware(
		f.convenienceMiddleware.RequireAuth(),
		f.convenienceMiddleware.RequireScope(scopes...),
		DebugMiddleware(), // Remove in production
	)
}

// RequirePermission - for endpoints that need specific permissions (CASBIN - Phase 3)
func (f *MiddlewareFactory) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return ChainMiddleware(
		f.convenienceMiddleware.RequireAuthWithCharacters(),
		f.convenienceMiddleware.RequirePermission(resource, action),
		DebugMiddleware(), // Remove in production
	)
}

// AdminOnly - for admin-only endpoints (requires admin permission - Phase 3)
func (f *MiddlewareFactory) AdminOnly() func(http.Handler) http.Handler {
	return f.RequirePermission("system", "admin")
}

// CorporationAccess - for corporation-level access
func (f *MiddlewareFactory) CorporationAccess(resource string) func(http.Handler) http.Handler {
	return f.RequirePermission("corporation."+resource, "read")
}

// AllianceAccess - for alliance-level access
func (f *MiddlewareFactory) AllianceAccess(resource string) func(http.Handler) http.Handler {
	return f.RequirePermission("alliance."+resource, "read")
}

// SchedulerAccess - for scheduler module access
func (f *MiddlewareFactory) SchedulerAccess(action string) func(http.Handler) http.Handler {
	return f.RequirePermission("scheduler.tasks", action)
}

// UsersAccess - for users module access
func (f *MiddlewareFactory) UsersAccess(action string) func(http.Handler) http.Handler {
	return f.RequirePermission("users.profiles", action)
}

// APIKeyRequired - for endpoints that require API key access (future implementation)
func (f *MiddlewareFactory) APIKeyRequired() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Implement API key validation
			// For now, just require basic auth
			f.RequireBasicAuth()(next).ServeHTTP(w, r)
		})
	}
}

// CreateCustomStack allows creating custom middleware stacks
func (f *MiddlewareFactory) CreateCustomStack(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	// Always add debug middleware in development
	allMiddlewares := append(middlewares, DebugMiddleware())
	return ChainMiddleware(allMiddlewares...)
}