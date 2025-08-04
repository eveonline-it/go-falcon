# Go Falcon - Go Gateway Project

A production-ready Go gateway application with modular architecture featuring Chi router, background services, and comprehensive observability.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Go Gateway Application                        │
│                           :3000                                  │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐   │
│  │ Auth Module │  │ User Module │  │  Notification Module    │   │
│  │             │  │             │  │                         │   │
│  │ /api/auth/* │  │ /api/users/*│  │  /api/notifications/*   │   │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │              Background Tasks & Services                    │ │
│  │   • Auth Tasks    • User Tasks    • Notification Tasks     │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                                  │
         ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
         │     MongoDB     │    │     Redis       │    │  Docker Stack   │
         │     :27017      │    │     :6379       │    │   Containers    │
         │   (Database)    │    │   (Cache/WS)    │    │                 │
         └─────────────────┘    └─────────────────┘    └─────────────────┘
                                         │
         ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
         │     SigNoz      │    │   Chi Router    │    │  OpenTelemetry  │
         │   :3301,:4318   │    │  HTTP Gateway   │    │    Tracing      │
         │ (Observability) │    │                 │    │   & Logging     │
         └─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🚀 Features

- **🏗️ Modular Gateway Architecture**: Scalable modular design with Chi router
- **📊 OpenTelemetry Integration**: Full observability with traces, metrics, and logs
- **🗄️ Multi-Database Support**: MongoDB (primary) + Redis (cache/sessions)
- **🔄 Background Tasks**: Module-based background processing
- **🔧 Docker Compose**: Complete development environment
- **🔄 Hot Reload**: Development mode with auto-reload
- **🛡️ Production Ready**: Multi-stage Dockerfiles, graceful shutdown
- **🌐 WebSocket Support**: Real-time communications via Socket.io + Redis
- **📋 OpenAPI 3.1.1**: Full API documentation compliance
- **🌍 Internationalization**: I18N support for multi-language
- **🎯 Modular Design**: Clean separation with internal modules
- **⚡ Auto CPU Tuning**: Automatic GOMAXPROCS optimization via automaxprocs

## 🏗️ Clean Architecture

The gateway follows clean architecture principles:

- **HTTP Layer**: Chi router handles HTTP requests and routes to modules
- **Module Layer**: Each module (auth, users, notifications) has its own domain logic
- **Shared Layer**: Common utilities, database connections, and middleware
- **Background Tasks**: Each module can run independent background services

## 📁 Project Structure

```
.
├── cmd/                         # Main applications for different services
│   ├── gateway/                # Gateway application entry point
│   │   └── main.go
│   ├── backup/                 # Backup application for MongoDB and Redis
│   └── restore/                # Restore application for MongoDB and Redis
├── internal/                    # Internal packages (not for external use)
│   ├── auth/                   # Authentication module
│   ├── users/                  # User management module
│   ├── notifications/          # Notification module
│   └── telemetry/              # Internal telemetry packages
├── pkg/                         # Public packages (can be imported by other projects)
│   ├── database/               # Database connectors (MongoDB, Redis)
│   ├── logging/                # OpenTelemetry logging system
│   ├── middleware/             # Common middleware (tracing, auth)
│   └── config/                 # Configuration packages
├── docs/                       # Documentation (deployment guides, API definitions)
├── examples/                   # Example code and usage samples
├── builders/                   # Build-related files (Dockerfiles)
│   └── Dockerfile
├── scripts/                    # Setup and utility scripts
│   └── init-mongo.js           # MongoDB initialization
├── docker-compose.infra.yml    # Infrastructure services (MongoDB + Redis)
├── docker-compose.prod.yml     # Production application deployment
└── .env.example               # Environment configuration template
```

## 🛠️ Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.24.5
- go-chi/chi v5.2.2
- Git

### Development Setup

1. **Clone and setup environment**:
   ```bash
   git clone <repository>
   cd go-falcon
   cp .env.example .env
   # Edit .env with your configuration
   ```

2. **Start infrastructure services**:
   ```bash
   # Start MongoDB and Redis for development
   docker-compose -f docker-compose.infra.yml up -d
   ```

3. **Run gateway locally with hot reload**:
   ```bash
   # Development mode with hot reload (recommended)
   make dev
   # Or use the development script directly
   ./scripts/dev.sh

   # Or traditional Go run (no hot reload)
   go run ./cmd/gateway
   ```

### Quick Test
```bash
# Health check
curl http://localhost:8080/health

# Module test
curl http://localhost:8080/api/auth/status
```

## 🛠️ Utility Applications

### Backup Utility
The backup application creates backups of MongoDB and Redis data:

```bash
# Run backup utility
go run ./cmd/backup

# Or build and run
go build -o backup ./cmd/backup
./backup
```

### Restore Utility  
The restore application restores data from backup files:

```bash
# Run restore utility
go run ./cmd/restore

# Or build and run
go build -o restore ./cmd/restore
./restore
```

## 🔧 Configuration

### Environment Variables

The project follows OpenTelemetry Specification 1.47.0 for logging configuration:

#### For SigNoz Integration (recommended)
```bash
ENABLE_TELEMETRY=true
SERVICE_NAME=your-service-name
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318
LOG_LEVEL=info
ENABLE_PRETTY_LOGS=true
```

#### For Development (pretty logs only)
```bash
ENABLE_TELEMETRY=false
LOG_LEVEL=debug
ENABLE_PRETTY_LOGS=true
```

#### For Production (JSON logs)
```bash
ENABLE_TELEMETRY=true
NODE_ENV=production
ENABLE_PRETTY_LOGS=false
LOG_LEVEL=info
```

### Database Configuration
```bash
# MongoDB (single database for gateway)
MONGODB_URI=mongodb://admin:password123@localhost:27017/gateway?authSource=admin

# Redis (shared cache and session store)
REDIS_URL=redis://localhost:6379

```

## 📊 Observability

### OpenTelemetry Integration
- **Traces**: Distributed tracing across all services
- **Metrics**: Performance and business metrics
- **Logs**: Structured logging with trace correlation
- **Context Propagation**: Automatic trace context across service boundaries

### SigNoz Dashboard
Access observability dashboard at: `http://localhost:3301`

### Health Checks
The gateway exposes health endpoints:
- Application: `http://localhost:8080/health`
- Auth Module: `http://localhost:8080/api/auth/health`
- Users Module: `http://localhost:8080/api/users/health`
- Notifications Module: `http://localhost:8080/api/notifications/health`

## 🔄 Development Workflow

### Hot Reload Development

#### Option 1: Using Air (Recommended)
```bash
# Install Air (if not already installed)
go install github.com/air-verse/air@latest

# Start development server with hot reload
./scripts/dev.sh

# Or using Make
make dev

# Or run Air directly
air
```

#### Option 2: Infrastructure + CLI Development (Recommended)
```bash
# Start infrastructure services
docker-compose -f docker-compose.infra.yml up -d

# Run application with hot reload via CLI
make dev
# Application will automatically reload on code changes
```

#### Available Make Commands
```bash
make help          # Show all available commands
make dev           # Start development server with hot reload
make build         # Build the application
make clean         # Clean build artifacts
make test          # Run tests
make install-tools # Install development tools
make docker-infra  # Start infrastructure services only
make health        # Check application health
```

### Testing Modules
```bash
# Test auth module
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'

# Test users module
curl http://localhost:8080/api/users/

# Test notifications module
curl http://localhost:8080/api/notifications/
```

### Adding New Modules
1. Create module directory in `internal/`
2. Implement module interface with Routes() and background tasks
3. Register module in `main.go`
4. Add module-specific database collections if needed

## 🛡️ Production Deployment

### Docker Compose Production Setup

#### Infrastructure + Application (All-in-One)
```bash
# Deploy complete production environment
docker-compose -f docker-compose.infra.yml -f docker-compose.prod.yml up -d
```

#### Separate Infrastructure and Application
```bash
# 1. Start infrastructure first
docker-compose -f docker-compose.infra.yml up -d

# 2. Deploy application
docker-compose -f docker-compose.prod.yml up -d
```

### Multi-stage Docker Builds
The gateway uses optimized multi-stage Dockerfile:
- Build stage: Compiles Go application
- Production stage: Minimal runtime image with security hardening

### Graceful Shutdown
The gateway implements graceful shutdown handling:
- HTTP server shutdown with connection draining
- Database connection cleanup
- Background task cleanup
- OpenTelemetry data flushing

### Database Organization
The gateway uses a single MongoDB database with collections:
- `gateway` - Single database with module-specific collections
- `auth_*` - Authentication collections
- `users_*` - User management collections
- `notifications_*` - Notification collections

## 📋 API Documentation

OpenAPI 3.1.1 compliant documentation will be available at:
- Swagger UI: `http://localhost:8080/docs`
- OpenAPI Spec: `http://localhost:8080/openapi.json`

## 🌍 Internationalization

I18N support for multiple languages:
- Translation files in `shared/i18n/`
- Request header detection: `Accept-Language`
- Fallback to default language


## 🤝 Contributing

1. **Follow established patterns**: Use shared libraries for common functionality
2. **Maintain test coverage**: Write tests for new features
3. **Update documentation**: Keep OpenAPI specs current
4. **Use conventional commits**: Follow commit message standards
5. **Feature branches**: Create branches for new development

> [!NOTE]
> We would very much appreciate any contribution. If you like to provide a fix or add a feature please feel free top open a PR. Or if you have any questions please contact us on Discord.

## 📄 License

[Your License Here]

## 🆘 Troubleshooting

### Common Issues
1. **Services won't start**: Check Docker logs and environment variables
2. **Database connection failed**: Verify MongoDB/Redis containers are running
3. **OpenTelemetry not working**: Check SigNoz endpoint configuration
4. **Hot reload not working**: Ensure Air is properly installed (`make install-tools`)

### Debug Commands
```bash
# Check infrastructure logs
docker-compose logs -f mongodb
docker-compose logs -f redis

# Check production application logs
docker-compose -f docker-compose.prod.yml logs -f gateway

# Check database connectivity
docker exec -it go-falcon-mongodb mongosh

# Check Redis connectivity
docker exec -it go-falcon-redis redis-cli ping

# Check gateway container (production only)
docker exec -it go-falcon-gateway ps aux
```