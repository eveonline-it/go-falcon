# Map Service Module (internal/mapservice)

## Overview

The Map Service module provides comprehensive wormhole mapping functionality for EVE Online, including signature tracking, wormhole connection management, and route calculation with wormhole integration. This module is essential for wormhole space operations, providing both public route calculation endpoints and protected signature/wormhole management with authentication.

## Architecture

### Core Components

- **Route Planning Service**: Advanced pathfinding algorithms with wormhole integration
- **Signature Management**: Protected CRUD operations for tracking cosmic signatures
- **Wormhole Management**: Protected connection tracking with mass/time status monitoring
- **SDE Integration**: Real-time system, region, and universe data access
- **Redis Caching**: Route caching and performance optimization
- **Authentication**: Centralized permission middleware for protected operations

### Files Structure

```
internal/mapservice/
├── dto/
│   ├── inputs.go          # Request DTOs with validation and auth variants
│   └── outputs.go         # Response DTOs and conversion utilities
├── models/
│   └── models.go          # Database models and structures
├── routes/
│   ├── simple_routes.go   # Public endpoints (status, search, routes)
│   ├── signature_routes.go # Protected signature management endpoints
│   └── wormhole_routes.go # Protected wormhole management endpoints
├── services/
│   ├── map_service.go     # Main service with signature operations
│   ├── wormhole_service.go # Specialized wormhole operations and static data
│   └── route_service.go   # Route calculation and pathfinding algorithms
├── module.go              # Module initialization and route registration
└── CLAUDE.md             # This documentation
```

## Data Models

### Database Collections

#### `map_signatures` - Cosmic Signatures
```go
type MapSignature struct {
    ID              primitive.ObjectID `bson:"_id,omitempty"`
    SystemID        int32             `bson:"system_id"`
    SignatureID     string            `bson:"signature_id"`     // In-game ID (ABC-123)
    Type            string            `bson:"type"`             // Combat/Data/Relic/Gas/Wormhole/Unknown
    Name            string            `bson:"name,omitempty"`
    Description     string            `bson:"description,omitempty"`
    Strength        float32           `bson:"strength,omitempty"` // Signal strength %
    CreatedBy       primitive.ObjectID `bson:"created_by"`
    CreatedByName   string            `bson:"created_by_name"`
    UpdatedBy       primitive.ObjectID `bson:"updated_by,omitempty"`
    UpdatedByName   string            `bson:"updated_by_name,omitempty"`
    SharingLevel    string            `bson:"sharing_level"`    // private/corporation/alliance
    GroupID         *primitive.ObjectID `bson:"group_id,omitempty"`
    ExpiresAt       *time.Time        `bson:"expires_at,omitempty"`
    CreatedAt       time.Time         `bson:"created_at"`
    UpdatedAt       time.Time         `bson:"updated_at"`
}
```

#### `map_wormholes` - Wormhole Connections
```go
type MapWormhole struct {
    ID               primitive.ObjectID `bson:"_id,omitempty"`
    FromSystemID     int32             `bson:"from_system_id"`
    ToSystemID       int32             `bson:"to_system_id"`
    FromSignatureID  string            `bson:"from_signature_id"`
    ToSignatureID    string            `bson:"to_signature_id,omitempty"`
    WormholeType     string            `bson:"wormhole_type,omitempty"` // B274, etc.
    MassStatus       string            `bson:"mass_status"`     // stable/destabilized/critical
    TimeStatus       string            `bson:"time_status"`     // stable/eol
    MaxMass          int64             `bson:"max_mass"`
    JumpMass         int64             `bson:"jump_mass"`
    MassRegenRate    int64             `bson:"mass_regen_rate"`
    RemainingMass    int64             `bson:"remaining_mass"`
    CreatedBy        primitive.ObjectID `bson:"created_by"`
    CreatedByName    string            `bson:"created_by_name"`
    UpdatedBy        primitive.ObjectID `bson:"updated_by,omitempty"`
    UpdatedByName    string            `bson:"updated_by_name,omitempty"`
    SharingLevel     string            `bson:"sharing_level"`   // private/corporation/alliance
    GroupID          *primitive.ObjectID `bson:"group_id,omitempty"`
    ExpiresAt        *time.Time        `bson:"expires_at,omitempty"`
    CreatedAt        time.Time         `bson:"created_at"`
    UpdatedAt        time.Time         `bson:"updated_at"`
}
```

#### `map_wormhole_statics` - Static Wormhole Information
```go
type WormholeStatic struct {
    ID            string `bson:"_id"`              // Wormhole type code (B274, etc.)
    LeadsTo       string `bson:"leads_to"`         // HS/LS/NS/C1-C6/Thera/Drifter
    MaxMass       int64  `bson:"max_mass"`         // Total mass capacity
    JumpMass      int64  `bson:"jump_mass"`        // Maximum ship mass
    MassRegenRate int64  `bson:"mass_regen_rate"`  // Mass regeneration per hour
    Lifetime      int    `bson:"lifetime"`         // Lifetime in hours
    Description   string `bson:"description"`      // Human-readable description
}
```

### Field Descriptions

- **Sharing Levels**: Control visibility - `private` (creator only), `corporation` (group members), `alliance` (all authenticated users)
- **Mass Status**: Wormhole stability - `stable` (100% mass), `destabilized` (50% mass), `critical` (10% mass)
- **Time Status**: Lifetime - `stable` (normal lifetime), `eol` (end of life, ≤4 hours remaining)
- **Wormhole Types**: Static codes like `B274` (C1), `C248` (null-sec), `K162` (exit hole)

## API Endpoints

### Public Endpoints (No Authentication Required)

#### Module Status
```
GET /map/status
```
Returns module health status and statistics.

**Response:**
```json
{
  "module": "map",
  "status": "healthy",
  "stats": {
    "signatures": 1250,
    "wormholes": 340,
    "notes": 89,
    "cached_routes": 456
  }
}
```

#### System Search
```
GET /map/search?q={query}&limit={limit}
```
Search for solar systems with fuzzy matching (exact, starts with, contains).

**Query Parameters:**
- `q`: Search query (min 2 characters)
- `limit`: Maximum results (1-100, default 20)

**Response:**
```json
[
  {
    "system_id": 30000142,
    "system_name": "Jita",
    "region_name": "The Forge",
    "constellation_name": "Kimotoro",
    "security": 0.946,
    "match_type": "exact"
  }
]
```

#### Route Calculation
```
GET /map/route?from={from}&to={to}&type={type}&include_wh={bool}&include_thera={bool}&avoid={ids}
```
Calculate optimal routes between systems with wormhole integration.

**Query Parameters:**
- `from`: Origin system ID (required)
- `to`: Destination system ID (required)
- `type`: Route type - `shortest`, `safest`, `avoid_null` (required)
- `include_wh`: Include player wormhole connections (default false)
- `include_thera`: Include Thera connections (default false)
- `avoid`: Comma-separated system IDs to avoid

**Response:**
```json
{
  "from_system_id": 30000142,
  "from_system_name": "Jita",
  "to_system_id": 31000005,
  "to_system_name": "J123456",
  "route_type": "shortest",
  "route": [
    {
      "system_id": 30000142,
      "system_name": "Jita",
      "security": 0.946,
      "region_id": 10000002,
      "region_name": "The Forge",
      "is_wormhole": false
    }
  ],
  "jumps": 12,
  "includes_wh": true,
  "includes_thera": false,
  "security_breakdown": {
    "high_sec": 8,
    "low_sec": 2,
    "null_sec": 1,
    "wormhole": 1
  }
}
```

#### Region Data
```
GET /map/region/{region_id}
```
Get complete region data with systems and gate connections.

### Protected Endpoints (Authentication Required)

All protected endpoints require JWT authentication via `Authorization: Bearer <token>` header or authentication cookie.

#### Signature Management

##### Create Signature
```
POST /map/signatures
Authorization: Bearer <token>
```
**Permission Required:** `map:signatures:manage` or `map:management:full`

**Request Body:**
```json
{
  "system_id": 31000005,
  "signature_id": "ABC-123",
  "type": "Wormhole",
  "name": "Connection to Amarr",
  "description": "High-sec static",
  "strength": 85.5,
  "sharing_level": "corporation",
  "expires_in": 24
}
```

##### List Signatures
```
GET /map/signatures?system_id={id}&type={type}&sharing={level}&include_expired={bool}
Authorization: Bearer <token>
```
**Permission Required:** Map access (any authenticated user)

##### Get Signature
```
GET /map/signatures/{signature_id}
Authorization: Bearer <token>
```

##### Update Signature
```
PUT /map/signatures/{signature_id}
Authorization: Bearer <token>
```
**Permission Required:** `map:signatures:manage` or `map:management:full`

##### Delete Signature
```
DELETE /map/signatures/{signature_id}
Authorization: Bearer <token>
```
**Permission Required:** `map:signatures:manage` or `map:management:full`

##### Batch Signature Operations
```
POST /map/signatures/batch
Authorization: Bearer <token>
```
**Permission Required:** `map:signatures:manage` or `map:management:full`

Create, update, or delete multiple signatures in one operation.

#### Wormhole Management

##### Create Wormhole Connection
```
POST /map/wormholes
Authorization: Bearer <token>
```
**Permission Required:** `map:wormholes:manage` or `map:management:full`

**Request Body:**
```json
{
  "from_system_id": 31000005,
  "to_system_id": 30000142,
  "from_signature_id": "ABC-123",
  "to_signature_id": "K162",
  "wormhole_type": "B274",
  "mass_status": "stable",
  "time_status": "stable",
  "sharing_level": "corporation"
}
```

##### List Wormhole Connections
```
GET /map/wormholes?system_id={id}&sharing={level}&include_expired={bool}
Authorization: Bearer <token>
```
**Permission Required:** Map access (any authenticated user)

##### Get Wormhole Connection
```
GET /map/wormholes/{wormhole_id}
Authorization: Bearer <token>
```

##### Update Wormhole Connection
```
PUT /map/wormholes/{wormhole_id}
Authorization: Bearer <token>
```
**Permission Required:** `map:wormholes:manage` or `map:management:full`

**Request Body:**
```json
{
  "to_signature_id": "K162",
  "wormhole_type": "B274",
  "mass_status": "destabilized",
  "time_status": "eol"
}
```

##### Delete Wormhole Connection
```
DELETE /map/wormholes/{wormhole_id}
Authorization: Bearer <token>
```
**Permission Required:** `map:wormholes:manage` or `map:management:full`

##### Batch Wormhole Operations
```
POST /map/wormholes/batch
Authorization: Bearer <token>
```
**Permission Required:** `map:wormholes:manage` or `map:management:full`

## Route Planning and Pathfinding

### Algorithm Features

- **Dijkstra's Algorithm**: Optimal pathfinding with configurable cost functions
- **Wormhole Integration**: Seamless integration of player-mapped wormhole connections
- **Security Preferences**: Route optimization based on security preferences
- **Thera Support**: Integration with Thera wormhole connections
- **System Avoidance**: Flexible system exclusion for custom routing
- **Redis Caching**: Intelligent route caching for performance

### Route Types

1. **Shortest**: Minimize total jumps regardless of security
2. **Safest**: Prefer high-security systems, avoid low/null when possible
3. **Avoid Null**: Completely avoid null-security systems if alternative exists

### Wormhole Integration

- **Player Connections**: Live wormhole data from signatures and connections
- **Mass Tracking**: Automatic mass depletion calculation and status updates
- **Expiration Handling**: Time-based connection lifecycle management
- **Static Information**: Comprehensive database of wormhole types and properties

## Authentication and Authorization

### Permission Model

The Map Service uses a hierarchical permission system:

#### Service: `map`

##### Resource: `signatures`
- **manage**: Create, update, delete signatures
- **read**: View signatures (implicit with map access)

##### Resource: `wormholes`
- **manage**: Create, update, delete wormhole connections
- **read**: View wormhole connections (implicit with map access)

##### Resource: `management`
- **full**: Complete administrative access to all map data

### Permission Requirements by Endpoint

| Endpoint | Method | Authentication | Permission Required | Description |
|----------|--------|---------------|-------------------|-------------|
| `/map/status` | GET | No | Public | Module status |
| `/map/search` | GET | No | Public | System search |
| `/map/route` | GET | No | Public | Route calculation |
| `/map/region/{id}` | GET | No | Public | Region data |
| `/map/signatures/*` | GET | Yes | Map access (any authenticated) | View signatures |
| `/map/signatures/*` | POST/PUT/DELETE | Yes | `map:signatures:manage` or `map:management:full` | Manage signatures |
| `/map/wormholes/*` | GET | Yes | Map access (any authenticated) | View wormholes |
| `/map/wormholes/*` | POST/PUT/DELETE | Yes | `map:wormholes:manage` or `map:management:full` | Manage wormholes |

### Authorization Logic

#### Visibility Rules
- **Private**: Only visible to creator
- **Corporation**: Visible to users in same groups
- **Alliance**: Visible to all authenticated users

#### Access Control
- **Signature Management**: Create/update/delete requires explicit permissions
- **Wormhole Management**: Create/update/delete requires explicit permissions
- **Map Access**: Any authenticated user can view shared data
- **Admin Override**: `map:management:full` grants complete access

## SDE Integration

### Real-time Data Access

The Map Service integrates with the SDE (Static Data Export) service for:

- **Solar System Data**: Names, security status, coordinates, celestial counts
- **Region Information**: Names, system membership, spatial organization
- **Constellation Data**: Names, system groupings, regional hierarchy
- **Universe Structure**: Complete navigational data for route calculation

### Performance Optimization

- **In-Memory Access**: O(1) system/region lookups via SDE service
- **Thread-Safe Operations**: Concurrent access to universe data
- **Efficient Lookups**: Optimized data structures for pathfinding algorithms

## Caching Strategy

### Redis Integration

- **Route Caching**: Pre-calculated routes cached by parameters hash
- **Cache Invalidation**: TTL-based expiration for route freshness
- **Wormhole Updates**: Automatic cache invalidation on connection changes
- **Performance Metrics**: Cache hit rates tracked in module status

### Caching Patterns

1. **Route Results**: 30-minute TTL for static route calculations
2. **Dynamic Routes**: 5-minute TTL for wormhole-inclusive routes
3. **System Data**: Indefinite caching via SDE service
4. **Static Info**: Permanent caching of wormhole type data

## Background Processing

### Automated Tasks

The Map Service integrates with the scheduler system for:

- **Signature Expiration**: Automatic cleanup of expired signatures
- **Wormhole Lifecycle**: Connection expiration and status updates
- **Cache Maintenance**: Periodic cleanup of stale route caches
- **Statistics Updates**: Regular aggregation of usage metrics

### Data Integrity

- **Orphan Cleanup**: Removal of signatures without valid systems
- **Consistency Checks**: Validation of wormhole connection integrity
- **Performance Monitoring**: Automatic detection of service degradation

## Error Handling

### Common HTTP Status Codes

- **200 OK**: Successful operation
- **400 Bad Request**: Invalid request parameters, malformed JSON, or validation errors
- **401 Unauthorized**: Missing or invalid authentication token
- **403 Forbidden**: Insufficient permissions for requested operation
- **404 Not Found**: Signature, wormhole, or system not found
- **500 Internal Server Error**: Database, Redis, or SDE service errors

### Error Response Format

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "Invalid system ID: system not found in SDE data",
  "instance": "/map/signatures"
}
```

## Database Indexing

### Recommended Indexes

#### map_signatures Collection
```javascript
db.map_signatures.createIndex({ "system_id": 1 })
db.map_signatures.createIndex({ "signature_id": 1, "system_id": 1 }, { unique: true })
db.map_signatures.createIndex({ "type": 1 })
db.map_signatures.createIndex({ "sharing_level": 1 })
db.map_signatures.createIndex({ "expires_at": 1 })
db.map_signatures.createIndex({ "created_by": 1 })
db.map_signatures.createIndex({ "group_id": 1 })
```

#### map_wormholes Collection
```javascript
db.map_wormholes.createIndex({ "from_system_id": 1 })
db.map_wormholes.createIndex({ "to_system_id": 1 })
db.map_wormholes.createIndex({ "from_signature_id": 1, "from_system_id": 1 })
db.map_wormholes.createIndex({ "sharing_level": 1 })
db.map_wormholes.createIndex({ "expires_at": 1 })
db.map_wormholes.createIndex({ "created_by": 1 })
db.map_wormholes.createIndex({ "group_id": 1 })
```

#### map_wormhole_statics Collection
```javascript
db.map_wormhole_statics.createIndex({ "_id": 1 }, { unique: true })
db.map_wormhole_statics.createIndex({ "leads_to": 1 })
```

## Testing Guidelines

### Unit Testing

Focus on testing:
- Route calculation algorithms with various parameters
- Signature CRUD operations and validation
- Wormhole connection logic and mass calculations
- Permission checking and authorization logic
- Cache invalidation patterns

### Integration Testing

Test scenarios:
- End-to-end route calculation with wormhole integration
- Authentication flow with protected endpoints
- Database operations with concurrent access
- Redis caching behavior and invalidation
- SDE integration and error handling

### Performance Testing

Key metrics:
- Route calculation time for various distances
- Database query performance with large datasets
- Redis cache hit rates and response times
- Concurrent user load on protected endpoints
- Memory usage with extensive wormhole networks

## Security Considerations

### Data Protection

- **Input Validation**: Comprehensive validation of all user inputs
- **SQL Injection Prevention**: MongoDB parameterized queries
- **Permission Enforcement**: Strict access control on all operations
- **Data Sanitization**: Proper handling of user-generated content

### Authentication Security

- **JWT Validation**: Proper token verification and expiration handling
- **Permission Caching**: Secure caching of user permissions
- **Audit Logging**: Complete audit trail of administrative actions
- **Rate Limiting**: Protection against abuse of public endpoints

## Configuration

### Environment Variables

```bash
# Database Configuration
MONGO_URI="mongodb://admin:password@localhost:27017/falcon?authSource=admin"
MONGO_DATABASE="falcon"

# Redis Configuration
REDIS_URI="redis://localhost:6379"
REDIS_PASSWORD="optional-password"

# Route Caching
ROUTE_CACHE_TTL_MINUTES=30          # Static route cache duration
DYNAMIC_ROUTE_CACHE_TTL_MINUTES=5   # Wormhole route cache duration
```

### Performance Tuning

```bash
# Route Calculation
MAX_ROUTE_DISTANCE=50               # Maximum jumps for route calculation
PATHFINDING_TIMEOUT_SECONDS=30      # Algorithm timeout protection
CONCURRENT_ROUTE_LIMIT=100          # Max concurrent route calculations

# Cache Configuration
ROUTE_CACHE_SIZE_MB=256             # Redis memory allocation for routes
CACHE_COMPRESSION=true              # Compress cached route data
```

## Development Guidelines

### Adding New Features

1. **DTO Design**: Create input/output DTOs with proper validation
2. **Service Logic**: Implement business logic in appropriate service
3. **Route Registration**: Add protected/public endpoints as needed
4. **Authentication**: Apply proper permission middleware
5. **Documentation**: Update API documentation and examples
6. **Testing**: Add comprehensive test coverage

### Code Patterns

- **Error Handling**: Use Huma error responses with proper HTTP status codes
- **Validation**: Leverage Huma's built-in validation with struct tags
- **Service Delegation**: Keep route handlers thin, delegate to services
- **Permission Checks**: Use MapAdapter for consistent authorization
- **Database Operations**: Follow MongoDB best practices with proper indexing

### Best Practices

- ✅ Use DTOs for all API input/output with validation
- ✅ Implement proper permission checking for protected endpoints
- ✅ Leverage SDE service for system/region data instead of hardcoding
- ✅ Cache expensive route calculations appropriately
- ✅ Handle wormhole expiration and mass depletion correctly
- ✅ Use transactions for multi-document operations
- ✅ Document all API endpoints with examples
- ❌ Never bypass authentication on protected endpoints
- ❌ Don't ignore cache invalidation on data updates
- ❌ Avoid hardcoding system/region names or IDs
- ❌ Don't expose sensitive internal data in API responses
- ❌ Never trust client-side validation alone

## Monitoring and Observability

### Key Metrics

- Route calculation latency and success rates
- Signature/wormhole creation and modification rates
- Authentication success/failure rates
- Cache hit rates and invalidation frequency
- Database query performance and connection health
- SDE service integration health and response times

### Health Checks

The module status endpoint provides comprehensive health information:
- Database connectivity and performance
- Redis connectivity and memory usage
- SDE service integration status
- Active signature/wormhole counts
- Cache statistics and performance metrics

### Logging

- Route calculation requests and performance
- Authentication failures and security events
- Database errors and connection issues
- Cache misses and invalidation events
- Permission checks and authorization decisions

## Future Enhancements

### Planned Features

- **Advanced Routing**: Multi-waypoint route optimization
- **Mass Tracking**: Real-time wormhole mass depletion
- **Map Visualization**: JSON data for interactive map rendering
- **Historical Data**: Route usage analytics and trending
- **Integration APIs**: Third-party tool integration endpoints
- **Mobile Optimization**: Enhanced mobile-friendly responses

### Scalability Improvements

- **Database Sharding**: Horizontal scaling for large datasets
- **Redis Clustering**: Distributed caching for high availability
- **Rate Limiting**: Advanced rate limiting and throttling
- **Async Processing**: Background route calculation for complex requests
- **CDN Integration**: Global distribution of static route data

This documentation provides comprehensive coverage of the Map Service module's functionality, architecture, and usage patterns. For implementation details, refer to the source code and individual service documentation.