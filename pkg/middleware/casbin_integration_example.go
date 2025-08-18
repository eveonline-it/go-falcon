package middleware

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/danielgtaylor/huma/v2"
)

// CasbinIntegrationExample demonstrates how to integrate CASBIN role management with HUMA
type CasbinIntegrationExample struct {
	enforcer     *casbin.Enforcer
	service      *RoleAssignmentService
	routes       *RoleManagementRoutes
	authChecker  *CasbinAuthMiddleware
}

// NewCasbinIntegration creates a new CASBIN integration example
func NewCasbinIntegration(enforcer *casbin.Enforcer, authChecker *CasbinAuthMiddleware) *CasbinIntegrationExample {
	service := NewRoleAssignmentService(enforcer)
	routes := NewRoleManagementRoutes(service, authChecker)
	
	return &CasbinIntegrationExample{
		enforcer:    enforcer,
		service:     service,
		routes:      routes,
		authChecker: authChecker,
	}
}

// RegisterRoleManagementAPI registers all role management endpoints on a HUMA API
func (c *CasbinIntegrationExample) RegisterRoleManagementAPI(api huma.API, basePath string) {
	c.routes.RegisterRoleManagementRoutes(api, basePath)
}

// SetupInitialRoles sets up some basic roles and policies for the system
func (c *CasbinIntegrationExample) SetupInitialRoles(ctx context.Context) error {
	// Create basic roles
	roles := []struct {
		role        string
		permissions []struct {
			resource string
			action   string
		}
	}{
		{
			role: "admin",
			permissions: []struct {
				resource string
				action   string
			}{
				{"scheduler", "read"},
				{"scheduler", "admin"},
				{"users", "read"},
				{"users", "admin"},
				{"roles", "read"},
				{"roles", "admin"},
				{"policies", "read"},
				{"policies", "admin"},
			},
		},
		{
			role: "monitoring",
			permissions: []struct {
				resource string
				action   string
			}{
				{"scheduler", "read"},
				{"users", "read"},
			},
		},
		{
			role: "scheduler_manager",
			permissions: []struct {
				resource string
				action   string
			}{
				{"scheduler", "read"},
				{"scheduler", "write"},
				{"scheduler", "delete"},
			},
		},
	}

	for _, role := range roles {
		for _, perm := range role.permissions {
			request := &PolicyAssignmentRequest{
				Subject:  fmt.Sprintf("role:%s", role.role),
				Resource: perm.resource,
				Action:   perm.action,
				Domain:   "global",
				Effect:   "allow",
			}
			
			_, err := c.service.AssignPolicy(ctx, request)
			if err != nil {
				return fmt.Errorf("failed to assign policy %s.%s to role %s: %w", 
					perm.resource, perm.action, role.role, err)
			}
		}
	}

	return nil
}

// GrantSchedulerReadPermission is a helper function to grant scheduler.read permission to a user
func (c *CasbinIntegrationExample) GrantSchedulerReadPermission(ctx context.Context, userID string) error {
	// Method 1: Grant directly to user
	request := &PolicyAssignmentRequest{
		Subject:  fmt.Sprintf("user:%s", userID),
		Resource: "scheduler",
		Action:   "read",
		Domain:   "global",
		Effect:   "allow",
	}
	
	_, err := c.service.AssignPolicy(ctx, request)
	return err
}

// GrantSchedulerReadPermissionViaRole grants scheduler.read permission via role assignment
func (c *CasbinIntegrationExample) GrantSchedulerReadPermissionViaRole(ctx context.Context, userID string) error {
	// Method 2: Assign monitoring role (which has scheduler.read)
	request := &RoleAssignmentRequest{
		UserID: userID,
		Role:   "monitoring",
		Domain: "global",
	}
	
	_, err := c.service.AssignRole(ctx, request)
	return err
}

// QuickSetupForUser provides a quick way to set up a user with basic permissions
func (c *CasbinIntegrationExample) QuickSetupForUser(ctx context.Context, userID string, role string) error {
	request := &RoleAssignmentRequest{
		UserID: userID,
		Role:   role,
		Domain: "global",
	}
	
	result, err := c.service.AssignRole(ctx, request)
	if err != nil {
		return err
	}
	
	fmt.Printf("✅ User %s assigned role '%s': %s\n", userID, role, result.Message)
	return nil
}

// QuickSetupForCharacter provides a quick way to set up a character with basic permissions
func (c *CasbinIntegrationExample) QuickSetupForCharacter(ctx context.Context, userID string, characterID int64, role string) error {
	request := &RoleAssignmentRequest{
		UserID:      userID,
		CharacterID: &characterID,
		Role:        role,
		Domain:      "global",
	}
	
	result, err := c.service.AssignRole(ctx, request)
	if err != nil {
		return err
	}
	
	fmt.Printf("✅ Character %d (User %s) assigned role '%s': %s\n", characterID, userID, role, result.Message)
	return nil
}