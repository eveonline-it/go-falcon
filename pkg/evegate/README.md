# EVE Gate - EVE Online ESI Client

A modular ESI (Electronic System Interface) client for EVE Online, organized by API categories.

## Structure

The evegate package is organized into subdirectories based on ESI API tags:

```
pkg/evegate/
├── client.go           # Main client with unified interface
├── interfaces.go       # Common interfaces and cache manager
├── retry.go           # Retry logic with exponential backoff
├── openapi.json       # ESI OpenAPI 3.1.1 specification reference
├── alliance/          # Alliance-related endpoints ✅
├── assets/            # Asset-related endpoints
├── calendar/          # Calendar-related endpoints
├── character/         # Character-related endpoints ✅
├── clones/            # Clone-related endpoints
├── contacts/          # Contact-related endpoints
├── contracts/         # Contract-related endpoints
├── corporation/       # Corporation-related endpoints
├── dogma/             # Dogma-related endpoints
├── faction_warfare/   # Faction warfare endpoints
├── fittings/          # Fitting-related endpoints
├── fleets/            # Fleet-related endpoints
├── incursions/        # Incursion-related endpoints
├── industry/          # Industry-related endpoints
├── insurance/         # Insurance-related endpoints
├── killmails/         # Killmail-related endpoints
├── location/          # Location-related endpoints
├── loyalty/           # Loyalty point endpoints
├── mail/              # Mail-related endpoints
├── market/            # Market-related endpoints
├── planetary_interaction/ # PI-related endpoints
├── routes/            # Route-related endpoints
├── search/            # Search-related endpoints
├── skills/            # Skill-related endpoints
├── sovereignty/       # Sovereignty endpoints
├── status/            # Server status endpoints ✅
├── universe/          # Universe data endpoints ✅
├── user_interface/    # UI-related endpoints
├── wallet/            # Wallet-related endpoints
└── wars/              # War-related endpoints
```

## Features

### ESI Best Practices Implementation ✅

- **User-Agent Compliance**: Proper User-Agent headers with contact information
- **HTTP Caching**: ETag and Last-Modified header support with conditional requests
- **Error Limit Monitoring**: Tracks ESI error limit headers and implements backoff
- **Exponential Backoff**: Smart retry logic for different error types:
  - 420 (ESI rate limit): Up to 10 minutes
  - 429 (Too Many Requests): Up to 60 seconds  
  - 5xx (Server errors): Up to 30 seconds
- **Thread-Safe Caching**: In-memory cache with proper mutex protection

### Modular Architecture

Each category has its own package with:
- Dedicated client interface
- Structured response types
- Category-specific tracing
- Consistent error handling

## Usage

### Main Client (Unified Interface)

```go
import "go-falcon/pkg/evegate"

client := evegate.NewClient()

// Access category clients
status, err := client.Status.GetServerStatus(ctx)
characterInfo, err := client.Character.GetCharacterInfo(ctx, characterID)
systemInfo, err := client.Universe.GetSystemInfo(ctx, systemID)
alliances, err := client.Alliance.GetAlliances(ctx)
allianceInfo, err := client.Alliance.GetAllianceInfo(ctx, allianceID)
```

### Direct Category Usage

```go
import (
    "go-falcon/pkg/evegate"
    "go-falcon/pkg/evegate/alliance"
    "go-falcon/pkg/evegate/character"
    "go-falcon/pkg/evegate/status"
    "go-falcon/pkg/evegate/universe"
)

// Create shared infrastructure
cacheManager := evegate.NewDefaultCacheManager()
httpClient := &http.Client{Timeout: 30 * time.Second}
retryClient := evegate.NewDefaultRetryClient(httpClient, errorLimits, limitsMutex)

// Create category-specific clients
statusClient := status.NewStatusClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
characterClient := character.NewCharacterClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
universeClient := universe.NewUniverseClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
allianceClient := alliance.NewAllianceClient(httpClient, baseURL, userAgent, cacheManager, retryClient)

// Use clients directly
serverStatus, err := statusClient.GetServerStatus(ctx)
characterInfo, err := characterClient.GetCharacterInfo(ctx, 123456)
systemInfo, err := universeClient.GetSystemInfo(ctx, 30000142)
alliances, err := allianceClient.GetAlliances(ctx)
allianceInfo, err := allianceClient.GetAllianceInfo(ctx, 99005065)
```

## Configuration

### Environment Variables

- `ESI_USER_AGENT`: ESI-compliant User-Agent header (default: "go-falcon/1.0.0 contact@example.com")
- `ENABLE_TELEMETRY`: Enable OpenTelemetry tracing (default: true)

### OpenTelemetry Integration

Full tracing support with:
- Span creation for each ESI call
- Cache hit/miss tracking
- Error recording and status codes
- Request/response metadata

## Implemented Endpoints

### Status Package ✅
- `GET /status` - EVE Online server status

### Character Package ✅  
- `GET /characters/{character_id}/` - Character public information
- `GET /characters/{character_id}/portrait/` - Character portrait URLs

### Universe Package ✅
- `GET /universe/systems/{system_id}/` - Solar system information
- `GET /universe/stations/{station_id}/` - Station information

### Alliance Package ✅
- `GET /alliances` - List all active player alliances
- `GET /alliances/{alliance_id}` - Alliance public information
- `GET /alliances/{alliance_id}/contacts` - Alliance contacts (requires authentication)
- `GET /alliances/{alliance_id}/contacts/labels` - Alliance contact labels (requires authentication)
- `GET /alliances/{alliance_id}/corporations` - Alliance member corporations
- `GET /alliances/{alliance_id}/icons` - Alliance icon URLs

## ESI OpenAPI Reference

The `openapi.json` file contains the complete ESI OpenAPI 3.1.1 specification, which serves as the authoritative reference for:

- **Endpoint URLs and HTTP methods**
- **Request/response schemas** 
- **Authentication requirements**
- **Query parameters and headers**
- **Response codes and error handling**

Use this specification to implement new endpoints by:

1. Finding the desired endpoint in the `paths` section
2. Checking the `tags` to determine the correct subdirectory
3. Reviewing the `responses` schema for struct definitions
4. Implementing following the patterns from existing categories

## Next Steps

Additional ESI endpoints can be implemented by:

1. **Reference the OpenAPI spec**: Find the endpoint in `openapi.json`
2. **Create the package**: Add to the appropriate subdirectory (e.g., `market/`, `corporation/`)
3. **Define response types**: Based on the OpenAPI schema definitions
4. **Implement the client**: With proper caching and retry logic following existing patterns
5. **Add tracing and logging**: Include comprehensive observability
6. **Follow ESI best practices**: Ensure responsible API usage with proper headers and backoff

### Example Implementation Process

```bash
# 1. Find market endpoints in openapi.json under "Market" tag
# 2. Implement in market/market.go:

type MarketClient struct {
    // ... standard fields
}

type MarketOrdersResponse struct {
    // ... based on OpenAPI schema
}

func (c *MarketClient) GetMarketOrders(ctx context.Context, regionID int) (*MarketOrdersResponse, error) {
    // ... follow established patterns
}
```

All implementations should follow ESI best practices for responsible API usage and match the OpenAPI specification exactly.