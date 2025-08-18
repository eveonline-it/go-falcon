# Configuration Package (pkg/config)

## Overview
Centralized configuration management using environment variables with sensible defaults. Handles all application settings including EVE Online integration, database connections, and feature flags.

## Core Features
- **Environment Variables**: Safe access with defaults and validation
- **Type Conversion**: Automatic string to bool/int conversion
- **Required Variables**: `MustGetEnv()` for critical configuration
- **EVE Online Config**: Complete SSO and ESI configuration management

## Key Functions
- `GetEnv(key, default)`: Get string with fallback
- `GetBoolEnv(key, default)`: Get boolean with fallback  
- `GetIntEnv(key, default)`: Get integer with fallback
- `MustGetEnv(key)`: Get required variable or panic
- `GetHumaPort()`: Get HUMA server port (HUMA_PORT)
- `GetHumaSeparateServer()`: Get separate server flag (HUMA_SEPARATE_SERVER)

## EVE Online Configuration
```go
// Required for EVE SSO
GetEVEClientID()       // EVE_CLIENT_ID
GetEVEClientSecret()   // EVE_CLIENT_SECRET  
GetJWTSecret()         // JWT_SECRET

// Optional with defaults
GetEVERedirectURI()    // EVE_REDIRECT_URI
GetEVEScopes()         // EVE_SCOPES
GetFrontendURL()       // FRONTEND_URL
```

## Configuration Categories
- **EVE Online**: SSO credentials, redirect URIs, scopes
- **Frontend**: React application URL for redirects
- **API**: Prefix configuration for versioning
- **HUMA Server**: Port configuration and server mode selection
- **Security**: JWT secrets and token management

## Usage Pattern
All modules use this package for consistent configuration access across the monolith.