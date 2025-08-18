# Role-Based Assignment API

## Overview

This document describes the comprehensive role-based assignment API endpoints implemented using HUMA v2 and CASBIN for the Go Falcon API server. These endpoints provide fine-grained permission management through roles, policies, and hierarchical permission checking.

## 🏗️ Architecture

### Core Components

- **CASBIN Enforcer**: Policy enforcement engine with MongoDB adapter
- **Role Assignment Service**: Business logic for role and policy operations  
- **HUMA Routes**: Type-safe API endpoints with automatic OpenAPI generation
- **DTOs**: Comprehensive input/output data transfer objects with validation
- **Integration Example**: Complete setup and usage examples

## 🔐 Permission System

### Hierarchical Subject Model

CASBIN checks permissions in this priority order:
1. **Character Level**: `character:123456` (highest priority)
2. **User Level**: `user:uuid-12345`  
3. **Corporation Level**: `corporation:987654`
4. **Alliance Level**: `alliance:456789` (lowest priority)

### Policy Format

CASBIN uses a 5-field policy model: `(subject, object, action, domain, effect)`

```
Example: user:12345, scheduler.read, read, global, allow
```

## 📋 API Endpoints

All endpoints are available with automatic OpenAPI 3.1.1 documentation at `/openapi.json`.

### 🎯 Role Management Endpoints

#### Assign Role to User/Character
```http
POST /admin/roles/assign
Content-Type: application/json

{
  "user_id": "uuid-12345",
  "character_id": 123456,  // optional
  "role": "monitoring", 
  "domain": "global"       // optional, defaults to "global"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Role 'monitoring' successfully assigned to user:uuid-12345",
  "user_id": "uuid-12345",
  "role": "monitoring",
  "domain": "global",
  "timestamp": "2024-08-18T21:30:00Z"
}
```

#### Remove Role from User/Character
```http
DELETE /admin/roles/remove
Content-Type: application/json

{
  "user_id": "uuid-12345",
  "character_id": 123456,  // optional
  "role": "monitoring",
  "domain": "global"       // optional
}
```

#### Bulk Assign Role to Multiple Users
```http
POST /admin/roles/bulk-assign
Content-Type: application/json

{
  "user_ids": ["uuid-12345", "uuid-67890", "uuid-11111"],
  "role": "monitoring",
  "domain": "global"       // optional
}
```

**Response:**
```json
{
  "success": ["uuid-12345", "uuid-67890"],
  "failed": ["uuid-11111"],
  "total": 3,
  "success_count": 2,
  "failure_count": 1,
  "role": "monitoring",
  "processed_at": "2024-08-18T21:30:00Z"
}
```

### 🔑 Policy Management Endpoints

#### Assign Permission Policy
```http
POST /admin/policies/assign
Content-Type: application/json

{
  "subject": "role:monitoring",     // or user:id, character:id, etc.
  "resource": "scheduler",
  "action": "read",
  "domain": "global",               // optional
  "effect": "allow"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Permission 'scheduler.read' (allow) successfully assigned to role:monitoring",
  "subject": "role:monitoring",
  "resource": "scheduler", 
  "action": "read",
  "effect": "allow",
  "timestamp": "2024-08-18T21:30:00Z"
}
```

#### Remove Permission Policy
```http
DELETE /admin/policies/remove
Content-Type: application/json

{
  "subject": "role:monitoring",
  "resource": "scheduler",
  "action": "read", 
  "domain": "global",
  "effect": "allow"
}
```

### 🔍 Permission Checking Endpoints

#### Check User Permission
```http
POST /permissions/check
Content-Type: application/json

{
  "user_id": "uuid-12345",
  "character_id": 123456,          // optional
  "resource": "scheduler",
  "action": "read",
  "domain": "global"               // optional
}
```

**Response:**
```json
{
  "has_permission": true,
  "user_id": "uuid-12345",
  "character_id": 123456,
  "resource": "scheduler",
  "action": "read",
  "matched_rules": [
    "character:123456 -> scheduler.read (allow)"
  ],
  "checked_at": "2024-08-18T21:30:00Z"
}
```

### 📊 Information Endpoints

#### Get User Roles
```http
GET /users/{user_id}/roles
```

**Response:**
```json
{
  "user_id": "uuid-12345",
  "roles": [
    {
      "role": "monitoring",
      "domain": "global"
    },
    {
      "role": "scheduler_manager", 
      "domain": "global"
    }
  ],
  "total": 2
}
```

#### Get Role Policies
```http
GET /roles/{role}/policies
```

**Response:**
```json
{
  "role": "monitoring",
  "policies": [
    {
      "subject": "role:monitoring",
      "resource": "scheduler.read",
      "action": "read", 
      "domain": "global",
      "effect": "allow"
    }
  ],
  "total": 1
}
```

#### List All Policies
```http
GET /admin/policies
```

#### List All Roles  
```http
GET /admin/roles
```

## 🚀 Usage Examples

### Quick Setup for Scheduler Access

```bash
# Method 1: Direct Permission Assignment
curl -X POST /admin/policies/assign \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "user:uuid-12345",
    "resource": "scheduler",
    "action": "read",
    "effect": "allow"
  }'

# Method 2: Role-Based Assignment (Recommended)
curl -X POST /admin/roles/assign \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "uuid-12345", 
    "role": "monitoring"
  }'
```

### Character-Specific Permissions
```bash
# Assign role to specific character
curl -X POST /admin/roles/assign \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "uuid-12345",
    "character_id": 123456,
    "role": "scheduler_manager"
  }'
```

### Bulk Operations
```bash
# Assign monitoring role to multiple users
curl -X POST /admin/roles/bulk-assign \
  -H "Content-Type: application/json" \
  -d '{
    "user_ids": ["uuid-12345", "uuid-67890", "uuid-11111"],
    "role": "monitoring"
  }'
```

## 🔧 Integration Guide

### 1. Setup CASBIN Integration

```go
package main

import (
    "context"
    "log"
    
    "go-falcon/pkg/middleware"
    "github.com/casbin/casbin/v2"
)

func main() {
    // Initialize CASBIN enforcer (configure with your adapter)
    enforcer, err := casbin.NewEnforcer("configs/casbin_model.conf", "configs/casbin_policy.csv")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create CASBIN auth middleware
    authChecker := middleware.NewCasbinAuthMiddleware(enforcer, true)
    
    // Create integration
    integration := middleware.NewCasbinIntegration(enforcer, authChecker)
    
    // Setup initial roles and policies
    ctx := context.Background()
    if err := integration.SetupInitialRoles(ctx); err != nil {
        log.Fatal("Failed to setup initial roles:", err)
    }
    
    // Register API routes (example with HUMA)
    api := // ... your HUMA API instance
    integration.RegisterRoleManagementAPI(api, "/api")
}
```

### 2. Grant Scheduler Access to User

```go
// Quick helper methods
integration := middleware.NewCasbinIntegration(enforcer, authChecker)

// Method 1: Direct permission
err := integration.GrantSchedulerReadPermission(ctx, "uuid-12345")

// Method 2: Via role assignment (recommended)
err := integration.GrantSchedulerReadPermissionViaRole(ctx, "uuid-12345")

// Method 3: Full setup with role
err := integration.QuickSetupForUser(ctx, "uuid-12345", "monitoring")
```

### 3. Available Predefined Roles

The system comes with these predefined roles:

#### Admin Role
```json
{
  "role": "admin",
  "permissions": [
    "scheduler.read", "scheduler.admin",
    "users.read", "users.admin", 
    "roles.read", "roles.admin",
    "policies.read", "policies.admin"
  ]
}
```

#### Monitoring Role
```json
{
  "role": "monitoring", 
  "permissions": [
    "scheduler.read",
    "users.read"
  ]
}
```

#### Scheduler Manager Role
```json
{
  "role": "scheduler_manager",
  "permissions": [
    "scheduler.read",
    "scheduler.write", 
    "scheduler.delete"
  ]
}
```

## 🎯 Common Use Cases

### 1. Grant Access to /scheduler/status

The `/scheduler/status` endpoint requires `scheduler.read` permission:

```bash
# Option A: Direct user permission
curl -X POST /admin/policies/assign \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "user:uuid-12345",
    "resource": "scheduler",
    "action": "read",
    "effect": "allow"
  }'

# Option B: Role assignment (better for management)
curl -X POST /admin/roles/assign \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "uuid-12345",
    "role": "monitoring"
  }'
```

### 2. Corporation-Level Permissions

```bash
# Grant scheduler access to entire corporation
curl -X POST /admin/policies/assign \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "corporation:98765432",
    "resource": "scheduler", 
    "action": "read",
    "effect": "allow"
  }'
```

### 3. Temporary Admin Access

```bash
# Assign admin role
curl -X POST /admin/roles/assign \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "uuid-12345",
    "role": "admin"
  }'

# Later remove admin role
curl -X DELETE /admin/roles/remove \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "uuid-12345", 
    "role": "admin"
  }'
```

## 🔒 Security Considerations

### Admin Protection
- All `/admin/*` endpoints require admin-level permissions
- Role assignment operations are logged and audited
- Bulk operations have limits (max 100 users per request)

### Permission Hierarchy
- Character permissions override user permissions
- Role permissions are inherited by all subjects with that role
- Deny policies take precedence over allow policies

### Input Validation
- All inputs validated with HUMA v2 validation tags
- Subject IDs must follow format: `type:identifier`
- Resource and action names are validated against allowed patterns

## 📊 Monitoring and Debugging

### Check Permission Status
```bash
# Verify user has required permission
curl -X POST /permissions/check \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "uuid-12345",
    "resource": "scheduler",
    "action": "read"
  }'
```

### Audit User Permissions
```bash
# Get all roles for user
curl /users/uuid-12345/roles

# Get all policies for role  
curl /roles/monitoring/policies

# List all system roles
curl /admin/roles

# List all system policies
curl /admin/policies
```

## ⚡ Performance Considerations

### Caching
- CASBIN policies are cached in memory for fast access
- Role assignments are cached for improved performance
- Cache invalidation on policy/role changes

### Batch Operations
- Use bulk assignment for multiple users
- Group related permission changes in single transactions
- Monitor CASBIN performance with large policy sets

## 🔧 Troubleshooting

### Common Issues

1. **403 Forbidden on /scheduler/status**
   ```bash
   # Check if user has scheduler.read permission
   curl -X POST /permissions/check \
     -H "Content-Type: application/json" \
     -d '{"user_id": "uuid-12345", "resource": "scheduler", "action": "read"}'
   ```

2. **Role Assignment Not Working**
   ```bash
   # Verify role exists and has policies
   curl /roles/monitoring/policies
   ```

3. **Permission Check Returns False**
   ```bash
   # Check user roles
   curl /users/uuid-12345/roles
   
   # Verify policy exists
   curl /admin/policies
   ```

## 🚀 Future Enhancements

### Planned Features
- **Time-based permissions**: Roles with expiration dates
- **Conditional policies**: Context-aware permission checking  
- **Advanced audit logging**: Comprehensive permission change tracking
- **Permission templates**: Predefined permission sets for common roles
- **UI Management**: Web interface for permission management

### API Extensions
- **Permission inheritance**: Corporation/alliance permission inheritance
- **Batch permission checking**: Multiple permissions in single request
- **Permission comparison**: Compare permissions between users
- **Permission history**: Track permission changes over time

This comprehensive role-based assignment API provides enterprise-grade permission management for the Go Falcon system while maintaining simplicity and performance.