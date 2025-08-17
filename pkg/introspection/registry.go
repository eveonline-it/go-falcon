package introspection

import (
	"reflect"
	
	// Import existing DTOs
	authDTO "go-falcon/internal/auth/dto"
	groupsDTO "go-falcon/internal/groups/dto"
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
	// Basic auth routes
	r.register("POST", "/register", authDTO.RegisterRequest{}, authDTO.TokenResponse{})
	r.register("POST", "/login", authDTO.LoginRequest{}, authDTO.TokenResponse{})
	
	// EVE SSO routes
	r.register("GET", "/eve/login", nil, authDTO.EVELoginResponse{})
	r.register("GET", "/eve/callback", nil, authDTO.TokenResponse{})
	r.register("POST", "/eve/refresh", authDTO.RefreshTokenRequest{}, authDTO.RefreshTokenResponse{})
	r.register("POST", "/eve/token", authDTO.EVETokenExchangeRequest{}, authDTO.TokenResponse{})
	
	// Profile routes
	r.register("GET", "/profile", nil, authDTO.ProfileResponse{})
	r.register("GET", "/profile/public", nil, authDTO.PublicProfileResponse{})
	r.register("POST", "/profile/refresh", authDTO.ProfileRefreshRequest{}, authDTO.ProfileResponse{})
	
	// Session routes
	r.register("GET", "/status", nil, authDTO.AuthStatusResponse{})
	r.register("GET", "/user", nil, authDTO.UserInfoResponse{})
	r.register("POST", "/logout", nil, authDTO.LogoutResponse{})
	r.register("GET", "/token", nil, authDTO.TokenResponse{})
}

func (r *RouteRegistry) registerGroupsRoutes() {
	// Group management - using existing DTOs
	r.register("GET", "/groups", nil, groupsDTO.GroupListResponse{})
	r.register("POST", "/groups", groupsDTO.GroupCreateRequest{}, groupsDTO.GroupResponse{})
	r.register("PUT", "/groups/{groupID}", groupsDTO.GroupUpdateRequest{}, groupsDTO.GroupResponse{})
	r.register("DELETE", "/groups/{groupID}", nil, nil)
	
	// Member management  
	r.register("GET", "/groups/{groupID}/members", nil, groupsDTO.GroupMemberListResponse{})
	r.register("POST", "/groups/{groupID}/members", groupsDTO.MembershipRequest{}, groupsDTO.MembershipResponse{})
	r.register("DELETE", "/groups/{groupID}/members/{memberID}", nil, nil)
	
	// Permission checking
	r.register("GET", "/permissions/check", nil, groupsDTO.UserPermissionsResponse{}) // Basic permission check returns user permissions
	r.register("GET", "/permissions/user", nil, groupsDTO.UserPermissionsResponse{})
	
	// Admin: Service management  
	r.register("GET", "/admin/permissions/services", nil, groupsDTO.ServiceListResponse{})
	r.register("POST", "/admin/permissions/services", groupsDTO.ServiceCreateRequest{}, groupsDTO.ServiceResponse{})
	r.register("GET", "/admin/permissions/services/{serviceName}", nil, groupsDTO.ServiceResponse{})
	r.register("PUT", "/admin/permissions/services/{serviceName}", groupsDTO.ServiceUpdateRequest{}, groupsDTO.ServiceResponse{})
	r.register("DELETE", "/admin/permissions/services/{serviceName}", nil, nil)
	
	// Admin: Permission assignments
	r.register("GET", "/admin/permissions/assignments", nil, groupsDTO.PermissionAssignmentListResponse{})
	r.register("POST", "/admin/permissions/assignments", groupsDTO.PermissionAssignmentRequest{}, groupsDTO.PermissionAssignmentResponse{})
	r.register("POST", "/admin/permissions/assignments/bulk", groupsDTO.BulkPermissionRequest{}, groupsDTO.BulkOperationResponse{})
	r.register("DELETE", "/admin/permissions/assignments/{assignmentID}", nil, nil)
	
	// Admin: Permission checking
	r.register("POST", "/admin/permissions/check", groupsDTO.PermissionCheckGranularRequest{}, groupsDTO.UserPermissionsResponse{})
	r.register("GET", "/admin/permissions/check/user/{characterID}", nil, groupsDTO.UserPermissionSummaryResponse{})
	r.register("GET", "/admin/permissions/check/service/{serviceName}", nil, groupsDTO.ServicePermissionSummaryResponse{})
	
	// Admin: Subject management
	r.register("GET", "/admin/permissions/subjects/groups", nil, groupsDTO.SubjectListResponse{})
	r.register("POST", "/admin/permissions/subjects/validate", groupsDTO.SubjectValidationRequest{}, groupsDTO.SubjectResponse{})
	
	// Admin: Audit
	r.register("GET", "/admin/permissions/audit", nil, groupsDTO.AuditLogResponse{})
}