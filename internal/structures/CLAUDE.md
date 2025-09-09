# Structures Module

## Overview

The Structures module manages EVE Online structure and station data, providing a unified interface for accessing information about both NPC stations and player-owned structures (Citadels, Engineering Complexes, etc.). It handles caching, access tracking, location hierarchy resolution, and shared Redis-based error tracking with the Assets module for failed structure access attempts.

## Architecture

### Module Structure

```
internal/structures/
├── dto/           # Data Transfer Objects
│   ├── inputs.go  # Request DTOs
│   └── outputs.go # Response DTOs
├── routes/        # HTTP route handlers
│   └── routes.go  # API endpoint definitions
├── services/      # Business logic
│   └── service.go # Structure service implementation
├── models/        # Database models
│   └── models.go  # Structure and access models
├── module.go      # Module initialization
└── CLAUDE.md      # This documentation
```

## Core Features

### 1. Structure Types

The module distinguishes between two types of structures:

#### NPC Stations (ID < 100,000,000)
- Static game data from SDE (Static Data Export)
- Always accessible to all players
- Cached indefinitely as they don't change

#### Player Structures (ID >= 100,000,000)
- Dynamic data from ESI API
- Requires character with docking access
- Cached for 1 hour with refresh capability
- Includes state information (anchoring, reinforced, etc.)

### 2. Access Management

The module tracks which characters have access to which structures:

```go
type StructureAccess struct {
    StructureID int64     // EVE structure ID
    CharacterID int32     // Character with access
    HasAccess   bool      // Access status
    LastChecked time.Time // Last verification
}
```

### 3. Location Hierarchy

Every structure is enriched with complete location data:
- Solar System (name and ID)
- Constellation (name and ID)
- Region (name and ID)
- Position (X, Y, Z coordinates for player structures)

## API Endpoints

### Public Endpoints

#### `GET /structures/status`
Returns module health status.

### Authenticated Endpoints

#### `GET /structures/{structure_id}`
Retrieves detailed information about a specific structure. **Requires authentication** with valid EVE SSO token.

**Request Parameters:**
- `structure_id` (path): EVE structure ID
- `Authorization` (header): Bearer token or Cookie authentication

**Authentication:**
- Character ID is automatically extracted from authenticated user context
- Character's EVE SSO access token is used for ESI calls to player structures
- Token expiration is validated before ESI requests

**Response:**
```json
{
  "structure_id": 1035466617946,
  "name": "Jita 4-4 Trading Hub",
  "owner_id": 98000001,
  "solar_system_id": 30000142,
  "solar_system_name": "Jita",
  "region_id": 10000002,
  "region_name": "The Forge",
  "type_id": 35832,
  "type_name": "Astrahus",
  "is_npc_station": false,
  "services": ["market", "fitting"],
  "state": "online",
  "fuel_expires": "2024-01-15T12:00:00Z"
}
```

#### `GET /structures/system/{solar_system_id}`
Lists all known structures in a solar system.

#### `GET /structures/owner/{owner_id}`
Lists all structures owned by a specific corporation.

#### `POST /structures/{structure_id}/refresh`
Forces a refresh of structure data from ESI.

#### `POST /structures/bulk-refresh`
Refreshes multiple structures at once.

**Request Body:**
```json
{
  "structure_ids": [1035466617946, 1035466617947],
  "character_id": 90000001
}
```

#### `GET /structures/character`
Returns all structures the authenticated character has access to.

## Service Layer

### StructureService

The main service provides these key methods:

```go
// Get structure by ID (handles both NPC and player structures)
GetStructure(ctx, structureID, characterID, token) (*Structure, error)

// Get structures by location
GetStructuresBySystem(ctx, solarSystemID) ([]*Structure, error)

// Get structures by owner
GetStructuresByOwner(ctx, ownerID) ([]*Structure, error)

// Bulk refresh structures
BulkRefreshStructures(ctx, structureIDs, characterID, token) (refreshed, failed []int64)

// Get character accessible structures
GetCharacterAccessibleStructures(ctx, characterID) ([]*Structure, error)
```

### Caching Strategy

**Cache Keys:**
- Structure data: `c:structure:{structure_id}` (TTL: 2 hours)

**Cache Invalidation:**
- Manual refresh via API endpoint
- Automatic refresh when data is older than 1 hour for player structures

### Redis Error Tracking Integration

The module integrates with the Assets module's `StructureAccessTracker` for intelligent error handling:

**Shared Redis Keys:**
- Failed access tracking: `falcon:assets:failed_structures:{character_id}:{structure_id}`
- Retry candidates: `falcon:assets:retry_candidates`
- ESI error budget: `falcon:assets:esi_errors:{date}`

**Error Handling Behavior:**
- **401 Authentication Errors**: Recorded in Redis with failure details
- **403 Access Denied**: Tracked with tier-based retry logic
- **Successful Access**: Clears any existing failure records for the character/structure pair
- **Character ID Storage**: Successful ESI calls save character_id to structure document

**Retry Logic:**
- Tier-based retry system with decreasing probability over time
- Daily error budget to prevent ESI abuse
- Intelligent structure selection for retry attempts

## Database Schema

### Structures Collection

```javascript
{
  _id: ObjectId,
  structure_id: Number,        // EVE structure ID
  character_id: Number,        // Character who successfully accessed structure (saved on ESI success)
  name: String,                // Structure name
  owner_id: Number,            // Corporation ID
  position: {                  // Optional for player structures
    x: Number,
    y: Number,
    z: Number
  },
  solar_system_id: Number,
  solar_system_name: String,
  region_id: Number,
  region_name: String,
  constellation_id: Number,
  constellation_name: String,
  type_id: Number,
  type_name: String,
  is_npc_station: Boolean,
  services: [String],          // Available services
  state: String,               // Structure state
  fuel_expires: Date,          // Fuel expiration
  state_timer_start: Date,     // State timer start
  state_timer_end: Date,       // State timer end
  unanchors_at: Date,          // Unanchoring time
  created_at: Date,
  updated_at: Date
}
```

### Structure Access Collection

```javascript
{
  _id: ObjectId,
  structure_id: Number,
  character_id: Number,
  has_access: Boolean,
  last_checked: Date,
  created_at: Date,
  updated_at: Date
}
```

### Indexes

- `structure_id` (unique)
- `character_id` (for access queries)
- `solar_system_id` (for system queries)
- `owner_id` (for corporation queries)

## Integration Points

### Dependencies

1. **Assets Module StructureAccessTracker**
   - Provides shared Redis-based error tracking infrastructure
   - Handles 401/403 error recording and retry logic
   - Manages ESI error budgets and failure statistics

2. **EVE Gateway Service**
   - Fetches player structure data from ESI
   - Handles OAuth token management

3. **SDE Service**
   - Provides static data for NPC stations
   - Resolves type names and system hierarchy

4. **Redis**
   - Caches structure data for performance
   - Stores shared structure access failure tracking
   - Reduces ESI API calls

### Used By

1. **Assets Module**
   - Resolves location names for assets
   - Provides structure accessibility information

2. **Corporation Module**
   - Lists corporation-owned structures
   - Tracks structure states and timers

## Error Handling

The module handles various error scenarios with intelligent Redis-based tracking:

1. **Structure Not Found**: Returns 404 when structure doesn't exist
2. **Access Denied (403)**: 
   - Returns 403 to client
   - Records failure in shared Redis tracking system
   - Implements tier-based retry logic
3. **Authentication Failed (401)**:
   - Returns 401 to client  
   - Records failure details in Redis
   - Prevents repeated ESI calls with invalid tokens
4. **ESI Errors**: Falls back to cached data when ESI is unavailable
5. **Invalid Structure ID**: Validates ID format and range
6. **Token Expiry**: Validates EVE SSO token expiration before ESI calls

**Redis Error Tracking:**
- Failed access attempts are stored with character_id and structure_id
- Implements intelligent retry selection with decreasing probability
- Maintains daily error budgets to prevent ESI abuse
- Successful access clears any existing failure records

## Performance Considerations

1. **Batch Operations**: Bulk refresh endpoint for multiple structures
2. **Caching**: Aggressive caching to minimize ESI calls
3. **Lazy Loading**: Structure data fetched only when requested
4. **Access Tracking**: Maintains access cache to avoid repeated ESI calls

## Security

1. **Authentication Required**: All structure access requires valid EVE SSO authentication
2. **Access Control**: Character must have docking rights for player structures
3. **Token Validation**: 
   - Validates JWT authentication tokens
   - Checks EVE SSO token expiration before ESI calls
   - Uses character's own access token for ESI requests
4. **Data Isolation**: Characters only see structures they have access to
5. **Rate Limiting**: Respects ESI rate limits with intelligent error budget management
6. **Failure Tracking**: Redis-based tracking prevents abuse of failed access attempts

## Future Enhancements

1. **Structure Notifications**: Alert on state changes (reinforced, low fuel)
2. **Market Integration**: Link market data to trade hubs
3. **Route Planning**: Calculate routes between structures
4. **Fuel Tracking**: Monitor and alert on low fuel levels
5. **Access Audit**: Track access changes over time
6. **Bulk Import**: Import structure lists from external sources

## Testing

### Unit Tests
- Structure type detection (NPC vs Player)
- Cache operations
- Access control logic

### Integration Tests
- ESI API integration
- Database operations
- Cache invalidation

### Manual Testing
```bash
# Get structure information (requires authentication)
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3000/api/structures/60003760

# Alternative: Using cookie authentication (development)
curl -H "Cookie: $(cat ./tmp/cookie.txt)" \
  http://localhost:3000/api/structures/60003760

# Test with player structure (requires character access)
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3000/api/structures/1035466617946

# Get structures in Jita (authenticated)
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3000/api/structures/system/30000142
```

**Authentication Notes:**
- All structure endpoints require valid authentication
- Character ID is extracted from authenticated user context
- Player structures require character to have docking access
- Failed access attempts (401/403) are tracked in Redis

## Troubleshooting

### Common Issues

1. **"Authentication required" (401)**
   - Verify JWT token is valid and not expired
   - Check EVE SSO access token hasn't expired
   - Ensure proper Authorization header or Cookie
   - Re-authenticate if tokens are expired

2. **"Structure not found" (404)**
   - Verify structure ID is correct
   - Check structure still exists in EVE Online

3. **"Access denied to structure" (403)**
   - Verify character has docking access to player structure
   - Check if structure allows public access
   - Access failure is recorded in Redis tracking system

4. **Stale Data**
   - Check Redis cache TTL settings
   - Verify ESI connectivity
   - Review structure access failure logs

5. **Missing Location Data**
   - Ensure SDE service is initialized
   - Check universe data is loaded
   - Verify system/constellation/region IDs

**Debugging Redis Error Tracking:**
- Check Redis keys: `falcon:assets:failed_structures:{character_id}:{structure_id}`
- Review daily error budget: `falcon:assets:esi_errors:{date}`
- Monitor retry candidates: `falcon:assets:retry_candidates`

## Module Status

- **Stability**: Production Ready
- **Performance**: Optimized with caching
- **Test Coverage**: Comprehensive
- **Documentation**: Complete
- **Maintenance**: Active

## Contact

For issues or questions about the Structures module, consult the main Go Falcon documentation or raise an issue in the repository.