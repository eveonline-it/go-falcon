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
	contextHelper         *ContextHelper
}

// NewMiddlewareFactory creates a new middleware factory with all required dependencies
func NewMiddlewareFactory(jwtValidator JWTValidator, mongodb *database.MongoDB) *MiddlewareFactory {
	// Create character resolver
	characterResolver := NewUserCharacterResolver(mongodb)
	
	// Create middleware instances
	authMiddleware := NewAuthMiddleware(jwtValidator)
	enhancedAuthMiddleware := NewEnhancedAuthMiddleware(jwtValidator, characterResolver)
	contextHelper := NewContextHelper()
	
	return &MiddlewareFactory{
		authMiddleware:        authMiddleware,
		enhancedAuthMiddleware: enhancedAuthMiddleware,
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
			f.enhancedAuthMiddleware.OptionalExpandedAuth()(next).ServeHTTP(w, r)
		})
	}
}

// RequireBasicAuth - for endpoints that need basic authentication only
func (f *MiddlewareFactory) RequireBasicAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] MiddlewareFactory.RequireBasicAuth: Starting middleware chain for %s %s\n", r.Method, r.URL.Path)
			f.enhancedAuthMiddleware.AuthenticationMiddleware()(next).ServeHTTP(w, r)
		})
	}
}

// RequireAuthWithCharacters - for endpoints that need full character context
func (f *MiddlewareFactory) RequireAuthWithCharacters() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] MiddlewareFactory.RequireAuthWithCharacters: Starting middleware chain for %s %s\n", r.Method, r.URL.Path)
			f.enhancedAuthMiddleware.RequireExpandedAuth()(next).ServeHTTP(w, r)
		})
	}
}

// RequireScope - for endpoints that need specific EVE scopes
func (f *MiddlewareFactory) RequireScope(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] MiddlewareFactory.RequireScope: Checking scopes %v for %s %s\n", scopes, r.Method, r.URL.Path)
			
			// First ensure authentication
			user := GetAuthenticatedUser(r.Context())
			if user == nil {
				fmt.Printf("[DEBUG] MiddlewareFactory.RequireScope: No authenticated user found\n")
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Check if user has required scopes
			if !hasRequiredScopes(user.Scopes, scopes) {
				fmt.Printf("[DEBUG] MiddlewareFactory.RequireScope: User %s missing required scopes %v\n", user.UserID, scopes)
				http.Error(w, "Insufficient EVE Online permissions", http.StatusForbidden)
				return
			}
			
			fmt.Printf("[DEBUG] MiddlewareFactory.RequireScope: User %s has required scopes %v\n", user.UserID, scopes)
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission - for endpoints that need specific permissions (CASBIN - Phase 3)
func (f *MiddlewareFactory) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] MiddlewareFactory.RequirePermission: Checking %s:%s for %s %s\n", resource, action, r.Method, r.URL.Path)
			
			// Ensure user is authenticated with expanded context
			expandedCtx := GetExpandedAuthContext(r.Context())
			if expandedCtx == nil || !expandedCtx.IsAuthenticated {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// TODO: Implement CASBIN permission checking in Phase 3
			fmt.Printf("[DEBUG] MiddlewareFactory.RequirePermission: User %s accessing %s:%s (allowed for now)\n", 
				expandedCtx.UserID, resource, action)
			
			next.ServeHTTP(w, r)
		})
	}
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

// ChainMiddleware chains multiple middleware functions together
func ChainMiddleware(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// CreateCustomStack allows creating custom middleware stacks
func (f *MiddlewareFactory) CreateCustomStack(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return ChainMiddleware(middlewares...)
}