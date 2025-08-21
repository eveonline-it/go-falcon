# Character Module

## Overview

The Character module provides character profile management functionality for EVE Online characters. It implements the standard Go-Falcon module pattern with database-first lookup and ESI fallback for data retrieval.

## Module Architecture

This module follows the **unified module architecture pattern** used throughout Go-Falcon. It demonstrates the standard structure that should be replicated for other similar modules.

### Directory Structure

```
internal/character/
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
- **Primary Source**: MongoDB database lookup for cached character data
- **Fallback Source**: EVE ESI API when character not found in database
- **Auto-Caching**: ESI responses automatically saved to database for future requests
- **Upsert Strategy**: Characters are inserted or updated using MongoDB upsert operations

### 2. Type-Safe API Design
- **Huma v2 Integration**: Full type safety with compile-time validation
- **Input Validation**: Path parameters validated with EVE character ID constraints
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
    service    *services.Service
    eveGateway *evegateway.Client
}

func New(mongodb *database.MongoDB, redis *database.Redis, eveGateway *evegateway.Client) *Module {
    service := services.NewService(mongodb, eveGateway)
    
    return &Module{
        BaseModule: module.NewBaseModule("character", mongodb, redis),
        service:    service,
        eveGateway: eveGateway,
    }
}

func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
    routes.RegisterCharacterRoutes(api, basePath, m.service)
}
```

**Key Patterns:**
- Embeds `module.BaseModule` for standard functionality
- Dependency injection via constructor
- Service layer abstraction
- Unified route registration for API gateway integration

### 2. DTO Layer (`dto/`)

**Input DTOs** (`inputs.go`):
```go
type GetCharacterProfileInput struct {
    CharacterID int `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
}
```

**Output DTOs** (`outputs.go`):
```go
type CharacterProfile struct {
    CharacterID     int       `json:"character_id" doc:"EVE Online character ID"`
    Name            string    `json:"name" doc:"Character name"`
    // ... additional fields
}

type CharacterProfileOutput struct {
    Body CharacterProfile `json:"body"`
}
```

**Key Patterns:**
- Separate input/output structures
- Huma v2 validation tags (`validate`, `minimum`, `maximum`)
- OpenAPI documentation tags (`doc`)
- Wrapper output structures for consistent response format

### 3. Route Registration (`routes/routes.go`)

```go
func RegisterCharacterRoutes(api huma.API, basePath string, service *services.Service) {
    huma.Register(api, huma.Operation{
        OperationID: "character-get-profile",
        Method:      "GET", 
        Path:        basePath + "/{character_id}",
        Summary:     "Get character profile",
        Description: "Get character profile from database or fetch from EVE ESI if not found",
        Tags:        []string{"Character"},
    }, func(ctx context.Context, input *dto.GetCharacterProfileInput) (*dto.CharacterProfileOutput, error) {
        return service.GetCharacterProfile(ctx, input.CharacterID)
    })
}
```

**Key Patterns:**
- Function-based route registration (not methods)
- Complete OpenAPI metadata (OperationID, Summary, Description, Tags)
- Direct service layer delegation
- Context-aware handlers
- Standard HTTP error handling via Huma

### 4. Service Layer (`services/service.go`)

```go
type Service struct {
    repository *Repository
    eveGateway *evegateway.Client
}

func (s *Service) GetCharacterProfile(ctx context.Context, characterID int) (*dto.CharacterProfileOutput, error) {
    // 1. Try database first
    character, err := s.repository.GetCharacterByID(ctx, characterID)
    if character != nil {
        profile := s.characterToProfile(character)
        return &dto.CharacterProfileOutput{Body: *profile}, nil
    }
    
    // 2. Fallback to ESI
    esiData, err := s.eveGateway.GetCharacterInfo(ctx, characterID)
    if err != nil {
        return nil, err
    }
    
    // 3. Parse and save to database
    character = s.parseESIData(esiData, characterID)
    if err := s.repository.SaveCharacter(ctx, character); err != nil {
        return nil, err
    }
    
    // 4. Return response
    profile := s.characterToProfile(character)
    return &dto.CharacterProfileOutput{Body: *profile}, nil
}
```

**Key Patterns:**
- Database-first strategy with ESI fallback
- Proper error propagation
- Conversion between models and DTOs
- Auto-caching of ESI responses
- Context propagation throughout the call chain

### 5. Repository Layer (`services/repository.go`)

```go
type Repository struct {
    mongodb    *database.MongoDB
    collection *mongo.Collection
}

func (r *Repository) GetCharacterByID(ctx context.Context, characterID int) (*models.Character, error) {
    filter := bson.M{"character_id": characterID}
    
    var character models.Character
    err := r.collection.FindOne(ctx, filter).Decode(&character)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, nil  // Not found, not an error
        }
        return nil, err
    }
    
    return &character, nil
}

func (r *Repository) SaveCharacter(ctx context.Context, character *models.Character) error {
    character.UpdatedAt = time.Now()
    if character.CreatedAt.IsZero() {
        character.CreatedAt = time.Now()
    }

    filter := bson.M{"character_id": character.CharacterID}
    update := bson.M{"$set": character}
    opts := options.Update().SetUpsert(true)
    
    _, err := r.collection.UpdateOne(ctx, filter, update, opts)
    return err
}
```

**Key Patterns:**
- Collection-based MongoDB operations
- Proper not-found handling (`mongo.ErrNoDocuments`)
- Upsert operations for insert/update behavior
- Automatic timestamp management
- Context-aware database operations

### 6. Database Models (`models/models.go`)

```go
type Character struct {
    ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    CharacterID    int                `bson:"character_id" json:"character_id"`
    Name           string             `bson:"name" json:"name"`
    // ... additional fields
    CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

func (c *Character) CollectionName() string {
    return "characters"
}
```

**Key Patterns:**
- MongoDB `primitive.ObjectID` for document ID
- Separate BSON and JSON tags for database/API serialization
- Business identifier fields (e.g., `character_id`) separate from MongoDB `_id`
- Standard timestamp fields (`created_at`, `updated_at`)
- Collection name method for dynamic collection access

## API Endpoints

### GET `/{character_id}` - Get Character Profile

**Description**: Retrieves character profile from database or ESI if not cached.

**Parameters**:
- `character_id` (path, required): EVE Online character ID (90000000-2147483647)

**Response**: Character profile with EVE game data

**Implementation Flow**:
1. Validate character ID format and range
2. Query MongoDB for cached character data
3. If found, return cached data
4. If not found, fetch from EVE ESI API
5. Parse ESI response and save to database
6. Return character profile

**Error Handling**:
- `400`: Invalid character ID format or range
- `404`: Character not found in ESI
- `500`: Database or ESI communication errors

## Database Schema

### Characters Collection

```json
{
  "_id": ObjectId("..."),
  "character_id": 90000001,
  "name": "Character Name",
  "corporation_id": 98000001,
  "alliance_id": 99000001,
  "birthday": ISODate("2003-05-06T00:00:00Z"),
  "security_status": 0.5,
  "description": "Character description",
  "gender": "Male",
  "race_id": 1,
  "bloodline_id": 4,
  "ancestry_id": 12,
  "faction_id": 500001,
  "created_at": ISODate("2024-01-01T00:00:00Z"),
  "updated_at": ISODate("2024-01-01T00:00:00Z")
}
```

**Indexes**:
- `character_id`: Unique index for fast character lookups

## ESI Integration

### Character Information Endpoint

**ESI Route**: `GET /v4/characters/{character_id}/`

**Caching Strategy**:
- **Cache Location**: MongoDB `characters` collection
- **Cache Duration**: Permanent (until manual refresh)
- **Cache Key**: `character_id` field
- **Update Strategy**: Upsert on ESI fetch

**Data Mapping**:
```go
ESI Field           -> MongoDB Field       -> API Output
name                -> name               -> name
corporation_id      -> corporation_id     -> corporation_id
alliance_id         -> alliance_id        -> alliance_id
birthday            -> birthday           -> birthday
security_status     -> security_status    -> security_status
description         -> description        -> description
gender              -> gender             -> gender
race_id             -> race_id            -> race_id
bloodline_id        -> bloodline_id       -> bloodline_id
ancestry_id         -> ancestry_id        -> ancestry_id
faction_id          -> faction_id         -> faction_id
```

## Replication Guidelines

Use this module as a template for creating similar EVE data modules:

### 1. Corporation Module
- Replace `Character` with `Corporation`
- Use `corporation_id` instead of `character_id`
- Adapt ESI endpoint to `/v4/corporations/{corporation_id}/`
- Update validation ranges for corporation IDs

### 2. Alliance Module
- Replace `Character` with `Alliance` 
- Use `alliance_id` instead of `character_id`
- Adapt ESI endpoint to `/v3/alliances/{alliance_id}/`
- Update validation ranges for alliance IDs

### 3. General Pattern
1. **Copy directory structure** from `internal/character`
2. **Update package names** throughout all files
3. **Modify DTO structures** for target data type
4. **Update validation rules** for ID ranges and formats
5. **Change ESI endpoints** in service layer
6. **Update MongoDB collection names** in models
7. **Adapt data mapping** between ESI and database fields
8. **Update route paths and descriptions**

### Key Files to Modify:
- `module.go`: Update module name and constructor
- `dto/inputs.go`: Change ID field names and validation
- `dto/outputs.go`: Update structure fields for target data
- `routes/routes.go`: Update paths, operation IDs, and descriptions
- `services/service.go`: Update ESI client calls and data parsing
- `services/repository.go`: Update collection name and filter fields
- `models/models.go`: Update struct fields and collection name
- `CLAUDE.md`: Update documentation for the new module

## Error Handling Standards

### Service Layer Errors
- **Database Errors**: Propagate MongoDB errors as internal server errors
- **ESI Errors**: Handle ESI client errors and convert to appropriate HTTP status
- **Not Found**: Return `nil` for database misses, handle at route level
- **Validation**: Let Huma handle input validation automatically

### Route Layer Errors
- Use Huma error helpers: `huma.Error404NotFound`, `huma.Error500InternalServerError`
- Provide meaningful error messages for client debugging
- Log errors with sufficient context for debugging

### Repository Layer Errors
- Return `nil` for `mongo.ErrNoDocuments` (not found is not an error)
- Propagate other MongoDB errors up the stack
- Use context for timeout and cancellation handling

## Testing Guidelines

### Unit Testing
- Test each layer independently
- Mock dependencies (database, ESI client)
- Test error conditions and edge cases
- Verify data transformations between layers

### Integration Testing  
- Test complete request flows
- Use test database for data persistence
- Mock ESI responses for predictable testing
- Verify OpenAPI schema compliance

### Performance Testing
- Test database query performance with indexes
- Verify ESI rate limiting compliance
- Test concurrent request handling
- Monitor memory usage for data transformations

## Future Enhancements

### 1. Cache TTL Management
- Add cache expiration times for character data
- Implement refresh triggers for stale data
- Add cache invalidation endpoints

### 2. Bulk Operations
- Batch character lookups for efficiency
- Bulk ESI requests where supported
- Optimized database bulk operations

### 3. Real-time Updates
- WebSocket notifications for character updates
- Event-driven cache invalidation
- ESI webhook integration for live data

### 4. Advanced Querying
- Search characters by name or corporation
- Range queries for character statistics
- Aggregation endpoints for analytics

This module demonstrates the standard patterns and practices for building robust, maintainable EVE data modules in the Go-Falcon ecosystem.