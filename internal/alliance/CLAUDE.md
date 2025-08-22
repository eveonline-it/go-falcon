# Alliance Module

## Overview

The Alliance module provides comprehensive alliance information management functionality for EVE Online alliances. It offers complete API coverage for alliance discovery, detailed information retrieval, and member corporation management. It implements the standard Go-Falcon module pattern with database-first lookup and ESI fallback for data retrieval, following the same architecture as the Character and Corporation modules.

**CRITICAL REQUIREMENT**: This module MUST strictly follow the official EVE Online ESI OpenAPI specification at https://esi.evetech.net/meta/openapi.json for all data structures, field names, and API interactions.

## Module Architecture

This module follows the **unified module architecture pattern** used throughout Go-Falcon, providing a template implementation for EVE alliance data management.

### Directory Structure

```
internal/alliance/
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
- **Primary Source**: MongoDB database lookup for cached alliance data
- **Fallback Source**: EVE ESI API when alliance not found in database
- **Auto-Caching**: ESI responses automatically saved to database for future requests
- **Upsert Strategy**: Alliances are inserted or updated using MongoDB upsert operations

### 2. Type-Safe API Design
- **Huma v2 Integration**: Full type safety with compile-time validation
- **Input Validation**: Path parameters validated with alliance ID constraints (99000000-2147483647)
- **Output Structure**: Consistent response format with proper JSON serialization
- **Error Handling**: Standard HTTP error responses with meaningful messages

### 3. Complete Alliance API Coverage
- **List All Alliances**: Get all 3,476+ active alliance IDs from EVE Online
- **Alliance Information**: Detailed alliance data including name, ticker, creation info, faction warfare status
- **Member Corporations**: List all corporation IDs that belong to a specific alliance
- **Performance Optimized**: Intelligent caching with sub-millisecond cached responses

### 4. ESI Integration Best Practices
- **ESI Specification Compliance**: MANDATORY - All implementations must follow the official EVE Online ESI OpenAPI specification at https://esi.evetech.net/meta/openapi.json
- **User-Agent Compliance**: Follows CCP ESI guidelines for API requests
- **Type-Safe Parsing**: Handles JSON unmarshaling with fallback type assertions
- **Error Propagation**: Proper error handling from ESI client to HTTP responses

## ESI Alliance Data Specification

Based on the official ESI OpenAPI specification (https://esi.evetech.net/meta/openapi.json), the alliance information endpoint returns:

### Required Fields
- `name` (string): Full name of the alliance
- `creator_id` (integer): ID of the character that created the alliance
- `creator_corporation_id` (integer): ID of the corporation that created the alliance
- `ticker` (string): Short name of the alliance
- `date_founded` (string, date-time): Date the alliance was founded

### Optional Fields
- `executor_corporation_id` (integer): ID of the executor corporation
- `faction_id` (integer): Faction ID the alliance is fighting for in factional warfare

### Example ESI Response
```json
{
  "name": "C C P Alliance",
  "creator_id": 12345,
  "creator_corporation_id": 45678,
  "ticker": "<C C P>",
  "date_founded": "2016-06-26T21:00:00Z",
  "executor_corporation_id": 98356193,
  "faction_id": null
}
```

## API Endpoints

### GET `/` - List All Alliances

**Description**: Retrieves a list of all active alliance IDs from EVE Online ESI API.

**Parameters**: None

**Response**: Array of alliance IDs (`[]int64`)

**Example Response**:
```json
[99000001, 99000002, 99000003, ...]  // 3,476 alliance IDs
```

**Implementation Flow**:
1. Fetch all active alliance IDs from EVE ESI (`/v1/alliances/`)
2. Return array of alliance IDs according to ESI specification
3. Response is cached by EVE Gateway for performance

**Performance**: 
- First request: ~200ms (ESI fetch + caching)
- Cached requests: <1ms (from EVE Gateway cache)

### GET `/{alliance_id}` - Get Alliance Information

**Description**: Retrieves detailed alliance information from database or ESI if not cached.

**Parameters**:
- `alliance_id` (path, required): EVE Online alliance ID (99000000-2147483647)

**Response**: Complete alliance information with EVE game data

**Example Response**:
```json
{
  "$schema": "http://localhost:3000/schemas/AllianceInfo.json",
  "name": "United Caldari Space Command.",
  "creator_id": 1458891505,
  "creator_corporation_id": 98639548,
  "ticker": "UCSC-",
  "date_founded": "2022-04-08T16:47:41Z",
  "executor_corporation_id": 98639548,
  "faction_id": 500001
}
```

**Implementation Flow**:
1. Validate alliance ID format and range
2. Query MongoDB for cached alliance data
3. If found, return cached data
4. If not found, fetch from EVE ESI API (`/v3/alliances/{alliance_id}/`)
5. Parse ESI response according to official specification
6. Save to database and return alliance information

**Error Handling**:
- `400`: Invalid alliance ID format or range
- `404`: Alliance not found in ESI
- `500`: Database or ESI communication errors

### GET `/{alliance_id}/corporations` - List Alliance Member Corporations

**Description**: Retrieves a list of corporation IDs that are members of the specified alliance.

**Parameters**:
- `alliance_id` (path, required): EVE Online alliance ID (99000000-2147483647)

**Response**: Array of corporation IDs that belong to the alliance (`[]int64`)

**Example Response**:
```json
[98052179, 98435559, 98613992, 98701142, 98717325, 98745996, 98785732, 1975749457]
```

**Implementation Flow**:
1. Validate alliance ID format and range
2. Fetch member corporations from EVE ESI API (`/v1/alliances/{alliance_id}/corporations/`)
3. Return array of corporation IDs according to ESI specification
4. Response is cached by EVE Gateway for performance

**Performance Examples**:
- Triumvirate Alliance (933731581): 8 corporations
  - First request: ~214ms (ESI fetch + caching)  
  - Cached requests: ~336µs (from EVE Gateway cache)
- UCSC Alliance (99011489): 3 corporations (~65ms)
- Non-existent Alliance: Empty array `[]` (~135ms)

**Error Handling**:
- `400`: Invalid alliance ID format or range
- `404`: Alliance not found in ESI (Note: ESI returns empty array for non-existent alliances)
- `500`: Database or ESI communication errors

### GET `/status` - Alliance Module Status

**Description**: Returns the health status of the alliance module.

**Response**: Module name, health status, and optional error message

**Example Response**:
```json
{
  "module": "alliance",
  "status": "healthy",
  "message": ""
}
```

## Database Schema

### Alliances Collection

```json
{
  "_id": ObjectId("..."),
  "alliance_id": 933731581,
  "name": "Triumvirate.",
  "ticker": "TRI",
  "date_founded": ISODate("2006-07-14T18:53:00Z"),
  "creator_corporation_id": 933677080,
  "creator_character_id": 1200648025,
  "executor_corporation_id": 98435559,
  "faction_id": null,
  "created_at": ISODate("2024-01-01T00:00:00Z"),
  "updated_at": ISODate("2024-01-01T00:00:00Z"),
  "deleted_at": null
}
```

**Indexes**:
- `alliance_id`: Unique index for fast alliance lookups
- `deleted_at`: Index for soft delete filtering

## ESI Integration

### Alliance Information Endpoint

**ESI Route**: `GET /v3/alliances/{alliance_id}/`
**ESI Specification**: Must follow https://esi.evetech.net/meta/openapi.json

**Caching Strategy**:
- **Cache Location**: MongoDB `alliances` collection
- **Cache Duration**: Permanent (until manual refresh)
- **Cache Key**: `alliance_id` field
- **Update Strategy**: Upsert on ESI fetch

**Data Mapping (ESI → MongoDB → API Output)**:
```go
ESI Field               -> MongoDB Field          -> API Output
name                    -> name                  -> name
ticker                  -> ticker                -> ticker
creator_id              -> creator_character_id  -> creator_character_id
creator_corporation_id  -> creator_corporation_id -> creator_corporation_id
date_founded            -> date_founded          -> date_founded
executor_corporation_id -> executor_corporation_id -> executor_corporation_id
faction_id              -> faction_id            -> faction_id
```

## Implementation Details

### Data Type Handling
JSON responses from ESI typically unmarshal numeric values as `float64` in Go when using `map[string]any`. The service layer handles this correctly:

```go
// Handle ESI numeric fields (typically float64 from JSON unmarshaling)
if creatorCorpIDFloat, ok := esiData["creator_corporation_id"].(float64); ok {
    alliance.CreatorCorporationID = int(creatorCorpIDFloat)
} else if creatorCorpID, ok := esiData["creator_corporation_id"].(int); ok {
    alliance.CreatorCorporationID = creatorCorpID
}
```

### Field Name Mapping
Critical: ESI returns `creator_id` but our model uses `creator_character_id`:
```go
// Correct field mapping from ESI specification
if creatorCharIDFloat, ok := esiData["creator_id"].(float64); ok {
    alliance.CreatorCharacterID = int(creatorCharIDFloat)
}
```

## Error Handling Standards

### Service Layer Errors
- **Database Errors**: Propagate MongoDB errors as internal server errors
- **ESI Errors**: Handle ESI client errors with specific 404 detection
- **Not Found**: Return `nil` for database misses, handle at route level
- **Validation**: Let Huma handle input validation automatically

### Route Layer Errors
- Use Huma error helpers: `huma.Error404NotFound`, `huma.Error500InternalServerError`
- Alliance-specific error detection with `isNotFoundError()`
- Meaningful error messages for client debugging
- Structured logging for debugging

### Repository Layer Errors
- Soft delete support with `deleted_at` filtering
- Propagate MongoDB errors up the stack
- Use context for timeout and cancellation handling

## Common Issues and Solutions

### 1. ESI Field Name Mismatches
**Issue**: ESI specification uses different field names than expected
**Solution**: Always verify field names against https://esi.evetech.net/meta/openapi.json
**Example**: ESI uses `creator_id`, not `creator_character_id`

### 2. JSON Number Type Handling
**Issue**: JSON numbers become `float64` in Go when unmarshaling to `map[string]any`
**Solution**: Check `float64` type first, then fallback to `int`

### 3. Date Parsing
**Issue**: ESI date format must match RFC3339
**Solution**: Use `time.Parse(time.RFC3339, dateString)` for date_founded field

### 4. Huma v2 Response Structure
**Issue**: Endpoints returning data in HTTP headers instead of JSON body
**Solution**: Ensure DTOs use the wrapper pattern:
```go
type AllianceInfoOutput struct {
    Body AllianceInfo `json:"body"`  // Body wrapper is required
}
```

## Testing Guidelines

### Unit Testing
- Test each layer independently with mocks
- Test ESI data parsing with actual ESI response formats
- Verify field mapping according to ESI specification
- Test error conditions and edge cases

### Integration Testing  
- Test complete alliance lookup flows
- Use test database for data persistence
- Mock ESI responses using real ESI specification data
- Verify OpenAPI schema compliance

### ESI Compliance Testing
- Verify all field names match https://esi.evetech.net/meta/openapi.json
- Test with real ESI responses for data accuracy
- Validate data type handling for all numeric fields
- Ensure date parsing works with ESI date format

## Replication Guidelines

Use this module as a template for creating other EVE entity modules:

### Key Steps:
1. **Verify ESI Specification**: Always check https://esi.evetech.net/meta/openapi.json for correct field names and types
2. **Copy directory structure** from `internal/alliance`
3. **Update package names** throughout all files
4. **Modify DTO structures** for target data type according to ESI spec
5. **Update validation rules** for ID ranges and formats
6. **Change ESI endpoints** in service layer
7. **Update MongoDB collection names** in models
8. **Adapt data mapping** between ESI and database fields per specification
9. **Update route paths and descriptions**

### For Corporation Module Enhancement:
Check that field mappings match ESI specification for corporations

### For Character Module Enhancement:
Verify character field mappings against ESI specification

This module demonstrates ESI-compliant patterns and practices for building robust, specification-accurate EVE alliance data modules in the Go-Falcon ecosystem.