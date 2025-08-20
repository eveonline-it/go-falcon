# Users Module (internal/users)

## Overview

The users module provides user management functionality.
An user can have multiple characters with different scopes and groups (so permissions)
Each user has his own refresh and access token

## Architecture

### Core Components

- **User Management**: User management
- **Character Management**: List all characters
- **User State Control**: Enable/disable, ban/unban, and validation status management
- **Administrative Tools**: User search, filtering, and bulk operations

### Files Structure

```
internal/users/
├── users.go          # Main module with route registration and core logic
├── handlers.go       # HTTP handlers for user operations
├── models.go         # User data structures and database models
├── service.go        # Business logic for user operations
└── CLAUDE.md         # This documentation
```

## User Data Model

### Database Schema (MongoDB Collection shared with Auth module: `user_profiles`)

```go
type User struct {
    CharacterID   int       `json:"character_id" bson:"character_id"`     // EVE character ID (unique)
    UserID        string    `json:"user_id" bson:"user_id"`               // UUID for internal identification
    AccessToken   string    `json:"-" bson:"access_token"`                // EVE SSO access token (hidden from JSON)
    RefreshToken  string    `json:"-" bson:"refresh_token"`               // EVE SSO refresh token (hidden from JSON)
    Enabled       bool      `json:"enabled" bson:"enabled"`               // User account status
    Banned        bool      `json:"banned" bson:"banned"`                 // Ban status
    Invalid       bool      `json:"invalid" bson:"invalid"`               // Token/account validity
    Scopes        string    `json:"scopes" bson:"scopes"`                 // EVE Online permissions
    Position      int       `json:"position" bson:"position"`             // User position/rank
    Notes         string    `json:"notes" bson:"notes"`                   // Administrative notes
    CreatedAt     time.Time `json:"created_at" bson:"created_at"`         // Registration timestamp
    UpdatedAt     time.Time `json:"updated_at" bson:"updated_at"`         // Last update timestamp
    LastLogin     time.Time `json:"last_login" bson:"last_login"`         // Last login timestamp
}
```

### Field Descriptions

- **character_id**: Primary identifier from EVE Online (unique index)
- **user_id**: Internal UUID for system operations and references
- **accessToken**: Current EVE SSO access token (encrypted/secure storage)
- **refreshToken**: EVE SSO refresh token for token renewal
- **enabled**: Controls user access (true = can access, false = disabled)
- **banned**: Administrative ban status (true = banned, false = not banned)
- **invalid**: Token/account validity flag (true = invalid tokens/data)
- **scopes**: EVE Online permissions granted during SSO
- **position**: Numerical position for ranking/hierarchy (0 = default)
- **notes**: Free-form administrative notes for user management

## API Endpoints

### Public Endpoints

#### Get User Statistics
```
GET /api/users/stats
```
Returns aggregate statistics about users in the system.

**Response:**
```json
{
  "total_users": 1250,
  "enabled_users": 1180,
  "disabled_users": 70,
  "banned_users": 15,
  "invalid_users": 25
}
```

### Administrative Endpoints

All administrative endpoints require JWT authentication and appropriate permissions.

#### List Users
```
GET /api/users?page=1&page_size=20&query=search&enabled=true&banned=false
```
**Authentication:** Required  
**Permission:** Authentication required

**Query Parameters:**
- `page`: Page number (default: 1)
- `page_size`: Items per page (default: 20, max: 100)
- `query`: Search by character name or ID
- `enabled`: Filter by enabled status (true/false)
- `banned`: Filter by banned status (true/false)
- `invalid`: Filter by invalid status (true/false)
- `position`: Filter by position value
- `sort_by`: Sort field (character_name, created_at, last_login, position)
- `sort_order`: Sort order (asc, desc)

**Response:**
```json
{
  "users": [...],
  "total": 1250,
  "page": 1,
  "page_size": 20,
  "total_pages": 63
}
```

#### Get User Details
```
GET /api/users/mgt/{character_id}
```
**Authentication:** Required  
**Permission:** Authentication required

**Response:** Complete user object with all fields.

#### Update User
```
PUT /api/users/mgt/{character_id}
```
**Authentication:** Required  
**Permission:** Authentication required

**Request Body:**
```json
{
  "enabled": true,
  "banned": false,
  "invalid": false,
  "position": 5,
  "notes": "Administrative notes"
}
```

**Response:**
```json
{
  "success": true,
  "message": "User updated successfully",
  "user": {...}
}
```


### User Management Endpoints

#### List User Characters
```
GET /api/users/{user_id}/characters
```
**Authentication:** Required  
**Permission:** Self-access or Authentication required

Users can always view their own characters. Admin users (super_admin) can view any user's characters.

**Response:**
```json
{
  "user_id": "uuid-string",
  "characters": [
    {
      "character_id": 123456,
      "character_name": "Character Name",
      "user_id": "uuid-string",
      "enabled": true,
      "banned": false,
      "position": 0,
      "last_login": "2024-01-01T12:00:00Z"
    }
  ],
  "count": 1
}
```

## Authentication and Authorization

The Users module integrates with the authentication and authorization system using JWT tokens and group-based permissions.

### Required Permissions

The following permissions should be configured in the Groups module:

#### Service: `users`

##### Resource: `profiles`
- **read**: View user information, list users, get user details
- **write**: Update user status, modify user settings

### Permission Requirements by Endpoint

| Endpoint | Method | Authentication | Permission Required | Description |
|----------|--------|---------------|-------------------|-------------|
| `/api/users/stats` | GET | No | Public | Get user statistics |
| `/api/users` | GET | Yes | Authentication required | List and search users |
| `/api/users/mgt/{character_id}` | GET | Yes | Authentication required | Get specific user details |
| `/api/users/mgt/{character_id}` | PUT | Yes | Authentication required | Update user status and settings |
| `/api/users/{user_id}/characters` | GET | Yes | Self or Authentication required | List characters for a user |

### Authorization Logic

#### Self-Access vs Admin Access
- **Character Lists**: Users can always view their own characters (when `user_id` matches authenticated user's `user_id`)
- **Admin Override**: Users with super_admin permission can view any user's characters
- **Status Updates**: Only users with super_admin permission can modify user status

#### Permission Model

The users module now uses a simplified permission model:

- **super_admin**: Full administrative access to all user management functions
- **authenticated**: Access to own character information and profile data
- **public**: Access to public user statistics only

### Integration with Auth Module

- **JWT Middleware**: All protected endpoints require valid JWT tokens
- **Context Access**: Authenticated user information via `auth.GetAuthenticatedUser(r)`
- **Token Types**: Supports both cookie-based (web) and Bearer token (mobile) authentication

### Security Features

- **Data Privacy**: User tokens (access/refresh) are never exposed in JSON responses
- **Admin Actions**: All administrative actions are logged with operator information  
- **Permission Checking**: Fine-grained permission checks prevent unauthorized access
- **Self-Service**: Users can view their own character information without admin permissions
- **Audit Trail**: User status changes include timestamp and administrative notes

## Error Handling

### Common HTTP Status Codes

- **200 OK**: Successful operation
- **400 Bad Request**: Invalid request parameters or malformed JSON
- **401 Unauthorized**: Missing or invalid authentication token
- **403 Forbidden**: Insufficient permissions for requested operation  
- **404 Not Found**: User not found
- **500 Internal Server Error**: Database or server error

### Error Response Format

```json
{
  "error": "permission_denied",
  "message": "Insufficient permissions for this operation"
}
```

## Database Integration

The Users module shares the `user_profiles` MongoDB collection with the Auth module, providing a unified user management system while maintaining separation of concerns.

### Indexes

Recommended database indexes for optimal performance:
- `character_id` (unique)
- `user_id`  
- `enabled`
- `banned`
- `character_name` (text index for search)
- `created_at`
- `last_login`

