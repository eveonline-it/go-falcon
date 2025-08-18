package routes

import (
	"context"
	"time"

	"go-falcon/internal/auth/models"
	"go-falcon/internal/groups/dto"
	"go-falcon/internal/groups/middleware"
	"go-falcon/internal/groups/services"
	humaMiddleware "go-falcon/pkg/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Routes handles HTTP routing for the Groups module
type Routes struct {
	groupService              *services.GroupService
	granularPermissionService *services.GranularPermissionService
	middleware                *middleware.Middleware
	humaAuth                  *humaMiddleware.HumaAuthMiddleware
	api                       huma.API
}

// NewRoutes creates a new Groups routes handler with both Chi and HUMA endpoints
func NewRoutes(
	groupService *services.GroupService,
	granularPermissionService *services.GranularPermissionService,
	middleware *middleware.Middleware,
	router chi.Router,
	authService humaMiddleware.JWTValidator,
) *Routes {
	// Create Huma API with Chi adapter
	config := huma.DefaultConfig("Go Falcon Groups Module", "1.0.0")
	config.Info.Description = "Group management and granular permission system for EVE Online applications"
	
	api := humachi.New(router, config)

	// Create Huma authentication middleware
	humaAuth := humaMiddleware.NewHumaAuthMiddleware(authService)

	routes := &Routes{
		groupService:              groupService,
		granularPermissionService: granularPermissionService,
		middleware:                middleware,
		humaAuth:                  humaAuth,
		api:                       api,
	}

	// Register all routes
	routes.registerRoutes()

	return routes
}

// RegisterGroupsRoutes registers groups routes on a shared Huma API
func RegisterGroupsRoutes(
	api huma.API,
	basePath string,
	groupService *services.GroupService,
	granularPermissionService *services.GranularPermissionService,
	middleware *middleware.Middleware,
	authService humaMiddleware.JWTValidator,
) {
	// Create Huma authentication middleware
	humaAuth := humaMiddleware.NewHumaAuthMiddleware(authService)

	routes := &Routes{
		groupService:              groupService,
		granularPermissionService: granularPermissionService,
		middleware:                middleware,
		humaAuth:                  humaAuth,
		api:                       api,
	}

	// Register HUMA v2 admin endpoints for granular permissions
	routes.registerHumaAdminRoutes(basePath)
}

// registerRoutes registers all Groups module routes
func (r *Routes) registerRoutes() {
	// Register traditional Chi routes for group management
	r.registerChiRoutes()
	
	// Register HUMA v2 admin routes for granular permissions
	r.registerHumaAdminRoutes("")
}

// registerChiRoutes registers traditional Chi routes for group management
func (r *Routes) registerChiRoutes() {
	// Traditional group management endpoints
	huma.Get(r.api, "/groups", r.listGroups)
	huma.Post(r.api, "/groups", r.createGroup)
	huma.Get(r.api, "/groups/{groupID}", r.getGroup)
	huma.Put(r.api, "/groups/{groupID}", r.updateGroup)
	huma.Delete(r.api, "/groups/{groupID}", r.deleteGroup)

	// Group membership management
	huma.Post(r.api, "/groups/{groupID}/members", r.addMember)
	huma.Delete(r.api, "/groups/{groupID}/members/{characterID}", r.removeMember)
	huma.Get(r.api, "/groups/{groupID}/members", r.listMembers)

	// Permission queries
	huma.Get(r.api, "/permissions/check", r.checkPermission)
	huma.Get(r.api, "/permissions/user", r.getUserPermissions)
}

// registerHumaAdminRoutes registers HUMA v2 admin endpoints for granular permissions
func (r *Routes) registerHumaAdminRoutes(basePath string) {
	// Service management endpoints
	huma.Get(r.api, basePath+"/admin/permissions/services", r.listServices)
	huma.Post(r.api, basePath+"/admin/permissions/services", r.createService)
	huma.Get(r.api, basePath+"/admin/permissions/services/{serviceName}", r.getService)
	huma.Put(r.api, basePath+"/admin/permissions/services/{serviceName}", r.updateService)
	huma.Delete(r.api, basePath+"/admin/permissions/services/{serviceName}", r.deleteService)

	// Permission assignment endpoints
	huma.Post(r.api, basePath+"/admin/permissions/assignments", r.createPermissionAssignment)
	huma.Get(r.api, basePath+"/admin/permissions/assignments", r.listPermissionAssignments)
	huma.Delete(r.api, basePath+"/admin/permissions/assignments/{assignmentID}", r.revokePermissionAssignment)

	// Permission checking endpoints
	huma.Post(r.api, basePath+"/admin/permissions/check", r.checkGranularPermission)
	huma.Get(r.api, basePath+"/admin/permissions/check/user/{characterID}", r.getUserPermissionSummary)
	huma.Get(r.api, basePath+"/admin/permissions/check/service/{serviceName}", r.getServicePermissions)

	// Utility endpoints
	huma.Get(r.api, basePath+"/admin/permissions/subjects/groups", r.listAvailableGroups)
	huma.Get(r.api, basePath+"/admin/permissions/subjects/validate", r.validateSubject)
	huma.Get(r.api, basePath+"/admin/permissions/audit", r.getAuditLogs)
}

// Helper function to check super admin permissions
func (r *Routes) requireSuperAdmin(ctx context.Context, user *models.AuthenticatedUser) error {
	// Use the new method we added to service extensions
	isSuperAdmin, err := r.granularPermissionService.IsSuperAdminByCharacterID(ctx, user.CharacterID)
	if err != nil {
		return huma.Error500InternalServerError("Super admin check failed", err)
	}
	if !isSuperAdmin {
		return huma.Error403Forbidden("Super admin privileges required")
	}
	return nil
}

// Traditional Group Management Handlers

func (r *Routes) listGroups(ctx context.Context, input *dto.GroupListInput) (*dto.GroupListOutput, error) {
	// Optional authentication
	if user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie); err == nil {
		_ = user.CharacterID // Could be used for membership checking
	}

	// Convert bool to *bool for optional filter
	var isDefault *bool
	// Only set the pointer if the user explicitly set the query parameter
	// For now, we'll assume false means not set. This is a limitation of removing the pointer
	// A better approach would be to use a string field with options "true", "false", ""
	if input.IsDefault {
		isDefault = &input.IsDefault
	}

	response, err := r.groupService.ListGroups(ctx, &dto.GroupListQuery{
		Page:        input.Page,
		PageSize:    input.PageSize,
		IsDefault:   isDefault,
		Search:      input.Search,
		ShowMembers: input.ShowMembers,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list groups", err)
	}

	return &dto.GroupListOutput{Body: *response}, nil
}

func (r *Routes) createGroup(ctx context.Context, input *dto.CreateGroupInput) (*dto.CreateGroupOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	// Check permission using service method that accepts context
	allowed, err := r.granularPermissionService.CheckPermission(ctx, convertDTOPermissionCheckToModel(&dto.PermissionCheckGranularRequest{
		Service:     "groups",
		Resource:    "management",
		Action:      "write",
		CharacterID: user.CharacterID,
	}))
	if err != nil {
		return nil, huma.Error500InternalServerError("Permission check failed", err)
	}
	if !allowed.Allowed {
		return nil, huma.Error403Forbidden("Insufficient permissions to create groups")
	}

	group, err := r.groupService.CreateGroup(ctx, &input.Body, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to create group", err)
	}

	return &dto.CreateGroupOutput{Body: *convertGroupToResponse(group, false, 0)}, nil
}

func (r *Routes) getGroup(ctx context.Context, input *dto.GetGroupInput) (*dto.GetGroupOutput, error) {
	var isMember bool
	if user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie); err == nil {
		_ = user.CharacterID
		isMember = false // TODO: Check actual membership
	}

	group, err := r.groupService.GetGroup(ctx, input.GroupID)
	if err != nil {
		return nil, huma.Error404NotFound("Group not found", err)
	}

	return &dto.GetGroupOutput{Body: *convertGroupToResponse(group, isMember, 0)}, nil
}

func (r *Routes) updateGroup(ctx context.Context, input *dto.UpdateGroupInput) (*dto.UpdateGroupOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	allowed, err := r.granularPermissionService.CheckPermission(ctx, convertDTOPermissionCheckToModel(&dto.PermissionCheckGranularRequest{
		Service:     "groups",
		Resource:    "management",
		Action:      "write",
		CharacterID: user.CharacterID,
	}))
	if err != nil {
		return nil, huma.Error500InternalServerError("Permission check failed", err)
	}
	if !allowed.Allowed {
		return nil, huma.Error403Forbidden("Insufficient permissions to update groups")
	}

	group, err := r.groupService.UpdateGroup(ctx, input.GroupID, &input.Body, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update group", err)
	}

	return &dto.UpdateGroupOutput{Body: *convertGroupToResponse(group, false, 0)}, nil
}

func (r *Routes) deleteGroup(ctx context.Context, input *dto.DeleteGroupInput) (*dto.DeleteGroupOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	allowed, err := r.granularPermissionService.CheckPermission(ctx, convertDTOPermissionCheckToModel(&dto.PermissionCheckGranularRequest{
		Service:     "groups",
		Resource:    "management",
		Action:      "delete",
		CharacterID: user.CharacterID,
	}))
	if err != nil {
		return nil, huma.Error500InternalServerError("Permission check failed", err)
	}
	if !allowed.Allowed {
		return nil, huma.Error403Forbidden("Insufficient permissions to delete groups")
	}

	err = r.groupService.DeleteGroup(ctx, input.GroupID, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete group", err)
	}

	return &dto.DeleteGroupOutput{
		Body: dto.DeleteResponse{
			Success: true,
			Message: "Group deleted successfully",
		},
	}, nil
}

// Group Membership Handlers

func (r *Routes) addMember(ctx context.Context, input *dto.AddMemberInput) (*dto.AddMemberOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	allowed, err := r.granularPermissionService.CheckPermission(ctx, convertDTOPermissionCheckToModel(&dto.PermissionCheckGranularRequest{
		Service:     "groups",
		Resource:    "membership",
		Action:      "write",
		CharacterID: user.CharacterID,
	}))
	if err != nil {
		return nil, huma.Error500InternalServerError("Permission check failed", err)
	}
	if !allowed.Allowed {
		return nil, huma.Error403Forbidden("Insufficient permissions to manage group membership")
	}

	membership, err := r.groupService.AddMember(ctx, input.GroupID, &input.Body, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to add member", err)
	}

	return &dto.AddMemberOutput{Body: *convertMembershipToResponse(membership)}, nil
}

func (r *Routes) removeMember(ctx context.Context, input *dto.RemoveMemberInput) (*dto.RemoveMemberOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	allowed, err := r.granularPermissionService.CheckPermission(ctx, convertDTOPermissionCheckToModel(&dto.PermissionCheckGranularRequest{
		Service:     "groups",
		Resource:    "membership",
		Action:      "delete",
		CharacterID: user.CharacterID,
	}))
	if err != nil {
		return nil, huma.Error500InternalServerError("Permission check failed", err)
	}
	if !allowed.Allowed {
		return nil, huma.Error403Forbidden("Insufficient permissions to manage group membership")
	}

	err = r.groupService.RemoveMember(ctx, input.GroupID, input.CharacterID, user.CharacterID, "Removed via API")
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to remove member", err)
	}

	return &dto.RemoveMemberOutput{
		Body: dto.DeleteResponse{
			Success: true,
			Message: "Member removed successfully",
		},
	}, nil
}

func (r *Routes) listMembers(ctx context.Context, input *dto.ListMembersInput) (*dto.ListMembersOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	allowed, err := r.granularPermissionService.CheckPermission(ctx, convertDTOPermissionCheckToModel(&dto.PermissionCheckGranularRequest{
		Service:     "groups",
		Resource:    "membership",
		Action:      "read",
		CharacterID: user.CharacterID,
	}))
	if err != nil {
		return nil, huma.Error500InternalServerError("Permission check failed", err)
	}
	if !allowed.Allowed {
		return nil, huma.Error403Forbidden("Insufficient permissions to view group membership")
	}

	groupID, err := primitive.ObjectIDFromHex(input.GroupID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid group ID format", err)
	}

	members, total, err := r.groupService.ListMembers(ctx, groupID, &dto.MemberListQuery{
		Page:     input.Page,
		PageSize: input.PageSize,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list members", err)
	}

	totalPages := (int(total) + input.PageSize - 1) / input.PageSize

	return &dto.ListMembersOutput{
		Body: dto.GroupMemberListResponse{
			Members:    members,
			Total:      total,
			Page:       input.Page,
			PageSize:   input.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// Permission Query Handlers

func (r *Routes) checkPermission(ctx context.Context, input *dto.CheckPermissionInput) (*dto.CheckPermissionOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	// TODO: Implement legacy permission check using groups
	_ = user.CharacterID

	result := &dto.PermissionResult{
		Allowed: false,
		Reason:  "Legacy permission system not implemented",
		Groups:  []string{},
	}

	return &dto.CheckPermissionOutput{Body: *result}, nil
}

func (r *Routes) getUserPermissions(ctx context.Context, input *dto.GetUserPermissionsInput) (*dto.GetUserPermissionsOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	groups, err := r.groupService.GetUserGroups(ctx, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get user groups", err)
	}

	groupNames := make([]string, len(groups))
	for i, group := range groups {
		groupNames[i] = group.Name
	}

	permissions := make(map[string]map[string][]string)

	response := &dto.UserPermissionsResponse{
		CharacterID: user.CharacterID,
		Groups:      groupNames,
		Permissions: permissions,
		LastUpdated: time.Now(),
	}

	return &dto.GetUserPermissionsOutput{Body: *response}, nil
}

// HUMA v2 Admin Handlers for Granular Permissions

func (r *Routes) listServices(ctx context.Context, input *dto.ListServicesInput) (*dto.ListServicesOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	services, err := r.granularPermissionService.ListServices(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list services", err)
	}

	serviceResponses := make([]dto.ServiceResponse, len(services))
	for i, service := range services {
		serviceResponses[i] = *convertServiceToResponse(&service)
	}

	totalPages := (len(services) + input.PageSize - 1) / input.PageSize

	return &dto.ListServicesOutput{
		Body: dto.ServiceListResponse{
			Services:   serviceResponses,
			Total:      int64(len(services)),
			Page:       input.Page,
			PageSize:   input.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

func (r *Routes) createService(ctx context.Context, input *dto.CreateServiceInput) (*dto.CreateServiceOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	service, err := r.granularPermissionService.CreateService(ctx, &input.Body, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to create service", err)
	}

	return &dto.CreateServiceOutput{Body: *convertServiceToResponse(service)}, nil
}

func (r *Routes) getService(ctx context.Context, input *dto.GetServiceInput) (*dto.GetServiceOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	service, err := r.granularPermissionService.GetService(ctx, input.ServiceName)
	if err != nil {
		return nil, huma.Error404NotFound("Service not found", err)
	}

	return &dto.GetServiceOutput{Body: *convertServiceToResponse(service)}, nil
}

func (r *Routes) updateService(ctx context.Context, input *dto.UpdateServiceInput) (*dto.UpdateServiceOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	service, err := r.granularPermissionService.UpdateService(ctx, input.ServiceName, &input.Body, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update service", err)
	}

	return &dto.UpdateServiceOutput{Body: *convertServiceToResponse(service)}, nil
}

func (r *Routes) deleteService(ctx context.Context, input *dto.DeleteServiceInput) (*dto.DeleteServiceOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	err = r.granularPermissionService.DeleteService(ctx, input.ServiceName, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete service", err)
	}

	return &dto.DeleteServiceOutput{
		Body: dto.DeleteResponse{
			Success: true,
			Message: "Service deleted successfully",
		},
	}, nil
}

// Permission Assignment Handlers

func (r *Routes) createPermissionAssignment(ctx context.Context, input *dto.CreatePermissionAssignmentInput) (*dto.CreatePermissionAssignmentOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	assignment, err := r.granularPermissionService.GrantPermission(ctx, &input.Body, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to grant permission", err)
	}

	return &dto.CreatePermissionAssignmentOutput{Body: *convertPermissionAssignmentToResponse(assignment, "")}, nil
}

func (r *Routes) listPermissionAssignments(ctx context.Context, input *dto.ListPermissionAssignmentsInput) (*dto.ListPermissionAssignmentsOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	assignments, total, err := r.granularPermissionService.ListPermissions(ctx, input.Service, input.Resource, input.Action, input.SubjectType, input.SubjectID, input.Page, input.PageSize)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list permission assignments", err)
	}

	totalPages := (int(total) + input.PageSize - 1) / input.PageSize

	return &dto.ListPermissionAssignmentsOutput{
		Body: dto.PermissionAssignmentListResponse{
			Assignments: assignments,
			Total:       total,
			Page:        input.Page,
			PageSize:    input.PageSize,
			TotalPages:  totalPages,
		},
	}, nil
}

func (r *Routes) revokePermissionAssignment(ctx context.Context, input *dto.RevokePermissionAssignmentInput) (*dto.RevokePermissionAssignmentOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	// TODO: Get assignment details for revocation parameters
	// For now, return a simplified response
	return &dto.RevokePermissionAssignmentOutput{
		Body: dto.DeleteResponse{
			Success: true,
			Message: "Permission revoked successfully",
		},
	}, nil
}

func (r *Routes) checkGranularPermission(ctx context.Context, input *dto.CheckGranularPermissionInput) (*dto.CheckGranularPermissionOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	result, err := r.granularPermissionService.CheckPermission(ctx, convertDTOPermissionCheckToModel(&input.Body))
	if err != nil {
		return nil, huma.Error500InternalServerError("Permission check failed", err)
	}

	return &dto.CheckGranularPermissionOutput{Body: *convertPermissionResultToDTO(result)}, nil
}

func (r *Routes) getUserPermissionSummary(ctx context.Context, input *dto.GetUserPermissionSummaryInput) (*dto.GetUserPermissionSummaryOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	summary, err := r.granularPermissionService.GetUserPermissionSummary(ctx, input.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get user permission summary", err)
	}

	return &dto.GetUserPermissionSummaryOutput{Body: *summary}, nil
}

func (r *Routes) getServicePermissions(ctx context.Context, input *dto.GetServicePermissionsInput) (*dto.GetServicePermissionsOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	permissions, err := r.granularPermissionService.GetServicePermissions(ctx, input.ServiceName)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get service permissions", err)
	}

	return &dto.GetServicePermissionsOutput{Body: *permissions}, nil
}

// Utility Handlers

func (r *Routes) listAvailableGroups(ctx context.Context, input *dto.ListAvailableGroupsInput) (*dto.ListAvailableGroupsOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	groups, total, err := r.groupService.ListGroupsForSubjects(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list available groups", err)
	}

	subjects := make([]dto.SubjectResponse, len(groups))
	for i, group := range groups {
		subjects[i] = dto.SubjectResponse{
			Type: "group",
			ID:   group.ID.Hex(),
			Name: group.Name,
		}
	}

	return &dto.ListAvailableGroupsOutput{
		Body: dto.SubjectListResponse{
			Subjects: subjects,
			Total:    total,
		},
	}, nil
}

func (r *Routes) validateSubject(ctx context.Context, input *dto.ValidateSubjectInput) (*dto.ValidateSubjectOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	valid, err := r.granularPermissionService.ValidateSubject(ctx, input.Type, input.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Subject validation failed", err)
	}

	return &dto.ValidateSubjectOutput{
		Body: dto.SubjectValidationResponse{
			Valid:   valid,
			Type:    input.Type,
			ID:      input.ID,
			Message: "Subject validation completed",
		},
	}, nil
}

func (r *Routes) getAuditLogs(ctx context.Context, input *dto.GetAuditLogsInput) (*dto.GetAuditLogsOutput, error) {
	user, err := r.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	if err := r.requireSuperAdmin(ctx, user); err != nil {
		return nil, err
	}

	// Parse date strings to time.Time pointers
	var startDate, endDate *time.Time
	if input.StartDate != "" {
		if parsed, err := time.Parse(time.RFC3339, input.StartDate); err == nil {
			startDate = &parsed
		}
	}
	if input.EndDate != "" {
		if parsed, err := time.Parse(time.RFC3339, input.EndDate); err == nil {
			endDate = &parsed
		}
	}

	logs, total, err := r.granularPermissionService.GetAuditLogs(ctx, &dto.AuditLogQuery{
		Page:      input.Page,
		PageSize:  input.PageSize,
		Service:   input.Service,
		Action:    input.Action,
		SubjectID: input.SubjectID,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get audit logs", err)
	}

	totalPages := (int(total) + input.PageSize - 1) / input.PageSize

	return &dto.GetAuditLogsOutput{
		Body: dto.AuditLogResponse{
			Entries:    logs,
			Total:      total,
			Page:       input.Page,
			PageSize:   input.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}