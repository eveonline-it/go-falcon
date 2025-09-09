# CLAUDE.md

This file provides guidance to Go Falcon - Monolithic API Gateway when working with code in this repository.

## AI Guidance

- Ignore GEMINI.md and GEMINI-\*.md files
- To save main context space, for code searches, inspections, troubleshooting or analysis, use code-searcher subagent where appropriate - giving the subagent full context background for the task(s) you assign it.
- After receiving tool results, carefully reflect on their quality and determine optimal next steps before proceeding. Use your thinking to plan and iterate based on this new information, and then take the best next action.
- For maximum efficiency, whenever you need to perform multiple independent operations, invoke all relevant tools simultaneously rather than sequentially.
- Before you finish, please verify your solution
- Do what has been asked; nothing more, nothing less.
- NEVER create files unless they're absolutely necessary for achieving your goal.
- ALWAYS prefer editing an existing file to creating a new one.
- NEVER proactively create documentation files (\*.md) or README files. Only create documentation files if explicitly requested by the User.
- When you update or modify core context files, also update markdown documentation and memory bank
- When asked to commit changes, exclude CLAUDE.md and CLAUDE-\*.md referenced memory bank system files from any commits. Never delete these files.

## Memory Bank System

This project uses a structured memory bank system with specialized context files. Always check these files for relevant information before starting work:

### Core Context Files

- **CLAUDE-activeContext.md** - Current session state, goals, and progress
- **CLAUDE-patterns.md** - Established code patterns and conventions
- **CLAUDE-decisions.md** - Architecture decisions and rationale

**Important:** Always reference the active context file first to understand what's currently being worked on and maintain session continuity.

### Archive System

- **archive/README.md** - Index of archived historical documentation
- **archive/historical-work-2025-08.md** - Historical development work (August 2025)

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

Multi-stage Docker builds, hot reload, graceful shutdown, distributed locking, database migrations, comprehensive error handling, request tracing, health checks.

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
HOST="0.0.0.0"                 # Host interface
PORT="3000"                    # Server port
API_PREFIX=                    # API route prefix
JWT_SECRET="your-secret-key"   # JWT signing key
EVE_CLIENT_ID="your-client-id" # EVE OAuth
EVE_CLIENT_SECRET="your-secret" # EVE OAuth
ENABLE_TELEMETRY="true"        # OpenTelemetry
```

See `.env.example` for complete configuration options.

## 🎯 Core Features

### 1. Modular Architecture

Each module in `internal/` is self-contained with:

- Dedicated routes and handlers
- Service-specific business logic
- Independent database collections
- Centralized authentication and permission middleware (pkg/middleware)
- Comprehensive documentation (CLAUDE.md)

### 2. Unified OpenAPI Architecture

Unified OpenAPI 3.1.1 spec with flexible API prefix support, environment-aware configuration, and automatic documentation generation via Huma v2.

### 3. Task Scheduling System

Distributed cron-based task scheduling with system tasks, execution cancellation, and module integration. See [`internal/scheduler/CLAUDE.md`](internal/scheduler/CLAUDE.md) for complete documentation.

### 4. EVE Online SDE Management

In-memory SDE service for ultra-fast EVE game data access. See [`pkg/sde/CLAUDE.md`](pkg/sde/CLAUDE.md).

### 5. Authentication & Security

EVE SSO OAuth2, JWT tokens, dual auth support (cookies/bearer), granular permissions. See [`internal/auth/CLAUDE.md`](internal/auth/CLAUDE.md).

### 6. Dynamic Routing System (Sitemap Module)

Backend-controlled frontend routing with permission-based access and dual-structure response. See [`internal/sitemap/CLAUDE.md`](internal/sitemap/CLAUDE.md).

## 📚 Module Documentation

### Core Modules with Detailed Documentation

| Module             | Location                                                           | Description                                                                                         |
| ------------------ | ------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------- |
| **Authentication** | [`internal/auth/CLAUDE.md`](internal/auth/CLAUDE.md)               | EVE SSO integration, JWT management, user profiles                                                  |
| **Scheduler**      | [`internal/scheduler/CLAUDE.md`](internal/scheduler/CLAUDE.md)     | Task scheduling, cron jobs, distributed execution, character/corporation/alliance automated updates |
| **Users**          | [`internal/users/CLAUDE.md`](internal/users/CLAUDE.md)             | User management and profile operations                                                              |
| **Groups**         | [`internal/groups/CLAUDE.md`](internal/groups/CLAUDE.md)           | Group and role-based access control, character name resolution                                      |
| **Sitemap**        | [`internal/sitemap/CLAUDE.md`](internal/sitemap/CLAUDE.md)         | Backend-managed dynamic routing with flat routes array and hierarchical navigation tree             |
| **Character**      | [`internal/character/CLAUDE.md`](internal/character/CLAUDE.md)     | Character information, portraits, background affiliation updates                                    |
| **Corporation**    | [`internal/corporation/CLAUDE.md`](internal/corporation/CLAUDE.md) | Corporation data and member management, automated ESI updates                                       |
| **Alliance**       | [`internal/alliance/CLAUDE.md`](internal/alliance/CLAUDE.md)       | Alliance information, member corporations, relationship data                                        |
| **Structures**     | [`internal/structures/CLAUDE.md`](internal/structures/CLAUDE.md)   | NPC stations and player structures with authentication, shared Redis error tracking, location hierarchy |
| **Assets**         | [`internal/assets/CLAUDE.md`](internal/assets/CLAUDE.md)           | Character/corporation assets, valuation, tracking, container hierarchy                              |

### Shared Package Quick Reference

| Package         | Purpose                   | Key Features                                                        |
| --------------- | ------------------------- | ------------------------------------------------------------------- |
| **App**         | Application lifecycle     | Graceful shutdown, telemetry management                             |
| **Config**      | Environment configuration | Cookie duration, days support                                       |
| **Database**    | MongoDB/Redis utilities   | Connection pooling, health checks                                   |
| **EVE Gateway** | ESI client library        | Rate limiting, caching, OAuth                                       |
| **Handlers**    | HTTP response utilities   | StandardResponse, JSON utilities, health checks                     |
| **Logging**     | OpenTelemetry integration | Conditional telemetry (ENABLE_TELEMETRY), trace correlation         |
| **Middleware**  | Request processing        | Centralized authentication, permission checking, tracing middleware |
| **Module**      | Module system base        | BaseModule interface, shared dependencies                           |
| **SDE Service** | In-memory data service    | EVE static data + complete universe, O(1) lookups, thread-safe      |

_For detailed implementation, see individual `pkg/[package]/CLAUDE.md` files_

## 🚀 EVE Online Integration

Comprehensive EVE integration with automated updates:

- **Character**: 30-minute affiliation updates ([details](internal/character/CLAUDE.md))
- **Corporation**: Daily 4 AM refresh ([details](internal/corporation/CLAUDE.md))
- **Alliance**: Complete sync ([details](internal/alliance/CLAUDE.md))
- **Structures**: NPC stations and player structures with authenticated access and shared Redis error tracking ([details](internal/structures/CLAUDE.md))
- **Assets**: Character/corporation asset tracking with valuation ([details](internal/assets/CLAUDE.md))
- **Universe**: 113 regions, 1,175 constellations, 8,437 systems in memory ([details](internal/sde_admin/CLAUDE.md))

Follows [CCP ESI guidelines](https://developers.eveonline.com/docs/services/esi/best-practices/). First user gets Super Administrator group. See module docs for details.

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
make migrate-up        # Run pending migrations
make migrate-status    # Check status
make migrate-down      # Rollback last
make migrate-create name=feature  # Create new
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

### Testing Authenticated Endpoints

For testing authenticated endpoints during development:

- **Authentication Cookie**: Use the cookie stored in `./tmp/cookie.txt`
- **curl Example**: `curl -H "Cookie: $(cat ./tmp/cookie.txt)" http://localhost:3000/endpoint`
- **Postman/Bruno**: Import cookie from `./tmp/cookie.txt` file
- **Session Management**: Cookie is automatically updated during EVE SSO authentication flow

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

- `API_PREFIX=` → `/auth/health`
- `API_PREFIX="/api"` → `/api/auth/health`
- `API_PREFIX="/v1"` → `/v1/auth/health`

### Unified OpenAPI 3.1.1 Specification

**Single comprehensive OpenAPI spec** documenting all modules:

- Default: `http://localhost:3000/openapi.json`
- With prefix: `http://localhost:3000/api/openapi.json`

### Scalar API Documentation

Interactive API docs at `http://localhost:3000/docs` with dark mode, JWT auth, endpoint testing, and live OpenAPI 3.1.1 spec generation.

### Available Endpoints

All endpoints under configurable `API_PREFIX`. See `/openapi.json` for complete API specification or `/docs` for interactive documentation.

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
