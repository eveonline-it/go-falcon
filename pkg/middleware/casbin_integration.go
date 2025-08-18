package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"
)

// CasbinEnhancedMiddleware combines enhanced authentication with Casbin authorization
type CasbinEnhancedMiddleware struct {
	*EnhancedAuthMiddleware
	casbinAuth    *CasbinAuthMiddleware
	casbinService *CasbinService
}

// NewCasbinEnhancedMiddleware creates a new enhanced middleware with Casbin integration
func NewCasbinEnhancedMiddleware(
	jwtValidator JWTValidator,
	characterResolver UserCharacterResolver,
	mongoClient *mongo.Client,
	dbName string,
) (*CasbinEnhancedMiddleware, error) {
	// Create base enhanced auth middleware
	enhancedAuth := NewEnhancedAuthMiddleware(jwtValidator, characterResolver)

	// Create Casbin auth middleware
	casbinAuth, err := NewCasbinAuthMiddleware(mongoClient, dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin middleware: %w", err)
	}

	// Create Casbin service
	database := mongoClient.Database(dbName)
	casbinService := NewCasbinService(casbinAuth, database)

	return &CasbinEnhancedMiddleware{
		EnhancedAuthMiddleware: enhancedAuth,
		casbinAuth:            casbinAuth,
		casbinService:         casbinService,
	}, nil
}

// RequireAuthWithPermission combines authentication, character resolution, and permission checking
func (m *CasbinEnhancedMiddleware) RequireAuthWithPermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Chain: Authentication -> Character Resolution -> Permission Check
		authMiddleware := m.AuthenticationMiddleware()
		charMiddleware := m.CharacterResolutionMiddleware()
		permissionMiddleware := m.casbinAuth.RequirePermission(resource, action)

		return authMiddleware(charMiddleware(permissionMiddleware(next)))
	}
}

// OptionalAuthWithPermission provides optional authentication with permission checking if authenticated
func (m *CasbinEnhancedMiddleware) OptionalAuthWithPermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] CasbinEnhancedMiddleware.OptionalAuthWithPermission: Processing %s %s for %s.%s\n", 
				r.Method, r.URL.Path, resource, action)

			// First, try optional expanded auth
			m.OptionalExpandedAuth()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expandedCtx := GetExpandedAuthContext(r.Context())
				
				if expandedCtx != nil && expandedCtx.IsAuthenticated {
					// User is authenticated, check permissions
					fmt.Printf("[DEBUG] CasbinEnhancedMiddleware: User authenticated, checking permission\n")
					
					allowed, err := m.casbinAuth.checkHierarchicalPermission(r.Context(), expandedCtx, resource, action)
					if err != nil {
						fmt.Printf("[DEBUG] CasbinEnhancedMiddleware: Permission check failed: %v\n", err)
						http.Error(w, "Internal server error", http.StatusInternalServerError)
						return
					}
					
					if !allowed {
						fmt.Printf("[DEBUG] CasbinEnhancedMiddleware: Permission denied\n")
						http.Error(w, "Permission denied", http.StatusForbidden)
						return
					}
					
					fmt.Printf("[DEBUG] CasbinEnhancedMiddleware: Permission granted\n")
				} else {
					fmt.Printf("[DEBUG] CasbinEnhancedMiddleware: No authentication, continuing without permission check\n")
				}
				
				next.ServeHTTP(w, r)
			})).ServeHTTP(w, r)
		})
	}
}

// RequirePermissionOnly checks permissions without authentication (assumes auth already done)
func (m *CasbinEnhancedMiddleware) RequirePermissionOnly(resource, action string) func(http.Handler) http.Handler {
	return m.casbinAuth.RequirePermission(resource, action)
}

// PopulateExpandedContextWithRoles enhances the expanded context with roles and permissions
func (m *CasbinEnhancedMiddleware) PopulateExpandedContextWithRoles() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expandedCtx := GetExpandedAuthContext(r.Context())
			if expandedCtx != nil && expandedCtx.IsAuthenticated {
				fmt.Printf("[DEBUG] CasbinEnhancedMiddleware: Populating roles and permissions for user %s\n", expandedCtx.UserID)
				
				// Get all subjects for this user
				subjects := m.casbinAuth.buildSubjects(expandedCtx)
				
				var allRoles []string
				var allPermissions []string
				
				// Collect roles for all subjects
				for _, subject := range subjects {
					roles, err := m.casbinAuth.GetRolesForUser(subject)
					if err != nil {
						slog.Warn("Failed to get roles for subject", "subject", subject, "error", err)
						continue
					}
					allRoles = append(allRoles, roles...)
				}
				
				// Collect permissions for all subjects
				for _, subject := range subjects {
					permissions, err := m.casbinAuth.GetPermissionsForUser(subject)
					if err != nil {
						slog.Warn("Failed to get permissions for subject", "subject", subject, "error", err)
						continue
					}
					
					for _, perm := range permissions {
						if len(perm) >= 2 {
							allPermissions = append(allPermissions, perm[1]) // Resource.Action
						}
					}
				}
				
				// Update expanded context
				expandedCtx.Roles = allRoles
				expandedCtx.Permissions = allPermissions
				
				// Update context
				ctx := context.WithValue(r.Context(), AuthContextKeyExpanded, expandedCtx)
				r = r.WithContext(ctx)
				
				fmt.Printf("[DEBUG] CasbinEnhancedMiddleware: Added %d roles and %d permissions to context\n", 
					len(allRoles), len(allPermissions))
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// SyncUserCharacterPermissions syncs user's character hierarchy for Casbin use
func (m *CasbinEnhancedMiddleware) SyncUserCharacterPermissions(ctx context.Context, userID string, characters []UserCharacter) error {
	return m.casbinService.SyncUserHierarchy(ctx, userID, characters)
}

// GetCasbinService returns the underlying Casbin service for advanced operations
func (m *CasbinEnhancedMiddleware) GetCasbinService() *CasbinService {
	return m.casbinService
}

// GetCasbinAuth returns the underlying Casbin auth middleware
func (m *CasbinEnhancedMiddleware) GetCasbinAuth() *CasbinAuthMiddleware {
	return m.casbinAuth
}

// AdminOnly creates middleware that requires admin permissions
func (m *CasbinEnhancedMiddleware) AdminOnly() func(http.Handler) http.Handler {
	return m.RequireAuthWithPermission("system", "admin")
}

// SuperAdminOnly creates middleware that requires super admin permissions
func (m *CasbinEnhancedMiddleware) SuperAdminOnly() func(http.Handler) http.Handler {
	return m.RequireAuthWithPermission("system", "super_admin")
}

// ModulePermission creates middleware that requires specific module permissions
func (m *CasbinEnhancedMiddleware) ModulePermission(module, action string) func(http.Handler) http.Handler {
	return m.RequireAuthWithPermission(module, action)
}

// CreateConvenienceMiddleware creates a convenience middleware with Casbin integration
func (m *CasbinEnhancedMiddleware) CreateConvenienceMiddleware() *CasbinConvenienceMiddleware {
	return NewCasbinConvenienceMiddleware(m)
}

// CasbinConvenienceMiddleware provides easy-to-use wrapper functions with Casbin integration
type CasbinConvenienceMiddleware struct {
	enhanced *CasbinEnhancedMiddleware
}

// NewCasbinConvenienceMiddleware creates a new convenience middleware with Casbin
func NewCasbinConvenienceMiddleware(enhanced *CasbinEnhancedMiddleware) *CasbinConvenienceMiddleware {
	return &CasbinConvenienceMiddleware{
		enhanced: enhanced,
	}
}

// RequireAuth requires basic authentication only
func (m *CasbinConvenienceMiddleware) RequireAuth() func(http.Handler) http.Handler {
	return m.enhanced.RequireExpandedAuth()
}

// RequireAuthWithCharacters requires authentication and character resolution
func (m *CasbinConvenienceMiddleware) RequireAuthWithCharacters() func(http.Handler) http.Handler {
	authMiddleware := m.enhanced.AuthenticationMiddleware()
	charMiddleware := m.enhanced.CharacterResolutionMiddleware()
	rolesMiddleware := m.enhanced.PopulateExpandedContextWithRoles()
	
	return func(next http.Handler) http.Handler {
		return authMiddleware(charMiddleware(rolesMiddleware(next)))
	}
}

// OptionalAuth provides optional authentication
func (m *CasbinConvenienceMiddleware) OptionalAuth() func(http.Handler) http.Handler {
	return m.enhanced.OptionalExpandedAuth()
}

// RequirePermission requires specific permission
func (m *CasbinConvenienceMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return m.enhanced.RequireAuthWithPermission(resource, action)
}

// OptionalPermission checks permission only if authenticated
func (m *CasbinConvenienceMiddleware) OptionalPermission(resource, action string) func(http.Handler) http.Handler {
	return m.enhanced.OptionalAuthWithPermission(resource, action)
}

// AdminOnly requires admin permissions
func (m *CasbinConvenienceMiddleware) AdminOnly() func(http.Handler) http.Handler {
	return m.enhanced.AdminOnly()
}

// SuperAdminOnly requires super admin permissions
func (m *CasbinConvenienceMiddleware) SuperAdminOnly() func(http.Handler) http.Handler {
	return m.enhanced.SuperAdminOnly()
}

// ModuleAccess requires specific module access
func (m *CasbinConvenienceMiddleware) ModuleAccess(module string, action string) func(http.Handler) http.Handler {
	return m.enhanced.ModulePermission(module, action)
}

// CorporationAccess requires corporation-level access to a resource
func (m *CasbinConvenienceMiddleware) CorporationAccess(resource string) func(http.Handler) http.Handler {
	return m.enhanced.RequireAuthWithPermission(resource, "corporation_access")
}

// AllianceAccess requires alliance-level access to a resource  
func (m *CasbinConvenienceMiddleware) AllianceAccess(resource string) func(http.Handler) http.Handler {
	return m.enhanced.RequireAuthWithPermission(resource, "alliance_access")
}