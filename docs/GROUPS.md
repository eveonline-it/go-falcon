# Group Management Module

## 🎯 Overview

The Group Management module provides hierarchical, role-based access control (RBAC) for EVE Online organizations. It manages permissions at character, corporation, and alliance levels with performance-optimized caching and comprehensive audit logging.

## 📋 Table of Contents

- [Core Concepts](#core-concepts)
- [Architecture](#architecture)
- [Key Features](#key-features)
- [Implementation Phases](#implementation-phases)
- [Permission Model](#permission-model)
- [Performance Optimization](#performance-optimization)
- [Security Design](#security-design)
- [Integration Points](#integration-points)
- [Database Schema](#database-schema)
- [API Endpoints](#api-endpoints)

## 🔑 Core Concepts

### Group Types

#### System Groups (Immutable)
- `administrator` - Full administrative access (managed by super_admin users)
- `authenticated` - Any logged-in character with valid session
- `guest` - Logged-in character with minimal/no ESI scopes

**Note**: `super_admin` is not a group but a boolean flag in the user_profiles collection. Users with `super_admin: true` bypass all permission checks and can manage the Administrator group.

#### EVE Organization Groups (Auto-assigned)
- `corporation_member` - Character in an allowed corporation
- `alliance_member` - Character in an allowed alliance

#### EVE Role Groups (Assignable)
**Public Groups** (visible to all members):
- `fleet_commander` - Can create/manage fleet operations
- `recruiter` - Access to recruitment tools and applicant data
- `trainer` - Can manage training programs and skill plans
- `logistics` - Logistics and supply chain management
- `diplomacy` - Diplomatic tools and contact management

**Hidden Groups** (visible only to administrators and group members):
- `recons` - Access to reconnaissance tools and intel
- `blackops` - Black ops fleet participation and planning
- `capitals` - Capital ship pilot permissions
- `titan` - Titan pilot specific permissions
- `special_ops` - Special operations and covert activities

#### Custom Groups (User-defined)
- Created by administrators for specific organizational needs
- Can combine multiple permissions and roles
- Support for temporary/time-limited memberships

### Hierarchical Permission Model

```
Priority Order (Highest → Lowest):
1. Explicit Denial (always wins)
2. Character-level permissions
3. Corporation-level permissions  
4. Alliance-level permissions
5. Role-based permissions
6. Default deny (implicit)
```

## 🏗️ Architecture

### High-Level Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Request                             │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│            Authentication Middleware                        │
│         (JWT Validation + Session Check)                    │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│          Character Context Middleware                       │
│  (Resolve Character/Corp/Alliance + super_admin flag)       │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  1. Load user profile from database                 │   │
│  │  2. Check super_admin flag                          │   │
│  │  3. If super_admin: true → Set bypass flag          │   │
│  │  4. Resolve character's corp/alliance               │   │
│  │  5. Attach context to request                       │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│         Permission Resolution Engine                        │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Permission Check Flow:                             │   │
│  │  1. Check context for super_admin bypass flag       │   │
│  │  2. If bypassed → Allow immediately                 │   │
│  │  3. Otherwise, proceed with normal checks:          │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Composite Permission Builder:                      │   │
│  │  1. Load from Redis cache (if valid)                │   │
│  │  2. Build permission set:                           │   │
│  │     - Collect all applicable permissions            │   │
│  │     - Apply hierarchical rules                      │   │
│  │     - Process denials                               │   │
│  │  3. Cache result (TTL: 5 minutes)                   │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Authorization Check:                               │   │
│  │  - Match requested resource against permissions     │   │
│  │  - Log access attempt (if configured)               │   │
│  │  - Return allow/deny decision                       │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────┬───────────────────────────────────────┘
                      │
         ┌────────────┴────────────┐
         │                         │
    ▼ ALLOWED                 ▼ DENIED
┌────────────┐           ┌─────────────┐
│  Handler   │           │  403 Error  │
└────────────┘           └─────────────┘
```

### Component Architecture

```
internal/groups/
├── dto/
│   ├── inputs.go          # Create/Update group requests
│   ├── outputs.go         # Group responses
│   └── validators.go      # Custom validation logic
├── middleware/
│   ├── authorization.go   # Group-based auth checks
│   ├── context.go         # Character context resolution (includes super_admin)
│   └── cache.go           # Redis cache management
├── routes/
│   ├── routes.go          # Main route registration
│   ├── groups.go          # Group CRUD operations
│   ├── members.go         # Membership management
│   └── permissions.go     # Permission assignments
├── services/
│   ├── group_service.go   # Group business logic
│   ├── permission.go      # Permission resolution
│   ├── cache.go           # Cache invalidation logic
│   └── audit.go           # Audit logging
├── models/
│   ├── group.go           # Group model
│   ├── membership.go      # Membership model
│   └── permission.go      # Permission model
├── module.go              # Module initialization
└── CLAUDE.md              # Module documentation
```

## ✨ Key Features

### Performance Optimized
- **Redis Caching**: Multi-layer caching with smart invalidation
- **Composite Permissions**: Pre-computed permission sets at login
- **Batch Operations**: Bulk membership updates
- **Indexed Queries**: Optimized MongoDB indexes
- **Connection Pooling**: Efficient database connections

### Security Focused
- **Default Deny**: No access unless explicitly granted
- **Explicit Denials**: Override any allow permission
- **Audit Logging**: Complete trail of permission changes
- **Session Management**: Automatic permission refresh on changes
- **Rate Limiting**: Prevent permission checking abuse

### Developer Friendly
- **Simple API**: `CanAccess(characterID, resource, action)`
- **Middleware Helpers**: Easy route protection
- **Clear Documentation**: Comprehensive examples
- **Testing Utilities**: Mock permission system for tests
- **Debug Mode**: Detailed permission resolution logs

### Integration Ready
- **Discord Bot**: Role synchronization via bot with Redis pub/sub
- **ESI Validation**: Verify EVE Online memberships
- **Event System**: Subscribe to permission changes
- **REST API**: Full CRUD operations
- **OpenAPI Spec**: Auto-generated documentation

## 📈 Implementation Phases

### Phase 1: Foundation (Week 1-2)
**Goal**: Basic group system integrated with existing auth

- [ ] Group CRUD operations
- [ ] Simple character membership
- [ ] System groups (super_admin, authenticated, guest)
- [ ] Basic middleware integration
- [ ] MongoDB schemas and indexes
- [ ] Unit tests

**Deliverables**:
- Working group creation/management
- Character assignment to groups
- Basic permission checks

### Phase 2: EVE Integration (Week 3-4)
**Goal**: Corporation/Alliance support with auto-assignment

- [ ] Corporation/Alliance group types
- [ ] Auto-assignment rules
- [ ] ESI membership validation
- [ ] Hierarchical permission resolution
- [ ] Redis caching layer
- [ ] Performance benchmarks

**Deliverables**:
- Automatic corp/alliance groups
- Inheritance working correctly
- <10ms permission checks

### Phase 3: Advanced Features (Week 5-6)
**Goal**: Production-ready with all features

- [ ] Custom groups
- [ ] Role-based groups (FC, recons, etc.)
- [ ] Discord bot integration (optional)
- [ ] Redis pub/sub for bot events
- [ ] Audit logging
- [ ] Admin UI endpoints
- [ ] Comprehensive testing

**Deliverables**:
- Full feature set
- Discord bot event publishing
- Admin management tools
- Production deployment guide

### Phase 4: Optimization (Week 7-8)
**Goal**: Production hardening and optimization

- [ ] Performance profiling
- [ ] Cache optimization
- [ ] Bulk operations
- [ ] Migration tools
- [ ] Documentation
- [ ] Load testing

**Deliverables**:
- <5ms average permission check
- 10,000+ concurrent users support
- Complete documentation

## 🛡️ Permission Model

### Permission Structure

```go
type Permission struct {
    ID         string    `bson:"_id"`
    GroupID    string    `bson:"group_id"`
    Resource   string    `bson:"resource"`   // e.g., "scheduler:tasks"
    Action     string    `bson:"action"`     // e.g., "create", "read", "update", "delete"
    Effect     string    `bson:"effect"`     // "allow" or "deny"
    Conditions []string  `bson:"conditions"` // Optional conditions
    Priority   int       `bson:"priority"`   // Resolution order
}
```

### Composite Permission Cache

```go
type CompositePermissions struct {
    CharacterID  int64                  `json:"character_id"`
    UserID       string                 `json:"user_id"`
    Denials      map[string]bool        `json:"denials"`      // Explicit denials (highest priority)
    Character    map[string]bool        `json:"character"`    // Character-specific
    Corporation  map[string]bool        `json:"corporation"`  // From corp membership
    Alliance     map[string]bool        `json:"alliance"`     // From alliance membership
    Roles        map[string]bool        `json:"roles"`        // From assigned roles
    Effective    map[string]bool        `json:"effective"`    // Pre-computed final permissions
    ComputedAt   time.Time              `json:"computed_at"`
    TTL          time.Duration          `json:"ttl"`
}
```

### Character Context Structure

```go
type CharacterContext struct {
    CharacterID    int64  `json:"character_id"`
    UserID         string `json:"user_id"`
    CorporationID  int32  `json:"corporation_id"`
    AllianceID     int32  `json:"alliance_id,omitempty"`
    SuperAdmin     bool   `json:"super_admin"`      // Set from user_profiles
    BypassPerms    bool   `json:"bypass_perms"`     // Computed flag for middleware
}
```

### Permission Check Flow

```go
func (s *PermissionService) CanAccess(ctx context.Context, characterID int64, resource, action string) (bool, error) {
    // 1. Check if context has bypass flag (set by Character Context Middleware)
    if charCtx := ctx.Value("character_context").(*CharacterContext); charCtx != nil && charCtx.BypassPerms {
        return true, nil
    }
    
    // 2. Try Redis cache for normal permissions
    cacheKey := fmt.Sprintf("perms:%d", characterID)
    if perms, err := s.cache.GetComposite(cacheKey); err == nil {
        return perms.Effective[fmt.Sprintf("%s:%s", resource, action)], nil
    }
    
    // 3. Build composite permissions
    composite := s.buildComposite(ctx, characterID)
    
    // 4. Cache for next time
    s.cache.SetComposite(cacheKey, composite, 5*time.Minute)
    
    // 5. Return decision
    return composite.Effective[fmt.Sprintf("%s:%s", resource, action)], nil
}
```

## ⚡ Performance Optimization

### Caching Strategy

```yaml
Redis Cache Layers:
  L1 - Session Cache (30 min TTL):
    - User's composite permissions
    - Frequently accessed resources
    
  L2 - Group Cache (5 min TTL):
    - Group memberships
    - Group permissions
    
  L3 - Organization Cache (1 hour TTL):
    - Corporation memberships
    - Alliance memberships
```

### Cache Invalidation

```go
// Smart invalidation using Redis pub/sub
type CacheInvalidator struct {
    redis   *redis.Client
    pubsub  *redis.PubSub
    
    // Invalidation patterns
    patterns map[string]func(message string)
}

// Events that trigger invalidation:
// - Character changes corporation
// - Corporation changes alliance  
// - Group membership changes
// - Permission updates
// - Manual admin flush
```

### Database Indexes

```javascript
// MongoDB indexes for optimal performance
db.groups.createIndex({ "name": 1 }, { unique: true })
db.groups.createIndex({ "type": 1, "active": 1 })

db.group_members.createIndex({ "group_id": 1, "character_id": 1 }, { unique: true })
db.group_members.createIndex({ "character_id": 1 })
db.group_members.createIndex({ "corporation_id": 1 })
db.group_members.createIndex({ "alliance_id": 1 })

db.permissions.createIndex({ "group_id": 1 })
db.permissions.createIndex({ "resource": 1, "action": 1 })
db.permissions.createIndex({ "effect": 1, "priority": -1 })
```

## 🔒 Security Design

### Principles

1. **Principle of Least Privilege**: Start with no access, grant explicitly
2. **Defense in Depth**: Multiple validation layers
3. **Fail Secure**: Errors result in denial, not access
4. **Audit Everything**: Log all permission changes and access attempts
5. **Time-bound Access**: Support temporary permissions with expiration

### Security Features

```go
// Audit log for compliance and debugging
type AuditLog struct {
    ID          string    `bson:"_id"`
    Timestamp   time.Time `bson:"timestamp"`
    ActorID     int64     `bson:"actor_id"`     // Who made the change
    Action      string    `bson:"action"`       // What they did
    Target      string    `bson:"target"`       // What was affected
    OldValue    any       `bson:"old_value"`    // Previous state
    NewValue    any       `bson:"new_value"`    // New state
    IPAddress   string    `bson:"ip_address"`
    UserAgent   string    `bson:"user_agent"`
    Success     bool      `bson:"success"`
    ErrorMsg    string    `bson:"error_msg,omitempty"`
}
```

## 🔌 Integration Points

### Discord Bot Integration (Optional)

```go
type DiscordIntegration struct {
    Enabled     bool              `json:"enabled"`
    BotToken    string            `json:"bot_token"`         // Discord bot token
    GuildID     string            `json:"guild_id"`          // Discord server ID
    RoleMapping map[string]string `json:"role_mapping"`      // group_name -> discord_role_id
    SyncEvents  []string          `json:"sync_events"`       // "join", "leave", "promote"
    SyncMethod  string            `json:"sync_method"`       // "push" or "pull"
}

// Bot Integration Pattern:
// 1. API publishes events to Redis pub/sub when group membership changes
// 2. Discord bot subscribes to Redis events
// 3. Bot updates Discord roles based on events
// 4. Bot can also pull current state via API for full sync

// Events published to Redis:
// - group:member:added   -> Bot adds Discord role
// - group:member:removed -> Bot removes Discord role  
// - group:deleted        -> Bot cleans up all related Discord roles
// - group:sync:requested -> Bot performs full synchronization
```

### ESI Validation

```go
// Periodic validation of EVE memberships
func (s *ValidationService) ValidateMemberships(ctx context.Context) error {
    // 1. Get all corp/alliance groups
    // 2. For each member, verify via ESI
    // 3. Remove invalid memberships
    // 4. Log changes for audit
    return nil
}
```

### Module Integration

```yaml
Auth Module:
  - Provides JWT validation
  - Supplies character context
  - Handles SSO callbacks

Users Module:
  - Links to group memberships
  - Shows user's groups in profile
  - Manages character associations

Scheduler Module:
  - Checks task creation permissions
  - Validates task execution rights
  - Supports group-based task visibility
```

## 📊 Database Schema

### Collections

```javascript
// groups collection
{
  "_id": "ObjectId",
  "name": "fleet_commander",
  "display_name": "Fleet Commander",
  "description": "Can create and lead fleet operations",
  "type": "role", // system|organization|role|custom
  "visibility": "public", // public|hidden|private
  "auto_assign": false,
  "auto_assign_rules": {
    "corporation_ids": [],
    "alliance_ids": [],
    "character_attributes": {}
  },
  "discord_role_id": "123456789",
  "created_at": "2024-01-01T00:00:00Z",
  "created_by": 95123456,
  "updated_at": "2024-01-01T00:00:00Z",
  "active": true
}

// Visibility levels:
// - public: Visible to all authenticated users
// - hidden: Visible only to administrators and group members
// - private: Visible only to administrators

// group_members collection
{
  "_id": "ObjectId",
  "group_id": "ObjectId",
  "member_type": "character", // character|corporation|alliance
  "member_id": "95123456", // character_id, corp_id, or alliance_id
  "assigned_by": 95123456,
  "assigned_at": "2024-01-01T00:00:00Z",
  "expires_at": null, // Optional expiration
  "notes": "Promoted after successful fleet",
  "active": true
}

// group_permissions collection
{
  "_id": "ObjectId",
  "group_id": "ObjectId",
  "resource": "scheduler:tasks",
  "action": "create", // create|read|update|delete|*
  "effect": "allow", // allow|deny
  "conditions": [], // Future: conditional permissions
  "priority": 100,
  "created_at": "2024-01-01T00:00:00Z"
}

// permission_audit_log collection
{
  "_id": "ObjectId",
  "timestamp": "2024-01-01T00:00:00Z",
  "actor_id": 95123456,
  "action": "grant_permission",
  "target_type": "group",
  "target_id": "ObjectId",
  "details": {
    "group_name": "fleet_commander",
    "permission": "scheduler:tasks:create",
    "effect": "allow"
  },
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "success": true
}
```

## 🌐 API Endpoints

### Group Management

```yaml
GET    /groups                    # List visible groups (filtered by visibility)
POST   /groups                    # Create new group (admin only)
GET    /groups/{id}               # Get group details (visibility check)
PUT    /groups/{id}               # Update group (admin only)
DELETE /groups/{id}               # Delete group (admin only)

GET    /groups/{id}/members       # List group members (visibility check)
POST   /groups/{id}/members       # Add members (admin only)
DELETE /groups/{id}/members/{mid} # Remove member (admin only)

GET    /groups/{id}/permissions   # List group permissions (admin only)
POST   /groups/{id}/permissions   # Add permissions (admin only)
DELETE /groups/{id}/permissions/{pid} # Remove permission (admin only)
```

### Group Visibility Logic

```go
// Service filters groups based on visibility and user context
func (s *GroupService) ListGroups(ctx context.Context, characterID int64) ([]Group, error) {
    charCtx := ctx.Value("character_context").(*CharacterContext)
    
    // Super admins and administrators see all groups
    if charCtx.SuperAdmin || s.IsInGroup(ctx, characterID, "administrator") {
        return s.repo.GetAllGroups(ctx)
    }
    
    // Get user's group memberships
    userGroups := s.GetUserGroups(ctx, characterID)
    userGroupIDs := make(map[string]bool)
    for _, g := range userGroups {
        userGroupIDs[g.ID] = true
    }
    
    // Filter groups by visibility
    allGroups, _ := s.repo.GetAllGroups(ctx)
    visibleGroups := []Group{}
    
    for _, group := range allGroups {
        switch group.Visibility {
        case "public":
            // Everyone can see public groups
            visibleGroups = append(visibleGroups, group)
        case "hidden":
            // Only members and admins can see hidden groups
            if userGroupIDs[group.ID] {
                visibleGroups = append(visibleGroups, group)
            }
        case "private":
            // Only admins can see private groups (already handled above)
            continue
        }
    }
    
    return visibleGroups, nil
}
```

### User Groups

```yaml
GET    /users/me/groups           # Current user's groups
GET    /users/{id}/groups         # User's groups (admin)
POST   /users/{id}/groups         # Assign user to group (admin)
DELETE /users/{id}/groups/{gid}   # Remove user from group (admin)
```

### Permission Checks

```yaml
POST   /permissions/check         # Check permission
{
  "character_id": 95123456,
  "resource": "scheduler:tasks",
  "action": "create"
}

GET    /permissions/effective/{character_id} # Get effective permissions
POST   /permissions/refresh/{character_id}   # Force cache refresh
```

### Audit Logs

```yaml
GET    /audit/permissions         # Permission change logs
GET    /audit/access              # Access attempt logs
GET    /audit/groups              # Group change logs
```

## 🧪 Testing Strategy

### Unit Tests
- Permission resolution logic
- Cache invalidation
- Group membership rules
- Hierarchical inheritance

### Integration Tests
- End-to-end permission flows
- MongoDB operations
- Redis caching
- ESI validation

### Performance Tests
- Permission check latency (<5ms target)
- Concurrent user load (10,000+ users)
- Cache hit rates (>95% target)
- Database query performance

### Security Tests
- Denial precedence
- Default deny behavior
- Audit log completeness
- Session invalidation

## 📝 Configuration

```yaml
groups:
  # Cache settings
  cache:
    enabled: true
    ttl_minutes: 5
    redis_prefix: "groups:"
  
  # Performance settings
  performance:
    max_groups_per_user: 50
    max_permissions_per_group: 100
    batch_size: 100
  
  # Security settings
  security:
    audit_enabled: true
    audit_access_attempts: false  # High volume
    default_deny: true
    session_refresh_on_change: true
  
  # Integration settings
  integrations:
    discord:
      enabled: false
      bot_token: "${DISCORD_BOT_TOKEN}"
      guild_id: "${DISCORD_GUILD_ID}"
      sync_method: "push"  # push (via Redis events) or pull (bot polls API)
    esi:
      validation_enabled: true
      validation_interval_hours: 24
```

## 🚦 Migration Strategy

### From Current System

1. **Compatibility Mode**: Run alongside existing `super_admin` flag and `authenticated` checks
2. **Gradual Migration**: Migrate permissions module by module
3. **Rollback Plan**: Feature flag to disable and revert
4. **Data Migration**: Script to convert existing permissions

**Important**: The `super_admin` boolean flag in user_profiles takes precedence over all group permissions. Users with this flag bypass all group checks.

### Migration Steps

```sql
-- Step 1: Create administrator group
-- Step 2: Create authenticated group  
-- Step 3: Auto-assign authenticated users
-- Step 4: Enable group middleware
-- Step 5: Update middleware to check super_admin flag first
-- Step 6: Remove old permission checks
```

## 📚 Additional Resources

- [EVE Online ESI Documentation](https://esi.evetech.net/)
- [Redis Caching Best Practices](https://redis.io/docs/manual/patterns/)
- [MongoDB Index Strategies](https://docs.mongodb.com/manual/indexes/)
- [RBAC Design Patterns](https://www.osohq.com/academy/rbac-patterns)

## 🎓 Examples

### Checking Permissions

```go
// Simple permission check
if can, _ := groups.CanAccess(ctx, characterID, "scheduler:tasks", "create"); can {
    // Create the task
}

// With detailed error handling
can, err := groups.CanAccess(ctx, characterID, "scheduler:tasks", "create")
if err != nil {
    log.Error("Permission check failed", "error", err)
    return handlers.ErrorResponse(500, "Permission system error")
}
if !can {
    return handlers.ErrorResponse(403, "Insufficient permissions")
}
```

### Middleware Implementation

```go
// Character Context Middleware
func CharacterContextMiddleware(userService *users.Service) func(next http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get character ID from JWT claims
            claims := r.Context().Value("claims").(*JWTClaims)
            
            // Load user profile
            user, err := userService.GetByCharacterID(r.Context(), claims.CharacterID)
            if err != nil {
                http.Error(w, "Failed to load user profile", http.StatusInternalServerError)
                return
            }
            
            // Build character context
            charCtx := &CharacterContext{
                CharacterID:   claims.CharacterID,
                UserID:        user.ID,
                CorporationID: claims.CorporationID,
                AllianceID:    claims.AllianceID,
                SuperAdmin:    user.SuperAdmin,        // From user_profiles collection
                BypassPerms:   user.SuperAdmin,        // Set bypass flag if super_admin
            }
            
            // Attach to request context
            ctx := context.WithValue(r.Context(), "character_context", charCtx)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Usage in route protection
router.Get("/admin/users", 
    middleware.RequireAuth(),
    middleware.CharacterContext(),    // Resolves super_admin flag
    middleware.RequireGroup("administrator"),
    handlers.ListUsers,
)

// Multiple group options (OR logic)
router.Post("/fleets/create",
    middleware.RequireAuth(),
    middleware.CharacterContext(),
    middleware.RequireAnyGroup("fleet_commander", "administrator"),
    handlers.CreateFleet,
)

// Custom permission check
router.Delete("/tasks/{id}",
    middleware.RequireAuth(),
    middleware.CharacterContext(),
    middleware.RequirePermission("scheduler:tasks", "delete"),
    handlers.DeleteTask,
)
```

### Managing Groups

```go
// Create a new group
group := &Group{
    Name:        "special_ops",
    DisplayName: "Special Operations",
    Type:        "custom",
    Description: "Access to special operations tools",
    DiscordRoleID: "123456789", // Optional Discord role mapping
}
created, err := groupService.Create(ctx, group)

// Add members in bulk (triggers Discord bot events)
members := []GroupMember{
    {MemberType: "character", MemberID: "95123456"},
    {MemberType: "corporation", MemberID: "98000001"},
}
err = groupService.AddMembers(ctx, groupID, members)

// Grant permissions
permission := &Permission{
    Resource: "intel:reports",
    Action:   "*",
    Effect:   "allow",
}
err = groupService.GrantPermission(ctx, groupID, permission)
```

### Discord Bot Event Publishing

```go
// Service publishes events to Redis for Discord bot
func (s *GroupService) AddMember(ctx context.Context, groupID string, member GroupMember) error {
    // 1. Add member to database
    if err := s.repo.AddMember(ctx, groupID, member); err != nil {
        return err
    }
    
    // 2. Get group details for Discord role mapping
    group, _ := s.repo.GetGroup(ctx, groupID)
    
    // 3. Publish event to Redis for Discord bot
    if group.DiscordRoleID != "" {
        event := map[string]interface{}{
            "type":           "group:member:added",
            "group_id":       groupID,
            "group_name":     group.Name,
            "discord_role":   group.DiscordRoleID,
            "character_id":   member.MemberID,
            "timestamp":      time.Now().Unix(),
        }
        
        eventJSON, _ := json.Marshal(event)
        s.redis.Publish(ctx, "discord:role:events", eventJSON)
    }
    
    return nil
}

// Discord bot (separate service) subscribes to events:
// - Listens to "discord:role:events" channel
// - Maps EVE character to Discord user
// - Applies/removes Discord roles based on events
```

## 🤝 Contributing

When contributing to the groups module:

1. Follow the standardized module structure
2. Include comprehensive tests
3. Update this documentation
4. Consider performance impact
5. Maintain backward compatibility
6. Add audit logs for changes

## 📅 Maintenance

### Daily Tasks
- Monitor cache hit rates
- Check permission check latency
- Review error logs

### Weekly Tasks
- Validate ESI memberships
- Review audit logs
- Check for orphaned permissions

### Monthly Tasks
- Performance profiling
- Security audit
- Documentation updates
- Database optimization

---

*This module is designed for production use in EVE Online community applications, providing enterprise-grade permission management with game-specific features.*