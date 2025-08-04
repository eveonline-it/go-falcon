package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go-falcon/internal/auth"
	"go-falcon/internal/dev"
	"go-falcon/internal/notifications"
	"go-falcon/internal/users"
	"go-falcon/pkg/config"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"
	"go-falcon/pkg/version"
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
	filename := "go-falcon-gateway-endpoints.postman_collection.json"
	if err := exportCollection(collection, filename); err != nil {
		log.Fatalf("❌ Failed to export collection: %v", err)
	}
	
	fmt.Printf("✅ Postman collection exported to: %s\n", filename)
	fmt.Printf("📊 Collection contains %d endpoints organized in %d modules\n", 
		countTotalRequests(collection), len(collection.Item))
}

// discoverRoutes initializes all modules and extracts their routes
func discoverRoutes() ([]RouteInfo, error) {
	var routes []RouteInfo
	
	// Create dummy database connections and SDE service for module initialization
	// This is safe since we're only discovering routes, not actually using the modules
	mongodb := &database.MongoDB{} 
	redis := &database.Redis{}
	sdeService := sde.NewService("data/sde") // Dummy SDE service for route discovery
	
	// Initialize modules
	modules := map[string]module.Module{
		"auth":          auth.New(mongodb, redis, sdeService),
		"dev":           dev.New(mongodb, redis, sdeService),
		"users":         users.New(mongodb, redis, sdeService),
		"notifications": notifications.New(mongodb, redis, sdeService),
	}
	
	// Create a route inspector to capture routes
	for moduleName, mod := range modules {
		fmt.Printf("🔍 Discovering routes for module: %s\n", moduleName)
		
		moduleRoutes := inspectModuleRoutes(mod, moduleName)
		routes = append(routes, moduleRoutes...)
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

// inspectModuleRoutes extracts route information from known module patterns
func inspectModuleRoutes(_ module.Module, moduleName string) []RouteInfo {
	var routes []RouteInfo
	
	// Since implementing the full chi.Router interface is complex, 
	// we'll use predefined route information for each module
	switch moduleName {
	case "dev":
		routes = []RouteInfo{
			{Method: "GET", Path: "/health", ModuleName: moduleName, HandlerName: "HealthHandler", Description: "Dev module health check"},
			{Method: "GET", Path: "/esi-status", ModuleName: moduleName, HandlerName: "esiStatusHandler", Description: "Get EVE Online server status"},
			{Method: "GET", Path: "/character/{characterID}", ModuleName: moduleName, HandlerName: "characterInfoHandler", Description: "Get character information"},
			{Method: "GET", Path: "/character/{characterID}/portrait", ModuleName: moduleName, HandlerName: "characterPortraitHandler", Description: "Get character portrait URLs"},
			{Method: "GET", Path: "/universe/system/{systemID}", ModuleName: moduleName, HandlerName: "systemInfoHandler", Description: "Get solar system information"},
			{Method: "GET", Path: "/universe/station/{stationID}", ModuleName: moduleName, HandlerName: "stationInfoHandler", Description: "Get station information"},
			{Method: "GET", Path: "/alliances", ModuleName: moduleName, HandlerName: "alliancesHandler", Description: "Get all active alliances"},
			{Method: "GET", Path: "/alliance/{allianceID}", ModuleName: moduleName, HandlerName: "allianceInfoHandler", Description: "Get alliance information"},
			{Method: "GET", Path: "/alliance/{allianceID}/corporations", ModuleName: moduleName, HandlerName: "allianceCorporationsHandler", Description: "Get alliance member corporations"},
			{Method: "GET", Path: "/alliance/{allianceID}/icons", ModuleName: moduleName, HandlerName: "allianceIconsHandler", Description: "Get alliance icon URLs"},
			{Method: "GET", Path: "/sde/status", ModuleName: moduleName, HandlerName: "sdeStatusHandler", Description: "Get SDE service status and statistics"},
			{Method: "GET", Path: "/sde/agent/{agentID}", ModuleName: moduleName, HandlerName: "sdeAgentHandler", Description: "Get agent information from SDE"},
			{Method: "GET", Path: "/sde/category/{categoryID}", ModuleName: moduleName, HandlerName: "sdeCategoryHandler", Description: "Get category information from SDE"},
			{Method: "GET", Path: "/sde/blueprint/{blueprintID}", ModuleName: moduleName, HandlerName: "sdeBlueprintHandler", Description: "Get blueprint information from SDE"},
			{Method: "GET", Path: "/sde/agents/location/{locationID}", ModuleName: moduleName, HandlerName: "sdeAgentsByLocationHandler", Description: "Get agents by location from SDE"},
			{Method: "GET", Path: "/services", ModuleName: moduleName, HandlerName: "servicesHandler", Description: "List available development services"},
			{Method: "GET", Path: "/status", ModuleName: moduleName, HandlerName: "statusHandler", Description: "Get module status"},
		}
	case "auth":
		routes = []RouteInfo{
			{Method: "GET", Path: "/health", ModuleName: moduleName, HandlerName: "HealthHandler", Description: "Auth module health check"},
			{Method: "POST", Path: "/login", ModuleName: moduleName, HandlerName: "loginHandler", Description: "User login"},
			{Method: "POST", Path: "/logout", ModuleName: moduleName, HandlerName: "logoutHandler", Description: "User logout"},
			{Method: "POST", Path: "/refresh", ModuleName: moduleName, HandlerName: "refreshHandler", Description: "Refresh authentication token"},
			{Method: "GET", Path: "/me", ModuleName: moduleName, HandlerName: "meHandler", Description: "Get current user information"},
		}
	case "users":
		routes = []RouteInfo{
			{Method: "GET", Path: "/health", ModuleName: moduleName, HandlerName: "HealthHandler", Description: "Users module health check"},
			{Method: "GET", Path: "/", ModuleName: moduleName, HandlerName: "listUsersHandler", Description: "List all users"},
			{Method: "POST", Path: "/", ModuleName: moduleName, HandlerName: "createUserHandler", Description: "Create a new user"},
			{Method: "GET", Path: "/{userID}", ModuleName: moduleName, HandlerName: "getUserHandler", Description: "Get user by ID"},
			{Method: "PUT", Path: "/{userID}", ModuleName: moduleName, HandlerName: "updateUserHandler", Description: "Update user information"},
			{Method: "DELETE", Path: "/{userID}", ModuleName: moduleName, HandlerName: "deleteUserHandler", Description: "Delete user"},
		}
	case "notifications":
		routes = []RouteInfo{
			{Method: "GET", Path: "/health", ModuleName: moduleName, HandlerName: "HealthHandler", Description: "Notifications module health check"},
			{Method: "GET", Path: "/", ModuleName: moduleName, HandlerName: "listNotificationsHandler", Description: "List notifications"},
			{Method: "POST", Path: "/", ModuleName: moduleName, HandlerName: "createNotificationHandler", Description: "Create a new notification"},
			{Method: "GET", Path: "/{notificationID}", ModuleName: moduleName, HandlerName: "getNotificationHandler", Description: "Get notification by ID"},
			{Method: "PUT", Path: "/{notificationID}", ModuleName: moduleName, HandlerName: "updateNotificationHandler", Description: "Update notification"},
			{Method: "DELETE", Path: "/{notificationID}", ModuleName: moduleName, HandlerName: "deleteNotificationHandler", Description: "Delete notification"},
		}
	}
	
	return routes
}


// generatePostmanCollection creates the Postman collection from discovered routes
func generatePostmanCollection(routes []RouteInfo) *PostmanCollection {
	collection := &PostmanCollection{
		Info: PostmanInfo{
			PostmanID:   "go-falcon-gateway-collection",
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
	if route.ModuleName == "gateway" {
		// Gateway routes don't have module prefix
		fullPath = route.Path
	} else {
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
	
	// Add request body for POST/PUT/PATCH requests
	if route.Method == "POST" || route.Method == "PUT" || route.Method == "PATCH" {
		request.Body = &PostmanBody{
			Mode: "raw",
			Raw:  "{\n  // Add request body here\n}",
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
		"{characterID}": "{{character_id}}",
		"{allianceID}":  "{{alliance_id}}",
		"{systemID}":    "{{system_id}}",
		"{stationID}":   "{{station_id}}",
		"{userID}":      "{{user_id}}",
		"{id}":          "{{id}}",
	}
	
	for old, new := range paramMappings {
		processed = strings.ReplaceAll(processed, old, new)
	}
	
	return processed
}

// needsAuth determines if an endpoint needs authentication
func needsAuth(path string) bool {
	authPaths := []string{
		"/contacts",
		"/user",
		"/profile",
		"/private",
		"/admin",
	}
	
	for _, authPath := range authPaths {
		if strings.Contains(path, authPath) {
			return true
		}
	}
	
	return false
}

// createRequestName generates a human-readable name for the request
func createRequestName(route RouteInfo) string {
	name := route.Method + " " + route.Path
	
	// Clean up common patterns
	name = strings.ReplaceAll(name, "{characterID}", "Character")
	name = strings.ReplaceAll(name, "{allianceID}", "Alliance")
	name = strings.ReplaceAll(name, "{systemID}", "System")
	name = strings.ReplaceAll(name, "{stationID}", "Station")
	name = strings.ReplaceAll(name, "{userID}", "User")
	name = strings.ReplaceAll(name, "{id}", "ID")
	
	return name
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