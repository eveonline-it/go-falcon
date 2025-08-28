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
- `GetHost()`: Get main server host interface (HOST, default: 0.0.0.0)
- `GetHumaHost()`: Get HUMA server host interface (HUMA_HOST, defaults to HOST)
- `GetSDEURL()`: Get SDE download URL (SDE_URL)
- `GetSDEChecksumsURL()`: Get SDE checksums file URL (SDE_CHECKSUMS_URL)

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
- **SDE**: Static Data Export download URLs and checksums

## SDE Configuration
```go
// SDE download configuration
GetSDEURL()            // SDE_URL (default: AWS S3 EVE SDE ZIP)
GetSDEChecksumsURL()   // SDE_CHECKSUMS_URL (default: AWS S3 checksum file)
```

## Usage Pattern
All modules use this package for consistent configuration access across the monolith.