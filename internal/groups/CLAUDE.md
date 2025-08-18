# Groups Module (internal/groups)

## Overview

The Groups module provides comprehensive authorization and role management functionality for the Go Falcon application using the **Granular Permission System**:

**Granular Permission System**: Administrator-controlled, service-level permissions with fine-grained access control where administrators can manually control permissions for every service and resource.

## Granular Permission System

### Core Concepts

#### Services
Each module/service in the system (e.g., "sde", "auth", "scheduler") that can have permissions assigned.

#### Resources  
Specific entities within services (e.g., "entities" in SDE, "users" in auth, "tasks" in scheduler).

#### Actions
Operations that can be performed on resources:
- **read**: View/access the resource
- **write**: Modify the resource  
- **delete**: Remove the resource
- **admin**: Full administrative control

#### Subjects
Who can receive permissions:
- **member**: Individual EVE character
- **corporation**: EVE Online corporation
- **alliance**: EVE Online alliance

### Database Schema

#### Services Collection
```json
{
  "_id": "ObjectId",
  "name": "sde",
  "display_name": "Static Data Export", 
  "description": "EVE Online static data management",
  "resources": [
    {
      "name": "entities",
      "display_name": "SDE Entities",
      "actions": ["read", "write", "delete", "admin"],
      "enabled": true
    }
  ],
  "enabled": true,
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### Permission Assignments Collection
```json
{
  "_id": "ObjectId",
  "service": "sde",
  "resource": "entities",
  "action": "read",
  "subject_type": "group", 
  "subject_id": "group_object_id",
  "granted_by": "admin_character_id",
  "granted_at": "timestamp",
  "expires_at": "timestamp (optional)",
  "reason": "Business justification",
  "enabled": true
}
```

### Admin API Endpoints

**Note**: All admin endpoints require super admin privileges via `SUPER_ADMIN_CHARACTER_ID` environment variable.

#### Service Management
```bash
# List all services
GET /admin/permissions/services

# Create a new service
POST /admin/permissions/services
{
  "name": "my_service",
  "display_name": "My Service",
  "description": "Custom service description",
  "resources": [
    {
      "name": "items",
      "display_name": "Service Items", 
      "actions": ["read", "write", "delete"]
    }
  ]
}

# Get specific service
GET /admin/permissions/services/{serviceName}

# Update service
PUT /admin/permissions/services/{serviceName}
{
  "description": "Updated description",
  "resources": [...]
}

# Delete service (removes all associated permissions)
DELETE /admin/permissions/services/{serviceName}
```

#### Permission Assignment
```bash
# Grant permission
POST /admin/permissions/assignments
{
  "service": "sde",
  "resource": "entities", 
  "action": "read",
  "subject_type": "group",
  "subject_id": "group_object_id",
  "expires_at": "2024-12-31T23:59:59Z",
  "reason": "Business requirement"
}

# Revoke permission  
DELETE /admin/permissions/assignments/{assignmentID}
?service=sde&resource=entities&action=read&subject_type=group&subject_id=group_id

# Check permission
POST /admin/permissions/check
{
  "service": "sde",
  "resource": "entities",
  "action": "read", 
  "character_id": 123456789
}
```

#### Utility Endpoints
```bash
# List available groups for assignment
GET /admin/permissions/subjects/groups

# Validate subject exists
GET /admin/permissions/subjects/validate?type=group&id=group_id

# Get audit logs
GET /admin/permissions/audit

# Get user permission summary
GET /admin/permissions/check/user/{characterID}

# Get service-specific permissions
GET /admin/permissions/check/service/{serviceName}
```

### Middleware Usage

#### Basic Permission Checking
```go
// Require specific permission (Chi router middleware)
r.With(groupsModule.RequireGranularPermission("sde", "entities", "read")).Get("/sde/data", handler)

// Optional permission (adds to context)
r.With(groupsModule.OptionalGranularPermission("sde", "entities", "write")).Get("/sde/info", handler)

// Check permission in handler context
func handler(w http.ResponseWriter, r *http.Request) {
    allowed, err := groupsModule.CheckGranularPermission(r.Context(), characterID, "sde", "entities", "admin")
    if err != nil || !allowed {
        // Handle permission denial
    }
}
```

#### HUMA v2 Integration
For HUMA v2 endpoints, permission checking is integrated into the request validation:

```go
// HUMA v2 operation with permission middleware
huma.Register(api, huma.Operation{
    OperationID: "getSdeEntities", 
    Method:      http.MethodGet,
    Path:        "/admin/permissions/services/{serviceName}",
    Summary:     "Get service definition",
    Tags:        []string{"admin", "permissions"},
    Middlewares: huma.Middlewares{
        groupsModule.RequireGranularPermission("groups", "management", "read"),
    },
}, func(ctx context.Context, req *GetServiceRequest) (*GetServiceResponse, error) {
    // Handler logic with validated permissions
    return &GetServiceResponse{...}, nil
})
```

#### Interface for Other Modules
Other modules can integrate with the groups module using these interfaces:

```go
// GroupsModule interface used by other modules
type GroupsModule interface {
    // Chi router middleware functions
    RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler
    OptionalGranularPermission(service, resource, action string) func(chi.Router)
    RequireSuperAdmin() func(chi.Router)
    
    // Direct permission checking
    CheckGranularPermission(ctx context.Context, characterID int, service, resource, action string) (bool, error)
    IsSuperAdmin(ctx context.Context, characterID int) (bool, error)
    
    // Service accessors
    GetGroupService() *services.GroupService
    GetGranularPermissionService() *services.GranularPermissionService
}
```

#### Example Module Integration
```go
// In module constructor
func New(mongodb *database.MongoDB, redis *database.Redis, groupsModule GroupsModule) *Module {
    return &Module{
        BaseModule:   module.NewBaseModule("example", mongodb, redis, nil),
        groupsModule: groupsModule,
    }
}

// In route registration
func (m *Module) Routes(r chi.Router) {
    // Protected endpoints with granular permissions
    r.With(m.groupsModule.RequireGranularPermission("example", "items", "read")).Get("/", m.listHandler)
    r.With(m.groupsModule.RequireGranularPermission("example", "items", "write")).Post("/", m.createHandler)
    r.With(m.groupsModule.RequireGranularPermission("example", "items", "delete")).Delete("/{id}", m.deleteHandler)
}
```

#### Pre-defined Permission Checks
```go
// Common permissions available as constants
groups.SDEReadPermission      // sde.entities.read
groups.SDEWritePermission     // sde.entities.write  
groups.SDEAdminPermission     // sde.entities.admin
groups.AuthReadPermission     // auth.users.read
groups.AuthWritePermission    // auth.users.write
groups.GroupsAdminPermission  // groups.management.admin
groups.SchedulerAdminPermission // scheduler.tasks.admin
```

### Implementation Examples

#### 1. Granting Permissions to a Group
```bash
# Grant SDE read access to the "corporate" group
curl -X POST /admin/permissions/assignments \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service": "sde",
    "resource": "entities",
    "action": "read", 
    "subject_type": "group",
    "subject_id": "corporate_group_object_id",
    "reason": "Corporate members need SDE access"
  }'
```

#### 2. Granting Direct Member Access
```bash
# Grant auth admin access to specific character
curl -X POST /admin/permissions/assignments \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service": "auth", 
    "resource": "users",
    "action": "admin",
    "subject_type": "member",
    "subject_id": "123456789",
    "expires_at": "2024-12-31T23:59:59Z",
    "reason": "Temporary admin access for user management"
  }'
```

#### 3. Corporation-Level Permissions
```bash
# Grant corporation-wide scheduler access
curl -X POST /admin/permissions/assignments \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service": "scheduler",
    "resource": "tasks", 
    "action": "write",
    "subject_type": "corporation",
    "subject_id": "98000001",
    "reason": "Corporation officers can manage scheduled tasks"
  }'
```

### Security Features

#### Access Control
- **Super Admin Only**: Only super admins can manage the granular permission system
- **Audit Logging**: All permission changes logged with actor, timestamp, and reason
- **Permission Validation**: Services and resources must exist before permissions can be granted
- **Subject Validation**: Subjects (groups, members, corps, alliances) validated before assignment

#### Performance Optimization
- **MongoDB Indexes**: Optimized compound indexes for fast permission checking
- **Redis Caching**: Future implementation for permission result caching
- **Efficient Queries**: Permission checks use optimized aggregation queries

### Migration Guide

#### From Legacy to Granular System

1. **Identify Current Permissions**: Review existing group permissions and middleware usage
2. **Define Services**: Create service definitions for each module/area that needs permissions
3. **Map Resources**: Define resources within each service  
4. **Create Permission Assignments**: Grant granular permissions to replace group-based ones
5. **Update Middleware**: Replace legacy middleware with granular permission middleware
6. **Test & Validate**: Ensure all endpoints work with new permission system

#### Migration Example
```go
// OLD: Legacy group permission
r.With(m.RequirePermission("groups", "admin")).Post("/create", handler)

// NEW: Granular permission  
r.With(m.RequireGranularPermission("groups", "management", "write")).Post("/create", handler)
```

**✅ Migration Complete**: The groups module has been fully migrated to granular permissions. All legacy permission code has been removed:
- Legacy permission service completely removed
- All endpoints now use granular permissions (`groups.management.read/write/delete`)
- Clean, modern permission system with no deprecated code

### Best Practices

#### Service Design
- **Single Responsibility**: Each service should represent one functional area
- **Logical Resources**: Group related functionality into meaningful resources
- **Consistent Naming**: Use clear, consistent naming conventions
- **Minimal Actions**: Only define actions that are actually needed

#### Permission Assignment
- **Principle of Least Privilege**: Grant minimum necessary permissions
- **Group-Based Assignment**: Prefer group assignments over individual assignments when possible  
- **Document Reasons**: Always provide business justification for permission grants
- **Regular Review**: Periodically review and clean up unused permissions
- **Expiration Dates**: Use expiration for temporary access grants

#### Security Considerations
- **Validate Inputs**: Always validate service, resource, and action parameters
- **Audit Everything**: Log all permission checks and changes
- **Monitor Usage**: Track permission usage patterns for anomaly detection
- **Separate Concerns**: Keep permission logic separate from business logic

## Group Management

The Groups module provides traditional group management alongside granular permissions:

### Key Features
- **Default Group System**: Predefined groups for common access levels
- **Custom Groups**: Administrator-managed groups for specific use cases
- **Resource-Level Permissions**: Granular read/write/delete access control
- **Discord Integration**: Automatic role assignment and synchronization
- **Automated Validation**: Scheduled tasks to maintain group membership integrity
- **EVE Corporation/Alliance Integration**: Dynamic group assignment based on corporation and alliance membership

## Architecture

The Groups module follows the standard Go Falcon module architecture with **dual API implementation**:

### Storage & Caching
- **MongoDB Storage**: Group definitions, memberships, and granular permissions
- **Redis Caching**: Fast permission lookups and session data
- **Optimized Indexes**: Compound indexes for efficient permission queries

### API Implementation  
- **Traditional Chi Routes**: `/groups/*` endpoints for basic group management
- **HUMA v2 Endpoints**: `/huma-groups/*` endpoints for granular permissions with OpenAPI 3.1.1
- **Unified OpenAPI Spec**: Complete API documentation via HUMA v2 framework
- **Type-Safe Operations**: Automatic request/response validation for admin endpoints

### Integration Points
- **Scheduler Integration**: Automated membership validation tasks
- **Auth Module Integration**: Seamless integration with EVE SSO authentication
- **Cross-Module Middleware**: Granular permission checking for all modules

## Default Groups

### Login
- **Purpose**: Authenticated users and limited access
- **Permissions**: Read-only access to public resources
- **Assignment**: Automatic for unauthenticated requests
- **Discord Role**: Optional

### Full
- **Purpose**: Authenticated EVE Online characters with full ESI scopes
- **Permissions**: Read-only access to public resources
- **Assignment**: Automatic upon successful EVE SSO authentication
- **Discord Roles**: Optional

### Personal
- **Purpose**: Member that is manually approved
- **Permissions**: Access to personal resources and channels
- **Assignment**: Automatic upon successful EVE SSO authentication
- **Discord Roles**: Optional

### Corporate
- **Purpose**: Members of enabled EVE Online corporations
- **Permissions**: Access to corporation resources and channels
- **Assignment**: Automatic when character belongs to enabled corporation
- **Discord Roles**: Multiple roles across servers (e.g., "Corporate Member" on main server)
- **Validation**: Checked via scheduled tasks using ESI corporation membership

### Alliance
- **Purpose**: Members of enabled EVE Online alliances
- **Permissions**: Access to alliance-specific resources and channels
- **Assignment**: Automatic when character belongs to enabled alliance
- **Discord Roles**: Multiple roles across servers (e.g., "Alliance" on corp server)
- **Validation**: Checked via scheduled tasks using ESI alliance membership

### Administrators
- **Purpose**: Application administrators with elevated privileges
- **Permissions**: Full system access, user management, group creation
- **Assignment**: Manual assignment by existing administrators
- **Discord Roles**: Multiple administrative roles across servers (e.g., "Admin" on all managed servers)

### Super Admin
- **Purpose**: Ultimate system authority with unrestricted access
- **Permissions**: All system permissions, cannot be removed from group
- **Assignment**: Configured via `SUPER_ADMIN_CHARACTER_ID` environment variable
- **Discord Roles**: Multiple super admin roles across all servers (e.g., "Owner" role on all managed servers)

## Custom Groups

Administrators can create custom groups for specific organizational needs:

### Group Properties
- **Name**: Unique identifier for the group
- **Description**: Human-readable purpose description
- **Permissions**: Configurable resource-level permissions
- **Discord Roles**: Multiple Discord roles across different servers (server_id:role_name pairs)
- **Auto-Assignment Rules**: Optional criteria for automatic membership
- **Expiration**: Optional group membership expiration

### Permission Types
- **Read**: View resource data and content
- **Write**: Modify resource data and content
- **Delete**: Remove resources and data
- **Admin**: Full control over specific resource categories

## Configuration

### Environment Variables
```bash
# Required
SUPER_ADMIN_CHARACTER_ID=123456789  # EVE character ID for super admin

# Optional
GROUPS_CACHE_TTL=300               # Permission cache TTL in seconds (default: 300)
GROUPS_VALIDATION_INTERVAL=3600    # Membership validation interval in seconds (default: 3600)
DISCORD_ROLE_SYNC=true            # Enable Discord role synchronization (default: true)
DISCORD_SERVICE_URL=http://localhost:8080  # Discord service endpoint for role retrieval
```

### MongoDB Collections

#### groups
```json
{
  "_id": "ObjectId",
  "name": "string (unique)",
  "description": "string",
  "is_default": "boolean",
  "permissions": {
    "resource_type": ["read", "write", "delete", "admin"]
  },
  "discord_roles": [
    {
      "server_id": "string",
      "server_name": "string (optional)",
      "role_name": "string"
    }
  ],
  "auto_assignment_rules": {
    "corporation_ids": ["number"],
    "alliance_ids": ["number"],
    "min_security_status": "number"
  },
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "created_by": "character_id"
}
```

#### group_memberships
```json
{
  "_id": "ObjectId",
  "character_id": "number",
  "group_id": "ObjectId",
  "assigned_at": "timestamp",
  "assigned_by": "character_id",
  "expires_at": "timestamp (optional)",
  "last_validated": "timestamp",
  "validation_status": "valid|invalid|pending"
}
```

## API Architecture

The Groups module implements a **dual API architecture** using both traditional Chi routes and modern HUMA v2 endpoints:

### Traditional Routes (Chi Router)
- **Legacy group management**: `/groups/*` endpoints
- **Basic permission queries**: Simple GET/POST operations
- **Quick integration**: Easy to integrate with existing middleware

### HUMA v2 API Endpoints
- **Granular permissions**: `/huma-groups/admin/permissions/*` endpoints  
- **Type-safe validation**: Automatic request/response validation
- **OpenAPI 3.1.1**: Complete API specification with schemas
- **Enhanced error handling**: Structured error responses

## API Endpoints

### OpenAPI 3.1.1 Specification

The Groups module provides comprehensive OpenAPI documentation via HUMA v2:

```bash
# Complete OpenAPI specification
GET /huma-groups/openapi.json

# Interactive documentation  
GET /huma-groups/docs

# Schema definitions
GET /huma-groups/openapi.json#/components/schemas
```

### Administrative Endpoints (HUMA v2)

All administrative endpoints require **super admin privileges** and are implemented using HUMA v2 for type safety and automatic OpenAPI generation.

#### Service Management
```bash
# List all services
GET /huma-groups/admin/permissions/services
Authorization: Bearer JWT
Response: ServiceListResponse with service definitions

# Create a new service
POST /huma-groups/admin/permissions/services  
Authorization: Bearer JWT
Content-Type: application/json
Body: CreateServiceRequest with validation
Response: ServiceResponse with created service

# Get specific service
GET /huma-groups/admin/permissions/services/{serviceName}
Authorization: Bearer JWT
Response: ServiceResponse with service details

# Update service  
PUT /huma-groups/admin/permissions/services/{serviceName}
Authorization: Bearer JWT
Body: UpdateServiceRequest
Response: ServiceResponse with updated service

# Delete service (removes all associated permissions)
DELETE /huma-groups/admin/permissions/services/{serviceName}
Authorization: Bearer JWT
Response: DeleteResponse with operation status
```

#### Permission Assignment (HUMA v2)
```bash
# Grant permission
POST /huma-groups/admin/permissions/assignments
Authorization: Bearer JWT  
Body: CreatePermissionAssignmentRequest
Response: PermissionAssignmentResponse

# List permission assignments
GET /huma-groups/admin/permissions/assignments
Authorization: Bearer JWT
Query Parameters: service, resource, action, subject_type, subject_id
Response: PermissionAssignmentListResponse

# Revoke permission
DELETE /huma-groups/admin/permissions/assignments/{assignmentID}
Authorization: Bearer JWT
Response: DeleteResponse with operation status

# Check permission
POST /huma-groups/admin/permissions/check
Authorization: Bearer JWT
Body: PermissionCheckRequest
Response: PermissionCheckResponse
```

#### Utility Endpoints (HUMA v2)
```bash
# List available subjects for assignment
GET /huma-groups/admin/permissions/subjects/groups
Authorization: Bearer JWT
Response: SubjectListResponse

# Validate subject exists
GET /huma-groups/admin/permissions/subjects/validate
Authorization: Bearer JWT  
Query Parameters: type, id
Response: SubjectValidationResponse

# Get audit logs
GET /huma-groups/admin/permissions/audit
Authorization: Bearer JWT
Query Parameters: limit, offset, service, action
Response: AuditLogListResponse

# Get user permission summary
GET /huma-groups/admin/permissions/check/user/{characterID}  
Authorization: Bearer JWT
Response: UserPermissionSummaryResponse

# Get service-specific permissions
GET /huma-groups/admin/permissions/check/service/{serviceName}
Authorization: Bearer JWT
Response: ServicePermissionListResponse
```

### Traditional Group Management (Chi Routes)

Traditional group endpoints for backward compatibility and simple operations:

#### List Groups
```bash
GET /groups
Authorization: Bearer JWT (any authenticated user)
Response: List of groups with user's membership status
```

#### Create Group  
```bash
POST /groups
Authorization: Bearer JWT (administrators only)
Content-Type: application/json
Body: Group creation parameters
Response: Created group information
```

#### Update Group
```bash  
PUT /groups/{group_id}
Authorization: Bearer JWT (administrators only)
Content-Type: application/json
Body: Group update parameters
Response: Updated group information
```

#### Delete Group
```bash
DELETE /groups/{group_id}
Authorization: Bearer JWT (administrators only)
Note: Cannot delete default groups
Response: Deletion confirmation
```

### Membership Management (Chi Routes)

#### Add Member
```bash
POST /groups/{group_id}/members
Authorization: Bearer JWT (administrators only)
Content-Type: application/json
Body: { "character_id": 123456789, "expires_at": "2024-12-31T23:59:59Z" }
Response: Membership confirmation
```

#### Remove Member
```bash
DELETE /groups/{group_id}/members/{character_id}
Authorization: Bearer JWT (administrators only) 
Response: Removal confirmation
```

#### List Members
```bash
GET /groups/{group_id}/members
Authorization: Bearer JWT (administrators only)
Response: List of group members with status
```

### Permission Queries (Chi Routes)

#### Check Permission
```bash
GET /permissions/check?resource={resource}&action={action}
Authorization: Bearer JWT (any authenticated user)
Response: { "allowed": boolean, "groups": ["group_name"] }
```

#### User Permissions
```bash
GET /permissions/user  
Authorization: Bearer JWT (any authenticated user)
Response: Complete permission matrix for authenticated user
```

### Request/Response Schemas (HUMA v2)

#### Service Management DTOs
```go
type CreateServiceRequest struct {
    Name        string            `json:"name" validate:"required,min=2,max=50" doc:"Unique service identifier"`
    DisplayName string            `json:"display_name" validate:"required,min=2,max=100" doc:"Human-readable service name"`
    Description string            `json:"description" validate:"max=500" doc:"Service description"`
    Resources   []ResourceRequest `json:"resources" validate:"required,min=1" doc:"Service resources"`
    Enabled     bool              `json:"enabled" doc:"Whether service is enabled"`
}

type ResourceRequest struct {
    Name        string   `json:"name" validate:"required,min=2,max=50" doc:"Resource identifier"`
    DisplayName string   `json:"display_name" validate:"required,min=2,max=100" doc:"Human-readable resource name"`
    Actions     []string `json:"actions" validate:"required,min=1" doc:"Allowed actions for resource"`
    Enabled     bool     `json:"enabled" doc:"Whether resource is enabled"`
}
```

#### Permission Assignment DTOs
```go
type CreatePermissionAssignmentRequest struct {
    Service     string    `json:"service" validate:"required" doc:"Target service name"`
    Resource    string    `json:"resource" validate:"required" doc:"Target resource name"`
    Action      string    `json:"action" validate:"required" doc:"Action to grant"`
    SubjectType string    `json:"subject_type" validate:"required,oneof=member corporation alliance" doc:"Type of subject"`
    SubjectID   string    `json:"subject_id" validate:"required" doc:"Subject identifier"`
    ExpiresAt   *time.Time `json:"expires_at,omitempty" doc:"Optional expiration timestamp"`
    Reason      string    `json:"reason" validate:"required,min=10,max=500" doc:"Business justification"`
}

type PermissionCheckRequest struct {
    Service     string `json:"service" validate:"required" doc:"Service to check"`
    Resource    string `json:"resource" validate:"required" doc:"Resource to check"`
    Action      string `json:"action" validate:"required" doc:"Action to check"`
    CharacterID int    `json:"character_id" validate:"required" doc:"EVE character ID"`
}
```

### Error Handling (HUMA v2)

HUMA v2 provides structured error responses with detailed validation information:

```json
{
  "status": 400,
  "title": "Bad Request",
  "detail": "Validation failed for request body",
  "errors": [
    {
      "location": "body.service",
      "message": "Service name is required",
      "value": ""
    },
    {
      "location": "body.reason", 
      "message": "Reason must be at least 10 characters",
      "value": "test"
    }
  ]
}
```

## Integration Points

### Auth Module Integration
- **User Registration**: Automatic assignment to default groups
- **Token Validation**: Permission injection into JWT claims
- **Profile Updates**: Group membership included in user profiles

### Discord Service Integration
The Groups module integrates with an external Discord service for role management:

#### Service API Endpoints
```
GET /discord/servers                 # List all managed Discord servers
GET /discord/servers/{server_id}/roles # List roles for specific server
GET /discord/roles/{server_id}/{name} # Get role by server and name
POST /discord/servers/{server_id}/members/{user_id}/roles # Assign role to user on server
DELETE /discord/servers/{server_id}/members/{user_id}/roles/{role_name} # Remove role from user on server
GET /discord/members/{user_id}/roles # Get user's roles across all servers
POST /discord/members/{user_id}/roles/bulk # Bulk assign/remove roles across servers
```

#### Role Resolution
- **Multi-Server Support**: Roles resolved across multiple Discord servers
- **Dynamic Lookup**: Discord roles resolved by server_id and name at runtime
- **Service Communication**: HTTP API calls to Discord service for each server
- **Fallback Behavior**: Graceful handling when Discord service or specific servers unavailable
- **Role Validation**: Verify role exists on target server before assignment
- **Bulk Operations**: Batch role assignments across multiple servers for efficiency
- **Server Management**: Automatic discovery and management of Discord servers

#### Error Handling
- **Service Timeout**: Configurable timeout for Discord service calls
- **Retry Logic**: Exponential backoff for failed role operations per server
- **Circuit Breaker**: Temporary disable Discord sync per server if repeatedly fails
- **Partial Failures**: Handle scenarios where some servers succeed, others fail
- **Server Availability**: Track individual server status and skip unavailable servers
- **Audit Logging**: Log all Discord role operations with server context for troubleshooting

### Scheduler Integration
- **Membership Validation**: Hourly validation of corporation and alliance-based memberships
- **Expiration Cleanup**: Daily cleanup of expired group memberships
- **Discord Sync**: Periodic synchronization with Discord roles

## Security Features

### Access Control
- **Hierarchical Permissions**: Groups can inherit from parent groups
- **Resource Scoping**: Permissions limited to specific resource types
- **Audit Logging**: All group changes logged with actor and timestamp
- **Rate Limiting**: API endpoints protected against abuse

### Validation Mechanisms
- **Corporation/Alliance Membership**: ESI validation of corporate and alliance group eligibility
- **Token Verification**: JWT token validation for all operations
- **Permission Caching**: Redis-based caching with automatic invalidation
- **Expiration Handling**: Automatic removal of expired memberships

## Scheduled Tasks

### Corporation/Alliance Validation Task
```go
// Runs every hour
// Validates corporate group memberships against ESI data
// Removes users no longer in valid corporations or alliances
// Checks both corporation and alliance membership status
Schedule: "0 */1 * * *"
```

### Membership Cleanup Task
```go
// Runs daily at 3 AM
// Removes expired group memberships
// Updates Discord roles
Schedule: "0 3 * * *"
```

### Discord Sync Task
```go
// Runs every 30 minutes
// Synchronizes group memberships with Discord roles across all servers via Discord service
// Retrieves current roles from Discord service for each server and updates assignments
// Handles role assignment failures and individual server unavailability
// Processes bulk role changes across multiple servers efficiently
Schedule: "*/30 * * * *"
```

## Error Handling

### Common Error Scenarios
- **Permission Denied**: User lacks required group membership
- **Group Not Found**: Invalid group ID or deleted group
- **Membership Conflict**: User already member of conflicting group
- **Discord Service Failure**: Discord service unavailable or specific server unreachable
- **Role Retrieval Error**: Unable to fetch roles from Discord service for specific servers
- **Partial Server Failure**: Some Discord servers accessible, others unavailable
- **Cross-Server Role Conflict**: User has conflicting roles across different servers
- **ESI Validation Failure**: EVE Online API unavailable

### Error Responses
```json
{
  "error": "permission_denied",
  "message": "Insufficient permissions for this operation",
  "required_groups": ["administrators"],
  "user_groups": ["full"]
}
```

## Performance Considerations

### Caching Strategy
- **Permission Matrix**: Cached in Redis for 5 minutes
- **Group Memberships**: Cached per user session
- **Discord Roles**: Retrieved from service per server and cached for 15 minutes
- **Server Status**: Discord server availability cached for 5 minutes
- **ESI Validation**: Corporation and alliance validation results cached for 1 hour

### Database Optimization
- **Indexes**: Composite indexes on character_id + group_id
- **Aggregation**: Efficient permission queries using MongoDB aggregation
- **Connection Pooling**: Shared database connections across modules

## Development & Testing

### Local Development
- **Test Groups**: Predefined test groups for development
- **Mock Discord Service**: Discord service integration can be mocked for testing
- **ESI Mocking**: Corporation and alliance validation can use mock ESI responses

### Testing Strategy
- **Unit Tests**: Individual group operations and permission checks
- **Integration Tests**: Full workflow testing with auth module
- **Load Tests**: Permission checking under high concurrent load

## Future Enhancements

### Planned Features
- **Group Hierarchies**: Parent-child group relationships
- **Conditional Permissions**: Time-based or location-based permissions
- **Approval Workflows**: Membership requests requiring approval
- **Advanced Analytics**: Group usage and permission analytics

### API Extensions
- **Bulk Operations**: Batch membership changes
- **Group Templates**: Predefined group configurations
- **Permission Inheritance**: Complex permission inheritance rules
