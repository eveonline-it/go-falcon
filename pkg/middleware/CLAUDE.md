# Middleware Package (pkg/middleware)

## Overview

Shared middleware components providing centralized authentication, permission checking, OpenTelemetry tracing, and request correlation across all HTTP handlers in the monolithic API gateway.

**Status**: Production Ready - Complete Centralized Permission System
**Latest Update**: Implemented centralized permission middleware to eliminate code duplication across modules
**Architecture**: Interface-based design with comprehensive testing and migration utilities

## Core Features

### 🔐 Centralized Permission Middleware
- **Unified Authentication**: Single source of truth for JWT validation and user authentication
- **Permission Checking**: Centralized permission validation with super admin bypass support
- **Module Adapters**: Pre-built adapters for existing module patterns (sitemap, scheduler, groups, etc.)
- **Fallback Support**: Graceful degradation when permission system is unavailable
- **Debug Logging**: Comprehensive logging for troubleshooting permission issues

### 📊 OpenTelemetry Tracing
- **Automatic HTTP Request Tracing**: Creates spans for all HTTP requests
- **Context Propagation**: Proper trace context handling
- **Span Management**: Request-scoped span lifecycle
- **Attribute Injection**: Rich metadata for observability

## Files Structure

```
pkg/middleware/
├── auth.go              # Base authentication utilities (JWT validation, token extraction)
├── permissions.go       # Centralized permission middleware with comprehensive checking
├── adapters.go          # Module-specific adapters (sitemap, scheduler, groups, etc.)
├── utils.go             # Factory functions, validators, and migration utilities  
├── permissions_test.go  # Comprehensive test suite with mocks
├── tracing.go           # OpenTelemetry tracing middleware
└── CLAUDE.md           # This documentation
```

## Centralized Permission Architecture

### Core Interface Design

The centralized middleware uses interface-based dependency injection for maximum flexibility:

```go
// PermissionChecker interface for permission operations
type PermissionChecker interface {
    HasPermission(ctx context.Context, characterID int64, permissionID string) (bool, error)
    CheckPermission(ctx context.Context, characterID int64, permissionID string) (*permissions.PermissionCheck, error)
}

// JWTValidator interface for authentication
type JWTValidator interface {
    ValidateJWT(token string) (*models.AuthenticatedUser, error)
}
```

### Permission Middleware Features

#### Core Methods
```go
// Authentication only
func (pm *PermissionMiddleware) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error)

// Single permission required
func (pm *PermissionMiddleware) RequirePermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, error)

// Any one permission required (OR logic)
func (pm *PermissionMiddleware) RequireAnyPermission(ctx context.Context, authHeader, cookieHeader string, permissionIDs []string) (*models.AuthenticatedUser, error)

// All permissions required (AND logic) 
func (pm *PermissionMiddleware) RequireAllPermissions(ctx context.Context, authHeader, cookieHeader string, permissionIDs []string) (*models.AuthenticatedUser, error)
```

#### Configuration Options
```go
// Factory with options
pm := NewPermissionMiddleware(
    authService,           // JWT validator
    permissionManager,     // Permission checker
    WithDebugLogging(),    // Enable detailed debug logs
    WithCircuitBreaker(),  // Enable circuit breaker pattern
    WithoutFallback(),     // Disable auth-only fallback
)
```

### Module Adapters

Pre-built adapters provide drop-in replacements for existing module middleware:

#### Sitemap Adapter
```go
adapter := NewSitemapAdapter(permissionMiddleware)

// Drop-in replacements for existing methods
user, err := adapter.RequireSitemapView(ctx, authHeader, cookieHeader)
user, err := adapter.RequireSitemapAdmin(ctx, authHeader, cookieHeader)  
user, err := adapter.RequireSitemapNavigation(ctx, authHeader, cookieHeader)
```

#### Scheduler Adapter
```go
adapter := NewSchedulerAdapter(permissionMiddleware)

// Handles complex permission arrays
user, err := adapter.RequireSchedulerManagement(ctx, authHeader, cookieHeader)
user, err := adapter.RequireTaskManagement(ctx, authHeader, cookieHeader)
```

#### Groups Adapter
```go
adapter := NewGroupsAdapter(permissionMiddleware)

user, err := adapter.RequireGroupManagement(ctx, authHeader, cookieHeader)
user, err := adapter.RequireGroupPermissions(ctx, authHeader, cookieHeader)
```

## Migration from Module-Specific Middleware

### Before (Module-Specific)
Each module had its own middleware with duplicated code:

```go
// internal/sitemap/middleware/auth.go (141 lines)
type AuthMiddleware struct {
    authService       *authServices.AuthService  
    permissionManager *permissions.PermissionManager
}

func (m *AuthMiddleware) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
    // 50+ lines of duplicated token extraction and validation logic
}

func (m *AuthMiddleware) RequireSitemapAdmin(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
    // 20+ lines of duplicated permission checking logic
}
```

### After (Centralized)
Single centralized middleware with module adapters:

```go
// Module initialization
permissionMiddleware := middleware.NewPermissionMiddleware(authService, permissionManager, middleware.WithDebugLogging())
sitemapAdapter := middleware.NewSitemapAdapter(permissionMiddleware)

// In route handlers - same interface, centralized logic
user, err := sitemapAdapter.RequireSitemapAdmin(ctx, input.Authorization, input.Cookie)
```

### Migration Benefits
- **800+ lines removed**: Eliminates duplicated auth logic across 8+ modules
- **Consistent behavior**: Same auth flow for all modules
- **Better testing**: Test once, works everywhere  
- **Easier maintenance**: Bug fixes and improvements in one place
- **Enhanced features**: Debug logging, circuit breakers, fallback options

## Factory Patterns

### Standard Factory
```go
factory := NewPermissionMiddlewareFactory()

// Development (with debug logging)
pm := factory.CreateDevelopment(authService, permissionManager)

// Production (with circuit breaker)
pm := factory.CreateProduction(authService, permissionManager)

// Standard (balanced configuration)
pm := factory.CreateStandard(authService, permissionManager)
```

### Migration Helper
```go
migrationHelper := NewMigrationHelper(permissionMiddleware)

// Create adapters for existing modules
sitemapAdapter := migrationHelper.CreateSitemapAdapter()
schedulerAdapter := migrationHelper.CreateSchedulerAdapter()
groupsAdapter := migrationHelper.CreateGroupsAdapter()
```

## Error Handling & Debugging

### Debug Logging
When enabled with `WithDebugLogging()`, provides detailed logging:

```
[Permission Middleware] Checking permission for character_id=12345, user_id=abc, character_name=Test, permission=sitemap:navigation:customize
[Permission Middleware] Permission granted for character_id=12345, character_name=Test, permission=sitemap:navigation:customize via Super Administrator
```

### Graceful Fallback
With `FallbackToAuth: true` (default), system continues operating if permission system is unavailable:

```go
// Permission manager unavailable - falls back to auth-only mode
user, err := pm.RequirePermission(ctx, authHeader, cookieHeader, "some:permission")
// Returns authenticated user instead of error
```

### Error Types
- **401 Unauthorized**: Invalid or missing authentication token
- **403 Forbidden**: Valid authentication but insufficient permissions
- **500 Internal Server Error**: Permission system failure (when fallback disabled)

## Testing Support

### Comprehensive Test Suite
- **Unit Tests**: All core functionality with mocks
- **Integration Tests**: Real auth service integration  
- **Adapter Tests**: Module-specific adapter validation
- **Error Scenarios**: Authentication failures, permission denials, system errors

### Mock Interfaces
```go
// Built-in mock support for testing
type MockJWTValidator struct { mock.Mock }
type MockPermissionManager struct { mock.Mock }

// Easy test setup
mockAuth := &MockJWTValidator{}
mockPerm := &MockPermissionManager{}
pm := NewPermissionMiddleware(mockAuth, mockPerm)
```

## Performance Considerations

### Optimized Design
- **Interface-based**: Minimal runtime overhead
- **Context Propagation**: Efficient request-scoped data
- **Caching Ready**: Architecture supports Redis caching integration
- **Circuit Breaker**: Optional pattern for permission service resilience

### Memory Efficiency  
- **Shared Instances**: Single middleware instance per module
- **No Session Storage**: Stateless JWT-based authentication
- **Minimal Allocations**: Reuses context and avoids unnecessary object creation

## Integration Examples

### Basic Module Integration
```go
// In module initialization
permissionMiddleware := middleware.NewPermissionMiddleware(
    authService,
    permissionManager,
    middleware.WithDebugLogging(),
)

// In route registration
huma.Register(api, huma.Operation{...}, func(ctx context.Context, input *CreateInput) (*Output, error) {
    // Check permission before processing
    user, err := permissionMiddleware.RequirePermission(ctx, input.Authorization, input.Cookie, "module:resource:action")
    if err != nil {
        return nil, err
    }
    
    // Process request with authenticated user
    return processRequest(ctx, user, input)
})
```

### Advanced Usage with Validators
```go
validator := middleware.NewPermissionValidator(permissionMiddleware)

requirements := []middleware.PermissionRequirement{
    {PermissionID: "sitemap:routes:view", Description: "View routes", Required: true},
    {PermissionID: "sitemap:admin:manage", Description: "Admin access", Required: false},
}

result := validator.ValidateUserPermissions(ctx, authHeader, cookieHeader, requirements)
if !result.Valid {
    return handleInsufficientPermissions(result.MissingPerms, result.Errors)
}
```

## Tracing Middleware (Legacy)

### OpenTelemetry Integration
- **Automatic Spans**: Creates spans for all HTTP requests
- **URL Path Tracking**: Records request paths and methods  
- **Error Capture**: Automatic error recording in spans
- **Response Status**: HTTP status code tracking

### Usage
```go
// Apply tracing middleware
r.Use(middleware.TracingMiddleware)

// Custom tracing in handlers
span := trace.SpanFromContext(r.Context())
span.SetAttributes(attribute.String("custom", "value"))
```

## Migration Checklist

### Phase 1: Module Preparation
- [ ] Identify current module middleware usage
- [ ] Create centralized permission middleware instance
- [ ] Initialize appropriate module adapter

### Phase 2: Route Migration  
- [ ] Replace module middleware calls with adapter calls
- [ ] Update error handling (should be transparent)
- [ ] Run comprehensive test suite
- [ ] Verify debug logging works

### Phase 3: Cleanup
- [ ] Remove module-specific middleware file
- [ ] Update module constructor dependencies  
- [ ] Clean up unused imports
- [ ] Update module documentation

## Dependencies

### Internal Dependencies
- `go-falcon/internal/auth/models` (authenticated user models)
- `go-falcon/pkg/permissions` (permission checking interface)
- `go-falcon/pkg/config` (cookie configuration)

### External Dependencies  
- `github.com/danielgtaylor/huma/v2` (HTTP error types)
- `github.com/stretchr/testify/mock` (testing mocks)
- `github.com/stretchr/testify/assert` (test assertions)

## Future Enhancements

### Planned Features
- **Redis Caching**: Cache frequent permission checks
- **Rate Limiting**: Request-based rate limiting middleware
- **Audit Logging**: Detailed access audit trails
- **Metrics Integration**: Prometheus metrics for auth/authz operations
- **Multi-Factor Auth**: Support for 2FA integration

### Advanced Features
- **Permission Templates**: Pre-configured permission sets for common roles
- **Time-Based Permissions**: Temporary permission grants  
- **Location-Based Access**: Geographical access restrictions
- **Risk-Based Authentication**: Dynamic authentication requirements

This centralized middleware system provides a solid foundation for scalable, maintainable authentication and authorization across the entire go-falcon monolith while preserving all existing functionality and improving developer experience.