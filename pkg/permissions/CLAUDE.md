# Permissions Package (pkg/permissions)

## Overview

The permissions package provides a comprehensive permission system for the go-falcon EVE Online API gateway. It implements a hybrid approach with static (hardcoded) permissions for core system functions and dynamic (configurable) permissions that can be registered by services and managed through the admin interface.

**Status**: Production Ready - Complete Implementation with Background Registration
**Integration**: Fully integrated with groups system, application startup, and background service registration
**Latest Update**: Implemented background registration to prevent MongoDB write operations from blocking startup

## Architecture

### Core Components

- **Permission Registry**: Static and dynamic permission definitions
- **Permission Manager**: Central management for permission storage and checking
- **Permission Middleware**: HTTP middleware for route protection
- **Group Integration**: Seamless integration with existing groups system
- **MongoDB Storage**: Persistent storage for dynamic permissions and group assignments

### Files Structure

```
pkg/permissions/
├── types.go           # Core data structures and types
├── registry.go        # Static permission definitions and categories
├── manager.go         # PermissionManager with registration and checking logic
├── middleware.go      # HTTP middleware for permission enforcement
└── CLAUDE.md         # This documentation
```

## Permission Model

### Permission Structure

```go
type Permission struct {
    ID          string    `json:"id"`                    // e.g., "intel:reports:write"
    Service     string    `json:"service"`               // e.g., "intel", "scheduler"
    Resource    string    `json:"resource"`              // e.g., "reports", "tasks"
    Action      string    `json:"action"`                // e.g., "write", "read", "create"
    IsStatic    bool      `json:"is_static"`             // true = hardcoded, false = configurable
    Name        string    `json:"name"`                  // Human-readable name
    Description string    `json:"description"`           // What this permission allows
    Category    string    `json:"category"`              // UI grouping
    CreatedAt   time.Time `json:"created_at"`
}
```

### Permission Format

Permissions follow the pattern: `service:resource:action`

**Examples:**
- `groups:management:full` - Full group management
- `intel:reports:write` - Create/edit intelligence reports
- `scheduler:tasks:create` - Create scheduled tasks
- `corporation:members:view` - View corporation member data

## Static Permissions (Hardcoded)

### System Administration
- `system:admin:full` - Complete system access
- `system:config:manage` - System configuration

### User Management
- `users:management:full` - User account management
- `users:profiles:view` - View user profiles
- `auth:tokens:manage` - Authentication token management

### Group Management
- `groups:management:full` - Create, modify, delete groups
- `groups:memberships:manage` - Add/remove group members
- `groups:permissions:manage` - Assign permissions to groups
- `groups:view:all` - View group information

### Task Scheduling
- `scheduler:tasks:full` - Complete task scheduler management

## Dynamic Permissions (Service-Registered)

Services can register their own permissions during initialization:

```go
// Example: Intel service registering permissions
func (m *IntelModule) RegisterPermissions(permissionManager *permissions.PermissionManager) {
    servicePermissions := []permissions.Permission{
        {
            ID:          "intel:reports:write",
            Service:     "intel",
            Resource:    "reports",
            Action:      "write",
            Name:        "Write Intel Reports",
            Description: "Create and edit intelligence reports",
            Category:    "Intelligence",
        },
        {
            ID:          "intel:reports:read",
            Service:     "intel",
            Resource:    "reports", 
            Action:      "read",
            Name:        "Read Intel Reports",
            Description: "View intelligence reports",
            Category:    "Intelligence",
        },
    }
    
    err := permissionManager.RegisterServicePermissions(ctx, servicePermissions)
    if err != nil {
        log.Fatal("Failed to register intel permissions:", err)
    }
}
```

## Permission Manager

### Core Functions

```go
// Permission checking
func (pm *PermissionManager) HasPermission(ctx context.Context, characterID int64, permissionID string) (bool, error)

// Detailed permission information
func (pm *PermissionManager) CheckPermission(ctx context.Context, characterID int64, permissionID string) (*PermissionCheck, error)

// Service registration
func (pm *PermissionManager) RegisterServicePermissions(ctx context.Context, permissions []Permission) error

// Group permission assignment
func (pm *PermissionManager) GrantPermissionToGroup(ctx context.Context, groupID primitive.ObjectID, permissionID string, grantedBy int64) error
func (pm *PermissionManager) RevokePermissionFromGroup(ctx context.Context, groupID primitive.ObjectID, permissionID string) error
```

### Permission Resolution Logic

1. **Super Admin Check**: Users in "Super Administrator" group get all permissions automatically
2. **Group Permission Check**: Query group memberships and their assigned permissions
3. **Multi-Character Support**: Permissions are evaluated across all characters belonging to the same user
4. **Permission Inheritance**: Group-based permission assignment with audit trail

## Middleware Integration

### Basic Permission Enforcement

```go
// Single permission requirement
middleware := permissionMiddleware.RequirePermission("intel:reports:write")
router.Use(middleware)

// Multiple permission options (any one required)
middleware := permissionMiddleware.RequireAnyPermission(
    "groups:management:full", 
    "groups:memberships:manage",
)
router.Use(middleware)
```

### Groups Module Integration

The permissions system integrates seamlessly with the existing groups middleware:

```go
// Enhanced groups middleware with permission support
authMiddleware := NewAuthMiddleware(authService, groupService, permissionManager)

// Permission-based access control
user, err := authMiddleware.RequirePermission(ctx, authHeader, cookieHeader, "groups:management:full")

// Fallback to super admin if permission system unavailable
user, err := authMiddleware.RequireGroupAccess(ctx, authHeader, cookieHeader)
```

## Background Registration System

### Implementation

To prevent MongoDB write operations from blocking application startup, the permission system now uses background goroutines for dynamic registration:

```go
// Background permission registration during startup
go func() {
    log.Printf("🔄 Starting permission registration in background...")
    
    // Register scheduler permissions
    if err := schedulerModule.RegisterPermissions(ctx, permissionManager); err != nil {
        log.Printf("❌ Failed to register scheduler permissions: %v", err)
    } else {
        log.Printf("   ⏰ Scheduler permissions registered successfully")
    }
    
    // Register character permissions  
    if err := characterModule.RegisterPermissions(ctx, permissionManager); err != nil {
        log.Printf("❌ Failed to register character permissions: %v", err)
    } else {
        log.Printf("   🚀 Character permissions registered successfully")
    }
    
    log.Printf("✅ Background permission registration completed")
}()

// Background system group initialization with delay
go func() {
    time.Sleep(3 * time.Second) // Wait for service to start
    log.Printf("🔄 Starting system group permission initialization in background...")
    
    if err := permissionManager.InitializeSystemGroupPermissions(ctx); err != nil {
        log.Printf("❌ Failed to initialize system group permissions: %v", err)
    } else {
        log.Printf("✅ System group permissions initialized successfully")
    }
}()
```

### Benefits

- **Fast Startup**: Application starts immediately without waiting for MongoDB write operations
- **Non-Blocking**: Permission registration doesn't halt the main service initialization
- **Graceful Handling**: If background tasks hang, the main service remains operational
- **Logging**: Comprehensive logging tracks background registration progress
- **Fault Tolerance**: Individual permission failures don't affect other registrations

### Use Cases

This background registration approach is particularly useful when:
- MongoDB write operations experience high latency or timeouts
- Large numbers of permissions need to be registered during startup
- Application availability is prioritized over immediate permission availability
- Development environments have unreliable database connections

## Database Schema

### Permissions Collection

```go
{
    "_id": "intel:reports:write",
    "service": "intel",
    "resource": "reports",
    "action": "write",
    "is_static": false,
    "name": "Write Intel Reports",
    "description": "Create and edit intelligence reports",
    "category": "Intelligence",
    "created_at": "2025-01-10T12:00:00Z"
}
```

### Group Permissions Collection

```go
{
    "_id": ObjectId("..."),
    "group_id": ObjectId("..."),           // Reference to groups collection
    "permission_id": "intel:reports:write",
    "granted_by": 123456789,               // Character ID who granted
    "granted_at": "2025-01-10T12:00:00Z",
    "is_active": true,
    "updated_at": "2025-01-10T12:00:00Z"
}
```

## Integration with Groups System

### Permission Checking Flow

1. **Authentication**: JWT token validation via auth service
2. **Character Context**: Groups middleware resolves character information
3. **Group Membership**: Query all groups the user belongs to (multi-character)
4. **Permission Resolution**: Check if any group has the required permission
5. **Super Admin Override**: Super Administrator group bypasses all permission checks

### Automatic System Group Permissions

Static permissions are automatically assigned to system groups:

- **Super Administrator**: All permissions (hardcoded)
- **Authenticated Users**: Basic view permissions
- **Guest Users**: Public endpoint access only

## Usage Examples

### Service Permission Registration

```go
// In service module initialization
func NewIntelModule(db *mongo.Database, permissionManager *permissions.PermissionManager) *Module {
    // Register service permissions
    permissions := []permissions.Permission{
        {ID: "intel:reports:write", Service: "intel", Resource: "reports", Action: "write", /*...*/ },
        {ID: "intel:reports:read", Service: "intel", Resource: "reports", Action: "read", /*...*/ },
    }
    
    err := permissionManager.RegisterServicePermissions(context.Background(), permissions)
    if err != nil {
        log.Fatal("Permission registration failed:", err)
    }
    
    return &Module{/* ... */}
}
```

### Route Protection

```go
// Using Huma v2 with permission middleware
func (m *Module) RegisterRoutes(api huma.API, permissionMiddleware *permissions.PermissionMiddleware) {
    // Protected endpoint requiring specific permission
    huma.Post(api, "/intel/reports", m.createReport, 
        huma.Middlewares(permissionMiddleware.RequirePermission("intel:reports:write")))
    
    // Multiple permission options
    huma.Get(api, "/intel/reports", m.listReports,
        huma.Middlewares(permissionMiddleware.RequireAnyPermission(
            "intel:reports:read", 
            "intel:reports:write",
        )))
}
```

### Permission Checking in Handlers

```go
func (s *Service) CreateReport(ctx context.Context, input *CreateReportInput) (*ReportOutput, error) {
    // Permission is already checked by middleware
    // Handler logic here
    
    // Optional: Additional permission checks
    user := ctx.Value("authenticated_user").(*models.AuthenticatedUser)
    
    // Check if user can access specific intel area
    hasAreaAccess, err := s.permissionManager.HasPermission(ctx, int64(user.CharacterID), "intel:area:classified")
    if err != nil {
        return nil, err
    }
    
    if !hasAreaAccess {
        // Handle restricted access
    }
    
    // Continue with report creation
}
```

## Admin Interface Integration

### Permission Management API Endpoints (Future Phase 2)

```go
// List all available permissions
GET /admin/permissions

// Assign permission to group
POST /admin/groups/{groupID}/permissions
{
    "permission_id": "intel:reports:write"
}

// Remove permission from group
DELETE /admin/groups/{groupID}/permissions/{permissionID}

// Get group permissions
GET /admin/groups/{groupID}/permissions
```

### Permission Categories for UI

Permissions are organized into categories for admin interface:

- **System Administration**
- **User Management** 
- **Group Management**
- **Content Management**
- **Fleet Operations**
- **Intelligence**
- **Corporation Management**
- **Alliance Operations**

## Security Features

### Permission Restrictions

- **Static Permission Protection**: Core system permissions cannot be manually granted/revoked
- **Super Admin Restrictions**: Some permissions are restricted to super admin group only
- **Audit Trail**: All permission grants/revokes are tracked with user information
- **Input Validation**: Permission structure validation during registration

### Super Admin Safeguards

- Super Administrator group permissions are hardcoded and cannot be modified
- System admin permissions require super admin group membership
- Permission system fallback to group-based auth if permission manager unavailable

## Performance Considerations

### Database Optimization

- **Compound Indexes**: Optimized for permission checking queries
- **Aggregation Pipelines**: Efficient group membership and permission resolution
- **Caching Ready**: Architecture supports future Redis caching implementation

### Query Patterns

```javascript
// Optimized permission check aggregation
[
    { $match: { character_id: 123456789, is_active: true } },
    { $lookup: { from: "group_permissions", localField: "group_id", foreignField: "group_id", as: "permissions" } },
    { $unwind: "$permissions" },
    { $match: { "permissions.permission_id": "intel:reports:write", "permissions.is_active": true } },
    { $limit: 1 }
]
```

## Error Handling

### Permission Check Errors

- **404**: Permission not found
- **403**: Permission denied
- **401**: Authentication required
- **500**: Permission system error

### Graceful Degradation

- Falls back to super admin check if permission system unavailable
- Continues operation if permission registration fails (with logging)
- Handles database connection issues gracefully

## Development Workflow

### Adding New Permissions

1. **Define Permission**: Add to service permission list
2. **Register During Init**: Call `RegisterServicePermissions` in module initialization
3. **Protect Routes**: Use permission middleware on relevant endpoints
4. **Document**: Update service documentation with new permissions

### Testing Permissions

```go
func TestPermissionCheck(t *testing.T) {
    // Setup test database and permission manager
    pm := permissions.NewPermissionManager(testDB)
    
    // Register test permissions
    testPerms := []permissions.Permission{
        {ID: "test:action:perform", Service: "test", Resource: "action", Action: "perform"},
    }
    err := pm.RegisterServicePermissions(context.Background(), testPerms)
    assert.NoError(t, err)
    
    // Test permission checking
    hasPermission, err := pm.HasPermission(context.Background(), testCharacterID, "test:action:perform")
    assert.NoError(t, err)
    assert.True(t, hasPermission)
}
```

## Future Enhancements (Phase 2)

### Planned Features

- **Permission Management API**: Full CRUD operations for permission assignment
- **Permission Templates**: Pre-configured permission sets for common roles
- **Time-Based Permissions**: Temporary permission grants with expiration
- **Conditional Permissions**: Location or context-based permission logic
- **Permission Delegation**: Users granting sub-permissions to others
- **Redis Caching**: Performance optimization for frequent permission checks
- **Advanced Audit Logging**: Detailed permission usage tracking

### Admin Interface Features

- **Drag & Drop Permission Assignment**: Visual permission management
- **Permission Conflict Detection**: Identify and resolve permission conflicts
- **Bulk Permission Operations**: Assign multiple permissions to multiple groups
- **Permission Usage Analytics**: Track which permissions are actually used
- **Permission History**: View permission changes over time

## Integration Points

### Current Integrations

- **Groups System**: Core integration for role-based access control
- **Auth System**: JWT validation and character context resolution
- **MongoDB**: Persistent storage for permissions and assignments

### Future Integrations

- **Redis**: Caching layer for permission checks
- **Audit System**: Comprehensive permission usage logging
- **Notification System**: Permission change notifications
- **EVE Online Integration**: ESI-based permission validation

## Dependencies

### Internal Dependencies
- `go-falcon/internal/auth` (authentication and user models)
- `go-falcon/internal/groups` (group membership resolution)
- `go-falcon/pkg/database` (MongoDB connection)

### External Dependencies
- `go.mongodb.org/mongo-driver` (MongoDB driver)
- `github.com/danielgtaylor/huma/v2` (HTTP framework integration)

## Contributing

1. Follow the established permission naming convention: `service:resource:action`
2. Use appropriate permission categories for UI organization
3. Include comprehensive descriptions for admin interface
4. Add proper error handling and logging
5. Update documentation for new permissions
6. Include tests for permission functionality

## Security Considerations

- **Principle of Least Privilege**: Only grant necessary permissions
- **Permission Validation**: All permissions validated during registration
- **Audit Requirements**: All permission changes logged
- **Static Permission Protection**: Core permissions cannot be modified
- **Super Admin Safeguards**: System-critical permissions restricted

This permissions system provides a solid foundation for fine-grained access control while maintaining compatibility with the existing groups system. Phase 1 focuses on the core infrastructure, with Phase 2 planned for advanced features and admin interface integration.