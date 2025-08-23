package routes

import (
	"context"
	"fmt"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	authModels "go-falcon/internal/auth/models"
	"go-falcon/internal/groups/dto"
	"go-falcon/internal/groups/middleware"
	"go-falcon/internal/groups/services"
)

// Module contains the dependencies for group routes
type Module struct {
	service    *services.Service
	middleware *middleware.AuthMiddleware
}

// NewModule creates a new routes module
func NewModule(service *services.Service, authMW *middleware.AuthMiddleware) *Module {
	return &Module{
		service:    service,
		middleware: authMW,
	}
}

// requireAuth safely checks authentication, handling nil middleware
func (m *Module) requireAuth(ctx context.Context, authHeader, cookieHeader string) (*authModels.AuthenticatedUser, error) {
	if m.middleware == nil {
		return nil, fmt.Errorf("authentication not available: module not fully initialized")
	}
	return m.middleware.RequireAuth(ctx, authHeader, cookieHeader)
}

// RegisterUnifiedRoutes registers all group routes with the API
func (m *Module) RegisterUnifiedRoutes(api huma.API) {
	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "groups-get-status",
		Method:      "GET",
		Path:        "/groups/status",
		Summary:     "Get groups module status",
		Description: "Returns the health status of the groups module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		status := m.service.GetStatus(ctx)
		return &dto.StatusOutput{Body: *status}, nil
	})

	// Health check endpoint (no auth required) - legacy
	huma.Register(api, huma.Operation{
		OperationID: "groups-health-check",
		Method:      "GET",
		Path:        "/groups/health",
		Summary:     "Group module health check",
		Description: "Check if the groups module is healthy",
		Tags:        []string{"Groups"},
	}, func(ctx context.Context, input *struct{}) (*dto.HealthOutput, error) {
		return &dto.HealthOutput{
			Body: dto.HealthResponse{
				Health: "healthy",
			},
		}, nil
	})

	// Group management endpoints
	huma.Register(api, huma.Operation{
		OperationID: "groups-create",
		Method:      "POST",
		Path:        "/groups",
		Summary:     "Create a new group",
		Description: "Create a new custom group (requires admin access)",
		Tags:        []string{"Groups / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.createGroup)

	huma.Register(api, huma.Operation{
		OperationID: "groups-list",
		Method:      "GET",
		Path:        "/groups",
		Summary:     "List groups",
		Description: "List groups with optional filtering (requires authentication)",
		Tags:        []string{"Groups / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.ListGroupsInput) (*dto.ListGroupsOutput, error) {
		// Validate authentication
		_, err := m.requireAuth(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		return m.service.ListGroups(ctx, input)
	})

	huma.Register(api, huma.Operation{
		OperationID: "groups-get",
		Method:      "GET",
		Path:        "/groups/{id}",
		Summary:     "Get a specific group",
		Description: "Retrieve details of a specific group (requires authentication)",
		Tags:        []string{"Groups / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.getGroup)

	huma.Register(api, huma.Operation{
		OperationID: "groups-update",
		Method:      "PUT",
		Path:        "/groups/{id}",
		Summary:     "Update a group",
		Description: "Update group details (requires admin access)",
		Tags:        []string{"Groups / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.updateGroup)

	huma.Register(api, huma.Operation{
		OperationID: "groups-delete",
		Method:      "DELETE",
		Path:        "/groups/{id}",
		Summary:     "Delete a group",
		Description: "Delete a group and all its memberships (requires admin access)",
		Tags:        []string{"Groups / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.deleteGroup)

	// Group membership endpoints
	huma.Register(api, huma.Operation{
		OperationID: "groups-add-member",
		Method:      "POST",
		Path:        "/groups/{group_id}/members",
		Summary:     "Add a member to a group",
		Description: "Add a character to a group (requires admin access)",
		Tags:        []string{"Groups / Memberships"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.addMember)

	huma.Register(api, huma.Operation{
		OperationID: "groups-remove-member",
		Method:      "DELETE",
		Path:        "/groups/{group_id}/members/{character_id}",
		Summary:     "Remove a member from a group",
		Description: "Remove a character from a group (requires admin access)",
		Tags:        []string{"Groups / Memberships"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.removeMember)

	huma.Register(api, huma.Operation{
		OperationID: "groups-list-members",
		Method:      "GET",
		Path:        "/groups/{group_id}/members",
		Summary:     "List group members",
		Description: "List all members of a group (requires authentication)",
		Tags:        []string{"Groups / Memberships"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.listMembers)

	huma.Register(api, huma.Operation{
		OperationID: "groups-check-membership",
		Method:      "GET",
		Path:        "/groups/{group_id}/members/{character_id}",
		Summary:     "Check group membership",
		Description: "Check if a character is a member of a group (requires authentication)",
		Tags:        []string{"Groups / Memberships"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.checkMembership)

	// Character-centric endpoints
	huma.Register(api, huma.Operation{
		OperationID: "groups-get-character-groups",
		Method:      "GET",
		Path:        "/characters/{character_id}/groups",
		Summary:     "Get character groups",
		Description: "Get all groups a character belongs to (requires authentication)",
		Tags:        []string{"Groups / Characters"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.getCharacterGroups)

	// Current user endpoints
	huma.Register(api, huma.Operation{
		OperationID: "groups-get-my-groups",
		Method:      "GET",
		Path:        "/groups/me",
		Summary:     "Get my groups",
		Description: "Get all groups the current authenticated user belongs to",
		Tags:        []string{"Groups / Current User"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.getMyGroups)

	// User-specific endpoints
	huma.Register(api, huma.Operation{
		OperationID: "groups-get-user-groups",
		Method:      "GET",
		Path:        "/users/{user_id}/groups",
		Summary:     "Get user groups",
		Description: "Get all groups that any character belonging to a user_id belongs to (requires authentication)",
		Tags:        []string{"Groups / Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.getUserGroups)
	
	// Permission Management Endpoints
	
	// List all permissions
	huma.Register(api, huma.Operation{
		OperationID: "permissions-list",
		Method:      "GET",
		Path:        "/permissions",
		Summary:     "List all permissions",
		Description: "Get all available permissions with optional filtering (requires authentication)",
		Tags:        []string{"Permissions"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.listPermissions)
	
	// Get specific permission
	huma.Register(api, huma.Operation{
		OperationID: "permissions-get",
		Method:      "GET",
		Path:        "/permissions/{permission_id}",
		Summary:     "Get permission",
		Description: "Get details of a specific permission (requires authentication)",
		Tags:        []string{"Permissions"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.getPermission)
	
	// Grant permission to group
	huma.Register(api, huma.Operation{
		OperationID: "groups-grant-permission",
		Method:      "POST",
		Path:        "/groups/{group_id}/permissions",
		Summary:     "Grant permission to group",
		Description: "Grant a specific permission to a group (requires groups:permissions:manage)",
		Tags:        []string{"Group Permissions"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.grantPermissionToGroup)
	
	// Revoke permission from group
	huma.Register(api, huma.Operation{
		OperationID: "groups-revoke-permission",
		Method:      "DELETE",
		Path:        "/groups/{group_id}/permissions/{permission_id}",
		Summary:     "Revoke permission from group",
		Description: "Revoke a specific permission from a group (requires groups:permissions:manage)",
		Tags:        []string{"Group Permissions"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.revokePermissionFromGroup)
	
	// List group permissions
	huma.Register(api, huma.Operation{
		OperationID: "groups-list-permissions",
		Method:      "GET",
		Path:        "/groups/{group_id}/permissions",
		Summary:     "List group permissions",
		Description: "Get all permissions assigned to a specific group (requires authentication)",
		Tags:        []string{"Group Permissions"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.listGroupPermissions)
	
	// Check permission
	huma.Register(api, huma.Operation{
		OperationID: "permissions-check",
		Method:      "GET",
		Path:        "/permissions/{permission_id}/check",
		Summary:     "Check permission",
		Description: "Check if the authenticated user (or specified character) has a specific permission",
		Tags:        []string{"Permissions"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, m.checkPermission)
}

// Route handlers

func (m *Module) healthCheck(ctx context.Context, input *struct{}) (*dto.HealthOutput, error) {
	return &dto.HealthOutput{
		Body: dto.HealthResponse{
			Health: "healthy",
		},
	}, nil
}

func (m *Module) createGroup(ctx context.Context, input *dto.CreateGroupInput) (*dto.GroupOutput, error) {
	// Validate authentication and admin access
	user, err := m.middleware.RequireGroupAccess(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.CreateGroup(ctx, input, int64(user.CharacterID))
}

func (m *Module) getGroup(ctx context.Context, input *dto.GetGroupInput) (*dto.GroupOutput, error) {
	// Validate authentication
	_, err := m.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.GetGroup(ctx, input)
}

func (m *Module) listGroups(ctx context.Context, input *dto.ListGroupsInput) (*dto.ListGroupsOutput, error) {
	// Validate authentication
	_, err := m.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.ListGroups(ctx, input)
}

func (m *Module) updateGroup(ctx context.Context, input *dto.UpdateGroupInput) (*dto.GroupOutput, error) {
	// Validate authentication and admin access
	_, err := m.middleware.RequireGroupAccess(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.UpdateGroup(ctx, input)
}

func (m *Module) deleteGroup(ctx context.Context, input *dto.DeleteGroupInput) (*dto.SuccessOutput, error) {
	// Validate authentication and admin access
	_, err := m.middleware.RequireGroupAccess(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.DeleteGroup(ctx, input)
}

func (m *Module) addMember(ctx context.Context, input *dto.AddMemberInput) (*dto.GroupMembershipOutput, error) {
	// Validate authentication and admin access
	user, err := m.middleware.RequireGroupMembershipAccess(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.AddMember(ctx, input, int64(user.CharacterID))
}

func (m *Module) removeMember(ctx context.Context, input *dto.RemoveMemberInput) (*dto.SuccessOutput, error) {
	// Validate authentication and admin access
	_, err := m.middleware.RequireGroupMembershipAccess(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.RemoveMember(ctx, input)
}

func (m *Module) listMembers(ctx context.Context, input *dto.ListMembersInput) (*dto.ListMembersOutput, error) {
	// Validate authentication
	_, err := m.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.ListMembers(ctx, input)
}

func (m *Module) checkMembership(ctx context.Context, input *dto.CheckMembershipInput) (*dto.MembershipCheckOutput, error) {
	// Validate authentication
	_, err := m.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.CheckMembership(ctx, input)
}

func (m *Module) getCharacterGroups(ctx context.Context, input *dto.GetCharacterGroupsInput) (*dto.CharacterGroupsOutput, error) {
	// Parse character ID from string
	characterID, err := strconv.ParseInt(input.CharacterID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid character ID: %w", err)
	}
	
	// Use bypass authentication for super admin character or normal auth for others
	_, err = m.middleware.GetCharacterContextWithBypass(ctx, characterID, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.GetCharacterGroups(ctx, input)
}

func (m *Module) getMyGroups(ctx context.Context, input *dto.GetMyGroupsInput) (*dto.CharacterGroupsOutput, error) {
	// Validate authentication and get character ID
	user, err := m.requireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.GetMyGroups(ctx, int64(user.CharacterID), input)
}

func (m *Module) getUserGroups(ctx context.Context, input *dto.GetUserGroupsInput) (*dto.UserGroupsOutput, error) {
	// Validate authentication
	_, err := m.requireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.GetUserGroups(ctx, input)
}

// Permission Management Route Handlers

func (m *Module) listPermissions(ctx context.Context, input *dto.ListPermissionsInput) (*dto.ListPermissionsOutput, error) {
	// Validate authentication
	_, err := m.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.ListPermissions(ctx, input)
}

func (m *Module) getPermission(ctx context.Context, input *dto.GetPermissionInput) (*dto.PermissionOutput, error) {
	// Validate authentication
	_, err := m.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.GetPermission(ctx, input)
}

func (m *Module) grantPermissionToGroup(ctx context.Context, input *dto.GrantPermissionToGroupInput) (*dto.GroupPermissionOutput, error) {
	// Validate authentication and permission management access
	user, err := m.middleware.RequirePermission(ctx, input.Authorization, input.Cookie, "groups:permissions:manage")
	if err != nil {
		return nil, err
	}

	return m.service.GrantPermissionToGroup(ctx, input, int64(user.CharacterID))
}

func (m *Module) revokePermissionFromGroup(ctx context.Context, input *dto.RevokePermissionFromGroupInput) (*dto.MessageOutput, error) {
	// Validate authentication and permission management access
	_, err := m.middleware.RequirePermission(ctx, input.Authorization, input.Cookie, "groups:permissions:manage")
	if err != nil {
		return nil, err
	}

	return m.service.RevokePermissionFromGroup(ctx, input)
}

func (m *Module) listGroupPermissions(ctx context.Context, input *dto.ListGroupPermissionsInput) (*dto.ListGroupPermissionsOutput, error) {
	// Validate authentication
	_, err := m.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.ListGroupPermissions(ctx, input)
}

func (m *Module) checkPermission(ctx context.Context, input *dto.CheckPermissionInput) (*dto.PermissionCheckOutput, error) {
	// Validate authentication
	user, err := m.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return m.service.CheckPermission(ctx, input, int64(user.CharacterID))
}