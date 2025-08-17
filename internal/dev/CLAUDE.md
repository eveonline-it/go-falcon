# Development Module (internal/dev)

## Overview

The development module provides comprehensive testing and debugging utilities for EVE Online ESI integration and Static Data Export (SDE) functionality. It serves as a development playground and API testing interface for the go-falcon monolith.

## Architecture

### Core Components

- **ESI Client Integration**: Full EVE Online ESI API access with caching
- **SDE Service Integration**: Static Data Export testing and validation
- **Cache Management**: Intelligent caching with expiration tracking
- **Telemetry Integration**: OpenTelemetry tracing and structured logging (if ENABLE_TELEMETRY=true)
- **Development Utilities**: Service discovery and status monitoring

### Files Structure

```
internal/dev/
├── dev.go                    # Main module with route registration and initialization
├── handlers_esi-alliance.go  # EVE Online ESI API alliance endpoint handlers
├── handlers_esi-assets.go    # EVE Online ESI API assets endpoint handlers
├── handlers_esi-calendar.go  # EVE Online ESI API calendar endpoint handlers
├── handlers_esi-?.go         # EVE Online ESI API other handlers (take the list from openapi.json)
├── handlers_sde.go           # Static Data Export endpoint handlers
└── handlers_utils.go         # Utility handlers and helper functions
```

## ESI Integration Features

### Client Architecture
- **Shared Cache Manager**: Consistent caching across all ESI calls
- **Error Limits Tracking**: Monitors ESI error limits and rates
- **Retry Logic**: Intelligent retry with exponential backoff
- **User Agent Compliance**: Proper CCP-compliant User-Agent headers
- **HTTP Client Pooling**: Efficient connection management

### ESI Clients Available
- **Status Client**: Server status and player counts  
- **Character Client**: Character information and portraits with full caching support
- **Universe Client**: Solar systems, stations, and universe data
- **Alliance Client**: Alliance data, contacts, corporations, and icons
- **Corporation Client**: Complete corporation management with authentication support

### Cache Strategy
- **Cache Keys**: URL-based cache keys for consistency
- **Expiration Tracking**: Respects ESI expires headers
- **Cache Miss Fallback**: Automatic API calls when cache expires
- **JSON Marshaling**: Efficient data serialization/deserialization

## API Endpoints

### ESI Testing Endpoints

#### Status Endpoints
TODO

### SDE Endpoints

#### Memory-Based SDE (pkg/sde service)
| Endpoint | Method | Description | Permission Required |
|----------|--------|-------------|-------------------|
| `/sde/status` | GET | SDE service status and statistics | `dev.tools.read` |
| `/sde/agent/{agentID}` | GET | Get specific agent data | `dev.tools.read` |
| `/sde/category/{categoryID}` | GET | Get category information | `dev.tools.read` |
| `/sde/blueprint/{blueprintID}` | GET | Get blueprint data | `dev.tools.read` |
| `/sde/types` | GET | Get all types | `dev.tools.read` |
| `/sde/types/published` | GET | Get published types only | `dev.tools.read` |

#### Redis-Based SDE (Direct Redis access)
| Endpoint | Method | Description | Permission Required |
|----------|--------|-------------|-------------------|
| `/sde/redis/{type}/{id}` | GET | Get specific SDE entity from Redis | `dev.tools.read` |
| `/sde/redis/{type}` | GET | Get all entities of type from Redis | `dev.tools.read` |

#### Universe SDE Data
| Endpoint | Method | Description | Permission Required |
|----------|--------|-------------|-------------------|
| `/sde/universe/{type}/{region}/systems` | GET | Get all solar systems in region | `dev.tools.read` |
| `/sde/universe/{type}/{region}/{constellation}/systems` | GET | Get all solar systems in constellation | `dev.tools.read` |
| `/sde/universe/{type}/{region}` | GET | Get region data | `dev.tools.read` |
| `/sde/universe/{type}/{region}/{constellation}` | GET | Get constellation data | `dev.tools.read` |
| `/sde/universe/{type}/{region}/{constellation}/{system}` | GET | Get system data | `dev.tools.read` |

### Utility Endpoints

| Endpoint | Method | Description | Permission Required |
|----------|--------|-------------|-------------------|
| `/services` | GET | List all available endpoints | `dev.tools.read` |
| `/status` | GET | Module status information | None (public) |
| `/health` | GET | Health check endpoint | None (public) |

## Response Format

All endpoints return a consistent JSON structure:

```json
{
  "source": "EVE Online ESI" | "Static Data Export",
  "endpoint": "/original/esi/endpoint",
  "reponse_time_ms": 33, 
  "status": "success" | "error",
  "data": { /* actual response data */ },
  "module": "dev",
  "timestamp": "2023-01-01T00:00:00Z",
  "cache": {
    "cached": true,
    "expires_at": "2023-01-01T01:00:00Z",
    "expires_in": 3600
  }
}
```

### Enhanced Cache Support

All ESI endpoints now support enhanced cache metadata:

```json
{
  "cache": {
    "cached": true,
    "expires_at": "2023-01-01T01:00:00Z",
    "cache_hit": true,
    "cache_key": "https://esi.evetech.net/v1/characters/123456789/"
  }
}
```
```

## Cache Information

### Cache Headers
- **cached**: Boolean indicating if data came from cache
- **expires_at**: ISO timestamp of cache expiration
- **expires_in**: Seconds until cache expiration

### Cache Keys
- ESI endpoints use full URL as cache key
- SDE data is always from memory (no external cache)
- Cache manager shared across all ESI clients

## OpenTelemetry Integration

### Tracing
- **Span Creation**: Each handler creates detailed spans
- **Attributes**: Rich metadata including operation type, IDs, cache status
- **Error Recording**: Automatic error capture and recording
- **Performance Metrics**: Response times and cache hit ratios

### Logging
- **Structured Logging**: JSON-formatted logs with context
- **Request Tracking**: Remote address and operation logging
- **Cache Logging**: Cache hit/miss information
- **Error Logging**: Detailed error information with context

### Telemetry Attributes
```go
attribute.String("dev.operation", "character_info")
attribute.String("dev.service", "evegate")
attribute.Int("character.id", characterID)
attribute.Bool("cache.hit", cached)
attribute.Bool("dev.success", true)
```

## Error Handling

### ESI Errors
- **Invalid IDs**: Parameter validation with clear error messages
- **API Failures**: Proper HTTP status codes and error details
- **Cache Failures**: Automatic fallback to API calls
- **Rate Limiting**: Respect for ESI error limits

### SDE Errors
- **Missing Data**: Clear error messages for missing SDE entries
- **Service Not Loaded**: Status information when SDE not initialized
- **Type Conversions**: Safe handling of ID conversions

### Response Examples

**Success Response:**
```json
{
  "source": "EVE Online ESI",
  "endpoint": "/characters/123456789/",
  "status": "success",
  "reponse_time_ms": 33, 
  "data": {
    "name": "Character Name",
    "corporation_id": 1000001
  },
  "cache": {
    "cached": true,
    "expires_at": "2023-01-01T01:00:00Z",
    "expires_in": 3600
  }
}
```

**Error Response:**
```json
{
  "error": "Invalid character ID",
  "details": "Character ID must be a valid integer"
}
```

## Development Usage

### Testing ESI Integration

**⚠️ Authentication Required**: All ESI testing endpoints now require JWT authentication.

```bash
# First, authenticate via EVE Online SSO
curl http://localhost:8080/auth/eve/login  # Follow the OAuth flow

# Or use Bearer token authentication
export JWT_TOKEN="your_jwt_token_here"

# Test EVE Online server status
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/esi-status

# Test character information
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/character/123456789

# Test universe data
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/universe/system/30000142

# Test alliance data
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/alliances
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/alliance/434243723
```

### Testing SDE Integration

**⚠️ Authentication Required**: All SDE testing endpoints now require JWT authentication.

#### Memory-Based SDE (pkg/sde service)
All existing SDE endpoints continue to work with the in-memory SDE service:

```bash
# Set your JWT token
export JWT_TOKEN="your_jwt_token_here"

# Get SDE status and statistics
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/status

# Get specific entities
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/agent/3008416
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/type/587
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/blueprint/1000001

# Get collections
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/types/published
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/marketgroups
```

#### Redis-Based SDE (Direct Redis access)
New endpoints for direct Redis SDE data access:

```bash
# Get specific SDE entity from Redis
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/redis/agents/3008416
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/redis/types/587
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/redis/flags/0

# Get all entities of a type from Redis
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/redis/agents
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/redis/categories
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/redis/types
```

#### Universe SDE Data
Access EVE Online universe data with hierarchical structure:

```bash
# Get all solar systems in a region
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/universe/eve/Derelik/systems

# Get all solar systems in a constellation
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/universe/eve/Derelik/Kador/systems

# Get specific universe data
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/universe/eve/Derelik                    # Region data
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/universe/eve/Derelik/Kador             # Constellation data
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/universe/eve/Derelik/Kador/Amarr       # System data

# Other universe types
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/universe/abyssal/RegionName/systems
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/dev/sde/universe/wormhole/RegionName/ConstellationName/systems
```

## Performance Characteristics

### ESI Endpoints
- **First Call**: Network latency + ESI response time
- **Cached Calls**: <3ms response time
- **Cache Duration**: Based on ESI expires header
- **Error Handling**: <10ms for validation errors

### SDE Endpoints
- **Memory Access**: Nanosecond to microsecond response times (pkg/sde service)
- **Redis Access**: Sub-millisecond response times for individual entities
- **Universe Queries**: Millisecond response for regional system collections
- **No Network Calls**: All data served from memory or local Redis
- **Consistent Performance**: Not affected by network conditions
- **Large Datasets**: Efficient retrieval of complete type collections

## Configuration

### Required Dependencies
- **EVE Gateway Package**: `go-falcon/pkg/evegateway`
- **SDE Service**: `go-falcon/pkg/sde`
- **Database Connections**: MongoDB and Redis (via base module)

### Environment Variables
Uses the same ESI configuration as other modules:
```bash
ESI_USER_AGENT=go-falcon/1.0.0 contact@example.com
```

### Client Configuration
- **HTTP Timeout**: 30 seconds
- **Base URL**: https://esi.evetech.net
- **Cache Manager**: Default implementation with memory storage
- **Retry Client**: Exponential backoff with error limit tracking

## Authentication Support

### Authenticated Endpoints
Some ESI endpoints require valid EVE Online access tokens:

check openapi.json

### Token Format
Provide access tokens via Authorization header:
```bash
curl -H "Authorization: Bearer <access_token>" <endpoint>
```

### Security Considerations
- Tokens are never logged or cached
- Invalid tokens result in 401/403 responses
- Role validation performed by ESI, not locally

## Background Tasks

### Module Background Processing
- **Health Monitoring**: Continuous service health checks
- **Cache Maintenance**: Automatic cleanup of expired cache entries
- **Error Limit Monitoring**: Tracks ESI error rates and prevents blocking
- **Token Validation**: Periodic validation of authentication tokens
- **Graceful Shutdown**: Proper cleanup on application termination

### Task Lifecycle
- **Startup**: Initialize ESI clients and cache managers
- **Runtime**: Process requests, maintain caches, and validate tokens
- **Shutdown**: Clean up resources and connections

## Monitoring and Debugging

### Health Checks
- Module health endpoint at `/dev/health`
- Service status endpoint at `/dev/status`
- SDE service status at `/dev/sde/status`

### Telemetry Data
- **Request Volume**: Track usage of different endpoints
- **Cache Efficiency**: Monitor cache hit ratios
- **Error Rates**: Track ESI and SDE error patterns
- **Performance Metrics**: Response time distributions

### Log Analysis
- **Request Patterns**: Understanding API usage
- **Cache Performance**: Optimizing cache strategies
- **Error Investigation**: Debugging integration issues

## Best Practices

### Development Testing
- Use dev module endpoints to validate ESI integration
- Test cache behavior with repeated requests
- Verify SDE data accuracy with known values
- Monitor telemetry data for performance insights

### Performance Optimization
- Cache frequently accessed data
- Monitor ESI error limits
- Use SDE data for static information
- Implement proper error handling

### Security Considerations
- **Authentication Required**: All endpoints require valid JWT tokens and granular permissions
- **Development Access Control**: Restricted to authorized developers and administrators
- **Rate limiting through ESI error limits
- Secure handling of character IDs
- Proper input validation

## Security and Permissions

### Granular Permission System

The development module implements comprehensive permission control to secure all development and testing functionality:

#### Service: `dev`

##### Resource: `tools`
- **read**: Access to development tools, ESI testing endpoints, SDE validation, and debugging utilities

### Required Group Configuration

To use the development module, configure the following groups:

#### Administrators Group
```json
{
  "name": "administrators",
  "permissions": {
    "dev": {
      "tools": ["read"]
    }
  }
}
```

#### Developers Group
```json
{
  "name": "developers",
  "permissions": {
    "dev": {
      "tools": ["read"]
    }
  }
}
```

### Permission Requirements by Endpoint

| Endpoint Category | Method | Authentication Required | Permission Required | Description |
|------------------|--------|----------------------|-------------------|-------------|
| `/dev/status` | GET | No | None (public) | Module status information |
| `/dev/health` | GET | No | None (public) | Health check endpoint |
| `/dev/services` | GET | No | None (public) | Service discovery endpoint |
| All ESI Endpoints | GET | **Yes (JWT)** | `dev.tools.read` | EVE Online ESI testing endpoints |
| All SDE Endpoints | GET | **Yes (JWT)** | `dev.tools.read` | Static Data Export testing endpoints |

### Authentication Methods

All protected endpoints support two authentication methods:

1. **Cookie-based Authentication** (Web applications):
   ```bash
   # After logging in via /auth/eve/login, cookies are automatically included
   curl -b cookies.txt http://localhost:8080/dev/esi-status
   ```

2. **Bearer Token Authentication** (API clients, mobile apps):
   ```bash
   # Using JWT token in Authorization header
   curl -H "Authorization: Bearer YOUR_JWT_TOKEN" http://localhost:8080/dev/esi-status
   ```

### Security Features

- **JWT Authentication Required**: All protected endpoints require valid JWT tokens (cookies or Bearer header)
- **Granular Permission System**: Fine-grained access control with `dev.tools.read` permission
- **Automatic Admin Access**: Users in `super_admin` or `administrators` groups get automatic access
- **Public Endpoints**: Only `/dev/status`, `/dev/health`, and `/dev/services` are publicly accessible
- **Secure ESI Testing**: Protected environment for EVE Online API testing
- **Token Validation**: Comprehensive JWT token validation with proper error handling
- **Cross-Domain Support**: CORS-enabled for cross-subdomain authentication

### Automatic Access for Administrators

The dev module implements automatic access for administrative users:

- **Super Admins**: Users in the `super_admin` group automatically bypass all permission checks
- **Administrators**: Users in the `administrators` group also get full access to all dev tools
- **No Manual Setup**: Admin access works immediately without requiring explicit permission assignments

### Authentication Flow

1. **User Authentication**: 
   - Web: Login via `/auth/eve/login` (EVE Online SSO)
   - API: Exchange EVE tokens for JWT via `/auth/eve/token`

2. **Request Authentication**:
   - JWT middleware extracts and validates token from cookies or Authorization header
   - User information is added to request context

3. **Permission Check**:
   - Granular permission middleware checks if user has `dev.tools.read` permission
   - Super admins and administrators automatically bypass permission checks
   - Regular users require explicit permission assignment

4. **Access Granted**: User can access development tools and ESI testing endpoints

## Troubleshooting

### Common Issues
- **Authentication Required**: All protected endpoints now require JWT authentication
- **Permission Denied**: User needs `dev.tools.read` permission or admin group membership
- **Invalid JWT Token**: Token may be expired, malformed, or missing
- **ESI Rate Limits**: Monitor error limit headers
- **Cache Misses**: Check cache expiration times
- **SDE Not Loaded**: Verify SDE service initialization
- **Invalid Parameters**: Validate ID formats

### Debug Steps
1. **Authentication Issues**:
   ```bash
   # Check if you're authenticated
   curl -v http://localhost:8080/auth/profile
   
   # Login via EVE SSO (web)
   curl http://localhost:8080/auth/eve/login
   
   # Test with Bearer token
   curl -H "Authorization: Bearer YOUR_JWT_TOKEN" http://localhost:8080/dev/esi-status
   ```

2. **Permission Issues**:
   ```bash
   # Check your permissions
   curl -H "Authorization: Bearer YOUR_JWT_TOKEN" http://localhost:8080/permissions/user
   
   # Verify admin status (requires super admin JWT)
   curl -H "Authorization: Bearer SUPER_ADMIN_JWT" http://localhost:8080/admin/permissions/check \
     -d '{"service": "dev", "resource": "tools", "action": "read", "character_id": YOUR_CHARACTER_ID}'
   ```

3. **Module Health**:
   - Check `/dev/status` for module health
   - Verify ESI connectivity with `/dev/esi-status` (requires auth)
   - Test SDE functionality with `/dev/sde/status` (requires auth)
   - Monitor logs for error patterns
   - Check telemetry data for performance issues

## Future Enhancements

