# CASBIN Authorization Implementation Guide

## Overview

This document provides a comprehensive guide to the CASBIN-based hierarchical authorization system implemented in Go Falcon. The system provides fine-grained permission control across EVE Online's three-tier structure: Alliance → Corporation → Character/Member.

## 🏗️ Architecture

### Core Components

1. **CasbinAuthMiddleware**: Core CASBIN enforcement engine
2. **CasbinService**: High-level service layer for policy management
3. **CasbinEnhancedMiddleware**: Integration with existing authentication
4. **CachedCasbinService**: Redis-based caching for performance
5. **CasbinAPIHandler**: HTTP endpoints for policy management
6. **CasbinMiddlewareFactory**: Easy setup and configuration

### Integration Flow

```
HTTP Request
    ↓
Authentication Middleware (JWT validation)
    ↓
Character Resolution Middleware (resolve all user's characters/corps/alliances)
    ↓
CASBIN Authorization Middleware (hierarchical permission checking)
    ↓
Handler (access granted)
```

## 🚀 Quick Start

### 1. Basic Setup

```go
// Create enhanced middleware with CASBIN
factory, err := middleware.NewCasbinMiddlewareFactory(
    jwtValidator,        // Your JWT validator
    characterResolver,   // Your character resolver
    mongoClient,        // MongoDB client
    "falcon_db",        // Database name
)
if err != nil {
    log.Fatal(err)
}

// Get convenience middleware
convenience := factory.GetConvenience()
```

### 2. Route Protection

```go
// Chi router example
r := chi.NewRouter()

// Public routes (no auth required)
r.Group(func(r chi.Router) {
    r.Use(convenience.OptionalAuth())
    r.Get("/health", healthHandler)
    r.Get("/public/info", publicInfoHandler)
})

// User routes (authentication required)
r.Group(func(r chi.Router) {
    r.Use(convenience.RequireAuth())
    r.Get("/profile", profileHandler)
    r.Post("/profile/update", updateProfileHandler)
})

// Admin routes (admin permission required)
r.Group(func(r chi.Router) {
    r.Use(convenience.AdminOnly())
    r.Get("/admin/users", listUsersHandler)
    r.Post("/admin/users/{userID}/ban", banUserHandler)
})

// Module-specific permissions
r.Group(func(r chi.Router) {
    r.Use(convenience.ModuleAccess("scheduler", "admin"))
    r.Get("/scheduler/admin", schedulerAdminHandler)
    r.Post("/scheduler/tasks/create", createTaskHandler)
})
```

### 3. Permission Management API

```go
// Register CASBIN management endpoints
apiHandler := factory.GetAPIHandler()
apiHandler.RegisterRoutes(r)

// Available endpoints:
// POST   /admin/permissions/policies     - Create policy
// GET    /admin/permissions/policies     - List policies
// DELETE /admin/permissions/policies/:id - Delete policy
// POST   /admin/permissions/roles        - Assign role
// GET    /admin/permissions/roles        - List roles
// POST   /admin/permissions/check        - Check permission
// GET    /admin/permissions/users/:id/effective - Get user permissions
```

## 🔐 Permission Model

### Subject Types

- **user:UUID** - Individual user account
- **character:CharacterID** - EVE Online character
- **corporation:CorporationID** - EVE Online corporation
- **alliance:AllianceID** - EVE Online alliance
- **role:RoleName** - Named role (collection of permissions)

### Resource.Action Pattern

Permissions follow the format: `service.resource.action`

```
Examples:
- scheduler.tasks.read           # Read scheduled tasks
- scheduler.tasks.write          # Modify scheduled tasks
- scheduler.tasks.admin          # Full task administration
- users.profiles.read            # Read user profiles
- users.profiles.write           # Modify user profiles
- users.profiles.admin           # Full user administration
- system.admin                   # System administration
- system.super_admin             # Super admin privileges
```

### Hierarchical Resolution

Permission checks follow priority order:

1. **Explicit Denials** (highest priority - overrides all allows)
2. **Character-level permissions** (individual character)
3. **Corporation-level permissions** (all corp members inherit)
4. **Alliance-level permissions** (all alliance members inherit)
5. **Default deny** (no permission granted)

## 📊 Usage Examples

### 1. Grant Permission to User

```bash
curl -X POST http://localhost:8080/admin/permissions/policies \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subject_type": "user",
    "subject_id": "user-uuid-123",
    "resource": "scheduler.tasks",
    "action": "admin",
    "effect": "allow",
    "reason": "Granting scheduler admin access"
  }'
```

### 2. Grant Corporation-wide Permission

```bash
curl -X POST http://localhost:8080/admin/permissions/policies \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subject_type": "corporation",
    "subject_id": "98765432",
    "resource": "users.profiles",
    "action": "read",
    "effect": "allow",
    "reason": "Allow corp members to view profiles"
  }'
```

### 3. Assign Role to User

```bash
curl -X POST http://localhost:8080/admin/permissions/roles \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_name": "admin",
    "subject_type": "user",
    "subject_id": "user-uuid-123",
    "reason": "Promoting to admin"
  }'
```

### 4. Check User Permissions

```bash
curl -X POST http://localhost:8080/admin/permissions/check \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-uuid-123",
    "resource": "scheduler.tasks",
    "action": "admin"
  }'
```

### 5. Get Effective Permissions

```bash
curl -X GET http://localhost:8080/admin/permissions/users/user-uuid-123/effective \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 🎯 Middleware Usage Patterns

### Basic Authentication

```go
r.Use(convenience.RequireAuth())
```

### Authentication with Character Resolution

```go
r.Use(convenience.RequireAuthWithCharacters())
```

### Specific Permission Required

```go
r.Use(convenience.RequirePermission("scheduler.tasks", "admin"))
```

### Optional Permission (if authenticated)

```go
r.Use(convenience.OptionalPermission("users.profiles", "read"))
```

### Module-specific Access

```go
r.Use(convenience.ModuleAccess("scheduler", "admin"))
r.Use(convenience.CorporationAccess("sensitive_data"))
r.Use(convenience.AllianceAccess("alliance_data"))
```

### Admin Only Routes

```go
r.Use(convenience.AdminOnly())           // Requires system.admin
r.Use(convenience.SuperAdminOnly())      // Requires system.super_admin
```

## ⚡ Performance Features

### 1. Redis Caching

```go
// Enable caching during setup
factory, err := middleware.ProductionCasbinSetup(
    jwtValidator,
    characterResolver,
    mongoClient,
    "falcon_db",
)

// Configure cache
cacheConfig := middleware.DefaultCacheConfig()
cacheConfig.TTL = 10 * time.Minute
cacheConfig.HierarchyTTL = 30 * time.Minute

cachedService := middleware.NewCachedCasbinService(
    factory.GetCasbinService(),
    redisClient,
    cacheConfig,
)
```

### 2. Cache Management

```go
// Invalidate specific user's cache
cachedService.InvalidateUserCaches(ctx, "user-uuid-123")

// Invalidate all caches
cachedService.InvalidateAllCaches(ctx)

// Get cache statistics
stats, err := cachedService.GetCacheStats(ctx)

// Warmup user cache with common permissions
commonPerms := []string{
    "users.profiles.read",
    "scheduler.tasks.read",
    "auth.profile.read",
}
cachedService.WarmupUserCache(ctx, "user-uuid-123", commonPerms)
```

## 🔧 Configuration

### Environment Variables

```bash
# MongoDB Configuration
MONGODB_URI="mongodb://localhost:27017"
MONGODB_DATABASE="falcon_db"

# Redis Configuration (for caching)
REDIS_ADDR="localhost:6379"
REDIS_PASSWORD=""
REDIS_DB=0

# CASBIN Configuration
CASBIN_MODEL_PATH="configs/casbin_model.conf"
CASBIN_ENABLE_CACHE=true
CASBIN_CACHE_TTL="5m"
```

### Policy Model Configuration

The CASBIN model is defined in `configs/casbin_model.conf`:

```ini
[request_definition]
r = sub, obj, act, dom

[policy_definition]
p = sub, obj, act, dom, eft

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.obj == p.obj && r.act == p.act && r.dom == p.dom
```

## 🚨 Security Best Practices

### 1. Permission Design

```go
// ✅ Good: Specific permissions
"scheduler.tasks.read"
"scheduler.tasks.write"
"scheduler.tasks.delete"
"scheduler.tasks.admin"

// ❌ Bad: Overly broad permissions
"scheduler.*"
"*.*"
```

### 2. Default Deny

```go
// ✅ Good: Explicit allows only
middleware.AddPolicy("user:123", "scheduler.tasks", "read", "allow")

// ❌ Bad: Never use wildcards for production
middleware.AddPolicy("user:123", "*", "*", "allow")
```

### 3. Explicit Denials

```go
// ✅ Good: Use explicit denials for overrides
middleware.AddPolicy("user:123", "sensitive.data", "read", "deny")
```

### 4. Regular Auditing

```go
// Regular audit of permissions
auditLogs, err := apiHandler.GetAuditLogs(ctx, filters)

// Monitor permission usage
stats, err := cachedService.GetCacheStats(ctx)
```

## 🧪 Testing

### Unit Tests

```bash
# Run CASBIN tests
go test ./pkg/middleware/... -v

# Run specific test
go test ./pkg/middleware/ -run TestCasbinAuthMiddleware_PolicyManagement -v

# Run benchmarks
go test ./pkg/middleware/ -bench=. -benchmem
```

### Integration Tests

```go
// Test complete permission flow
func TestCompletePermissionFlow(t *testing.T) {
    // Setup test factory
    factory, err := middleware.QuickCasbinSetup(...)
    
    // Grant permission
    err = factory.GetCasbinService().GrantPermission(...)
    
    // Test middleware
    middleware := factory.GetConvenience().RequirePermission("resource", "action")
    
    // Test request
    req := httptest.NewRequest("GET", "/test", nil)
    // Add auth headers...
    
    rr := httptest.NewRecorder()
    handler := middleware(testHandler)
    handler.ServeHTTP(rr, req)
    
    assert.Equal(t, http.StatusOK, rr.Code)
}
```

## 🔍 Monitoring and Debugging

### Debug Logging

Enable debug logging to see permission evaluation:

```bash
ENABLE_TELEMETRY=true go run cmd/falcon/main.go
```

Debug output includes:
- Permission check decisions
- Subject resolution
- Cache hits/misses
- Policy evaluations

### Metrics Collection

```go
// Get permission statistics
stats, err := factory.GetCasbinService().GetCacheStats(ctx)

// Monitor permission check frequency
// Monitor cache hit rates
// Monitor policy changes
```

### Common Issues

1. **Permission Denied Unexpectedly**
   - Check subject building in logs
   - Verify user's character/corp/alliance data
   - Check for explicit denials

2. **Performance Issues**
   - Enable Redis caching
   - Check cache hit rates
   - Consider warmup strategies

3. **Policy Not Working**
   - Verify policy syntax
   - Check domain matching
   - Ensure policy is active

## 📈 Advanced Usage

### Custom Subject Types

```go
// Add custom subject types for specific use cases
middleware.AddPolicy("fleet:12345", "fleet.command", "admin", "allow")
```

### Time-based Permissions

```go
// Assign role with expiration
request := &middleware.RoleCreateRequest{
    RoleName:    "temporary_admin",
    SubjectType: "user",
    SubjectID:   "user-123",
    ExpiresAt:   &time.Now().Add(24 * time.Hour),
    Reason:      "Temporary admin access for maintenance",
}
```

### Batch Operations

```go
// Check multiple permissions at once
batchRequest := &middleware.BatchPermissionCheckRequest{
    UserID: "user-123",
    Permissions: []middleware.PermissionCheckRequest{
        {Resource: "scheduler.tasks", Action: "read"},
        {Resource: "users.profiles", Action: "write"},
        {Resource: "system", Action: "admin"},
    },
}
```

## 🤝 Contributing

### Adding New Features

1. **New Permission Types**: Add to the resource.action pattern
2. **New Subject Types**: Extend subject building logic
3. **New Middleware**: Follow existing patterns in convenience middleware
4. **New Caching**: Extend cache keys and invalidation logic

### Code Standards

- All permission changes must be audited
- All new middleware must have tests
- Performance impact must be measured
- Documentation must be updated

This comprehensive implementation provides a robust, scalable, and secure authorization system that integrates seamlessly with the existing Go Falcon architecture while supporting EVE Online's complex organizational hierarchy.