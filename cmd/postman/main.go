package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go-falcon/pkg/config"
	"go-falcon/pkg/version"
	"go-falcon/pkg/introspection"
)

// PostmanCollection represents the top-level Postman collection structure
type PostmanCollection struct {
	Info      PostmanInfo         `json:"info"`
	Item      []PostmanItem       `json:"item"`
	Auth      *PostmanAuth        `json:"auth,omitempty"`
	Event     []PostmanEvent      `json:"event,omitempty"`
	Variable  []PostmanVariable   `json:"variable"`
}

// PostmanInfo contains collection metadata
type PostmanInfo struct {
	PostmanID   string `json:"_postman_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      string `json:"schema"`
	ExporterID  string `json:"_exporter_id,omitempty"`
}

// PostmanItem represents a folder or request in the collection
type PostmanItem struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Item        []PostmanItem          `json:"item,omitempty"`
	Request     *PostmanRequest        `json:"request,omitempty"`
	Response    []PostmanResponse      `json:"response,omitempty"`
}

// PostmanRequest represents an HTTP request
type PostmanRequest struct {
	Method      string            `json:"method"`
	Header      []PostmanHeader   `json:"header,omitempty"`
	Body        *PostmanBody      `json:"body,omitempty"`
	URL         PostmanURL        `json:"url"`
	Description string            `json:"description,omitempty"`
	Auth        *PostmanAuth      `json:"auth,omitempty"`
}

// PostmanURL represents the request URL structure
type PostmanURL struct {
	Raw      string            `json:"raw"`
	Host     []string          `json:"host"`
	Path     []string          `json:"path"`
	Query    []PostmanQuery    `json:"query,omitempty"`
}

// PostmanHeader represents an HTTP header
type PostmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// PostmanQuery represents a URL query parameter
type PostmanQuery struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// PostmanBody represents request body
type PostmanBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw,omitempty"`
}

// PostmanResponse represents example responses
type PostmanResponse struct {
	Name            string          `json:"name"`
	OriginalRequest PostmanRequest  `json:"originalRequest"`
	Status          string          `json:"status"`
	Code            int             `json:"code"`
	Header          []PostmanHeader `json:"header"`
	Cookie          []interface{}   `json:"cookie"`
	Body            string          `json:"body"`
}

// PostmanAuth represents authentication configuration
type PostmanAuth struct {
	Type   string                 `json:"type"`
	Bearer []PostmanAuthBearer    `json:"bearer,omitempty"`
}

// PostmanAuthBearer represents bearer token auth
type PostmanAuthBearer struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// PostmanEvent represents collection-level scripts
type PostmanEvent struct {
	Listen string        `json:"listen"`
	Script PostmanScript `json:"script"`
}

// PostmanScript represents JavaScript code
type PostmanScript struct {
	Type string   `json:"type"`
	Exec []string `json:"exec"`
}

// PostmanVariable represents collection variables
type PostmanVariable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// RouteInfo holds information about discovered routes
type RouteInfo struct {
	Method      string
	Path        string
	ModuleName  string
	HandlerName string
	Description string
}

func main() {
	fmt.Println("🚀 Go-Falcon Postman Collection Exporter")
	
	versionInfo := version.Get()
	fmt.Printf("📦 Version: %s\n", version.GetVersionString())
	fmt.Printf("🔧 Build: %s (%s)\n", versionInfo.BuildDate, versionInfo.Platform)
	
	// Initialize modules to discover routes
	routes, err := discoverRoutes()
	if err != nil {
		log.Fatalf("❌ Failed to discover routes: %v", err)
	}
	
	fmt.Printf("📋 Discovered %d routes across %d modules\n", 
		len(routes), countUniqueModules(routes))
	
	// Generate Postman collection
	collection := generatePostmanCollection(routes)
	
	// Export to JSON file
	filename := "falcon-postman.json"
	if err := exportCollection(collection, filename); err != nil {
		log.Fatalf("❌ Failed to export collection: %v", err)
	}
	
	fmt.Printf("✅ Postman collection exported to: %s\n", filename)
	fmt.Printf("📊 Collection contains %d endpoints organized in %d modules\n", 
		countTotalRequests(collection), len(collection.Item))
}

// discoverRoutes uses static route definitions instead of module initialization
func discoverRoutes() ([]RouteInfo, error) {
	var routes []RouteInfo
	
	// Use static route definitions for all modules to avoid environment dependencies
	moduleRoutes := map[string][]RouteInfo{
		"auth": getAuthRoutes(),
		"groups": getGroupsRoutes(),
		"dev": getDevRoutes(),
		"users": getUsersRoutes(),
		"notifications": getNotificationsRoutes(),
		"scheduler": getSchedulerRoutes(),
		"sde": getSdeRoutes(),
	}
	
	// Collect routes from all modules
	for moduleName, moduleRouteList := range moduleRoutes {
		fmt.Printf("🔍 Discovering routes for module: %s\n", moduleName)
		routes = append(routes, moduleRouteList...)
	}
	
	// Add gateway-level routes
	gatewayRoutes := []RouteInfo{
		{
			Method:      "GET",
			Path:        "/health",
			ModuleName:  "gateway",
			HandlerName: "enhancedHealthHandler",
			Description: "Gateway health check with version information",
		},
	}
	routes = append(routes, gatewayRoutes...)
	
	return routes, nil
}

// getAuthRoutes returns static route definitions for the auth module
func getAuthRoutes() []RouteInfo {
	return []RouteInfo{
		{Method: "GET", Path: "/health", ModuleName: "auth", HandlerName: "HealthHandler", Description: "Auth module health check"},
		// Basic auth endpoints
		{Method: "POST", Path: "/login", ModuleName: "auth", HandlerName: "loginHandler", Description: "Basic login endpoint"},
		{Method: "POST", Path: "/register", ModuleName: "auth", HandlerName: "registerHandler", Description: "User registration endpoint"},
		{Method: "GET", Path: "/status", ModuleName: "auth", HandlerName: "statusHandler", Description: "Check authentication status"},
		// EVE SSO endpoints  
		{Method: "GET", Path: "/eve/login", ModuleName: "auth", HandlerName: "eveLoginHandler", Description: "Initiate EVE SSO login"},
		{Method: "GET", Path: "/eve/callback", ModuleName: "auth", HandlerName: "eveCallbackHandler", Description: "Handle EVE SSO callback"},
		{Method: "POST", Path: "/eve/refresh", ModuleName: "auth", HandlerName: "eveRefreshHandler", Description: "Refresh access token"},
		{Method: "GET", Path: "/eve/verify", ModuleName: "auth", HandlerName: "eveVerifyHandler", Description: "Verify JWT token"},
		// Profile endpoints (public)
		{Method: "GET", Path: "/profile/public", ModuleName: "auth", HandlerName: "publicProfileHandler", Description: "Get public profile by ID"},
		// Protected endpoints (require JWT)
		{Method: "GET", Path: "/user", ModuleName: "auth", HandlerName: "getCurrentUserHandler", Description: "Get current user information"},
		{Method: "GET", Path: "/status", ModuleName: "auth", HandlerName: "authStatusHandler", Description: "Get authentication status"},
		{Method: "POST", Path: "/logout", ModuleName: "auth", HandlerName: "logoutHandler", Description: "User logout"},
		{Method: "GET", Path: "/profile", ModuleName: "auth", HandlerName: "profileHandler", Description: "Get user profile"},
		{Method: "POST", Path: "/profile/refresh", ModuleName: "auth", HandlerName: "profileRefreshHandler", Description: "Refresh profile from ESI"},
		{Method: "GET", Path: "/token", ModuleName: "auth", HandlerName: "tokenHandler", Description: "Retrieve current bearer token"},
	}
}

// getGroupsRoutes returns static route definitions for the groups module
func getGroupsRoutes() []RouteInfo {
	return []RouteInfo{
		{Method: "GET", Path: "/health", ModuleName: "groups", HandlerName: "HealthHandler", Description: "Groups module health check"},
		
		// Legacy Group Management endpoints
		{Method: "GET", Path: "/groups", ModuleName: "groups", HandlerName: "listGroupsHandler", Description: "List all groups with user's membership status"},
		{Method: "POST", Path: "/groups", ModuleName: "groups", HandlerName: "createGroupHandler", Description: "Create a new custom group (admin only)"},
		{Method: "PUT", Path: "/groups/{groupsID}", ModuleName: "groups", HandlerName: "updateGroupHandler", Description: "Update an existing group (admin only)"},
		{Method: "DELETE", Path: "/groups/{groupsID}", ModuleName: "groups", HandlerName: "deleteGroupHandler", Description: "Delete a custom group (admin only)"},
		
		// Legacy Group Membership endpoints
		{Method: "GET", Path: "/groups/{groupsID}/members", ModuleName: "groups", HandlerName: "listMembersHandler", Description: "List all members of a group (admin only)"},
		{Method: "POST", Path: "/groups/{groupsID}/members", ModuleName: "groups", HandlerName: "addMemberHandler", Description: "Add a member to a group (admin only)"},
		{Method: "DELETE", Path: "/groups/{groupsID}/members/{characterID}", ModuleName: "groups", HandlerName: "removeMemberHandler", Description: "Remove a member from a group (admin only)"},
		
		// Legacy Permission endpoints
		{Method: "GET", Path: "/permissions/check", ModuleName: "groups", HandlerName: "checkPermissionHandler", Description: "Check if user has specific permission (legacy system)"},
		{Method: "GET", Path: "/permissions/user", ModuleName: "groups", HandlerName: "getUserPermissionsHandler", Description: "Get user's complete permission matrix (legacy system)"},
		
		// NEW Granular Permission System - Service Management
		{Method: "GET", Path: "/admin/permissions/services", ModuleName: "groups", HandlerName: "listServicesHandler", Description: "List all permission services (super admin only)"},
		{Method: "POST", Path: "/admin/permissions/services", ModuleName: "groups", HandlerName: "createServiceHandler", Description: "Create a new permission service (super admin only)"},
		{Method: "GET", Path: "/admin/permissions/services/{serviceName}", ModuleName: "groups", HandlerName: "getServiceHandler", Description: "Get specific permission service details (super admin only)"},
		{Method: "PUT", Path: "/admin/permissions/services/{serviceName}", ModuleName: "groups", HandlerName: "updateServiceHandler", Description: "Update permission service configuration (super admin only)"},
		{Method: "DELETE", Path: "/admin/permissions/services/{serviceName}", ModuleName: "groups", HandlerName: "deleteServiceHandler", Description: "Delete permission service and all assignments (super admin only)"},
		
		// NEW Granular Permission System - Permission Assignments
		{Method: "GET", Path: "/admin/permissions/assignments", ModuleName: "groups", HandlerName: "listPermissionAssignmentsHandler", Description: "List permission assignments with filtering (super admin only)"},
		{Method: "POST", Path: "/admin/permissions/assignments", ModuleName: "groups", HandlerName: "grantPermissionHandler", Description: "Grant a granular permission to a subject (super admin only)"},
		{Method: "POST", Path: "/admin/permissions/assignments/bulk", ModuleName: "groups", HandlerName: "bulkGrantPermissionsHandler", Description: "Grant multiple permissions in bulk (super admin only)"},
		{Method: "DELETE", Path: "/admin/permissions/assignments/{assignmentID}", ModuleName: "groups", HandlerName: "revokePermissionHandler", Description: "Revoke a specific permission assignment (super admin only)"},
		
		// NEW Granular Permission System - Permission Checking
		{Method: "POST", Path: "/admin/permissions/check", ModuleName: "groups", HandlerName: "adminCheckPermissionHandler", Description: "Check granular permission for any user (super admin only)"},
		{Method: "GET", Path: "/admin/permissions/check/user/{characterID}", ModuleName: "groups", HandlerName: "getUserPermissionSummaryHandler", Description: "Get comprehensive permission summary for user (super admin only)"},
		{Method: "GET", Path: "/admin/permissions/check/service/{serviceName}", ModuleName: "groups", HandlerName: "getServicePermissionsHandler", Description: "Get all permissions for a specific service (super admin only)"},
		
		// NEW Granular Permission System - Utility Endpoints
		{Method: "GET", Path: "/admin/permissions/subjects/groups", ModuleName: "groups", HandlerName: "listGroupSubjectsHandler", Description: "List available groups for permission assignment (super admin only)"},
		{Method: "GET", Path: "/admin/permissions/subjects/validate", ModuleName: "groups", HandlerName: "validateSubjectHandler", Description: "Validate if a subject exists for permission assignment (super admin only)"},
		
		// NEW Granular Permission System - Audit & Monitoring
		{Method: "GET", Path: "/admin/permissions/audit", ModuleName: "groups", HandlerName: "getPermissionAuditLogsHandler", Description: "Get permission audit logs with filtering (super admin only)"},
	}
}

// getDevRoutes returns static route definitions for the dev module
func getDevRoutes() []RouteInfo {
	return []RouteInfo{
		{Method: "GET", Path: "/health", ModuleName: "dev", HandlerName: "HealthHandler", Description: "Dev module health check"},
		// ESI Server and character endpoints
		{Method: "GET", Path: "/esi-status", ModuleName: "dev", HandlerName: "esiStatusHandler", Description: "Get EVE Online server status"},
		{Method: "GET", Path: "/character/{characterID}", ModuleName: "dev", HandlerName: "characterInfoHandler", Description: "Get character information"},
		{Method: "GET", Path: "/character/{characterID}/portrait", ModuleName: "dev", HandlerName: "characterPortraitHandler", Description: "Get character portrait URLs"},
		// Universe endpoints
		{Method: "GET", Path: "/universe/system/{systemID}", ModuleName: "dev", HandlerName: "systemInfoHandler", Description: "Get solar system information"},
		{Method: "GET", Path: "/universe/station/{stationID}", ModuleName: "dev", HandlerName: "stationInfoHandler", Description: "Get station information"},
		// Alliance endpoints
		{Method: "GET", Path: "/alliances", ModuleName: "dev", HandlerName: "alliancesHandler", Description: "Get all active alliances"},
		{Method: "GET", Path: "/alliance/{allianceID}", ModuleName: "dev", HandlerName: "allianceInfoHandler", Description: "Get alliance information"},
		{Method: "GET", Path: "/alliance/{allianceID}/contacts", ModuleName: "dev", HandlerName: "allianceContactsHandler", Description: "Get alliance contacts (requires auth)"},
		{Method: "GET", Path: "/alliance/{allianceID}/contacts/labels", ModuleName: "dev", HandlerName: "allianceContactLabelsHandler", Description: "Get alliance contact labels (requires auth)"},
		{Method: "GET", Path: "/alliance/{allianceID}/corporations", ModuleName: "dev", HandlerName: "allianceCorporationsHandler", Description: "Get alliance member corporations"},
		{Method: "GET", Path: "/alliance/{allianceID}/icons", ModuleName: "dev", HandlerName: "allianceIconsHandler", Description: "Get alliance icon URLs"},
		// Corporation endpoints
		{Method: "GET", Path: "/corporation/{corporationID}", ModuleName: "dev", HandlerName: "corporationInfoHandler", Description: "Get corporation information"},
		{Method: "GET", Path: "/corporation/{corporationID}/icons", ModuleName: "dev", HandlerName: "corporationIconsHandler", Description: "Get corporation icon URLs"},
		{Method: "GET", Path: "/corporation/{corporationID}/alliancehistory", ModuleName: "dev", HandlerName: "corporationAllianceHistoryHandler", Description: "Get corporation alliance history"},
		{Method: "GET", Path: "/corporation/{corporationID}/members", ModuleName: "dev", HandlerName: "corporationMembersHandler", Description: "Get corporation members (requires auth)"},
		{Method: "GET", Path: "/corporation/{corporationID}/membertracking", ModuleName: "dev", HandlerName: "corporationMemberTrackingHandler", Description: "Get corporation member tracking (requires auth)"},
		{Method: "GET", Path: "/corporation/{corporationID}/roles", ModuleName: "dev", HandlerName: "corporationMemberRolesHandler", Description: "Get corporation member roles (requires auth)"},
		{Method: "GET", Path: "/corporation/{corporationID}/structures", ModuleName: "dev", HandlerName: "corporationStructuresHandler", Description: "Get corporation structures (requires auth)"},
		{Method: "GET", Path: "/corporation/{corporationID}/standings", ModuleName: "dev", HandlerName: "corporationStandingsHandler", Description: "Get corporation standings (requires auth)"},
		{Method: "GET", Path: "/corporation/{corporationID}/wallets", ModuleName: "dev", HandlerName: "corporationWalletsHandler", Description: "Get corporation wallets (requires auth)"},
		// SDE endpoints
		{Method: "GET", Path: "/sde/status", ModuleName: "dev", HandlerName: "sdeStatusHandler", Description: "Get SDE service status and statistics"},
		{Method: "GET", Path: "/sde/agent/{agentID}", ModuleName: "dev", HandlerName: "sdeAgentHandler", Description: "Get agent information from SDE"},
		{Method: "GET", Path: "/sde/category/{categoryID}", ModuleName: "dev", HandlerName: "sdeCategoryHandler", Description: "Get category information from SDE"},
		{Method: "GET", Path: "/sde/blueprint/{blueprintID}", ModuleName: "dev", HandlerName: "sdeBlueprintHandler", Description: "Get blueprint information from SDE"},
		{Method: "GET", Path: "/sde/agents/location/{locationID}", ModuleName: "dev", HandlerName: "sdeAgentsByLocationHandler", Description: "Get agents by location from SDE"},
		{Method: "GET", Path: "/sde/blueprints", ModuleName: "dev", HandlerName: "sdeBlueprintIdsHandler", Description: "Get all available blueprint IDs from SDE"},
		{Method: "GET", Path: "/sde/marketgroup/{marketGroupID}", ModuleName: "dev", HandlerName: "sdeMarketGroupHandler", Description: "Get market group information from SDE"},
		{Method: "GET", Path: "/sde/marketgroups", ModuleName: "dev", HandlerName: "sdeMarketGroupsHandler", Description: "Get all market groups from SDE"},
		{Method: "GET", Path: "/sde/metagroup/{metaGroupID}", ModuleName: "dev", HandlerName: "sdeMetaGroupHandler", Description: "Get meta group information from SDE"},
		{Method: "GET", Path: "/sde/metagroups", ModuleName: "dev", HandlerName: "sdeMetaGroupsHandler", Description: "Get all meta groups from SDE"},
		{Method: "GET", Path: "/sde/npccorp/{corpID}", ModuleName: "dev", HandlerName: "sdeNPCCorpHandler", Description: "Get NPC corporation information from SDE"},
		{Method: "GET", Path: "/sde/npccorps", ModuleName: "dev", HandlerName: "sdeNPCCorpsHandler", Description: "Get all NPC corporations from SDE"},
		{Method: "GET", Path: "/sde/npccorps/faction/{factionID}", ModuleName: "dev", HandlerName: "sdeNPCCorpsByFactionHandler", Description: "Get NPC corporations by faction from SDE"},
		{Method: "GET", Path: "/sde/typeid/{typeID}", ModuleName: "dev", HandlerName: "sdeTypeIDHandler", Description: "Get type ID information from SDE"},
		{Method: "GET", Path: "/sde/type/{typeID}", ModuleName: "dev", HandlerName: "sdeTypeHandler", Description: "Get type information from SDE"},
		{Method: "GET", Path: "/sde/types", ModuleName: "dev", HandlerName: "sdeTypesHandler", Description: "Get all types from SDE"},
		{Method: "GET", Path: "/sde/types/published", ModuleName: "dev", HandlerName: "sdePublishedTypesHandler", Description: "Get all published types from SDE"},
		{Method: "GET", Path: "/sde/types/group/{groupID}", ModuleName: "dev", HandlerName: "sdeTypesByGroupHandler", Description: "Get types by group ID from SDE"},
		{Method: "GET", Path: "/sde/typematerials/{typeID}", ModuleName: "dev", HandlerName: "sdeTypeMaterialsHandler", Description: "Get type materials from SDE"},
		// Redis SDE endpoints
		{Method: "GET", Path: "/sde/redis/{type}/{id}", ModuleName: "dev", HandlerName: "sdeRedisEntityHandler", Description: "Get specific SDE entity from Redis"},
		{Method: "GET", Path: "/sde/redis/{type}", ModuleName: "dev", HandlerName: "sdeRedisEntitiesByTypeHandler", Description: "Get all entities of type from Redis"},
		// Universe SDE endpoints
		{Method: "GET", Path: "/sde/universe/{universeType}/{regionName}/systems", ModuleName: "dev", HandlerName: "sdeUniverseRegionSystemsHandler", Description: "Get all solar systems in region"},
		{Method: "GET", Path: "/sde/universe/{universeType}/{regionName}/{constellationName}/systems", ModuleName: "dev", HandlerName: "sdeUniverseConstellationSystemsHandler", Description: "Get all solar systems in constellation"},
		{Method: "GET", Path: "/sde/universe/{universeType}/{regionName}", ModuleName: "dev", HandlerName: "sdeUniverseDataHandler", Description: "Get region data from universe SDE"},
		{Method: "GET", Path: "/sde/universe/{universeType}/{regionName}/{constellationName}", ModuleName: "dev", HandlerName: "sdeUniverseDataHandler", Description: "Get constellation data from universe SDE"},
		{Method: "GET", Path: "/sde/universe/{universeType}/{regionName}/{constellationName}/{systemName}", ModuleName: "dev", HandlerName: "sdeUniverseDataHandler", Description: "Get system data from universe SDE"},
		// Service endpoints
		{Method: "GET", Path: "/services", ModuleName: "dev", HandlerName: "servicesHandler", Description: "List available development services"},
		{Method: "GET", Path: "/status", ModuleName: "dev", HandlerName: "statusHandler", Description: "Get module status"},
	}
}

// getUsersRoutes returns static route definitions for the users module
func getUsersRoutes() []RouteInfo {
	return []RouteInfo{
		{Method: "GET", Path: "/health", ModuleName: "users", HandlerName: "HealthHandler", Description: "Users module health check"},
		// Public endpoints
		{Method: "GET", Path: "/stats", ModuleName: "users", HandlerName: "getUserStatsHandler", Description: "Get user statistics (public endpoint)"},
		// Administrative endpoints (require authentication and admin permissions)
		{Method: "GET", Path: "/", ModuleName: "users", HandlerName: "listUsersHandler", Description: "List users with pagination and filtering (requires users:read permission)"},
		{Method: "GET", Path: "/{character_id}", ModuleName: "users", HandlerName: "getUserHandler", Description: "Get user by character ID (requires users:read permission)"},
		{Method: "PUT", Path: "/{character_id}", ModuleName: "users", HandlerName: "updateUserHandler", Description: "Update user status and settings (requires users:write permission)"},
		// User-specific character management (requires authentication)
		{Method: "GET", Path: "/by-user-id/{user_id}/characters", ModuleName: "users", HandlerName: "listCharactersHandler", Description: "List characters for a user (requires authentication, self-access or users:read permission)"},
	}
}

// getNotificationsRoutes returns static route definitions for the notifications module
func getNotificationsRoutes() []RouteInfo {
	return []RouteInfo{
		{Method: "GET", Path: "/health", ModuleName: "notifications", HandlerName: "HealthHandler", Description: "Notifications module health check"},
		{Method: "GET", Path: "/", ModuleName: "notifications", HandlerName: "getNotificationsHandler", Description: "List notifications"},
		{Method: "POST", Path: "/", ModuleName: "notifications", HandlerName: "sendNotificationHandler", Description: "Send a new notification"},
		{Method: "PUT", Path: "/{id}", ModuleName: "notifications", HandlerName: "markReadHandler", Description: "Mark notification as read"},
	}
}

// getSchedulerRoutes returns static route definitions for the scheduler module  
func getSchedulerRoutes() []RouteInfo {
	return []RouteInfo{
		{Method: "GET", Path: "/health", ModuleName: "scheduler", HandlerName: "HealthHandler", Description: "Scheduler module health check"},
		// Task management
		{Method: "GET", Path: "/tasks", ModuleName: "scheduler", HandlerName: "listTasksHandler", Description: "List all scheduled tasks"},
		{Method: "POST", Path: "/tasks", ModuleName: "scheduler", HandlerName: "createTaskHandler", Description: "Create a new scheduled task"},
		{Method: "GET", Path: "/tasks/{taskID}", ModuleName: "scheduler", HandlerName: "getTaskHandler", Description: "Get task details by ID"},
		{Method: "PUT", Path: "/tasks/{taskID}", ModuleName: "scheduler", HandlerName: "updateTaskHandler", Description: "Update task configuration"},
		{Method: "DELETE", Path: "/tasks/{taskID}", ModuleName: "scheduler", HandlerName: "deleteTaskHandler", Description: "Delete scheduled task"},
		// Task control
		{Method: "POST", Path: "/tasks/{taskID}/start", ModuleName: "scheduler", HandlerName: "startTaskHandler", Description: "Start/enable scheduled task"},
		{Method: "POST", Path: "/tasks/{taskID}/stop", ModuleName: "scheduler", HandlerName: "stopTaskHandler", Description: "Stop/disable scheduled task"},
		{Method: "POST", Path: "/tasks/{taskID}/pause", ModuleName: "scheduler", HandlerName: "pauseTaskHandler", Description: "Pause scheduled task"},
		{Method: "POST", Path: "/tasks/{taskID}/resume", ModuleName: "scheduler", HandlerName: "resumeTaskHandler", Description: "Resume paused task"},
		// Execution history
		{Method: "GET", Path: "/tasks/{taskID}/history", ModuleName: "scheduler", HandlerName: "getTaskHistoryHandler", Description: "Get task execution history"},
		{Method: "GET", Path: "/tasks/{taskID}/executions/{executionID}", ModuleName: "scheduler", HandlerName: "getExecutionHandler", Description: "Get specific execution details"},
		// System endpoints
		{Method: "GET", Path: "/stats", ModuleName: "scheduler", HandlerName: "getStatsHandler", Description: "Get scheduler statistics and metrics"},
		{Method: "POST", Path: "/reload", ModuleName: "scheduler", HandlerName: "reloadTasksHandler", Description: "Reload scheduler configuration"},
		{Method: "GET", Path: "/status", ModuleName: "scheduler", HandlerName: "getStatusHandler", Description: "Get scheduler service status"},
	}
}

// getSdeRoutes returns static route definitions for the sde module
func getSdeRoutes() []RouteInfo {
	return []RouteInfo{
		{Method: "GET", Path: "/health", ModuleName: "sde", HandlerName: "HealthHandler", Description: "SDE module health check"},
		// SDE Management endpoints
		{Method: "GET", Path: "/status", ModuleName: "sde", HandlerName: "handleGetStatus", Description: "Get current SDE version and status"},
		{Method: "POST", Path: "/check", ModuleName: "sde", HandlerName: "handleCheckForUpdates", Description: "Check for new SDE versions"},
		{Method: "POST", Path: "/update", ModuleName: "sde", HandlerName: "handleStartUpdate", Description: "Initiate SDE update process"},
		{Method: "GET", Path: "/progress", ModuleName: "sde", HandlerName: "handleGetProgress", Description: "Get real-time SDE update progress"},
		// Individual SDE entity access endpoints
		{Method: "GET", Path: "/entity/{type}/{id}", ModuleName: "sde", HandlerName: "handleGetEntity", Description: "Get individual SDE entity by type and ID"},
		{Method: "GET", Path: "/entities/{type}", ModuleName: "sde", HandlerName: "handleGetEntitiesByType", Description: "Get all entities of a specific type"},
		// Search endpoints
		{Method: "GET", Path: "/search/solarsystem", ModuleName: "sde", HandlerName: "handleSearchSolarSystem", Description: "Search for solar systems by name (query param: name)"},
		// Test endpoints for individual key storage
		{Method: "POST", Path: "/test/store-sample", ModuleName: "sde", HandlerName: "handleTestStoreSample", Description: "Store sample test data for development"},
		{Method: "GET", Path: "/test/verify", ModuleName: "sde", HandlerName: "handleTestVerify", Description: "Verify individual key storage functionality"},
	}
}


// generatePostmanCollection creates the Postman collection from discovered routes
func generatePostmanCollection(routes []RouteInfo) *PostmanCollection {
	collection := &PostmanCollection{
		Info: PostmanInfo{
			PostmanID:   "go-falcon-collection",
			Name:        "Go-Falcon Gateway - All Endpoints",
			Description: fmt.Sprintf("Complete collection of all endpoints in the Go-Falcon API Gateway. Generated automatically from route discovery.\n\nVersion: %s\nGenerated: %s", version.GetVersionString(), time.Now().Format(time.RFC3339)),
			Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
			ExporterID:  "go-falcon-exporter",
		},
		Variable: []PostmanVariable{
			{
				Key:         "gateway_url",
				Value:       "http://localhost:8080",
				Description: "Base URL for the Go-Falcon API Gateway",
			},
			{
				Key:         "api_prefix",
				Value:       "/api/v1",
				Description: "API prefix used by the gateway",
			},
			{
				Key:         "character_id",
				Value:       "123456789",
				Description: "Example character ID for testing",
			},
			{
				Key:         "alliance_id",
				Value:       "1354830081",
				Description: "Example alliance ID (Goonswarm Federation)",
			},
			{
				Key:         "system_id",
				Value:       "30000142",
				Description: "Example system ID (Jita)",
			},
			{
				Key:         "station_id",
				Value:       "60003760",
				Description: "Example station ID (Jita IV - Moon 4)",
			},
			{
				Key:         "user_id",
				Value:       "1",
				Description: "Example user ID for testing",
			},
			{
				Key:         "notification_id",
				Value:       "1",
				Description: "Example notification ID for testing",
			},
			{
				Key:         "access_token",
				Value:       "",
				Description: "JWT access token for authenticated endpoints",
			},
			{
				Key:         "agent_id",
				Value:       "3008416",
				Description: "Example agent ID for SDE testing",
			},
			{
				Key:         "category_id",
				Value:       "6",
				Description: "Example category ID (Ship) for SDE testing",
			},
			{
				Key:         "blueprint_id",
				Value:       "1000001",
				Description: "Example blueprint ID for SDE testing",
			},
			{
				Key:         "location_id",
				Value:       "60003760",
				Description: "Example location ID for SDE testing",
			},
			{
				Key:         "market_group_id",
				Value:       "4",
				Description: "Example market group ID for SDE testing",
			},
			{
				Key:         "meta_group_id",
				Value:       "1",
				Description: "Example meta group ID for SDE testing",
			},
			{
				Key:         "npc_corp_id",
				Value:       "1000001",
				Description: "Example NPC corporation ID for SDE testing",
			},
			{
				Key:         "faction_id",
				Value:       "500001",
				Description: "Example faction ID (Caldari State) for SDE testing",
			},
			{
				Key:         "type_id",
				Value:       "34",
				Description: "Example type ID (Tritanium) for SDE testing",
			},
			{
				Key:         "group_id",
				Value:       "18",
				Description: "Example group ID (Mineral) for SDE testing",
			},
			{
				Key:         "corporation_id",
				Value:       "98000001",
				Description: "Example corporation ID for testing",
			},
			{
				Key:         "task_id",
				Value:       "1",
				Description: "Example task ID for scheduler testing",
			},
			{
				Key:         "execution_id",
				Value:       "1",
				Description: "Example execution ID for scheduler testing",
			},
			{
				Key:         "group_id_groups",
				Value:       "507f1f77bcf86cd799439011",
				Description: "Example group ID (MongoDB ObjectID) for groups testing",
			},
			{
				Key:         "sde_type",
				Value:       "types",
				Description: "Example SDE data type (types, agents, categories, blueprints, etc.)",
			},
			{
				Key:         "sde_entity_id",
				Value:       "3008416",
				Description: "Example SDE entity ID for individual entity access",
			},
			{
				Key:         "universe_type",
				Value:       "eve",
				Description: "Example universe type (eve, abyssal, hidden, void, wormhole)",
			},
			{
				Key:         "region_name",
				Value:       "Derelik",
				Description: "Example region name for universe SDE data",
			},
			{
				Key:         "constellation_name",
				Value:       "Kador",
				Description: "Example constellation name for universe SDE data",
			},
			{
				Key:         "system_name",
				Value:       "Amarr",
				Description: "Example system name for universe SDE data",
			},
			{
				Key:         "search_name",
				Value:       "Jita",
				Description: "Example search query for solar system search",
			},
			{
				Key:         "service_name",
				Value:       "sde",
				Description: "Example service name for granular permission system",
			},
			{
				Key:         "resource_name",
				Value:       "entities",
				Description: "Example resource name for granular permission system",
			},
			{
				Key:         "action_name",
				Value:       "read",
				Description: "Example action name for granular permission system (read, write, delete, admin)",
			},
			{
				Key:         "subject_type",
				Value:       "group",
				Description: "Example subject type for granular permission system (group, member, corporation, alliance)",
			},
			{
				Key:         "subject_id",
				Value:       "507f1f77bcf86cd799439011",
				Description: "Example subject ID for granular permission system",
			},
			{
				Key:         "assignment_id",
				Value:       "507f1f77bcf86cd799439012",
				Description: "Example assignment ID for granular permission system",
			},
		},
		Event: []PostmanEvent{
			{
				Listen: "prerequest",
				Script: PostmanScript{
					Type: "text/javascript",
					Exec: []string{
						"// Set common headers",
						"pm.request.headers.add({",
						"    key: 'Accept',",
						"    value: 'application/json'",
						"});",
						"",
						"// Add timestamp for request tracking",
						"pm.globals.set('request_timestamp', new Date().toISOString());",
					},
				},
			},
			{
				Listen: "test",
				Script: PostmanScript{
					Type: "text/javascript",
					Exec: []string{
						"// Test for successful response or expected error",
						"pm.test('Response status is valid', function () {",
						"    pm.expect(pm.response.code).to.be.oneOf([200, 201, 204, 400, 401, 403, 404, 500]);",
						"});",
						"",
						"// Test for JSON response when content type is JSON",
						"if (pm.response.headers.get('Content-Type') && pm.response.headers.get('Content-Type').includes('application/json')) {",
						"    pm.test('Response is valid JSON', function () {",
						"        pm.response.to.be.json;",
						"    });",
						"}",
					},
				},
			},
		},
	}
	
	// Group routes by module
	moduleGroups := make(map[string][]RouteInfo)
	for _, route := range routes {
		moduleGroups[route.ModuleName] = append(moduleGroups[route.ModuleName], route)
	}
	
	// Create items for each module
	for moduleName, moduleRoutes := range moduleGroups {
		moduleItem := PostmanItem{
			Name:        strings.ToUpper(string(moduleName[0])) + moduleName[1:] + " Module",
			Description: fmt.Sprintf("Endpoints for the %s module", moduleName),
			Item:        []PostmanItem{},
		}
		
		// Add routes for this module
		for _, route := range moduleRoutes {
			request := createPostmanRequest(route)
			
			routeItem := PostmanItem{
				Name:        createRequestName(route),
				Description: route.Description,
				Request:     &request,
				Response:    []PostmanResponse{},
			}
			
			moduleItem.Item = append(moduleItem.Item, routeItem)
		}
		
		collection.Item = append(collection.Item, moduleItem)
	}
	
	return collection
}

// createPostmanRequest creates a Postman request from route info
func createPostmanRequest(route RouteInfo) PostmanRequest {
	apiPrefix := config.GetAPIPrefix()
	
	var fullPath string
	switch route.ModuleName {
	case "gateway":
		// Gateway routes don't have module prefix
		fullPath = route.Path
	case "groups":
		// Groups routes are mounted at API root level (no /groups prefix)
		if apiPrefix != "" {
			fullPath = apiPrefix + route.Path
		} else {
			fullPath = route.Path
		}
	default:
		// Module routes: apiPrefix + /module + route.Path
		if apiPrefix != "" {
			fullPath = apiPrefix + "/" + route.ModuleName + route.Path
		} else {
			fullPath = "/" + route.ModuleName + route.Path
		}
	}
	
	// Replace path parameters with Postman variables
	processedPath := processPathParameters(fullPath)
	
	request := PostmanRequest{
		Method: route.Method,
		Header: []PostmanHeader{
			{
				Key:   "Accept",
				Value: "application/json",
				Type:  "text",
			},
		},
		URL: PostmanURL{
			Raw:  "{{gateway_url}}" + processedPath,
			Host: []string{"{{gateway_url}}"},
			Path: strings.Split(strings.Trim(processedPath, "/"), "/"),
		},
		Description: route.Description,
	}
	
	// Add authentication for protected endpoints
	if needsAuth(route.Path) {
		request.Auth = &PostmanAuth{
			Type: "bearer",
			Bearer: []PostmanAuthBearer{
				{
					Key:   "token",
					Value: "{{access_token}}",
					Type:  "string",
				},
			},
		}
	}
	
	// Add query parameters for specific endpoints
	if route.Path == "/search/solarsystem" {
		request.URL.Query = []PostmanQuery{
			{
				Key:         "name",
				Value:       "{{search_name}}",
				Description: "Solar system name to search for (supports partial matching)",
			},
		}
	}

	// Add request body for POST/PUT/PATCH requests
	if route.Method == "POST" || route.Method == "PUT" || route.Method == "PATCH" {
		bodyContent := generateRequestBody(route)
		request.Body = &PostmanBody{
			Mode: "raw",
			Raw:  bodyContent,
		}
		request.Header = append(request.Header, PostmanHeader{
			Key:   "Content-Type",
			Value: "application/json",
			Type:  "text",
		})
	}
	
	return request
}

// processPathParameters converts path parameters to Postman variables
func processPathParameters(path string) string {
	// Convert {paramName} to {{paramName}}
	processed := path
	
	// Common parameter mappings
	paramMappings := map[string]string{
		"{characterID}":    "{{character_id}}",
		"{character_id}":   "{{character_id}}",
		"{allianceID}":     "{{alliance_id}}",
		"{systemID}":       "{{system_id}}",
		"{stationID}":      "{{station_id}}",
		"{userID}":         "{{user_id}}",
		"{user_id}":        "{{user_id}}",
		"{agentID}":        "{{agent_id}}",
		"{categoryID}":     "{{category_id}}",
		"{blueprintID}":    "{{blueprint_id}}",
		"{locationID}":     "{{location_id}}",
		"{marketGroupID}":  "{{market_group_id}}",
		"{metaGroupID}":    "{{meta_group_id}}",
		"{corpID}":         "{{npc_corp_id}}",
		"{factionID}":      "{{faction_id}}",
		"{typeID}":         "{{type_id}}",
		"{groupID}":        "{{group_id}}",
		"{corporationID}":  "{{corporation_id}}",
		"{taskID}":         "{{task_id}}",
		"{executionID}":    "{{execution_id}}",
		"{groupsID}":       "{{group_id_groups}}",
		"{type}":           "{{sde_type}}",
		"{id}":             "{{sde_entity_id}}",
		"{universeType}":   "{{universe_type}}",
		"{regionName}":     "{{region_name}}",
		"{constellationName}": "{{constellation_name}}",
		"{systemName}":     "{{system_name}}",
		"{serviceName}":    "{{service_name}}",
		"{assignmentID}":   "{{assignment_id}}",
	}
	
	for old, new := range paramMappings {
		processed = strings.ReplaceAll(processed, old, new)
	}
	
	return processed
}

// needsAuth determines if an endpoint needs authentication
func needsAuth(path string) bool {
	// Public endpoints that never require authentication
	publicPaths := []string{
		"/health",
		"/stats", // users stats endpoint is public
		"/esi-status",
		"/alliances",
		"/alliance/", // public alliance info endpoints
		"/corporation/", // public corporation info endpoints  
		"/universe/",
		"/character/", // public character info endpoints
		"/sde/", // SDE endpoints are generally public for read access
		"/search/", // Search endpoints are generally public
		"/services",
		"/status",
		"/eve/login",
		"/eve/callback",
	}
	
	// Check if it's a public endpoint
	for _, publicPath := range publicPaths {
		if strings.Contains(path, publicPath) {
			// Special case: some endpoints under these paths still need auth
			if strings.Contains(path, "/contacts") || 
			   strings.Contains(path, "/members") ||
			   strings.Contains(path, "/membertracking") ||
			   strings.Contains(path, "/roles") ||
			   strings.Contains(path, "/structures") ||
			   strings.Contains(path, "/standings") ||
			   strings.Contains(path, "/wallets") {
				return true
			}
			return false
		}
	}
	
	// All /admin paths require super admin authentication
	if strings.Contains(path, "/admin") {
		return true
	}
	
	// All other paths require authentication
	return true
}

// createRequestName generates a human-readable name for the request
func createRequestName(route RouteInfo) string {
	name := route.Method + " " + route.Path
	
	// Clean up common patterns
	name = strings.ReplaceAll(name, "{characterID}", "Character")
	name = strings.ReplaceAll(name, "{character_id}", "Character")
	name = strings.ReplaceAll(name, "{allianceID}", "Alliance")
	name = strings.ReplaceAll(name, "{systemID}", "System")
	name = strings.ReplaceAll(name, "{stationID}", "Station")
	name = strings.ReplaceAll(name, "{userID}", "User")
	name = strings.ReplaceAll(name, "{user_id}", "User")
	name = strings.ReplaceAll(name, "{id}", "ID")
	name = strings.ReplaceAll(name, "{agentID}", "Agent")
	name = strings.ReplaceAll(name, "{categoryID}", "Category")
	name = strings.ReplaceAll(name, "{blueprintID}", "Blueprint")
	name = strings.ReplaceAll(name, "{locationID}", "Location")
	name = strings.ReplaceAll(name, "{marketGroupID}", "MarketGroup")
	name = strings.ReplaceAll(name, "{metaGroupID}", "MetaGroup")
	name = strings.ReplaceAll(name, "{corpID}", "NPCCorp")
	name = strings.ReplaceAll(name, "{factionID}", "Faction")
	name = strings.ReplaceAll(name, "{typeID}", "Type")
	name = strings.ReplaceAll(name, "{groupID}", "Group")
	name = strings.ReplaceAll(name, "{corporationID}", "Corporation")
	name = strings.ReplaceAll(name, "{taskID}", "Task")
	name = strings.ReplaceAll(name, "{executionID}", "Execution")
	name = strings.ReplaceAll(name, "{groupsID}", "Group")
	name = strings.ReplaceAll(name, "{type}", "Type")
	
	return name
}

// generateRequestBody creates a JSON request body template based on DTO introspection
func generateRequestBody(route RouteInfo) string {
	// Try to get schema from introspection registry
	registry := introspection.NewRouteRegistry()
	if routeSchema, found := registry.GetRouteSchema(route.Method, route.Path); found && routeSchema.Request != nil {
		return generateJSONExample(routeSchema.Request)
	}
	
	// Fallback to generic template
	return "{\n  // Add request body here\n}"
}

// generateJSONExample creates a JSON example from an introspection schema
func generateJSONExample(schema *introspection.OpenAPISchema) string {
	if schema == nil {
		return "{\n  // Add request body here\n}"
	}
	
	if schema.Type == "object" && schema.Properties != nil {
		result := "{\n"
		for name, prop := range schema.Properties {
			value := getExampleValue(prop)
			result += fmt.Sprintf("  \"%s\": %s,\n", name, value)
		}
		result = strings.TrimSuffix(result, ",\n") + "\n}"
		return result
	}
	
	return "{\n  // Add request body here\n}"
}

// getExampleValue generates an example value for a schema property
func getExampleValue(schema *introspection.OpenAPISchema) string {
	if schema.Example != nil {
		if str, ok := schema.Example.(string); ok {
			return fmt.Sprintf("\"%s\"", str)
		}
		return fmt.Sprintf("%v", schema.Example)
	}
	
	switch schema.Type {
	case "string":
		if schema.Format == "email" {
			return "\"user@example.com\""
		}
		if schema.Format == "date-time" {
			return "\"2024-01-01T00:00:00Z\""
		}
		return "\"string\""
	case "integer":
		return "0"
	case "number":
		return "0.0"
	case "boolean":
		return "false"
	case "array":
		if schema.Items != nil {
			itemExample := getExampleValue(schema.Items)
			return fmt.Sprintf("[%s]", itemExample)
		}
		return "[]"
	case "object":
		if schema.Properties != nil {
			result := "{"
			for name, prop := range schema.Properties {
				value := getExampleValue(prop)
				result += fmt.Sprintf("\"%s\": %s, ", name, value)
			}
			result = strings.TrimSuffix(result, ", ") + "}"
			return result
		}
		return "{}"
	default:
		return "null"
	}
}

// exportCollection writes the collection to a JSON file
func exportCollection(collection *PostmanCollection, filename string) error {
	data, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal collection: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// Helper functions for counting and statistics
func countUniqueModules(routes []RouteInfo) int {
	modules := make(map[string]bool)
	for _, route := range routes {
		modules[route.ModuleName] = true
	}
	return len(modules)
}

func countTotalRequests(collection *PostmanCollection) int {
	total := 0
	for _, item := range collection.Item {
		total += len(item.Item)
	}
	return total
}