# Go Falcon - Go Monolith Project

## Project Overview

This is a GO architecture project featuring:
- Monolith modular architecture
- API Gateway using net/http with Chi
- Multiple services running background tasks
- Shared libraries
- Redis (caching/session management)
- MongoDB (primary database)
- Docker Compose orchestration
- Websocket using Socket.io via Redis Adapter
- OpenAPI 3.1.1 compliance
- Internationalization/Translations (I18N) 
- OpenTelemetry logs and tracing
- EVE Online SSO authentication with JWT tokens

## Project details

- Production Ready: multi-stage Dockerfiles
- Development Flexibility: run services individually
- Auto-reload: Hot reload in development mode
- Metrics, Traces and Logs are collected via Opentelemetry
- Graceful Shutdown
- Opentelemetry logging should adhere to OpenTelemetry Specification 1.47.0

## Directory Structure
The project's code is organized into several directories:

- cmd: main runnable applications
  - gateway: Main API gateway application
  - backup: backup application for MongoDb and Redis
  - restore: a restore application for MongoDb and Redis
  - postman: export all the endpoints of the gateway
  - sde: convert SDE to JSON files stored in data/sde
- internal: private packages
  - auth: authentication service module with EVE Online SSO integration
  - dev: development module for testing and calling other services
  - notifications: notification service module
  - users: user management service module
- pkg: shared libraries
  - app: application initialization and context management
  - config: configuration management and environment variables
  - database: MongoDB and Redis connection utilities
  - handlers: shared HTTP handlers and utilities
  - logging: OpenTelemetry logging and telemetry management
  - module: base module interface and common functionality
  - sde: EVE Online Static Data Export (SDE) in-memory service
  - version: application version information
  - evegateway: EVE Online ESI client library for API integration
- docs: This directory contains the documentation for the project, including deployment guides, API definitions, and functional requirements.
- examples: This directory contains example code, such as an example for exporting data.
- builders: This directory contains files related to building the project, such as Dockerfiles.
- scripts: Development and deployment scripts for the project.

## Quick Start

### Prerequisites
- Docker & Docker Compose
- GOLang 1.24.5
- go-chi/chi v5.2.2

## Best Practices

1. **Code Organization**: Use shared libraries for common functionality
2. **Error Handling**: Implement global exception filters
4. **Security**: Implement JWT authentication
5. **Documentation**: Keep OpenAPI specs updated
6. **Testing**: Maintain good test coverage
7. **Monitoring**: Use health checks and logging
8. **Performance**: Implement caching strategies

## Contributing

1. Follow the established coding standards
2. Write tests for new features
3. Update documentation
4. Use conventional commits
5. Create feature branches for new development

## Development Notes

- Keep shared libraries lightweight and focused
- Document API changes in OpenAPI specs

## EVE Online ESI Integration Guidelines

The `pkg/evegateway` package handles EVE Online ESI (Electronic System Interface) API calls and must follow CCP's best practices:

### User Agent Requirements
- **REQUIRED**: All ESI requests must include a proper User-Agent header
- **Format**: Include email, app name/version, source code URL, Discord username, or EVE character name
- **Example**: `"go-falcon/1.0.0 (contact@yourapp.com) +https://github.com/yourorg/go-falcon"`
- **Browser Fallback**: Use `X-User-Agent` header or `user_agent` query parameter if headers can't be set

### Rate Limiting & Error Handling  
- **Error Limit System**: ESI tracks errors per application
- **Monitor Headers**: Check `X-ESI-Error-Limit-Remain` and `X-ESI-Error-Limit-Reset` headers
- **Consequences**: Exceeding error limit results in request blocking
- **Implementation**: Always handle HTTP error responses properly

### Caching Strategy
- **MANDATORY**: Respect the `expires` header for cache duration
- **Conditional Requests**: Use `Last-Modified` and `ETag` headers for efficient caching
- **Cache Headers**: Implement proper HTTP caching with `If-None-Match` and `If-Modified-Since`
- **No Circumvention**: Never request data before cache expiration
- **Consequences**: Ignoring cache requirements can lead to API access restrictions

### Best Practices Implementation
```go
// Example ESI client configuration
client := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        // Connection pooling for efficiency
    },
}

// Required headers for all requests
req.Header.Set("User-Agent", "go-falcon/1.0.0 (your-email@domain.com)")
req.Header.Set("Accept", "application/json")

// Implement caching headers
if cachedETag != "" {
    req.Header.Set("If-None-Match", cachedETag)
}
if lastModified != "" {
    req.Header.Set("If-Modified-Since", lastModified)
}
```

### Error Handling Requirements
- Check HTTP status codes (200, 304, 404, 420, 500, etc.)
- Parse error limit headers and implement backoff
- Handle 304 Not Modified responses for cached data
- Implement exponential backoff for 5xx errors
- Never retry 4xx errors (except 420 rate limit)

### ESI as Shared Resource
- Treat ESI as a shared resource across all EVE applications
- Implement responsible usage patterns
- Cache data appropriately to reduce server load
- Follow all CCP guidelines to maintain API access for the community

**Reference**: [EVE Online ESI Best Practices](https://developers.eveonline.com/docs/services/esi/best-practices/)

## EVE Online SDE (Static Data Export) Integration

The `pkg/sde` package provides in-memory access to EVE Online's Static Data Export, offering fast lookups for game data like agents, categories, blueprints, and more.

### Architecture Overview
- **Single Instance**: One SDE service instance shared across all modules in the monolith
- **In-Memory Storage**: Data loaded at startup for ultra-fast access (nanosecond lookups)
- **Type-Safe Access**: Structured Go types with proper JSON unmarshaling
- **Lazy Loading**: Data loaded on first access to optimize startup time
- **Thread-Safe**: Concurrent access via read-write mutexes

### Data Sources
- **Source Data**: `data/sde/*.json` files converted from CCP's YAML format
- **Conversion Tool**: `cmd/sde/main.go` downloads and converts SDE data
- **Update Process**: Run SDE tool when CCP releases new static data

### Usage Patterns
```go
// Access SDE data from any module
agent, err := sdeService.GetAgent("3008416")
category, err := sdeService.GetCategory("1")
blueprint, err := sdeService.GetBlueprint("1000001")

// Query operations
agents := sdeService.GetAgentsByLocation(60000004)
categories := sdeService.GetPublishedCategories()
```

### Integration with Modules
- **Initialization**: SDE service initialized in `pkg/app/init.go`
- **Module Access**: Available through base module interface
- **ESI Enrichment**: Combines with `pkg/evegateway` for live + static data
- **Internationalization**: Supports multiple languages from SDE data

### Performance Characteristics
- **Memory Usage**: ~50-500MB depending on SDE data size
- **Access Speed**: Direct map/slice lookups (O(1) or O(log n))
- **Startup Impact**: 1-2 second initial load time
- **No External Dependencies**: No Redis/database calls for SDE data

### Data Types Available
Current SDE data includes:
- **Agents**: Mission agents with location and corporation info
- **Categories**: Item categories with internationalized names
- **Blueprints**: Manufacturing blueprints with material requirements
- **Extensible**: Easy to add more SDE data types as needed

## OpenTelemetry Logging

The project implements logging with OpenTelemetry integration following the OpenTelemetry Specification 1.47.0:

### Configuration
- **Environment Control**: Telemetry is only active when `ENABLE_TELEMETRY=true` in the environment
- **Default Behavior**: When `ENABLE_TELEMETRY=false` or unset, telemetry features are disabled
- **Production Ready**: Safe to deploy with telemetry disabled in environments where it's not needed

### Features
- **Structured Logging**: JSON-formatted logs with trace correlation
- **OpenTelemetry Integration**: Automatic trace and span ID injection
- **Multiple Transports**: Console for development, OTLP HTTP transport
- **Context Support**: Service-specific logging contexts
- **Graceful Shutdown**: Proper cleanup of telemetry resources
- **Conditional Activation**: All telemetry features respect the ENABLE_TELEMETRY environment variable

## EVE Online SSO Integration

The project includes complete EVE Online Single Sign-On (SSO) authentication integration:

### Configuration
- **EVE Application Registration**: Applications must be registered at [developers.eveonline.com](https://developers.eveonline.com/)
- **Environment Variables**: Required: `EVE_CLIENT_ID`, `EVE_CLIENT_SECRET`, `JWT_SECRET`
- **Optional Settings**: `EVE_REDIRECT_URI`, `EVE_SCOPES`, `ESI_USER_AGENT`

### Features
- **OAuth2 Authorization Code Flow**: Secure authentication with state validation
- **JWT Token Management**: Internal session tokens with configurable expiration
- **Character Profile Integration**: Automatic retrieval and storage of character data
- **ESI Integration**: Real-time data from EVE's ESI API (character, corporation, alliance)
- **Security Best Practices**: CSRF protection, secure cookies, proper token validation

### Authentication Endpoints
- `GET /auth/eve/login` - Initiate EVE SSO authentication
- `GET /auth/eve/callback` - Handle OAuth2 callback
- `GET /auth/eve/verify` - Verify JWT token validity
- `POST /auth/eve/refresh` - Refresh access tokens

### Profile Endpoints
- `GET /auth/profile` - Get authenticated user's full profile
- `POST /auth/profile/refresh` - Refresh profile data from ESI
- `GET /auth/profile/public` - Get public character information

### Middleware
- **JWTMiddleware**: Require valid authentication
- **OptionalJWTMiddleware**: Add user context if authenticated
- **RequireScopes**: Enforce specific EVE Online permissions

### Documentation
Complete integration documentation available in `docs/EVE_SSO_INTEGRATION.md`
