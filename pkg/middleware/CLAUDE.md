# Middleware Package (pkg/middleware)

## Overview

Comprehensive authentication and authorization middleware system for the Go Falcon API server, implementing the complete middleware plan with JWT validation, character resolution, and permission-based access control.

## 🏗️ Architecture

### Core Components

- **Authentication Middleware**: JWT validation from cookies and bearer tokens
- **Character Resolution**: Expands auth context with all user characters, corporations, and alliances
- **Authorization System**: Permission-based access control (CASBIN integration ready)
- **Convenience Wrappers**: Easy-to-use middleware stacks for common patterns
- **Debug & Observability**: Comprehensive logging and tracing throughout the stack

### Files Structure

```
pkg/middleware/
├── auth.go                    # Basic JWT authentication middleware
├── enhanced_auth.go           # Authentication + character resolution middleware
├── user_resolver.go           # Character/user relationship resolution
├── convenience.go             # Easy-to-use wrapper functions
├── factory.go                 # Pre-configured middleware stacks
├── helpers.go                 # Context utilities and debug tools
├── integration.go             # Huma v2 framework integration helpers
├── init.go                    # Initialization and setup utilities
├── debug_test_handler.go      # Debug endpoints for testing middleware
├── huma_helpers.go           # Huma-specific authentication helpers
├── tracing.go                # OpenTelemetry tracing middleware
└── CLAUDE.md                 # This documentation
```

## 🔐 Authentication System

### Dual Authentication Support

Supports both web and mobile authentication patterns:

1. **Web Applications**: Cookie-based JWT authentication
   - Cookie name: `falcon_auth_token`
   - Cross-subdomain support with proper domain settings
   - Secure, HttpOnly, SameSite=Lax attributes

2. **Mobile Applications**: Bearer token authentication
   - Standard `Authorization: Bearer <token>` header
   - Stateless authentication for API clients

### JWT Token Structure

```go
type AuthenticatedUser struct {
    UserID        string `json:"user_id"`
    CharacterID   int    `json:"character_id"`
    CharacterName string `json:"character_name"`
    Scopes        string `json:"scopes"`
}
```

### Authentication Context

```go
type AuthContext struct {
    UserID          string `json:"user_id"`
    PrimaryCharID   int64  `json:"primary_character_id"`
    RequestType     string `json:"request_type"` // "cookie" or "bearer"
    IsAuthenticated bool   `json:"is_authenticated"`
}
```

### Expanded Authentication Context

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
    
    // Additional Context (for CASBIN integration)
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
}
```

## 🔧 Middleware Components

### 1. Basic Authentication Middleware

```go
// AuthMiddleware provides basic JWT validation
type AuthMiddleware struct {
    jwtValidator JWTValidator
}

// Key methods
func (m *AuthMiddleware) ValidateAuthFromHeaders(authHeader, cookieHeader string) (*models.AuthenticatedUser, error)
func (m *AuthMiddleware) ValidateOptionalAuthFromHeaders(authHeader, cookieHeader string) *models.AuthenticatedUser
func (m *AuthMiddleware) ValidateScopesFromHeaders(authHeader, cookieHeader string, requiredScopes ...string) (*models.AuthenticatedUser, error)
```

### 2. Enhanced Authentication Middleware

```go
// EnhancedAuthMiddleware provides authentication + character resolution
type EnhancedAuthMiddleware struct {
    jwtValidator      JWTValidator
    characterResolver UserCharacterResolver
}

// Key middleware methods
func (m *EnhancedAuthMiddleware) AuthenticationMiddleware() func(http.Handler) http.Handler
func (m *EnhancedAuthMiddleware) CharacterResolutionMiddleware() func(http.Handler) http.Handler
func (m *EnhancedAuthMiddleware) RequireExpandedAuth() func(http.Handler) http.Handler
func (m *EnhancedAuthMiddleware) OptionalExpandedAuth() func(http.Handler) http.Handler
```

### 3. Character Resolution System

```go
// UserCharacterResolver interface for resolving user characters
type UserCharacterResolver interface {
    GetUserWithCharacters(ctx context.Context, userID string) (*UserWithCharacters, error)
}

// UserWithCharacters represents a user with all their characters
type UserWithCharacters struct {
    ID         string           `json:"id"`
    Characters []UserCharacter  `json:"characters"`
}

// UserCharacter represents a character linked to a user
type UserCharacter struct {
    CharacterID   int64     `json:"character_id"`
    Name          string    `json:"name"`
    CorporationID int64     `json:"corporation_id"`
    AllianceID    int64     `json:"alliance_id,omitempty"`
    IsPrimary     bool      `json:"is_primary"`
    AddedAt       time.Time `json:"added_at"`
    LastActive    time.Time `json:"last_active"`
}
```

### 4. Convenience Middleware

```go
// ConvenienceMiddleware provides easy-to-use wrapper functions
type ConvenienceMiddleware struct {
    enhanced *EnhancedAuthMiddleware
}

// Available convenience methods
func (m *ConvenienceMiddleware) RequireAuth() func(http.Handler) http.Handler
func (m *ConvenienceMiddleware) RequireAuthWithCharacters() func(http.Handler) http.Handler
func (m *ConvenienceMiddleware) OptionalAuth() func(http.Handler) http.Handler
func (m *ConvenienceMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler
func (m *ConvenienceMiddleware) RequireScope(scopes ...string) func(http.Handler) http.Handler
```

### 5. Middleware Factory

```go
// MiddlewareFactory creates pre-configured middleware stacks
type MiddlewareFactory struct {
    authMiddleware        *AuthMiddleware
    enhancedAuthMiddleware *EnhancedAuthMiddleware
    convenienceMiddleware *ConvenienceMiddleware
    contextHelper         *ContextHelper
}

// Pre-configured middleware stacks
func (f *MiddlewareFactory) PublicWithOptionalAuth() func(http.Handler) http.Handler
func (f *MiddlewareFactory) RequireBasicAuth() func(http.Handler) http.Handler
func (f *MiddlewareFactory) RequireAuthWithCharacters() func(http.Handler) http.Handler
func (f *MiddlewareFactory) RequireScope(scopes ...string) func(http.Handler) http.Handler
func (f *MiddlewareFactory) RequirePermission(resource, action string) func(http.Handler) http.Handler
func (f *MiddlewareFactory) AdminOnly() func(http.Handler) http.Handler
func (f *MiddlewareFactory) CorporationAccess(resource string) func(http.Handler) http.Handler
func (f *MiddlewareFactory) AllianceAccess(resource string) func(http.Handler) http.Handler
```

## 🎯 Usage Examples

### Quick Setup

```go
// Initialize middleware factory
factory, err := middleware.QuickSetup(authService, mongodb)
if err != nil {
    log.Fatal(err)
}

// Use in Chi router
r.Use(factory.RequireBasicAuth())

// Use specific middleware stacks
r.Route("/admin", func(r chi.Router) {
    r.Use(factory.AdminOnly())
    // admin routes
})

r.Route("/corp", func(r chi.Router) {
    r.Use(factory.CorporationAccess("data"))
    // corporation routes
})
```

### Context Access

```go
// Get authenticated user from context
user := middleware.GetAuthenticatedUser(r.Context())
if user != nil {
    userID := user.UserID
    characterID := user.CharacterID
}

// Get basic auth context
authCtx := middleware.GetAuthContext(r.Context())
if authCtx != nil && authCtx.IsAuthenticated {
    userID := authCtx.UserID
    requestType := authCtx.RequestType // "cookie" or "bearer"
}

// Get expanded auth context with all characters
expandedCtx := middleware.GetExpandedAuthContext(r.Context())
if expandedCtx != nil {
    allCharacterIDs := expandedCtx.CharacterIDs
    corporationIDs := expandedCtx.CorporationIDs
    allianceIDs := expandedCtx.AllianceIDs
}
```

### Context Helper Utilities

```go
helper := middleware.NewContextHelper()

// Check authentication status
if helper.IsAuthenticated(r) {
    userID := helper.GetUserID(r)
    primaryCharID := helper.GetPrimaryCharacterID(r)
    allCharIDs := helper.GetAllCharacterIDs(r)
}

// Get comprehensive auth info
authInfo := helper.GetAuthInfo(r)
fmt.Printf("User: %s, Characters: %v", authInfo.UserID, authInfo.CharacterIDs)

// Debug auth context
helper.DebugAuthContext(r)
```

### Huma v2 Integration

```go
// For Huma route handlers
authMiddleware := middleware.NewAuthMiddleware(authService)

// Validate authentication in Huma handlers
user, err := authMiddleware.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
if err != nil {
    return nil, huma.Error401Unauthorized("Authentication required", err)
}

// Validate with specific scopes
user, err := authMiddleware.ValidateScopesFromHeaders(
    input.Authorization, 
    input.Cookie, 
    "esi-characters.read_contacts.v1",
)
```

## 🔍 Debug and Observability

### Debug Logging

Comprehensive debug logging throughout the middleware stack:

```bash
# HTTP Middleware Stack
[DEBUG] TracingMiddleware: Processing request GET /api/endpoint
[DEBUG] MiddlewareFactory.RequireAuthWithCharacters: Starting middleware chain
[DEBUG] ConvenienceMiddleware.RequireAuthWithCharacters: Processing GET /api/endpoint
[DEBUG] EnhancedAuthMiddleware: Found Authorization header: "Bearer eyJ..."
[DEBUG] EnhancedAuthMiddleware: Extracted bearer token (length=156)
[DEBUG] CharacterResolutionMiddleware: Resolving characters for user 12345
[DEBUG] UserCharacterResolver: Found 3 profiles for user 12345

# Huma Route Handlers
[DEBUG] ===== /auth/status HUMA HANDLER START =====
[DEBUG] AuthStatus Handler: Authorization header: "Bearer eyJ..."
[DEBUG] AuthService.GetAuthStatusFromHeaders: authHeader="Bearer..." cookieHeader="..."
[DEBUG] AuthService: Extracted JWT from Authorization header (length=156)
[DEBUG] AuthService: Validating JWT token
```

### Debug Middleware

```go
// Comprehensive debug middleware
func DebugMiddleware() func(http.Handler) http.Handler

// Output includes:
// - Request method and path
// - Header information (Authorization, Cookie)
// - Authentication context details
// - Character resolution results
// - Performance timing
```

### Debug Test Endpoints

```go
// Debug test handler for middleware testing
func DebugTestHandler() http.HandlerFunc

// Setup debug routes
func SetupDebugRoutes(factory *MiddlewareFactory, mux *http.ServeMux)

// Available debug endpoints:
// - /debug/public (optional auth)
// - /debug/auth (require auth)
// - /debug/characters (require auth + characters)
```

## 🚀 Performance Features

### Optimizations

- **Conditional Processing**: Middleware only processes when needed
- **Context Reuse**: Efficient context value propagation
- **Database Optimization**: Character resolution uses projection queries
- **Caching Ready**: Architecture supports Redis caching for character data

### Memory Management

- **Minimal Allocations**: Reuses structures where possible
- **Context Cleanup**: Proper cleanup of context values
- **Efficient Parsing**: Optimized JWT and cookie parsing

## 🔒 Security Features

### Token Security

- **JWT Validation**: Proper signature verification
- **Expiration Checking**: Automatic token expiration handling
- **Scope Validation**: EVE Online scope requirement checking
- **Secure Headers**: Proper cookie security attributes

### Request Security

- **CSRF Protection**: State parameter validation for EVE SSO
- **Rate Limiting Ready**: Architecture supports rate limiting integration
- **Audit Logging**: Comprehensive request and auth logging

## 🔧 Configuration

### Environment Variables

```bash
# JWT Configuration
JWT_SECRET="your-jwt-secret-key"

# Cookie Configuration
COOKIE_DOMAIN=".eveonline.it"  # Cross-subdomain support

# Debug Configuration
ENABLE_TELEMETRY="true"        # Enable tracing and debug output

# EVE Online Integration
EVE_CLIENT_ID="your-eve-client-id"
EVE_CLIENT_SECRET="your-eve-secret"
```

### Factory Configuration

```go
// Default configuration
config := middleware.DefaultConfig()

// Production configuration
config := middleware.ProductionSetup(authService, mongodb)

// Custom configuration
config := &middleware.MiddlewareConfig{
    EnableDebug:   false,  // Disable debug in production
    EnableCaching: true,   // Enable character caching
    CacheTTL:      1800,   // 30 minutes
}
```

## 🔮 Future Enhancements (Ready for Implementation)

### CASBIN Integration

The middleware is architected to support CASBIN authorization:

```go
// Subject building for CASBIN
subjects := []string{
    fmt.Sprintf("user:%s", userID),
    fmt.Sprintf("character:%d", characterID),
    fmt.Sprintf("corporation:%d", corporationID),
    fmt.Sprintf("alliance:%d", allianceID),
}

// Permission checking
authorized := enforcer.Enforce(subject, resource, action)
```

### Caching Layer

```go
// Character cache interface (ready for implementation)
type CharacterCache interface {
    GetUserCharacters(userID string) (*ExpandedAuthContext, error)
    SetUserCharacters(userID string, ctx *ExpandedAuthContext) error
}
```

### API Key Authentication

```go
// API key middleware (future implementation)
func (f *MiddlewareFactory) APIKeyRequired() func(http.Handler) http.Handler
```

## 📚 Integration Points

### Chi Router Integration

- Seamless integration with go-chi middleware stack
- Supports nested route groups with different auth requirements
- Compatible with existing Chi middleware

### Huma v2 Framework Integration

- Direct integration with Huma route handlers
- Type-safe input/output with authentication context
- Automatic OpenAPI documentation for auth requirements

### Database Integration

- MongoDB integration for user/character data
- Efficient queries with proper indexing
- Ready for Redis caching integration

### EVE Online Integration

- Complete EVE SSO authentication flow
- ESI scope validation
- Character/corporation/alliance relationship tracking

## 🧪 Testing

### Unit Tests

```bash
go test ./pkg/middleware/...
```

### Integration Testing

- Complete authentication flow testing
- Character resolution testing
- Middleware chain testing
- Context propagation testing

### Debug Testing

Use the debug endpoints to test middleware behavior:

```bash
# Test public endpoint with optional auth
curl -H "Authorization: Bearer <token>" http://localhost:8080/debug/public

# Test authenticated endpoint
curl -H "Authorization: Bearer <token>" http://localhost:8080/debug/auth

# Test character resolution
curl -H "Authorization: Bearer <token>" http://localhost:8080/debug/characters
```

## 📈 Monitoring

### Metrics Available

- Authentication success/failure rates
- Character resolution performance
- Middleware execution timing
- Context building performance

### OpenTelemetry Integration

- Automatic request tracing when `ENABLE_TELEMETRY=true`
- Span creation for authentication and character resolution
- Attribute injection for debugging and monitoring

## 🤝 Contributing

### Adding New Middleware

1. Implement the middleware function following the pattern
2. Add convenience wrapper to `ConvenienceMiddleware`
3. Add factory method to `MiddlewareFactory`
4. Add debug logging throughout
5. Update tests and documentation

### Best Practices

- Always add comprehensive debug logging
- Follow the established context key patterns
- Implement proper error handling with meaningful messages
- Add factory methods for common use cases
- Maintain backward compatibility

## 📄 License

This middleware package is part of the Go Falcon project and follows the same license terms.