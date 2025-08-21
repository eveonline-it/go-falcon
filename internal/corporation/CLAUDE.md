# Corporation Module

## Overview

The Corporation module provides corporation information management functionality for EVE Online corporations. It implements the standard Go-Falcon module pattern with database-first lookup and ESI fallback for data retrieval, following the same architecture as the Character module.

## Module Architecture

This module follows the **unified module architecture pattern** used throughout Go-Falcon, providing a template implementation for EVE corporate data management.

### Directory Structure

```
internal/corporation/
├── dto/                    # Data Transfer Objects
│   ├── inputs.go          # Request input DTOs with Huma validation
│   └── outputs.go         # Response output DTOs with proper JSON structure
├── middleware/            # Module-specific middleware (currently empty)
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

## Key Features

### 1. Database-First with ESI Fallback
- **Primary Source**: MongoDB database lookup for cached corporation data
- **Fallback Source**: EVE ESI API when corporation not found in database
- **Auto-Caching**: ESI responses automatically saved to database for future requests
- **Upsert Strategy**: Corporations are inserted or updated using MongoDB upsert operations

### 2. Type-Safe API Design
- **Huma v2 Integration**: Full type safety with compile-time validation
- **Input Validation**: Path parameters validated with corporation ID constraints
- **Output Structure**: Consistent response format with proper JSON serialization
- **Error Handling**: Standard HTTP error responses with meaningful messages

### 3. ESI Integration Best Practices
- **User-Agent Compliance**: Follows CCP ESI guidelines for API requests
- **Type-Safe Parsing**: Handles JSON unmarshaling with fallback type assertions
- **Error Propagation**: Proper error handling from ESI client to HTTP responses

## Implementation Pattern

### 1. Module Interface (`module.go`)

```go
type Module struct {
    *module.BaseModule
    service *services.Service
    routes  *routes.Module
}

func NewModule(mongodb *database.MongoDB, redis *database.Redis, eveClient *evegateway.Client) *Module {
    repository := services.NewRepository(mongodb)
    service := services.NewService(repository, eveClient)
    routesModule := routes.NewModule(service)
    
    return &Module{
        BaseModule: module.NewBaseModule("corporation", mongodb, redis),
        service:    service,
        routes:     routesModule,
    }
}

func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
    m.routes.RegisterUnifiedRoutes(api, basePath)
}
```

**Key Patterns:**
- Embeds `module.BaseModule` for standard functionality
- Dependency injection via constructor with separate routes module
- Service layer abstraction with repository pattern
- Unified route registration for API gateway integration

### 2. DTO Layer (`dto/`)

**Input DTOs** (`inputs.go`):
```go
type GetCorporationInput struct {
    CorporationID int `path:"corporation_id" minimum:"1" description:"Corporation ID to retrieve information for" example:"98000001"`
}
```

**Output DTOs** (`outputs.go`):
```go
type CorporationInfo struct {
    AllianceID      *int      `json:"alliance_id,omitempty" description:"Alliance ID if corporation is in an alliance"`
    CEOCharacterID  int       `json:"ceo_id" description:"Character ID of the corporation CEO"`
    Name            string    `json:"name" description:"Corporation name"`
    // ... additional fields
}

type CorporationInfoOutput struct {
    Body CorporationInfo `json:"body"`
}
```

**Key Patterns:**
- Separate input/output structures
- Huma v2 validation tags (`minimum`, `description`, `example`)
- OpenAPI documentation tags for automatic spec generation
- **Wrapper output structures** following Huma v2 pattern (Body field is crucial)

### 3. Route Registration (`routes/routes.go`)

```go
type Module struct {
    service *services.Service
}

func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
    // Corporation information endpoint
    huma.Register(api, huma.Operation{
        OperationID: "corporation-get-info",
        Method:      "GET",
        Path:        basePath + "/{corporation_id}",
        Summary:     "Get Corporation Information",
        Description: "Retrieve detailed information about a corporation from EVE Online ESI API. Data is cached locally for performance.",
        Tags:        []string{"Corporations"},
    }, func(ctx context.Context, input *dto.GetCorporationInput) (*dto.CorporationInfoOutput, error) {
        return m.getCorporationInfo(ctx, input)
    })
    
    // Health check endpoint
    huma.Register(api, huma.Operation{
        OperationID: "corporation-health-check",
        Method:      "GET",
        Path:        basePath + "/health",
        Summary:     "Corporation Module Health Check",
        Description: "Check if the corporation module is functioning properly",
        Tags:        []string{"Health"},
    }, func(ctx context.Context, input *struct{}) (*HealthResponse, error) {
        return &HealthResponse{
            Body: HealthCheck{Healthy: true, Module: "corporation"},
        }, nil
    })
}
```

**Key Patterns:**
- Separate routes module structure
- Complete OpenAPI metadata (OperationID, Summary, Description, Tags)
- Direct service layer delegation through helper methods
- Context-aware handlers
- Standard HTTP error handling via Huma
- **Proper wrapper response structures** for both data and health endpoints

### 4. Service Layer (`services/service.go`)

```go
type Service struct {
    repository *Repository
    eveClient  *evegateway.Client
}

func (s *Service) GetCorporationInfo(ctx context.Context, corporationID int) (*dto.CorporationInfoOutput, error) {
    // 1. Try database first
    corporation, err := s.repository.GetCorporationByID(ctx, corporationID)
    if err != nil && err != mongo.ErrNoDocuments {
        slog.ErrorContext(ctx, "Failed to get corporation from database", "error", err)
    }
    
    // 2. If not found in database, fetch from ESI
    if corporation == nil || err == mongo.ErrNoDocuments {
        esiData, err := s.eveClient.GetCorporationInfo(ctx, corporationID)
        if err != nil {
            return nil, fmt.Errorf("failed to get corporation information: %w", err)
        }
        
        // 3. Convert ESI data to model
        corporation = s.convertESIDataToModel(esiData, corporationID)
        
        // 4. Save to database
        if err := s.repository.UpdateCorporation(ctx, corporation); err != nil {
            slog.WarnContext(ctx, "Failed to save corporation to database", "error", err)
        }
    }
    
    // 5. Convert to output DTO
    return s.convertModelToOutput(corporation), nil
}
```

**Key Patterns:**
- Database-first strategy with ESI fallback
- Structured logging with context
- Proper error propagation with wrapping
- Conversion between ESI data, models, and DTOs
- Auto-caching of ESI responses
- Context propagation throughout the call chain
- **Wrapper DTO creation** in convertModelToOutput

### 5. Repository Layer (`services/repository.go`)

```go
type Repository struct {
    mongodb    *database.MongoDB
    collection *mongo.Collection
}

func (r *Repository) GetCorporationByID(ctx context.Context, corporationID int) (*models.Corporation, error) {
    var corporation models.Corporation
    filter := bson.M{"corporation_id": corporationID, "deleted_at": bson.M{"$exists": false}}
    
    err := r.collection.FindOne(ctx, filter).Decode(&corporation)
    if err != nil {
        return nil, err  // mongo.ErrNoDocuments handled in service layer
    }
    
    return &corporation, nil
}

func (r *Repository) UpdateCorporation(ctx context.Context, corporation *models.Corporation) error {
    corporation.UpdatedAt = time.Now().UTC()
    
    filter := bson.M{"corporation_id": corporation.CorporationID, "deleted_at": bson.M{"$exists": false}}
    update := bson.M{"$set": corporation}
    
    _, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
    return err
}
```

**Key Patterns:**
- Collection-based MongoDB operations
- Soft delete support with `deleted_at` filtering
- Upsert operations for insert/update behavior
- Automatic timestamp management
- Context-aware database operations
- Proper not-found handling (delegates to service layer)

### 6. Database Models (`models/models.go`)

```go
type Corporation struct {
    ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    CorporationID   int                `bson:"corporation_id" json:"corporation_id"`
    Name            string             `bson:"name" json:"name"`
    Ticker          string             `bson:"ticker" json:"ticker"`
    // ... EVE-specific fields
    AllianceID      *int               `bson:"alliance_id,omitempty" json:"alliance_id,omitempty"`
    CEOCharacterID  int                `bson:"ceo_character_id" json:"ceo_character_id"`
    
    // Metadata
    CreatedAt time.Time  `bson:"created_at" json:"created_at"`
    UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
    DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

const CorporationCollection = "corporations"
```

**Key Patterns:**
- MongoDB `primitive.ObjectID` for document ID
- Separate BSON and JSON tags for database/API serialization
- Business identifier fields (`corporation_id`) separate from MongoDB `_id`
- Standard timestamp fields with soft delete support
- Collection name constants for consistency
- Pointer types for optional EVE fields

## API Endpoints

### GET `/{corporation_id}` - Get Corporation Information

**Description**: Retrieves corporation information from database or ESI if not cached.

**Parameters**:
- `corporation_id` (path, required): EVE Online corporation ID (minimum: 1)

**Response**: Corporation information with EVE game data

**Implementation Flow**:
1. Validate corporation ID format
2. Query MongoDB for cached corporation data
3. If found, return cached data
4. If not found, fetch from EVE ESI API (`/v4/corporations/{corporation_id}/`)
5. Parse ESI response and save to database
6. Return corporation information

**Error Handling**:
- `400`: Invalid corporation ID format
- `404`: Corporation not found in ESI
- `500`: Database or ESI communication errors

### GET `/health` - Corporation Module Health Check

**Description**: Verifies the corporation module is functioning properly.

**Response**: Health status and module name

## Database Schema

### Corporations Collection

```json
{
  "_id": ObjectId("..."),
  "corporation_id": 98000001,
  "name": "Vertex Dryrun Test Corp",
  "ticker": "VDTC",
  "description": "Enter a description of your corporation here",
  "url": "http://",
  "alliance_id": null,
  "ceo_character_id": 1,
  "creator_id": 1592221698,
  "date_founded": ISODate("2010-11-02T11:39:00Z"),
  "faction_id": null,
  "home_station_id": 60000001,
  "member_count": 0,
  "shares": 1000,
  "tax_rate": 0.05,
  "war_eligible": false,
  "created_at": ISODate("2024-01-01T00:00:00Z"),
  "updated_at": ISODate("2024-01-01T00:00:00Z"),
  "deleted_at": null
}
```

**Indexes**:
- `corporation_id`: Unique index for fast corporation lookups
- `deleted_at`: Index for soft delete filtering

## ESI Integration

### Corporation Information Endpoint

**ESI Route**: `GET /v4/corporations/{corporation_id}/`

**Caching Strategy**:
- **Cache Location**: MongoDB `corporations` collection
- **Cache Duration**: Permanent (until manual refresh)
- **Cache Key**: `corporation_id` field
- **Update Strategy**: Upsert on ESI fetch

**Data Mapping**:
```go
ESI Field           -> MongoDB Field       -> API Output
name                -> name               -> name
ticker              -> ticker             -> ticker
description         -> description        -> description
url                 -> url                -> url
alliance_id         -> alliance_id        -> alliance_id
ceo_id              -> ceo_character_id   -> ceo_id
creator_id          -> creator_id         -> creator_id
date_founded        -> date_founded       -> date_founded
faction_id          -> faction_id         -> faction_id
home_station_id     -> home_station_id    -> home_station_id
member_count        -> member_count       -> member_count
shares              -> shares             -> shares
tax_rate            -> tax_rate           -> tax_rate
war_eligible        -> war_eligible       -> war_eligible
```

## Key Differences from Character Module

### 1. Module Structure
- **Separate Routes Module**: Corporation uses a dedicated `routes.Module` struct
- **Enhanced Organization**: Routes are encapsulated in their own module for better separation

### 2. Data Complexity
- **More Fields**: Corporations have more optional fields than characters
- **Complex Types**: Includes financial data (shares, tax_rate) and organizational data
- **Nullable Fields**: Extensive use of pointer types for optional EVE data

### 3. ESI Data Handling
- **Complex Parsing**: More sophisticated type assertion logic for numeric fields
- **Optional Field Management**: Careful handling of nullable corporation data
- **Financial Precision**: Special handling for floating-point financial data

## Error Handling Standards

### Service Layer Errors
- **Database Errors**: Propagate MongoDB errors as internal server errors
- **ESI Errors**: Handle ESI client errors with specific 404 detection
- **Not Found**: Return `nil` for database misses, handle at route level
- **Validation**: Let Huma handle input validation automatically

### Route Layer Errors
- Use Huma error helpers: `huma.Error404NotFound`, `huma.Error500InternalServerError`
- Corporation-specific error detection with `isNotFoundError()`
- Meaningful error messages for client debugging
- Structured logging for debugging

### Repository Layer Errors
- Soft delete support with `deleted_at` filtering
- Propagate MongoDB errors up the stack
- Use context for timeout and cancellation handling

## Testing Guidelines

### Unit Testing
- Test each layer independently with mocks
- Test ESI data parsing with various input formats
- Verify soft delete behavior
- Test error conditions and edge cases

### Integration Testing  
- Test complete corporation lookup flows
- Use test database for data persistence
- Mock ESI responses for predictable testing
- Verify OpenAPI schema compliance

### Performance Testing
- Test database query performance with indexes
- Verify ESI rate limiting compliance
- Test concurrent request handling
- Monitor memory usage for complex data transformations

## Common Issues and Solutions

### 1. Huma v2 Response Structure
**Issue**: Endpoints returning data in HTTP headers instead of JSON body
**Solution**: Ensure DTOs use the wrapper pattern:
```go
type CorporationInfoOutput struct {
    Body CorporationInfo `json:"body"`  // Body wrapper is required
}
```

### 2. ESI Data Type Assertions
**Issue**: ESI returns inconsistent number types (int vs float64)
**Solution**: Use fallback type assertions:
```go
if ceoID, ok := esiData["ceo_id"].(int); ok {
    corporation.CEOCharacterID = ceoID
} else if ceoIDFloat, ok := esiData["ceo_id"].(float64); ok {
    corporation.CEOCharacterID = int(ceoIDFloat)
}
```

### 3. Soft Delete Implementation
**Issue**: Deleted corporations appearing in results
**Solution**: Always include `deleted_at` filter:
```go
filter := bson.M{"corporation_id": corporationID, "deleted_at": bson.M{"$exists": false}}
```

## Future Enhancements

### 1. Real-time Updates
- WebSocket notifications for corporation changes
- Event-driven cache invalidation
- ESI webhook integration for live data

### 2. Enhanced Querying
- Search corporations by name or ticker
- Filter by alliance membership
- Range queries for member count and other metrics

### 3. Historical Data
- Track corporation changes over time
- Historical member count tracking
- Leadership change history

### 4. Alliance Integration
- Automatic alliance data fetching for corporation members
- Cross-reference alliance and corporation data
- Alliance-wide corporation statistics

## Replication Guidelines

Use this module as a template for creating other EVE entity modules:

### For Alliance Module:
1. Copy directory structure from `internal/corporation`
2. Replace `Corporation` with `Alliance` throughout
3. Update `alliance_id` validation ranges  
4. Change ESI endpoint to `/v3/alliances/{alliance_id}/`
5. Adapt data fields for alliance-specific information
6. Update collection name to `alliances`

### For System/Region Modules:
1. Follow the same pattern but adapt for static data
2. Consider different caching strategies for static game data
3. Remove ESI integration if using SDE data instead

This module demonstrates the mature patterns and practices for building robust, maintainable EVE corporate data modules in the Go-Falcon ecosystem.