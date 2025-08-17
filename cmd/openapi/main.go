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
)

// OpenAPISpec represents the OpenAPI 3.1.0 specification structure
type OpenAPISpec struct {
	OpenAPI      string                    `json:"openapi"`
	Info         OpenAPIInfo               `json:"info"`
	Servers      []OpenAPIServer           `json:"servers"`
	Paths        map[string]OpenAPIPath    `json:"paths"`
	Components   *OpenAPIComponents        `json:"components,omitempty"`
	Security     []OpenAPISecurityReq      `json:"security,omitempty"`
	Tags         []OpenAPITag              `json:"tags,omitempty"`
	ExternalDocs *OpenAPIExternalDocs      `json:"externalDocs,omitempty"`
}

// OpenAPIInfo contains API metadata
type OpenAPIInfo struct {
	Title          string             `json:"title"`
	Description    string             `json:"description"`
	TermsOfService string             `json:"termsOfService,omitempty"`
	Contact        *OpenAPIContact    `json:"contact,omitempty"`
	License        *OpenAPILicense    `json:"license,omitempty"`
	Version        string             `json:"version"`
}

// OpenAPIContact contains contact information
type OpenAPIContact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// OpenAPILicense contains license information
type OpenAPILicense struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier,omitempty"`
	URL        string `json:"url,omitempty"`
}

// OpenAPIServer represents a server configuration
type OpenAPIServer struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description,omitempty"`
	Variables   map[string]OpenAPIVariable `json:"variables,omitempty"`
}

// OpenAPIVariable represents a server variable
type OpenAPIVariable struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`
}

// OpenAPIPath represents all operations for a path
type OpenAPIPath map[string]OpenAPIOperation

// OpenAPIOperation represents an API operation
type OpenAPIOperation struct {
	Tags         []string                      `json:"tags,omitempty"`
	Summary      string                        `json:"summary,omitempty"`
	Description  string                        `json:"description,omitempty"`
	OperationID  string                        `json:"operationId,omitempty"`
	Parameters   []OpenAPIParameter            `json:"parameters,omitempty"`
	RequestBody  *OpenAPIRequestBody           `json:"requestBody,omitempty"`
	Responses    map[string]OpenAPIResponse    `json:"responses"`
	Deprecated   bool                          `json:"deprecated,omitempty"`
	Security     []OpenAPISecurityReq          `json:"security,omitempty"`
	Servers      []OpenAPIServer               `json:"servers,omitempty"`
	ExternalDocs *OpenAPIExternalDocs          `json:"externalDocs,omitempty"`
}

// OpenAPIParameter represents an operation parameter
type OpenAPIParameter struct {
	Name            string              `json:"name"`
	In              string              `json:"in"`
	Description     string              `json:"description,omitempty"`
	Required        bool                `json:"required,omitempty"`
	Deprecated      bool                `json:"deprecated,omitempty"`
	AllowEmptyValue bool                `json:"allowEmptyValue,omitempty"`
	Style           string              `json:"style,omitempty"`
	Explode         bool                `json:"explode,omitempty"`
	AllowReserved   bool                `json:"allowReserved,omitempty"`
	Schema          *OpenAPISchema      `json:"schema,omitempty"`
	Example         interface{}         `json:"example,omitempty"`
	Examples        map[string]OpenAPIExample `json:"examples,omitempty"`
}

// OpenAPIRequestBody represents a request body
type OpenAPIRequestBody struct {
	Description string                      `json:"description,omitempty"`
	Content     map[string]OpenAPIMediaType `json:"content"`
	Required    bool                        `json:"required,omitempty"`
}

// OpenAPIResponse represents an API response
type OpenAPIResponse struct {
	Description string                      `json:"description"`
	Headers     map[string]OpenAPIHeader    `json:"headers,omitempty"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
	Links       map[string]OpenAPILink      `json:"links,omitempty"`
}

// OpenAPIMediaType represents a media type
type OpenAPIMediaType struct {
	Schema   *OpenAPISchema             `json:"schema,omitempty"`
	Example  interface{}                `json:"example,omitempty"`
	Examples map[string]OpenAPIExample  `json:"examples,omitempty"`
	Encoding map[string]OpenAPIEncoding `json:"encoding,omitempty"`
}

// OpenAPISchema represents a JSON schema
type OpenAPISchema struct {
	Type                 string                    `json:"type,omitempty"`
	Format               string                    `json:"format,omitempty"`
	Description          string                    `json:"description,omitempty"`
	Enum                 []interface{}             `json:"enum,omitempty"`
	Default              interface{}               `json:"default,omitempty"`
	Example              interface{}               `json:"example,omitempty"`
	Examples             []interface{}             `json:"examples,omitempty"`
	Items                *OpenAPISchema            `json:"items,omitempty"`
	Properties           map[string]*OpenAPISchema `json:"properties,omitempty"`
	AdditionalProperties interface{}               `json:"additionalProperties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	AllOf                []*OpenAPISchema          `json:"allOf,omitempty"`
	AnyOf                []*OpenAPISchema          `json:"anyOf,omitempty"`
	OneOf                []*OpenAPISchema          `json:"oneOf,omitempty"`
	Not                  *OpenAPISchema            `json:"not,omitempty"`
	Minimum              *float64                  `json:"minimum,omitempty"`
	Maximum              *float64                  `json:"maximum,omitempty"`
	ExclusiveMinimum     *bool                     `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum     *bool                     `json:"exclusiveMaximum,omitempty"`
	MinLength            *int                      `json:"minLength,omitempty"`
	MaxLength            *int                      `json:"maxLength,omitempty"`
	Pattern              string                    `json:"pattern,omitempty"`
	MinItems             *int                      `json:"minItems,omitempty"`
	MaxItems             *int                      `json:"maxItems,omitempty"`
	UniqueItems          bool                      `json:"uniqueItems,omitempty"`
	MinProperties        *int                      `json:"minProperties,omitempty"`
	MaxProperties        *int                      `json:"maxProperties,omitempty"`
}

// OpenAPIHeader represents a header parameter
type OpenAPIHeader struct {
	Description     string              `json:"description,omitempty"`
	Required        bool                `json:"required,omitempty"`
	Deprecated      bool                `json:"deprecated,omitempty"`
	AllowEmptyValue bool                `json:"allowEmptyValue,omitempty"`
	Style           string              `json:"style,omitempty"`
	Explode         bool                `json:"explode,omitempty"`
	AllowReserved   bool                `json:"allowReserved,omitempty"`
	Schema          *OpenAPISchema      `json:"schema,omitempty"`
	Example         interface{}         `json:"example,omitempty"`
	Examples        map[string]OpenAPIExample `json:"examples,omitempty"`
}

// OpenAPIExample represents an example
type OpenAPIExample struct {
	Summary       string      `json:"summary,omitempty"`
	Description   string      `json:"description,omitempty"`
	Value         interface{} `json:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty"`
}

// OpenAPILink represents a link
type OpenAPILink struct {
	OperationRef string                 `json:"operationRef,omitempty"`
	OperationID  string                 `json:"operationId,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	RequestBody  interface{}            `json:"requestBody,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Server       *OpenAPIServer         `json:"server,omitempty"`
}

// OpenAPIEncoding represents encoding configuration
type OpenAPIEncoding struct {
	ContentType   string                   `json:"contentType,omitempty"`
	Headers       map[string]OpenAPIHeader `json:"headers,omitempty"`
	Style         string                   `json:"style,omitempty"`
	Explode       bool                     `json:"explode,omitempty"`
	AllowReserved bool                     `json:"allowReserved,omitempty"`
}

// OpenAPIComponents represents reusable components
type OpenAPIComponents struct {
	Schemas         map[string]*OpenAPISchema         `json:"schemas,omitempty"`
	Responses       map[string]OpenAPIResponse        `json:"responses,omitempty"`
	Parameters      map[string]OpenAPIParameter       `json:"parameters,omitempty"`
	Examples        map[string]OpenAPIExample         `json:"examples,omitempty"`
	RequestBodies   map[string]OpenAPIRequestBody     `json:"requestBodies,omitempty"`
	Headers         map[string]OpenAPIHeader          `json:"headers,omitempty"`
	SecuritySchemes map[string]OpenAPISecurityScheme  `json:"securitySchemes,omitempty"`
	Links           map[string]OpenAPILink            `json:"links,omitempty"`
	Callbacks       map[string]map[string]OpenAPIPath `json:"callbacks,omitempty"`
}

// OpenAPISecurityScheme represents a security scheme
type OpenAPISecurityScheme struct {
	Type             string            `json:"type"`
	Description      string            `json:"description,omitempty"`
	Name             string            `json:"name,omitempty"`
	In               string            `json:"in,omitempty"`
	Scheme           string            `json:"scheme,omitempty"`
	BearerFormat     string            `json:"bearerFormat,omitempty"`
	Flows            *OpenAPIFlows     `json:"flows,omitempty"`
	OpenIDConnectURL string            `json:"openIdConnectUrl,omitempty"`
}

// OpenAPIFlows represents OAuth2 flows
type OpenAPIFlows struct {
	Implicit          *OpenAPIFlow `json:"implicit,omitempty"`
	Password          *OpenAPIFlow `json:"password,omitempty"`
	ClientCredentials *OpenAPIFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OpenAPIFlow `json:"authorizationCode,omitempty"`
}

// OpenAPIFlow represents an OAuth2 flow
type OpenAPIFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

// OpenAPISecurityReq represents a security requirement
type OpenAPISecurityReq map[string][]string

// OpenAPITag represents a tag
type OpenAPITag struct {
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	ExternalDocs *OpenAPIExternalDocs `json:"externalDocs,omitempty"`
}

// OpenAPIExternalDocs represents external documentation
type OpenAPIExternalDocs struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// RouteInfo holds information about discovered routes (reused from postman)
type RouteInfo struct {
	Method      string
	Path        string
	ModuleName  string
	HandlerName string
	Description string
}

func main() {
	fmt.Println("🚀 Go-Falcon OpenAPI 3.1 Exporter")
	
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
	
	// Generate OpenAPI specification
	spec := generateOpenAPISpec(routes)
	
	// Export to JSON file
	filename := "falcon-openapi.json"
	if err := exportSpec(spec, filename); err != nil {
		log.Fatalf("❌ Failed to export specification: %v", err)
	}
	
	fmt.Printf("✅ OpenAPI 3.1 specification exported to: %s\n", filename)
	fmt.Printf("📊 Specification contains %d paths across %d modules\n", 
		len(spec.Paths), len(spec.Tags))
}

// discoverRoutes reuses the same route discovery logic as the postman command
func discoverRoutes() ([]RouteInfo, error) {
	var routes []RouteInfo
	
	// Use static route definitions for all modules to avoid environment dependencies
	moduleRoutes := map[string][]RouteInfo{
		"auth":          getAuthRoutes(),
		"groups":        getGroupsRoutes(),
		"dev":           getDevRoutes(),
		"users":         getUsersRoutes(),
		"notifications": getNotificationsRoutes(),
		"scheduler":     getSchedulerRoutes(),
		"sde":           getSdeRoutes(),
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

// generateOpenAPISpec creates the OpenAPI 3.1 specification from discovered routes
func generateOpenAPISpec(routes []RouteInfo) *OpenAPISpec {
	spec := &OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: OpenAPIInfo{
			Title:       "Go-Falcon Gateway API",
			Description: fmt.Sprintf("Complete OpenAPI 3.1 specification for the Go-Falcon API Gateway.\n\nThis specification covers all endpoints across multiple modules including authentication, user management, EVE Online integration, static data access, and task scheduling.\n\nVersion: %s\nGenerated: %s", version.GetVersionString(), time.Now().Format(time.RFC3339)),
			Version:     version.GetVersionString(),
			Contact: &OpenAPIContact{
				Name: "Go-Falcon Team",
				URL:  "https://github.com/go-falcon/go-falcon",
			},
			License: &OpenAPILicense{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Servers: []OpenAPIServer{
			{
				URL:         "http://localhost:8080",
				Description: "Development server",
			},
			{
				URL:         "https://api.go-falcon.example.com",
				Description: "Production server",
			},
		},
		Paths:      make(map[string]OpenAPIPath),
		Components: generateComponents(),
		Security: []OpenAPISecurityReq{
			{"bearerAuth": []string{}},
		},
		Tags: generateTags(routes),
		ExternalDocs: &OpenAPIExternalDocs{
			Description: "Go-Falcon Documentation",
			URL:         "https://github.com/go-falcon/go-falcon/blob/main/README.md",
		},
	}
	
	// Group routes by path
	pathGroups := make(map[string][]RouteInfo)
	for _, route := range routes {
		fullPath := buildFullPath(route)
		pathGroups[fullPath] = append(pathGroups[fullPath], route)
	}
	
	// Create OpenAPI paths
	for path, pathRoutes := range pathGroups {
		operations := make(OpenAPIPath)
		
		for _, route := range pathRoutes {
			operation := createOperation(route)
			operations[strings.ToLower(route.Method)] = operation
		}
		
		spec.Paths[path] = operations
	}
	
	return spec
}

// buildFullPath constructs the full API path for a route
func buildFullPath(route RouteInfo) string {
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
	
	return fullPath
}

// createOperation creates an OpenAPI operation from route info
func createOperation(route RouteInfo) OpenAPIOperation {
	operation := OpenAPIOperation{
		Tags:        []string{strings.Title(route.ModuleName)},
		Summary:     generateSummary(route),
		Description: route.Description,
		OperationID: generateOperationID(route),
		Parameters:  extractParameters(route.Path),
		Responses:   generateResponses(route),
	}
	
	// Add request body for POST/PUT/PATCH requests
	if route.Method == "POST" || route.Method == "PUT" || route.Method == "PATCH" {
		operation.RequestBody = generateRequestBody(route)
	}
	
	// Add security for protected endpoints
	if needsAuth(route.Path) {
		operation.Security = []OpenAPISecurityReq{
			{"bearerAuth": []string{}},
		}
	}
	
	return operation
}

// generateSummary creates a human-readable summary for the operation
func generateSummary(route RouteInfo) string {
	verb := strings.ToUpper(route.Method)
	
	// Extract resource from path
	pathParts := strings.Split(strings.Trim(route.Path, "/"), "/")
	if len(pathParts) > 0 {
		resource := pathParts[0]
		if strings.Contains(route.Path, "{") {
			// Path has parameters, likely a specific resource operation
			if verb == "GET" {
				return fmt.Sprintf("Get %s", resource)
			} else if verb == "PUT" {
				return fmt.Sprintf("Update %s", resource)
			} else if verb == "DELETE" {
				return fmt.Sprintf("Delete %s", resource)
			}
		} else {
			// No parameters, likely a collection operation
			if verb == "GET" {
				return fmt.Sprintf("List %s", resource)
			} else if verb == "POST" {
				return fmt.Sprintf("Create %s", resource)
			}
		}
	}
	
	return fmt.Sprintf("%s %s", verb, route.Path)
}

// generateOperationID creates a unique operation ID
func generateOperationID(route RouteInfo) string {
	method := strings.ToLower(route.Method)
	path := strings.ReplaceAll(route.Path, "/", "_")
	path = strings.ReplaceAll(path, "{", "")
	path = strings.ReplaceAll(path, "}", "")
	path = strings.Trim(path, "_")
	
	if path == "" {
		return fmt.Sprintf("%s_%s_root", method, route.ModuleName)
	}
	
	return fmt.Sprintf("%s_%s_%s", method, route.ModuleName, path)
}

// extractParameters extracts path parameters from a route path
func extractParameters(path string) []OpenAPIParameter {
	var parameters []OpenAPIParameter
	
	// Extract path parameters
	pathParams := extractPathParams(path)
	for _, param := range pathParams {
		parameters = append(parameters, OpenAPIParameter{
			Name:        param,
			In:          "path",
			Description: fmt.Sprintf("The %s identifier", param),
			Required:    true,
			Schema: &OpenAPISchema{
				Type: "string",
			},
		})
	}
	
	// Add query parameters for search endpoints
	if strings.Contains(path, "/search/") {
		parameters = append(parameters, OpenAPIParameter{
			Name:        "name",
			In:          "query",
			Description: "Search query string",
			Required:    false,
			Schema: &OpenAPISchema{
				Type: "string",
			},
		})
	}
	
	return parameters
}

// extractPathParams extracts parameter names from path template
func extractPathParams(path string) []string {
	var params []string
	parts := strings.Split(path, "/")
	
	for _, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := strings.Trim(part, "{}")
			params = append(params, paramName)
		}
	}
	
	return params
}

// generateRequestBody creates request body specification for POST/PUT/PATCH
func generateRequestBody(route RouteInfo) *OpenAPIRequestBody {
	return &OpenAPIRequestBody{
		Description: "Request payload",
		Required:    true,
		Content: map[string]OpenAPIMediaType{
			"application/json": {
				Schema: &OpenAPISchema{
					Type: "object",
					Properties: map[string]*OpenAPISchema{
						"data": {
							Type:        "object",
							Description: "Request data",
						},
					},
				},
			},
		},
	}
}

// generateResponses creates response specifications
func generateResponses(route RouteInfo) map[string]OpenAPIResponse {
	responses := map[string]OpenAPIResponse{
		"200": {
			Description: "Successful response",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type: "object",
						Properties: map[string]*OpenAPISchema{
							"success": {
								Type:        "boolean",
								Description: "Operation success status",
							},
							"data": {
								Type:        "object",
								Description: "Response data",
							},
						},
					},
				},
			},
		},
		"400": {
			Description: "Bad request",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type: "object",
						Properties: map[string]*OpenAPISchema{
							"error": {
								Type:        "string",
								Description: "Error message",
							},
						},
					},
				},
			},
		},
		"500": {
			Description: "Internal server error",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type: "object",
						Properties: map[string]*OpenAPISchema{
							"error": {
								Type:        "string",
								Description: "Error message",
							},
						},
					},
				},
			},
		},
	}
	
	// Add specific responses for different methods
	if route.Method == "POST" {
		responses["201"] = OpenAPIResponse{
			Description: "Resource created successfully",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type: "object",
						Properties: map[string]*OpenAPISchema{
							"success": {
								Type: "boolean",
							},
							"data": {
								Type: "object",
							},
						},
					},
				},
			},
		}
	}
	
	if route.Method == "DELETE" {
		responses["204"] = OpenAPIResponse{
			Description: "Resource deleted successfully",
		}
	}
	
	// Add authentication responses for protected endpoints
	if needsAuth(route.Path) {
		responses["401"] = OpenAPIResponse{
			Description: "Authentication required",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type: "object",
						Properties: map[string]*OpenAPISchema{
							"error": {
								Type:        "string",
								Description: "Authentication error message",
							},
						},
					},
				},
			},
		}
		
		responses["403"] = OpenAPIResponse{
			Description: "Forbidden - insufficient permissions",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type: "object",
						Properties: map[string]*OpenAPISchema{
							"error": {
								Type:        "string",
								Description: "Authorization error message",
							},
						},
					},
				},
			},
		}
	}
	
	return responses
}

// generateComponents creates reusable components
func generateComponents() *OpenAPIComponents {
	return &OpenAPIComponents{
		SecuritySchemes: map[string]OpenAPISecurityScheme{
			"bearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "JWT token obtained from EVE Online SSO authentication",
			},
		},
		Schemas: map[string]*OpenAPISchema{
			"Error": {
				Type: "object",
				Properties: map[string]*OpenAPISchema{
					"error": {
						Type:        "string",
						Description: "Error message",
					},
					"code": {
						Type:        "string",
						Description: "Error code",
					},
				},
				Required: []string{"error"},
			},
			"HealthResponse": {
				Type: "object",
				Properties: map[string]*OpenAPISchema{
					"status": {
						Type:        "string",
						Description: "Service status",
						Example:     "healthy",
					},
					"version": {
						Type:        "string",
						Description: "Service version",
					},
					"timestamp": {
						Type:        "string",
						Format:      "date-time",
						Description: "Response timestamp",
					},
				},
				Required: []string{"status"},
			},
		},
	}
}

// generateTags creates tags for grouping operations
func generateTags(routes []RouteInfo) []OpenAPITag {
	moduleSet := make(map[string]bool)
	for _, route := range routes {
		moduleSet[route.ModuleName] = true
	}
	
	var tags []OpenAPITag
	
	tagDescriptions := map[string]string{
		"gateway":       "Gateway health and status endpoints",
		"auth":          "Authentication and authorization endpoints using EVE Online SSO",
		"groups":        "Group management and permission system endpoints",
		"dev":           "Development and testing endpoints for EVE Online ESI and SDE data",
		"users":         "User management and profile endpoints",
		"notifications": "Notification system endpoints",
		"scheduler":     "Task scheduling and management endpoints",
		"sde":           "EVE Online Static Data Export management endpoints",
	}
	
	for module := range moduleSet {
		description := tagDescriptions[module]
		if description == "" {
			description = fmt.Sprintf("%s module endpoints", strings.Title(module))
		}
		
		tags = append(tags, OpenAPITag{
			Name:        strings.Title(module),
			Description: description,
		})
	}
	
	return tags
}

// needsAuth determines if an endpoint needs authentication (reused from postman)
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

// exportSpec writes the OpenAPI specification to a JSON file
func exportSpec(spec *OpenAPISpec, filename string) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal specification: %w", err)
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

// Import all route definitions from postman command
// This ensures consistency between both exporters

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