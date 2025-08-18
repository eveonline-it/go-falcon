# Role Assignment Usage Examples

## Quick Start: Grant scheduler.read Permission to User

Here are the practical examples for granting `scheduler.read` permission to a user so they can access `/scheduler/status`:

### Method 1: Direct User Permission (Quick)

```bash
# Grant scheduler.read directly to user
curl -X POST http://localhost:8080/api/admin/policies/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "subject": "user:uuid-12345",
    "resource": "scheduler",
    "action": "read",
    "effect": "allow"
  }'
```

### Method 2: Role-Based Assignment (Recommended)

```bash
# Step 1: Assign monitoring role to user
curl -X POST http://localhost:8080/api/admin/roles/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "user_id": "uuid-12345",
    "role": "monitoring"
  }'

# The monitoring role already has scheduler.read permission
```

### Method 3: Character-Specific Permission

```bash
# Grant permission to specific character
curl -X POST http://localhost:8080/api/admin/roles/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "user_id": "uuid-12345",
    "character_id": 123456789,
    "role": "monitoring"
  }'
```

## Complete Workflow Example

### 1. Setup Initial Admin

```bash
# First, you need an admin user to manage permissions
# This is typically done during system initialization
curl -X POST http://localhost:8080/api/admin/roles/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <bootstrap_token>" \
  -d '{
    "user_id": "bootstrap-admin-uuid",
    "role": "admin"
  }'
```

### 2. Create Role-Based Permissions

```bash
# Assign monitoring role (which includes scheduler.read)
curl -X POST http://localhost:8080/api/admin/roles/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "user_id": "regular-user-uuid", 
    "role": "monitoring"
  }'
```

### 3. Verify Permission

```bash
# Check if user has scheduler.read permission
curl -X POST http://localhost:8080/api/permissions/check \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "user_id": "regular-user-uuid",
    "resource": "scheduler",
    "action": "read"
  }'

# Expected response:
# {
#   "has_permission": true,
#   "user_id": "regular-user-uuid", 
#   "resource": "scheduler",
#   "action": "read",
#   "matched_rules": ["user:regular-user-uuid -> scheduler.read (allow)"],
#   "checked_at": "2024-08-18T21:30:00Z"
# }
```

### 4. Test Access

```bash
# User should now be able to access scheduler status
curl -H "Authorization: Bearer <user_token>" \
  http://localhost:8080/api/scheduler/status

# Should return 200 OK instead of 403 Forbidden
```

## Bulk Operations Example

### Assign Role to Multiple Users

```bash
# Assign monitoring role to multiple users at once
curl -X POST http://localhost:8080/api/admin/roles/bulk-assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "user_ids": [
      "user-uuid-1",
      "user-uuid-2", 
      "user-uuid-3",
      "user-uuid-4"
    ],
    "role": "monitoring"
  }'

# Response shows success/failure for each user:
# {
#   "success": ["user-uuid-1", "user-uuid-2", "user-uuid-3"],
#   "failed": ["user-uuid-4"],
#   "total": 4,
#   "success_count": 3,
#   "failure_count": 1,
#   "role": "monitoring",
#   "processed_at": "2024-08-18T21:30:00Z"
# }
```

## Advanced Permission Management

### Corporation-Level Permissions

```bash
# Grant scheduler access to entire corporation
curl -X POST http://localhost:8080/api/admin/policies/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "subject": "corporation:98765432",
    "resource": "scheduler",
    "action": "read",
    "effect": "allow"
  }'
```

### Custom Role Creation

```bash
# Create a custom role for task managers
# Step 1: Assign permissions to role
curl -X POST http://localhost:8080/api/admin/policies/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "subject": "role:task_manager",
    "resource": "scheduler",
    "action": "read",
    "effect": "allow"
  }'

curl -X POST http://localhost:8080/api/admin/policies/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "subject": "role:task_manager",
    "resource": "scheduler",
    "action": "write",
    "effect": "allow"
  }'

# Step 2: Assign role to users
curl -X POST http://localhost:8080/api/admin/roles/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "user_id": "task-manager-uuid",
    "role": "task_manager"
  }'
```

## Auditing and Monitoring

### Check User's Roles and Permissions

```bash
# Get all roles for a user
curl -H "Authorization: Bearer <admin_token>" \
  http://localhost:8080/api/users/uuid-12345/roles

# Check specific permission
curl -X POST http://localhost:8080/api/permissions/check \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "user_id": "uuid-12345",
    "resource": "scheduler", 
    "action": "read"
  }'
```

### System Overview

```bash
# List all roles in system
curl -H "Authorization: Bearer <admin_token>" \
  http://localhost:8080/api/admin/roles

# List all policies in system  
curl -H "Authorization: Bearer <admin_token>" \
  http://localhost:8080/api/admin/policies
```

## Integration with Go Code

### Using the Integration Helper

```go
package main

import (
    "context"
    "log"
    
    "go-falcon/pkg/middleware"
)

func setupUserPermissions() {
    // Initialize CASBIN integration
    integration := middleware.NewCasbinIntegration(enforcer, authChecker)
    
    ctx := context.Background()
    
    // Setup predefined roles
    if err := integration.SetupInitialRoles(ctx); err != nil {
        log.Fatal("Failed to setup roles:", err)
    }
    
    // Grant scheduler access to user (Method 1: Direct)
    err := integration.GrantSchedulerReadPermission(ctx, "uuid-12345")
    if err != nil {
        log.Printf("Failed to grant direct permission: %v", err)
    }
    
    // Grant scheduler access to user (Method 2: Role-based)
    err = integration.GrantSchedulerReadPermissionViaRole(ctx, "uuid-67890")
    if err != nil {
        log.Printf("Failed to grant role-based permission: %v", err)
    }
    
    // Quick setup with role
    err = integration.QuickSetupForUser(ctx, "uuid-11111", "monitoring")
    if err != nil {
        log.Printf("Failed quick setup: %v", err)
    }
    
    // Character-specific role
    err = integration.QuickSetupForCharacter(ctx, "uuid-22222", 123456789, "scheduler_manager")
    if err != nil {
        log.Printf("Failed character setup: %v", err)
    }
}
```

## Common Troubleshooting

### Problem: 403 Forbidden on /scheduler/status

```bash
# Step 1: Check if user exists and is authenticated
curl -H "Authorization: Bearer <user_token>" \
  http://localhost:8080/api/auth/profile

# Step 2: Check if user has scheduler.read permission
curl -X POST http://localhost:8080/api/permissions/check \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "user_id": "uuid-from-profile",
    "resource": "scheduler",
    "action": "read"
  }'

# Step 3: If no permission, grant it
curl -X POST http://localhost:8080/api/admin/roles/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin_token>" \
  -d '{
    "user_id": "uuid-from-profile",
    "role": "monitoring"
  }'
```

### Problem: Role Assignment Failed

```bash
# Check if role exists
curl -H "Authorization: Bearer <admin_token>" \
  http://localhost:8080/api/admin/roles

# Check if user ID is correct format
# Should be: user:uuid-12345, not just uuid-12345
```

## Environment-Specific Examples

### Development Environment

```bash
# For local development, you might use localhost
BASE_URL="http://localhost:8080/api"

curl -X POST $BASE_URL/admin/roles/assign \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "dev-user-123",
    "role": "admin"
  }'
```

### Production Environment

```bash
# For production, use your actual domain and require proper auth
BASE_URL="https://api.yourdomain.com/api"

curl -X POST $BASE_URL/admin/roles/assign \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -d '{
    "user_id": "prod-user-uuid-12345",
    "role": "monitoring"
  }'
```

These examples provide a complete guide for implementing role-based permissions to grant `scheduler.read` access and manage the broader permission system.