# Development Guide

## Quick Start for Hot Reload Development

### 1. Setup
```bash
# Clone and setup
git clone <your-repo>
cd go-falcon

# Install dependencies and tools
go mod download
make install-tools

# Copy environment configuration
cp .env.example .env
# Edit .env with your local settings if needed
```

### 2. Start Infrastructure Services

```bash
# Start MongoDB and Redis (required for development)
docker-compose -f docker-compose.infra.yml up -d

# Or explicitly use the infrastructure file
docker-compose -f docker-compose.infra.yml up -d

# Check services are running
docker-compose ps
```

### 3. Start Development Server

#### Option A: Using the Development Script (Recommended)
```bash
./scripts/dev.sh
```

#### Option B: Using Make
```bash
make dev
```

#### Option C: Using Air Directly
```bash
air
```

### 4. What Happens with Hot Reload

When you save any `.go` file, Air will:
1. **Detect** the file change
2. **Stop** the running application gracefully
3. **Rebuild** the application
4. **Show** any build errors in the terminal
5. **Restart** the application if build succeeds
6. **Clear** the terminal and show the new startup logs

### 5. Watching These Directories
- `cmd/` - Main application entry points
- `internal/` - Internal packages
- `pkg/` - Public packages
- Root `.go` files

### 6. Excluded from Watching
- `*_test.go` - Test files
- `tmp/` - Temporary build files
- `docs/` - Documentation
- `examples/` - Example code
- `assets/` - Static assets

## Development Workflow

### Making Changes

1. **Edit Code**: Make changes to any `.go` file
2. **Save File**: Air automatically detects and rebuilds
3. **Check Terminal**: Watch for build errors or success
4. **Test API**: Your changes are immediately available

### Example Development Session

```bash
# Terminal 1: Start development server
./scripts/dev.sh

# Terminal 2: Test your changes
curl http://localhost:8080/health
curl http://localhost:8080/api/auth/status

# Note: Infrastructure services run in Docker,
# but the application runs locally with hot reload

# Edit code in your IDE, save file
# Watch Terminal 1 for automatic rebuild and restart
```

### Debugging

#### Common Issues

1. **Build Errors**: Air shows build errors in the terminal
2. **Port in Use**: Kill existing processes or change port
3. **Dependencies Missing**: Run `go mod download`

#### Air Configuration

Air is configured via `.air.toml`:
- **Build Command**: `go build -o ./tmp/gateway ./cmd/gateway`
- **Binary Path**: `./tmp/gateway`
- **Watch Extensions**: `.go`, `.tpl`, `.tmpl`, `.html`
- **Exclude Directories**: `tmp`, `vendor`, `testdata`, `docs`, `examples`

### Database Development

#### Infrastructure Services (Recommended)
The new Docker Compose setup separates infrastructure from application:

```bash
# Start infrastructure services (MongoDB + Redis)
docker-compose -f docker-compose.infra.yml up -d
# Or
docker-compose -f docker-compose.infra.yml up -d

# Stop infrastructure services
docker-compose down
# Or
docker-compose -f docker-compose.infra.yml down

# View database logs
docker-compose logs mongodb
docker-compose logs redis

# Check database status
docker-compose ps
```

#### Using Local Databases (Alternative)
If you have MongoDB and Redis installed locally, update your `.env`:
```bash
MONGODB_URI=mongodb://localhost:27017/gateway
REDIS_URL=redis://localhost:6379
```

#### Database Management
```bash
# Connect to MongoDB
docker exec -it go-falcon-mongodb mongosh

# Connect to Redis
docker exec -it go-falcon-redis redis-cli

# View MongoDB logs
docker-compose logs -f mongodb

# View Redis logs
docker-compose logs -f redis
```

### Environment Variables

Key variables for development:
```bash
# Disable telemetry for faster startup
ENABLE_TELEMETRY=false

# Enable pretty console logs
ENABLE_PRETTY_LOGS=true

# Set debug level for verbose logging
LOG_LEVEL=debug

# Development service name
SERVICE_NAME=gateway-dev
```

### Makefile Commands

```bash
make help          # Show all commands
make dev           # Start with hot reload
make build         # Build production binary
make clean         # Clean artifacts
make test          # Run tests
make docker-infra  # Start infrastructure services
make health        # Check app health
make db-up         # Start databases only
make db-down       # Stop databases
```

### Performance Tips

1. **Exclude Large Directories**: Air is configured to ignore `vendor/`, `node_modules/`, etc.
2. **Use SSD**: Faster file watching and rebuilding
3. **Limit File Watchers**: Close unnecessary IDE file watchers
4. **Use `.air.toml`**: Customize Air configuration for your needs

### IDE Integration

#### VS Code
- Install Go extension
- Air will work automatically with VS Code's file watching
- Use integrated terminal to run `make dev`

#### GoLand/IntelliJ
- Air works with GoLand's external changes detection
- Consider disabling GoLand's auto-save for better control

### Development Workflow Summary

1. **Start Infrastructure**: `docker-compose -f docker-compose.infra.yml up -d` (MongoDB + Redis)
2. **Start Application**: `make dev` (CLI with hot reload)
3. **Develop**: Edit code, auto-reload on save
4. **Test**: API endpoints available at `http://localhost:8080`
5. **Debug**: Check logs in terminal running `make dev`

### Testing During Development

```bash
# Run specific module tests
go test ./internal/auth

# Run all tests
make test

# Test API endpoints while developing
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'

# Test health endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/auth/health
```

### Why This Setup?

- **Separation of Concerns**: Infrastructure (databases) vs application
- **Faster Development**: No container rebuilds, true hot reload
- **Better Debugging**: Direct access to application logs and debugger
- **Resource Efficient**: Only databases in containers
- **Production Parity**: Same infrastructure, different app deployment