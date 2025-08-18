# CASBIN Hierarchical Permission System Implementation Plan

## Overview

This document outlines the implementation plan for a CASBIN-based hierarchical permission system that manages roles and permissions across EVE Online's three-tier structure: Alliance → Corporation → Member.

## Current State Analysis

The codebase currently has:
- ✅ **Clean Slate**: Old permission system has been completely removed
- ✅ **User Management**: Full user management with character/corporation/alliance data via EVE integration  
- ✅ **Authentication**: JWT-based authentication with EVE SSO integration
- ✅ **Super Admin**: Basic super admin detection via environment variable
- 🔄 **Ready for CASBIN**: Fresh start with no legacy permission code to conflict

## 1. Hierarchical Permission Model Design

### Core Hierarchy Levels
```
Alliance Level (Broad permissions - inherited by all corp members)
├── Corporation Level (Corp-specific permissions - override/extend alliance)
    └── Member Level (Individual permissions - most granular)
```

### Permission Inheritance Rules
- **Additive Model**: Higher levels grant additional permissions to lower levels
- **Override Capability**: Lower levels can restrict permissions granted at higher levels
- **Explicit Deny**: Direct denial takes precedence over inherited grants

### CASBIN Model Structure
```ini
[request_definition]
r = sub, obj, act, dom

[policy_definition]
p = sub, obj, act, dom, eft

[role_definition]
g = _, _, _
g2 = _, _, _  # Hierarchy relationships (alliance -> corp -> member)

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = (g(r.sub, p.sub, r.dom) || g2(r.sub, p.sub, r.dom)) && r.obj == p.obj && r.act == p.act && r.dom == p.dom
```

## 2. Database Schema Design

### Core Collections

#### casbin_policies (CASBIN policy storage)
```go
type CasbinRule struct {
    ID    primitive.ObjectID `bson:"_id,omitempty"`
    PType string             `bson:"ptype"` // p, g, g2
    V0    string             `bson:"v0"`    // subject
    V1    string             `bson:"v1"`    // object/role
    V2    string             `bson:"v2"`    // action/domain
    V3    string             `bson:"v3"`    // domain/additional
    V4    string             `bson:"v4"`    // effect (allow/deny)
}
```

#### permission_hierarchies (Track EVE entity relationships)
```go
type PermissionHierarchy struct {
    ID           primitive.ObjectID `bson:"_id,omitempty"`
    AllianceID   int               `bson:"alliance_id,omitempty"`
    CorporationID int              `bson:"corporation_id,omitempty"`
    CharacterID  int               `bson:"character_id"`
    CreatedAt    time.Time         `bson:"created_at"`
    UpdatedAt    time.Time         `bson:"updated_at"`
}
```

#### role_assignments (Track role assignments at different levels)
```go
type RoleAssignment struct {
    ID          primitive.ObjectID `bson:"_id,omitempty"`
    RoleID      string            `bson:"role_id"`
    SubjectType string            `bson:"subject_type"` // alliance, corporation, member
    SubjectID   string            `bson:"subject_id"`   // actual ID
    GrantedBy   int               `bson:"granted_by"`   // character_id who granted
    GrantedAt   time.Time         `bson:"granted_at"`
    ExpiresAt   *time.Time        `bson:"expires_at,omitempty"`
    Reason      string            `bson:"reason"`
}
```

## 3. CASBIN Policy Syntax

### Policy Examples
```
# Alliance-level permissions
p, alliance:123456, scheduler.tasks, read, global, allow
p, alliance:123456, users.profiles, read, global, allow

# Corporation-level permissions (more specific)
p, corp:789012, scheduler.tasks, write, global, allow
p, corp:789012, users.profiles, admin, global, deny  # Override alliance permission

# Member-level permissions (most granular)
p, member:345678, scheduler.tasks, admin, global, allow
p, member:345678, sensitive.data, read, global, deny  # Explicit denial
```

### Hierarchy Relationships
```
# Alliance contains corporations
g2, corp:789012, alliance:123456, global

# Corporation contains members
g2, member:345678, corp:789012, global
```

## 4. Permission Evaluation Algorithm

### Multi-Level Resolution Process
1. **Collect All Applicable Policies**: Gather policies from member → corp → alliance
2. **Apply Hierarchy Rules**: Process inheritance and overrides
3. **Evaluate Explicit Denials**: Check for explicit deny policies (highest priority)
4. **Evaluate Grants**: Check for allow policies at all levels
5. **Default Behavior**: Deny if no explicit allow found

### Implementation Strategy
```go
type HierarchicalPermissionChecker struct {
    enforcer   *casbin.Enforcer
    eveClient  *evegateway.Client
    hierarchyRepo HierarchyRepository
}

func (h *HierarchicalPermissionChecker) CheckPermission(
    ctx context.Context, 
    characterID int, 
    service, resource, action string,
) (bool, error) {
    // 1. Get user's hierarchy (alliance, corp, character)
    hierarchy := h.getUserHierarchy(characterID)
    
    // 2. Check explicit denials first
    for _, level := range []string{
        fmt.Sprintf("member:%d", characterID),
        fmt.Sprintf("corp:%d", hierarchy.CorporationID),
        fmt.Sprintf("alliance:%d", hierarchy.AllianceID),
    } {
        if denied, _ := h.enforcer.Enforce(level, service+"."+resource, action, "global"); denied {
            if h.hasDenyPolicy(level, service+"."+resource, action) {
                return false, nil // Explicit deny
            }
        }
    }
    
    // 3. Check for allows at any level
    for _, level := range []string{
        fmt.Sprintf("member:%d", characterID),
        fmt.Sprintf("corp:%d", hierarchy.CorporationID), 
        fmt.Sprintf("alliance:%d", hierarchy.AllianceID),
    } {
        if allowed, _ := h.enforcer.Enforce(level, service+"."+resource, action, "global"); allowed {
            return true, nil
        }
    }
    
    return false, nil // Default deny
}
```

## 5. API Endpoints Design

### Core Permission Management
```
# Hierarchy management
GET    /admin/permissions/hierarchies/{characterID}
POST   /admin/permissions/hierarchies/sync  # Sync with EVE ESI

# Alliance-level permissions
POST   /admin/permissions/alliance/{allianceID}/roles
DELETE /admin/permissions/alliance/{allianceID}/roles/{roleID}
GET    /admin/permissions/alliance/{allianceID}/members

# Corporation-level permissions  
POST   /admin/permissions/corporation/{corpID}/roles
DELETE /admin/permissions/corporation/{corpID}/roles/{roleID}
GET    /admin/permissions/corporation/{corpID}/members

# Member-level permissions
POST   /admin/permissions/member/{characterID}/roles
DELETE /admin/permissions/member/{characterID}/roles/{roleID}
GET    /admin/permissions/member/{characterID}/effective

# Policy management
GET    /admin/permissions/policies
POST   /admin/permissions/policies
DELETE /admin/permissions/policies/{policyID}

# Permission checking
POST   /admin/permissions/check
POST   /admin/permissions/check/batch
```

## 6. Integration Points

### EVE ESI Integration
- **Character/Corp/Alliance Data**: Sync hierarchy from EVE ESI
- **Real-time Updates**: Handle corporation/alliance changes
- **Validation**: Verify EVE entity relationships

### Existing Systems Integration
- **Auth Module**: Integrate with existing JWT authentication (no changes needed)
- **Users Module**: Extend user management with hierarchy data  
- **Super Admin**: Integrate with existing super admin environment variable
- **Route Protection**: Add CASBIN middleware to existing unprotected routes

## 7. Implementation Phases

### Phase 1: Foundation (Week 1-2)
- [ ] Add CASBIN dependency to go.mod
- [ ] Implement MongoDB adapter for CASBIN
- [ ] Create core data models (CasbinRule, PermissionHierarchy, RoleAssignment)
- [ ] Set up CASBIN enforcer with custom model
- [ ] Create new hierarchical permission checker interface (clean implementation)
- [ ] Create new permission middleware to replace removed system

### Phase 2: Core Logic (Week 2-3)
- [ ] Implement HierarchicalPermissionChecker
- [ ] Create EVE hierarchy sync service using ESI
- [ ] Build policy management service layer
- [ ] Integrate new CASBIN middleware with existing routes
- [ ] Implement permission inheritance algorithm
- [ ] Add super admin bypass logic to CASBIN system

### Phase 3: Management APIs (Week 3-4)
- [ ] Implement alliance-level permission endpoints
- [ ] Implement corporation-level permission endpoints
- [ ] Implement member-level permission endpoints
- [ ] Create policy management APIs
- [ ] Add batch operations support
- [ ] Implement audit logging for all permission changes

### Phase 4: Testing & Deployment (Week 4-5)
- [ ] Write comprehensive unit tests for permission checker
- [ ] Create integration tests for API endpoints
- [ ] Performance testing and optimization
- [ ] Create initial permission policies for existing services
- [ ] Create admin documentation and training materials
- [ ] Deploy and monitor in staging environment
- [ ] Set up initial super admin and test user permissions

## 8. Security Considerations

### Core Security Principles
- **Principle of Least Privilege**: Default deny, explicit grants only
- **Explicit Denials**: Override any inherited permissions
- **Audit Trails**: Log all permission changes and evaluations
- **Validation**: Verify EVE entity relationships before granting permissions
- **Expiration**: Support time-bound role assignments
- **Emergency Override**: Super admin capabilities for crisis situations

### Implementation Details
```go
// Audit logging for all permission operations
type PermissionAuditLog struct {
    ID          primitive.ObjectID `bson:"_id,omitempty"`
    Operation   string            `bson:"operation"`   // grant, revoke, check
    SubjectType string            `bson:"subject_type"` // alliance, corp, member
    SubjectID   string            `bson:"subject_id"`
    Permission  string            `bson:"permission"`  // service.resource.action
    Result      bool              `bson:"result"`      // true/false for checks
    PerformedBy int               `bson:"performed_by"` // character_id
    Timestamp   time.Time         `bson:"timestamp"`
    IPAddress   string            `bson:"ip_address"`
    UserAgent   string            `bson:"user_agent"`
}
```

## 9. Performance Considerations

### Caching Strategy
- **Policy Cache**: Cache CASBIN policies in Redis for fast evaluation
- **Hierarchy Cache**: Cache EVE entity relationships to avoid ESI calls
- **Result Cache**: Short-term caching of permission check results
- **Invalidation**: Smart cache invalidation on policy changes

### Database Optimization
```sql
-- MongoDB indexes for optimal performance
db.casbin_policies.createIndex({ptype: 1, v0: 1, v1: 1, v2: 1})
db.permission_hierarchies.createIndex({character_id: 1})
db.permission_hierarchies.createIndex({corporation_id: 1})
db.permission_hierarchies.createIndex({alliance_id: 1})
db.role_assignments.createIndex({subject_type: 1, subject_id: 1})
```

## 10. Monitoring & Observability

### Key Metrics
- Permission check latency
- Cache hit/miss ratios  
- Policy evaluation frequency
- Failed permission checks
- Hierarchy sync success rate

### Logging Requirements
- All permission grants/revokes
- Failed permission checks with context
- Policy changes with full audit trail
- EVE ESI sync operations and failures
- Performance metrics and slow queries

## 11. Rollout Strategy

### Development Environment
1. Implement and test core functionality
2. Create test data for all scenarios
3. Performance testing with realistic data volumes

### Staging Environment  
1. Deploy with existing permission system running in parallel
2. Compare results between old and new systems
3. Load testing with production-like traffic
4. Train administrators on new system

### Production Deployment
1. Deploy in read-only mode initially
2. Shadow testing - log differences but don't enforce
3. Gradual rollout by service (start with non-critical services)
4. Full cutover once confidence is established
5. Keep rollback plan ready for 48 hours

## Dependencies

### Go Packages Required
```go
// Add to go.mod
github.com/casbin/casbin/v2
github.com/casbin/mongodb-adapter/v3
```

### EVE ESI Endpoints Used
- `/characters/{character_id}/` - Character information
- `/corporations/{corporation_id}/` - Corporation details  
- `/alliances/{alliance_id}/` - Alliance information
- `/characters/{character_id}/corporationhistory/` - Corporation changes

## Success Criteria

- [ ] All existing permissions continue to work without disruption
- [ ] New hierarchical permissions work across all three levels
- [ ] Permission check latency under 10ms (95th percentile)
- [ ] Zero security vulnerabilities in permission evaluation
- [ ] Complete audit trail for all permission changes
- [ ] Successful migration of existing permissions
- [ ] Administrator training completed and documented
- [ ] Monitoring and alerting fully operational

## Timeline Summary

| Phase | Duration | Key Deliverables |
|-------|----------|------------------|
| Phase 1: Foundation | Weeks 1-2 | CASBIN integration, core models |
| Phase 2: Core Logic | Weeks 2-3 | Permission checker, ESI sync |
| Phase 3: Management APIs | Weeks 3-4 | Admin endpoints, audit logging |
| Phase 4: Testing & Migration | Weeks 4-5 | Testing, migration, deployment |

**Total Estimated Duration: 5 weeks**

This plan provides a comprehensive roadmap for implementing a robust, scalable, and secure hierarchical permission system using CASBIN that properly handles EVE Online's complex organizational structure.