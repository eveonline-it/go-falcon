# Authentication Module (internal/auth)

## Overview

The authentication module provides complete EVE Online Single Sign-On (SSO) integration with JWT-based session management, user profiles, and secure cookie handling for cross-subdomain authentication.
Reference document: https://developers.eveonline.com/docs/services/sso/

## Architecture

### Core Components

- **Module**: Main auth module with route registration and background tasks
- **EVE SSO Handler**: OAuth2 flow implementation for EVE Online authentication
- **Profile Management**: User profile creation and management with ESI integration
- **JWT Middleware**: Authentication middleware for protected routes
- **Cross-Domain Cookies**: Secure cookie handling across subdomains

### Files Structure

```
internal/auth/
├── auth.go           # Main module with route handlers
├── eve_sso.go        # EVE Online SSO OAuth2 implementation
├── middleware.go     # JWT authentication middleware
└── profile.go        # User profile management with ESI data
```

## Authentication Flow

### 1. Login Initiation (Basic - No Scopes)
```
GET /auth/eve/login
```
- Generates cryptographically secure state parameter
- Creates EVE Online OAuth2 authorization URL without scopes
- Sets secure state cookie for CSRF protection
- Returns auth URL and state to frontend
- Checks the cookie, if present, and get the user_id
- Used for basic authentication without additional EVE permissions

### 2. Registration (Full Scopes)
```
GET /auth/eve/register
```
- Generates cryptographically secure state parameter
- Creates EVE Online OAuth2 authorization URL with full scopes from EVE_SCOPES environment variable
- Sets secure state cookie for CSRF protection
- Returns auth URL and state to frontend
- Used for full registration with all required EVE permissions


### 3. OAuth2 Callback
```
GET /auth/eve/callback?code=...&state=...
```
- Validates state parameter against stored cookie
- Exchanges authorization code for access token
- Validates JWT token using JWKS signature verification
- Verifies issuer, audience, and token claims
- Checks if there is the falcon_auth_token cookie and if it's valid extract the character_id
- Find the character_id in the database, if any get the user_id, if not generate a new user_id as UUID
- Save the character to the database using data from the Eveonline SSO response and the user_id
- Set valid as true
- Generates internal JWT token
- Sets secure authentication cookie
- Redirects to frontend application

### 4. Authentication Status
```
GET /auth/status
```
- Quick authentication check
- Returns `{authenticated: boolean}`

### 5. User Information
```
GET /auth/user
```
- Returns full user details if authenticated
- Includes character info, scopes, and expiration

### 6. Mobile Token Exchange
```
POST /auth/eve/token
```
- Accepts EVE SSO access token and optional refresh token
- Validates EVE access token with CCP
- Creates or updates user profile
- Returns JWT token for API access
- Designed for mobile apps that can't use cookies

### 7. Logout
```
POST /auth/logout
```
- Clears authentication cookie
- Returns success confirmation

## Security Features

### Cookie Security
- **Name**: `falcon_auth_token`
- **Domain**: `.eveonline.it` (cross-subdomain support)
- **Attributes**: Secure, HttpOnly, SameSite=Lax
- **Expiration**: 24 hours

### CSRF Protection
- State parameter validation
- Secure random state generation
- 15-minute state expiration
- Automatic cleanup of expired states

### JWT Tokens
- HMAC-SHA256 signed tokens
- 24-hour expiration
- Contains character ID, name, and scopes
- Server-side validation
- Supports both cookie and Bearer token authentication

## User Profile Management

### Profile Data
- Character ID and name
- Corporation and alliance information
- Security status and birthday
- EVE Online scopes
- Login timestamps
- Encrypted refresh token storage

### Bulk Token Management
- **RefreshExpiringTokens**: Batch refresh tokens for users with expiring access tokens
- **Configurable Batch Size**: Process tokens in configurable batches (default: 100)
- **Smart Expiration Detection**: Finds tokens expiring within the next hour
- **Comprehensive Error Handling**: Individual user failures don't stop the batch
- **Performance Optimized**: MongoDB aggregation pipeline for efficient queries

### JWT Token Validation
- **JWKS Integration**: Fetches and caches EVE Online's JSON Web Key Set (JWKS)
- **Signature Verification**: Validates JWT tokens using RSA public keys from JWKS
- **Claims Validation**: Verifies issuer ("login.eveonline.com") and audience ("EVE Online")
- **Key Caching**: Caches JWKS keys for 1 hour to reduce network requests
- **Security Compliance**: Follows EVE Online SSO best practices and OAuth 2.0 standards

### ESI Integration
- Real-time character data from EVE Online API
- Corporation and alliance information lookup
- Proper User-Agent headers for CCP compliance
- Error handling for ESI failures

### Database Storage
- MongoDB collection: `user_profiles`
- Upsert operations for create/update
- Indexed by character ID
- Refresh token encryption

## Middleware Usage

### JWT Middleware
```go
// Require authentication
r.With(m.JWTMiddleware).Get("/protected", handler)

// Optional authentication
r.With(m.OptionalJWTMiddleware).Get("/public", handler)

// Require specific EVE scopes
r.With(m.RequireScopes("esi-characters.read_contacts.v1")).Get("/contacts", handler)
```

### Context Access
```go
// Get authenticated user from context
user, ok := GetAuthenticatedUser(r)
if ok {
    characterID := user.CharacterID
    characterName := user.CharacterName
    scopes := user.Scopes
}
```

## Configuration

### Required Environment Variables
```bash
# EVE Online Application (register at developers.eveonline.com)
EVE_CLIENT_ID=your_application_client_id
EVE_CLIENT_SECRET=your_application_client_secret
EVE_REDIRECT_URI=https://go.eveonline.it/auth/eve/callback
EVE_SCOPES=publicData esi-characters.read_contacts.v1

# JWT Security
JWT_SECRET=your_very_long_random_jwt_secret_key

# Frontend Integration
FRONTEND_URL=https://react.eveonline.it
```

### Optional Configuration
```bash
# ESI User Agent (recommended)
ESI_USER_AGENT=go-falcon/1.0.0 (contact@example.com)
```

## API Endpoints

| Endpoint | Method | Auth Required | Description |
|----------|--------|---------------|-------------|
| `/auth/eve/login` | GET | No | Initiate EVE SSO login (basic, no scopes) |
| `/auth/eve/register` | GET | No | Initiate EVE SSO registration (full scopes from ENV) |
| `/auth/eve/callback` | GET | No | OAuth2 callback handler |
| `/auth/eve/verify` | GET | No | Verify JWT token |
| `/auth/eve/refresh` | POST | No | Refresh access token |
| `/auth/status` | GET | No | Quick auth status check |
| `/auth/user` | GET | No | Get current user info |
| `/auth/logout` | POST | No | Clear auth cookie |
| `/auth/profile` | GET | Yes | Get full user profile |
| `/auth/profile/refresh` | POST | Yes | Refresh profile from ESI |
| `/auth/profile/public` | GET | No | Get public profile by ID |
| `/auth/token` | GET | Yes | Retrieve current bearer token |
| `/auth/eve/token` | POST | No | Exchange EVE token for JWT (mobile) |

### Internal Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `RefreshExpiringTokens` | Batch refresh expiring tokens | `ctx`, `batchSize` | `successCount`, `failureCount`, `error` |
| `RefreshUserProfile` | Refresh single user profile | `ctx`, `characterID` | `*UserProfile`, `error` |
| `GetUserProfile` | Get user profile by character ID | `ctx`, `characterID` | `*UserProfile`, `error` |
| `CreateOrUpdateUserProfile` | Create/update user profile | `ctx`, `charInfo`, `userID`, `accessToken`, `refreshToken` | `*UserProfile`, `error` |

## Frontend Integration

### Authentication Check
```javascript
// Check if user is authenticated
fetch('/auth/status', { credentials: 'include' })
  .then(res => res.json())
  .then(data => {
    if (data.authenticated) {
      // User is logged in
    }
  });
```

### Web Login Flow
```javascript
// Basic Login (no scopes)
// 1. Get auth URL from backend for basic login
fetch('/auth/eve/login', { credentials: 'include' })
  .then(res => res.json())
  .then(data => {
    // 2. Redirect user to EVE Online for basic authentication
    window.location.href = data.auth_url;
  });

// Full Registration (with scopes)
// 1. Get auth URL from backend for full registration
fetch('/auth/eve/register', { credentials: 'include' })
  .then(res => res.json())
  .then(data => {
    // 2. Redirect user to EVE Online with full scope permissions
    window.location.href = data.auth_url;
  });

// 3. Handle callback redirect (automatic for both flows)
// 4. User will be redirected back to frontend with auth cookie
```

### Mobile Authentication Flow
```javascript
// 1. Mobile app handles EVE SSO flow directly
// 2. Once app has EVE access token, exchange for JWT
fetch('/auth/eve/token', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    access_token: eveAccessToken,
    refresh_token: eveRefreshToken // optional
  })
})
.then(res => res.json())
.then(data => {
  // Store JWT token for API calls
  const jwtToken = data.jwt_token;
  
  // Use Bearer token for subsequent API calls
  fetch('/protected-endpoint', {
    headers: {
      'Authorization': `Bearer ${jwtToken}`
    }
  });
});
```

### Logout
```javascript
fetch('/auth/logout', { 
  method: 'POST',
  credentials: 'include' 
})
.then(res => res.json())
.then(data => {
  if (data.success) {
    // User logged out, update UI
  }
});
```

## Background Tasks

### State Cleanup
- Runs every 5 minutes
- Removes expired OAuth2 states
- Prevents memory leaks

### Token Refresh Integration
- **Scheduler Integration**: Provides `RefreshExpiringTokens` method for the scheduler module
- **Automated Execution**: System task `system-token-refresh` runs every 15 minutes
- **Batch Processing**: Configurable batch size (default: 100 users per run)
- **Performance Monitoring**: Detailed logging and metrics for success/failure rates

### Implementation Notes
- Thread-safe operations
- Graceful shutdown handling
- Context-aware cancellation

## Error Handling

### Common Error Scenarios
- Missing or invalid state parameters
- Expired authentication tokens
- ESI API failures
- Database connection issues
- Invalid JWT tokens

### Logging
- Structured logging with slog
- Authentication events
- Error conditions
- Security warnings
- Performance metrics (excluding health checks)

## Best Practices

### Security
- Always use HTTPS in production
- Regularly rotate JWT secrets
- Monitor for suspicious authentication patterns
- Implement rate limiting on auth endpoints

### Performance
- Profile data caching
- Efficient database queries
- Background token refresh
- Health check exclusion from logs

### Maintenance
- Regular cleanup of expired sessions
- Monitor ESI error rates
- Update EVE scopes as needed
- Keep CCP User-Agent requirements current

## Dependencies

### External Services
- EVE Online SSO (login.eveonline.com)
- EVE Online ESI (esi.evetech.net)
- MongoDB (user profiles)
- Redis (session storage - future)

### Go Packages
- `github.com/golang-jwt/jwt/v5` - JWT handling and validation
- `go.mongodb.org/mongo-driver` - MongoDB client
- `go-falcon/pkg/evegateway` - ESI client
- `go-falcon/pkg/config` - Configuration management

### EVE Online SSO Compliance
- **JWT Validation**: Proper JWT token validation using JWKS endpoint
- **Signature Verification**: RSA signature verification with cached public keys
- **Claims Validation**: Validates issuer, audience, and token expiration
- **JWKS Caching**: Implements 1-hour caching as recommended by CCP
- **Error Handling**: Comprehensive error handling for token validation failures
- **Security**: Follows OAuth 2.0 and EVE Online SSO security best practices

## Testing

### Unit Tests
```bash
go test ./internal/auth/...
```

### Integration Tests
- OAuth2 flow validation
- JWT token lifecycle
- Profile management
- Middleware functionality

### Manual Testing
- Complete authentication flow
- Cookie persistence across domains
- Token refresh functionality
- Logout behavior