# Go Falcon - Go Monolith Project

## Project Overview

This is a GO architecture project featuring:
- Monolith modular architecture
- API Gateway using net/http with Chi
- Multiple services running background tasks
- Comprehensive task scheduler with cron scheduling and distributed locking
- Shared libraries
- Redis (caching/session management/distributed locking)
- MongoDB (primary database)
- Docker Compose orchestration
- Websocket using Socket.io via Redis Adapter
- OpenAPI 3.1.1 compliance
- Internationalization/Translations (I18N) 
- OpenTelemetry logs and tracing (check env ENABLE_TELEMETRY variable)
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
  - postman: export all endpoints of the gateway in postman format
  - openapi: export all endpoints in openapi 3.1 format
- internal: private packages
  - auth: authentication service module with EVE Online SSO integration
  - dev: development module for testing and calling other services
  - notifications: notification service module
  - scheduler: comprehensive task scheduling and management service
  - sde: web-based SDE management with automated processing and scheduler integration
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
6. Use cmd/postman when removing, updating or inserting new endpoints

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

The project provides comprehensive EVE Online SDE management through both in-memory access (`pkg/sde`) and web-based management (`internal/sde`).

### SDE Management Architecture
- **Web-Based Management**: `internal/sde` module provides REST API for SDE operations
- **Background Processing**: Automated download, conversion, and storage of SDE data
- **Scheduler Integration**: System task checks for updates every 6 hours
- **Redis JSON Storage**: Individual SDE entities stored as separate Redis JSON keys for granular access
- **Progress Tracking**: Real-time progress updates during SDE processing

### SDE Service (pkg/sde)
- **Single Instance**: One SDE service instance shared across all modules in the monolith
- **In-Memory Storage**: Data loaded at startup for ultra-fast access (nanosecond lookups)
- **Type-Safe Access**: Structured Go types with proper JSON unmarshaling
- **Lazy Loading**: Data loaded on first access to optimize startup time
- **Thread-Safe**: Concurrent access via read-write mutexes

### Data Sources and Processing
- **Source Data**: Downloaded automatically from CCP's SDE distribution
- **Processing Pipeline**: YAML → JSON conversion with individual Redis JSON key storage
- **Update Detection**: MD5 hash comparison for new version detection
- **Web Management**: RESTful API for manual updates and status monitoring

### SDE Management Endpoints
- `GET /sde/status` - Current SDE version and status
- `POST /sde/check` - Check for new SDE versions
- `POST /sde/update` - Initiate SDE update process (processes ALL YAML files in bsd, fsd, and universe directories)
- `GET /sde/progress` - Real-time update progress
- `GET /sde/entity/{type}/{id}` - Get individual SDE entity
- `GET /sde/entities/{type}` - Get all entities of a specific type

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

```bash
# Direct API access to individual entities
curl http://localhost:8080/sde/entity/agents/3008416
curl http://localhost:8080/sde/entities/categories
```

### Scheduler Integration
- **System Task**: `system-sde-check` runs every 6 hours
- **Automatic Detection**: Checks for new SDE versions and notifies
- **Background Updates**: Optional automatic SDE processing
- **Status Monitoring**: Comprehensive update status tracking

### Performance Characteristics
- **Memory Usage**: ~50-500MB for in-memory data
- **Redis Storage**: ~50-500MB for individual JSON keys (same total, different structure)
- **Access Speed**: Direct map/slice lookups (O(1) or O(log n))
- **Update Processing**: 2-5 minutes for full SDE conversion
- **Network Efficient**: Only downloads when updates are available

### Data Types Available
Current SDE data includes:
- **Agents**: Mission agents with location and corporation info
- **Categories**: Item categories with internationalized names
- **Blueprints**: Manufacturing blueprints with material requirements
- **Market Groups**: Market categorization and hierarchy
- **Types**: Complete item database with attributes
- **NPC Corporations**: Corporation data with faction information

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

The project includes complete EVE Online Single Sign-On (SSO) authentication integration with support for both web and mobile applications:

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
- **Multi-Platform Support**: Cookie-based auth for web, Bearer token auth for mobile apps

### Authentication Endpoints
- `GET /auth/eve/login` - Initiate EVE SSO authentication
- `GET /auth/eve/callback` - Handle OAuth2 callback
- `GET /auth/eve/verify` - Verify JWT token validity
- `POST /auth/eve/refresh` - Refresh access tokens
- `POST /auth/eve/token` - Exchange EVE tokens for JWT (mobile apps)

### Profile Endpoints
- `GET /auth/profile` - Get authenticated user's full profile
- `POST /auth/profile/refresh` - Refresh profile data from ESI
- `GET /auth/profile/public` - Get public character information

### Middleware
- **JWTMiddleware**: Require valid authentication (supports cookies and Bearer tokens)
- **OptionalJWTMiddleware**: Add user context if authenticated (supports both methods)
- **RequireScopes**: Enforce specific EVE Online permissions

### Mobile App Integration
For mobile applications that cannot use HTTP-only cookies:

1. **EVE SSO Flow**: Mobile app handles EVE Online OAuth2 flow directly
2. **Token Exchange**: POST EVE access token to `/auth/eve/token`
3. **JWT Token**: Receive JWT token for API authentication
4. **Bearer Authentication**: Use `Authorization: Bearer <jwt_token>` header for all API calls

```javascript
// Mobile app token exchange example
const response = await fetch('/auth/eve/token', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    access_token: eveAccessToken,
    refresh_token: eveRefreshToken // optional
  })
});

const { jwt_token } = await response.json();

// Use JWT for API calls
fetch('/protected-endpoint', {
  headers: { 'Authorization': `Bearer ${jwt_token}` }
});
```

### Documentation
Complete integration documentation available in `docs/EVE_SSO_INTEGRATION.md`

## Module-Specific Documentation

The following modules have detailed CLAUDE.md documentation files with comprehensive implementation details:

### Authentication Module
- **Location**: `internal/auth/CLAUDE.md`
- **Coverage**: Complete EVE Online SSO integration, JWT middleware, user profile management, security features, API endpoints, and frontend integration examples
- **Key Features**: OAuth2 flow, cross-subdomain cookies, CSRF protection, ESI integration, background tasks

### Development Module
- **Location**: `internal/dev/CLAUDE.md`
- **Coverage**: ESI testing endpoints, SDE data access, cache management, telemetry integration, and development utilities
- **Key Features**: EVE Online API testing, static data validation, performance monitoring, debugging tools

### Scheduler Module
- **Location**: `internal/scheduler/CLAUDE.md`
- **Coverage**: Complete task scheduling and management system with cron scheduling, worker pool execution, distributed locking, and comprehensive API
- **Key Features**: HTTP/Function/System task types, Redis-based distributed locking, execution history, monitoring capabilities, hardcoded system tasks

### SDE Module
- **Location**: `internal/sde/CLAUDE.md`
- **Coverage**: Web-based EVE Online SDE management with automated processing, progress tracking, and scheduler integration
- **Key Features**: REST API for SDE operations, individual Redis JSON key storage, background processing, hash-based update detection, granular entity access, real-time progress updates

### Package Documentation

The following shared packages have detailed CLAUDE.md documentation:

#### Core Infrastructure
- **Application**: `pkg/app/CLAUDE.md` - Application initialization and context management
- **Configuration**: `pkg/config/CLAUDE.md` - Environment variable management and settings
- **Database**: `pkg/database/CLAUDE.md` - MongoDB and Redis connection utilities
- **Module System**: `pkg/module/CLAUDE.md` - Base module interface and common functionality

#### EVE Online Integration
- **EVE Gateway**: `pkg/evegateway/CLAUDE.md` - Complete ESI client library with caching and compliance
- **Static Data Export**: `pkg/sde/CLAUDE.md` - In-memory EVE Online static data service

#### Observability & Utilities
- **Logging**: `pkg/logging/CLAUDE.md` - OpenTelemetry logging and telemetry management
- **Handlers**: `pkg/handlers/CLAUDE.md` - Shared HTTP handlers and utilities
- **Middleware**: `pkg/middleware/CLAUDE.md` - OpenTelemetry tracing middleware
- **Version**: `pkg/version/CLAUDE.md` - Application version and build information

### Future Module Documentation
As additional modules are enhanced with detailed documentation, they will be listed here:
- `internal/users/CLAUDE.md` - User management system (planned)  
- `internal/notifications/CLAUDE.md` - Notification service (planned)

## How to Use Module Documentation

1. **Start with this root CLAUDE.md** for project overview and architecture
2. **Navigate to specific module CLAUDE.md files** for detailed implementation guidance
3. **Use module docs for**:
   - API endpoint references
   - Configuration requirements
   - Integration examples
   - Security considerations
   - Best practices

Each module's CLAUDE.md provides complete documentation for developers working with that specific component, including code examples, configuration, and troubleshooting guides.
