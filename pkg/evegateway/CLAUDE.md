# EVE Gateway Package (pkg/evegateway)

## Overview
Complete EVE Online ESI (Electronic System Interface) client library with intelligent caching, rate limiting, and CCP compliance. Provides type-safe access to all EVE Online APIs with proper error handling and performance optimization.

## Core Features
- **CCP Compliance**: Proper User-Agent headers and rate limiting
- **Intelligent Caching**: Respects ESI expires headers and ETags
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

## Performance
- **Cache Hit**: <1ms response time
- **Cache Miss**: Network latency + ESI response time
- **Memory Usage**: Configurable cache size limits
- **Connection Pooling**: Efficient HTTP client reuse