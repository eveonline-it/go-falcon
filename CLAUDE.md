# Go Falcon - Monolithic API Gateway

A production-ready Go monolithic architecture featuring modular design, EVE Online integration, and comprehensive task scheduling capabilities.

## 🚀 Overview

Go Falcon is a monolithic API gateway built with Go that provides:

- **Type-Safe APIs**: Huma v2 framework with compile-time validation
- **Modular Architecture**: Clean separation of concerns with internal modules
- **EVE Online Integration**: Complete SSO authentication and ESI API integration
- **Task Scheduling**: Distributed task scheduling with cron support
- **Real-time Communication**: WebSocket support via Socket.io and Redis
- **Observability**: OpenTelemetry logging and tracing
- **API Standards**: Automatic OpenAPI 3.1.1 generation via Huma v2

## 📋 Table of Contents

- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Core Features](#core-features)
- [Module Documentation](#module-documentation)
- [EVE Online Integration](#eve-online-integration)
- [Permission System](#permission-system)
- [Development Guidelines](#development-guidelines)
- [API Documentation](#api-documentation)

## 🏗️ Architecture

### Directory Structure

```
go-falcon/
├── cmd/                    # Executable applications
│   ├── gateway/           # Main API gateway
│   ├── backup/            # MongoDB/Redis backup utility
│   └── restore/           # Data restoration utility
├── internal/              # Private application modules
│   ├── auth/             # EVE SSO authentication
│   │   ├── dto/          # Request/Response structures
│   │   ├── middleware/   # Auth-specific middleware
│   │   ├── routes/       # Route definitions
│   │   ├── services/     # Business logic
│   │   ├── models/       # Database schemas
│   │   └── CLAUDE.md     # Module documentation
│   ├── scheduler/        # Task scheduling system
│   │   ├── dto/          # Request/Response structures
│   │   ├── middleware/   # Auth-specific middleware
│   │   ├── routes/       # Route definitions
│   │   ├── services/     # Business logic
│   │   ├── models/       # Database schemas
│   │   └── CLAUDE.md     # Module documentation
│   └── users/            # User management
│   │   ├── dto/          # Request/Response structures
│   │   ├── middleware/   # Auth-specific middleware
│   │   ├── routes/       # Route definitions
│   │   ├── services/     # Business logic
│   │   ├── models/       # Database schemas
│   │   └── CLAUDE.md     # Module documentation
├── pkg/                   # Shared libraries
│   ├── app/              # Application lifecycle
│   ├── config/           # Configuration management
│   ├── database/         # Database connections
│   ├── evegateway/       # EVE ESI client
│   ├── handlers/         # HTTP utilities
│   ├── logging/          # Telemetry & logging
│   ├── middleware/       # HTTP middleware
│   ├── module/           # Module system
│   ├── sde/              # In-memory SDE service
│   └── version/          # Version information
├── docs/                  # Documentation
├── builders/             # Docker configurations
└── scripts/              # Automation scripts
```

### Technology Stack

- **Language**: Go 1.24.5
- **API Framework**: Huma v2 with Chi v5.2.2 adapter
- **Databases**: MongoDB (primary), Redis (caching/sessions)
- **Container**: Docker & Docker Compose
- **Observability**: OpenTelemetry
- **API Spec**: Automatic OpenAPI 3.1.1 generation
- **Authentication**: JWT with EVE Online SSO

### Production Features

- ✅ Multi-stage Docker builds
- ✅ Hot reload in development
- ✅ Graceful shutdown
- ✅ Distributed locking
- ✅ Comprehensive error handling
- ✅ Request tracing and correlation
- ✅ Health checks and metrics

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.24.5+
- MongoDB 6.0+
- Redis 7.0+

### Environment Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-org/go-falcon.git
   cd go-falcon
   ```

2. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start services**
   ```bash
   docker-compose up -d
   ```

### Key Environment Variables

```bash
# Server Configuration
HOST="0.0.0.0"                 # Host interface to bind to (all interfaces)
PORT="8080"                    # Main server port

# API Configuration
API_PREFIX="/api"              # API route prefix (empty for root)
JWT_SECRET="your-secret-key"   # JWT signing key

# OpenAPI Configuration
OPENAPI_SERVERS=""             # Custom OpenAPI servers (optional)
                              # Format: "url1|description1,url2|description2"
                              # Example: "https://api.prod.com|Production,https://api.staging.com|Staging"

# HUMA Server Configuration (Future Feature)
# HUMA_PORT="8081"               # Reserved for future separate HUMA server
# HUMA_HOST="0.0.0.0"            # Reserved for future separate HUMA server  
# HUMA_SEPARATE_SERVER="false"   # Currently disabled - will be reimplemented

# EVE Online Integration
EVE_CLIENT_ID="your-client-id"
EVE_CLIENT_SECRET="your-secret"
SUPER_ADMIN_CHARACTER_ID="123456789"

# Observability
ENABLE_TELEMETRY="true"        # Enable OpenTelemetry
```

## 🎯 Core Features

### 1. Modular Architecture

Each module in `internal/` is self-contained with:
- Dedicated routes and handlers
- Service-specific business logic
- Independent database collections
- Module-specific middleware
- Comprehensive documentation (CLAUDE.md)

### 2. Unified OpenAPI Architecture

Modern API gateway with unified OpenAPI 3.1.1 specification:

**Single API Specification**
- All modules documented in one comprehensive OpenAPI spec
- Unified schema registry with shared types across modules
- Environment-aware server configuration
- Modern API standards compliance

**Flexible API Prefix Support**
- Configure API versioning via `API_PREFIX` environment variable
- Supports deployment patterns: `/api`, `/v1`, `/v2`, etc.
- OpenAPI servers field automatically configured for different environments
- Backward compatible with existing deployment strategies

**Future: Separate Server Mode**
- HUMA separate server mode currently disabled during architectural refactor
- Will be reimplemented with unified OpenAPI support
- For now, all APIs served from main server with unified specification

### 3. Task Scheduling System

The scheduler module provides:
- **Cron Scheduling**: Standard cron expression support
- **Task Types**: HTTP webhooks, functions, system tasks
- **Distributed Locking**: Redis-based coordination
- **Execution History**: Complete audit trail
- **Worker Pool**: Configurable concurrent execution

### 4. EVE Online SDE Management

In-memory SDE (Static Data Export) service:
- **In-Memory Service** (`pkg/sde`): Ultra-fast data access for EVE game data
- **Preserved Interface**: Maintains compatibility for modules that may need SDE data

### 5. Authentication & Security

- **EVE Online SSO**: OAuth2 integration
- **JWT Tokens**: Stateless authentication
- **Dual Auth Support**: Cookies (web) and Bearer tokens (mobile)
- **Granular Permissions**: Fine-grained access control
- **CSRF Protection**: State validation

## 📚 Module Documentation

### Core Modules with Detailed Documentation

| Module | Location | Description |
|--------|----------|-------------|
| **Authentication** | [`internal/auth/CLAUDE.md`](internal/auth/CLAUDE.md) | EVE SSO integration, JWT management, user profiles |
| **Scheduler** | [`internal/scheduler/CLAUDE.md`](internal/scheduler/CLAUDE.md) | Task scheduling, cron jobs, distributed execution |
| **Users** | [`internal/users/CLAUDE.md`](internal/users/CLAUDE.md) | User management and profile operations |

### Shared Package Documentation

| Package | Location | Purpose |
|---------|----------|---------|
| **App** | [`pkg/app/CLAUDE.md`](pkg/app/CLAUDE.md) | Application lifecycle management |
| **Config** | [`pkg/config/CLAUDE.md`](pkg/config/CLAUDE.md) | Environment configuration |
| **Database** | [`pkg/database/CLAUDE.md`](pkg/database/CLAUDE.md) | MongoDB/Redis utilities |
| **EVE Gateway** | [`pkg/evegateway/CLAUDE.md`](pkg/evegateway/CLAUDE.md) | ESI client library |
| **Handlers** | [`pkg/handlers/CLAUDE.md`](pkg/handlers/CLAUDE.md) | HTTP response utilities |
| **Logging** | [`pkg/logging/CLAUDE.md`](pkg/logging/CLAUDE.md) | OpenTelemetry integration |
| **Middleware** | [`pkg/middleware/CLAUDE.md`](pkg/middleware/CLAUDE.md) | Request processing |
| **Module** | [`pkg/module/CLAUDE.md`](pkg/module/CLAUDE.md) | Module system base |
| **SDE Service** | [`pkg/sde/CLAUDE.md`](pkg/sde/CLAUDE.md) | In-memory data service |

## 🚀 EVE Online Integration

### ESI (EVE Swagger Interface) Best Practices

The project strictly follows [CCP's ESI guidelines](https://developers.eveonline.com/docs/services/esi/best-practices/):

#### Required Headers
```go
// User-Agent is MANDATORY
req.Header.Set("User-Agent", "go-falcon/1.0.0 (admin@example.com)")
req.Header.Set("Accept", "application/json")
```

#### Caching Requirements
- **Respect `expires` headers**: Never request before expiration
- **Use conditional requests**: Implement ETag/Last-Modified
- **Handle 304 responses**: Properly use cached data
- **Monitor error limits**: Check `X-ESI-Error-Limit-*` headers

#### Error Handling
- Implement exponential backoff for 5xx errors
- Never retry 4xx errors (except 420)
- Track error budget to avoid blocking
- Log all error responses for debugging

### Authentication Flow

1. **Web Applications**: Cookie-based authentication
   ```
   GET /auth/eve/login → EVE SSO → /auth/eve/callback → JWT Cookie
   ```

2. **Mobile Applications**: Bearer token authentication
   ```
   POST /auth/eve/token → Exchange EVE token → JWT Bearer token
   ```

### Profile Management

- **Full Profile**: `GET /auth/profile` (authenticated)
- **Public Info**: `GET /auth/profile/public` (open)
- **Refresh Data**: `POST /auth/profile/refresh`
- **Token Access**: `GET /auth/token` (get bearer token)

## 🔐 Permission System

### Granular Permissions Model

Permissions follow a **Service.Resource.Action** pattern:

```
scheduler.tasks.read     # Read scheduled tasks
sde.entities.write      # Modify SDE data
users.profiles.admin    # Full user management
```

### Action Hierarchy

1. **read** - View data
2. **write** - Modify data
3. **delete** - Remove data
4. **admin** - Full control

### Permission Management

Super admin endpoints for permission control:

```bash
# Create service definition
POST /admin/permissions/services

# Assign permissions to groups
POST /admin/permissions/assignments

# Check user permissions
POST /admin/permissions/check
```

### Subject Types

- **member** - Individual character
- **group** - User groups (recommended)
- **corporation** - EVE corporation
- **alliance** - EVE alliance

## 🛠️ Development Guidelines

### Module Structure Standards

Each module in `internal/` **MUST** follow this standardized structure:

```
internal/modulename/
├── dto/                    # Data Transfer Objects
│   ├── requests.go        # Request DTOs with validation
│   ├── responses.go       # Response DTOs
│   └── validators.go      # Custom validation logic
├── middleware/            # Module-specific middleware
│   ├── auth.go           # Authentication middleware
│   ├── validation.go     # Request validation
│   └── ratelimit.go      # Rate limiting (if needed)
├── routes/               # Route definitions
│   ├── routes.go         # Main route registration
│   ├── health.go         # Health check endpoints
│   └── api.go            # API endpoint handlers
├── services/             # Business logic
│   ├── service.go        # Main service implementation
│   └── repository.go     # Database operations
├── models/               # Database models
│   └── models.go         # MongoDB/Redis schemas
├── module.go             # Module initialization
└── CLAUDE.md             # Module documentation

```

#### Example Module Structure Implementation

```go
// internal/mymodule/dto/requests.go
package dto

import "github.com/go-playground/validator/v10"

type CreateTaskRequest struct {
    Name        string `json:"name" validate:"required,min=3,max=100"`
    Description string `json:"description" validate:"max=500"`
    CronExpr    string `json:"cron_expression" validate:"required,cron"`
}

// internal/mymodule/dto/responses.go
package dto

type TaskResponse struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
}

// internal/mymodule/routes/routes.go
package routes

func (m *Module) RegisterRoutes(r chi.Router) {
    // Public routes
    r.Group(func(r chi.Router) {
        r.Get("/health", m.HealthCheck)
    })
    
    // Protected routes
    r.Group(func(r chi.Router) {
        r.Use(m.middleware.RequireAuth)
        r.Use(m.middleware.ValidateRequest)
        
        r.Post("/tasks", m.CreateTask)
        r.Get("/tasks", m.ListTasks)
        r.Get("/tasks/{id}", m.GetTask)
    })
}

// internal/mymodule/middleware/validation.go
package middleware

func ValidateRequest(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Validation logic here
        next.ServeHTTP(w, r)
    })
}
```

### Code Standards

1. **DTO Requirements**
   - All request/response structures in `dto/` package
   - Use struct tags for validation
   - Separate files for requests and responses
   - Include OpenAPI annotations

2. **Route Organization**
   - All routes defined in `routes/` package
   - Group by authentication requirements
   - Use middleware composition
   - Document each endpoint

3. **Middleware Standards**
   - Module-specific middleware in `middleware/` package
   - Reuse shared middleware from `pkg/middleware`
   - Clear naming conventions
   - Proper error handling

4. **Service Layer**
   - Business logic in `services/` package
   - No HTTP concerns in services
   - Dependency injection
   - Testable design

### Error Handling

- Use `pkg/handlers` for consistent responses
- Implement proper error logging
- Return meaningful error messages
- Use appropriate HTTP status codes

### Testing Requirements

- Unit tests for all services
- Integration tests for routes
- DTO validation tests
- Middleware behavior tests
- Mock external dependencies

### Development Workflow

1. **Feature Development**
   ```bash
   git checkout -b feature/your-feature
   # Make changes
   go test ./...
   git commit -m "feat: add new feature"
   ```

2. **API Changes**
   ```bash
   # OpenAPI specification is automatically generated by Huma v2
   # Access the live specification at: http://localhost:8080/huma-{module}/openapi.json
   # No manual generation required
   ```

3. **Documentation**
   - Update module CLAUDE.md files
   - Keep OpenAPI specs current
   - Document configuration changes

### Best Practices

- ✅ Follow the standardized module structure (dto/, routes/, middleware/, services/)
- ✅ Use DTOs for all request/response handling
- ✅ Implement validation at the DTO level
- ✅ Keep routes clean - delegate to services
- ✅ Use shared libraries for common functionality
- ✅ Implement middleware for cross-cutting concerns
- ✅ Keep modules loosely coupled
- ✅ Document all API endpoints
- ✅ Use conventional commits
- ✅ Cache ESI responses appropriately
- ❌ Never put business logic in route handlers
- ❌ Don't mix HTTP concerns with service logic
- ❌ Never run gateway directly (use Docker)
- ❌ Don't ignore cache headers from ESI
- ❌ Avoid tight coupling between modules

## 📖 API Documentation

### API Prefix Configuration

Control the API prefix via `API_PREFIX` environment variable:

- `API_PREFIX=""` → `/auth/health`
- `API_PREFIX="/api"` → `/api/auth/health`
- `API_PREFIX="/v1"` → `/v1/auth/health`

### Unified OpenAPI 3.1.1 Specification

The API gateway now provides a **single, comprehensive OpenAPI specification** that documents all modules in one unified specification:

```bash
# Unified OpenAPI specification (replaces per-module specs):

# No API prefix (default):
# Single spec: http://localhost:3000/openapi.json

# With API prefix (/api):  
# Single spec: http://localhost:3000/api/openapi.json

# All modules documented in one place:
# - Auth Module: /auth/* endpoints
# - Dev Module: /dev/* endpoints  
# - Users Module: /users/* endpoints
# - Scheduler Module: /scheduler/* endpoints
# - SDE Module: /sde/* endpoints
# - Notifications Module: /notifications/* endpoints
```

**Modern API Features:**
- **Single OpenAPI 3.1.1 Specification**: All modules documented together
- **Unified Schema Registry**: Shared schemas across all modules
- **Environment-aware Servers**: Multiple server URLs for different environments
- **Type-Safe Operations**: Complete type safety with compile-time validation
- **Real Request/Response Bodies**: Accurate JSON schemas with proper field types
- **Postman Compatible**: Generated specs can be imported directly into Postman
- **Live Documentation**: Specification updates automatically with code changes
- **Modern API Standards**: Follows OpenAPI 3.1.1 best practices

**Important**: OpenAPI specifications are generated in real-time and automatically reflect the current `API_PREFIX` configuration.

### Available Endpoints

**Traditional Endpoints:** Available at standard module paths (`/auth`, `/dev`, `/users`, etc.)

**Huma v2 Endpoints:** Available at prefixed paths (`/huma-auth`, `/huma-dev`, `/huma-users`, etc.) with:
- Automatic OpenAPI 3.1.1 documentation
- Type-safe request/response validation
- Enhanced error handling

Access the live OpenAPI specifications for complete endpoint documentation with accurate schemas and request examples. All Huma specifications can be imported directly into Postman for testing.

## 🔧 Observability

### OpenTelemetry Integration

When `ENABLE_TELEMETRY=true`:

- Structured JSON logging
- Automatic trace correlation
- Request/response tracking
- Performance metrics
- Error tracking

### Logging Standards

Following OpenTelemetry Specification 1.47.0:
- Service-specific contexts
- Trace and span ID injection
- Structured metadata
- Configurable verbosity

## 🤝 Contributing

1. **Fork the repository**
2. **Create feature branch**
3. **Write tests**
4. **Update documentation**
5. **Submit pull request**

### Commit Convention

```
feat: add new feature
fix: resolve bug
docs: update documentation
refactor: improve code structure
test: add tests
chore: maintenance tasks
```

## 📄 License

[Your License Here]

## 🙏 Acknowledgments

- EVE Online and CCP Games for EVE SSO and ESI
- The Go community for excellent libraries
- Contributors and maintainers