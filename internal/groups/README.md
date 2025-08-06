# Groups Module Implementation

This document provides implementation details and API reference for the groups module.

## Overview

The groups module provides a comprehensive permission-based authorization system with the following features:

- **Default Groups**: Predefined groups for common access levels (Guest, Full, Corporate, Administrators, Super Admin)
- **Custom Groups**: Administrator-managed groups for specific use cases
- **Resource-Level Permissions**: Granular read/write/delete/admin access control
- **Discord Integration**: Automatic role assignment and synchronization (when Discord service is available)
- **Automated Validation**: Scheduled tasks to maintain group membership integrity
- **EVE Corporation/Alliance Integration**: Dynamic group assignment based on corporation and alliance membership

## API Endpoints

### Group Management

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/groups` | GET | Optional | List all groups with user's membership status |
| `/api/groups` | POST | Admin | Create a new custom group |
| `/api/groups/{id}` | PUT | Admin | Update an existing group |
| `/api/groups/{id}` | DELETE | Admin | Delete a custom group |

### Membership Management

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/groups/{id}/members` | GET | Admin | List all members of a group |
| `/api/groups/{id}/members` | POST | Admin | Add a member to a group |
| `/api/groups/{id}/members/{characterID}` | DELETE | Admin | Remove a member from a group |

### Permission Queries

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/permissions/check` | GET | Optional | Check if user has specific permission |
| `/api/permissions/user` | GET | Optional | Get user's complete permission matrix |

## Request/Response Examples

### Create Group

```bash
POST /api/groups
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "beta-testers",
  "description": "Beta testing group with special access",
  "permissions": {
    "user": ["read", "write"],
    "beta": ["read", "write", "admin"]
  },
  "discord_roles": [
    {
      "server_id": "123456789",
      "server_name": "Main Server",
      "role_name": "Beta Tester"
    }
  ],
  "auto_assignment_rules": {
    "corporation_ids": [98000001, 98000002],
    "min_security_status": 0.5
  }
}
```

Response:
```json
{
  "id": "507f1f77bcf86cd799439011",
  "name": "beta-testers",
  "description": "Beta testing group with special access",
  "is_default": false,
  "permissions": {
    "user": ["read", "write"],
    "beta": ["read", "write", "admin"]
  },
  "discord_roles": [
    {
      "server_id": "123456789",
      "server_name": "Main Server",
      "role_name": "Beta Tester"
    }
  ],
  "auto_assignment_rules": {
    "corporation_ids": [98000001, 98000002],
    "min_security_status": 0.5
  },
  "created_at": "2024-01-01T10:00:00Z",
  "updated_at": "2024-01-01T10:00:00Z",
  "created_by": 2112625428
}
```

### Check Permission

```bash
GET /api/permissions/check?resource=user&action=write
Authorization: Bearer <jwt_token>
```

Response:
```json
{
  "allowed": true,
  "groups": ["full", "beta-testers"]
}
```

### Get User Permissions

```bash
GET /api/permissions/user
Authorization: Bearer <jwt_token>
```

Response:
```json
{
  "character_id": 2112625428,
  "groups": ["full", "beta-testers"],
  "permissions": {
    "public": {
      "read": true
    },
    "user": {
      "read": true,
      "write": true
    },
    "profile": {
      "read": true,
      "write": true
    },
    "beta": {
      "read": true,
      "write": true,
      "admin": true
    }
  },
  "is_guest": false
}
```

## Default Groups

### Guest
- **Purpose**: Unauthenticated users and limited access
- **Permissions**: Read-only access to public resources
- **Assignment**: Automatic for unauthenticated requests
- **Discord Role**: None

### Full
- **Purpose**: Authenticated EVE Online characters with standard access
- **Permissions**: Read/write access to user-specific resources
- **Assignment**: Automatic upon successful EVE SSO authentication
- **Discord Roles**: Configurable across multiple servers

### Corporate
- **Purpose**: Members of enabled EVE Online corporations or alliances
- **Permissions**: Access to corporation/alliance-specific resources
- **Assignment**: Automatic when character belongs to enabled corporation or alliance
- **Discord Roles**: Configurable across multiple servers
- **Validation**: Checked via scheduled tasks using ESI

### Administrators
- **Purpose**: Application administrators with elevated privileges
- **Permissions**: Full system access, user management, group creation
- **Assignment**: Manual assignment by existing administrators
- **Discord Roles**: Configurable administrative roles across servers

### Super Admin
- **Purpose**: Ultimate system authority with unrestricted access
- **Permissions**: All system permissions (wildcard access)
- **Assignment**: Configured via `SUPER_ADMIN_CHARACTER_ID` environment variable
- **Discord Roles**: Configurable super admin roles across all servers

## Middleware Usage

### Require Permission

```go
// Require specific permission
r.With(groupsModule.RequirePermission("user", "write")).Post("/endpoint", handler)

// Require admin access to groups
r.With(groupsModule.RequirePermission("groups", "admin")).Post("/groups", handler)
```

### Require Group Membership

```go
// Require specific group
r.With(groupsModule.RequireGroup("administrators")).Get("/admin", handler)

// Require any of multiple groups
r.With(groupsModule.RequireAnyGroup("administrators", "moderators")).Get("/manage", handler)
```

### Combined EVE Scopes and Group Permissions

```go
// Require both EVE scopes and group permissions
r.With(groupsModule.RequireEVEScopesAndPermissions(
    "corporation", "read", 
    "esi-corporations.read_structures.v1")).Get("/structures", handler)
```

### Resource Owner or Permission

```go
// Allow if user owns resource OR has permission
ownerExtractor := func(r *http.Request) int {
    userIDStr := chi.URLParam(r, "userID")
    userID, _ := strconv.Atoi(userIDStr)
    return userID
}

r.With(groupsModule.ResourceOwnerOrPermission(
    ownerExtractor, "user", "admin")).Put("/users/{userID}", handler)
```

## Environment Configuration

```bash
# Required
SUPER_ADMIN_CHARACTER_ID=123456789  # EVE character ID for super admin

# Optional
GROUPS_CACHE_TTL=300               # Permission cache TTL in seconds (default: 300)
GROUPS_VALIDATION_INTERVAL=3600    # Membership validation interval in seconds (default: 3600)
DISCORD_ROLE_SYNC=true            # Enable Discord role synchronization (default: true)
DISCORD_SERVICE_URL=http://localhost:8080  # Discord service endpoint

# Corporation/Alliance Integration
ENABLED_CORPORATION_IDS=98000001,98000002,98000003  # Comma-separated corp IDs
ENABLED_ALLIANCE_IDS=99000001,99000002              # Comma-separated alliance IDs
```

## Database Schema

### groups Collection

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

### group_memberships Collection

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

## Integration with Other Modules

### Auth Module Integration

The groups module integrates with the auth module through:

1. **Automatic Group Assignment**: When users authenticate via EVE SSO, they are automatically assigned to appropriate default groups
2. **JWT Token Enhancement**: User permissions can be included in JWT tokens
3. **Profile Integration**: Group memberships are included in user profiles

```go
// Example: Assign user to default groups after authentication
err := groupsModule.AssignUserToDefaultGroups(ctx, characterID, corporationID, allianceID)
```

### Scheduler Module Integration

The groups module provides scheduled tasks for:

1. **Corporate Membership Validation**: Hourly validation against ESI data
2. **Expired Membership Cleanup**: Daily cleanup of expired memberships
3. **Discord Role Synchronization**: Regular sync with Discord service

```go
// Example scheduled tasks
"system-groups-corporate-validation": {
    Schedule: "0 */1 * * *", // Every hour
    Type:     "system",
    Function: groupTask.ValidateCorporateMemberships,
}

"system-groups-cleanup-expired": {
    Schedule: "0 3 * * *", // Daily at 3 AM
    Type:     "system", 
    Function: groupTask.CleanupExpiredMemberships,
}

"system-groups-discord-sync": {
    Schedule: "*/30 * * * *", // Every 30 minutes
    Type:     "system",
    Function: groupTask.SyncDiscordRoles,
}
```

## Performance Considerations

### Caching Strategy

1. **Permission Matrix**: Cached in Redis for 5 minutes
2. **Group Memberships**: Cached per user session
3. **Discord Roles**: Retrieved from service and cached for 15 minutes
4. **ESI Validation**: Corporation and alliance validation results cached for 1 hour

### Database Optimization

1. **Indexes**: Composite indexes on character_id + group_id for fast membership lookups
2. **Aggregation**: Efficient permission queries using MongoDB aggregation pipelines
3. **Connection Pooling**: Shared database connections across modules

## Security Features

### Access Control
- **Hierarchical Permissions**: Groups build upon each other's permissions
- **Resource Scoping**: Permissions limited to specific resource types
- **Audit Logging**: All group changes logged with actor and timestamp
- **Rate Limiting**: API endpoints protected against abuse

### Validation Mechanisms
- **Corporation/Alliance Membership**: ESI validation of corporate group eligibility
- **Token Verification**: JWT token validation for all operations
- **Permission Caching**: Redis-based caching with automatic invalidation
- **Expiration Handling**: Automatic removal of expired memberships

## Error Handling

### Common Error Scenarios
- **Permission Denied**: User lacks required group membership
- **Group Not Found**: Invalid group ID or deleted group
- **Membership Conflict**: User already member of conflicting group
- **Discord Service Failure**: Discord service unavailable
- **ESI Validation Failure**: EVE Online API unavailable

### Error Response Format
```json
{
  "error": "permission_denied",
  "message": "Insufficient permissions for this operation",
  "required_groups": ["administrators"],
  "user_groups": ["full"]
}
```

## Future Enhancements

### Planned Features
- **Group Hierarchies**: Parent-child group relationships for inheritance
- **Conditional Permissions**: Time-based or location-based permissions
- **Approval Workflows**: Membership requests requiring approval
- **Advanced Analytics**: Group usage and permission analytics

### API Extensions
- **Bulk Operations**: Batch membership changes for efficiency
- **Group Templates**: Predefined group configurations
- **Permission Inheritance**: Complex permission inheritance rules
- **Advanced Discord Integration**: Full Discord bot integration with real-time sync