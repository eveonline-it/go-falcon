# Assets Module

## Overview

The Assets module manages EVE Online character and corporation assets, providing comprehensive asset tracking, valuation, and monitoring capabilities. It handles asset hierarchy (containers/ships), market valuation, location resolution, and automated tracking with notifications.

## Architecture

### Module Structure

```
internal/assets/
├── dto/                    # Data Transfer Objects
│   ├── inputs.go          # Request DTOs
│   └── outputs.go         # Response DTOs
├── routes/                # HTTP route handlers
│   └── routes.go          # API endpoint definitions
├── services/              # Business logic
│   ├── service.go         # Asset service implementation
│   └── scheduled_tasks.go # Background tasks
├── models/                # Database models
│   └── models.go          # Asset models and constants
├── module.go              # Module initialization
└── CLAUDE.md              # This documentation
```

## Core Features

### 1. Asset Management

The module provides comprehensive asset tracking for:

#### Character Assets
- Personal items across all locations
- Ships and their fitted modules
- Items inside containers and ships
- Blueprint copies and originals

#### Corporation Assets
- Division-based organization (1-7)
- Role-based access (Director/Accountant required)
- Shared hangar contents
- Corporate blueprints and materials

### 2. Asset Hierarchy

The module maintains parent-child relationships for nested assets:

```go
type Asset struct {
    ItemID       int64  // Unique item instance
    ParentItemID *int64 // Container/ship this item is in
    LocationID   int64  // Station/structure/container
    IsContainer  bool   // Can contain other items
}
```

**Container Types:**
- Ships (all types)
- Station containers
- Secure containers
- Freight containers
- Specialized holds

### 3. Market Valuation

Real-time market value calculation:
- Jita 4-4 market prices (hub: 60003760)
- Per-unit and total value calculations
- Cached price data (2-hour TTL)
- Value tracking over time

### 4. Asset Tracking

Configurable monitoring system:
- Location-based tracking
- Type-specific filtering
- Value change notifications
- Historical snapshots
- Time-series data

## API Endpoints

### Public Endpoints

#### `GET /assets/status`
Returns module health status.

### Character Asset Endpoints

#### `GET /assets/character/{character_id}`
Retrieves assets for a specific character.

**Query Parameters:**
- `location_id` (optional): Filter by location
- `page` (default: 1): Page number
- `page_size` (default: 100): Items per page

**Response:**
```json
{
  "assets": [
    {
      "item_id": 1000000001,
      "type_id": 34,
      "type_name": "Tritanium",
      "location_id": 60003760,
      "location_name": "Jita IV - Moon 4 - Caldari Navy Assembly Plant",
      "location_flag": "Hangar",
      "quantity": 1000000,
      "market_price": 5.5,
      "total_value": 5500000,
      "solar_system_name": "Jita",
      "region_name": "The Forge"
    }
  ],
  "total": 150,
  "total_value": 2500000000,
  "page": 1,
  "page_size": 100,
  "last_updated": "2024-01-10T12:00:00Z"
}
```

#### `POST /assets/character/{character_id}/refresh`
Forces a refresh of character assets from ESI.

### Corporation Asset Endpoints

#### `GET /assets/corporation/{corporation_id}`
Retrieves corporation assets (requires appropriate roles).

**Query Parameters:**
- `location_id` (optional): Filter by location
- `division` (optional): Filter by division (1-7)
- `page` (default: 1): Page number
- `page_size` (default: 100): Items per page

#### `POST /assets/corporation/{corporation_id}/refresh`
Forces a refresh of corporation assets from ESI.

### Location-Based Endpoints

#### `GET /assets/location/{location_id}`
Retrieves all assets at a specific location for the authenticated character.

### Asset Tracking Endpoints

#### `POST /assets/tracking`
Creates a new asset tracking configuration.

**Request Body:**
```json
{
  "character_id": 90000001,
  "name": "Jita Trading Stock",
  "description": "Monitor trading inventory",
  "location_ids": [60003760],
  "type_ids": [34, 35, 36],
  "notify_threshold": 1000000000,
  "enabled": true
}
```

#### `GET /assets/tracking`
Retrieves tracking configurations for the authenticated user.

**Query Parameters:**
- `character_id` (optional): Filter by character
- `corporation_id` (optional): Filter by corporation
- `enabled` (optional): Filter by status

#### `PUT /assets/tracking/{tracking_id}`
Updates an existing tracking configuration.

#### `DELETE /assets/tracking/{tracking_id}`
Deletes a tracking configuration.

#### `GET /assets/snapshots`
Retrieves historical asset snapshots.

**Query Parameters:**
- `character_id` (optional): Filter by character
- `location_id` (optional): Filter by location
- `start_date` (optional): Start date (ISO 8601)
- `end_date` (optional): End date (ISO 8601)
- `limit` (default: 100): Maximum results

## Service Layer

### AssetService

Core service methods:

```go
// Character assets
GetCharacterAssets(ctx, characterID, locationID, page, pageSize) ([]*Asset, total, error)
RefreshCharacterAssets(ctx, characterID) (updated, new, removed, error)

// Corporation assets
GetCorporationAssets(ctx, corporationID, characterID, locationID, division, page, pageSize) ([]*Asset, total, error)

// Asset tracking
CreateAssetTracking(ctx, tracking) error
UpdateAssetTracking(ctx, trackingID, updates) error
DeleteAssetTracking(ctx, trackingID) error
GetAssetTracking(ctx, filter) ([]*AssetTracking, error)
ProcessAssetTracking(ctx) error

// Snapshots
GetAssetSummary(ctx, characterID, corporationID) (*AssetSnapshot, error)
```

### Scheduled Tasks

Background tasks managed by the scheduler:

1. **Asset Tracking Processor** (Every 30 minutes)
   - Processes active tracking configurations
   - Calculates value changes
   - Triggers notifications if thresholds exceeded

2. **Asset Snapshot Creator** (Daily at 4 AM)
   - Creates point-in-time snapshots
   - Stores historical data for trends
   - Maintains 365-day history

3. **Stale Asset Refresher** (Every 2 hours)
   - Identifies assets not updated recently
   - Refreshes data from ESI
   - Processes maximum 10 characters per run

## Database Schema

### Assets Collection

```javascript
{
  _id: ObjectId,
  character_id: Number,
  corporation_id: Number,     // Optional
  item_id: Number,            // Unique item instance
  type_id: Number,            // EVE type ID
  type_name: String,          // Resolved from SDE
  location_id: Number,        // Structure/station/container
  location_type: String,      // station/structure/other
  location_flag: String,      // Hangar/Cargo/CorpSAG1-7/etc
  location_name: String,      // Resolved location name
  quantity: Number,
  is_singleton: Boolean,
  is_blueprint_copy: Boolean,
  name: String,              // Custom name for ships/containers
  market_price: Number,
  total_value: Number,
  solar_system_id: Number,
  solar_system_name: String,
  region_id: Number,
  region_name: String,
  parent_item_id: Number,    // Container/ship this is in
  is_container: Boolean,
  created_at: Date,
  updated_at: Date
}
```

### Asset Snapshots Collection

```javascript
{
  _id: ObjectId,
  character_id: Number,
  corporation_id: Number,
  location_id: Number,
  total_value: Number,
  item_count: Number,
  unique_types: Number,
  snapshot_time: Date,
  created_at: Date
}
```

### Asset Tracking Collection

```javascript
{
  _id: ObjectId,
  user_id: ObjectId,
  character_id: Number,
  corporation_id: Number,
  name: String,
  description: String,
  location_ids: [Number],
  type_ids: [Number],        // Optional: specific types
  enabled: Boolean,
  notify_threshold: Number,
  last_checked: Date,
  last_value: Number,
  created_at: Date,
  updated_at: Date
}
```

### Indexes

**Assets Collection:**
- Compound: `character_id, location_id`
- Compound: `corporation_id, location_id`
- Single: `item_id`, `type_id`, `location_flag`, `updated_at`

**Snapshots Collection:**
- Compound: `character_id, snapshot_time`
- Compound: `corporation_id, snapshot_time`
- Single: `location_id`

**Tracking Collection:**
- Single: `user_id`, `character_id`, `corporation_id`, `enabled`

## Location Flags

Common location flags and their meanings:

```go
const (
    LocationFlagAssetSafety     = "AssetSafety"      // Asset safety
    LocationFlagCargo           = "Cargo"            // Ship cargo hold
    LocationFlagCorpSAG1        = "CorpSAG1"        // Corp Division 1
    LocationFlagCorpSAG2        = "CorpSAG2"        // Corp Division 2
    LocationFlagCorpSAG3        = "CorpSAG3"        // Corp Division 3
    LocationFlagCorpSAG4        = "CorpSAG4"        // Corp Division 4
    LocationFlagCorpSAG5        = "CorpSAG5"        // Corp Division 5
    LocationFlagCorpSAG6        = "CorpSAG6"        // Corp Division 6
    LocationFlagCorpSAG7        = "CorpSAG7"        // Corp Division 7
    LocationFlagDroneBay        = "DroneBay"         // Drone bay
    LocationFlagFleetHangar     = "FleetHangar"      // Fleet hangar
    LocationFlagHangar          = "Hangar"           // Personal hangar
    LocationFlagOfficeFolder    = "OfficeFolder"     // Corporation office
    LocationFlagShipHangar      = "ShipHangar"       // Ship maintenance bay
    LocationFlagStructureFuel   = "StructureFuel"    // Structure fuel bay
)
```

## Integration Points

### Dependencies

1. **Structures Module**
   - Resolves location names
   - Provides structure access information
   - Handles NPC station data

2. **EVE Gateway Service**
   - Fetches asset data from ESI
   - Handles pagination
   - Manages OAuth tokens

3. **SDE Service**
   - Resolves type names
   - Identifies ships and containers
   - Provides market group data

4. **Scheduler Module**
   - Manages background tasks
   - Handles task execution
   - Provides cron scheduling

### Cache Strategy

**Cache Keys:**
- Character assets: `c:assets:char:{character_id}[:loc:{location_id}]` (30 min)
- Corporation assets: `c:assets:corp:{corp_id}[:loc:{location_id}]` (30 min)
- Market prices: `marketHub:60003760:{type_id}` (2 hours)

## Performance Optimizations

1. **Bulk Operations**: Process multiple assets in single database operations
2. **Pagination**: Large asset lists are paginated
3. **Caching**: Aggressive caching reduces ESI calls
4. **Batch Updates**: Assets updated in batches during refresh
5. **Lazy Loading**: Location/market data fetched only when needed

## Security Considerations

1. **Access Control**: 
   - Character assets require ownership
   - Corporation assets require Director/Accountant roles
   - Tracking configurations are user-specific

2. **Data Isolation**:
   - Characters only see their own assets
   - Corporation roles enforced at API level
   - Tracking data isolated by user

3. **Rate Limiting**:
   - Respects ESI rate limits
   - Throttles refresh operations
   - Limits bulk operations

## Error Handling

1. **ESI Errors**: Falls back to cached data when available
2. **Missing Structures**: Gracefully handles unknown locations
3. **Invalid Types**: Skips unrecognized type IDs
4. **Access Denied**: Returns 403 for unauthorized access
5. **Stale Data**: Returns cached data with warning

## Future Enhancements

1. **Advanced Filtering**:
   - Market group filtering
   - Value range filters
   - Meta level filtering

2. **Notifications**:
   - Discord/Slack webhooks
   - Email alerts
   - In-app notifications

3. **Analytics**:
   - Asset value trends
   - Portfolio analysis
   - Trade opportunity detection

4. **Import/Export**:
   - CSV/Excel export
   - Bulk import from files
   - API for external tools

5. **Automation**:
   - Auto-refresh based on activity
   - Smart caching based on usage
   - Predictive pre-fetching

## Testing

### Unit Tests
- Asset hierarchy building
- Market value calculations
- Container detection logic

### Integration Tests
- ESI API integration
- Database operations
- Cache behavior
- Scheduled tasks

### Manual Testing
```bash
# Get character assets
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:3000/api/assets/character/90000001

# Refresh assets
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:3000/api/assets/character/90000001/refresh

# Create tracking
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"character_id": 90000001, "name": "Test", "location_ids": [60003760]}' \
  http://localhost:3000/api/assets/tracking
```

## Troubleshooting

### Common Issues

1. **Missing Assets**
   - Verify character has logged in recently
   - Check ESI token scopes include assets
   - Ensure refresh has been triggered

2. **Wrong Values**
   - Market prices update every 2 hours
   - Check market hub configuration
   - Verify type IDs are tradeable

3. **Slow Performance**
   - Check cache hit rates
   - Verify database indexes exist
   - Monitor ESI response times

4. **Access Errors**
   - Verify corporation roles
   - Check OAuth token validity
   - Ensure character is in corporation

## Module Status

- **Stability**: Production Ready
- **Performance**: Optimized with caching and pagination
- **Test Coverage**: Comprehensive
- **Documentation**: Complete
- **Maintenance**: Active

## Contact

For issues or questions about the Assets module, consult the main Go Falcon documentation or raise an issue in the repository.