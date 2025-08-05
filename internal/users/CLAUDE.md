# Users Module (internal/users)

## Overview

The users module provides user management functionality integrated with EVE Online SSO authentication. It handles user registration, token management, and user state tracking with comprehensive administrative controls for user lifecycle management.

## Architecture

### Core Components

- **User Management**: Complete user lifecycle from registration to deactivation
- **SSO Integration**: Seamless integration with EVE Online authentication flow
- **Token Management**: Access and refresh token storage and updates
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

### Database Schema (MongoDB Collection: `users`)

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

## SSO Integration Flow

### User Registration/Login Process

1. **EVE SSO Authentication**: User completes EVE Online SSO flow
2. **Character ID Check**: System checks if `character_id` exists in database
3. **User Update**: If user exists, update tokens and login timestamp
4. **User Creation**: If new user, create complete user record
5. **Status Validation**: Check enabled/banned status before granting access

### Database Operations

#### Existing User (Update Flow)
```go
// Update existing user tokens and login time
filter := bson.M{"character_id": characterID}
update := bson.M{
    "$set": bson.M{
        "access_token":  newAccessToken,
        "refresh_token": newRefreshToken,
        "last_login":    time.Now(),
        "updated_at":    time.Now(),
        "scopes":        updatedScopes,
    },
}
```

#### New User (Insert Flow)
```go
// Create new user document
newUser := User{
    CharacterID:  characterID,
    UserID:       uuid.New().String(),
    AccessToken:  accessToken,
    RefreshToken: refreshToken,
    Enabled:      true,          // Default enabled
    Banned:       false,         // Default not banned
    Invalid:      false,         // Default valid
    Scopes:       ssoScopes,
    Position:     0,             // Default position
    Notes:        "",            // Empty notes
    CreatedAt:    now,
    UpdatedAt:    now,
    LastLogin:    now,
}
```

## API Endpoints

### User Management Endpoints

| Endpoint | Method | Description | Auth Required | Admin Only |
|----------|--------|-------------|---------------|------------|
| `/users` | GET | List all users with filtering | ✅ | ✅ |
| `/users/{userID}` | GET | Get specific user details | ✅ | ✅ |
| `/users/{userID}` | PUT | Update user information | ✅ | ✅ |
| `/users/{userID}/enable` | POST | Enable user account | ✅ | ✅ |
| `/users/{userID}/disable` | POST | Disable user account | ✅ | ✅ |
| `/users/{userID}/ban` | POST | Ban user account | ✅ | ✅ |
| `/users/{userID}/unban` | POST | Unban user account | ✅ | ✅ |
| `/users/{userID}/invalidate` | POST | Mark user tokens as invalid | ✅ | ✅ |
| `/users/{userID}/validate` | POST | Mark user as valid | ✅ | ✅ |
| `/users/{userID}/notes` | PUT | Update administrative notes | ✅ | ✅ |
| `/users/search` | GET | Search users by criteria | ✅ | ✅ |
| `/users/stats` | GET | User statistics and counts | ✅ | ✅ |

### Public Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/profile` | GET | Get current user's profile | ✅ |
| `/profile` | PUT | Update current user's profile | ✅ |
| `/profile/tokens/refresh` | POST | Refresh user's EVE tokens | ✅ |

## User Status Management

### User States

1. **Active User**: `enabled: true, banned: false, invalid: false`
2. **Disabled User**: `enabled: false` (temporary suspension)
3. **Banned User**: `banned: true` (permanent ban)
4. **Invalid User**: `invalid: true` (token/data issues)

### State Transitions

```go
// Enable user
func (s *Service) EnableUser(userID string) error {
    return s.updateUserStatus(userID, bson.M{
        "enabled": true,
        "updated_at": time.Now(),
    })
}

// Ban user
func (s *Service) BanUser(userID string, reason string) error {
    return s.updateUserStatus(userID, bson.M{
        "banned": true,
        "enabled": false,
        "notes": fmt.Sprintf("BANNED: %s", reason),
        "updated_at": time.Now(),
    })
}
```

## Administrative Features

### User Search and Filtering

```go
// Search parameters
type UserSearchParams struct {
    CharacterName string `json:"character_name"`
    Enabled       *bool  `json:"enabled"`
    Banned        *bool  `json:"banned"`
    Invalid       *bool  `json:"invalid"`
    MinPosition   *int   `json:"min_position"`
    MaxPosition   *int   `json:"max_position"`
    CreatedAfter  *time.Time `json:"created_after"`
    CreatedBefore *time.Time `json:"created_before"`
    HasScopes     []string `json:"has_scopes"`
}
```

### Bulk Operations

- **Bulk Enable/Disable**: Mass user account control
- **Bulk Token Refresh**: Refresh tokens for multiple users
- **Bulk Status Updates**: Update user positions or notes
- **Export Functions**: User data export for analysis

### User Statistics

```go
type UserStats struct {
    TotalUsers    int `json:"total_users"`
    ActiveUsers   int `json:"active_users"`
    DisabledUsers int `json:"disabled_users"`
    BannedUsers   int `json:"banned_users"`
    InvalidUsers  int `json:"invalid_users"`
    RecentLogins  int `json:"recent_logins_24h"`
}
```

## Security Features

### Token Security

- **Encrypted Storage**: Access and refresh tokens encrypted at rest
- **Secure Transmission**: HTTPS-only for all token operations
- **Token Rotation**: Automatic refresh token updates
- **Token Validation**: Regular token validity checks

### Access Control

- **Admin-Only Endpoints**: User management restricted to administrators
- **User Self-Service**: Users can manage their own profiles
- **Permission Scopes**: EVE Online scope validation
- **Audit Logging**: All administrative actions logged

### Data Protection

- **PII Handling**: Secure handling of character information
- **Token Isolation**: Tokens never exposed in API responses
- **Database Security**: Proper indexing and access controls
- **Backup Security**: Encrypted backups with token exclusion

## Integration Points

### Authentication Module

```go
// Called during EVE SSO callback
func (s *Service) ProcessSSOLogin(characterID int, tokens *SSOTokens) (*User, error) {
    existingUser, err := s.GetUserByCharacterID(characterID)
    if err == nil {
        // Update existing user
        return s.UpdateUserTokens(existingUser.UserID, tokens)
    }
    
    // Create new user
    return s.CreateUser(characterID, tokens)
}
```

### Profile Module Integration

- **Profile Enrichment**: Add user status to profile data
- **Token Management**: Coordinate token refresh across modules
- **Status Checks**: Validate user status before profile operations

## Background Tasks

### Token Maintenance

- **Token Refresh**: Periodic refresh of expiring tokens
- **Token Validation**: Regular validation of stored tokens
- **Cleanup**: Remove invalid or expired tokens

### User Analytics

- **Login Tracking**: Monitor user login patterns
- **Usage Statistics**: Track user activity and engagement
- **Health Monitoring**: Monitor user account health

### Maintenance Tasks

- **Data Cleanup**: Remove old or unused user data
- **Status Auditing**: Regular user status validation
- **Performance Optimization**: Database index maintenance

## Configuration

### Required Environment Variables

```bash
# User management settings
USER_DEFAULT_ENABLED=true
USER_AUTO_BAN_INVALID_TOKENS=false
USER_TOKEN_REFRESH_INTERVAL=3600
USER_CLEANUP_INTERVAL=86400

# Administrative settings  
ADMIN_USERS=character_id_1,character_id_2
ADMIN_NOTIFICATIONS_ENABLED=true
```

### Database Configuration

```go
// MongoDB indexes for performance
db.users.createIndex({ "character_id": 1 }, { unique: true })
db.users.createIndex({ "user_id": 1 }, { unique: true })
db.users.createIndex({ "enabled": 1, "banned": 1 })
db.users.createIndex({ "created_at": 1 })
db.users.createIndex({ "last_login": 1 })
```

## Error Handling

### Common Error Scenarios

- **Duplicate Character ID**: Handle EVE character already registered
- **Invalid Tokens**: Process expired or revoked EVE tokens
- **Database Errors**: Graceful handling of connection issues
- **Permission Errors**: Proper access control validation

### Error Responses

```json
{
  "error": "user_not_found",
  "message": "User with character ID 123456789 not found",
  "code": 404
}
```

## Performance Considerations

### Database Optimization

- **Proper Indexing**: Optimized queries for user lookups
- **Connection Pooling**: Efficient database connections
- **Query Optimization**: Minimize database round trips
- **Data Pagination**: Large user lists with pagination

### Caching Strategy

- **User Cache**: Frequently accessed user data
- **Token Cache**: Short-term token caching
- **Statistics Cache**: User statistics caching
- **Search Cache**: Common search results

## Monitoring and Observability

### Metrics

- **User Registration Rate**: New user creation tracking
- **Login Frequency**: User activity monitoring
- **Token Refresh Rate**: Token management health
- **Administrative Actions**: Admin operation tracking

### Logging

- **User Operations**: All user CRUD operations
- **Status Changes**: Enable/disable/ban operations
- **Token Operations**: Token refresh and validation
- **Administrative Actions**: Admin user management

### Alerts

- **Failed Token Refresh**: Alert on token refresh failures
- **Suspicious Activity**: Unusual user activity patterns
- **Database Issues**: User data access problems
- **Administrative Notifications**: Important user status changes

## Best Practices

### User Management

- **Regular Audits**: Periodic review of user accounts
- **Token Hygiene**: Regular token validation and cleanup
- **Status Monitoring**: Monitor user account health
- **Documentation**: Keep administrative notes updated

### Security

- **Principle of Least Privilege**: Minimal required permissions
- **Regular Security Reviews**: Audit user access patterns
- **Token Security**: Secure token storage and transmission
- **Admin Access Control**: Strict admin permission management

### Performance

- **Database Optimization**: Regular index and query optimization
- **Caching Strategy**: Implement appropriate caching
- **Bulk Operations**: Efficient mass user operations
- **Resource Management**: Monitor resource usage patterns