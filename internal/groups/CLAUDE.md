# Groups Module (internal/groups)

## Overview

The groups module provides group and role-based access control management for the go-falcon EVE Online API gateway. This module implements a hierarchical permission system that supports EVE-specific groups (characters, corporations, alliances) as well as custom groups for fine-grained access control.

**Current Status**: Phase 1 - Foundation completed
**Authentication**: Simple auth middleware for testing (Phase 1 only)

## Architecture

### Core Components

- **Group Management**: CRUD operations for custom groups and system groups
- **Membership Management**: Character assignment to groups with audit trail
- **Permission System**: Hierarchical permission checking (future phases)
- **System Groups**: Built-in groups (super_admin, authenticated, guest)
- **MongoDB Storage**: Groups and memberships collections with proper indexing

### Files Structure

```
internal/groups/
├── dto/
│   ├── inputs.go         # Request input DTOs with Huma v2 validation
│   └── outputs.go        # Response output DTOs
├── middleware/
│   └── auth.go          # Authentication and authorization middleware
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
- **`corporation`**: EVE Corporation groups (future phases)
- **`alliance`**: EVE Alliance groups (future phases)
- **`custom`**: User-created custom groups

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

### Phase 1 Implementation (Current)

**Simple Auth Middleware**: For testing and development purposes, the module currently uses a dummy authentication system that returns a hardcoded super admin user.

```go
// Dummy user for Phase 1 testing
User{
    UserID:        "00000000-0000-0000-0000-000000000000",
    CharacterID:   99999999,
    CharacterName: "Test SuperAdmin",
    Scopes:        "publicData",
}
```

### Future Phase Implementation

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

### Permission Checking (Phase 1)
- All operations currently require dummy "super admin" authentication
- Future phases will implement granular permission checking
- System groups have predefined permissions

## Integration Points

### Auth Module Integration (Future)
- JWT token validation
- Character ID extraction from authenticated user
- Permission checking against user's groups

### EVE Online Integration (Future Phases)
- Corporation and alliance group auto-creation
- ESI API validation for corp/alliance memberships
- Character corporation/alliance synchronization

### Scheduler Integration (Future)
- Automated group synchronization tasks
- Periodic permission cache refresh
- Corp/alliance membership updates

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

### Phase 1 Testing
- Use dummy authentication for testing group operations
- Test group CRUD operations with mock data
- Verify database indexes and constraints
- Test pagination and filtering

### Future Testing Requirements
- Integration tests with real authentication
- Permission checking test scenarios
- EVE Online integration testing
- Performance testing with large datasets

## Migration Path

### From Phase 1 to Phase 2
1. Replace dummy authentication with real auth integration
2. Implement actual permission checking logic
3. Add EVE corporation/alliance group support
4. Integrate with scheduler for automated tasks
5. Add Redis caching for performance

### Database Migrations
- Groups and memberships collections are ready for production
- System groups are automatically created on first run
- No data migration required for Phase 2

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

## Known Limitations (Phase 1)

1. **Authentication**: Uses dummy authentication for testing
2. **Permissions**: All operations treated as admin operations
3. **EVE Integration**: No corporation/alliance groups yet
4. **Caching**: No Redis caching implemented
5. **Background Tasks**: No automated synchronization

## Roadmap

### Phase 2: EVE Integration
- Real authentication integration
- Corporation and alliance groups
- Auto-assignment rules
- ESI membership validation

### Phase 3: Advanced Features
- Custom role-based groups
- Discord integration
- Advanced permission model
- Performance optimizations

### Phase 4: Production Hardening
- Full caching implementation
- Bulk operations
- Migration tools
- Comprehensive testing

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
- **Authentication**: Currently using dummy auth (Phase 1 only)
- **Authorization**: Role-based access control implementation
- **Audit Trail**: All operations tracked with user information
- **Data Integrity**: Database constraints prevent invalid states

## Monitoring and Observability

- **Structured Logging**: All operations logged with context
- **Health Checks**: Module health endpoint available
- **Error Tracking**: Comprehensive error logging
- **Performance Metrics**: Database operation tracking (future)

This documentation reflects the current Phase 1 implementation and serves as the foundation for future development phases.