# CLAUDE.md

This file provides guidance to Go Falcon - Monolithic API Gateway when working with code in this repository.

## AI Guidance

* Ignore GEMINI.md and GEMINI-*.md files
* To save main context space, for code searches, inspections, troubleshooting or analysis, use code-searcher subagent where appropriate - giving the subagent full context background for the task(s) you assign it.
* After receiving tool results, carefully reflect on their quality and determine optimal next steps before proceeding. Use your thinking to plan and iterate based on this new information, and then take the best next action.
* For maximum efficiency, whenever you need to perform multiple independent operations, invoke all relevant tools simultaneously rather than sequentially.
* Before you finish, please verify your solution
* Do what has been asked; nothing more, nothing less.
* NEVER create files unless they're absolutely necessary for achieving your goal.
* ALWAYS prefer editing an existing file to creating a new one.
* NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.
* When you update or modify core context files, also update markdown documentation and memory bank
* When asked to commit changes, exclude CLAUDE.md and CLAUDE-*.md referenced memory bank system files from any commits. Never delete these files.

## Memory Bank System

This project uses a structured memory bank system with specialized context files. Always check these files for relevant information before starting work:

### Core Context Files

* **CLAUDE-activeContext.md** - Current session state, goals, and progress
* **CLAUDE-patterns.md** - Established code patterns and conventions
* **CLAUDE-decisions.md** - Architecture decisions and rationale

**Important:** Always reference the active context file first to understand what's currently being worked on and maintain session continuity.

### Archive System

* **archive/README.md** - Index of archived historical documentation
* **archive/historical-work-2025-08.md** - Historical development work (August 2025)

Historical content is archived to reduce active context usage while preserving implementation history for reference.

### Memory Bank System Backups

When asked to backup Memory Bank System files, you will copy the core context files above and @.claude settings directory to directory @/path/to/backup-directory. If files already exist in the backup directory, you will overwrite them.

## 🚀 Project Overview

Go Falcon is a production-ready monolithic API gateway built with Go that provides:

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
- [Database Management](#database-management)
- [Development Guidelines](#development-guidelines)
- [API Documentation](#api-documentation)
- [Observability](#observability)
- [Contributing](#contributing)

## 🏗️ Architecture

### Directory Structure

```
go-falcon/
├── cmd/           # Executables (falcon, backup, restore)
├── internal/      # Private modules (auth, scheduler, sitemap, users, etc.)
│   └── [module]/  # Each module: dto/, routes/, services/, models/, CLAUDE.md
├── pkg/           # Shared libraries (app, config, database, handlers, etc.)
├── docs/          # Documentation
├── builders/      # Docker configurations
└── scripts/       # Automation scripts
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
- ✅ Database migrations with rollback support
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

4. **Run database migrations**
   ```bash
   make migrate-up
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
- Centralized authentication and permission middleware (pkg/middleware)
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

### 6. Dynamic Routing System (Sitemap Module)

The sitemap module provides backend-controlled frontend routing with a dual-structure response:
- **Flat Routes Array**: Simple list of available routes for React Router configuration (no nested children)
- **Hierarchical Navigation**: Tree structure with folders for rendering vertical navigation menus
- **Permission-Based Access**: Routes filtered based on user permissions and group memberships
- **Separation of Concerns**: Routes define what's accessible, navigation defines how it's organized
- **Real-time Updates**: Route access can change without frontend deployments

## 📚 Module Documentation

### Core Modules with Detailed Documentation

| Module | Location | Description |
|--------|----------|-------------|
| **Authentication** | [`internal/auth/CLAUDE.md`](internal/auth/CLAUDE.md) | EVE SSO integration, JWT management, user profiles |
| **Scheduler** | [`internal/scheduler/CLAUDE.md`](internal/scheduler/CLAUDE.md) | Task scheduling, cron jobs, distributed execution, character/corporation/alliance automated updates |
| **Users** | [`internal/users/CLAUDE.md`](internal/users/CLAUDE.md) | User management and profile operations |
| **Groups** | [`internal/groups/CLAUDE.md`](internal/groups/CLAUDE.md) | Group and role-based access control, character name resolution |
| **Sitemap** | [`internal/sitemap/CLAUDE.md`](internal/sitemap/CLAUDE.md) | Backend-managed dynamic routing with flat routes array and hierarchical navigation tree |
| **Character** | [`internal/character/CLAUDE.md`](internal/character/CLAUDE.md) | Character information, portraits, background affiliation updates |
| **Corporation** | [`internal/corporation/CLAUDE.md`](internal/corporation/CLAUDE.md) | Corporation data and member management, automated ESI updates |
| **Alliance** | [`internal/alliance/CLAUDE.md`](internal/alliance/CLAUDE.md) | Alliance information, member corporations, relationship data |

### Shared Package Quick Reference

| Package | Purpose | Key Features |
|---------|---------|--------------|
| **App** | Application lifecycle | Graceful shutdown, telemetry management |
| **Config** | Environment configuration | Cookie duration, days support |
| **Database** | MongoDB/Redis utilities | Connection pooling, health checks |
| **EVE Gateway** | ESI client library | Rate limiting, caching, OAuth |
| **Handlers** | HTTP response utilities | StandardResponse, JSON utilities, health checks |
| **Logging** | OpenTelemetry integration | Conditional telemetry (ENABLE_TELEMETRY), trace correlation |
| **Middleware** | Request processing | Centralized authentication, permission checking, tracing middleware |
| **Module** | Module system base | BaseModule interface, shared dependencies |
| **SDE Service** | In-memory data service | EVE static data + complete universe, O(1) lookups, thread-safe |

*For detailed implementation, see individual `pkg/[package]/CLAUDE.md` files*

## 🚀 EVE Online Integration

The system provides comprehensive EVE Online integration with automated background updates and complete universe data support:

- **Character Updates**: Affiliation tracking every 30 minutes ([details](internal/character/CLAUDE.md))
- **Corporation Updates**: Daily data refresh at 4 AM ([details](internal/corporation/CLAUDE.md))  
- **Alliance Updates**: Complete alliance data synchronization ([details](internal/alliance/CLAUDE.md))
- **Scheduler Integration**: All updates managed via system tasks ([details](internal/scheduler/CLAUDE.md))
- **Universe Data**: Complete EVE universe in memory - 113 regions, 1,175 constellations, 8,437 solar systems ([details](internal/sde_admin/CLAUDE.md))

### ESI Best Practices

The project strictly follows [CCP's ESI guidelines](https://developers.eveonline.com/docs/services/esi/best-practices/) and adheres to the official [ESI OpenAPI specification](https://esi.evetech.net/meta/openapi.json).

### Authentication & Security

Complete EVE SSO integration with JWT tokens and group-based access control. See [Authentication Module](internal/auth/CLAUDE.md) for detailed implementation including authentication flows, profile management, and automatic group assignment system.

#### Authentication Flow
- `/auth/eve/register` → Full EVE scopes → **Authenticated Users** group
- `/auth/eve/login` → Basic login only → **Guest Users** group
- First user → Automatically assigned to **Super Administrator** group

**Migration**: Existing super admins need manual assignment to "Super Administrator" group via groups API.

## 🗄️ Database Management

### Migration System

Go Falcon uses a comprehensive migration system for version-controlled database schema management:

**Features:**
- Version-controlled schema changes with Git integration
- Atomic operations with MongoDB transaction support
- Rollback capability for safe deployments  
- Migration status tracking and integrity checks
- Dry-run mode for previewing changes
- Auto-generation of migration templates

**Migration Commands:**
```bash
# Run all pending migrations
make migrate-up

# Check migration status
make migrate-status  

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create name=feature_name

# Preview migrations (dry run)
make migrate-dry-run
```

**Migration Files:**
- Located in `migrations/` directory
- Naming convention: `{version}_{description}.go`
- Each migration has `up()` and `down()` functions
- All migrations tracked in `_migrations` collection

**Current Migrations:**
1. `001_create_groups_indexes` - Groups and memberships indexes
2. `002_create_scheduler_indexes` - Scheduler tasks and executions indexes  
3. `003_seed_system_groups` - System groups (super_admin, authenticated, guest)
4. `004_create_character_indexes` - Character collection indexes with text search
5. `005_create_users_indexes` - User collection indexes

**Deployment Integration:**
```yaml
# docker-compose.yml
services:
  migrate:
    command: ["/app/migrate", "-command=up"]
    depends_on: [mongodb]
    
  app:
    depends_on:
      migrate:
        condition: service_completed_successfully
```

See `migrations/README.md` for complete documentation.

## 🛠️ Development Guidelines

### Module Structure Standards

Each module in `internal/` **MUST** follow this standardized structure:

```
internal/modulename/
├── dto/           # Data Transfer Objects (inputs.go, outputs.go, validators.go)
├── routes/        # Route definitions (routes.go, health.go, api.go)  
├── services/      # Business logic (service.go, repository.go)
├── models/        # Database models (models.go)
├── module.go      # Module initialization
└── CLAUDE.md      # Module documentation

Note: Authentication and permission middleware now centralized in pkg/middleware/
```

See individual module CLAUDE.md files for detailed implementation examples and patterns.

### Code Standards

- **DTOs**: Use `dto/` package with Huma v2 struct tags and OpenAPI documentation
- **Routes**: Define in `routes/` package with centralized middleware adapters
- **Middleware**: Use centralized system from `pkg/middleware` with module-specific adapters
- **Services**: Business logic separation with testable design

### Module Status Endpoint Standard

Every module **MUST** implement `GET /{module}/status` endpoint:
- Public endpoint returning module health status
- Response: `{module: string, status: "healthy|unhealthy", message?: string}`
- Use `Tags: ["Module Status"]` for consistent OpenAPI grouping
- Check database connectivity, external services, and critical configuration

### Development Workflow

1. **Feature Development**: Branch → Changes → Tests → Commit
2. **API Changes**: Huma v2 auto-generates OpenAPI specs at `/openapi.json`
3. **Documentation**: Update module CLAUDE.md files
4. **Testing**: Unit tests (services), integration tests (routes), DTO validation
5. **Error Handling**: Use `pkg/handlers` for consistent responses

### Best Practices

- ✅ Follow the standardized module structure (dto/, routes/, services/, models/)
- ✅ Use DTOs for all input/output handling with Huma v2 patterns
- ✅ Implement validation at the DTO level
- ✅ Keep routes clean - delegate to services
- ✅ Use shared libraries for common functionality
- ✅ Use centralized middleware (pkg/middleware) for authentication and permissions
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

**Single comprehensive OpenAPI spec** documenting all modules:
- Default: `http://localhost:3000/openapi.json`
- With prefix: `http://localhost:3000/api/openapi.json`

### Scalar API Documentation

The falcon gateway includes **Scalar**, a modern, interactive API documentation interface:

```bash
# Access Scalar Documentation:
http://localhost:3000/docs
```

**Scalar Features:**
- Modern UI with dark mode, interactive testing, search (Ctrl/Cmd + K)
- JWT authentication support, code generation, server selection
- Try API endpoints directly from documentation

**API Features:**
- Single OpenAPI 3.1.1 spec with unified schema registry
- Type-safe operations with compile-time validation
- Postman compatible, live documentation updates
- Environment-aware server configuration

**Important**: OpenAPI specifications are generated in real-time and automatically reflect the current `API_PREFIX` configuration.

### Available Endpoints

**Unified API Endpoints** (Huma v2 type-safe operations):
- `/auth/*` - EVE SSO integration, `/users/*` - User management
- `/scheduler/*` - Task scheduling, `/character/*` - Character info
- `/corporations/*` - Corp data, `/alliances/*` - Alliance info

**Features:** Auto OpenAPI 3.1.1 docs, type-safe validation, enhanced error handling

## 🔧 Observability

**OpenTelemetry Integration** (`ENABLE_TELEMETRY=true`):
- Structured JSON logging with trace correlation
- Request/response tracking, performance metrics, error tracking
- Follows OpenTelemetry Specification 1.47.0

## 🤝 Contributing

1. Fork → Feature branch → Tests → Documentation → Pull request
2. **Commit Convention**: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`

## 📄 License

[Your License Here]

## 🙏 Acknowledgments

- EVE Online and CCP Games for EVE SSO and ESI
- The Go community for excellent libraries
- Contributors and maintainers