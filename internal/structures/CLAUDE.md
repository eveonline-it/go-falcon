# Structures Module

## Overview

The Structures module manages EVE Online structure and station data, providing a unified interface for accessing information about both NPC stations and player-owned structures (Citadels, Engineering Complexes, etc.). It handles caching, access tracking, and location hierarchy resolution.

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
Retrieves detailed information about a specific structure.

**Request Parameters:**
- `structure_id` (path): EVE structure ID

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
GetStructure(ctx, structureID, characterID) (*Structure, error)

// Get structures by location
GetStructuresBySystem(ctx, solarSystemID) ([]*Structure, error)

// Get structures by owner
GetStructuresByOwner(ctx, ownerID) ([]*Structure, error)

// Bulk refresh structures
BulkRefreshStructures(ctx, structureIDs, characterID) (refreshed, failed []int64)

// Get character accessible structures
GetCharacterAccessibleStructures(ctx, characterID) ([]*Structure, error)
```

### Caching Strategy

**Cache Keys:**
- Structure data: `c:structure:{structure_id}` (TTL: 2 hours)

**Cache Invalidation:**
- Manual refresh via API endpoint
- Automatic refresh when data is older than 1 hour for player structures

## Database Schema

### Structures Collection

```javascript
{
  _id: ObjectId,
  structure_id: Number,        // EVE structure ID
  character_id: Number,        // Character with access (optional)
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

1. **EVE Gateway Service**
   - Fetches player structure data from ESI
   - Handles OAuth token management

2. **SDE Service**
   - Provides static data for NPC stations
   - Resolves type names and system hierarchy

3. **Redis**
   - Caches structure data for performance
   - Reduces ESI API calls

### Used By

1. **Assets Module**
   - Resolves location names for assets
   - Provides structure accessibility information

2. **Corporation Module**
   - Lists corporation-owned structures
   - Tracks structure states and timers

## Error Handling

The module handles various error scenarios:

1. **Structure Not Found**: Returns 404 when structure doesn't exist
2. **Access Denied**: Returns 403 when character lacks access to player structure
3. **ESI Errors**: Falls back to cached data when ESI is unavailable
4. **Invalid Structure ID**: Validates ID format and range

## Performance Considerations

1. **Batch Operations**: Bulk refresh endpoint for multiple structures
2. **Caching**: Aggressive caching to minimize ESI calls
3. **Lazy Loading**: Structure data fetched only when requested
4. **Access Tracking**: Maintains access cache to avoid repeated ESI calls

## Security

1. **Access Control**: Character must have docking rights for player structures
2. **Data Isolation**: Characters only see structures they have access to
3. **Rate Limiting**: Respects ESI rate limits
4. **Token Validation**: Ensures valid OAuth tokens for ESI calls

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
# Get structure information
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:3000/api/structures/60003760

# Refresh structure data
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:3000/api/structures/60003760/refresh

# Get structures in Jita
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:3000/api/structures/system/30000142
```

## Troubleshooting

### Common Issues

1. **"Structure not found"**
   - Verify structure ID is correct
   - Check character has docking access
   - Ensure ESI token has required scopes

2. **Stale Data**
   - Use refresh endpoint to force update
   - Check Redis cache TTL settings
   - Verify ESI connectivity

3. **Missing Location Data**
   - Ensure SDE service is initialized
   - Check universe data is loaded
   - Verify system/constellation/region IDs

## Module Status

- **Stability**: Production Ready
- **Performance**: Optimized with caching
- **Test Coverage**: Comprehensive
- **Documentation**: Complete
- **Maintenance**: Active

## Contact

For issues or questions about the Structures module, consult the main Go Falcon documentation or raise an issue in the repository.