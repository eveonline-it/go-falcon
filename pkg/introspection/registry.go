package introspection

import (
	"reflect"
	
	// Import existing DTOs
	authDTO "go-falcon/internal/auth/dto"
	groupsDTO "go-falcon/internal/groups/dto"
	usersDTO "go-falcon/internal/users/dto"
	schedulerDTO "go-falcon/internal/scheduler/dto"
	sdeDTO "go-falcon/internal/sde/dto"
	devDTO "go-falcon/internal/dev/dto"
	notificationsDTO "go-falcon/internal/notifications/dto"
)

// RouteSchema holds request and response schemas for a route
type RouteSchema struct {
	Request  *OpenAPISchema
	Response *OpenAPISchema
	ErrorResponses map[int]*OpenAPISchema
}

// RouteRegistry maps route patterns to their DTO types
type RouteRegistry struct {
	routes map[string]map[string]RouteSchema // map[method]map[path]RouteSchema
}

// NewRouteRegistry creates a new route registry with all DTOs registered
func NewRouteRegistry() *RouteRegistry {
	r := &RouteRegistry{
		routes: make(map[string]map[string]RouteSchema),
	}
	
	// Register existing routes
	r.registerAuthRoutes()
	r.registerGroupsRoutes()
	r.registerUsersRoutes()
	r.registerSchedulerRoutes()
	r.registerSdeRoutes()
	r.registerDevRoutes()
	r.registerNotificationsRoutes()
	
	return r
}

// GetRouteSchema returns the schema for a specific route
func (r *RouteRegistry) GetRouteSchema(method, path string) (RouteSchema, bool) {
	if methodRoutes, ok := r.routes[method]; ok {
		if schema, ok := methodRoutes[path]; ok {
			return schema, true
		}
	}
	return RouteSchema{}, false
}

// register adds a route schema to the registry
func (r *RouteRegistry) register(method, path string, requestType, responseType interface{}) {
	if r.routes[method] == nil {
		r.routes[method] = make(map[string]RouteSchema)
	}
	
	schema := RouteSchema{}
	
	if requestType != nil {
		schema.Request = GenerateSchemaFromType(reflect.TypeOf(requestType))
	}
	
	if responseType != nil {
		schema.Response = GenerateSchemaFromType(reflect.TypeOf(responseType))
	}
	
	r.routes[method][path] = schema
}

// registerWithErrors adds a route schema with error responses
func (r *RouteRegistry) registerWithErrors(method, path string, requestType, responseType interface{}, errors map[int]interface{}) {
	if r.routes[method] == nil {
		r.routes[method] = make(map[string]RouteSchema)
	}
	
	schema := RouteSchema{
		ErrorResponses: make(map[int]*OpenAPISchema),
	}
	
	if requestType != nil {
		schema.Request = GenerateSchemaFromType(reflect.TypeOf(requestType))
	}
	
	if responseType != nil {
		schema.Response = GenerateSchemaFromType(reflect.TypeOf(responseType))
	}
	
	for code, errorType := range errors {
		schema.ErrorResponses[code] = GenerateSchemaFromType(reflect.TypeOf(errorType))
	}
	
	r.routes[method][path] = schema
}

func (r *RouteRegistry) registerAuthRoutes() {
	// Health check
	r.register("GET", "/health", nil, nil)
	
	// Basic auth endpoints
	r.register("POST", "/login", authDTO.RefreshTokenRequest{}, authDTO.TokenResponse{})
	r.register("POST", "/register", authDTO.RefreshTokenRequest{}, authDTO.TokenResponse{})
	
	// EVE SSO routes (public)
	r.register("GET", "/eve/login", nil, authDTO.EVELoginResponse{})
	r.register("GET", "/eve/register", nil, authDTO.EVELoginResponse{})
	r.register("GET", "/eve/callback", nil, authDTO.TokenResponse{})
	r.register("POST", "/eve/refresh", authDTO.RefreshTokenRequest{}, authDTO.RefreshTokenResponse{})
	r.register("GET", "/eve/verify", nil, authDTO.TokenResponse{})
	r.register("POST", "/eve/token", authDTO.EVETokenExchangeRequest{}, authDTO.TokenResponse{})
	
	// Authentication status (public)
	r.register("GET", "/status", nil, authDTO.AuthStatusResponse{})
	r.register("GET", "/user", nil, authDTO.UserInfoResponse{})
	
	// Profile routes (protected)
	r.register("GET", "/profile", nil, authDTO.ProfileResponse{})
	r.register("POST", "/profile/refresh", authDTO.ProfileRefreshRequest{}, authDTO.ProfileResponse{})
	r.register("GET", "/token", nil, authDTO.TokenResponse{})
	
	// Public profile
	r.register("GET", "/profile/public", nil, authDTO.PublicProfileResponse{})
	
	// Logout
	r.register("POST", "/logout", nil, authDTO.LogoutResponse{})
}

func (r *RouteRegistry) registerGroupsRoutes() {
	// Health check
	r.register("GET", "/health", nil, nil)
	
	// Group management  
	r.register("GET", "/", groupsDTO.GroupListQuery{}, groupsDTO.GroupListResponse{})
	r.register("GET", "/{groupID}", nil, groupsDTO.GroupResponse{})
	r.register("POST", "/", groupsDTO.GroupCreateRequest{}, groupsDTO.GroupResponse{})
	r.register("PUT", "/{groupID}", groupsDTO.GroupUpdateRequest{}, groupsDTO.GroupResponse{})
	r.register("DELETE", "/{groupID}", nil, nil)
	
	// Member management  
	r.register("GET", "/{groupID}/members", nil, groupsDTO.GroupMemberListResponse{})
	r.register("POST", "/{groupID}/members", groupsDTO.MembershipRequest{}, groupsDTO.MembershipResponse{})
	r.register("DELETE", "/{groupID}/members/{characterID}", nil, nil)
	
	// Permission checking (only granular check is implemented)
	r.register("POST", "/permissions/granular/check", groupsDTO.PermissionCheckGranularRequest{}, groupsDTO.UserPermissionsResponse{})
	
	// Admin endpoints (super admin only)
	// Service management
	r.register("GET", "/admin/services", nil, groupsDTO.ServiceListResponse{})
	r.register("POST", "/admin/services", groupsDTO.ServiceCreateRequest{}, groupsDTO.ServiceResponse{})
	r.register("GET", "/admin/services/{serviceName}", nil, groupsDTO.ServiceResponse{})
	r.register("PUT", "/admin/services/{serviceName}", groupsDTO.ServiceUpdateRequest{}, groupsDTO.ServiceResponse{})
	r.register("DELETE", "/admin/services/{serviceName}", nil, nil)
	
	// Permission assignment management 
	r.register("POST", "/admin/permissions", groupsDTO.PermissionAssignmentRequest{}, groupsDTO.PermissionAssignmentResponse{})
	r.register("DELETE", "/admin/permissions", nil, nil)
	r.register("GET", "/admin/permissions/assignments", nil, groupsDTO.PermissionAssignmentListResponse{})
	
	// Utility endpoints
	r.register("GET", "/admin/subjects/groups", nil, groupsDTO.SubjectListResponse{})
	r.register("GET", "/admin/audit", nil, groupsDTO.AuditLogResponse{})
	r.register("GET", "/admin/stats", nil, groupsDTO.GroupStatsResponse{})
}

func (r *RouteRegistry) registerUsersRoutes() {
	// Health check
	r.register("GET", "/health", nil, nil)
	
	// User management routes
	r.register("GET", "/", usersDTO.UserSearchRequest{}, usersDTO.UserListResponse{})
	r.register("GET", "/{character_id}", nil, usersDTO.UserResponse{})
	r.register("PUT", "/{character_id}", usersDTO.UserUpdateRequest{}, usersDTO.UserUpdateResponse{})
	r.register("GET", "/stats", nil, usersDTO.UserStatsResponse{})
	r.register("GET", "/by-user-id/{user_id}/characters", nil, usersDTO.CharacterListResponse{})
}

func (r *RouteRegistry) registerSchedulerRoutes() {
	// Health check
	r.register("GET", "/health", nil, nil)
	
	// Task management routes
	r.register("GET", "/tasks", schedulerDTO.TaskListQuery{}, schedulerDTO.TaskListResponse{})
	r.register("POST", "/tasks", schedulerDTO.TaskCreateRequest{}, schedulerDTO.TaskResponse{})
	r.register("GET", "/tasks/{taskID}", nil, schedulerDTO.TaskResponse{})
	r.register("PUT", "/tasks/{taskID}", schedulerDTO.TaskUpdateRequest{}, schedulerDTO.TaskResponse{})
	r.register("DELETE", "/tasks/{taskID}", nil, nil)
	
	// Task control routes
	r.register("POST", "/tasks/{taskID}/start", schedulerDTO.ManualExecutionRequest{}, schedulerDTO.ExecutionResponse{})
	r.register("POST", "/tasks/{taskID}/stop", nil, schedulerDTO.TaskExecutionResponse{})
	r.register("POST", "/tasks/{taskID}/pause", nil, schedulerDTO.TaskExecutionResponse{})
	r.register("POST", "/tasks/{taskID}/resume", nil, schedulerDTO.TaskExecutionResponse{})
	
	// Execution history routes
	r.register("GET", "/tasks/{taskID}/history", schedulerDTO.TaskExecutionQuery{}, schedulerDTO.ExecutionListResponse{})
	r.register("GET", "/tasks/{taskID}/executions/{executionID}", nil, schedulerDTO.ExecutionResponse{})
	
	// System routes
	r.register("GET", "/stats", nil, schedulerDTO.SchedulerStatsResponse{})
	r.register("POST", "/reload", nil, schedulerDTO.BulkOperationResponse{})
	r.register("GET", "/status", nil, schedulerDTO.SchedulerStatusResponse{})
}

func (r *RouteRegistry) registerSdeRoutes() {
	// SDE module currently only has basic routes implemented
	r.register("GET", "/status", nil, sdeDTO.SDEStatusResponse{})
	r.register("GET", "/health", nil, nil)
}

func (r *RouteRegistry) registerDevRoutes() {
	// Health check
	r.register("GET", "/health", nil, nil)
	
	// ESI Server and character endpoints
	r.register("GET", "/esi-status", nil, devDTO.ESIStatusResponse{})
	r.register("GET", "/character/{characterID}", nil, devDTO.CharacterResponse{})
	r.register("GET", "/character/{characterID}/portrait", nil, devDTO.CharacterResponse{})
	
	// Universe endpoints
	r.register("GET", "/universe/system/{systemID}", nil, devDTO.SystemResponse{})
	r.register("GET", "/universe/station/{stationID}", nil, devDTO.SystemResponse{})
	
	// Alliance endpoints
	r.register("GET", "/alliances", nil, devDTO.DevResponse{})
	r.register("GET", "/alliance/{allianceID}", nil, devDTO.AllianceResponse{})
	r.register("GET", "/alliance/{allianceID}/contacts", nil, devDTO.DevResponse{})
	r.register("GET", "/alliance/{allianceID}/contacts/labels", nil, devDTO.DevResponse{})
	r.register("GET", "/alliance/{allianceID}/corporations", nil, devDTO.DevResponse{})
	r.register("GET", "/alliance/{allianceID}/icons", nil, devDTO.DevResponse{})
	
	// Corporation endpoints
	r.register("GET", "/corporation/{corporationID}", nil, devDTO.CorporationResponse{})
	r.register("GET", "/corporation/{corporationID}/icons", nil, devDTO.DevResponse{})
	r.register("GET", "/corporation/{corporationID}/alliancehistory", nil, devDTO.DevResponse{})
	r.register("GET", "/corporation/{corporationID}/members", nil, devDTO.DevResponse{})
	r.register("GET", "/corporation/{corporationID}/membertracking", nil, devDTO.DevResponse{})
	r.register("GET", "/corporation/{corporationID}/roles", nil, devDTO.DevResponse{})
	r.register("GET", "/corporation/{corporationID}/structures", nil, devDTO.DevResponse{})
	r.register("GET", "/corporation/{corporationID}/standings", nil, devDTO.DevResponse{})
	r.register("GET", "/corporation/{corporationID}/wallets", nil, devDTO.DevResponse{})
	
	// SDE endpoints
	r.register("GET", "/sde/status", nil, devDTO.SDEStatusResponse{})
	r.register("GET", "/sde/agent/{agentID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/category/{categoryID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/blueprint/{blueprintID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/agents/location/{locationID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/blueprints", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/marketgroup/{marketGroupID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/marketgroups", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/metagroup/{metaGroupID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/metagroups", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/npccorp/{corpID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/npccorps", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/npccorps/faction/{factionID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/typeid/{typeID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/type/{typeID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/types", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/types/published", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/types/group/{groupID}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/typematerials/{typeID}", nil, devDTO.DevResponse{})
	
	// Redis SDE endpoints
	r.register("GET", "/sde/redis/{type}/{id}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/redis/{type}", nil, devDTO.DevResponse{})
	
	// Universe SDE endpoints
	r.register("GET", "/sde/universe/{universeType}/{regionName}/systems", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/universe/{universeType}/{regionName}/{constellationName}/systems", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/universe/{universeType}/{regionName}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/universe/{universeType}/{regionName}/{constellationName}", nil, devDTO.DevResponse{})
	r.register("GET", "/sde/universe/{universeType}/{regionName}/{constellationName}/{systemName}", nil, devDTO.DevResponse{})
	
	// Service endpoints
	r.register("GET", "/services", nil, devDTO.ServiceDiscoveryResponse{})
	r.register("GET", "/status", nil, devDTO.HealthResponse{})
}

func (r *RouteRegistry) registerNotificationsRoutes() {
	// Health check
	r.register("GET", "/health", nil, nil)
	
	// Notification management routes
	r.register("GET", "/", notificationsDTO.NotificationSearchRequest{}, notificationsDTO.NotificationListResponse{})
	r.register("POST", "/", notificationsDTO.NotificationRequest{}, notificationsDTO.NotificationCreateResponse{})
	r.register("PUT", "/{id}", notificationsDTO.NotificationUpdateRequest{}, notificationsDTO.NotificationUpdateResponse{})
}