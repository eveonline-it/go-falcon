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

- cmd: This directory contains the main applications for the different services. Each subdirectory is a runnable application:
  - gateway: Main API gateway application
  - backup: backup application for MongoDb and Redis
  - restore: a restore application for MongoDb and Redis
- internal: This directory contains packages that are shared across the different services but are not intended to be used by other applications.
- pkg: This directory contains packages that are intended to be shared and used by other applications.
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

## OpenTelemetry Logging

The project implements logging with OpenTelemetry integration following the OpenTelemetry Specification 1.47.0:

### Features
- **Structured Logging**: JSON-formatted logs with trace correlation
- **OpenTelemetry Integration**: Automatic trace and span ID injection
- **Multiple Transports**: Console for development, OTLP HTTP transport
- **Context Support**: Service-specific logging contexts
- **Graceful Shutdown**: Proper cleanup of telemetry resources
