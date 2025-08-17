# Deployment Guide

## Docker Deployment

### Prerequisites
- Docker
- Docker Compose

### Production Deployment

#### Option 1: All-in-One Deployment (Recommended)
```bash
# Deploy infrastructure and application together
docker-compose -f docker-compose.infra.yml -f docker-compose.prod.yml up -d
```

#### Option 2: Separate Infrastructure and Application
```bash
# 1. Start infrastructure services first
docker-compose -f docker-compose.infra.yml up -d

# 2. Deploy application
docker-compose -f docker-compose.prod.yml up -d
```


### Post-Deployment Setup

#### Granular Permission System Setup

After deploying the application, you need to set up the granular permission system:

1. **Set Super Admin Character ID**:
   ```bash
   # Add to your .env file or environment variables
   SUPER_ADMIN_CHARACTER_ID=123456789  # Your EVE character ID
   ```

2. **Obtain Super Admin JWT Token**:
   ```bash
   # Log in via EVE SSO to get a JWT token
   # Use the /auth/eve/login endpoint or frontend interface
   # The JWT token will be needed for admin operations
   ```

3. **Run Permission Setup Script**:
   ```bash
   # Set your JWT token
   export SUPER_ADMIN_JWT="your_jwt_token_here"
   
   # Run the setup script to create service definitions
   ./scripts/setup-granular-permissions.sh
   ```

4. **Verify Service Creation**:
   ```bash
   # List all services
   curl -H "Authorization: Bearer $SUPER_ADMIN_JWT" \
        http://localhost:8080/admin/permissions/services
   ```

5. **Grant Initial Permissions** (example):
   ```bash
   # Allow general users to read SDE data
   curl -X POST "http://localhost:8080/admin/permissions/assignments" \
     -H "Authorization: Bearer $SUPER_ADMIN_JWT" \
     -H "Content-Type: application/json" \
     -d '{
       "service": "sde",
       "resource": "entities", 
       "action": "read",
       "subject_type": "group",
       "subject_id": "full_group_object_id",
       "reason": "Allow authenticated users to access SDE data"
     }'
   ```

#### Permission Testing

Test the permission system setup:

```bash
# Set environment variable for testing
export API_BASE_URL="http://localhost:8080"

# Run permission tests
./scripts/test-permissions.sh
```

The test script will verify:
- ✅ Public endpoints are accessible without authentication
- ✅ Protected endpoints require authentication (return 401)
- ✅ Admin endpoints require super admin access (return 401 for regular users)

### Production Management

1. **Check service status**:
   ```bash
   # Check infrastructure
   docker-compose -f docker-compose.infra.yml ps
   
   # Check application
   docker-compose -f docker-compose.prod.yml ps
   
   # Check all (if using all-in-one)
   docker-compose -f docker-compose.infra.yml -f docker-compose.prod.yml ps
   ```

2. **View logs**:
   ```bash
   # Infrastructure logs
   docker-compose -f docker-compose.infra.yml logs -f
   
   # Application logs
   docker-compose -f docker-compose.prod.yml logs -f gateway
   
   # Specific service logs
   docker-compose -f docker-compose.infra.yml logs -f mongodb
   docker-compose -f docker-compose.infra.yml logs -f redis
   ```

3. **Stop services**:
   ```bash
   # Stop all (if using all-in-one)
   docker-compose -f docker-compose.infra.yml -f docker-compose.prod.yml down
   
   # Stop application only
   docker-compose -f docker-compose.prod.yml down
   
   # Stop infrastructure only
   docker-compose -f docker-compose.infra.yml down
   ```

### Development Deployment

**New Development Workflow**: Infrastructure in Docker + Application via CLI

1. **Start infrastructure services**:
   ```bash
   # Start MongoDB and Redis
   docker-compose -f docker-compose.infra.yml up -d
   # Or explicitly
   docker-compose -f docker-compose.infra.yml up -d
   ```

2. **Run application locally with hot reload**:
   ```bash
   # Start development server with hot reload
   make dev
   # Or
   ./scripts/dev.sh
   ```

**Note**: `docker-compose.dev.yml` has been removed. Development now uses CLI for better hot reload and debugging experience.

## Local Development

### Prerequisites
- Go 1.24.5
- Docker & Docker Compose (for infrastructure)

### Setup

1. **Install dependencies and tools**:
   ```bash
   go mod download
   make install-tools
   ```

2. **Set environment variables** (create `.env` file):
   ```bash
   # Copy example environment file
   cp .env.example .env
   
   # Edit .env with your settings (defaults work with Docker infrastructure)
   MONGODB_URI="mongodb://admin:password123@localhost:27017/gateway?authSource=admin"
   REDIS_URL="redis://localhost:6379"
   ENABLE_TELEMETRY=false
   ENABLE_PRETTY_LOGS=true
   LOG_LEVEL=debug
   ```

3. **Start infrastructure services**:
   ```bash
   # Start MongoDB and Redis in Docker
   docker-compose -f docker-compose.infra.yml up -d
   ```

4. **Start development server with hot reload**:
   ```bash
   # Using the development script (recommended)
   ./scripts/dev.sh
   
   # Or using Make
   make dev
   
   # Or manually with Air
   air
   
   # Or traditional Go run (no hot reload)
   go run ./cmd/gateway
   ```

### Hot Reload Features

With Air, the application will automatically:
- **Rebuild** when you change `.go` files
- **Restart** the server
- **Display** build errors in the terminal
- **Watch** multiple directories (`cmd/`, `internal/`, `pkg/`)
- **Exclude** test files and temporary directories

### Development Architecture

The new development setup provides:
- **Infrastructure in Docker**: MongoDB and Redis run in containers
- **Application via CLI**: Go application runs locally with hot reload
- **Better Debugging**: Direct access to application process and logs
- **Faster Rebuilds**: No container rebuilds, just Go compilation
- **Production Parity**: Same infrastructure containers as production

## Environment Variables

### Core Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `MONGODB_URI` | MongoDB connection string | - | Yes |
| `REDIS_URL` | Redis connection string | `redis://localhost:6379` | Yes |
| `ENABLE_TELEMETRY` | Enable OpenTelemetry | `true` | No |
| `SERVICE_NAME` | Service name for telemetry | `gateway` | No |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP endpoint | `localhost:4318` | No |
| `LOG_LEVEL` | Logging level | `info` | No |
| `ENABLE_PRETTY_LOGS` | Pretty console logs | `false` | No |

### Permission System

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SUPER_ADMIN_CHARACTER_ID` | EVE character ID for super admin | - | Yes |

### EVE Online Integration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `EVE_CLIENT_ID` | EVE SSO application client ID | - | Yes |
| `EVE_CLIENT_SECRET` | EVE SSO application client secret | - | Yes |
| `JWT_SECRET` | Secret key for JWT token signing | - | Yes |
| `EVE_REDIRECT_URI` | OAuth2 redirect URI | - | No |
| `EVE_SCOPES` | Required EVE Online scopes | - | No |
| `ESI_USER_AGENT` | User agent for ESI requests | - | No |

## Health Checks

### Application Health Endpoints
- Application: `http://localhost:8080/health`
- Auth Module: `http://localhost:8080/auth/health`
- Users Module: `http://localhost:8080/users/health`
- Notifications Module: `http://localhost:8080/notifications/health`

### Infrastructure Health Checks
```bash
# Check MongoDB
docker exec -it go-falcon-mongodb mongosh --eval "db.adminCommand('ping')"

# Check Redis
docker exec -it go-falcon-redis redis-cli ping

# Check all container status
docker-compose ps
```

### Production Health Monitoring
```bash
# Application health in production
curl -f http://localhost:8080/health

# Container health status
docker-compose -f docker-compose.prod.yml ps

# Application logs
docker-compose -f docker-compose.prod.yml logs -f gateway
```

## Troubleshooting

### Common Issues

1. **Port already in use**:
   ```bash
   # Check what's using port 8080
   lsof -i :8080
   
   # Kill the process
   kill -9 <PID>
   ```

2. **Database connection failed**:
   ```bash
   # Check if infrastructure is running
   docker-compose ps
   
   # Check MongoDB connectivity
   docker exec -it go-falcon-mongodb mongosh
   
   # Check Redis connectivity
   docker exec -it go-falcon-redis redis-cli ping
   
   # Verify connection strings in .env
   cat .env | grep -E "MONGODB_URI|REDIS_URL"
   ```

3. **Module not responding**:
   - Check application logs (in terminal running `make dev`)
   - Verify module is properly initialized
   - Check for background task issues
   - Ensure infrastructure services are running

4. **Docker Compose issues**:
   ```bash
   # Rebuild containers if needed
   docker-compose -f docker-compose.infra.yml down
   docker-compose -f docker-compose.infra.yml up -d
   
   # Check Docker network
   docker network ls | grep gateway
   
   # Reset everything
   docker-compose -f docker-compose.infra.yml -f docker-compose.prod.yml down -v
   docker-compose -f docker-compose.infra.yml -f docker-compose.prod.yml up -d
   ```