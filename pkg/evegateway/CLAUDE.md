# EVE Gateway Package (pkg/evegateway)

## Overview
Complete EVE Online ESI client library with intelligent caching, rate limiting, and CCP compliance. 
Provides type-safe access to all EVE Online APIs with proper error handling and performance optimization.

## Core Features
- **CCP Compliance**: Proper User-Agent headers and rate limiting
- **Intelligent Caching**: Respects ESI expires headers and ETags
- **Modern Pagination**: Support for both legacy offset-based and new token-based pagination
- **Error Limit Tracking**: Monitors ESI error limits to prevent blocking
- **Retry Logic**: Exponential backoff with configurable retry policies  
- **Type Safety**: Structured Go types for all ESI responses

## ESI Client Categories
- **Alliance**: Alliance information, corporations, icons
- **Character**: Character data, portraits, skills, assets
- **Corporation**: Corporation information, members, structures
- **Universe**: Systems, stations, types, market data
- **Status**: Server status, player counts, maintenance
- **And many more**: Complete ESI API coverage

## Cache Management
- **Default Cache Manager**: In-memory caching with expiration
- **Cache Keys**: URL-based for consistency
- **Conditional Requests**: If-None-Match and If-Modified-Since headers
- **Cache Metadata**: Expiration tracking and hit/miss analytics

## Rate Limiting & Compliance
```go
// Required User-Agent format
"go-falcon/1.0.0 (contact@example.com) +https://github.com/org/repo"

// Error limit monitoring
type ESIErrorLimits struct {
    Remain int
    Reset  time.Time
}
```

## Usage Examples
```go
// Initialize client
client := evegateway.NewClient()

// Get server status with caching
status, err := client.GetServerStatus(ctx)

// Character information with cache support
result, err := characterClient.GetCharacterInfoWithCache(ctx, characterID)
```

## Key Interfaces
- `Client`: Main ESI client interface
- `CacheManager`: Cache storage and retrieval
- `RetryClient`: Request retry and error handling
- Individual service clients for each ESI category

## Best Practices
- Always check cache before API calls
- Monitor error limit headers
- Use appropriate timeouts (30s recommended)
- Implement proper error handling for 4xx/5xx responses
- Respect cache expiration times
- **Future Pagination**: Prepare for token-based pagination on new endpoints, treat tokens as opaque strings
- **Current Pagination**: Most endpoints return complete data or use simple offset-based pagination
- **Data Collection**: Plan for potential pagination requirements in long-running operations

## ESI Pagination Support

### Token-Based Pagination (Future Implementation)
**Note**: CCP has announced a new token-based pagination system for improved performance and consistency. This is documented here for future implementation planning:

#### Key Features
- **Token-Based Navigation**: Uses opaque `before` and `after` tokens instead of numeric offsets
- **Time-Ordered Data**: Datasets sorted by "last modified" time for consistency
- **Bidirectional Crawling**: Navigate forwards and backwards through datasets
- **Long-Term Tokens**: Tokens remain valid for hours or weeks

#### Implementation Details
```go
// Token-based pagination parameters
type PaginationParams struct {
    Before string `url:"before,omitempty"` // Get entries before this token
    After  string `url:"after,omitempty"`  // Get entries after this token
    Limit  int    `url:"limit,omitempty"`  // Number of entries per page (default varies by endpoint)
}

// Response with pagination tokens
type PaginatedResponse struct {
    Data   []interface{} `json:"data"`
    Before *string       `json:"before,omitempty"` // Token for previous page
    After  *string       `json:"after,omitempty"`  // Token for next page
}
```

#### Usage Patterns (Future Implementation)
```go
// Example implementation for future token-based endpoints:
// Initial request (get most recent data)
resp, err := client.GetCorporationProjects(ctx, corporationID, PaginationParams{Limit: 100})

// Navigate to older data using 'before' token
if resp.Before != nil {
    olderResp, err := client.GetCorporationProjects(ctx, corporationID, PaginationParams{
        Before: *resp.Before,
        Limit:  100,
    })
}

// Monitor for new data using 'after' token
if resp.After != nil {
    newerResp, err := client.GetCorporationProjects(ctx, corporationID, PaginationParams{
        After: *resp.After,
        Limit: 100,
    })
}
```

#### Best Practices
1. **Treat Tokens as Opaque**: Never attempt to validate or interpret token contents
2. **Full Dataset Scanning**: Use `before` token to crawl through entire dataset once
3. **Change Monitoring**: Use `after` token to monitor for new/updated entries
4. **Handle Duplicates**: Expect potential duplicate records due to concurrent modifications
5. **Empty Results**: Empty response indicates reaching dataset boundary
6. **Token Persistence**: Store tokens for resuming pagination sessions later

#### Migration Strategy (When Implemented)  
- **New Endpoints**: Corporation Projects and future endpoints will use token-based pagination
- **Legacy Endpoints**: Existing routes will continue using offset-based pagination
- **Gradual Rollout**: CCP plans to migrate endpoints progressively over time

### Current Offset-Based Pagination  
Traditional pagination currently used by existing endpoints:

```go
// Legacy pagination parameters
type LegacyPaginationParams struct {
    Page int `url:"page,omitempty"` // Page number (1-based)
}

// Usage example
members, err := client.Corporation.GetCorporationMembers(ctx, corporationID, token)
// Most endpoints return all data in single response or use simple page parameter
```

### Endpoint-Specific Pagination Status
Current pagination support across ESI endpoints:
- **Corporation Projects**: Will use token-based pagination (future)
- **Corporation Members**: Single response (no pagination)
- **Market Orders**: Offset-based pagination (current)
- **Character Assets**: Single response with potential foldering

## Performance
- **Cache Hit**: <1ms response time
- **Cache Miss**: Network latency + ESI response time
- **Memory Usage**: Configurable cache size limits
- **Connection Pooling**: Efficient HTTP client reuse
- **Pagination Efficiency**: Token-based pagination reduces server load and improves consistency