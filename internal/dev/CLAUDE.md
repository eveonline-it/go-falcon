# Development Module (internal/dev)

## Overview

The development module provides comprehensive testing and debugging utilities for EVE Online ESI integration and Static Data Export (SDE) functionality. It serves as a development playground and API testing interface for the go-falcon monolith.

## Architecture

### Core Components

- **ESI Client Integration**: Full EVE Online ESI API access with caching
- **SDE Service Integration**: Static Data Export testing and validation
- **Cache Management**: Intelligent caching with expiration tracking
- **Telemetry Integration**: OpenTelemetry tracing and structured logging
- **Development Utilities**: Service discovery and status monitoring

### Files Structure

```
internal/dev/
├── dev.go              # Main module with route registration and initialization
├── handlers_esi.go     # EVE Online ESI API endpoint handlers
├── handlers_sde.go     # Static Data Export endpoint handlers
└── handlers_utils.go   # Utility handlers and helper functions
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
- **Character Client**: Character information and portraits
- **Universe Client**: Solar systems, stations, and universe data
- **Alliance Client**: Alliance data, members, and icons

### Cache Strategy
- **Cache Keys**: URL-based cache keys for consistency
- **Expiration Tracking**: Respects ESI expires headers
- **Cache Miss Fallback**: Automatic API calls when cache expires
- **JSON Marshaling**: Efficient data serialization/deserialization

## API Endpoints

### ESI Testing Endpoints

| Endpoint | Method | Description | Cache | Telemetry |
|----------|--------|-------------|-------|-----------|
| `/esi-status` | GET | EVE Online server status | ✅ | ✅ |
| `/character/{characterID}` | GET | Character information | ✅ | ✅ |
| `/character/{characterID}/portrait` | GET | Character portrait URLs | ✅ | ✅ |
| `/universe/system/{systemID}` | GET | Solar system information | ✅ | ✅ |
| `/universe/station/{stationID}` | GET | Station information | ✅ | ✅ |
| `/alliances` | GET | All active alliances | ✅ | ✅ |
| `/alliance/{allianceID}` | GET | Alliance information | ✅ | ✅ |
| `/alliance/{allianceID}/corporations` | GET | Alliance member corporations | ✅ | ✅ |
| `/alliance/{allianceID}/icons` | GET | Alliance icon URLs | ✅ | ✅ |

### SDE Testing Endpoints

| Endpoint | Method | Description | Source | Performance |
|----------|--------|-------------|---------|-------------|
| `/sde/status` | GET | SDE service status and statistics | Memory | Instant |
| `/sde/agent/{agentID}` | GET | Agent information | Memory | Nanoseconds |
| `/sde/category/{categoryID}` | GET | Item category information | Memory | Nanoseconds |
| `/sde/blueprint/{blueprintID}` | GET | Blueprint information | Memory | Nanoseconds |
| `/sde/agents/location/{locationID}` | GET | Agents by location | Memory | Microseconds |
| `/sde/blueprints` | GET | All blueprint IDs | Memory | Microseconds |
| `/sde/marketgroup/{marketGroupID}` | GET | Market group information | Memory | Nanoseconds |
| `/sde/marketgroups` | GET | All market groups | Memory | Microseconds |
| `/sde/metagroup/{metaGroupID}` | GET | Meta group information | Memory | Nanoseconds |
| `/sde/metagroups` | GET | All meta groups | Memory | Microseconds |
| `/sde/npccorp/{corpID}` | GET | NPC corporation information | Memory | Nanoseconds |
| `/sde/npccorps` | GET | All NPC corporations | Memory | Microseconds |
| `/sde/npccorps/faction/{factionID}` | GET | NPC corporations by faction | Memory | Microseconds |
| `/sde/typeid/{typeID}` | GET | Type ID information | Memory | Nanoseconds |
| `/sde/type/{typeID}` | GET | Type information | Memory | Nanoseconds |
| `/sde/types` | GET | All types | Memory | Milliseconds |
| `/sde/types/published` | GET | Published types only | Memory | Milliseconds |
| `/sde/types/group/{groupID}` | GET | Types by group | Memory | Microseconds |
| `/sde/typematerials/{typeID}` | GET | Type materials | Memory | Nanoseconds |

### Utility Endpoints

| Endpoint | Method | Description | Purpose |
|----------|--------|-------------|---------|
| `/services` | GET | List all available endpoints | Discovery |
| `/status` | GET | Module status information | Health |
| `/health` | GET | Health check endpoint | Monitoring |

## Response Format

All endpoints return a consistent JSON structure:

```json
{
  "source": "EVE Online ESI" | "Static Data Export",
  "endpoint": "/original/esi/endpoint",
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
```bash
# Test server status
curl http://localhost:3000/dev/esi-status

# Test character information
curl http://localhost:3000/dev/character/2112625428

# Test alliance information
curl http://localhost:3000/dev/alliance/99005065
```

### Testing SDE Integration
```bash
# Get SDE status and statistics
curl http://localhost:3000/dev/sde/status

# Test agent lookup
curl http://localhost:3000/dev/sde/agent/3008416

# Test blueprint information
curl http://localhost:3000/dev/sde/blueprint/1000001
```

### Service Discovery
```bash
# List all available endpoints
curl http://localhost:3000/dev/services
```

## Performance Characteristics

### ESI Endpoints
- **First Call**: Network latency + ESI response time
- **Cached Calls**: <1ms response time
- **Cache Duration**: Based on ESI expires header
- **Error Handling**: <10ms for validation errors

### SDE Endpoints
- **Memory Access**: Nanosecond to microsecond response times
- **No Network Calls**: All data served from memory
- **Consistent Performance**: Not affected by network conditions
- **Large Datasets**: Millisecond response for complete type lists

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

## Background Tasks

### Module Background Processing
- **Health Monitoring**: Continuous service health checks
- **Cache Maintenance**: Automatic cleanup of expired cache entries
- **Error Limit Monitoring**: Tracks ESI error rates
- **Graceful Shutdown**: Proper cleanup on application termination

### Task Lifecycle
- **Startup**: Initialize ESI clients and cache managers
- **Runtime**: Process requests and maintain caches
- **Shutdown**: Clean up resources and connections

## Integration Examples

### Using in Development
```go
// Example of accessing dev module endpoints programmatically
resp, err := http.Get("http://localhost:3000/dev/character/2112625428")
if err != nil {
    log.Fatal(err)
}

var result map[string]interface{}
json.NewDecoder(resp.Body).Decode(&result)
fmt.Printf("Character: %v\n", result["data"])
```

### Cache Testing
```go
// First call - from ESI
resp1, _ := http.Get("http://localhost:3000/dev/esi-status")
// Second call - from cache
resp2, _ := http.Get("http://localhost:3000/dev/esi-status")
```

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
- No authentication required (development only)
- Rate limiting through ESI error limits
- Secure handling of character IDs
- Proper input validation

## Troubleshooting

### Common Issues
- **ESI Rate Limits**: Monitor error limit headers
- **Cache Misses**: Check cache expiration times
- **SDE Not Loaded**: Verify SDE service initialization
- **Invalid Parameters**: Validate ID formats

### Debug Steps
1. Check `/dev/status` for module health
2. Verify ESI connectivity with `/dev/esi-status`
3. Test SDE functionality with `/dev/sde/status`
4. Monitor logs for error patterns
5. Check telemetry data for performance issues

## Future Enhancements

### Planned Features
- **More ESI Endpoints**: Additional EVE Online API coverage
- **Batch Operations**: Multi-request handling
- **Advanced Caching**: Distributed cache support
- **WebSocket Integration**: Real-time data streaming
- **Performance Metrics**: Detailed performance analytics