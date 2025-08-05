# EVE Online SSO Integration

This document describes the EVE Online Single Sign-On (SSO) integration implemented in the go-falcon project.

## Overview

The EVE Online SSO integration provides secure authentication using CCP Games' OAuth2 implementation, allowing users to authenticate with their EVE Online characters and access EVE-specific data through the ESI (EVE Swagger Interface) API.

## Features

- **OAuth2 Authorization Code Flow**: Secure authentication flow with state validation
- **JWT Token Management**: Internal JWT tokens for session management  
- **Character Profile Integration**: Automatic character data retrieval and storage
- **ESI Integration**: Character, corporation, and alliance data from EVE's ESI API
- **Refresh Token Support**: Automatic token refresh for long-lived sessions
- **Security Best Practices**: CSRF protection, secure cookies, proper token validation

## Architecture

### Components

1. **EVE SSO Handler** (`internal/auth/eve_sso.go`)
   - OAuth2 flow implementation
   - Token exchange and validation
   - JWT generation and validation
   - State management for CSRF protection

2. **Authentication Module** (`internal/auth/auth.go`)
   - HTTP handlers for SSO endpoints
   - Integration with user profile system
   - Background cleanup tasks

3. **User Profile System** (`internal/auth/profile.go`)
   - Character data storage and retrieval
   - ESI integration for real-time data
   - MongoDB-based persistence

4. **JWT Middleware** (`internal/auth/middleware.go`)
   - Request authentication
   - Scope-based authorization
   - User context injection

## Configuration

### Environment Variables

Required environment variables for EVE Online SSO:

```bash
# EVE Online Application Credentials (from developers.eveonline.com)
EVE_CLIENT_ID="your_application_client_id"
EVE_CLIENT_SECRET="your_application_client_secret"

# JWT Secret for internal token signing
JWT_SECRET="your_jwt_secret_key"

# Optional Configuration
EVE_REDIRECT_URI="http://localhost:8080/auth/eve/callback"  # Default redirect URI
EVE_SCOPES="publicData"                                     # Default scopes
ESI_USER_AGENT="go-falcon/1.0.0 contact@example.com"       # ESI User-Agent header
```

### Application Registration

1. Visit [EVE Online Developers Portal](https://developers.eveonline.com/)
2. Create a new application
3. Set the callback URL to match your `EVE_REDIRECT_URI`
4. Configure required scopes for your application
5. Note the Client ID and Secret for configuration

## API Endpoints

### Authentication Flow

#### 1. Initiate Authentication
```http
GET /auth/eve/login
```

Response:
```json
{
  "auth_url": "https://login.eveonline.com/v2/oauth/authorize?...",
  "state": "secure_random_state"
}
```

#### 2. Handle Callback
```http
GET /auth/eve/callback?code=auth_code&state=state_value
```

Response:
```json
{
  "success": true,
  "character_id": 123456789,
  "character_name": "Character Name",
  "scopes": "publicData esi-characters.read_blueprints.v1",
  "token": "jwt_token_string"
}
```

#### 3. Verify Token
```http
GET /auth/eve/verify
Authorization: Bearer jwt_token
```

Response:
```json
{
  "valid": true,
  "character_id": 123456789,
  "character_name": "Character Name",
  "scopes": "publicData",
  "expires_at": 1640995200
}
```

#### 4. Refresh Token
```http
POST /auth/eve/refresh
Content-Type: application/json

{
  "refresh_token": "refresh_token_string"
}
```

### User Profile Endpoints

#### Get User Profile
```http
GET /auth/profile
Authorization: Bearer jwt_token
```

Response:
```json
{
  "character_id": 123456789,
  "character_name": "Character Name",
  "corporation_id": 98000001,
  "corporation_name": "Corporation Name",
  "alliance_id": 99000001,
  "alliance_name": "Alliance Name",
  "security_status": 5.0,
  "birthday": "2003-05-06T00:00:00Z",
  "scopes": "publicData",
  "last_login": "2024-01-01T12:00:00Z",
  "created_at": "2024-01-01T10:00:00Z",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

#### Refresh Profile Data
```http
POST /auth/profile/refresh
Authorization: Bearer jwt_token
```

#### Get Public Profile
```http
GET /auth/profile/public?character_id=123456789
```

## Security Features

### CSRF Protection
- Cryptographically secure state parameter generation
- State validation on callback
- 15-minute state expiration

### Secure Token Handling
- HttpOnly cookies for JWT storage
- Secure flag for HTTPS connections
- SameSite protection against CSRF attacks
- 24-hour JWT expiration

### ESI Best Practices
- Proper User-Agent headers as required by CCP
- Cache management with HTTP headers
- Error limit monitoring
- Rate limiting compliance

## Middleware Usage

### Required Authentication
```go
// Require valid JWT token
r.With(authModule.JWTMiddleware).Get("/protected", handler)
```

### Optional Authentication
```go
// Add user to context if token is present
r.With(authModule.OptionalJWTMiddleware).Get("/optional", handler)
```

### Scope-Based Authorization
```go
// Require specific EVE Online scopes
r.With(authModule.RequireScopes("esi-characters.read_blueprints.v1")).Get("/blueprints", handler)
```

### Accessing Authenticated User
```go
func handler(w http.ResponseWriter, r *http.Request) {
    user, ok := auth.GetAuthenticatedUser(r)
    if !ok {
        http.Error(w, "Authentication required", http.StatusUnauthorized)
        return
    }
    
    // Use user.CharacterID, user.CharacterName, user.Scopes
}
```

## Database Schema

### User Profiles Collection
```javascript
{
  _id: ObjectId("..."),
  character_id: 123456789,
  character_name: "Character Name",
  corporation_id: 98000001,
  corporation_name: "Corporation Name",
  alliance_id: 99000001,
  alliance_name: "Alliance Name",
  security_status: 5.0,
  birthday: ISODate("2003-05-06T00:00:00Z"),
  scopes: "publicData esi-characters.read_blueprints.v1",
  last_login: ISODate("2024-01-01T12:00:00Z"),
  refresh_token: "encrypted_refresh_token",
  created_at: ISODate("2024-01-01T10:00:00Z"),
  updated_at: ISODate("2024-01-01T12:00:00Z")
}
```

Indexes:
- `character_id` (unique)
- `last_login` (for cleanup)

## Error Handling

### Common HTTP Status Codes
- `400 Bad Request`: Missing or invalid parameters
- `401 Unauthorized`: Invalid or missing authentication token
- `403 Forbidden`: Insufficient scopes/permissions
- `500 Internal Server Error`: Server-side errors (ESI failures, database errors)

### Logging
All authentication events are logged with structured logging including:
- Character ID and name
- IP addresses
- Error details
- ESI response status

## Deployment Considerations

### Production Settings
- Use HTTPS for all EVE SSO endpoints
- Set secure JWT secrets (32+ characters)
- Configure proper CORS policies
- Enable rate limiting
- Set up monitoring for ESI error limits

### Monitoring
- JWT token expiration rates
- ESI API response times and error rates
- User profile update frequencies
- Authentication success/failure rates

## Integration with Other Modules

### ESI Gateway Integration
The auth module integrates with `pkg/evegateway` for:
- Character public information
- Corporation details
- Alliance information
- Real-time data updates

### Module System Integration
Authentication middleware can be used across all modules:
- User-specific data filtering
- Character-based permissions
- Corp/alliance-based access control

## Troubleshooting

### Common Issues

1. **Invalid redirect URI**: Ensure `EVE_REDIRECT_URI` matches your application settings
2. **Missing scopes**: Verify requested scopes are registered with your application
3. **Token validation failures**: Check JWT secret configuration
4. **ESI call failures**: Verify User-Agent header and check EVE server status

### Debug Logging
Enable debug logging with:
```bash
LOG_LEVEL=debug
```

This will provide detailed information about:
- OAuth2 flow steps
- ESI API calls
- Token validation processes
- Database operations

## Migration and Updates

When updating EVE SSO integration:
1. Test with EVE's Singularity (test) server first
2. Monitor ESI error limits during deployment
3. Implement graceful fallbacks for ESI failures
4. Update user agent strings for new versions

## Compliance and Legal

This integration complies with:
- CCP Games' Developer License Agreement  
- EVE Online's Terms of Service
- ESI API usage guidelines
- GDPR requirements for user data handling

## Support and Resources

- [EVE Online Developers Portal](https://developers.eveonline.com/)
- [ESI API Documentation](https://esi.evetech.net/ui/)
- [EVE Online Third Party Developer Resources](https://www.eveonline.com/developers/)
- [CCP Games Developer License Agreement](https://developers.eveonline.com/license-agreement)