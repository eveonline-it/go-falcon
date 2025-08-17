package groups

import (
	"context"
	"log/slog"
	"net/http"

	"go-falcon/internal/auth"
)

// GranularPermissionContextKey key for storing granular permission results in request context
type GranularPermissionContextKey string

const (
	GranularPermissionContextKeyResult = GranularPermissionContextKey("granular_permission_result")
)

// RequireGranularPermission middleware ensures the user has a specific granular permission
func (m *Module) RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			if !authenticated {
				slog.Warn("Granular permission check failed: user not authenticated",
					slog.String("service", service),
					slog.String("resource", resource),
					slog.String("action", action),
					slog.String("path", r.URL.Path))
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check granular permission
			check := &GranularPermissionCheck{
				Service:     service,
				Resource:    resource,
				Action:      action,
				CharacterID: user.CharacterID,
			}

			result, err := m.granularPermissionService.CheckPermission(r.Context(), check)
			if err != nil {
				slog.Error("Granular permission check failed", 
					slog.String("error", err.Error()),
					slog.String("service", service),
					slog.String("resource", resource),
					slog.String("action", action),
					slog.Int("character_id", user.CharacterID))
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				slog.Warn("Granular permission denied",
					slog.String("service", service),
					slog.String("resource", resource),
					slog.String("action", action),
					slog.Int("character_id", user.CharacterID),
					slog.String("character_name", user.CharacterName),
					slog.Any("granted_through", result.GrantedThrough))
				
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			// Add permission result to context for potential use by handlers
			ctx := context.WithValue(r.Context(), GranularPermissionContextKeyResult, result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalGranularPermission middleware adds granular permission checking to context without requiring it
func (m *Module) OptionalGranularPermission(service, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			
			var result *PermissionResult
			
			if authenticated {
				// Check granular permission
				check := &GranularPermissionCheck{
					Service:     service,
					Resource:    resource,
					Action:      action,
					CharacterID: user.CharacterID,
				}

				var err error
				result, err = m.granularPermissionService.CheckPermission(r.Context(), check)
				if err != nil {
					slog.Warn("Optional granular permission check failed", 
						slog.String("error", err.Error()),
						slog.String("service", service),
						slog.String("resource", resource),
						slog.String("action", action),
						slog.Int("character_id", user.CharacterID))
					
					// Continue without permission rather than failing
					next.ServeHTTP(w, r)
					return
				}
			} else {
				// Create a denied result for unauthenticated users
				result = &PermissionResult{
					Service:     service,
					Resource:    resource,
					Action:      action,
					CharacterID: 0,
					Allowed:     false,
					GrantedThrough: []string{},
				}
			}

			// Add permission result to context
			ctx := context.WithValue(r.Context(), GranularPermissionContextKeyResult, result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CheckGranularPermissionInHandler checks granular permission within a handler (for dynamic permission checking)
func (m *Module) CheckGranularPermissionInHandler(r *http.Request, service, resource, action string) (*PermissionResult, error) {
	user, authenticated := auth.GetAuthenticatedUser(r)
	
	var characterID int
	if authenticated {
		characterID = user.CharacterID
	}

	check := &GranularPermissionCheck{
		Service:     service,
		Resource:    resource,
		Action:      action,
		CharacterID: characterID,
	}

	return m.granularPermissionService.CheckPermission(r.Context(), check)
}

// GetGranularPermissionFromContext retrieves granular permission result from request context
func GetGranularPermissionFromContext(r *http.Request) (*PermissionResult, bool) {
	result, ok := r.Context().Value(GranularPermissionContextKeyResult).(*PermissionResult)
	return result, ok
}

// RequireAnyGranularPermission middleware ensures the user has at least one of the specified granular permissions
func (m *Module) RequireAnyGranularPermission(permissions []GranularPermissionCheck) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			if !authenticated {
				slog.Warn("Any granular permission check failed: user not authenticated",
					slog.String("path", r.URL.Path))
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			var allowedResults []*PermissionResult
			var hasAccess bool

			// Check each permission
			for _, perm := range permissions {
				check := &GranularPermissionCheck{
					Service:     perm.Service,
					Resource:    perm.Resource,
					Action:      perm.Action,
					CharacterID: user.CharacterID,
				}

				result, err := m.granularPermissionService.CheckPermission(r.Context(), check)
				if err != nil {
					slog.Error("Any granular permission check failed", 
						slog.String("error", err.Error()),
						slog.String("service", perm.Service),
						slog.String("resource", perm.Resource),
						slog.String("action", perm.Action))
					continue
				}

				if result.Allowed {
					hasAccess = true
					allowedResults = append(allowedResults, result)
				}
			}

			if !hasAccess {
				slog.Warn("All granular permissions denied",
					slog.Int("character_id", user.CharacterID),
					slog.String("character_name", user.CharacterName),
					slog.Int("permissions_checked", len(permissions)))
				
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			// Add all allowed permission results to context
			ctx := context.WithValue(r.Context(), GranularPermissionContextKeyResult, allowedResults)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ResourceOwnerOrGranularPermission middleware allows access if user owns the resource OR has granular permission
func (m *Module) ResourceOwnerOrGranularPermission(ownerIDExtractor func(*http.Request) int, service, resource, action string) func(http.Handler) http.Handler {
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

			// Check granular permissions
			check := &GranularPermissionCheck{
				Service:     service,
				Resource:    resource,
				Action:      action,
				CharacterID: user.CharacterID,
			}

			result, err := m.granularPermissionService.CheckPermission(r.Context(), check)
			if err != nil {
				slog.Error("Resource owner or granular permission check failed", 
					slog.String("error", err.Error()),
					slog.String("service", service),
					slog.String("resource", resource),
					slog.String("action", action))
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				slog.Warn("Access denied: not owner and no granular permission",
					slog.String("service", service),
					slog.String("resource", resource),
					slog.String("action", action),
					slog.Int("character_id", user.CharacterID),
					slog.Int("owner_id", ownerID))
				
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			// Add permission result to context
			ctx := context.WithValue(r.Context(), GranularPermissionContextKeyResult, result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ConditionalGranularPermission middleware applies granular permission checks based on conditions
func (m *Module) ConditionalGranularPermission(condition func(*http.Request) bool, service, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !condition(r) {
				// Condition not met, proceed without permission check
				next.ServeHTTP(w, r)
				return
			}

			// Apply granular permission check
			user, authenticated := auth.GetAuthenticatedUser(r)
			if !authenticated {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			check := &GranularPermissionCheck{
				Service:     service,
				Resource:    resource,
				Action:      action,
				CharacterID: user.CharacterID,
			}

			result, err := m.granularPermissionService.CheckPermission(r.Context(), check)
			if err != nil {
				slog.Error("Conditional granular permission check failed", 
					slog.String("error", err.Error()),
					slog.String("service", service),
					slog.String("resource", resource),
					slog.String("action", action))
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			// Add permission result to context
			ctx := context.WithValue(r.Context(), GranularPermissionContextKeyResult, result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LogGranularPermissions middleware logs granular permission checks for auditing
func (m *Module) LogGranularPermissions(service, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, authenticated := auth.GetAuthenticatedUser(r)
			
			var characterID int
			var characterName string
			if authenticated {
				characterID = user.CharacterID
				characterName = user.CharacterName
			}

			check := &GranularPermissionCheck{
				Service:     service,
				Resource:    resource,
				Action:      action,
				CharacterID: characterID,
			}

			result, err := m.granularPermissionService.CheckPermission(r.Context(), check)
			
			// Log the permission check
			slog.Info("Granular permission check audit",
				slog.String("service", service),
				slog.String("resource", resource),
				slog.String("action", action),
				slog.Int("character_id", characterID),
				slog.String("character_name", characterName),
				slog.Bool("authenticated", authenticated),
				slog.Bool("allowed", result != nil && result.Allowed),
				slog.Any("granted_through", func() []string {
					if result != nil {
						return result.GrantedThrough
					}
					return []string{}
				}()),
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method),
				slog.String("user_agent", r.UserAgent()),
				slog.String("remote_addr", r.RemoteAddr))

			if err != nil {
				slog.Error("Granular permission check error in audit middleware",
					slog.String("error", err.Error()),
					slog.String("service", service),
					slog.String("resource", resource),
					slog.String("action", action))
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if result == nil || !result.Allowed {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			// Add permission result to context
			ctx := context.WithValue(r.Context(), GranularPermissionContextKeyResult, result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Helper method to create granular permission checks easily
func NewGranularCheck(service, resource, action string) GranularPermissionCheck {
	return GranularPermissionCheck{
		Service:  service,
		Resource: resource,
		Action:   action,
	}
}

// Common granular permission checks for convenience
var (
	// SDE permissions
	SDEReadPermission   = NewGranularCheck("sde", "entities", "read")
	SDEWritePermission  = NewGranularCheck("sde", "entities", "write")
	SDEAdminPermission  = NewGranularCheck("sde", "entities", "admin")
	SDEUpdatePermission = NewGranularCheck("sde", "updates", "write")

	// Auth permissions
	AuthReadPermission   = NewGranularCheck("auth", "users", "read")
	AuthWritePermission  = NewGranularCheck("auth", "users", "write")
	AuthDeletePermission = NewGranularCheck("auth", "users", "delete")
	AuthAdminPermission  = NewGranularCheck("auth", "users", "admin")

	// Groups permissions
	GroupsReadPermission   = NewGranularCheck("groups", "management", "read")
	GroupsWritePermission  = NewGranularCheck("groups", "management", "write")
	GroupsDeletePermission = NewGranularCheck("groups", "management", "delete")
	GroupsAdminPermission  = NewGranularCheck("groups", "management", "admin")

	// Scheduler permissions
	SchedulerReadPermission   = NewGranularCheck("scheduler", "tasks", "read")
	SchedulerWritePermission  = NewGranularCheck("scheduler", "tasks", "write")
	SchedulerDeletePermission = NewGranularCheck("scheduler", "tasks", "delete")
	SchedulerAdminPermission  = NewGranularCheck("scheduler", "tasks", "admin")
)