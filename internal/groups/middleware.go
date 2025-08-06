package groups

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"go-falcon/internal/auth"
)

// GroupPermissionContext key for storing group permissions in request context
type GroupPermissionContextKey string

const (
	GroupPermissionContextKeyPermissions = GroupPermissionContextKey("group_permissions")
)

// RequirePermission middleware ensures the user has a specific permission
func (m *Module) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			if !authenticated {
				slog.Warn("Permission check failed: user not authenticated",
					slog.String("resource", resource),
					slog.String("action", action),
					slog.String("path", r.URL.Path))
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			allowed, groups, err := m.permissionService.CheckPermission(r.Context(), user.CharacterID, resource, action)
			if err != nil {
				slog.Error("Permission check failed", 
					slog.String("error", err.Error()),
					slog.String("resource", resource),
					slog.String("action", action),
					slog.Int("character_id", user.CharacterID))
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !allowed {
				slog.Warn("Permission denied",
					slog.String("resource", resource),
					slog.String("action", action),
					slog.Int("character_id", user.CharacterID),
					slog.String("character_name", user.CharacterName),
					slog.Any("user_groups", groups))
				
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			// Add permission info to context for potential use by handlers
			ctx := context.WithValue(r.Context(), GroupPermissionContextKeyPermissions, map[string]interface{}{
				"resource": resource,
				"action":   action,
				"groups":   groups,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireGroup middleware ensures the user is a member of a specific group
func (m *Module) RequireGroup(groupName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			if !authenticated {
				slog.Warn("Group membership check failed: user not authenticated",
					slog.String("required_group", groupName),
					slog.String("path", r.URL.Path))
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			isMember, err := m.permissionService.IsUserInGroup(r.Context(), user.CharacterID, groupName)
			if err != nil {
				slog.Error("Group membership check failed", 
					slog.String("error", err.Error()),
					slog.String("required_group", groupName),
					slog.Int("character_id", user.CharacterID))
				http.Error(w, "Group membership check failed", http.StatusInternalServerError)
				return
			}

			if !isMember {
				slog.Warn("Group membership denied",
					slog.String("required_group", groupName),
					slog.Int("character_id", user.CharacterID),
					slog.String("character_name", user.CharacterName))
				
				http.Error(w, "Group membership required", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyGroup middleware ensures the user is a member of at least one of the specified groups
func (m *Module) RequireAnyGroup(groupNames ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			if !authenticated {
				slog.Warn("Group membership check failed: user not authenticated",
					slog.Any("required_groups", groupNames),
					slog.String("path", r.URL.Path))
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			hasAccess := false
			var memberOf []string

			for _, groupName := range groupNames {
				isMember, err := m.permissionService.IsUserInGroup(r.Context(), user.CharacterID, groupName)
				if err != nil {
					slog.Error("Group membership check failed", 
						slog.String("error", err.Error()),
						slog.String("group", groupName),
						slog.Int("character_id", user.CharacterID))
					continue
				}

				if isMember {
					hasAccess = true
					memberOf = append(memberOf, groupName)
				}
			}

			if !hasAccess {
				slog.Warn("Group membership denied",
					slog.Any("required_groups", groupNames),
					slog.Int("character_id", user.CharacterID),
					slog.String("character_name", user.CharacterName))
				
				http.Error(w, "Group membership required", http.StatusForbidden)
				return
			}

			// Add group membership info to context
			ctx := context.WithValue(r.Context(), GroupPermissionContextKeyPermissions, map[string]interface{}{
				"required_groups": groupNames,
				"member_of":      memberOf,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalPermissionMiddleware adds user permissions to context without requiring them
func (m *Module) OptionalPermissionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, authenticated := auth.GetAuthenticatedUser(r)
		
		var characterID int
		if authenticated {
			characterID = user.CharacterID
		}

		// Get user permissions (will return guest permissions for unauthenticated users)
		permissions, err := m.permissionService.GetUserPermissions(r.Context(), characterID)
		if err != nil {
			slog.Warn("Failed to get user permissions for optional middleware", 
				slog.String("error", err.Error()),
				slog.Int("character_id", characterID))
			
			// Continue without permissions rather than failing
			next.ServeHTTP(w, r)
			return
		}

		// Add permissions to context
		ctx := context.WithValue(r.Context(), GroupPermissionContextKeyPermissions, permissions)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CheckPermissionInHandler checks permission within a handler (for dynamic permission checking)
func (m *Module) CheckPermissionInHandler(r *http.Request, resource, action string) (bool, error) {
	user, authenticated := auth.GetAuthenticatedUser(r)
	
	var characterID int
	if authenticated {
		characterID = user.CharacterID
	}

	allowed, _, err := m.permissionService.CheckPermission(r.Context(), characterID, resource, action)
	return allowed, err
}

// GetUserPermissionsFromContext retrieves user permissions from request context
func GetUserPermissionsFromContext(r *http.Request) (*UserPermissionMatrix, bool) {
	permissions, ok := r.Context().Value(GroupPermissionContextKeyPermissions).(*UserPermissionMatrix)
	return permissions, ok
}

// RequireAdmin middleware ensures the user has admin permissions
func (m *Module) RequireAdmin() func(http.Handler) http.Handler {
	return m.RequireGroup("administrators")
}

// RequireSuperAdmin middleware ensures the user has super admin permissions
func (m *Module) RequireSuperAdmin() func(http.Handler) http.Handler {
	return m.RequireGroup("super_admin")
}

// RequireEVEScopes middleware that works with groups - requires both EVE scopes AND group permissions
func (m *Module) RequireEVEScopesAndPermissions(resource, action string, requiredScopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			if !authenticated {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check EVE scopes first
			userScopes := strings.Split(user.Scopes, " ")
			scopeMap := make(map[string]bool)
			for _, scope := range userScopes {
				scopeMap[strings.TrimSpace(scope)] = true
			}

			for _, requiredScope := range requiredScopes {
				if !scopeMap[requiredScope] {
					slog.Warn("User missing required EVE scope", 
						slog.String("character_name", user.CharacterName),
						slog.String("required_scope", requiredScope),
						slog.String("user_scopes", user.Scopes))
					http.Error(w, "Insufficient EVE Online permissions", http.StatusForbidden)
					return
				}
			}

			// Check group permissions
			allowed, groups, err := m.permissionService.CheckPermission(r.Context(), user.CharacterID, resource, action)
			if err != nil {
				slog.Error("Permission check failed", 
					slog.String("error", err.Error()),
					slog.String("resource", resource),
					slog.String("action", action))
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !allowed {
				slog.Warn("Group permission denied",
					slog.String("resource", resource),
					slog.String("action", action),
					slog.Int("character_id", user.CharacterID),
					slog.Any("user_groups", groups))
				
				http.Error(w, "Insufficient group permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ResourceOwnerOrPermission middleware allows access if user owns the resource OR has permission
func (m *Module) ResourceOwnerOrPermission(ownerIDExtractor func(*http.Request) int, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			if !authenticated {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check if user is the resource owner
			ownerID := ownerIDExtractor(r)
			if ownerID == user.CharacterID {
				// User owns the resource, allow access
				next.ServeHTTP(w, r)
				return
			}

			// Check group permissions
			allowed, _, err := m.permissionService.CheckPermission(r.Context(), user.CharacterID, resource, action)
			if err != nil {
				slog.Error("Permission check failed", 
					slog.String("error", err.Error()),
					slog.String("resource", resource),
					slog.String("action", action))
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !allowed {
				slog.Warn("Access denied: not owner and no permission",
					slog.String("resource", resource),
					slog.String("action", action),
					slog.Int("character_id", user.CharacterID),
					slog.Int("owner_id", ownerID))
				
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ConditionalPermission middleware applies permission checks based on conditions
func (m *Module) ConditionalPermission(condition func(*http.Request) bool, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !condition(r) {
				// Condition not met, proceed without permission check
				next.ServeHTTP(w, r)
				return
			}

			// Apply permission check
			user, authenticated := auth.GetAuthenticatedUser(r)
			if !authenticated {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			allowed, _, err := m.permissionService.CheckPermission(r.Context(), user.CharacterID, resource, action)
			if err != nil {
				slog.Error("Conditional permission check failed", 
					slog.String("error", err.Error()),
					slog.String("resource", resource),
					slog.String("action", action))
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit middleware that applies different rate limits based on user groups
func (m *Module) GroupBasedRateLimit(guestRate, userRate, adminRate int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			
			var characterID int
			if authenticated {
				characterID = user.CharacterID
			}

			// Determine rate limit based on user groups
			rateLimit := guestRate // Default to guest rate
			
			if authenticated {
				// Check if user is admin
				isAdmin, err := m.permissionService.IsUserInGroup(r.Context(), characterID, "administrators")
				if err == nil && isAdmin {
					rateLimit = adminRate
				} else {
					rateLimit = userRate
				}
			}

			// TODO: Implement actual rate limiting logic here
			// This would typically use Redis or another store to track request counts
			// For now, just pass through with rate limit info in context
			
			ctx := context.WithValue(r.Context(), "rate_limit", rateLimit)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LogGroupPermissions middleware logs permission checks for auditing
func (m *Module) LogGroupPermissions(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			
			var characterID int
			var characterName string
			if authenticated {
				characterID = user.CharacterID
				characterName = user.CharacterName
			}

			allowed, groups, err := m.permissionService.CheckPermission(r.Context(), characterID, resource, action)
			
			// Log the permission check
			slog.Info("Permission check audit",
				slog.String("resource", resource),
				slog.String("action", action),
				slog.Int("character_id", characterID),
				slog.String("character_name", characterName),
				slog.Bool("authenticated", authenticated),
				slog.Bool("allowed", allowed),
				slog.Any("user_groups", groups),
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method),
				slog.String("user_agent", r.UserAgent()),
				slog.String("remote_addr", r.RemoteAddr))

			if err != nil {
				slog.Error("Permission check error in audit middleware",
					slog.String("error", err.Error()),
					slog.String("resource", resource),
					slog.String("action", action))
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}