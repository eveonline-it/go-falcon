package routes

import (
	"net/http"
	"strconv"

	"go-falcon/internal/groups/dto"
	"go-falcon/internal/groups/middleware"
	"go-falcon/internal/groups/models"
	"go-falcon/internal/groups/services"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
)

// Routes handles Groups module route registration
type Routes struct {
	groupService            *services.GroupService
	granularPermissionService *services.GranularPermissionService
	middleware              *middleware.Middleware
}

// NewRoutes creates a new routes handler
func NewRoutes(
	groupService *services.GroupService,
	granularPermissionService *services.GranularPermissionService,
	middleware *middleware.Middleware,
) *Routes {
	return &Routes{
		groupService:            groupService,
		granularPermissionService: granularPermissionService,
		middleware:              middleware,
	}
}

// RegisterRoutes registers all Groups module routes
func (rt *Routes) RegisterRoutes(r chi.Router) {
	// Groups routes should be mounted on a sub-path, so we create a sub-router
	// to avoid middleware conflicts with the main router
	r.Route("/groups", func(gr chi.Router) {
		// Apply common middleware to the groups sub-router
		gr.Use(rt.middleware.RequestLogging)
		gr.Use(rt.middleware.RateLimiting)
		gr.Use(rt.middleware.SecurityHeaders)
		gr.Use(rt.middleware.CORS)
		
		rt.registerGroupRoutes(gr)
	})
}

// registerGroupRoutes registers the actual group routes on the sub-router
func (rt *Routes) registerGroupRoutes(r chi.Router) {

	// Health check (public)
	r.Get("/health", rt.HealthCheck)

	// Public endpoints (no authentication required)
	r.Group(func(r chi.Router) {
		// List groups with optional filtering (public access)
		r.With(rt.middleware.ValidateQueryParams).Get("/", rt.ListGroups)
	})

	// Group Management (requires granular permissions)
	r.Group(func(r chi.Router) {
		// Require basic authentication for all group operations
		r.Use(rt.middleware.RequireGranularPermission("groups", "management", "read"))
		
		// Get specific group details
		r.Get("/{groupID}", rt.GetGroup)

		// Group creation and management (write permissions)
		r.Group(func(r chi.Router) {
			r.Use(rt.middleware.RequireGranularPermission("groups", "management", "write"))

			r.With(rt.middleware.GetValidationMiddleware().ValidateGroupCreateRequest).Post("/", rt.CreateGroup)
			r.With(rt.middleware.GetValidationMiddleware().ValidateGroupUpdateRequest).Put("/{groupID}", rt.UpdateGroup)
		})

		// Group deletion (delete permissions)
		r.With(rt.middleware.RequireGranularPermission("groups", "management", "delete")).Delete("/{groupID}", rt.DeleteGroup)

		// Membership management
		r.Route("/{groupID}/members", func(r chi.Router) {
			r.Use(rt.middleware.RequireGranularPermission("groups", "management", "read"))
			
			r.Get("/", rt.GetGroupMembers)
			
			// Add/remove members (write permissions)
			r.With(rt.middleware.RequireGranularPermission("groups", "management", "write")).With(rt.middleware.GetValidationMiddleware().ValidateMembershipRequest).Post("/", rt.AddMember)
			r.With(rt.middleware.RequireGranularPermission("groups", "management", "write")).Delete("/{characterID}", rt.RemoveMember)
		})
	})

	// Permission checking endpoints
	r.Route("/permissions", func(r chi.Router) {
		// Granular permission checking
		r.Route("/granular", func(r chi.Router) {
			r.With(rt.middleware.GetValidationMiddleware().ValidatePermissionCheckRequest).Post("/check", rt.CheckGranularPermission)
		})
	})

	// Admin endpoints for granular permission system (super admin only)
	r.Route("/admin", func(r chi.Router) {
		r.Use(rt.middleware.RequireSuperAdmin())

		// Service management
		r.Route("/services", func(r chi.Router) {
			r.Get("/", rt.ListServices)
			r.With(rt.middleware.GetValidationMiddleware().ValidateServiceCreateRequest).Post("/", rt.CreateService)
			r.Get("/{serviceName}", rt.GetService)
			r.With(rt.middleware.GetValidationMiddleware().ValidateServiceUpdateRequest).Put("/{serviceName}", rt.UpdateService)
			r.Delete("/{serviceName}", rt.DeleteService)
		})

		// Permission assignment management
		r.Route("/permissions", func(r chi.Router) {
			r.With(rt.middleware.GetValidationMiddleware().ValidatePermissionAssignmentRequest).Post("/", rt.GrantPermission)
			r.Delete("/", rt.RevokePermission)
			r.Get("/assignments", rt.ListPermissionAssignments)
		})

		// Utility endpoints
		r.Get("/subjects/groups", rt.ListSubjectGroups)
		r.Get("/audit", rt.GetAuditLogs)
		r.Get("/stats", rt.GetGroupStats)
	})
}

// Health Check

func (rt *Routes) HealthCheck(w http.ResponseWriter, r *http.Request) {
	handlers.JSONResponse(w, map[string]interface{}{
		"status": "healthy",
		"module": "groups",
	}, http.StatusOK)
}

// Group Management Handlers

func (rt *Routes) ListGroups(w http.ResponseWriter, r *http.Request) {
	// Get validated query from middleware
	query, ok := handlers.GetValidatedQuery(r.Context()).(*dto.GroupListQuery)
	if !ok {
		handlers.ErrorResponse(w, "Invalid query parameters", http.StatusBadRequest)
		return
	}

	response, err := rt.groupService.ListGroups(r.Context(), query)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to list groups", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

func (rt *Routes) GetGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	if groupID == "" {
		handlers.ErrorResponse(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	group, err := rt.groupService.GetGroup(r.Context(), groupID)
	if err != nil {
		handlers.ErrorResponse(w, "Group not found", http.StatusNotFound)
		return
	}

	// Convert to response DTO
	response := dto.GroupResponse{
		ID:                  group.ID.Hex(),
		Name:                group.Name,
		Description:         group.Description,
		IsDefault:           group.IsDefault,
		MemberCount:         group.MemberCount,
		DiscordRoles:        group.DiscordRoles,
		AutoAssignmentRules: group.AutoAssignmentRules,
		CreatedAt:           group.CreatedAt,
		UpdatedAt:           group.UpdatedAt,
		CreatedBy:           group.CreatedBy,
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

func (rt *Routes) CreateGroup(w http.ResponseWriter, r *http.Request) {
	// Get validated request from middleware
	req, ok := handlers.GetValidatedRequest(r.Context()).(*dto.GroupCreateRequest)
	if !ok {
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get character ID from auth context
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	group, err := rt.groupService.CreateGroup(r.Context(), req, characterID)
	if err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to response DTO
	response := dto.GroupResponse{
		ID:                  group.ID.Hex(),
		Name:                group.Name,
		Description:         group.Description,
		IsDefault:           group.IsDefault,
		MemberCount:         group.MemberCount,
		DiscordRoles:        group.DiscordRoles,
		AutoAssignmentRules: group.AutoAssignmentRules,
		CreatedAt:           group.CreatedAt,
		UpdatedAt:           group.UpdatedAt,
		CreatedBy:           group.CreatedBy,
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

func (rt *Routes) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	if groupID == "" {
		handlers.ErrorResponse(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Get validated request from middleware
	req, ok := handlers.GetValidatedRequest(r.Context()).(*dto.GroupUpdateRequest)
	if !ok {
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get character ID from auth context
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	group, err := rt.groupService.UpdateGroup(r.Context(), groupID, req, characterID)
	if err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to response DTO
	response := dto.GroupResponse{
		ID:                  group.ID.Hex(),
		Name:                group.Name,
		Description:         group.Description,
		IsDefault:           group.IsDefault,
		MemberCount:         group.MemberCount,
		DiscordRoles:        group.DiscordRoles,
		AutoAssignmentRules: group.AutoAssignmentRules,
		CreatedAt:           group.CreatedAt,
		UpdatedAt:           group.UpdatedAt,
		CreatedBy:           group.CreatedBy,
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

func (rt *Routes) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	if groupID == "" {
		handlers.ErrorResponse(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Get character ID from auth context
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := rt.groupService.DeleteGroup(r.Context(), groupID, characterID); err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	handlers.JSONResponse(w, map[string]interface{}{
		"message": "Group deleted successfully",
	}, http.StatusOK)
}

// Membership Management Handlers

func (rt *Routes) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	if groupID == "" {
		handlers.ErrorResponse(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	response, err := rt.groupService.GetGroupMembers(r.Context(), groupID, page, pageSize)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to get group members", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

func (rt *Routes) AddMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	if groupID == "" {
		handlers.ErrorResponse(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Get validated request from middleware
	req, ok := handlers.GetValidatedRequest(r.Context()).(*dto.MembershipRequest)
	if !ok {
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get character ID from auth context
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	membership, err := rt.groupService.AddMember(r.Context(), groupID, req, characterID)
	if err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to response DTO
	response := dto.MembershipResponse{
		CharacterID:        membership.CharacterID,
		AssignedAt:         membership.AssignedAt,
		ExpiresAt:          membership.ExpiresAt,
		ValidationStatus:   membership.ValidationStatus,
		LastValidated:      membership.LastValidated,
		AssignmentSource:   membership.AssignmentSource,
		AssignmentMetadata: membership.AssignmentMetadata,
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

func (rt *Routes) RemoveMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	if groupID == "" {
		handlers.ErrorResponse(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	characterIDStr := chi.URLParam(r, "characterID")
	if characterIDStr == "" {
		handlers.ErrorResponse(w, "Character ID is required", http.StatusBadRequest)
		return
	}

	memberCharacterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		handlers.ErrorResponse(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	// Get character ID from auth context
	actorCharacterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	reason := r.URL.Query().Get("reason")
	if reason == "" {
		reason = "Removed by administrator"
	}

	if err := rt.groupService.RemoveMember(r.Context(), groupID, memberCharacterID, actorCharacterID, reason); err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	handlers.JSONResponse(w, map[string]interface{}{
		"message": "Member removed successfully",
	}, http.StatusOK)
}

// Permission Checking Handlers

func (rt *Routes) CheckGranularPermission(w http.ResponseWriter, r *http.Request) {
	// Get validated request from middleware
	req, ok := handlers.GetValidatedRequest(r.Context()).(*dto.PermissionCheckGranularRequest)
	if !ok {
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get character ID from auth context if not provided
	if req.CharacterID == 0 {
		characterID, err := handlers.GetCharacterIDFromRequest(r)
		if err != nil {
			handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		req.CharacterID = characterID
	}

	checkReq := &models.GranularPermissionCheck{
		CharacterID: req.CharacterID,
		Service:     req.Service,
		Resource:    req.Resource,
		Action:      req.Action,
	}

	result, err := rt.granularPermissionService.CheckPermission(r.Context(), checkReq)
	if err != nil {
		handlers.ErrorResponse(w, "Permission check failed", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, result, http.StatusOK)
}

// Admin Handlers

func (rt *Routes) ListServices(w http.ResponseWriter, r *http.Request) {
	services, err := rt.granularPermissionService.ListServices(r.Context())
	if err != nil {
		handlers.ErrorResponse(w, "Failed to list services", http.StatusInternalServerError)
		return
	}

	// Convert to response DTOs
	var serviceResponses []dto.ServiceResponse
	for _, service := range services {
		serviceResponses = append(serviceResponses, dto.ServiceResponse{
			ID:          service.ID.Hex(),
			Name:        service.Name,
			DisplayName: service.DisplayName,
			Description: service.Description,
			Resources:   service.Resources,
			Enabled:     service.Enabled,
			CreatedAt:   service.CreatedAt,
			UpdatedAt:   service.UpdatedAt,
		})
	}

	handlers.JSONResponse(w, serviceResponses, http.StatusOK)
}

func (rt *Routes) CreateService(w http.ResponseWriter, r *http.Request) {
	// Get validated request from middleware
	req, ok := handlers.GetValidatedRequest(r.Context()).(*dto.ServiceCreateRequest)
	if !ok {
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get character ID from auth context
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	service, err := rt.granularPermissionService.CreateService(r.Context(), req, characterID)
	if err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := dto.ServiceResponse{
		ID:          service.ID.Hex(),
		Name:        service.Name,
		DisplayName: service.DisplayName,
		Description: service.Description,
		Resources:   service.Resources,
		Enabled:     service.Enabled,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

func (rt *Routes) GetService(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "serviceName")
	if serviceName == "" {
		handlers.ErrorResponse(w, "Service name is required", http.StatusBadRequest)
		return
	}

	service, err := rt.granularPermissionService.GetService(r.Context(), serviceName)
	if err != nil {
		handlers.ErrorResponse(w, "Service not found", http.StatusNotFound)
		return
	}

	response := dto.ServiceResponse{
		ID:          service.ID.Hex(),
		Name:        service.Name,
		DisplayName: service.DisplayName,
		Description: service.Description,
		Resources:   service.Resources,
		Enabled:     service.Enabled,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

func (rt *Routes) UpdateService(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "serviceName")
	if serviceName == "" {
		handlers.ErrorResponse(w, "Service name is required", http.StatusBadRequest)
		return
	}

	// Get validated request from middleware
	req, ok := handlers.GetValidatedRequest(r.Context()).(*dto.ServiceUpdateRequest)
	if !ok {
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get character ID from auth context
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	service, err := rt.granularPermissionService.UpdateService(r.Context(), serviceName, req, characterID)
	if err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := dto.ServiceResponse{
		ID:          service.ID.Hex(),
		Name:        service.Name,
		DisplayName: service.DisplayName,
		Description: service.Description,
		Resources:   service.Resources,
		Enabled:     service.Enabled,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

func (rt *Routes) DeleteService(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "serviceName")
	if serviceName == "" {
		handlers.ErrorResponse(w, "Service name is required", http.StatusBadRequest)
		return
	}

	// Get character ID from auth context
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := rt.granularPermissionService.DeleteService(r.Context(), serviceName, characterID); err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	handlers.JSONResponse(w, map[string]interface{}{
		"message": "Service deleted successfully",
	}, http.StatusOK)
}

func (rt *Routes) GrantPermission(w http.ResponseWriter, r *http.Request) {
	// Get validated request from middleware
	req, ok := handlers.GetValidatedRequest(r.Context()).(*dto.PermissionAssignmentRequest)
	if !ok {
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get character ID from auth context
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	assignment, err := rt.granularPermissionService.GrantPermission(r.Context(), req, characterID)
	if err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := dto.PermissionAssignmentResponse{
		ID:          assignment.ID.Hex(),
		Service:     assignment.Service,
		Resource:    assignment.Resource,
		Action:      assignment.Action,
		SubjectType: assignment.SubjectType,
		SubjectID:   assignment.SubjectID,
		GrantedBy:   assignment.GrantedBy,
		GrantedAt:   assignment.GrantedAt,
		ExpiresAt:   assignment.ExpiresAt,
		Reason:      assignment.Reason,
		Enabled:     assignment.Enabled,
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

func (rt *Routes) RevokePermission(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	service := r.URL.Query().Get("service")
	resource := r.URL.Query().Get("resource")
	action := r.URL.Query().Get("action")
	subjectType := r.URL.Query().Get("subject_type")
	subjectID := r.URL.Query().Get("subject_id")
	reason := r.URL.Query().Get("reason")

	if service == "" || resource == "" || action == "" || subjectType == "" || subjectID == "" {
		handlers.ErrorResponse(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	if reason == "" {
		reason = "Revoked by administrator"
	}

	// Get character ID from auth context
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := rt.granularPermissionService.RevokePermission(r.Context(), service, resource, action, subjectType, subjectID, characterID, reason); err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	handlers.JSONResponse(w, map[string]interface{}{
		"message": "Permission revoked successfully",
	}, http.StatusOK)
}

func (rt *Routes) ListPermissionAssignments(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page := 1
	pageSize := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Build filter from query parameters
	filter := make(map[string]interface{})
	if service := r.URL.Query().Get("service"); service != "" {
		filter["service"] = service
	}
	if resource := r.URL.Query().Get("resource"); resource != "" {
		filter["resource"] = resource
	}
	if subjectType := r.URL.Query().Get("subject_type"); subjectType != "" {
		filter["subject_type"] = subjectType
	}
	if subjectID := r.URL.Query().Get("subject_id"); subjectID != "" {
		filter["subject_id"] = subjectID
	}

	// TODO: Implement ListPermissionAssignments in repository
	// For now, return empty response
	handlers.JSONResponse(w, map[string]interface{}{
		"assignments": []interface{}{},
		"pagination": map[string]interface{}{
			"page":        page,
			"page_size":   pageSize,
			"total":       0,
			"total_pages": 0,
		},
	}, http.StatusOK)
}

func (rt *Routes) ListSubjectGroups(w http.ResponseWriter, r *http.Request) {
	query := &dto.GroupListQuery{
		Page:     1,
		PageSize: 1000, // Get all groups for subject selection
	}

	response, err := rt.groupService.ListGroups(r.Context(), query)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to list groups", http.StatusInternalServerError)
		return
	}

	// Convert to simple subject format
	var subjects []map[string]interface{}
	for _, group := range response.Groups {
		subjects = append(subjects, map[string]interface{}{
			"type": "group",
			"id":   group.ID,
			"name": group.Name,
		})
	}

	handlers.JSONResponse(w, subjects, http.StatusOK)
}

func (rt *Routes) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page := 1
	pageSize := 50

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Build filter from query parameters
	filter := make(map[string]interface{})
	if action := r.URL.Query().Get("action"); action != "" {
		filter["action"] = action
	}
	if service := r.URL.Query().Get("service"); service != "" {
		filter["service"] = service
	}

	// TODO: Implement GetAuditLogs in repository
	// For now, return empty response
	handlers.JSONResponse(w, map[string]interface{}{
		"logs": []interface{}{},
		"pagination": map[string]interface{}{
			"page":        page,
			"page_size":   pageSize,
			"total":       0,
			"total_pages": 0,
		},
	}, http.StatusOK)
}

func (rt *Routes) GetGroupStats(w http.ResponseWriter, r *http.Request) {
	stats, err := rt.groupService.GetMembershipStats(r.Context())
	if err != nil {
		handlers.ErrorResponse(w, "Failed to get group statistics", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, stats, http.StatusOK)
}