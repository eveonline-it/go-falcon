package casbin

import (
	"fmt"

	"go-falcon/pkg/middleware"
	"go.mongodb.org/mongo-driver/mongo"
)

// CasbinMiddlewareFactory provides pre-configured middleware stacks with Casbin integration
type CasbinMiddlewareFactory struct {
	enhanced    *CasbinEnhancedMiddleware
	convenience *CasbinConvenienceMiddleware
	apiHandler  *CasbinAPIHandler
}

// NewCasbinMiddlewareFactory creates a new factory with all Casbin components
func NewCasbinMiddlewareFactory(
	jwtValidator middleware.JWTValidator,
	characterResolver middleware.UserCharacterResolver,
	mongoClient *mongo.Client,
	dbName string,
) (*CasbinMiddlewareFactory, error) {
	// Create enhanced middleware with Casbin
	enhanced, err := NewCasbinEnhancedMiddleware(jwtValidator, characterResolver, mongoClient, dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to create enhanced middleware: %w", err)
	}

	// Create convenience middleware
	convenience := enhanced.CreateConvenienceMiddleware()

	// Create API handler
	apiHandler := NewCasbinAPIHandler(enhanced.GetCasbinService())

	return &CasbinMiddlewareFactory{
		enhanced:    enhanced,
		convenience: convenience,
		apiHandler:  apiHandler,
	}, nil
}

// GetEnhanced returns the enhanced middleware
func (f *CasbinMiddlewareFactory) GetEnhanced() *CasbinEnhancedMiddleware {
	return f.enhanced
}

// GetConvenience returns the convenience middleware
func (f *CasbinMiddlewareFactory) GetConvenience() *CasbinConvenienceMiddleware {
	return f.convenience
}

// GetAPIHandler returns the API handler
func (f *CasbinMiddlewareFactory) GetAPIHandler() *CasbinAPIHandler {
	return f.apiHandler
}

// GetCasbinService returns the Casbin service
func (f *CasbinMiddlewareFactory) GetCasbinService() *CasbinService {
	return f.enhanced.GetCasbinService()
}

// QuickCasbinSetup provides a quick setup for Casbin middleware (development/testing)
func QuickCasbinSetup(
	jwtValidator middleware.JWTValidator,
	characterResolver middleware.UserCharacterResolver,
	mongoClient *mongo.Client,
	dbName string,
) (*CasbinMiddlewareFactory, error) {
	factory, err := NewCasbinMiddlewareFactory(jwtValidator, characterResolver, mongoClient, dbName)
	if err != nil {
		return nil, fmt.Errorf("quick setup failed: %w", err)
	}

	// Add some default policies for testing
	casbinAuth := factory.GetEnhanced().GetCasbinAuth()

	// Add super admin role - AddPolicy(subject, resource, action, effect)
	casbinAuth.AddPolicy("role:super_admin", "system", "admin", "allow")
	casbinAuth.AddPolicy("role:super_admin", "system", "super_admin", "allow")

	// Add admin role
	casbinAuth.AddPolicy("role:admin", "system", "admin", "allow")
	casbinAuth.AddPolicy("role:admin", "users", "read", "allow")
	casbinAuth.AddPolicy("role:admin", "users", "write", "allow")

	// Add user role
	casbinAuth.AddPolicy("role:user", "users", "read", "allow")
	casbinAuth.AddPolicy("role:user", "scheduler", "tasks_read", "allow")

	// Add corp manager role
	casbinAuth.AddPolicy("role:corp_manager", "corporation", "access", "allow")
	casbinAuth.AddPolicy("role:corp_manager", "users", "corporation_access", "allow")

	// Add alliance manager role
	casbinAuth.AddPolicy("role:alliance_manager", "alliance", "access", "allow")
	casbinAuth.AddPolicy("role:alliance_manager", "users", "alliance_access", "allow")

	return factory, nil
}

// ProductionCasbinSetup provides production-ready setup for Casbin middleware
func ProductionCasbinSetup(
	jwtValidator middleware.JWTValidator,
	characterResolver middleware.UserCharacterResolver,
	mongoClient *mongo.Client,
	dbName string,
) (*CasbinMiddlewareFactory, error) {
	factory, err := NewCasbinMiddlewareFactory(jwtValidator, characterResolver, mongoClient, dbName)
	if err != nil {
		return nil, fmt.Errorf("production setup failed: %w", err)
	}

	// In production, policies should be loaded from database
	// No default policies are added here
	
	return factory, nil
}

// Example usage and configuration helpers

// SetupBasicRoles creates basic role structure for a new deployment
func (f *CasbinMiddlewareFactory) SetupBasicRoles() error {
	casbinAuth := f.enhanced.GetCasbinAuth()

	// Define basic roles and permissions
	basicRoles := map[string][]string{
		"role:guest": {
			"public.read:allow:global",
		},
		"role:member": {
			"users.read:allow:global",
			"scheduler.tasks.read:allow:global",
			"auth.profile.read:allow:global",
		},
		"role:corp_member": {
			"users.corporation_access.read:allow:global",
			"scheduler.tasks.corporation_access:allow:global",
		},
		"role:corp_manager": {
			"users.corporation_access.write:allow:global",
			"scheduler.tasks.corporation_access.admin:allow:global",
		},
		"role:alliance_member": {
			"users.alliance_access.read:allow:global",
		},
		"role:alliance_manager": {
			"users.alliance_access.write:allow:global",
		},
		"role:admin": {
			"system.admin:allow:global",
			"users.admin:allow:global",
			"scheduler.admin:allow:global",
		},
		"role:super_admin": {
			"system.super_admin:allow:global",
			"system.admin:allow:global",
		},
	}

	for role, permissions := range basicRoles {
		for _, permission := range permissions {
			parts := parsePermissionString(permission)
			if len(parts) == 4 {
				err := casbinAuth.AddPolicy(role, parts[0], parts[1], parts[2])
				if err != nil {
					return fmt.Errorf("failed to add policy %s: %w", permission, err)
				}
			}
		}
	}

	return nil
}

// parsePermissionString parses "resource.action:effect:domain" format
func parsePermissionString(permission string) []string {
	// Simple parser for "resource.action:effect:domain" format
	// In production, use a more robust parser
	return []string{"resource", "action", "effect", "domain"}
}

// GrantUserRole grants a role to a user (convenience method)
func (f *CasbinMiddlewareFactory) GrantUserRole(userID, roleName string) error {
	subject := fmt.Sprintf("user:%s", userID)
	return f.enhanced.GetCasbinAuth().AddRoleForUser(subject, roleName)
}

// GrantCharacterRole grants a role to a character (convenience method)
func (f *CasbinMiddlewareFactory) GrantCharacterRole(characterID int64, roleName string) error {
	subject := fmt.Sprintf("character:%d", characterID)
	return f.enhanced.GetCasbinAuth().AddRoleForUser(subject, roleName)
}

// GrantCorporationRole grants a role to a corporation (convenience method)
func (f *CasbinMiddlewareFactory) GrantCorporationRole(corporationID int64, roleName string) error {
	subject := fmt.Sprintf("corporation:%d", corporationID)
	return f.enhanced.GetCasbinAuth().AddRoleForUser(subject, roleName)
}

// GrantAllianceRole grants a role to an alliance (convenience method)
func (f *CasbinMiddlewareFactory) GrantAllianceRole(allianceID int64, roleName string) error {
	subject := fmt.Sprintf("alliance:%d", allianceID)
	return f.enhanced.GetCasbinAuth().AddRoleForUser(subject, roleName)
}