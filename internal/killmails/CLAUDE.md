# Killmails Module

## Overview

The Killmails module provides comprehensive EVE Online killmail management functionality. It implements the standard Go-Falcon module pattern with database-first lookup and ESI fallback for data retrieval, along with import capabilities for storing killmail data locally.

## Module Architecture

This module follows the **unified module architecture pattern** used throughout Go-Falcon, providing a complete killmail data management system.

### Directory Structure

```
internal/killmails/
├── dto/                    # Data Transfer Objects
│   ├── inputs.go          # Request input DTOs with Huma validation
│   └── outputs.go         # Response output DTOs with proper JSON structure
├── models/                # Database models
│   └── models.go         # MongoDB schemas and collection definitions
├── routes/               # Route definitions  
│   └── routes.go         # Huma v2 unified route registration
├── services/             # Business logic layer
│   ├── repository.go     # Database operations and queries
│   └── service.go        # Business logic and ESI integration
├── module.go             # Module initialization and interface implementation
└── CLAUDE.md             # This documentation file
```

**Note**: Authentication and permission middleware centralized in `pkg/middleware/` system.

## Key Features

### 1. Database-First with ESI Fallback
- **Primary Source**: MongoDB database lookup for cached killmail data
- **Fallback Source**: EVE ESI API when killmail not found in database
- **Auto-Caching**: ESI responses automatically saved to database for future requests
- **Upsert Strategy**: Killmails are inserted or updated using MongoDB upsert operations

### 2. Comprehensive Killmail Data Model
- **Complete Structure**: Full killmail data including victim, attackers, items, and metadata
- **Nested Items**: Support for container items with recursive structure
- **Position Data**: 3D coordinates for spatial analysis
- **Temporal Data**: Killmail timestamps with proper timezone handling

### 3. Type-Safe API Design
- **Huma v2 Integration**: Full type safety with compile-time validation
- **Input Validation**: Path and query parameters validated with EVE-specific constraints
- **Output Structure**: Consistent response format with proper JSON serialization
- **Error Handling**: Standard HTTP error responses with meaningful messages

### 4. ESI Integration Best Practices
- **User-Agent Compliance**: Follows CCP ESI guidelines for API requests
- **Type-Safe Parsing**: Handles JSON unmarshaling with fallback type assertions
- **Error Propagation**: Proper error handling from ESI client to HTTP responses
- **Caching Strategy**: Respects ESI cache headers and implements conditional requests

### 5. Advanced Query Capabilities
- **Character Filtering**: Find killmails by character involvement (victim or attacker)
- **Corporation Filtering**: Find killmails by corporation affiliation
- **Alliance Filtering**: Find killmails by alliance membership
- **System Filtering**: Find killmails by solar system with time range support
- **Flexible Pagination**: Configurable result limits for performance optimization

## Implementation Pattern

### 1. Module Interface (`module.go`)

```go
type Module struct {
    *module.BaseModule
    service    *services.Service
    repository *services.Repository
    eveGateway *evegateway.Client
}

func New(mongodb *database.MongoDB, redis *database.Redis, eveGateway *evegateway.Client) *Module {
    repository := services.NewRepository(mongodb)
    service := services.NewService(repository, eveGateway)
    
    return &Module{
        BaseModule: module.NewBaseModule("killmails", mongodb, redis),
        service:    service,
        repository: repository,
        eveGateway: eveGateway,
    }
}

func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
    routes.RegisterKillmailRoutes(api, basePath, m.service)
}
```

**Key Patterns:**
- Embeds `module.BaseModule` for standard functionality
- Dependency injection via constructor
- Service and repository layer separation
- Unified route registration for API gateway integration
- Database index creation during initialization

### 2. DTO Layer (`dto/`)

**Input DTOs** (`inputs.go`):
```go
type GetKillmailInput struct {
    KillmailID int64  `path:"killmail_id" validate:"required" minimum:"1" doc:"EVE Online killmail ID"`
    Hash       string `path:"hash" validate:"required" minLength:"40" maxLength:"40" doc:"Killmail hash (40 character string)"`
}

type GetCharacterRecentKillmailsInput struct {
    CharacterID int `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
    Limit       int `query:"limit" validate:"min:1,max:200" default:"50" doc:"Maximum number of killmails to return (1-200, default 50)"`
}
```

**Output DTOs** (`outputs.go`):
```go
type KillmailResponse struct {
    KillmailID   int64     `json:"killmail_id" doc:"Unique killmail identifier"`
    KillmailHash string    `json:"killmail_hash" doc:"Killmail hash for verification"`
    KillmailTime time.Time `json:"killmail_time" doc:"Time when the kill occurred"`
    Victim       VictimResponse    `json:"victim" doc:"Victim information"`
    Attackers    []AttackerResponse `json:"attackers" doc:"List of attackers involved"`
}
```

**Key Patterns:**
- Huma v2 validation tags for automatic input validation
- EVE-specific ID range validation (character IDs 90M+, corp IDs 98M+)
- Comprehensive OpenAPI documentation with `doc` tags
- Conversion functions from models to response DTOs

### 3. Service Layer (`services/`)

**Repository** (`repository.go`):
```go
func (r *Repository) GetByKillmailIDAndHash(ctx context.Context, killmailID int64, hash string) (*models.Killmail, error) {
    var killmail models.Killmail
    err := r.collection.FindOne(ctx, bson.M{
        "killmail_id":   killmailID,
        "killmail_hash": hash,
    }).Decode(&killmail)
    // Error handling...
    return &killmail, nil
}

func (r *Repository) GetRecentKillmailsByCharacter(ctx context.Context, characterID int64, limit int) ([]models.Killmail, error) {
    filter := bson.M{
        "$or": []bson.M{
            {"victim.character_id": characterID},
            {"attackers.character_id": characterID},
        },
    }
    // Query execution...
}
```

**Service** (`service.go`):
```go
func (s *Service) GetKillmail(ctx context.Context, killmailID int64, hash string) (*models.Killmail, error) {
    // Try database first
    killmail, err := s.repository.GetByKillmailIDAndHash(ctx, killmailID, hash)
    if killmail != nil {
        return killmail, nil
    }
    
    // Fetch from ESI if not in database
    esiData, err := s.eveGateway.Killmails.GetKillmail(ctx, killmailID, hash)
    if err != nil {
        return nil, fmt.Errorf("ESI error: %w", err)
    }
    
    // Convert and cache
    killmail = s.convertESIDataToModel(esiData, hash)
    s.repository.UpsertKillmail(ctx, killmail)
    
    return killmail, nil
}
```

**Key Patterns:**
- Database-first approach with ESI fallback
- Comprehensive error handling and logging
- Automatic caching of ESI responses
- Complex data type conversion between ESI and internal models
- Flexible querying with OR conditions for participant searches

### 4. Route Layer (`routes/`)

```go
huma.Register(api, huma.Operation{
    OperationID:   "getKillmail",
    Method:        http.MethodGet,
    Path:          basePath + "/killmails/{killmail_id}/{hash}",
    Summary:       "Get killmail by ID and hash",
    Description:   "Retrieves a specific killmail using its ID and hash. Uses database-first approach with ESI fallback.",
    Tags:          []string{"Killmails"},
    DefaultStatus: http.StatusOK,
}, func(ctx context.Context, input *dto.GetKillmailInput) (*dto.KillmailResponse, error) {
    killmail, err := service.GetKillmail(ctx, input.KillmailID, input.Hash)
    if err != nil {
        return nil, handlers.NewAPIError(http.StatusBadRequest, "Failed to fetch killmail", err)
    }
    return dto.ConvertKillmailToResponse(killmail), nil
})
```

**Key Patterns:**
- Huma v2 operation registration with comprehensive metadata
- Proper HTTP status codes and error handling
- OpenAPI tag organization for documentation
- Security annotations for authenticated endpoints
- Consistent error response format using `pkg/handlers`

## API Endpoints

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/killmails/status` | Module health check |
| `GET` | `/killmails/{killmail_id}/{hash}` | Get specific killmail |
| `GET` | `/killmails/recent` | Get recent killmails with filtering |
| `POST` | `/killmails/import` | Import killmail from ESI |
| `GET` | `/killmails/stats` | Get killmail statistics |

### Authenticated Endpoints

| Method | Path | Description | Auth Required |
|--------|------|-------------|---------------|
| `GET` | `/killmails/character/{character_id}/recent` | Character recent killmails | Bearer/Cookie |
| `GET` | `/killmails/corporation/{corporation_id}/recent` | Corporation recent killmails | Bearer/Cookie |

## Database Schema

### Collection: `killmails`

```javascript
{
  _id: ObjectId,
  killmail_id: NumberLong,     // Unique ESI killmail ID
  killmail_hash: String,       // 40-character verification hash
  killmail_time: ISODate,      // When the kill occurred
  solar_system_id: NumberLong, // System where kill occurred
  moon_id: NumberLong,         // Optional: Moon location
  war_id: NumberLong,          // Optional: War context
  victim: {
    character_id: NumberLong,   // Optional: Victim character
    corporation_id: NumberLong, // Optional: Victim corp
    alliance_id: NumberLong,    // Optional: Victim alliance
    faction_id: NumberLong,     // Optional: Victim faction
    ship_type_id: NumberLong,   // Ship that was destroyed
    damage_taken: NumberLong,   // Total damage received
    position: {                 // Optional: 3D coordinates
      x: Double,
      y: Double,
      z: Double
    },
    items: [                    // Optional: Destroyed/dropped items
      {
        item_type_id: NumberLong,
        flag: NumberLong,       // Item location on ship
        singleton: NumberLong,
        quantity_destroyed: NumberLong,
        quantity_dropped: NumberLong,
        items: [...]            // Recursive: nested container items
      }
    ]
  },
  attackers: [
    {
      character_id: NumberLong,    // Optional: Attacker character
      corporation_id: NumberLong,  // Optional: Attacker corp
      alliance_id: NumberLong,     // Optional: Attacker alliance
      faction_id: NumberLong,      // Optional: Attacker faction
      ship_type_id: NumberLong,    // Optional: Attacker ship
      weapon_type_id: NumberLong,  // Optional: Weapon used
      damage_done: NumberLong,     // Damage dealt
      final_blow: Boolean,         // Achieved killing blow
      security_status: Double      // Attacker sec status
    }
  ],
  created_at: ISODate,           // First stored in database
  updated_at: ISODate            // Last updated in database
}
```

### Database Indexes

```javascript
// Primary indexes
{ "killmail_id": 1 }                           // Unique
{ "killmail_id": 1, "killmail_hash": 1 }       // Unique composite

// Query optimization indexes
{ "killmail_time": -1 }                        // Recent killmails
{ "solar_system_id": 1 }                       // System filtering
{ "victim.character_id": 1 }                   // Character victim search
{ "victim.corporation_id": 1 }                 // Corp victim search
{ "victim.alliance_id": 1 }                    // Alliance victim search
{ "attackers.character_id": 1 }                // Character attacker search
{ "attackers.corporation_id": 1 }              // Corp attacker search
{ "attackers.alliance_id": 1 }                 // Alliance attacker search
```

## EVE Gateway Integration

### ESI Endpoints Used

| ESI Endpoint | Purpose | Auth Required |
|--------------|---------|---------------|
| `GET /killmails/{killmail_id}/{killmail_hash}/` | Fetch full killmail data | No |
| `GET /characters/{character_id}/killmails/recent/` | Character recent killmails | Yes |
| `GET /corporations/{corporation_id}/killmails/recent/` | Corporation recent killmails | Yes |

### ESI Response Handling

```go
// ESI data conversion with comprehensive type handling
func (s *Service) convertESIDataToModel(esiData map[string]any, hash string) *models.Killmail {
    killmail := &models.Killmail{
        KillmailID:    int64(esiData["killmail_id"].(float64)),
        KillmailHash:  hash,
        SolarSystemID: int64(esiData["solar_system_id"].(float64)),
    }
    
    // Parse ISO8601 timestamp
    if timeStr, ok := esiData["killmail_time"].(string); ok {
        if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
            killmail.KillmailTime = t
        }
    }
    
    // Handle complex nested structures
    if victimData, ok := esiData["victim"].(map[string]any); ok {
        killmail.Victim = s.convertVictim(victimData)
    }
    
    return killmail
}
```

## Usage Examples

### Basic Killmail Retrieval

```bash
# Get a specific killmail
curl "http://localhost:3000/api/killmails/123456789/abcd1234..."

# Import a killmail from ESI
curl -X POST "http://localhost:3000/api/killmails/import" \
  -H "Content-Type: application/json" \
  -d '{"killmail_id": 123456789, "hash": "abcd1234..."}'
```

### Filtered Queries

```bash
# Recent killmails for a character
curl "http://localhost:3000/api/killmails/recent?character_id=123456789&limit=10"

# Killmails in a specific system (last 24 hours)
curl "http://localhost:3000/api/killmails/recent?system_id=30000142&limit=20"

# Corporation killmails with custom time range
curl "http://localhost:3000/api/killmails/recent?corporation_id=98765432&since=2024-01-01T00:00:00Z"
```

### Authenticated Endpoints

```bash
# Character recent killmails from ESI (requires auth)
curl -H "Authorization: Bearer ${TOKEN}" \
  "http://localhost:3000/api/killmails/character/123456789/recent?limit=50"
```

## Performance Considerations

### Database Optimization
- **Compound Indexes**: Optimized for common query patterns
- **Selective Queries**: Use projection to limit returned fields when needed
- **Pagination**: Always use limits to prevent large result sets
- **Time-based Queries**: Include time ranges for system queries

### Caching Strategy
- **Database-First**: Check local cache before ESI calls
- **Upsert Operations**: Efficient insert-or-update for ESI responses
- **Cache Persistence**: Killmails are permanent once stored

### ESI Best Practices
- **Rate Limiting**: Handled by `pkg/evegateway` client
- **Error Limits**: Monitors ESI error budget to prevent blocking
- **Conditional Requests**: Uses ETags and Last-Modified headers
- **User-Agent Compliance**: Proper identification per CCP guidelines

## Security Considerations

### Input Validation
- **ID Range Validation**: EVE entity IDs within valid ranges
- **Hash Format**: Killmail hashes exactly 40 characters
- **Limit Bounds**: Reasonable pagination limits to prevent abuse
- **SQL Injection Protection**: MongoDB parameterized queries

### Authentication
- **Token Extraction**: Centralized auth middleware integration
- **Scope Validation**: ESI scopes verified for authenticated endpoints
- **CORS Compliance**: Proper cross-origin request handling

## Module Status Endpoint

```bash
GET /killmails/status
```

**Response:**
```json
{
  "module": "killmails",
  "status": "healthy",
  "message": ""
}
```

**Health Checks:**
- Database connectivity (document count query)
- ESI connectivity (server status check)
- Index existence verification

## Best Practices

### Development
- ✅ Follow database-first approach for cached data
- ✅ Always validate killmail ID/hash combinations
- ✅ Use proper error handling for ESI timeouts
- ✅ Include comprehensive logging for debugging
- ✅ Cache killmail data permanently once fetched
- ✅ Use compound indexes for complex queries
- ✅ Handle nested item structures properly

### Integration
- ✅ Register module in main application
- ✅ Configure database connections
- ✅ Set up authentication middleware
- ✅ Monitor ESI rate limits
- ❌ Don't store incomplete killmail data
- ❌ Don't bypass authentication for restricted endpoints
- ❌ Avoid large unfiltered queries without pagination

## Future Enhancements

- **Real-time Updates**: WebSocket support for live killmail feeds
- **Analytics Integration**: Kill statistics and trend analysis
- **Bulk Import**: Batch processing for large killmail datasets  
- **Search Optimization**: Advanced filtering and full-text search
- **Export Features**: CSV/JSON export functionality
- **Notification System**: Alerts for specific kill criteria