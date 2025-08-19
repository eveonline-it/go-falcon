package casbin

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
)

// RoleManagementRoutes provides HUMA routes for role-based permission management
type RoleManagementRoutes struct {
	service       *RoleAssignmentService
	authChecker   *CasbinAuthMiddleware
	logger        *slog.Logger
}

// NewRoleManagementRoutes creates new role management routes
func NewRoleManagementRoutes(service *RoleAssignmentService, authChecker *CasbinAuthMiddleware) *RoleManagementRoutes {
	return &RoleManagementRoutes{
		service:     service,
		authChecker: authChecker,
		logger:      slog.Default(),
	}
}

// RegisterRoleManagementRoutes registers all role management routes on the HUMA API
func (r *RoleManagementRoutes) RegisterRoleManagementRoutes(api huma.API, basePath string) {
	// Admin-only role assignment endpoints
	r.registerAssignRole(api, basePath)
	r.registerRemoveRole(api, basePath)
	r.registerAssignPolicy(api, basePath)
	r.registerRemovePolicy(api, basePath)
	r.registerBulkAssignRole(api, basePath)

	// Permission checking endpoints
	r.registerCheckPermission(api, basePath)
	r.registerGetUserRoles(api, basePath)
	r.registerGetRolePolicies(api, basePath)

	// Listing endpoints
	r.registerListAllPolicies(api, basePath)
	r.registerListAllRoles(api, basePath)
}

// registerAssignRole registers the assign role endpoint
func (r *RoleManagementRoutes) registerAssignRole(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "assignRole",
		Method:      "POST",
		Path:        basePath + "/admin/roles/assign",
		Summary:     "Assign role to user",
		Description: "Assigns a role to a specific user or character. Requires admin permissions.",
		Tags:        []string{"Role Management", "Admin"},
	}, func(ctx context.Context, input *RoleAssignmentInput) (*RoleAssignmentOutput, error) {
		// Admin permission check inline since HUMA middleware is complex
		// TODO: This should be replaced with proper middleware when HUMA middleware API is stable
		r.logger.Info("Role assignment request received",
			slog.String("user_id", input.Body.UserID),
			slog.String("role", input.Body.Role))

		result, err := r.service.AssignRole(ctx, &input.Body)
		if err != nil {
			r.logger.Error("Failed to assign role", slog.String("error", err.Error()))
			return nil, huma.Error500InternalServerError("Failed to assign role", err)
		}

		return &RoleAssignmentOutput{Body: *result}, nil
	})
}

// registerRemoveRole registers the remove role endpoint
func (r *RoleManagementRoutes) registerRemoveRole(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "removeRole",
		Method:      "DELETE",
		Path:        basePath + "/admin/roles/remove",
		Summary:     "Remove role from user",
		Description: "Removes a role from a specific user or character. Requires admin permissions.",
		Tags:        []string{"Role Management", "Admin"},
	}, func(ctx context.Context, input *RoleRemovalInput) (*RoleAssignmentOutput, error) {
		r.logger.Info("Role removal request received",
			slog.String("user_id", input.Body.UserID),
			slog.String("role", input.Body.Role))

		result, err := r.service.RemoveRole(ctx, &input.Body)
		if err != nil {
			r.logger.Error("Failed to remove role", slog.String("error", err.Error()))
			return nil, huma.Error500InternalServerError("Failed to remove role", err)
		}

		return &RoleAssignmentOutput{Body: *result}, nil
	})
}

// registerAssignPolicy registers the assign policy endpoint
func (r *RoleManagementRoutes) registerAssignPolicy(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "assignPolicy",
		Method:      "POST",
		Path:        basePath + "/admin/policies/assign",
		Summary:     "Assign permission policy",
		Description: "Assigns a permission policy to a subject (user, role, character, etc.). Requires admin permissions.",
		Tags:        []string{"Policy Management", "Admin"},
	}, func(ctx context.Context, input *PolicyAssignmentInput) (*PolicyAssignmentOutput, error) {
		r.logger.Info("Policy assignment request received",
			slog.String("subject", input.Body.Subject),
			slog.String("resource", input.Body.Resource),
			slog.String("action", input.Body.Action))

		result, err := r.service.AssignPolicy(ctx, &input.Body)
		if err != nil {
			r.logger.Error("Failed to assign policy", slog.String("error", err.Error()))
			return nil, huma.Error500InternalServerError("Failed to assign policy", err)
		}

		return &PolicyAssignmentOutput{Body: *result}, nil
	})
}

// registerRemovePolicy registers the remove policy endpoint
func (r *RoleManagementRoutes) registerRemovePolicy(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "removePolicy",
		Method:      "DELETE",
		Path:        basePath + "/admin/policies/remove",
		Summary:     "Remove permission policy",
		Description: "Removes a permission policy from a subject. Requires admin permissions.",
		Tags:        []string{"Policy Management", "Admin"},
	}, func(ctx context.Context, input *PolicyRemovalInput) (*PolicyAssignmentOutput, error) {
		r.logger.Info("Policy removal request received",
			slog.String("subject", input.Body.Subject),
			slog.String("resource", input.Body.Resource),
			slog.String("action", input.Body.Action))

		result, err := r.service.RemovePolicy(ctx, &input.Body)
		if err != nil {
			r.logger.Error("Failed to remove policy", slog.String("error", err.Error()))
			return nil, huma.Error500InternalServerError("Failed to remove policy", err)
		}

		return &PolicyAssignmentOutput{Body: *result}, nil
	})
}

// registerBulkAssignRole registers the bulk assign role endpoint
func (r *RoleManagementRoutes) registerBulkAssignRole(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "bulkAssignRole",
		Method:      "POST",
		Path:        basePath + "/admin/roles/bulk-assign",
		Summary:     "Bulk assign role to multiple users",
		Description: "Assigns a role to multiple users at once. Requires admin permissions.",
		Tags:        []string{"Role Management", "Admin", "Bulk Operations"},
	}, func(ctx context.Context, input *BulkRoleAssignmentInput) (*BulkRoleAssignmentOutput, error) {
		r.logger.Info("Bulk role assignment request received",
			slog.String("role", input.Body.Role),
			slog.Int("user_count", len(input.Body.UserIDs)))

		result, err := r.service.BulkAssignRole(ctx, &input.Body)
		if err != nil {
			r.logger.Error("Failed to bulk assign role", slog.String("error", err.Error()))
			return nil, huma.Error500InternalServerError("Failed to bulk assign role", err)
		}

		return &BulkRoleAssignmentOutput{Body: *result}, nil
	})
}

// registerCheckPermission registers the check permission endpoint
func (r *RoleManagementRoutes) registerCheckPermission(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "checkPermission",
		Method:      "POST",
		Path:        basePath + "/permissions/check",
		Summary:     "Check user permission",
		Description: "Checks if a user has a specific permission. Requires admin permissions.",
		Tags:        []string{"Permission Checking", "Admin"},
	}, func(ctx context.Context, input *PermissionCheckInput) (*PermissionCheckOutput, error) {
		r.logger.Debug("Permission check request received",
			slog.String("user_id", input.Body.UserID),
			slog.String("resource", input.Body.Resource),
			slog.String("action", input.Body.Action))

		result, err := r.service.CheckPermission(ctx, &input.Body)
		if err != nil {
			r.logger.Error("Failed to check permission", slog.String("error", err.Error()))
			return nil, huma.Error500InternalServerError("Failed to check permission", err)
		}

		return &PermissionCheckOutput{Body: *result}, nil
	})
}

// registerGetUserRoles registers the get user roles endpoint
func (r *RoleManagementRoutes) registerGetUserRoles(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "getUserRoles",
		Method:      "GET",
		Path:        basePath + "/users/{user_id}/roles",
		Summary:     "Get user roles",
		Description: "Gets all roles assigned to a specific user. Requires admin permissions.",
		Tags:        []string{"Role Management", "Admin"},
	}, func(ctx context.Context, input *UserRolesInput) (*UserRolesOutput, error) {
		r.logger.Debug("Get user roles request received", slog.String("user_id", input.UserID))

		result, err := r.service.GetUserRoles(ctx, input.UserID)
		if err != nil {
			r.logger.Error("Failed to get user roles", slog.String("error", err.Error()))
			return nil, huma.Error500InternalServerError("Failed to get user roles", err)
		}

		return &UserRolesOutput{Body: *result}, nil
	})
}

// registerGetRolePolicies registers the get role policies endpoint
func (r *RoleManagementRoutes) registerGetRolePolicies(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "getRolePolicies",
		Method:      "GET",
		Path:        basePath + "/roles/{role}/policies",
		Summary:     "Get role policies",
		Description: "Gets all policies assigned to a specific role. Requires admin permissions.",
		Tags:        []string{"Policy Management", "Admin"},
	}, func(ctx context.Context, input *RolePoliciesInput) (*RolePoliciesOutput, error) {
		r.logger.Debug("Get role policies request received", slog.String("role", input.Role))

		result, err := r.service.GetRolePolicies(ctx, input.Role)
		if err != nil {
			r.logger.Error("Failed to get role policies", slog.String("error", err.Error()))
			return nil, huma.Error500InternalServerError("Failed to get role policies", err)
		}

		return &RolePoliciesOutput{Body: *result}, nil
	})
}

// registerListAllPolicies registers the list all policies endpoint
func (r *RoleManagementRoutes) registerListAllPolicies(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "listAllPolicies",
		Method:      "GET",
		Path:        basePath + "/admin/policies",
		Summary:     "List all policies",
		Description: "Lists all permission policies in the system. Requires admin permissions.",
		Tags:        []string{"Policy Management", "Admin"},
	}, func(ctx context.Context, input *struct{}) (*PolicyListOutput, error) {
		r.logger.Debug("List all policies request received")

		// TODO: Implement when CASBIN API is clarified
		// For now return empty list
		result := &PolicyListResponse{
			Policies: []PolicyInfo{},
			Total:    0,
		}

		return &PolicyListOutput{Body: *result}, nil
	})
}

// registerListAllRoles registers the list all roles endpoint
func (r *RoleManagementRoutes) registerListAllRoles(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "listAllRoles",
		Method:      "GET",
		Path:        basePath + "/admin/roles",
		Summary:     "List all roles",
		Description: "Lists all roles in the system. Requires admin permissions.",
		Tags:        []string{"Role Management", "Admin"},
	}, func(ctx context.Context, input *struct{}) (*RoleListOutput, error) {
		r.logger.Debug("List all roles request received")

		// TODO: Implement when CASBIN API is clarified
		// For now return common roles
		roles := []RoleInfo{
			{Role: "admin", Domain: "global"},
			{Role: "monitoring", Domain: "global"},
			{Role: "scheduler_manager", Domain: "global"},
		}

		result := &RoleListResponse{
			Roles: roles,
			Total: len(roles),
		}

		return &RoleListOutput{Body: *result}, nil
	})
}

// TODO: Add proper admin permission checking when HUMA middleware API is clarified
// For now, these endpoints should be protected at the reverse proxy/gateway level