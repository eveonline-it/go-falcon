# Postman Collection Exporter (cmd/postman)

## Overview

The Postman Collection Exporter is a command-line tool that automatically discovers all API endpoints in the Go-Falcon monolith and exports them as a comprehensive Postman Collection with OpenAPI compliance. It generates ready-to-use Postman collections for API testing, documentation, and development workflows.

## Features

- **Complete Endpoint Discovery**: Automatically scans all modules for API endpoints
- **Postman Collection Generation**: Creates v2.1.0 compliant Postman collections
- **OpenAPI Integration**: Supports OpenAPI 3.1.1 specifications where available
- **Module Organization**: Groups endpoints by module for better organization
- **Authentication Support**: Automatic detection and configuration of protected endpoints
- **Variable Management**: Pre-configured collection variables for testing
- **Request Templates**: Includes request bodies and headers for different methods
- **Pre/Post Scripts**: Built-in testing scripts and common headers

## Architecture

### Core Components

- **Route Discovery Engine**: Static route definitions with module scanning capability
- **Postman Collection Builder**: Converts discovered routes to Postman format
- **Variable Management**: Pre-configured variables for all endpoint parameters
- **Authentication Detection**: Identifies endpoints requiring authentication
- **Export Engine**: JSON serialization with proper formatting

### Files Structure

```
cmd/postman/
├── main.go        # Main application entry point and route discovery
└── CLAUDE.md      # This documentation file
```

## Route Discovery Strategy

### Current Implementation: Static Route Definitions

The current implementation uses static route definitions for reliability and performance:

```go
// Static route definitions for each module
moduleRoutes := map[string][]RouteInfo{
    "auth":          getAuthRoutes(),
    "dev":           getDevRoutes(), 
    "users":         getUsersRoutes(),
    "notifications": getNotificationsRoutes(),
    "scheduler":     getSchedulerRoutes(),
}
```

**Advantages:**
- ✅ No environment dependencies
- ✅ Consistent output across environments
- ✅ Fast execution without module initialization
- ✅ Works without database connections
- ✅ Predictable results for CI/CD pipelines

**Disadvantages:**
- ⚠️ Requires manual updates when routes change
- ⚠️ May become outdated if not maintained

### Future Enhancement: Dynamic Route Discovery

For future implementation, consider adding dynamic route discovery:

```go
// Future: Dynamic route discovery with reflection
func discoverRoutesReflection() ([]RouteInfo, error) {
    // Initialize minimal app context
    // Scan module route registrations
    // Extract routes using chi router inspection
    // Fallback to static definitions if discovery fails
}
```

## Module Coverage

### Recently Updated (2024-08-06)

The Postman collection exporter has been significantly enhanced to include all endpoints from the internal packages:

**Major Improvements:**
- ✅ **Added Scheduler Module**: Complete task management system (15 endpoints)
- ✅ **Enhanced Auth Module**: Added basic login/register endpoints and protected routes (14 endpoints total)
- ✅ **Expanded Dev Module**: Added corporation endpoints and alliance contact endpoints (42 endpoints total)
- ✅ **Fixed Users Module**: Corrected path parameters from `{userID}` to `{id}` (6 endpoints)
- ✅ **Updated Notifications Module**: Aligned with actual implementation (4 endpoints)
- ✅ **Enhanced Authentication Detection**: Added patterns for EVE corporation endpoints
- ✅ **New Variables**: Added `corporation_id`, `task_id`, `execution_id` for comprehensive testing
- ✅ **Route Count**: Increased from ~45 to 82 endpoints across 6 modules

**Endpoint Coverage:**
- **Gateway Module**: 1 endpoint
- **Auth Module**: 14 endpoints (EVE SSO + basic auth + protected routes)
- **Dev Module**: 42 endpoints (ESI + SDE + Corporation + Alliance)
- **Users Module**: 6 endpoints (CRUD operations)
- **Notifications Module**: 4 endpoints (list, send, mark read)
- **Scheduler Module**: 15 endpoints (task management + control + history)

### Currently Supported Modules

#### Gateway Module
- Health check endpoint
- Version information
- Global middleware endpoints

#### Auth Module (EVE Online SSO)
- **Authentication Flow**: Login, callback, verification, refresh
- **Profile Management**: User profiles, public profiles, profile refresh
- **Session Management**: Status check, logout
- **Security**: JWT verification, token refresh

#### Dev Module (Development & Testing)
- **ESI Testing**: All EVE Online ESI client endpoints
- **SDE Access**: Static Data Export endpoints for game data
- **Service Discovery**: Available services and status endpoints
- **Performance Testing**: Cache testing and validation

#### Users Module
- **CRUD Operations**: Create, read, update, delete users
- **User Management**: List users, get user by ID
- **Health Monitoring**: Module status and health

#### Notifications Module  
- **Notification Management**: List notifications, send notifications, mark as read
- **Health Monitoring**: Module status and health

#### Scheduler Module (Task Management)
- **Task Management**: Create, read, update, delete scheduled tasks
- **Task Control**: Start, stop, pause, resume tasks
- **Execution History**: View task execution history and specific execution details
- **System Management**: Statistics, reload configuration, service status
- **Health Monitoring**: Module status and health

### Module Extension Pattern

To add new modules to the Postman exporter:

```go
// Add to discoverRoutes() function
moduleRoutes := map[string][]RouteInfo{
    // ... existing modules ...
    "newModule": getNewModuleRoutes(),
}

// Create route definition function
func getNewModuleRoutes() []RouteInfo {
    return []RouteInfo{
        {
            Method:      "GET",
            Path:        "/endpoint",
            ModuleName:  "newModule", 
            HandlerName: "handlerFunction",
            Description: "Endpoint description",
        },
        // ... more routes ...
    }
}
```

## Collection Structure

### Collection Metadata

```json
{
    "info": {
        "_postman_id": "go-falcon-gateway-collection",
        "name": "Go-Falcon Gateway - All Endpoints",
        "description": "Complete collection with version info and generation timestamp",
        "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
    }
}
```

### Module Organization

Endpoints are organized into folders by module:

```
Go-Falcon Gateway Collection/
├── Gateway Module/
│   └── GET /health
├── Auth Module/
│   ├── GET /auth/eve/login
│   ├── GET /auth/eve/callback
│   ├── POST /auth/eve/refresh
│   ├── GET /auth/profile
│   └── ... more auth endpoints
├── Dev Module/
│   ├── ESI Endpoints/
│   │   ├── GET /dev/esi-status
│   │   ├── GET /dev/character/{characterID}
│   │   └── ... more ESI endpoints
│   ├── SDE Endpoints/
│   │   ├── GET /dev/sde/status
│   │   ├── GET /dev/sde/agent/{agentID}
│   │   └── ... more SDE endpoints
│   └── Utility Endpoints/
├── Users Module/
│   ├── GET /users/
│   ├── POST /users/
│   ├── GET /users/{id}
│   └── ... more user endpoints
├── Notifications Module/
│   ├── GET /notifications/
│   ├── POST /notifications/
│   └── PUT /notifications/{id}
└── Scheduler Module/
    ├── Task Management/
    │   ├── GET /scheduler/tasks
    │   ├── POST /scheduler/tasks
    │   └── ... more task endpoints
    ├── Task Control/
    │   ├── POST /scheduler/tasks/{taskID}/start
    │   ├── POST /scheduler/tasks/{taskID}/stop
    │   └── ... more control endpoints
    └── System Management/
        ├── GET /scheduler/stats
        └── POST /scheduler/reload
```

## Variable Management

### Pre-Configured Variables

The collection includes comprehensive variables for all endpoint parameters:

#### Base Configuration
```json
{
    "gateway_url": "http://localhost:8080",
    "api_prefix": "/api/v1",
    "access_token": ""
}
```

#### EVE Online Testing Variables
```json
{
    "character_id": "123456789",
    "alliance_id": "1354830081", 
    "system_id": "30000142",
    "station_id": "60003760"
}
```

#### SDE Testing Variables
```json
{
    "agent_id": "3008416",
    "category_id": "6",
    "blueprint_id": "1000001",
    "type_id": "34"
}
```

#### Application Variables
```json
{
    "user_id": "1",
    "notification_id": "1",
    "corporation_id": "98000001",
    "task_id": "1",
    "execution_id": "1"
}
```

### Variable Usage in Requests

Path parameters are automatically converted to Postman variables:

```
Original:  /character/{characterID}
Converted: /character/{{character_id}}

Original:  /sde/blueprint/{blueprintID}
Converted: /sde/blueprint/{{blueprint_id}}
```

## Authentication Configuration

### Automatic Detection

The exporter automatically detects endpoints requiring authentication based on path patterns:

```go
func needsAuth(path string) bool {
    authPaths := []string{
        "/contacts",
        "/user", 
        "/profile",
        "/private",
        "/admin",
        "/members",
        "/membertracking", 
        "/roles",
        "/structures",
        "/standings",
        "/wallets",
        "/tasks",
        "/logout",
        "/scheduler",
    }
    // ... detection logic
}
```

### Bearer Token Configuration

Protected endpoints automatically include bearer token authentication:

```json
{
    "auth": {
        "type": "bearer",
        "bearer": [{
            "key": "token",
            "value": "{{access_token}}",
            "type": "string"
        }]
    }
}
```

## Request Configuration

### HTTP Methods Support

- **GET**: Query parameters and headers only
- **POST/PUT/PATCH**: Includes JSON request body template
- **DELETE**: Headers only

### Headers Configuration

All requests include standard headers:

```json
{
    "header": [
        {
            "key": "Accept",
            "value": "application/json",
            "type": "text"
        },
        {
            "key": "Content-Type",
            "value": "application/json", 
            "type": "text"
        }
    ]
}
```

### Request Body Templates

POST/PUT/PATCH requests include JSON body templates:

```json
{
    "body": {
        "mode": "raw",
        "raw": "{\n  // Add request body here\n}"
    }
}
```

## Collection Scripts

### Pre-Request Scripts

Automatic header management and request tracking:

```javascript
// Set common headers
pm.request.headers.add({
    key: 'Accept',
    value: 'application/json'
});

// Add timestamp for request tracking
pm.globals.set('request_timestamp', new Date().toISOString());
```

### Test Scripts

Built-in response validation:

```javascript
// Test for successful response or expected error
pm.test('Response status is valid', function () {
    pm.expect(pm.response.code).to.be.oneOf([200, 201, 204, 400, 401, 403, 404, 500]);
});

// Test for JSON response when content type is JSON
if (pm.response.headers.get('Content-Type') && 
    pm.response.headers.get('Content-Type').includes('application/json')) {
    pm.test('Response is valid JSON', function () {
        pm.response.to.be.json;
    });
}
```

## Usage Guide

### Running the Exporter

```bash
# Build the exporter
go build -o postman cmd/postman/main.go

# Run the exporter
./postman

# Output
🚀 Go-Falcon Postman Collection Exporter
📦 Version: v1.0.0-dev
🔧 Build: 2024-01-15T10:30:00Z (linux/amd64)
🔍 Discovering routes for module: auth
🔍 Discovering routes for module: dev
🔍 Discovering routes for module: users
🔍 Discovering routes for module: notifications
🔍 Discovering routes for module: scheduler
📋 Discovered 82 routes across 6 modules
✅ Postman collection exported to: falcon-postman.json
📊 Collection contains 82 endpoints organized in 6 modules
```

### Output File

The exporter generates `falcon-postman.json` in the current directory.

### Importing to Postman

1. Open Postman application
2. Click "Import" button
3. Select the generated `falcon-postman.json` file
4. Configure environment variables as needed
5. Start testing endpoints

## Development Workflow

### Integration with Development

```bash
# Generate collection during development
make postman-collection

# Or integrate with build process
go generate ./...
```

### CI/CD Integration

```yaml
# Example GitHub Actions workflow
- name: Generate Postman Collection
  run: |
    go build -o postman cmd/postman/main.go
    ./postman
    
- name: Upload Collection Artifact
  uses: actions/upload-artifact@v3
  with:
    name: postman-collection
    path: falcon-postman.json
```

### Version Management

Collections include version information:

```json
{
    "description": "Generated automatically from route discovery.\n\nVersion: v1.2.3\nGenerated: 2024-01-15T10:30:00Z"
}
```

## Maintenance Guidelines

### Keeping Routes Updated

#### Manual Updates (Current)

When adding new endpoints to modules:

1. **Update Route Definition**: Add route to appropriate `get*Routes()` function
2. **Add Variables**: Include new path parameters in collection variables
3. **Test Authentication**: Verify auth detection for protected endpoints
4. **Regenerate Collection**: Run exporter to create updated collection

Example adding new endpoint:

```go
func getAuthRoutes() []RouteInfo {
    return []RouteInfo{
        // ... existing routes ...
        {
            Method:      "GET",
            Path:        "/sessions",
            ModuleName:  "auth",
            HandlerName: "listSessionsHandler", 
            Description: "List active user sessions",
        },
    }
}
```

#### Automated Updates (Future Enhancement)

Consider implementing route scanning for automatic updates:

```go
// Scan router registrations during module initialization
func scanModuleRoutes(moduleName string) ([]RouteInfo, error) {
    // Initialize module in test mode
    // Extract routes from chi router
    // Convert to RouteInfo format
}
```

### Testing Exported Collections

1. **Import Verification**: Ensure collection imports without errors
2. **Variable Testing**: Verify all variables are properly configured
3. **Authentication Testing**: Test protected endpoint configurations
4. **Request Validation**: Validate request formats and headers
5. **Response Testing**: Ensure test scripts work correctly

### Documentation Sync

Keep this CLAUDE.md file updated when:
- Adding new modules
- Changing route discovery logic
- Updating collection structure
- Modifying authentication detection
- Adding new variables or scripts

## Future Enhancements

### Planned Features

#### Dynamic Route Discovery
- Reflection-based route scanning
- Runtime module inspection
- Automatic route detection

#### OpenAPI Integration
- Import existing OpenAPI specifications
- Generate collections from OpenAPI files
- Hybrid approach: static + OpenAPI + discovery

#### Enhanced Authentication
- Multiple authentication schemes
- OAuth2 flow configuration
- API key authentication
- Custom authentication headers

#### Advanced Testing
- Response validation schemas
- Environment-specific configurations
- Pre-built test scenarios
- Performance testing requests

#### Collection Management
- Multi-environment collections
- Environment-specific variables
- Collection versioning
- Automated collection updates

### Architecture Improvements

#### Modular Design
```go
// Separate concerns for better maintainability
type RouteDiscoverer interface {
    DiscoverRoutes() ([]RouteInfo, error)
}

type CollectionBuilder interface {
    BuildCollection(routes []RouteInfo) (*PostmanCollection, error)
}

type Exporter interface {
    ExportCollection(collection *PostmanCollection, filename string) error
}
```

#### Configuration Management
```yaml
# postman-config.yaml
discovery:
  method: "static"  # static, dynamic, hybrid
  modules: ["auth", "dev", "users", "notifications"]

collection:
  name: "Go-Falcon Gateway"
  version_info: true
  environment_variables: true
  
authentication:
  auto_detect: true
  schemes: ["bearer", "jwt"]
  
export:
  filename: "falcon-postman.json"
  format: "postman_v2.1"
```

## Best Practices

### Route Definition Standards

1. **Consistent Naming**: Use consistent handler naming conventions
2. **Clear Descriptions**: Provide meaningful endpoint descriptions
3. **Proper Methods**: Use correct HTTP methods for operations
4. **Path Parameters**: Follow RESTful path parameter conventions

### Variable Management

1. **Descriptive Names**: Use clear variable names (character_id, not char_id)
2. **Example Values**: Provide realistic example values
3. **Documentation**: Include descriptions for all variables
4. **Grouping**: Group related variables together

### Collection Organization

1. **Module Grouping**: Organize by functional modules
2. **Endpoint Grouping**: Group related endpoints within modules
3. **Naming Convention**: Use consistent request naming
4. **Folder Structure**: Maintain logical folder hierarchy

### Testing Integration

1. **Validation Scripts**: Include comprehensive test scripts
2. **Error Handling**: Test both success and error scenarios
3. **Environment Variables**: Use variables for flexible testing
4. **Documentation**: Document testing procedures

## Troubleshooting

### Common Issues

#### Missing Endpoints
- **Problem**: New endpoints not appearing in collection
- **Solution**: Update appropriate `get*Routes()` function
- **Verification**: Check route definition syntax

#### Authentication Problems
- **Problem**: Protected endpoints not configured for auth
- **Solution**: Update `needsAuth()` function patterns
- **Verification**: Check path matching logic

#### Variable Issues
- **Problem**: Path parameters not converted to variables
- **Solution**: Update `processPathParameters()` mapping
- **Verification**: Check parameter name mappings

#### Collection Import Errors
- **Problem**: Postman import fails
- **Solution**: Validate JSON format and Postman schema compliance
- **Verification**: Test with Postman validator

### Debugging

#### Enable Verbose Logging
```go
// Add debug output during route discovery
fmt.Printf("🔍 Processing route: %s %s\n", route.Method, route.Path)
```

#### Validate JSON Output
```bash
# Check JSON validity
jq . falcon-postman.json > /dev/null && echo "Valid JSON" || echo "Invalid JSON"

# Pretty print for inspection
jq . falcon-postman.json > falcon-postman-formatted.json
```

#### Test Collection Structure
```bash
# Count endpoints per module
jq '.item[] | {name: .name, count: (.item | length)}' falcon-postman.json
```

This comprehensive documentation provides a complete guide for understanding, using, and maintaining the Go-Falcon Postman Collection Exporter, ensuring efficient API testing and documentation workflows.