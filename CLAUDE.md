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
- internal: private packages
  - auth: authentication service module
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
  - version: application version information
  - evegate: EVE Online ESI client library for API integration
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

The `pkg/evegate` package handles EVE Online ESI (Electronic System Interface) API calls and must follow CCP's best practices:

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

## OpenTelemetry Logging

The project implements logging with OpenTelemetry integration following the OpenTelemetry Specification 1.47.0:

### Features
- **Structured Logging**: JSON-formatted logs with trace correlation
- **OpenTelemetry Integration**: Automatic trace and span ID injection
- **Multiple Transports**: Console for development, OTLP HTTP transport
- **Context Support**: Service-specific logging contexts
- **Graceful Shutdown**: Proper cleanup of telemetry resources
