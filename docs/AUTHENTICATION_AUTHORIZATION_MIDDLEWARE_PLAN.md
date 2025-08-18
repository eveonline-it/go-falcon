# Authentication & Authorization Middleware Plan

## Overview

This document outlines the comprehensive plan for implementing authentication and authorization middleware with CASBIN integration for the Go Falcon API server. The middleware will manage incoming requests (via cookie or bearer token), extract user relationships, and provide the necessary identifiers for role-based access control.

## Current Authentication State

### Existing Authentication Flow

Go Falcon currently supports dual authentication methods:

1. **Web Applications**: Cookie-based JWT authentication
   ```
   GET /auth/eve/login → EVE SSO → /auth/eve/callback → JWT Cookie
   ```

2. **Mobile Applications**: Bearer token authentication  
   ```
   POST /auth/eve/token → Exchange EVE token → JWT Bearer token
   ```

### Current JWT Structure

```go
type JWTClaims struct {
    UserID      string    `json:"user_id"`
    CharacterID int64     `json:"character_id"`
    Name        string    `json:"name"`
    ExpiresAt   time.Time `json:"exp"`
}
```

## Proposed Middleware Architecture

### 1. Authentication Middleware Layer

```go
type AuthContext struct {
    UserID        string    `json:"user_id"`
    PrimaryCharID int64     `json:"primary_character_id"`
    RequestType   string    `json:"request_type"` // "cookie" or "bearer"
    IsAuthenticated bool    `json:"is_authenticated"`
}
```

### 2. Character Resolution Middleware Layer

This middleware will expand the authentication context with all related identifiers:

```go
type ExpandedAuthContext struct {
    *AuthContext
    
    // Character Information
    CharacterIDs    []int64 `json:"character_ids"`
    CorporationIDs  []int64 `json:"corporation_ids"`
    AllianceIDs     []int64 `json:"alliance_ids,omitempty"`
    
    // Primary Character Details
    PrimaryCharacter struct {
        ID            int64  `json:"id"`
        Name          string `json:"name"`
        CorporationID int64  `json:"corporation_id"`
        AllianceID    int64  `json:"alliance_id,omitempty"`
    } `json:"primary_character"`
    
    // Additional Context
    Roles        []string `json:"roles"`
    Permissions  []string `json:"permissions"`
}
```

## Data Model Requirements

### User-Character Relationships

```go
type User struct {
    ID         string    `bson:"_id" json:"id"`
    Characters []UserCharacter `bson:"characters" json:"characters"`
    CreatedAt  time.Time `bson:"created_at" json:"created_at"`
    UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
}

type UserCharacter struct {
    CharacterID   int64     `bson:"character_id" json:"character_id"`
    Name          string    `bson:"name" json:"name"`
    CorporationID int64     `bson:"corporation_id" json:"corporation_id"`
    AllianceID    int64     `bson:"alliance_id,omitempty" json:"alliance_id,omitempty"`
    IsPrimary     bool      `bson:"is_primary" json:"is_primary"`
    AddedAt       time.Time `bson:"added_at" json:"added_at"`
    LastActive    time.Time `bson:"last_active" json:"last_active"`
}
```

### Character Details Cache

```go
type CachedCharacter struct {
    CharacterID   int64     `bson:"_id" json:"character_id"`
    Name          string    `bson:"name" json:"name"`
    CorporationID int64     `bson:"corporation_id" json:"corporation_id"`
    AllianceID    int64     `bson:"alliance_id,omitempty" json:"alliance_id,omitempty"`
    LastUpdated   time.Time `bson:"last_updated" json:"last_updated"`
    ExpiresAt     time.Time `bson:"expires_at" json:"expires_at"`
}
```

## Middleware Implementation Plan

### 1. Base Authentication Middleware

```go
// pkg/middleware/auth.go
func AuthenticationMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            var token string
            var requestType string
            
            // Check for Bearer token first
            if authHeader := r.Header.Get("Authorization"); authHeader != "" {
                if strings.HasPrefix(authHeader, "Bearer ") {
                    token = strings.TrimPrefix(authHeader, "Bearer ")
                    requestType = "bearer"
                }
            }
            
            // Fallback to cookie
            if token == "" {
                if cookie, err := r.Cookie("jwt"); err == nil {
                    token = cookie.Value
                    requestType = "cookie"
                }
            }
            
            if token == "" {
                http.Error(w, "Authentication required", http.StatusUnauthorized)
                return
            }
            
            // Validate JWT and extract claims
            claims, err := validateJWT(token)
            if err != nil {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }
            
            // Create base auth context
            authCtx := &AuthContext{
                UserID:          claims.UserID,
                PrimaryCharID:   claims.CharacterID,
                RequestType:     requestType,
                IsAuthenticated: true,
            }
            
            // Add to request context
            ctx := context.WithValue(r.Context(), "auth", authCtx)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### 2. Character Resolution Middleware

```go
// pkg/middleware/character_resolution.go
func CharacterResolutionMiddleware(userService *users.Service) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authCtx := GetAuthContext(r.Context())
            if authCtx == nil || !authCtx.IsAuthenticated {
                http.Error(w, "Authentication required", http.StatusUnauthorized)
                return
            }
            
            // Resolve all characters for the user
            expandedCtx, err := resolveUserCharacters(authCtx, userService)
            if err != nil {
                log.WithError(err).Error("Failed to resolve user characters")
                http.Error(w, "Internal server error", http.StatusInternalServerError)
                return
            }
            
            // Add expanded context to request
            ctx := context.WithValue(r.Context(), "expanded_auth", expandedCtx)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func resolveUserCharacters(authCtx *AuthContext, userService *users.Service) (*ExpandedAuthContext, error) {
    // Get user with all characters
    user, err := userService.GetUserWithCharacters(authCtx.UserID)
    if err != nil {
        return nil, err
    }
    
    var characterIDs []int64
    var corporationIDs []int64
    var allianceIDs []int64
    
    // Extract unique IDs
    corpMap := make(map[int64]bool)
    allianceMap := make(map[int64]bool)
    
    for _, char := range user.Characters {
        characterIDs = append(characterIDs, char.CharacterID)
        
        if !corpMap[char.CorporationID] {
            corporationIDs = append(corporationIDs, char.CorporationID)
            corpMap[char.CorporationID] = true
        }
        
        if char.AllianceID > 0 && !allianceMap[char.AllianceID] {
            allianceIDs = append(allianceIDs, char.AllianceID)
            allianceMap[char.AllianceID] = true
        }
    }
    
    // Find primary character details
    var primaryChar UserCharacter
    for _, char := range user.Characters {
        if char.CharacterID == authCtx.PrimaryCharID {
            primaryChar = char
            break
        }
    }
    
    return &ExpandedAuthContext{
        AuthContext:    authCtx,
        CharacterIDs:   characterIDs,
        CorporationIDs: corporationIDs,
        AllianceIDs:    allianceIDs,
        PrimaryCharacter: struct {
            ID            int64  `json:"id"`
            Name          string `json:"name"`
            CorporationID int64  `json:"corporation_id"`
            AllianceID    int64  `json:"alliance_id,omitempty"`
        }{
            ID:            primaryChar.CharacterID,
            Name:          primaryChar.Name,
            CorporationID: primaryChar.CorporationID,
            AllianceID:    primaryChar.AllianceID,
        },
    }, nil
}
```

## CASBIN Integration Strategy

### 1. Policy Model Definition

```conf
# casbin_model.conf
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _
g2 = _, _
g3 = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
```

### 2. Subject Types for CASBIN

CASBIN subjects will follow the pattern: `{type}:{id}`

- `user:{user_id}` - Individual user
- `character:{character_id}` - Specific character
- `corporation:{corporation_id}` - Corporation membership
- `alliance:{alliance_id}` - Alliance membership

### 3. Permission Middleware with CASBIN

```go
// pkg/middleware/casbin.go
func CASBINAuthorizationMiddleware(enforcer *casbin.Enforcer) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            expandedAuth := GetExpandedAuthContext(r.Context())
            if expandedAuth == nil {
                http.Error(w, "Authorization context missing", http.StatusInternalServerError)
                return
            }
            
            // Extract resource and action from request
            resource := extractResource(r)
            action := extractAction(r)
            
            // Check permissions for all subject types
            subjects := buildSubjects(expandedAuth)
            
            authorized := false
            for _, subject := range subjects {
                if enforcer.Enforce(subject, resource, action) {
                    authorized = true
                    break
                }
            }
            
            if !authorized {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

func buildSubjects(ctx *ExpandedAuthContext) []string {
    subjects := []string{
        fmt.Sprintf("user:%s", ctx.UserID),
        fmt.Sprintf("character:%d", ctx.PrimaryCharacter.ID),
    }
    
    // Add all character subjects
    for _, charID := range ctx.CharacterIDs {
        subjects = append(subjects, fmt.Sprintf("character:%d", charID))
    }
    
    // Add corporation subjects
    for _, corpID := range ctx.CorporationIDs {
        subjects = append(subjects, fmt.Sprintf("corporation:%d", corpID))
    }
    
    // Add alliance subjects
    for _, allianceID := range ctx.AllianceIDs {
        subjects = append(subjects, fmt.Sprintf("alliance:%d", allianceID))
    }
    
    return subjects
}
```

## Request Flow Examples

### 1. Web Application Request Flow

```
1. User makes request with JWT cookie
2. AuthenticationMiddleware extracts JWT from cookie
3. JWT validated, AuthContext created with user_id and primary_character_id
4. CharacterResolutionMiddleware queries database for all user characters
5. ExpandedAuthContext created with all character_ids, corporation_ids, alliance_ids
6. CASBINAuthorizationMiddleware builds subject list and checks permissions
7. Request proceeds to business logic if authorized
```

### 2. Mobile Application Request Flow

```
1. User makes request with Bearer token
2. AuthenticationMiddleware extracts JWT from Authorization header
3. JWT validated, AuthContext created with user_id and primary_character_id
4. CharacterResolutionMiddleware queries database for all user characters
5. ExpandedAuthContext created with all character_ids, corporation_ids, alliance_ids
6. CASBINAuthorizationMiddleware builds subject list and checks permissions
7. Request proceeds to business logic if authorized
```

## Database Operations

### Required Service Methods

```go
// internal/users/services/service.go
type Service interface {
    GetUserWithCharacters(userID string) (*models.User, error)
    AddCharacterToUser(userID string, character *models.UserCharacter) error
    UpdateCharacterDetails(characterID int64, corporationID, allianceID int64) error
    RemoveCharacterFromUser(userID string, characterID int64) error
}
```

### Caching Strategy

```go
// pkg/cache/character_cache.go
type CharacterCache struct {
    redis  *redis.Client
    ttl    time.Duration
}

func (c *CharacterCache) GetUserCharacters(userID string) (*ExpandedAuthContext, error) {
    // Try cache first
    key := fmt.Sprintf("user_characters:%s", userID)
    cached, err := c.redis.Get(key).Result()
    if err == nil {
        var ctx ExpandedAuthContext
        if json.Unmarshal([]byte(cached), &ctx) == nil {
            return &ctx, nil
        }
    }
    
    // Cache miss - fall back to database
    return nil, cache.ErrCacheMiss
}

func (c *CharacterCache) SetUserCharacters(userID string, ctx *ExpandedAuthContext) error {
    key := fmt.Sprintf("user_characters:%s", userID)
    data, err := json.Marshal(ctx)
    if err != nil {
        return err
    }
    
    return c.redis.Set(key, data, c.ttl).Err()
}
```

## Implementation Phases

### Phase 1: Core Authentication Enhancement
- [ ] Enhance JWT claims structure
- [ ] Implement AuthContext and ExpandedAuthContext
- [ ] Create base authentication middleware
- [ ] Update existing auth endpoints to support new structure

### Phase 2: Character Resolution System
- [ ] Implement character resolution middleware
- [ ] Create user service methods for character management
- [ ] Add character caching layer
- [ ] ESI integration for corporation/alliance updates

### Phase 3: CASBIN Integration
- [ ] Set up CASBIN enforcer with policy model
- [ ] Implement authorization middleware
- [ ] Create permission management endpoints
- [ ] Integrate with existing modules

### Phase 4: Testing & Optimization
- [ ] Unit tests for all middleware components
- [ ] Integration tests for auth flows
- [ ] Performance testing and optimization
- [ ] Documentation and examples

## Security Considerations

### 1. Token Validation
- Implement proper JWT validation with signature verification
- Token expiration and refresh mechanism
- Secure token storage recommendations

### 2. Permission Caching
- Cache permissions with appropriate TTL
- Implement cache invalidation on permission changes
- Monitor cache hit rates and performance

### 3. Rate Limiting
- Implement rate limiting per user/character
- Different limits for different subject types
- Anti-abuse measures for permission checks

### 4. Audit Logging
- Log all permission checks and results
- Track permission changes and assignments
- Monitor for unauthorized access attempts

## Configuration

### Environment Variables

```bash
# CASBIN Configuration
CASBIN_MODEL_PATH="./configs/casbin_model.conf"
CASBIN_POLICY_PATH="./configs/casbin_policy.csv"
CASBIN_ADAPTER="database"  # or "file"

# Cache Configuration
AUTH_CACHE_TTL="3600"      # 1 hour
CHAR_CACHE_TTL="1800"      # 30 minutes

# Permission System
PERMISSION_CHECK_TIMEOUT="5s"
MAX_SUBJECTS_PER_REQUEST="100"
```

### CASBIN Policy Examples

```csv
# Policy file (casbin_policy.csv)
p, user:123e4567-e89b-12d3-a456-426614174000, scheduler.tasks, read
p, corporation:1000001, users.profiles, read  
p, alliance:99000001, scheduler.tasks, write
p, character:2112625428, scheduler.tasks, admin

# Role definitions
g, user:123e4567-e89b-12d3-a456-426614174000, admin
g, corporation:1000001, alliance:99000001
```

## Conclusion

This comprehensive middleware architecture provides:

- **Flexible Authentication**: Support for both cookie and bearer token authentication
- **Complete Context**: All character, corporation, and alliance identifiers available
- **CASBIN Integration**: Role-based access control with hierarchical permissions
- **Performance**: Caching layers for optimal response times
- **Security**: Proper validation, logging, and rate limiting
- **Scalability**: Designed to handle multiple characters per user efficiently

The implementation will be done in phases to ensure stability and proper testing at each stage.