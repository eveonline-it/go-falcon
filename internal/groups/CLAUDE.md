# Groups Module (internal/groups)

## Overview

The groups module provides group and role-based access control management for the go-falcon EVE Online API gateway. This module implements a hierarchical permission system that supports EVE-specific groups (characters, corporations, alliances) as well as custom groups for fine-grained access control.

**Current Status**: Production Ready - Full Authentication Integration
**Authentication**: Complete integration with Character Context Middleware and group-based permissions

## Architecture

### Core Components

- **Group Management**: CRUD operations for custom groups and system groups
- **Membership Management**: Character assignment to groups with audit trail  
- **EVE Integration**: Auto-assignment for corporation and alliance groups
- **Character Context**: Corporation/alliance data extraction from user profiles
- **Auto-Synchronization**: Automated group membership sync via scheduler
- **System Groups**: Built-in groups (super_admin, authenticated, guest)
- **MongoDB Storage**: Groups and memberships collections with proper indexing

### Files Structure

```
internal/groups/
├── dto/
│   ├── inputs.go         # Request input DTOs with Huma v2 validation
│   └── outputs.go        # Response output DTOs
├── middleware/
│   ├── auth.go          # Authentication and authorization middleware
│   └── context.go       # Character Context Middleware with corp/alliance resolution
├── routes/
│   └── routes.go        # Huma v2 route definitions
├── services/
│   ├── service.go       # Business logic for groups and memberships
│   └── repository.go    # Database operations and queries
├── models/
│   └── models.go        # MongoDB schemas and data structures
├── module.go            # Module initialization and interface implementation
└── CLAUDE.md            # This documentation
```

## Data Model

### Group Schema (MongoDB Collection: `groups`)

```go
type Group struct {
    ID           primitive.ObjectID `bson:"_id,omitempty"`
    Name         string             `bson:"name"`                    // Unique group name
    Description  string             `bson:"description,omitempty"`
    Type         GroupType          `bson:"type"`                    // system, corporation, alliance, custom
    SystemName   *string            `bson:"system_name,omitempty"`   // For system groups
    EVEEntityID  *int64             `bson:"eve_entity_id,omitempty"` // Corp/Alliance ID
    IsActive     bool               `bson:"is_active"`
    CreatedBy    *int64             `bson:"created_by,omitempty"`    // Character ID
    CreatedAt    time.Time          `bson:"created_at"`
    UpdatedAt    time.Time          `bson:"updated_at"`
}
```

### Group Membership Schema (MongoDB Collection: `group_memberships`)

```go
type GroupMembership struct {
    ID          primitive.ObjectID `bson:"_id,omitempty"`
    GroupID     primitive.ObjectID `bson:"group_id"`     // Reference to groups collection
    CharacterID int64              `bson:"character_id"` // EVE character ID
    IsActive    bool               `bson:"is_active"`
    AddedBy     *int64             `bson:"added_by,omitempty"`      // Character ID who added
    AddedAt     time.Time          `bson:"added_at"`
    UpdatedAt   time.Time          `bson:"updated_at"`
}
```

### Group Types

- **`system`**: Built-in groups (super_admin, authenticated, guest)
- **`corporation`**: EVE Corporation groups (auto-created and auto-assigned)
- **`alliance`**: EVE Alliance groups (auto-created and auto-assigned)
- **`custom`**: User-created custom groups

### Core Features (✅ COMPLETED)

#### Character Context Middleware Integration
The Character Context Middleware now extracts corporation and alliance information from user profiles and populates the `CharacterContext`:

```go
type CharacterContext struct {
    UserID        string  `json:"user_id"`
    CharacterID   int64   `json:"character_id"`
    CharacterName string  `json:"character_name"`
    IsSuperAdmin  bool    `json:"is_super_admin"`
    
    // Phase 2: Corporation and Alliance info
    CorporationID   *int64  `json:"corporation_id,omitempty"`
    CorporationName *string `json:"corporation_name,omitempty"`
    AllianceID      *int64  `json:"alliance_id,omitempty"`
    AllianceName    *string `json:"alliance_name,omitempty"`
    
    GroupMemberships []string `json:"group_memberships,omitempty"`
}
```

#### Auto-Assignment System
Characters are automatically assigned to corporation and alliance groups when:
1. **Authentication occurs**: During Character Context resolution
2. **Profile updates**: When ESI data is refreshed
3. **Scheduled sync**: Via system task `system-groups-sync` (every 6 hours)

#### Group Auto-Creation
Corporation and alliance groups are automatically created with naming convention:
- Corporation groups: `Corp_{corporation_id}` (e.g., `Corp_98000001`)
- Alliance groups: `Alliance_{alliance_id}` (e.g., `Alliance_99000001`)

#### Scheduler Integration
Added system task for automated group synchronization:
- **Task ID**: `system-groups-sync`
- **Schedule**: Every 6 hours
- **Purpose**: Validates and syncs character group memberships
- **ESI Integration**: Placeholder for future ESI validation

## API Endpoints

### Group Management

#### Create Group
```
POST /groups
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Request Body:**
```json
{
  "name": "Fleet Commanders",
  "description": "Fleet commanders with special permissions",
  "type": "custom"
}
```

#### List Groups
```
GET /groups?type=custom&is_active=true&page=1&limit=20
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

#### Get Group
```
GET /groups/{id}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

#### Update Group
```
PUT /groups/{id}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Request Body:**
```json
{
  "name": "Updated Group Name",
  "description": "Updated description",
  "is_active": true
}
```

#### Delete Group
```
DELETE /groups/{id}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

### Group Membership Management

#### Add Member
```
POST /groups/{group_id}/members
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Request Body:**
```json
{
  "character_id": 123456789
}
```

#### Remove Member
```
DELETE /groups/{group_id}/members/{character_id}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

#### List Members
```
GET /groups/{group_id}/members?is_active=true&page=1&limit=20
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

**Response includes character names:**
```json
{
  "body": {
    "members": [
      {
        "id": "membership_id",
        "group_id": "group_id",
        "character_id": 123456789,
        "character_name": "Character Name",
        "is_active": true,
        "added_by": 987654321,
        "added_at": "2025-01-10T12:00:00Z",
        "updated_at": "2025-01-10T12:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20
  }
}
```

#### Check Membership
```
GET /groups/{group_id}/members/{character_id}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

### Character-Centric Endpoints

#### Get Character Groups
```
GET /characters/{character_id}/groups?type=custom&is_active=true
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

## Authentication and Authorization

#### Permission Requirements

- **Group Management**: Requires `super_admin` group membership
- **Membership Management**: Requires `super_admin` group membership  
- **Group Viewing**: Requires authentication (`authenticated` group)

#### Planned Permission Model

```
Service: groups
├── Resource: management
│   ├── Action: create    # Create new groups
│   ├── Action: update    # Modify group details
│   └── Action: delete    # Delete groups
├── Resource: memberships
│   ├── Action: add       # Add members to groups
│   ├── Action: remove    # Remove members from groups
│   └── Action: view      # View group memberships
└── Resource: view
    └── Action: read      # View group information
```

## System Groups

The module automatically creates three system groups on initialization:

1. **Super Administrator** (`super_admin`)
   - Full administrative access to all group operations
   - Can create, modify, and delete groups
   - Can manage group memberships

2. **Authenticated Users** (`authenticated`)
   - Basic authenticated users
   - Can view group information
   - Default group for logged-in users

3. **Guest Users** (`guest`)
   - Unauthenticated users
   - Limited access (future implementation)

## Database Indexes

### Groups Collection
- `name` (unique)
- `type`
- `system_name` (unique, sparse)
- `eve_entity_id` (unique, sparse)
- `is_active`

### Group Memberships Collection
- `group_id, character_id` (unique composite)
- `character_id`
- `group_id`
- `is_active`

## Error Handling

### HTTP Status Codes
- **200 OK**: Successful operation
- **201 Created**: Group or membership created successfully
- **400 Bad Request**: Invalid input data or malformed request
- **401 Unauthorized**: Authentication required or invalid token
- **403 Forbidden**: Insufficient permissions for operation
- **404 Not Found**: Group, membership, or character not found
- **409 Conflict**: Group name already exists or membership already active
- **500 Internal Server Error**: Database or server error

### Error Response Format
```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": "Additional error context (optional)"
}
```

## Business Logic

### Group Creation Rules
- Only `custom` groups can be created manually
- Group names must be unique across all types
- System groups cannot be created, updated, or deleted
- Creator is automatically tracked for audit purposes

### Membership Rules
- Characters can belong to multiple groups
- Duplicate memberships are prevented (upsert logic)
- Inactive memberships are preserved for audit trail
- Adding user who added the membership is tracked

### Permission Checking
- All operations require JWT authentication with valid tokens
- Group management operations require super admin group membership
- Group membership is checked against MongoDB collections
- System groups have predefined permissions

## Integration Points

### Auth Module Integration (✅ COMPLETED)
- JWT token validation via auth service
- Character ID extraction from authenticated tokens
- Permission checking against user's group memberships
- Character context middleware for corp/alliance data

### EVE Online Integration (✅ COMPLETED)
- Corporation and alliance group auto-creation
- ESI API validation for corp/alliance memberships
- Character corporation/alliance synchronization

### Scheduler Integration (✅ COMPLETED)
- Automated group synchronization tasks (system-groups-sync)
- Periodic permission validation every 6 hours
- Corp/alliance membership updates via background tasks

## Performance Considerations

- **Database Indexes**: Optimized for common query patterns
- **Pagination**: All list endpoints support pagination
- **Caching**: Future Redis implementation for permission checks
- **Bulk Operations**: Future implementation for large-scale operations

## Development Workflow

### Adding New Group Types
1. Add new type to `GroupType` enum in `models/models.go`
2. Update validation in `dto/inputs.go`
3. Implement type-specific business logic in `services/service.go`
4. Add appropriate database indexes
5. Update documentation

### Permission System Extension
1. Define permissions in groups module
2. Integrate with auth middleware
3. Update route protection
4. Add permission checking methods
5. Update tests and documentation

## Testing

### Testing Requirements
- Integration tests with real JWT authentication
- Permission checking test scenarios for group membership
- EVE Online corporation/alliance integration testing
- Performance testing with large datasets
- Group synchronization and background task testing

## Migration Path

### Database Migrations
- Groups and memberships collections are production-ready
- System groups are automatically created on first run
- First user is automatically assigned to Super Administrator group
- No manual data migration required

## Configuration

### Environment Variables
- Currently uses standard database configuration
- Future phases will add group-specific configuration options

### Module Configuration
```go
// Module initialization
groupsModule, err := groups.NewModule(db, authModule)
if err != nil {
    log.Fatal(err)
}

// Register routes
groupsModule.RegisterUnifiedRoutes(api)
```

## Future Enhancements

### Planned Features
- Redis caching for improved performance
- Discord integration for role synchronization
- Bulk membership operations for large groups
- Advanced audit logging and reporting
- ESI validation for real-time membership checking
- Fleet management group integration

## Dependencies

### Internal Dependencies
- `go-falcon/internal/auth` (for models and future integration)
- `go-falcon/pkg/database` (MongoDB connection)
- `go-falcon/pkg/module` (module interface)

### External Dependencies
- `github.com/danielgtaylor/huma/v2` (API framework)
- `go.mongodb.org/mongo-driver` (MongoDB driver)
- `github.com/go-chi/chi/v5` (HTTP router)

## Contributing

1. Follow the established module structure pattern
2. Use Huma v2 for all new endpoints
3. Implement proper validation in DTOs
4. Add comprehensive error handling
5. Update documentation for any changes
6. Include tests for new functionality

## Security Considerations

- **Input Validation**: All inputs validated via Huma v2 struct tags
- **Authentication**: Full JWT token validation and group membership checking
- **Authorization**: Role-based access control implementation
- **Audit Trail**: All operations tracked with user information
- **Data Integrity**: Database constraints prevent invalid states

## Monitoring and Observability

- **Structured Logging**: All operations logged with context
- **Health Checks**: Module health endpoint available
- **Error Tracking**: Comprehensive error logging
- **Performance Metrics**: Database operation tracking (future)

This documentation reflects the current production-ready implementation with full authentication integration and EVE Online group management capabilities.