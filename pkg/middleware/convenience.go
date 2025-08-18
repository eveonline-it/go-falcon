package middleware

import (
	"fmt"
	"net/http"
)

// ConvenienceMiddleware provides easy-to-use middleware functions for common authentication patterns
type ConvenienceMiddleware struct {
	enhanced *EnhancedAuthMiddleware
}

// NewConvenienceMiddleware creates a new convenience middleware with an enhanced auth middleware
func NewConvenienceMiddleware(enhanced *EnhancedAuthMiddleware) *ConvenienceMiddleware {
	return &ConvenienceMiddleware{
		enhanced: enhanced,
	}
}

// RequireAuth is a convenience middleware that requires authentication only
func (m *ConvenienceMiddleware) RequireAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] ConvenienceMiddleware.RequireAuth: Processing %s %s\n", r.Method, r.URL.Path)
			m.enhanced.AuthenticationMiddleware()(next).ServeHTTP(w, r)
		})
	}
}

// RequireAuthWithCharacters requires authentication and resolves all user characters
func (m *ConvenienceMiddleware) RequireAuthWithCharacters() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] ConvenienceMiddleware.RequireAuthWithCharacters: Processing %s %s\n", r.Method, r.URL.Path)
			m.enhanced.RequireExpandedAuth()(next).ServeHTTP(w, r)
		})
	}
}

// OptionalAuth provides optional authentication - continues even if not authenticated
func (m *ConvenienceMiddleware) OptionalAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] ConvenienceMiddleware.OptionalAuth: Processing %s %s\n", r.Method, r.URL.Path)
			m.enhanced.OptionalExpandedAuth()(next).ServeHTTP(w, r)
		})
	}
}

// RequirePermission requires specific permission using CASBIN (to be implemented in Phase 3)
func (m *ConvenienceMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] ConvenienceMiddleware: RequirePermission for %s:%s (CASBIN not yet implemented)\n", resource, action)
			
			// For now, just ensure user is authenticated with expanded context
			expandedCtx := GetExpandedAuthContext(r.Context())
			if expandedCtx == nil || !expandedCtx.IsAuthenticated {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// TODO: Implement CASBIN permission checking in Phase 3
			fmt.Printf("[DEBUG] ConvenienceMiddleware: User %s accessing %s:%s (allowed for now)\n", 
				expandedCtx.UserID, resource, action)
			
			next.ServeHTTP(w, r)
		})
	}
}

// RequireScope requires specific EVE Online scopes
func (m *ConvenienceMiddleware) RequireScope(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] ConvenienceMiddleware: RequireScope for %v\n", scopes)
			
			user := GetAuthenticatedUser(r.Context())
			if user == nil {
				fmt.Printf("[DEBUG] ConvenienceMiddleware: No authenticated user found\n")
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Check if user has required scopes (using existing logic from auth.go)
			if !hasRequiredScopes(user.Scopes, scopes) {
				fmt.Printf("[DEBUG] ConvenienceMiddleware: User %s missing required scopes %v\n", user.UserID, scopes)
				http.Error(w, "Insufficient EVE Online permissions", http.StatusForbidden)
				return
			}
			
			fmt.Printf("[DEBUG] ConvenienceMiddleware: User %s has required scopes %v\n", user.UserID, scopes)
			next.ServeHTTP(w, r)
		})
	}
}

// ChainMiddleware chains multiple middleware functions together for convenience
func ChainMiddleware(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}