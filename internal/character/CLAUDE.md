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
│   ├── outputs.go         # Response output DTOs with proper JSON structure
│   └── affiliation.go     # Affiliation-specific DTOs for background updates
├── models/                # Database models
│   └── models.go         # MongoDB schemas and collection definitions
├── routes/               # Route definitions  
│   └── routes.go         # Huma v2 unified route registration
├── services/             # Business logic layer
│   ├── repository.go     # Database operations and queries
│   ├── service.go        # Business logic and ESI integration
│   └── update_service.go # Background affiliation update service with parallel processing
├── module.go             # Module initialization and interface implementation
└── CLAUDE.md             # This documentation file

**Note**: Authentication and permission middleware now centralized in `pkg/middleware/` system.
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
- **Character Affiliation Updates**: Automated background updates via ESI affiliation endpoint
- **Batch Processing**: Efficiently handles large-scale character updates with parallel workers

### 4. Background Affiliation Updates
- **Scheduler Integration**: Automated updates every 30 minutes via system task
- **Batch Processing**: Processes up to 1000 character IDs per ESI request
- **Parallel Workers**: 3 concurrent ESI requests for optimal performance
- **Debug Logging**: Detailed console output for tracking character changes
- **Error Recovery**: Retry logic with graceful failure handling

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

### GET `/search` - Search Characters by Name

**Description**: Searches characters by name using optimized database strategies.

**Parameters**:
- `name` (query, required): Character name (minimum 3 characters)

**Response**: List of matching characters with relevance scoring

**Search Strategies**:
- **Multi-word queries**: Full-text search with MongoDB text indexes
- **Single-word queries**: Case-insensitive regex with prefix optimization
- **Short queries**: Prefix-only search for performance

**Performance Features**:
- Text search indexes for multi-word queries
- Case-insensitive collation indexes
- Result limits (20-50) for optimal response times
- Relevance scoring for text search results

## Background Services

### Affiliation Update Service

**Description**: Automated background service that updates character corporation and alliance affiliations using ESI data.

**Schedule**: Every 30 minutes via scheduler system task (`system-character-affiliation-update`)

**Processing Flow**:
1. **Character Discovery**: Retrieve all character IDs from MongoDB
2. **Batch Creation**: Split characters into batches of 1000 (ESI limit)
3. **Parallel Processing**: Process 3 batches concurrently for optimal performance
4. **ESI Integration**: Call `/characters/affiliation/` endpoint for each batch
5. **Data Parsing**: Convert ESI response with proper type handling
6. **Database Updates**: Bulk update character affiliations with upsert logic
7. **Statistics Reporting**: Track updated, failed, and skipped characters

**Key Features**:
- **Parallel Workers**: 3 concurrent ESI requests to maximize throughput
- **Batch Optimization**: 1000 characters per ESI request (maximum allowed)
- **Debug Logging**: Console output showing character changes:
  ```
  🔄 Character 90000001 affiliation UPDATED: corp: 98000001→98000002, alliance: 0→99000001
  📊 Character 90000002 affiliation checked (no changes)
  ➕ Character 90000003 NOT FOUND in database, creating new record
  ```
- **Error Recovery**: Individual character failures don't stop batch processing
- **Performance Monitoring**: Execution time and batch statistics logging
- **Type Safety**: Handles ESI JSON number parsing (float64 → int conversion)

**Service Methods**:
```go
// Update all characters in database
UpdateAllAffiliations(ctx context.Context) (*dto.AffiliationUpdateStats, error)

// Update specific character list
UpdateCharacterAffiliations(ctx context.Context, characterIDs []int) (*dto.AffiliationUpdateStats, error)

// Get single character affiliation (DB-first with ESI fallback)
GetCharacterAffiliation(ctx context.Context, characterID int) (*dto.CharacterAffiliation, error)

// Force ESI refresh for single character
RefreshCharacterAffiliation(ctx context.Context, characterID int) (*dto.CharacterAffiliation, error)

// Validate character existence via ESI
ValidateCharacters(ctx context.Context, characterIDs []int) ([]int, []int, error)
```

**ESI Integration**:
- **Endpoint**: `POST /characters/affiliation/` (EVE ESI v1)
- **Rate Limit**: Respects ESI error limits and caching headers
- **Batch Size**: Maximum 1000 character IDs per request
- **Cache Duration**: 3600 seconds (1 hour) as per ESI specification
- **Error Handling**: Exponential backoff for transient failures

**Database Operations**:
- **Bulk Updates**: MongoDB bulk write operations for efficiency
- **Upsert Logic**: Creates new records for unknown characters
- **Change Detection**: Compares existing vs new data for logging
- **Index Optimization**: Uses `character_id` unique index for fast lookups
- **Timestamp Management**: Automatic `updated_at` field maintenance

**Performance Metrics**:
- **Throughput**: ~3000 characters per minute (with 3 parallel workers)
- **Database Impact**: Bulk operations minimize connection overhead
- **Memory Usage**: Streaming cursor processing for large datasets
- **Error Resilience**: Individual failures don't affect other characters

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
- `name_text`: Full-text search index for multi-word name queries
- `name_regular`: Case-insensitive index for prefix and regex searches
- `name_created_compound`: Compound index optimizing name search with timestamp sorting

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

## Scheduler Integration

### System Task Definition

The character module integrates with the scheduler system through a predefined system task:

```go
// Task: system-character-affiliation-update
Name:        "Character Affiliation Update"
Description: "Updates character corporation and alliance affiliations from ESI"
Schedule:    "0 */30 * * * *" // Every 30 minutes
Priority:    Normal
Retries:     3 // Retry up to 3 times on failure
Interval:    5 * time.Minute // 5-minute retry interval
```

### Task Execution Flow

1. **Scheduler Trigger**: Task runs every 30 minutes automatically
2. **Service Invocation**: Calls `character.UpdateService.UpdateAllAffiliations()`
3. **Batch Processing**: Processes all characters in database with parallel workers
4. **ESI Communication**: Makes authenticated requests to EVE ESI affiliation endpoint
5. **Database Updates**: Bulk updates character records with new affiliation data
6. **Statistics Logging**: Reports execution results with detailed metrics
7. **Error Handling**: Retries up to 3 times with 5-minute intervals on failure

### Monitoring and Observability

**Success Logging**:
```
🎯 AFFILIATION UPDATE COMPLETED: 1250 updated, 3 failed, 15 skipped in 45 seconds (processed 5 batches)
```

**Error Logging**:
```
❌ ERROR fetching affiliations from ESI: rate limit exceeded
⚠️  Character 90000001 not found in ESI response, skipping
```

**Performance Metrics**:
- Execution duration tracking
- Batch processing statistics
- Character update counters
- ESI request success rates

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

## Advanced Features

### Search Optimization

The character module implements advanced search capabilities optimized for different query patterns:

**Multi-Strategy Search Engine**:
- **Full-Text Search**: MongoDB text indexes with relevance scoring for multi-word queries
- **Regex Search**: Case-insensitive pattern matching for single-word searches
- **Prefix Optimization**: Fast prefix searches for short queries
- **Result Limiting**: Automatic result caps (20-50) for optimal performance

**Database Index Strategy**:
```go
// Multiple specialized indexes for different search patterns
indexes := []mongo.IndexModel{
    characterIDIndex,        // Unique lookup
    nameTextIndex,          // Full-text search
    nameRegularIndex,       // Case-insensitive regex
    nameWithTimestampIndex, // Compound sorting
}
```

### Error Recovery and Resilience

**Affiliation Update Resilience**:
- **Individual Character Isolation**: Single character failures don't affect batch processing
- **Concurrent Update Handling**: Race condition protection with MongoDB duplicate key handling
- **ESI Error Management**: Proper handling of rate limits and temporary failures
- **Data Consistency**: Upsert operations ensure consistent data state

**Retry Mechanisms**:
- **Database Retries**: Automatic retry for concurrent insert conflicts
- **ESI Retries**: Built into EVE Gateway with exponential backoff
- **Scheduler Retries**: System task retries (3 attempts, 5-minute intervals)

### Performance Optimizations

**Database Performance**:
- **Bulk Operations**: MongoDB bulk write for affiliation updates
- **Projection Optimization**: Character ID-only queries for large dataset operations
- **Index Utilization**: Multiple specialized indexes for different access patterns
- **Connection Pooling**: Efficient database connection management

**ESI Performance**:
- **Parallel Processing**: 3 concurrent workers for maximum ESI throughput
- **Batch Optimization**: 1000-character batches matching ESI limits
- **Caching Integration**: Respects ESI cache headers and expiration times
- **Rate Limit Compliance**: Built-in ESI error limit monitoring

## Future Enhancements

### 1. Real-time Affiliation Monitoring
- WebSocket notifications for character affiliation changes
- Event-driven updates triggered by specific character activities
- Push notifications for corporation/alliance membership changes

### 2. Advanced Analytics
- Affiliation history tracking with change timestamps
- Corporation/alliance membership trend analysis
- Character movement pattern detection
- Statistical dashboards for affiliation data

### 3. Enhanced Search Features
- Fuzzy name matching with edit distance algorithms
- Search suggestions and autocomplete functionality
- Advanced filtering by corporation, alliance, or faction
- Full-text search across character descriptions

### 4. Integration Expansions
- Character skill queue monitoring
- Asset location tracking
- Activity status detection
- Corporation role change notifications

### 5. Performance Improvements
- Redis caching layer for frequently accessed characters
- Database read replicas for search operations
- Elasticsearch integration for advanced text search
- API response compression and CDN integration

This module demonstrates the standard patterns and practices for building robust, maintainable EVE data modules in the Go-Falcon ecosystem. The combination of database-first architecture, ESI integration, background processing, and comprehensive search capabilities provides a solid foundation for EVE Online data management applications.