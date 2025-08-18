# CASBIN Authorization Integration Examples

This document shows how to integrate CASBIN authorization into existing Go Falcon modules.

## Current Implementation Status

✅ **CASBIN is now integrated into the main application startup**
- CASBIN middleware factory is initialized in `cmd/falcon/main.go`
- Initial roles and permissions are automatically configured
- CASBIN management API is available at `/admin/permissions/*`
- Character resolution is connected to auth module

## How to Apply CASBIN to Routes

### Method 1: Using Chi Middleware (Recommended for New Routes)

For routes using Chi router directly:

```go
// In your module's route registration
func (m *Module) RegisterRoutesWithCasbin(r chi.Router, casbin *middleware.CasbinConvenienceMiddleware) {
    // Public routes - no auth required
    r.Get("/public/info", publicHandler)
    
    // Authenticated routes - require login only
    r.Group(func(r chi.Router) {
        r.Use(casbin.RequireAuth())
        r.Get("/profile", profileHandler)
    })
    
    // Permission-protected routes
    r.Group(func(r chi.Router) {
        r.Use(casbin.RequirePermission("scheduler.tasks", "admin"))
        r.Post("/tasks/create", createTaskHandler)
        r.Delete("/tasks/{id}", deleteTaskHandler)
    })
    
    // Admin-only routes
    r.Group(func(r chi.Router) {
        r.Use(casbin.AdminOnly())
        r.Get("/admin/users", listUsersHandler)
    })
}
```

### Method 2: Inside Huma Handlers (Current Module Pattern)

For existing Huma routes, check permissions inside the handler:

```go
// Example: Protected endpoint in auth module
huma.Get(api, basePath+"/admin/users", func(ctx context.Context, input *dto.AdminUsersInput) (*dto.AdminUsersOutput, error) {
    // Manual permission check inside handler
    user, err := authMiddleware.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
    if err != nil {
        return nil, huma.Error401Unauthorized("Authentication required", err)
    }
    
    // Check CASBIN permission (would need to pass casbin factory to routes)
    allowed := checkCasbinPermission(user.UserID, "users", "admin")
    if !allowed {
        return nil, huma.Error403Forbidden("Insufficient permissions")
    }
    
    // Handler logic here...
})
```

## Quick Integration Guide

### Step 1: Access CASBIN Factory in Module

Since modules are already initialized, you have two options:

#### Option A: Pass CASBIN to Module Constructor (Requires Module Changes)

```go
// In main.go
usersModule := users.NewWithCasbin(
    appCtx.MongoDB, 
    appCtx.Redis, 
    appCtx.SDEService, 
    authModule, 
    casbinFactory, // Pass CASBIN factory
)
```

#### Option B: Use Global CASBIN Instance (Quick Solution)

```go
// In main.go - make CASBIN globally accessible
var GlobalCasbin *middleware.CasbinMiddlewareFactory

// After initialization
GlobalCasbin = casbinFactory

// In your module handlers
if main.GlobalCasbin != nil {
    // Use CASBIN for authorization
}
```

### Step 2: Apply Permissions to Critical Endpoints

Start with high-risk endpoints that need immediate protection:

#### Auth Module Critical Endpoints
- `/auth/admin/*` - Admin functions → Require `auth.admin`
- `/auth/profile/delete` - Account deletion → Require `auth.profile.delete`

#### Users Module Critical Endpoints  
- `/users/admin/*` - User management → Require `users.admin`
- `/users/{id}/ban` - Ban users → Require `users.ban`
- `/users/bulk/*` - Bulk operations → Require `users.bulk`

#### Scheduler Module Critical Endpoints
- `/scheduler/tasks/delete/*` - Task deletion → Require `scheduler.tasks.delete`
- `/scheduler/admin/*` - Admin functions → Require `scheduler.admin`
- `/scheduler/system/*` - System tasks → Require `scheduler.system`

## Example: Protecting an Existing Endpoint

Here's how to add CASBIN to an existing endpoint without major refactoring:

### Before (No Authorization):
```go
huma.Post(api, basePath+"/tasks/delete/{id}", func(ctx context.Context, input *dto.DeleteTaskInput) (*dto.DeleteTaskOutput, error) {
    // Direct deletion - dangerous!
    err := taskService.DeleteTask(ctx, input.ID)
    if err != nil {
        return nil, huma.Error500InternalServerError("Failed to delete task", err)
    }
    return &dto.DeleteTaskOutput{Success: true}, nil
})
```

### After (With CASBIN):
```go
huma.Post(api, basePath+"/tasks/delete/{id}", func(ctx context.Context, input *dto.DeleteTaskInput) (*dto.DeleteTaskOutput, error) {
    // 1. Validate authentication
    user, err := authMiddleware.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
    if err != nil {
        return nil, huma.Error401Unauthorized("Authentication required", err)
    }
    
    // 2. Check CASBIN permission
    // Note: This would require access to CASBIN factory
    allowed := checkPermission(user.UserID, "scheduler.tasks", "delete")
    if !allowed {
        return nil, huma.Error403Forbidden("You don't have permission to delete tasks")
    }
    
    // 3. Proceed with deletion
    err = taskService.DeleteTask(ctx, input.ID)
    if err != nil {
        return nil, huma.Error500InternalServerError("Failed to delete task", err)
    }
    
    return &dto.DeleteTaskOutput{Success: true}, nil
})
```

## Testing CASBIN Integration

### 1. Test Authentication Flow
```bash
# Login and get JWT token
curl -X GET http://localhost:8080/auth/eve/login

# Check authentication status
curl -X GET http://localhost:8080/auth/status \
  -H "Cookie: falcon_auth_token=YOUR_JWT_TOKEN"
```

### 2. Test Permission Check
```bash
# Check if user has permission (via API)
curl -X POST http://localhost:8080/admin/permissions/check \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "YOUR_USER_ID",
    "resource": "scheduler.tasks",
    "action": "admin"
  }'
```

### 3. Grant Permission
```bash
# Grant permission to a user
curl -X POST http://localhost:8080/admin/permissions/policies \
  -H "Authorization: Bearer SUPER_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subject_type": "user",
    "subject_id": "USER_ID",
    "resource": "scheduler.tasks",
    "action": "admin",
    "effect": "allow"
  }'
```

## Default Roles and Permissions

The following roles are automatically configured:

### role:super_admin
- `system.super_admin` → Full system control
- `system.admin` → All admin functions

### role:admin
- `system.admin` → General admin
- `users.*` → User management
- `scheduler.admin` → Scheduler admin
- `auth.admin` → Auth admin

### role:user
- `auth.profile.*` → Own profile management
- `users.profiles.read` → View other profiles
- `scheduler.tasks.read` → View tasks
- `scheduler.tasks.write` → Create/modify tasks

### role:corp_manager
- `corporation.admin` → Corp administration
- `users.corporation.*` → Corp user management
- `scheduler.corporation.admin` → Corp scheduler

### role:alliance_manager
- `alliance.admin` → Alliance administration
- `users.alliance.*` → Alliance user management
- `scheduler.alliance.admin` → Alliance scheduler

## Environment Variables

```bash
# Assign super admin role to a specific character
SUPER_ADMIN_CHARACTER_ID=123456789

# This character will automatically get role:super_admin on startup
```

## Migration Path

### Phase 1: Core Protection (Immediate)
1. ✅ CASBIN initialized in main.go
2. ✅ Initial roles configured
3. ✅ Management API available
4. ⏳ Protect critical admin endpoints

### Phase 2: Module Integration (Next)
1. ⏳ Update module constructors to accept CASBIN
2. ⏳ Add permission checks to sensitive operations
3. ⏳ Implement corporation/alliance-based access

### Phase 3: Full Coverage (Future)
1. ⏳ All endpoints have appropriate permissions
2. ⏳ Redis caching enabled for performance
3. ⏳ Audit logging for all permission changes
4. ⏳ Permission delegation system

## Current Limitations

1. **Modules not CASBIN-aware**: Existing modules don't have direct access to CASBIN factory
2. **Huma integration**: Need to pass CASBIN through to Huma handlers
3. **No caching yet**: Redis caching not activated (performance impact for high traffic)

## Next Steps

To fully integrate CASBIN:

1. **Update module interfaces** to accept CASBIN factory
2. **Add middleware wrapper** for Huma routes
3. **Enable Redis caching** for production performance
4. **Create permission UI** for administrators
5. **Document all permissions** for each endpoint

The CASBIN system is now ready and operational. The next step is to gradually apply it to existing endpoints based on security priorities.