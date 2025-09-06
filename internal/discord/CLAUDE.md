# Discord Module (internal/discord)

## Overview

The Discord module provides comprehensive Discord bot integration for Go-Falcon, enabling Discord OAuth authentication and automatic role synchronization based on group memberships. It allows users to authenticate with Discord and automatically manages Discord roles based on their Go Falcon group assignments.

**Status**: Production Ready - Complete API Implementation  
**Latest Update**: Migrated to centralized authentication with DiscordAdapter, implemented all missing role-mapping endpoints
**Authentication**: Dual authentication support (Bearer tokens + cookies) for seamless frontend integration

## Architecture

### Core Components

- **Routes**: HTTP route handlers with DiscordAdapter integration for centralized authentication
- **Service Layer**: Main business logic coordinating OAuth, bot management, and synchronization
- **Repository**: Database operations for Discord user data and guild configurations
- **DiscordAdapter**: Centralized authentication adapter supporting dual auth (Bearer + Cookie)
- **OAuth Service**: Discord OAuth2 authentication flow implementation
- **Bot Service**: Discord Bot API client for role management operations
- **Sync Service**: Role synchronization engine coordinating groups and Discord roles
- **Scheduled Tasks**: Automated token refresh and role synchronization

### Files Structure

```
internal/discord/
├── dto/
│   ├── inputs.go        # Request DTOs for all Discord endpoints
│   └── outputs.go       # Response DTOs for all Discord operations
├── models/
│   └── models.go        # Database models and schemas
├── routes/
│   └── routes.go        # API route definitions and handlers
├── services/
│   ├── repository.go    # Database operations layer
│   ├── oauth_service.go # Discord OAuth2 implementation
│   ├── bot_service.go   # Discord Bot API client
│   ├── sync_service.go  # Role synchronization engine
│   └── service.go       # Main service layer coordinating all operations
├── module.go            # Module initialization and registration
└── CLAUDE.md           # This documentation file
```

## Recent Architecture Improvements

### Centralized Authentication (2025-01-06)
The Discord module has been migrated to use the centralized authentication system:

#### DiscordAdapter Integration
- **Unified Authentication**: Uses `DiscordAdapter` from `pkg/middleware` for consistent auth patterns
- **Dual Authentication Support**: Supports both Bearer tokens and cookie authentication seamlessly
- **Permission Integration**: Integrates with centralized permission middleware for consistent access control
- **Code Reduction**: Eliminated 200+ lines of duplicated authentication code

#### API Implementation Completion
- **All Endpoints Implemented**: Previously documented role-mapping endpoints are now fully functional
- **Complete CRUD Operations**: Full Create, Read, Update, Delete support for all Discord resources
- **Frontend Ready**: Cookie authentication enables seamless frontend integration without CORS issues

#### Module Structure Updates
```go
// Before: Old module pattern with separate middleware
type Module struct {
    service *services.Service
    middleware *middleware.Interface  // Module-specific middleware
}

// After: New pattern with centralized authentication
type Module struct {
    service        *services.Service
    routes         *routes.Routes     // Integrated route handlers
    discordAdapter *middleware.DiscordAdapter  // Centralized auth adapter
}
```

### Migration Benefits
- ✅ **Frontend Compatibility**: Cookie auth enables seamless SPA integration
- ✅ **API Consistency**: Same authentication patterns across all modules  
- ✅ **Reduced Complexity**: Single source of truth for authentication logic
- ✅ **Better Testing**: Centralized auth logic is easier to test and maintain
- ✅ **Complete Implementation**: All documented endpoints are now functional

## Features

### Discord OAuth Authentication

Complete OAuth2 flow implementation for Discord login:

#### Authentication Flow
1. **Initiate OAuth**: User clicks Discord login, gets redirected to Discord with state validation
2. **Authorization**: Discord redirects back with authorization code
3. **Token Exchange**: System exchanges code for access and refresh tokens
4. **User Profile**: Fetch Discord user profile and guild memberships
5. **Account Linking**: Link Discord account to Go Falcon user or create new user
6. **Session Management**: Establish authenticated session with JWT tokens

#### Security Features
- **State Validation**: Cryptographically secure state parameter prevents CSRF attacks
- **Token Management**: Automatic refresh token handling and validation
- **Scope Management**: Configurable OAuth scopes (identify, guilds, etc.)
- **Rate Limiting**: Built-in Discord API rate limiting compliance

### Discord Bot Management

Comprehensive Discord bot functionality for role management:

#### Bot Capabilities
- **Role Assignment**: Add and remove roles from guild members
- **Permission Validation**: Verify bot permissions before operations
- **Guild Management**: Manage multiple Discord guilds simultaneously
- **Member Lookup**: Find guild members by Discord user ID
- **Health Monitoring**: Bot connectivity and permission health checks

#### Rate Limiting & Compliance
- **Discord API Limits**: Automatic rate limit detection and backoff
- **Batch Processing**: Efficient bulk role operations
- **Error Recovery**: Automatic retry for transient failures
- **Audit Logging**: Complete audit trail of bot operations

### Role Synchronization Engine

Sophisticated role synchronization system:

#### Synchronization Process
1. **Group Membership Query**: Fetch user's Go Falcon group memberships
2. **Role Mapping Resolution**: Determine required Discord roles based on group mappings
3. **Current State Analysis**: Compare current Discord roles with required roles
4. **Role Operations**: Execute add/remove operations as needed
5. **Audit Logging**: Record all role changes with detailed metadata

#### Sync Features
- **Batch Processing**: Process multiple users efficiently
- **Conflict Resolution**: Handle role conflicts and edge cases
- **Permission Aggregation**: Support multiple character permissions per user
- **Selective Sync**: Sync specific users, guilds, or role mappings
- **Dry Run Mode**: Preview changes before execution

### Automatic Guild Joining

**NEW FEATURE**: Automatic Discord server joining based on group role mappings.

#### Auto-Join Process
1. **Trigger Events**: Auto-join occurs during Discord OAuth authentication or account linking
2. **Group Analysis**: System fetches user's Go Falcon group memberships
3. **Role Mapping Resolution**: Determines Discord guilds where user has role mappings
4. **Guild Addition**: Automatically adds user to mapped guilds with appropriate roles
5. **Sync Integration**: Role synchronization also handles guild membership during periodic syncs

#### Auto-Join Features
- **OAuth Integration**: Seamless auto-join during Discord authentication flow
- **Role Assignment**: Users get appropriate Discord roles upon joining
- **Graceful Failure**: Auto-join failures don't prevent successful authentication
- **Sync Recovery**: Periodic role sync will attempt to add users who should be in guilds
- **Comprehensive Logging**: Detailed audit trail of all auto-join operations
- **Dry Run Support**: Preview auto-join operations during sync dry runs

#### Technical Implementation
- Uses Discord API `PUT /guilds/{guild.id}/members/{user.id}` endpoint
- Requires user's OAuth `access_token` with `guilds.join` scope
- Leverages existing role mapping configuration - no additional setup needed
- Respects guild configuration (disabled guilds are skipped)
- Handles Discord API rate limits and error responses

## API Endpoints

### Authentication Support
All authenticated endpoints support **dual authentication**:
- **Bearer Token**: `Authorization: Bearer your-jwt-token` (for API clients)  
- **Cookie**: `Cookie: falcon_auth_token=your-token` (for frontend applications)

### Available Endpoints

| Endpoint | Method | Description | Authentication |
|----------|--------|-------------|---------------|
| `/discord/status` | GET | Get Discord module status | None (public) |
| `/discord/auth/login` | GET | Get Discord OAuth authorization URL | None (public) |
| `/discord/auth/callback` | GET | Handle Discord OAuth callback | None (public) |
| `/discord/auth/link` | POST | Link Discord account to current user | Bearer/Cookie |
| `/discord/auth/unlink/{discord_id}` | DELETE | Unlink Discord account from user | Bearer/Cookie |
| `/discord/auth/status` | GET | Get Discord authentication status | None (enhanced with auth) |
| `/discord/users` | GET | List Discord users with pagination | Bearer/Cookie |
| `/discord/users/{user_id}` | GET | Get Discord user information | Bearer/Cookie |
| `/discord/guilds` | GET | List configured Discord guilds | Bearer/Cookie |
| `/discord/guilds` | POST | Create Discord guild configuration | Bearer/Cookie |
| `/discord/guilds/{guild_id}` | GET | Get guild configuration details | Bearer/Cookie |
| `/discord/guilds/{guild_id}` | PUT | Update guild configuration | Bearer/Cookie |
| `/discord/guilds/{guild_id}` | DELETE | Remove guild configuration | Bearer/Cookie |
| `/discord/guilds/{guild_id}/role-mappings` | GET | List role mappings for guild | Bearer/Cookie |
| `/discord/guilds/{guild_id}/role-mappings` | POST | Create new role mapping | Bearer/Cookie |
| `/discord/role-mappings/{mapping_id}` | GET | Get role mapping details | Bearer/Cookie |
| `/discord/role-mappings/{mapping_id}` | PUT | Update role mapping | Bearer/Cookie |
| `/discord/role-mappings/{mapping_id}` | DELETE | Delete role mapping | Bearer/Cookie |
| `/discord/sync/manual` | POST | Trigger manual role synchronization | Bearer/Cookie |
| `/discord/sync/user/{user_id}` | POST | Sync roles for specific user | Bearer/Cookie |
| `/discord/sync/status` | GET | Get synchronization status | Bearer/Cookie |

### API Examples

#### Initiate Discord Login
```bash
curl "http://localhost:3000/api/discord/auth/login"
```
**Response:**
```json
{
  "auth_url": "https://discord.com/api/oauth2/authorize?client_id=...&redirect_uri=...&response_type=code&scope=identify+guilds&state=secure-state-token"
}
```

#### Configure Discord Guild
```bash
# Using Bearer token
curl -X POST /api/discord/guilds \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-jwt-token" \
  -d '{
    "guild_id": "123456789012345678",
    "guild_name": "My Discord Server",
    "is_enabled": true
  }'

# Using cookie authentication (frontend)
curl -X POST /api/discord/guilds \
  -H "Content-Type: application/json" \
  -H "Cookie: falcon_auth_token=your-cookie-token" \
  -d '{
    "guild_id": "123456789012345678", 
    "guild_name": "My Discord Server",
    "is_enabled": true
  }'
```

#### Create Role Mapping
```bash
# Create role mapping for specific guild
curl -X POST /api/discord/guilds/123456789012345678/role-mappings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-jwt-token" \
  -d '{
    "group_id": "go-falcon-group-id",
    "discord_role_id": "987654321098765432",
    "discord_role_name": "EVE Pilot",
    "is_active": true
  }'
```

#### List Role Mappings
```bash
# List role mappings for a guild
curl /api/discord/guilds/123456789012345678/role-mappings \
  -H "Authorization: Bearer your-jwt-token"

# Using cookie authentication
curl /api/discord/guilds/123456789012345678/role-mappings \
  -H "Cookie: falcon_auth_token=your-cookie-token"
```

#### Sync User Roles
```bash
curl -X POST /api/discord/sync/user/user-id-123 \
  -H "Authorization: Bearer your-jwt-token"
```
**Response:**
```json
{
  "user_id": "user-id-123",
  "discord_user_id": "123456789012345678",
  "sync_results": [
    {
      "guild_id": "123456789012345678",
      "guild_name": "My Discord Server",
      "roles_added": ["EVE Pilot", "Corporation Member"],
      "roles_removed": ["Inactive"],
      "total_roles": 5,
      "errors": []
    }
  ],
  "total_guilds": 1,
  "successful_guilds": 1,
  "failed_guilds": 0,
  "synced_at": "2024-01-15T10:30:00Z"
}
```

#### Get Sync Status
```bash
curl /api/discord/sync/status \
  -H "Authorization: Bearer your-jwt-token"
```
**Response:**
```json
{
  "last_full_sync": "2024-01-15T10:00:00Z",
  "sync_in_progress": false,
  "total_users": 150,
  "synced_users": 147,
  "failed_users": 3,
  "total_guilds": 2,
  "active_role_mappings": 12,
  "recent_errors": [
    {
      "user_id": "user-123",
      "guild_id": "guild-456",
      "error": "Bot missing permissions",
      "occurred_at": "2024-01-15T09:45:00Z"
    }
  ]
}
```

## Database Schema

### Discord Users Collection (`discord_users`)
```javascript
{
  "_id": "user-uuid",
  "go_falcon_user_id": "user-id-123",
  "discord_user_id": "123456789012345678",
  "username": "DiscordUser#1234",
  "discriminator": "1234",
  "avatar": "avatar-hash",
  "access_token": "encrypted-access-token",
  "refresh_token": "encrypted-refresh-token",
  "token_expires_at": "2024-01-15T11:30:00Z",
  "scopes": ["identify", "guilds"],
  "guilds": [
    {
      "id": "123456789012345678",
      "name": "My Discord Server",
      "permissions": "2147483647",
      "joined_at": "2024-01-10T10:00:00Z"
    }
  ],
  "last_sync": "2024-01-15T10:30:00Z",
  "is_active": true,
  "created_at": "2024-01-10T09:15:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### Discord Guild Configs Collection (`discord_guild_configs`)
```javascript
{
  "_id": "config-uuid",
  "guild_id": "123456789012345678",
  "guild_name": "My Discord Server",
  "bot_token": "encrypted-bot-token",
  "enabled": true,
  "settings": {
    "auto_sync": true,
    "sync_interval": "15m",
    "sync_on_join": true,
    "remove_roles_on_leave": true,
    "audit_channel_id": "123456789012345679",
    "notification_channel_id": "123456789012345680",
    "max_role_changes_per_sync": 100
  },
  "bot_permissions": "2147483647",
  "bot_user_id": "987654321098765432",
  "owner_id": "user-id-123",
  "last_health_check": "2024-01-15T10:30:00Z",
  "health_status": "healthy",
  "created_at": "2024-01-10T09:15:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### Discord Role Mappings Collection (`discord_role_mappings`)
```javascript
{
  "_id": "mapping-uuid",
  "guild_id": "123456789012345678",
  "discord_role_id": "987654321098765432",
  "discord_role_name": "EVE Pilot",
  "group_id": "go-falcon-group-id",
  "group_name": "EVE Players",
  "conditions": {
    "require_all": false,
    "character_count_min": 1,
    "character_count_max": null,
    "required_permissions": [],
    "excluded_groups": []
  },
  "priority": 1,
  "enabled": true,
  "created_by": "user-id-123",
  "last_sync": "2024-01-15T10:30:00Z",
  "sync_count": 147,
  "created_at": "2024-01-10T09:15:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### Discord Sync Status Collection (`discord_sync_status`)
```javascript
{
  "_id": "sync-uuid",
  "sync_type": "user|guild|mapping|full",
  "target_id": "user-id-123|guild-id-456|mapping-id-789",
  "status": "pending|running|completed|failed",
  "started_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:30:45Z",
  "duration": 45000,
  "results": {
    "users_processed": 150,
    "users_synced": 147,
    "users_failed": 3,
    "roles_added": 25,
    "roles_removed": 8,
    "guilds_processed": 2,
    "errors": []
  },
  "initiated_by": "user-id-123|system",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Discord OAuth States Collection (`discord_oauth_states`)
```javascript
{
  "_id": "state-uuid",
  "state": "secure-random-state-token",
  "user_id": "user-id-123",
  "redirect_url": "https://frontend.com/auth/discord/callback",
  "scopes": ["identify", "guilds"],
  "created_at": "2024-01-15T10:25:00Z",
  "expires_at": "2024-01-15T10:35:00Z",
  "used": false
}
```

## Configuration

### Environment Variables
```bash
# Discord OAuth Application Settings (Required for authentication)
DISCORD_CLIENT_ID=your_discord_application_client_id
DISCORD_CLIENT_SECRET=your_discord_application_client_secret
DISCORD_REDIRECT_URI=http://localhost:3000/api/discord/auth/callback

# Discord Bot Token (Configure in guild settings via API or frontend)
# Example bot token format: MTxxxxxxxxxxxxxxxxx.xxxxxx.xxxxxxxxxxxxxxxxxxxxxxxxxxx
# Note: Bot tokens are stored per-guild in the database, not as environment variables

# Discord OAuth Scopes (space-separated)
DISCORD_SCOPES="identify guilds guilds.join"

# Discord Sync Settings
DISCORD_SYNC_INTERVAL=15m
DISCORD_RATE_LIMIT_DELAY=1s

# Security Settings
DISCORD_STATE_EXPIRY=10m
DISCORD_TOKEN_REFRESH_THRESHOLD=24h
```

### Current Configuration Status
- ✅ **OAuth Setup**: Discord OAuth application configured with client credentials
- ✅ **Bot Token**: Bot tokens are configured per-guild via the Discord admin interface  
- ✅ **Authentication**: Dual authentication (Bearer + Cookie) enabled for all endpoints
- ✅ **API Integration**: All role-mapping endpoints implemented and functional

### OAuth Scopes

| Scope | Purpose | Required |
|-------|---------|----------|
| `identify` | Get user profile information | Yes |
| `guilds` | See user's Discord servers | Yes |
| `guilds.join` | Add users to servers (required for auto-join) | Yes |
| `role_connections.write` | Manage role connections | Optional |

### Bot Permissions

Required Discord bot permissions for role management and auto-join:

| Permission | Purpose | Required | Auto-Join |
|------------|---------|----------|-----------|
| `Manage Roles` | Add/remove roles from members | Yes | Yes |
| `View Channels` | Read guild information | Yes | No |
| `Create Instant Invite` | Add members to guild | Yes | **Yes** |
| `Send Messages` | Send audit/notification messages | Optional | No |
| `Embed Links` | Send rich embed messages | Optional | No |

**Note for Auto-Join**: The bot must have `Create Instant Invite` permission to add users to the guild using the `AddGuildMember` API. Without this permission, auto-join operations will fail with a 403 Forbidden error.

## Scheduled Tasks Integration

The Discord module integrates with the scheduler system for automated maintenance:

### Discord Token Refresh Task
```go
{
    ID:          "system-discord-token-refresh",
    Name:        "Discord Token Refresh",
    Description: "Refreshes expired Discord access tokens for users with linked Discord accounts",
    Type:        models.TaskTypeSystem,
    Schedule:    "0 */30 * * * *", // Every 30 minutes
    Status:      models.TaskStatusPending,
    Priority:    models.TaskPriorityNormal,
    Enabled:     true,
    Config: map[string]interface{}{
        "task_name": "discord_token_refresh",
        "parameters": map[string]interface{}{
            "batch_size": 50,
            "timeout":    "10m",
        },
    },
    Metadata: models.TaskMetadata{
        MaxRetries:    3,
        RetryInterval: models.Duration(5 * time.Minute),
        Timeout:       models.Duration(15 * time.Minute),
        Tags:          []string{"system", "discord", "tokens", "refresh"},
        IsSystem:      true,
        Source:        "system",
        Version:       1,
    },
}
```

### Discord Role Sync Task
```go
{
    ID:          "system-discord-role-sync",
    Name:        "Discord Role Synchronization",
    Description: "Synchronizes Discord roles with Go Falcon group memberships for all configured guilds",
    Type:        models.TaskTypeSystem,
    Schedule:    "0 */15 * * * *", // Every 15 minutes
    Status:      models.TaskStatusPending,
    Priority:    models.TaskPriorityNormal,
    Enabled:     true,
    Config: map[string]interface{}{
        "task_name": "discord_role_sync",
        "parameters": map[string]interface{}{
            "timeout": "30m",
        },
    },
    Metadata: models.TaskMetadata{
        MaxRetries:    2,
        RetryInterval: models.Duration(10 * time.Minute),
        Timeout:       models.Duration(45 * time.Minute),
        Tags:          []string{"system", "discord", "roles", "synchronization"},
        IsSystem:      true,
        Source:        "system",
        Version:       1,
    },
}
```

## Integration Examples

### User Authentication Flow
```go
// Initiate Discord OAuth
func (s *Service) InitiateOAuth(ctx context.Context, input *dto.InitiateDiscordOAuthInput) (*dto.InitiateDiscordOAuthOutput, error) {
    // Generate secure state
    state := generateSecureState()
    
    // Store OAuth state
    oauthState := &models.DiscordOAuthState{
        State:       state,
        UserID:      input.UserID, // Optional for login flow
        RedirectURL: input.RedirectURL,
        Scopes:      input.Scopes,
        CreatedAt:   time.Now(),
        ExpiresAt:   time.Now().Add(10 * time.Minute),
    }
    
    if err := s.repository.CreateOAuthState(ctx, oauthState); err != nil {
        return nil, fmt.Errorf("failed to store OAuth state: %w", err)
    }
    
    // Build Discord authorization URL
    authURL := s.buildAuthURL(state, input.Scopes, input.RedirectURL)
    
    return &dto.InitiateDiscordOAuthOutput{
        AuthURL: authURL,
        State:   state,
    }, nil
}
```

### Role Synchronization
```go
// Sync user roles across all configured guilds
func (s *Service) SyncUserRoles(ctx context.Context, userID string) (*dto.SyncUserRolesOutput, error) {
    // Get Discord user
    discordUser, err := s.repository.GetDiscordUserByGoFalconUserID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("user not linked to Discord: %w", err)
    }
    
    // Get user's group memberships
    groups, err := s.groupsService.GetUserGroups(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user groups: %w", err)
    }
    
    // Process each configured guild
    var syncResults []dto.GuildSyncResult
    for _, guildConfig := range s.getActiveGuildConfigs(ctx) {
        result, err := s.syncService.SyncUserInGuild(ctx, discordUser, guildConfig, groups)
        if err != nil {
            result = dto.GuildSyncResult{
                GuildID:   guildConfig.GuildID,
                GuildName: guildConfig.GuildName,
                Errors:    []string{err.Error()},
            }
        }
        syncResults = append(syncResults, result)
    }
    
    // Update last sync time
    s.repository.UpdateDiscordUserLastSync(ctx, discordUser.ID, time.Now())
    
    return &dto.SyncUserRolesOutput{
        UserID:           userID,
        DiscordUserID:    discordUser.DiscordUserID,
        SyncResults:      syncResults,
        TotalGuilds:      len(syncResults),
        SuccessfulGuilds: countSuccessfulSyncs(syncResults),
        FailedGuilds:     countFailedSyncs(syncResults),
        SyncedAt:         time.Now(),
    }, nil
}
```

### Bot Management
```go
// Discord bot service for role management
type BotService struct {
    httpClient *http.Client
    rateLimiter *rate.Limiter
}

func (bs *BotService) ManageUserRoles(ctx context.Context, guildID, userID string, rolesToAdd, rolesToRemove []string, botToken string) error {
    // Add roles
    for _, roleID := range rolesToAdd {
        if err := bs.addRoleToMember(ctx, guildID, userID, roleID, botToken); err != nil {
            return fmt.Errorf("failed to add role %s: %w", roleID, err)
        }
    }
    
    // Remove roles
    for _, roleID := range rolesToRemove {
        if err := bs.removeRoleFromMember(ctx, guildID, userID, roleID, botToken); err != nil {
            return fmt.Errorf("failed to remove role %s: %w", roleID, err)
        }
    }
    
    return nil
}

func (bs *BotService) addRoleToMember(ctx context.Context, guildID, userID, roleID, botToken string) error {
    // Wait for rate limit
    if err := bs.rateLimiter.Wait(ctx); err != nil {
        return err
    }
    
    url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s/roles/%s", guildID, userID, roleID)
    req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", "Bot "+botToken)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := bs.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusNoContent {
        return fmt.Errorf("Discord API error: %d", resp.StatusCode)
    }
    
    return nil
}
```

### Module Dependencies

The Discord module integrates with other Go Falcon modules:

```go
// GroupsService interface for accessing user groups
type GroupsService interface {
    GetUserGroups(ctx context.Context, userID string) ([]services.GroupInfo, error)
}

// Module initialization with dependencies
func NewModule(db *database.MongoDB, redis *database.Redis, groupsService GroupsService) *Module {
    baseModule := module.NewBaseModule("discord", db, redis)
    
    // Create service with groups service dependency
    service := services.NewService(db, groupsService)
    
    return &Module{
        BaseModule: baseModule,
        service:    service,
        routes:     routes.NewModule(service, nil),
    }
}
```

## Security Considerations

### OAuth Security
- **State Validation**: All OAuth flows use cryptographically secure state parameters
- **Token Encryption**: Access and refresh tokens encrypted at rest
- **Scope Limitation**: Minimal required scopes to limit access surface
- **State Expiration**: OAuth states expire after 10 minutes
- **CSRF Protection**: State parameter prevents cross-site request forgery

### Bot Security
- **Token Protection**: Bot tokens encrypted and never logged
- **Permission Validation**: Bot permissions verified before operations
- **Rate Limiting**: Strict compliance with Discord API rate limits
- **Audit Logging**: All role changes logged with detailed metadata
- **Error Sanitization**: Sensitive data removed from error messages

### Data Protection
- **Encryption at Rest**: Sensitive tokens encrypted in database
- **Access Control**: API endpoints require appropriate authentication
- **Data Retention**: Configurable data retention and cleanup policies
- **Privacy Controls**: Users can unlink accounts and delete data

### API Security
- **Authentication**: Integration with Go Falcon JWT authentication
- **Authorization**: Permission-based access control
- **Input Validation**: All inputs validated and sanitized
- **Rate Limiting**: Protection against abuse and DoS attacks

## Error Handling

### Common Error Scenarios
- **OAuth Failures**: Invalid codes, expired states, network errors
- **Bot Permission Issues**: Missing permissions, invalid tokens
- **Discord API Errors**: Rate limits, server errors, invalid data
- **Sync Conflicts**: Role conflicts, membership changes
- **Database Errors**: Connection issues, validation failures

### Error Recovery
```go
// Automatic retry with exponential backoff
func (s *Service) withRetry(ctx context.Context, operation func() error) error {
    var lastErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        if err := operation(); err != nil {
            lastErr = err
            if !isRetryableError(err) {
                return err
            }
            
            backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(backoff):
                continue
            }
        }
        return nil
    }
    return lastErr
}
```

### Discord API Error Handling
```go
// Handle Discord API rate limiting
func (bs *BotService) handleDiscordResponse(resp *http.Response) error {
    switch resp.StatusCode {
    case 429: // Rate limited
        retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
        return &RateLimitError{RetryAfter: retryAfter}
    case 401:
        return &AuthenticationError{Message: "Invalid bot token"}
    case 403:
        return &PermissionError{Message: "Bot lacks required permissions"}
    case 404:
        return &NotFoundError{Message: "Guild, user, or role not found"}
    default:
        if resp.StatusCode >= 500 {
            return &ServerError{StatusCode: resp.StatusCode}
        }
        return nil
    }
}
```

## Performance Considerations

### Database Optimization
- **Indexes**: Automatic creation of performance indexes
- **Connection Pooling**: Efficient database connection usage
- **Pagination**: Large result sets paginated for memory efficiency
- **Aggregation**: Complex queries use MongoDB aggregation pipeline

### Discord API Optimization
- **Rate Limiting**: Intelligent rate limit management with exponential backoff
- **Batch Operations**: Bulk role operations where possible
- **Caching**: Guild and role information cached to reduce API calls
- **Concurrent Processing**: Parallel processing with worker pools

### Sync Performance
```go
// Batch sync configuration
type SyncConfig struct {
    BatchSize           int           `json:"batch_size"`           // Users per batch
    ConcurrentWorkers   int           `json:"concurrent_workers"`   // Parallel workers
    RateLimitDelay      time.Duration `json:"rate_limit_delay"`     // Delay between API calls
    MaxRetries          int           `json:"max_retries"`          // Retry attempts
    RetryBackoff        time.Duration `json:"retry_backoff"`        // Retry delay
}

// Default performance settings
var DefaultSyncConfig = SyncConfig{
    BatchSize:         50,
    ConcurrentWorkers: 5,
    RateLimitDelay:    1 * time.Second,
    MaxRetries:        3,
    RetryBackoff:      2 * time.Second,
}
```

## Monitoring & Health Checks

### Health Endpoints
```bash
# Module health check
curl /api/discord/status

# Response
{
  "module": "discord",
  "status": "healthy",
  "message": ""
}
```

### Monitoring Metrics
- **OAuth Success/Failure Rates**: Track authentication flow success
- **Role Sync Statistics**: Monitor sync performance and errors
- **Discord API Usage**: Track API calls and rate limit status
- **Bot Health**: Monitor bot connectivity and permissions
- **Database Performance**: Track query performance and connection health

### Alerting Integration
```go
// Health monitoring in background task
func (m *Module) runHealthMonitoring(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Check Discord API connectivity
            if err := m.service.CheckDiscordAPIHealth(ctx); err != nil {
                slog.Error("Discord API health check failed", "error", err)
            }
            
            // Check bot permissions for all guilds
            guilds := m.service.GetActiveGuildConfigs(ctx)
            for _, guild := range guilds {
                if err := m.service.CheckBotPermissions(ctx, guild.GuildID, guild.BotToken); err != nil {
                    slog.Error("Bot permission check failed", 
                        "guild_id", guild.GuildID,
                        "guild_name", guild.GuildName,
                        "error", err)
                }
            }
        }
    }
}
```

## Best Practices

### Configuration Management
1. **Secure Token Storage**: Always encrypt bot tokens and OAuth secrets
2. **Permission Validation**: Verify bot permissions before configuration
3. **Guild Validation**: Validate guild access and bot membership
4. **Monitoring Setup**: Configure health checks and alerting

### Role Management
1. **Mapping Design**: Create clear, logical role mappings
2. **Priority Handling**: Use priority to resolve role conflicts
3. **Condition Logic**: Design inclusive conditions for group membership
4. **Audit Trails**: Maintain detailed logs of all role changes

### Synchronization
1. **Batch Processing**: Process users in manageable batches
2. **Error Resilience**: Handle individual failures gracefully
3. **Rate Compliance**: Respect Discord API rate limits
4. **Monitoring**: Track sync performance and error rates

### Security
1. **Minimal Scopes**: Request only necessary OAuth scopes
2. **Token Rotation**: Regularly refresh access tokens
3. **Permission Reviews**: Regularly audit bot permissions
4. **Data Cleanup**: Implement data retention policies

## Troubleshooting

### Common Issues

**OAuth Flow Failures**
- Verify Discord application configuration
- Check redirect URI matches exactly
- Ensure client ID and secret are correct
- Validate OAuth state handling

**Bot Permission Issues**
```bash
# Check bot permissions
curl /api/discord/guilds/guild-id \
  -H "Authorization: Bearer token"

# Look for permission issues in response
{
  "bot_permissions": "2147483647",
  "health_status": "unhealthy",
  "last_error": "Bot missing Manage Roles permission"
}
```

**Role Sync Failures**
- Verify role mappings are correct
- Check bot has required permissions
- Review group membership data
- Examine sync error logs

**Discord API Rate Limits**
- Monitor rate limit headers in responses
- Increase delays between operations
- Implement exponential backoff
- Consider reducing batch sizes

### Debugging Commands
```bash
# Get Discord user info
curl /api/discord/user/profile \
  -H "Authorization: Bearer token"

# Check guild configuration
curl /api/discord/guilds/guild-id \
  -H "Authorization: Bearer token"

# Review role mappings
curl /api/discord/role-mappings?guild_id=guild-id \
  -H "Authorization: Bearer token"

# Get sync status
curl /api/discord/sync/status \
  -H "Authorization: Bearer token"

# Manual user sync (for testing)
curl -X POST /api/discord/sync/user/user-id \
  -H "Authorization: Bearer token"
```

### Log Analysis
```bash
# Discord service logs
grep "discord" /var/log/go-falcon/app.log

# OAuth flow logs
grep "discord_oauth" /var/log/go-falcon/app.log

# Bot operation logs
grep "discord_bot" /var/log/go-falcon/app.log

# Sync operation logs
grep "discord_sync" /var/log/go-falcon/app.log
```

## Dependencies

### External Services
- **Discord API**: OAuth and Bot API endpoints
- **MongoDB**: User data and configuration storage
- **Redis**: Caching and session management

### Go Packages
- `golang.org/x/oauth2` - OAuth2 client implementation
- `github.com/bwmarrin/discordgo` - Discord API library (optional)
- `github.com/danielgtaylor/huma/v2` - API framework
- `go.mongodb.org/mongo-driver` - MongoDB client
- `golang.org/x/time/rate` - Rate limiting

### Internal Dependencies
- `go-falcon/pkg/module` - Base module interface
- `go-falcon/pkg/database` - Database connections  
- `go-falcon/pkg/handlers` - HTTP utilities
- `go-falcon/internal/groups` - Group membership service
- `go-falcon/internal/auth` - Authentication integration

## Production Deployment Checklist

### Pre-Deployment Requirements

**Discord OAuth Application Configuration:**
- ✅ Discord application created at https://discord.com/developers/applications
- ✅ OAuth redirect URI configured: `https://go.eveonline.it/discord/auth/callback`
- ✅ Required OAuth scopes enabled: `identify`, `guilds`, `guilds.join`

**Environment Configuration:**
- ✅ `DISCORD_CLIENT_ID` set to Discord application client ID  
- ✅ `DISCORD_CLIENT_SECRET` set to Discord application client secret
- ✅ `DISCORD_REDIRECT_URI` matches OAuth app configuration
- ✅ `DISCORD_SCOPES="identify guilds guilds.join"` includes auto-join scope

**Discord Bot Configuration:**
- ✅ Discord bot created and added to target guilds
- ✅ Bot tokens encrypted and stored via guild configuration API
- ✅ Bot permissions: `Manage Roles` + `Create Instant Invite` (for auto-join)
- ✅ Role mappings configured between Go Falcon groups and Discord roles

**Database Setup:**
- ✅ MongoDB collections created and indexed
- ✅ Redis available for session management
- ✅ Database migration status verified

### Deployment Verification Steps

**1. OAuth Flow Testing:**
```bash
# Test OAuth login URL generation
curl "https://go.eveonline.it/api/discord/auth/login"

# Verify guilds.join scope is requested in auth_url response
```

**2. Auto-Join Functionality Testing:**
```bash
# Complete OAuth flow and verify auto-join occurs
# Check application logs for auto-join success/failure messages
```

**3. Guild Configuration Verification:**
```bash
# Verify guild configurations
curl -H "Authorization: Bearer token" \
  "https://go.eveonline.it/api/discord/guilds"

# Check bot permissions in each guild
curl -H "Authorization: Bearer token" \
  "https://go.eveonline.it/api/discord/guilds/{guild_id}"
```

**4. Role Mapping Validation:**
```bash
# Verify role mappings exist
curl -H "Authorization: Bearer token" \
  "https://go.eveonline.it/api/discord/guilds/{guild_id}/role-mappings"
```

**5. Sync System Testing:**
```bash
# Test manual sync including auto-join
curl -X POST -H "Authorization: Bearer token" \
  "https://go.eveonline.it/api/discord/sync/user/{user_id}"
```

### Monitoring & Observability

**Key Metrics to Monitor:**
- Discord OAuth success/failure rates
- Auto-join success/failure rates  
- Discord API rate limiting incidents
- Bot permission errors
- Guild membership changes

**Log Analysis:**
```bash
# Monitor auto-join logs
grep "auto-join" /var/log/go-falcon/app.log

# Monitor Discord API errors  
grep "discord.*error" /var/log/go-falcon/app.log

# Monitor role synchronization
grep "discord.*sync" /var/log/go-falcon/app.log
```

**Health Check Endpoints:**
- `GET /api/discord/status` - Module health status
- `GET /api/discord/sync/status` - Sync operation status

### Troubleshooting Common Issues

**Auto-Join Failures:**
1. **"Bot lacks permission to add members"** - Verify bot has `Create Instant Invite` permission
2. **"User lacks guilds.join scope"** - Verify `DISCORD_SCOPES` environment variable is correct  
3. **"Guild not found"** - Verify bot is member of target guild
4. **"Access token expired"** - Token refresh scheduled task should handle this automatically

**OAuth Flow Issues:**
1. **"Invalid redirect URI"** - Verify Discord app redirect URI matches `DISCORD_REDIRECT_URI` 
2. **"Invalid client credentials"** - Verify `DISCORD_CLIENT_ID` and `DISCORD_CLIENT_SECRET`
3. **"State parameter invalid"** - Check OAuth state expiration settings

## Future Enhancements

### Planned Features
- **Web UI**: Browser-based guild and role mapping management
- **Advanced Conditions**: More sophisticated role assignment logic
- **Webhook Support**: Discord webhook integration for events
- **Slash Commands**: Discord bot slash command support
- **Multi-Server Management**: Enhanced multi-guild management tools
- **Advanced Audit Logs**: More detailed activity tracking
- **Template System**: Role mapping templates for common configurations
- **Bulk Import**: CSV/JSON import for role mappings

### API Extensions
- **Webhook Endpoints**: Discord event webhook handlers
- **Advanced Filtering**: More sophisticated query capabilities
- **Bulk Operations**: Batch role mapping operations
- **Export/Import**: Configuration backup and restore
- **Analytics**: Discord usage and engagement metrics