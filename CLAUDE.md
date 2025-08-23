# Go Falcon - Monolithic API Gateway

A production-ready Go monolithic architecture featuring modular design, EVE Online integration, and comprehensive task scheduling capabilities.

## 🚀 Overview

Go Falcon is a monolithic API gateway built with Go that provides:

- **Type-Safe APIs**: Huma v2 framework with compile-time validation
- **Modular Architecture**: Clean separation of concerns with internal modules
- **EVE Online Integration**: Complete SSO authentication and ESI API integration
- **Task Scheduling**: Distributed task scheduling with cron support and execution cancellation
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
│   ├── falcon/            # Main API falcon
│   ├── backup/            # MongoDB/Redis backup utility
│   └── restore/           # Data restoration utility
├── internal/              # Private application modules
│   ├── auth/             # EVE SSO authentication
│   │   ├── dto/          # Input/Output structures
│   │   ├── middleware/   # Auth-specific middleware
│   │   ├── routes/       # Route definitions
│   │   ├── services/     # Business logic
│   │   ├── models/       # Database schemas
│   │   └── CLAUDE.md     # Module documentation
│   ├── scheduler/        # Task scheduling system
│   │   ├── dto/          # Input/Output structures
│   │   ├── middleware/   # Auth-specific middleware
│   │   ├── routes/       # Route definitions
│   │   ├── services/     # Business logic
│   │   ├── models/       # Database schemas
│   │   └── CLAUDE.md     # Module documentation
│   └── users/            # User management
│       ├── dto/          # Input/Output structures
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
PORT="3000"                    # Main server port

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
# SUPER_ADMIN_CHARACTER_ID="123456789" # DEPRECATED: First user is now auto-assigned to super_admin group

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
- **Cron Scheduling**: Standard cron expression support with 6-field format (including seconds)
- **Task Types**: HTTP webhooks, functions, system tasks, custom executors
- **System Tasks**: Automated background operations including character affiliation updates, corporation data updates, and alliance bulk imports
- **Distributed Locking**: Redis-based coordination preventing duplicate executions
- **Execution Cancellation**: Real-time cancellation of running task executions via context-based system
- **Execution History**: Complete audit trail with performance metrics
- **Worker Pool**: Configurable concurrent execution (default: 10 workers)
- **Module Integration**: Direct integration with character, corporation, and alliance modules for ESI-based updates

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
| **Scheduler** | [`internal/scheduler/CLAUDE.md`](internal/scheduler/CLAUDE.md) | Task scheduling, cron jobs, distributed execution, character/corporation/alliance automated updates |
| **Users** | [`internal/users/CLAUDE.md`](internal/users/CLAUDE.md) | User management and profile operations |
| **Groups** | [`internal/groups/CLAUDE.md`](internal/groups/CLAUDE.md) | Group and role-based access control, character name resolution |
| **Character** | [`internal/character/CLAUDE.md`](internal/character/CLAUDE.md) | Character information, portraits, background affiliation updates |
| **Corporation** | [`internal/corporation/CLAUDE.md`](internal/corporation/CLAUDE.md) | Corporation data and member management, automated ESI updates |
| **Alliance** | [`internal/alliance/CLAUDE.md`](internal/alliance/CLAUDE.md) | Alliance information, member corporations, relationship data |

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

### Background Data Updates

The system provides automated background updates for EVE Online data:

**Character Affiliation Updates**:
- **Schedule**: Every 30 minutes via scheduler system task
- **ESI Endpoint**: `POST /characters/affiliation/` (batch processing up to 1000 characters)
- **Processing**: Parallel workers (3 concurrent) for optimal throughput
- **Database Strategy**: Bulk updates with upsert logic for new characters
- **Performance**: ~3000 characters per minute processing capability
- **Monitoring**: Detailed debug logging showing character changes:
  ```
  🔄 Character 90000001 affiliation UPDATED: corp: 98000001→98000002, alliance: 0→99000001
  📊 Character 90000002 affiliation checked (no changes)
  ➕ Character 90000003 NOT FOUND in database, creating new record
  ```

**Corporation Data Updates**:
- **Schedule**: Daily at 4 AM via scheduler system task
- **ESI Endpoint**: `GET /corporations/{corporation_id}/` (individual corporation updates)
- **Processing**: Parallel workers (10 concurrent by default) with rate limiting
- **Database Strategy**: Upsert operations for all corporation data fields
- **Performance**: Rate limit compliant with 50ms delays between requests
- **Monitoring**: Progress logging every 100 corporations with success/failure tracking

**Alliance Bulk Import**:
- **Schedule**: Weekly on Sunday at 3 AM via scheduler system task
- **ESI Endpoints**: `GET /alliances/` + `GET /alliances/{alliance_id}/`
- **Processing**: Batch import with configurable delays and retry logic
- **Database Strategy**: Full alliance data import with relationship mapping
- **Performance**: Optimized for large-scale alliance data synchronization

**Integration Benefits**:
- **Data Freshness**: Corporation, character, and alliance data updated automatically
- **Scalability**: Handles large databases efficiently with parallel processing and batch operations
- **Reliability**: Retry logic with graceful failure handling across all update types
- **Observability**: Complete execution statistics and error reporting for all system tasks

### ESI (EVE Swagger Interface) Best Practices

The project strictly follows [CCP's ESI guidelines](https://developers.eveonline.com/docs/services/esi/best-practices/) and **MUST adhere to the official EVE Online ESI OpenAPI specification**:

#### ESI Specification Compliance
**MANDATORY**: All ESI integrations must follow the official specification at https://esi.evetech.net/meta/openapi.json
- Field names must match the specification exactly
- Data types must handle JSON unmarshaling correctly (numbers become `float64`)
- Response structures must reflect the official schema
- Endpoint paths and parameters must match the specification

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

### Super Admin System

The super admin system now uses **group-based membership** instead of user profile flags:

- **First User Auto-Assignment**: The very first user to authenticate is automatically added to the "Super Administrator" group
- **Group-Based Permissions**: Super admin status is determined by membership in the "Super Administrator" system group
- **No Entity Groups for First User**: First user only gets system group assignment, no corporation/alliance groups until entities are configured
- **No Configuration Required**: The `SUPER_ADMIN_CHARACTER_ID` environment variable is deprecated and no longer needed
- **Database-Driven**: All super admin permissions are managed through the groups module
- **Site Settings Control**: Corporation/alliance groups are only created when entities are enabled in site settings

**Migration**: Existing super admins will need to be manually added to the "Super Administrator" group via the groups API or by clearing the database to trigger first-user assignment.

## 🛠️ Development Guidelines

### Module Structure Standards

Each module in `internal/` **MUST** follow this standardized structure:

```
internal/modulename/
├── dto/                    # Data Transfer Objects
│   ├── inputs.go          # Request input DTOs with validation
│   ├── outputs.go         # Response output DTOs
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
// internal/mymodule/dto/inputs.go
package dto

type CreateTaskInput struct {
    Name        string `json:"name" minLength:"3" maxLength:"100" required:"true" description:"Task name"`
    Description string `json:"description" maxLength:"500" description:"Task description"`
    CronExpr    string `json:"cron_expression" required:"true" pattern:"^(@(hourly|daily|weekly|monthly)|\\S+\\s+\\S+\\s+\\S+\\s+\\S+\\s+\\S+)$" description:"Cron expression"`
}

// internal/mymodule/dto/outputs.go
package dto

type TaskOutput struct {
    ID          string    `json:"id" description:"Task ID"`
    Name        string    `json:"name" description:"Task name"`
    Description string    `json:"description" description:"Task description"`
    CreatedAt   time.Time `json:"created_at" description:"Creation timestamp"`
}

// internal/mymodule/routes/routes.go
package routes

import (
    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/adapters/humachi"
)

func (m *Module) RegisterUnifiedRoutes(api huma.API) {
    // Health check endpoint
    huma.Get(api, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
        return &HealthOutput{Status: "healthy"}, nil
    })
    
    // Task management endpoints with authentication
    huma.Post(api, "/tasks", func(ctx context.Context, input *CreateTaskInput) (*TaskOutput, error) {
        // Delegate to service layer
        return m.service.CreateTask(ctx, input)
    }, huma.Middlewares(m.middleware.RequireAuth))
    
    huma.Get(api, "/tasks", func(ctx context.Context, input *ListTasksInput) (*ListTasksOutput, error) {
        return m.service.ListTasks(ctx, input)
    }, huma.Middlewares(m.middleware.RequireAuth))
    
    huma.Get(api, "/tasks/{id}", func(ctx context.Context, input *GetTaskInput) (*TaskOutput, error) {
        return m.service.GetTask(ctx, input.ID)
    }, huma.Middlewares(m.middleware.RequireAuth))
}
```

### Code Standards

1. **DTO Requirements**
   - All input/output structures in `dto/` package
   - Use Huma v2 struct tags for validation and OpenAPI generation
   - Separate files for inputs and outputs
   - Include descriptions in struct tags for documentation

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

### Module Status Endpoint Standard

Every internal module **MUST** implement a standardized status endpoint that provides module health information:

#### Endpoint Definition
```
GET /{module}/status
```
- Public endpoint (no authentication required)
- Returns module name, status, and optional message
- Used for health monitoring and debugging

#### Response Structure
```go
// dto/outputs.go
type StatusOutput struct {
    Body StatusResponse `json:"body"`
}

type StatusResponse struct {
    Module  string `json:"module" description:"Module name"`
    Status  string `json:"status" enum:"healthy,unhealthy" description:"Module health status"`
    Message string `json:"message,omitempty" description:"Optional status message or error details"`
}
```

#### Implementation Example
```go
// routes/routes.go
func (r *Routes) RegisterUnifiedRoutes(api huma.API) {
    // Status endpoint (public, no auth required)
    huma.Register(api, huma.Operation{
        OperationID: "get-module-status",
        Method:      "GET",
        Path:        basePath + "/status",
        Summary:     "Get module status",
        Description: "Returns the health status of the module",
        Tags:        []string{"Module Status"},
    }, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
        status := r.service.GetStatus(ctx)
        return &dto.StatusOutput{Body: *status}, nil
    })
    
    // Other module endpoints...
}

// services/service.go
func (s *Service) GetStatus(ctx context.Context) *dto.StatusResponse {
    // Check module dependencies (database, external services, etc.)
    if err := s.checkDependencies(); err != nil {
        return &dto.StatusResponse{
            Module:  "module-name",
            Status:  "unhealthy",
            Message: fmt.Sprintf("Dependency check failed: %v", err),
        }
    }
    
    return &dto.StatusResponse{
        Module: "module-name",
        Status: "healthy",
    }
}
```

#### Status Check Requirements
- Database connectivity (if applicable)
- External service availability (ESI, Redis, etc.)
- Critical configuration presence
- Resource availability (memory, connections)

#### OpenAPI Documentation Tags
- **REQUIRED**: All status endpoints must use `Tags: []string{"Module Status"}` for consistent organization in Scalar API documentation
- This ensures all module status endpoints are grouped together in a single folder
- Do not use module-specific tags (e.g., "Auth", "Users") for status endpoints
- **AVOID**: Multiple tags for single endpoints (e.g., `["Alliances", "Import"]`) as this creates duplicate entries in different folders
- **PREFERRED**: Use single, module-specific tags (e.g., `["Alliances"]`) to keep related endpoints together

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
   # Access the unified specification at: http://localhost:3000/openapi.json
   # No manual generation required - updates in real-time
   ```

3. **Documentation**
   - Update module CLAUDE.md files
   - Keep OpenAPI specs current
   - Document configuration changes

### Best Practices

- ✅ Follow the standardized module structure (dto/, routes/, middleware/, services/)
- ✅ Use DTOs for all input/output handling with Huma v2 patterns
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
- ❌ Never run falcon directly because it's running with air in background (started manually by dev)
- ❌ Don't ignore cache headers from ESI
- ❌ Avoid tight coupling between modules

## 📖 API Documentation

### API Prefix Configuration

Control the API prefix via `API_PREFIX` environment variable:

- `API_PREFIX=""` → `/auth/health`
- `API_PREFIX="/api"` → `/api/auth/health`
- `API_PREFIX="/v1"` → `/v1/auth/health`

### Unified OpenAPI 3.1.1 Specification

The Falcon API gateway now provides a **single, comprehensive OpenAPI specification** that documents all modules in one unified specification:

```bash
# Unified OpenAPI specification (replaces per-module specs):

# No API prefix (default):
# Single spec: http://localhost:3000/openapi.json

# With API prefix (/api):  
# Single spec: http://localhost:3000/api/openapi.json

# All modules documented in one place:
# - Auth Module: /auth/* endpoints
# - Users Module: /users/* endpoints
# - Scheduler Module: /scheduler/* endpoints
```

### Scalar API Documentation

The falcon gateway includes **Scalar**, a modern, interactive API documentation interface:

```bash
# Access Scalar Documentation:
http://localhost:3000/docs
```

**Scalar Features:**
- **Modern Interface**: Beautiful, responsive documentation UI with purple theme
- **Dark Mode**: Built-in dark theme for comfortable viewing
- **Interactive Testing**: Try API endpoints directly from the documentation
- **Search**: Quick search with keyboard shortcut (Ctrl/Cmd + K)
- **Server Selection**: Switch between different API servers
- **Authentication**: Built-in support for JWT Bearer tokens
- **Code Generation**: Generate API client code in multiple languages
- **Request Examples**: View and copy request examples

**Modern API Features:**
- **Single OpenAPI 3.1.1 Specification**: All modules documented together
- **Unified Schema Registry**: Shared schemas across all modules
- **Environment-aware Servers**: Multiple server URLs for different environments
- **Type-Safe Operations**: Complete type safety with compile-time validation
- **Real Request/Response Bodies**: Accurate JSON schemas with proper field types
- **Scalar Documentation**: Interactive API documentation with try-it-out functionality
- **Postman Compatible**: Generated specs can be imported directly into Postman
- **Live Documentation**: Specification updates automatically with code changes
- **Modern API Standards**: Follows OpenAPI 3.1.1 best practices

**Important**: OpenAPI specifications are generated in real-time and automatically reflect the current `API_PREFIX` configuration.

### Available Endpoints

**Unified API Endpoints:** All modules use Huma v2 for type-safe operations:
- Auth Module: `/auth/*` endpoints with EVE SSO integration
- Users Module: `/users/*` endpoints for user management  
- Scheduler Module: `/scheduler/*` endpoints for task scheduling and system task management
- Character Module: `/character/*` endpoints for character information and name search
- Corporation Module: `/corporations/*` endpoints for corporation data
- Alliance Module: `/alliances/*` endpoints for alliance information

**Features:**
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