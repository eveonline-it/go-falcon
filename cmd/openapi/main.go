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
	
	"github.com/joho/godotenv"
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
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}

	fmt.Println("🚀 Go-Falcon OpenAPI 3.1 Exporter")
	
	// Show current API prefix configuration
	apiPrefix := config.GetAPIPrefix()
	if apiPrefix == "" {
		fmt.Printf("🔗 API Prefix: (none - using root paths)\n")
	} else {
		fmt.Printf("🔗 API Prefix: %s\n", apiPrefix)
	}
	
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
				URL:         "https://go.eveonline.it",
				Description: "Development server",
			},
			{
				URL:         "https://go.eveonline.it",
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
		// Groups routes are mounted under /groups prefix via sub-router
		if apiPrefix != "" {
			fullPath = apiPrefix + "/groups" + route.Path
		} else {
			fullPath = "/groups" + route.Path
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

// convertIntrospectionSchema converts an introspection schema to OpenAPI schema
func convertIntrospectionSchema(introspectionSchema *introspection.OpenAPISchema) *OpenAPISchema {
	if introspectionSchema == nil {
		return nil
	}
	
	schema := &OpenAPISchema{
		Type:        introspectionSchema.Type,
		Format:      introspectionSchema.Format,
		Description: introspectionSchema.Description,
		Required:    introspectionSchema.Required,
		Example:     introspectionSchema.Example,
		MinLength:   introspectionSchema.MinLength,
		MaxLength:   introspectionSchema.MaxLength,
		Pattern:     introspectionSchema.Pattern,
		Minimum:     introspectionSchema.Minimum,
		Maximum:     introspectionSchema.Maximum,
		MinItems:    introspectionSchema.MinItems,
		MaxItems:    introspectionSchema.MaxItems,
		UniqueItems: introspectionSchema.UniqueItems,
	}
	
	// Convert properties
	if introspectionSchema.Properties != nil {
		schema.Properties = make(map[string]*OpenAPISchema)
		for name, prop := range introspectionSchema.Properties {
			schema.Properties[name] = convertIntrospectionSchema(prop)
		}
	}
	
	// Convert items (for arrays)
	if introspectionSchema.Items != nil {
		schema.Items = convertIntrospectionSchema(introspectionSchema.Items)
	}
	
	// Convert enum values
	if introspectionSchema.Enum != nil {
		schema.Enum = introspectionSchema.Enum
	}
	
	return schema
}

// generateSummary creates a human-readable summary for the operation
func generateSummary(route RouteInfo) string {
	// Use the route description as summary if available and meaningful
	if route.Description != "" && route.Description != "No description available" {
		return route.Description
	}
	
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
	
	// Add character_id query parameter for public profile endpoint
	if path == "/profile/public" {
		parameters = append(parameters, OpenAPIParameter{
			Name:        "character_id",
			In:          "query",
			Description: "EVE character ID",
			Required:    true,
			Schema: &OpenAPISchema{
				Type: "integer",
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
				Schema: generateRequestSchema(route),
			},
		},
	}
}

// generateRequestSchema creates appropriate request schema based on route characteristics  
func generateRequestSchema(route RouteInfo) *OpenAPISchema {
	// Try to get schema from introspection registry
	registry := introspection.NewRouteRegistry()
	if routeSchema, found := registry.GetRouteSchema(route.Method, route.Path); found && routeSchema.Request != nil {
		return convertIntrospectionSchema(routeSchema.Request)
	}
	// User update endpoints
	if route.ModuleName == "users" && route.Method == "PUT" {
		return &OpenAPISchema{
			Type: "object",
			Properties: map[string]*OpenAPISchema{
				"enabled": {
					Type:        "boolean",
					Description: "Enable/disable user",
				},
				"banned": {
					Type:        "boolean", 
					Description: "Ban/unban user",
				},
				"invalid": {
					Type:        "boolean",
					Description: "Set validity status",
				},
				"position": {
					Type:        "integer",
					Description: "Update position/rank",
				},
				"notes": {
					Type:        "string",
					Description: "Update administrative notes",
				},
			},
		}
	}
	
	// Groups permission assignment
	if route.ModuleName == "groups" && strings.Contains(route.Path, "/admin/permissions/assignments") && route.Method == "POST" {
		return &OpenAPISchema{
			Type: "object",
			Properties: map[string]*OpenAPISchema{
				"service": {
					Type:        "string",
					Description: "Service name",
				},
				"resource": {
					Type:        "string",
					Description: "Resource name",
				},
				"action": {
					Type:        "string", 
					Description: "Action name (read, write, delete, admin)",
				},
				"subject_type": {
					Type:        "string",
					Description: "Subject type (group, member, corporation, alliance)",
				},
				"subject_id": {
					Type:        "string",
					Description: "Subject identifier",
				},
				"expires_at": {
					Type:        "string",
					Format:      "date-time",
					Description: "Optional expiration timestamp",
				},
				"reason": {
					Type:        "string",
					Description: "Business justification for permission grant",
				},
			},
			Required: []string{"service", "resource", "action", "subject_type", "subject_id", "reason"},
		}
	}
	
	// Groups service creation
	if route.ModuleName == "groups" && strings.Contains(route.Path, "/admin/permissions/services") && route.Method == "POST" {
		return &OpenAPISchema{
			Type: "object",
			Properties: map[string]*OpenAPISchema{
				"name": {
					Type:        "string",
					Description: "Service name (unique identifier)",
				},
				"display_name": {
					Type:        "string",
					Description: "Human-readable service name",
				},
				"description": {
					Type:        "string",
					Description: "Service description",
				},
				"resources": {
					Type: "array",
					Items: &OpenAPISchema{
						Type: "object",
						Properties: map[string]*OpenAPISchema{
							"name": {
								Type:        "string",
								Description: "Resource name",
							},
							"display_name": {
								Type:        "string",
								Description: "Human-readable resource name",
							},
							"actions": {
								Type: "array",
								Items: &OpenAPISchema{
									Type: "string",
								},
								Description: "Available actions for this resource",
							},
						},
					},
					Description: "Service resources",
				},
			},
			Required: []string{"name", "display_name", "resources"},
		}
	}
	
	// Scheduler task creation
	if route.ModuleName == "scheduler" && strings.Contains(route.Path, "/tasks") && route.Method == "POST" {
		return &OpenAPISchema{
			Type: "object",
			Properties: map[string]*OpenAPISchema{
				"name": {
					Type:        "string",
					Description: "Task name",
				},
				"type": {
					Type:        "string",
					Description: "Task type (http, function, system)",
				},
				"schedule": {
					Type:        "string",
					Description: "Cron schedule expression",
				},
				"config": {
					Type:        "object",
					Description: "Task-specific configuration",
				},
				"enabled": {
					Type:        "boolean",
					Description: "Whether task is enabled",
				},
			},
			Required: []string{"name", "type", "schedule"},
		}
	}
	
	// Generic request body for unspecified endpoints
	return &OpenAPISchema{
		Type:        "object",
		Description: "Request data varies by endpoint",
		Properties: map[string]*OpenAPISchema{
			"data": {
				Type:        "object",
				Description: "Request payload",
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
					Schema: generateResponseSchema(route),
				},
			},
		},
		"400": {
			Description: "Bad request",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type:        "string",
						Description: "Error message",
					},
				},
			},
		},
		"500": {
			Description: "Internal server error",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type:        "string", 
						Description: "Error message",
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
					Schema: generateResponseSchema(route),
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
						Type:        "string",
						Description: "Authentication error message",
					},
				},
			},
		}
		
		responses["403"] = OpenAPIResponse{
			Description: "Forbidden - insufficient permissions",
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type:        "string",
						Description: "Authorization error message",
					},
				},
			},
		}
	}
	
	return responses
}

// generateResponseSchema creates appropriate response schema based on route characteristics
func generateResponseSchema(route RouteInfo) *OpenAPISchema {
	// Try to get schema from introspection registry
	registry := introspection.NewRouteRegistry()
	if routeSchema, found := registry.GetRouteSchema(route.Method, route.Path); found && routeSchema.Response != nil {
		return convertIntrospectionSchema(routeSchema.Response)
	}
	// Health endpoints have a specific schema
	if route.Path == "/health" {
		return &OpenAPISchema{
			Type: "object",
			Properties: map[string]*OpenAPISchema{
				"status": {
					Type:        "string",
					Description: "Health status",
					Example:     "healthy",
				},
				"module": {
					Type:        "string",
					Description: "Module name",
					Example:     route.ModuleName,
				},
			},
			Required: []string{"status"},
		}
	}
	
	// Stats endpoints (users module)
	if route.Path == "/stats" && route.ModuleName == "users" {
		return &OpenAPISchema{
			Type: "object",
			Properties: map[string]*OpenAPISchema{
				"total_users": {
					Type:        "integer",
					Description: "Total number of users",
				},
				"enabled_users": {
					Type:        "integer", 
					Description: "Number of enabled users",
				},
				"disabled_users": {
					Type:        "integer",
					Description: "Number of disabled users",
				},
				"banned_users": {
					Type:        "integer",
					Description: "Number of banned users",
				},
				"invalid_users": {
					Type:        "integer",
					Description: "Number of invalid users",
				},
			},
		}
	}
	
	// List endpoints (users module)
	if route.Path == "/" && route.ModuleName == "users" && route.Method == "GET" {
		return &OpenAPISchema{
			Type: "object", 
			Properties: map[string]*OpenAPISchema{
				"users": {
					Type: "array",
					Items: &OpenAPISchema{
						Type: "object",
						Description: "User object",
					},
					Description: "Array of users",
				},
				"total": {
					Type:        "integer",
					Description: "Total number of users",
				},
				"page": {
					Type:        "integer",
					Description: "Current page number",
				},
				"page_size": {
					Type:        "integer", 
					Description: "Number of items per page",
				},
				"total_pages": {
					Type:        "integer",
					Description: "Total number of pages",
				},
			},
		}
	}
	
	// Admin permission endpoints (groups module)
	if route.ModuleName == "groups" && (strings.Contains(route.Path, "/admin/permissions/services") || 
		strings.Contains(route.Path, "/admin/permissions/assignments")) {
		if strings.Contains(route.HandlerName, "list") || route.Method == "GET" {
			// List endpoints return {items: [], count: int} format
			var itemName string
			if strings.Contains(route.Path, "services") {
				itemName = "services"
			} else if strings.Contains(route.Path, "assignments") {
				itemName = "assignments"
			} else {
				itemName = "items"
			}
			
			return &OpenAPISchema{
				Type: "object",
				Properties: map[string]*OpenAPISchema{
					itemName: {
						Type: "array",
						Items: &OpenAPISchema{
							Type: "object",
							Description: "Resource item",
						},
						Description: fmt.Sprintf("Array of %s", itemName),
					},
					"count": {
						Type:        "integer",
						Description: "Total number of items",
					},
				},
			}
		}
	}
	
	// Auth profile endpoints
	if route.ModuleName == "auth" && strings.Contains(route.Path, "/profile") {
		return &OpenAPISchema{
			Type: "object",
			Properties: map[string]*OpenAPISchema{
				"character_id": {
					Type:        "integer",
					Description: "EVE character ID",
				},
				"character_name": {
					Type:        "string", 
					Description: "EVE character name",
				},
				"user_id": {
					Type:        "string",
					Description: "Internal user ID",
				},
				"corporation_id": {
					Type:        "integer",
					Description: "EVE corporation ID",
				},
				"alliance_id": {
					Type:        "integer", 
					Description: "EVE alliance ID",
				},
			},
		}
	}
	
	// SDE status endpoint
	if route.ModuleName == "sde" && route.Path == "/status" {
		return &OpenAPISchema{
			Type: "object",
			Properties: map[string]*OpenAPISchema{
				"version": {
					Type:        "string",
					Description: "Current SDE version",
				},
				"last_update": {
					Type:        "string",
					Format:      "date-time",
					Description: "Last update timestamp",
				},
				"status": {
					Type:        "string", 
					Description: "SDE status",
				},
			},
		}
	}
	
	// Scheduler task endpoints
	if route.ModuleName == "scheduler" && strings.Contains(route.Path, "/tasks") {
		if route.Method == "GET" && !strings.Contains(route.Path, "{taskID}") {
			// List tasks
			return &OpenAPISchema{
				Type: "object",
				Properties: map[string]*OpenAPISchema{
					"tasks": {
						Type: "array",
						Items: &OpenAPISchema{
							Type: "object",
							Description: "Task object",
						},
						Description: "Array of scheduled tasks",
					},
					"total": {
						Type:        "integer",
						Description: "Total number of tasks",
					},
				},
			}
		}
		return &OpenAPISchema{
			Type: "object",
			Properties: map[string]*OpenAPISchema{
				"id": {
					Type:        "string",
					Description: "Task ID",
				},
				"name": {
					Type:        "string",
					Description: "Task name",
				},
				"status": {
					Type:        "string",
					Description: "Task status",
				},
				"next_run": {
					Type:        "string",
					Format:      "date-time", 
					Description: "Next execution time",
				},
			},
		}
	}
	
	// Generic response for endpoints we haven't specifically defined
	return &OpenAPISchema{
		Type:        "object",
		Description: "Response data varies by endpoint",
		Properties: map[string]*OpenAPISchema{
			"message": {
				Type:        "string",
				Description: "Response message",
			},
		},
	}
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
		{Method: "GET", Path: "/eve/register", ModuleName: "auth", HandlerName: "eveFullLoginHandler", Description: "Initiate EVE SSO registration with full scopes"},
		{Method: "GET", Path: "/eve/callback", ModuleName: "auth", HandlerName: "eveCallbackHandler", Description: "Handle EVE SSO callback"},
		{Method: "POST", Path: "/eve/refresh", ModuleName: "auth", HandlerName: "eveRefreshHandler", Description: "Refresh access token"},
		{Method: "GET", Path: "/eve/verify", ModuleName: "auth", HandlerName: "eveVerifyHandler", Description: "Verify JWT token"},
		{Method: "POST", Path: "/eve/token", ModuleName: "auth", HandlerName: "eveTokenExchangeHandler", Description: "Exchange EVE token for JWT (mobile apps)"},
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
	// Only include routes that are actually implemented in internal/groups/routes/routes.go
	return []RouteInfo{
		{Method: "GET", Path: "/health", ModuleName: "groups", HandlerName: "HealthCheck", Description: "Groups module health check"},
		
		// Group management (public list, protected operations)
		{Method: "GET", Path: "/", ModuleName: "groups", HandlerName: "ListGroups", Description: "List all groups with optional filtering"},
		{Method: "GET", Path: "/{groupID}", ModuleName: "groups", HandlerName: "GetGroup", Description: "Get specific group details (requires groups:management:read)"},
		{Method: "POST", Path: "/", ModuleName: "groups", HandlerName: "CreateGroup", Description: "Create a new group (requires groups:management:write)"},
		{Method: "PUT", Path: "/{groupID}", ModuleName: "groups", HandlerName: "UpdateGroup", Description: "Update existing group (requires groups:management:write)"},
		{Method: "DELETE", Path: "/{groupID}", ModuleName: "groups", HandlerName: "DeleteGroup", Description: "Delete group (requires groups:management:delete)"},
		
		// Member management
		{Method: "GET", Path: "/{groupID}/members", ModuleName: "groups", HandlerName: "GetGroupMembers", Description: "List group members (requires groups:management:read)"},
		{Method: "POST", Path: "/{groupID}/members", ModuleName: "groups", HandlerName: "AddMember", Description: "Add member to group (requires groups:management:write)"},
		{Method: "DELETE", Path: "/{groupID}/members/{characterID}", ModuleName: "groups", HandlerName: "RemoveMember", Description: "Remove member from group (requires groups:management:write)"},
		
		// Permission checking (only granular check is implemented)
		{Method: "POST", Path: "/permissions/granular/check", ModuleName: "groups", HandlerName: "CheckGranularPermission", Description: "Check granular permission for authenticated user"},
		
		// Admin endpoints (super admin only) - Service management
		{Method: "GET", Path: "/admin/services", ModuleName: "groups", HandlerName: "ListServices", Description: "List all permission services (super admin only)"},
		{Method: "POST", Path: "/admin/services", ModuleName: "groups", HandlerName: "CreateService", Description: "Create new permission service (super admin only)"},
		{Method: "GET", Path: "/admin/services/{serviceName}", ModuleName: "groups", HandlerName: "GetService", Description: "Get permission service details (super admin only)"},
		{Method: "PUT", Path: "/admin/services/{serviceName}", ModuleName: "groups", HandlerName: "UpdateService", Description: "Update permission service (super admin only)"},
		{Method: "DELETE", Path: "/admin/services/{serviceName}", ModuleName: "groups", HandlerName: "DeleteService", Description: "Delete permission service (super admin only)"},
		
		// Admin endpoints - Permission assignment management
		{Method: "POST", Path: "/admin/permissions", ModuleName: "groups", HandlerName: "GrantPermission", Description: "Grant permission to subject (super admin only)"},
		{Method: "DELETE", Path: "/admin/permissions", ModuleName: "groups", HandlerName: "RevokePermission", Description: "Revoke permission from subject (super admin only)"},
		{Method: "GET", Path: "/admin/permissions/assignments", ModuleName: "groups", HandlerName: "ListPermissionAssignments", Description: "List permission assignments (super admin only)"},
		
		// Admin endpoints - Utility
		{Method: "GET", Path: "/admin/subjects/groups", ModuleName: "groups", HandlerName: "ListSubjectGroups", Description: "List groups for permission assignment (super admin only)"},
		{Method: "GET", Path: "/admin/audit", ModuleName: "groups", HandlerName: "GetAuditLogs", Description: "Get permission audit logs (super admin only)"},
		{Method: "GET", Path: "/admin/stats", ModuleName: "groups", HandlerName: "GetGroupStats", Description: "Get group statistics (super admin only)"},
	}
}

// getDevRoutes returns static route definitions for the dev module
func getDevRoutes() []RouteInfo {
	return []RouteInfo{
		// Public endpoints (no authentication required)
		{Method: "GET", Path: "/health", ModuleName: "dev", HandlerName: "HealthCheck", Description: "Dev module health check"},
		{Method: "GET", Path: "/status", ModuleName: "dev", HandlerName: "GetStatus", Description: "Module status information"},
		{Method: "GET", Path: "/services", ModuleName: "dev", HandlerName: "GetServices", Description: "Service discovery information"},
		
		// ESI testing endpoints (require auth)
		{Method: "GET", Path: "/esi/status", ModuleName: "dev", HandlerName: "GetESIStatus", Description: "Get EVE Online server status"},
		{Method: "GET", Path: "/character/{characterID}", ModuleName: "dev", HandlerName: "GetCharacter", Description: "Get character information"},
		{Method: "GET", Path: "/alliance/{allianceID}", ModuleName: "dev", HandlerName: "GetAlliance", Description: "Get alliance information"},
		{Method: "GET", Path: "/corporation/{corporationID}", ModuleName: "dev", HandlerName: "GetCorporation", Description: "Get corporation information"},
		{Method: "GET", Path: "/universe/system/{systemID}", ModuleName: "dev", HandlerName: "GetSystem", Description: "Get solar system information"},
		{Method: "POST", Path: "/esi/test", ModuleName: "dev", HandlerName: "TestESIEndpoint", Description: "Test custom ESI endpoints"},
		
		// SDE testing endpoints (require auth)
		{Method: "GET", Path: "/sde/status", ModuleName: "dev", HandlerName: "GetSDEStatus", Description: "Get SDE service status"},
		{Method: "GET", Path: "/sde/entity/{type}/{id}", ModuleName: "dev", HandlerName: "GetSDEEntity", Description: "Get specific SDE entity"},
		{Method: "GET", Path: "/sde/redis/{type}", ModuleName: "dev", HandlerName: "GetRedisSDEEntities", Description: "Get all Redis-based SDE entities of a type"},
		{Method: "GET", Path: "/sde/redis/{type}/{id}", ModuleName: "dev", HandlerName: "GetRedisSDEEntity", Description: "Get specific Redis-based SDE entity"},
		{Method: "GET", Path: "/sde/types", ModuleName: "dev", HandlerName: "GetSDETypes", Description: "Get all SDE types"},
		{Method: "GET", Path: "/sde/types/published", ModuleName: "dev", HandlerName: "GetSDETypesPublished", Description: "Get published SDE types only"},
		{Method: "GET", Path: "/sde/agent/{agentID}", ModuleName: "dev", HandlerName: "GetSDEAgent", Description: "Get SDE agent information"},
		{Method: "GET", Path: "/sde/category/{categoryID}", ModuleName: "dev", HandlerName: "GetSDECategory", Description: "Get SDE category information"},
		{Method: "GET", Path: "/sde/blueprint/{blueprintID}", ModuleName: "dev", HandlerName: "GetSDEBlueprint", Description: "Get SDE blueprint information"},
		
		// Universe SDE endpoints (require auth)
		{Method: "GET", Path: "/sde/universe/{type}/{region}/systems", ModuleName: "dev", HandlerName: "GetUniverseRegionSystems", Description: "Get all systems in a region"},
		{Method: "GET", Path: "/sde/universe/{type}/{region}/{constellation}/systems", ModuleName: "dev", HandlerName: "GetUniverseConstellationSystems", Description: "Get all systems in a constellation"},
		{Method: "GET", Path: "/sde/universe/{type}/{region}", ModuleName: "dev", HandlerName: "GetUniverseRegion", Description: "Get region data"},
		{Method: "GET", Path: "/sde/universe/{type}/{region}/{constellation}", ModuleName: "dev", HandlerName: "GetUniverseConstellation", Description: "Get constellation data"},
		{Method: "GET", Path: "/sde/universe/{type}/{region}/{constellation}/{system}", ModuleName: "dev", HandlerName: "GetUniverseSystem", Description: "Get system data"},
		
		// Testing and validation endpoints (require auth)
		{Method: "POST", Path: "/test/validate", ModuleName: "dev", HandlerName: "RunValidationTest", Description: "Run validation tests"},
		{Method: "POST", Path: "/test/performance", ModuleName: "dev", HandlerName: "RunPerformanceTest", Description: "Run performance tests"},
		{Method: "POST", Path: "/test/bulk", ModuleName: "dev", HandlerName: "RunBulkTest", Description: "Run bulk tests"},
		
		// Cache testing endpoints (require auth)
		{Method: "GET", Path: "/cache/stats", ModuleName: "dev", HandlerName: "GetCacheStats", Description: "Get cache statistics"},
		{Method: "POST", Path: "/cache/test", ModuleName: "dev", HandlerName: "TestCache", Description: "Test cache operations"},
		{Method: "DELETE", Path: "/cache/{key}", ModuleName: "dev", HandlerName: "DeleteCacheKey", Description: "Delete cache key"},
		
		// Mock data generation (require auth)
		{Method: "POST", Path: "/mock", ModuleName: "dev", HandlerName: "GenerateMockData", Description: "Generate mock data for testing"},
		
		// Debug endpoints (require auth)
		{Method: "POST", Path: "/debug/session", ModuleName: "dev", HandlerName: "CreateDebugSession", Description: "Create a new debug session"},
		{Method: "GET", Path: "/debug/session/{sessionID}", ModuleName: "dev", HandlerName: "GetDebugSession", Description: "Retrieve a debug session"},
		{Method: "POST", Path: "/debug/session/{sessionID}/action", ModuleName: "dev", HandlerName: "PerformDebugAction", Description: "Perform a debug action"},
		
		// Health check endpoints (require auth)
		{Method: "GET", Path: "/health/components", ModuleName: "dev", HandlerName: "GetComponentHealth", Description: "Get component health information"},
		{Method: "POST", Path: "/health/check", ModuleName: "dev", HandlerName: "RunHealthCheck", Description: "Run comprehensive health checks"},
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
		// Health check (not listed as it might be public) 
		{Method: "GET", Path: "/health", ModuleName: "notifications", HandlerName: "HealthHandler", Description: "Notifications module health check"},
		
		// Main notification operations (require notifications.messages.read/write)
		{Method: "GET", Path: "/", ModuleName: "notifications", HandlerName: "GetNotifications", Description: "Get user's notifications with filtering"},
		{Method: "GET", Path: "/stats", ModuleName: "notifications", HandlerName: "GetNotificationStats", Description: "Get notification statistics"},
		{Method: "POST", Path: "/", ModuleName: "notifications", HandlerName: "SendNotification", Description: "Send a new notification"},
		{Method: "POST", Path: "/bulk", ModuleName: "notifications", HandlerName: "BulkUpdateNotifications", Description: "Bulk operations on notifications"},
		
		// Individual notification operations
		{Method: "GET", Path: "/{id}", ModuleName: "notifications", HandlerName: "GetNotification", Description: "Get specific notification"},
		{Method: "PUT", Path: "/{id}", ModuleName: "notifications", HandlerName: "UpdateNotification", Description: "Update notification status"},
		{Method: "DELETE", Path: "/{id}", ModuleName: "notifications", HandlerName: "DeleteNotification", Description: "Delete specific notification"},
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
	// SDE module currently only has minimal implementation
	return []RouteInfo{
		{Method: "GET", Path: "/health", ModuleName: "sde", HandlerName: "HealthCheck", Description: "SDE module health check"},
		{Method: "GET", Path: "/status", ModuleName: "sde", HandlerName: "GetStatus", Description: "Get current SDE status (placeholder implementation)"},
	}
}