# Corporation Client - EVE Online ESI

Complete corporation API client for EVE Online's ESI (Electronic System Interface) with intelligent caching, authentication, and OpenTelemetry tracing.

## Features

- **Complete Corporation API Coverage**: All EVE Online corporation endpoints
- **Intelligent Caching**: Respects ESI cache headers and ETags  
- **Authentication Support**: Handles both public and authenticated endpoints  
- **Future-Ready Pagination**: Prepared for CCP's upcoming token-based pagination system
- **OpenTelemetry Integration**: Comprehensive tracing and observability
- **Type Safety**: Strongly typed Go structs for all responses
- **Cache Information**: Optional cache metadata for each request

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "go-falcon/pkg/evegateway"
)

func main() {
    client := evegateway.NewClient()
    
    // Get corporation information
    corp, err := client.Corporation.GetCorporationInfo(context.Background(), 98579006)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Corporation: %s [%s]\n", corp.Name, corp.Ticker)
    fmt.Printf("Members: %d\n", corp.MemberCount)
    fmt.Printf("Tax Rate: %.1f%%\n", corp.TaxRate*100)
}
```

## Available Endpoints

### Public Information (No Authentication Required)

#### Corporation Information
```go
// Basic corporation information
corp, err := client.Corporation.GetCorporationInfo(ctx, corporationID)

// With cache information
result, err := client.Corporation.GetCorporationInfoWithCache(ctx, corporationID)
fmt.Printf("Cached: %v, Expires: %v\n", result.Cache.Cached, result.Cache.ExpiresAt)
```

#### Corporation Icons
```go
// Corporation logo URLs
icons, err := client.Corporation.GetCorporationIcons(ctx, corporationID)
fmt.Printf("Logo URLs: 64x64=%s, 128x128=%s, 256x256=%s\n", 
    icons.Px64x64, icons.Px128x128, icons.Px256x256)
```

#### Alliance History
```go
// Corporation's alliance membership history
history, err := client.Corporation.GetCorporationAllianceHistory(ctx, corporationID)
for _, entry := range history {
    fmt.Printf("Alliance %d from %v\n", entry.AllianceID, entry.StartDate)
}
```

### Authenticated Endpoints (Requires Token)

#### Corporation Members
```go
// List all corporation members (requires esi-corporations.read_corporation_membership.v1)
members, err := client.Corporation.GetCorporationMembers(ctx, corporationID, accessToken)
for _, member := range members {
    fmt.Printf("Member: %d\n", member.CharacterID)
}
```

#### Member Tracking
```go
// Detailed member tracking information (requires director roles)
tracking, err := client.Corporation.GetCorporationMemberTracking(ctx, corporationID, accessToken)
for _, member := range tracking {
    fmt.Printf("Character %d: Last login %v\n", member.CharacterID, member.LogonDate)
}
```

#### Member Roles
```go
// Corporation member roles and permissions
roles, err := client.Corporation.GetCorporationMemberRoles(ctx, corporationID, accessToken)
for _, member := range roles {
    fmt.Printf("Character %d roles: %v\n", member.CharacterID, member.Roles)
}
```

#### Corporation Structures
```go
// Corporation-owned structures (requires esi-corporations.read_structures.v1)
structures, err := client.Corporation.GetCorporationStructures(ctx, corporationID, accessToken)
for _, structure := range structures {
    fmt.Printf("Structure %d in system %d: %s\n", 
        structure.StructureID, structure.SystemID, structure.State)
}
```

#### Corporation Standings
```go
// Corporation standings with other entities
standings, err := client.Corporation.GetCorporationStandings(ctx, corporationID, accessToken)
for _, standing := range standings {
    fmt.Printf("Standing with %d (%s): %.2f\n", 
        standing.FromID, standing.FromType, standing.Standing)
}
```

#### Corporation Wallets
```go
// Corporation wallet balances (requires esi-corporations.read_wallets.v1)
wallets, err := client.Corporation.GetCorporationWallets(ctx, corporationID, accessToken)
for _, wallet := range wallets {
    fmt.Printf("Division %d: %.2f ISK\n", wallet.Division, wallet.Balance)
}
```

## Data Structures

### CorporationInfoResponse
```go
type CorporationInfoResponse struct {
    CorporationID   int       `json:"corporation_id"`
    Name            string    `json:"name"`
    Ticker          string    `json:"ticker"`
    Description     string    `json:"description"`
    URL             string    `json:"url,omitempty"`
    AllianceID      int       `json:"alliance_id,omitempty"`
    CEOCharacterID  int       `json:"ceo_id"`
    CreatorID       int       `json:"creator_id"`
    DateFounded     time.Time `json:"date_founded"`
    FactionID       int       `json:"faction_id,omitempty"`
    HomeStationID   int       `json:"home_station_id,omitempty"`
    MemberCount     int       `json:"member_count"`
    Shares          int64     `json:"shares,omitempty"`
    TaxRate         float64   `json:"tax_rate"`
    WarEligible     bool      `json:"war_eligible,omitempty"`
}
```

### CorporationStructure
```go
type CorporationStructure struct {
    StructureID         int64     `json:"structure_id"`
    TypeID              int       `json:"type_id"`
    SystemID            int       `json:"system_id"`
    ProfileID           int       `json:"profile_id"`
    FuelExpires         time.Time `json:"fuel_expires,omitempty"`
    StateTimerStart     time.Time `json:"state_timer_start,omitempty"`
    StateTimerEnd       time.Time `json:"state_timer_end,omitempty"`
    UnanchorsAt         time.Time `json:"unanchors_at,omitempty"`
    State               string    `json:"state"`
    ReinforceHour       int       `json:"reinforce_hour,omitempty"`
    ReinforceWeekday    int       `json:"reinforce_weekday,omitempty"`
    CorporationID       int       `json:"corporation_id"`
    Services            []Service `json:"services,omitempty"`
}
```

### CorporationMemberTracking
```go
type CorporationMemberTracking struct {
    BaseID              int       `json:"base_id,omitempty"`
    CharacterID         int       `json:"character_id"`
    LocationID          int64     `json:"location_id,omitempty"`
    LogoffDate          time.Time `json:"logoff_date,omitempty"`
    LogonDate           time.Time `json:"logon_date,omitempty"`
    ShipTypeID          int       `json:"ship_type_id,omitempty"`
    StartDate           time.Time `json:"start_date,omitempty"`
}
```

## Authentication & Scopes

### Required EVE Online Scopes

| Endpoint | Required Scope |
|----------|----------------|
| `GetCorporationMembers` | `esi-corporations.read_corporation_membership.v1` |
| `GetCorporationMemberTracking` | `esi-corporations.read_corporation_membership.v1` |
| `GetCorporationMemberRoles` | `esi-corporations.read_corporation_membership.v1` |
| `GetCorporationStructures` | `esi-corporations.read_structures.v1` |
| `GetCorporationStandings` | `esi-corporations.read_standings.v1` |
| `GetCorporationWallets` | `esi-corporations.read_wallets.v1` |

### Getting Access Tokens

```go
// Example: Using access token from EVE SSO authentication
func getCorporationData(corporationID int, accessToken string) {
    client := evegateway.NewClient()
    
    // This requires proper authentication
    members, err := client.Corporation.GetCorporationMembers(
        context.Background(), 
        corporationID, 
        accessToken,
    )
    if err != nil {
        // Handle authentication or permission errors
        log.Printf("Error: %v", err)
        return
    }
    
    fmt.Printf("Corporation has %d members\n", len(members))
}
```

## Caching

### Automatic Caching
All endpoints automatically cache responses based on ESI headers:
- Respects `Expires` header from ESI
- Uses `ETag` for conditional requests  
- Handles `304 Not Modified` responses
- Cache keys based on endpoint URL

### Cache Information
Use `*WithCache` methods to get cache metadata:

```go
result, err := client.Corporation.GetCorporationInfoWithCache(ctx, corporationID)
if err != nil {
    return err
}

fmt.Printf("Corporation: %s\n", result.Data.Name)
fmt.Printf("From Cache: %v\n", result.Cache.Cached)
if result.Cache.ExpiresAt != nil {
    fmt.Printf("Cache Expires: %v\n", *result.Cache.ExpiresAt)
}
```

## Error Handling

### Common Error Scenarios

```go
corp, err := client.Corporation.GetCorporationInfo(ctx, corporationID)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "status 404"):
        fmt.Println("Corporation not found")
    case strings.Contains(err.Error(), "status 403"):
        fmt.Println("Access forbidden - check token scopes")
    case strings.Contains(err.Error(), "status 420"):
        fmt.Println("Rate limited - slow down requests")
    case strings.Contains(err.Error(), "status 500"):
        fmt.Println("ESI server error - retry later")
    default:
        fmt.Printf("Unexpected error: %v\n", err)
    }
    return
}
```

### ESI Compliance
- Proper User-Agent headers included automatically
- Rate limiting respected through built-in retry logic
- Error limit tracking to prevent API access restrictions
- Cache headers strictly followed per CCP guidelines

## OpenTelemetry Integration

### Automatic Tracing
When `ENABLE_TELEMETRY=true`, all requests include:
- Span creation with detailed attributes
- HTTP request/response tracing
- Cache hit/miss metrics
- Error recording and status

### Example Trace Attributes
```
esi.endpoint: "corporation"
esi.corporation_id: 98579006
esi.base_url: "https://esi.evetech.net"
cache.key: "https://esi.evetech.net/corporations/98579006/"
cache.hit: true
http.method: "GET"
http.status_code: 200
```

## Performance Considerations

### Cache Strategy
- **Cache Hit**: Sub-millisecond response time
- **Cache Miss**: Network latency + ESI response time  
- **304 Not Modified**: Minimal network overhead

### Best Practices
```go
// Good: Use cache-aware methods for frequently accessed data
result, err := client.Corporation.GetCorporationInfoWithCache(ctx, corporationID)

// Good: Check cache expiry to optimize refresh timing
if result.Cache.ExpiresAt != nil && time.Until(*result.Cache.ExpiresAt) < 5*time.Minute {
    // Data expires soon, consider refreshing
}

// Good: Handle rate limiting gracefully
if strings.Contains(err.Error(), "status 420") {
    time.Sleep(time.Minute) // Wait before retry
}
```

## Configuration

### Environment Variables
```bash
# ESI User-Agent (required for CCP compliance)
ESI_USER_AGENT="go-falcon/1.0.0 (contact@example.com) +https://github.com/org/repo"

# OpenTelemetry (optional)
ENABLE_TELEMETRY=true
```

### Recommended User-Agent Format
```
"YourApp/1.0.0 (your-email@domain.com) +https://github.com/yourorg/yourrepo"
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "go-falcon/pkg/evegateway"
)

func main() {
    client := evegateway.NewClient()
    ctx := context.Background()
    
    // Example corporation: Goonswarm Federation
    corporationID := 1344654522
    
    // Get basic corporation info (public)
    fmt.Println("=== Corporation Information ===")
    corp, err := client.Corporation.GetCorporationInfo(ctx, corporationID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Name: %s [%s]\n", corp.Name, corp.Ticker)
    fmt.Printf("CEO: %d\n", corp.CEOCharacterID)
    fmt.Printf("Members: %d\n", corp.MemberCount)
    fmt.Printf("Founded: %v\n", corp.DateFounded.Format("2006-01-02"))
    fmt.Printf("Tax Rate: %.1f%%\n", corp.TaxRate*100)
    
    // Get corporation icons (public)
    fmt.Println("\n=== Corporation Icons ===")
    icons, err := client.Corporation.GetCorporationIcons(ctx, corporationID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("64x64: %s\n", icons.Px64x64)
    fmt.Printf("128x128: %s\n", icons.Px128x128)
    fmt.Printf("256x256: %s\n", icons.Px256x256)
    
    // Get alliance history (public)
    fmt.Println("\n=== Alliance History ===")
    history, err := client.Corporation.GetCorporationAllianceHistory(ctx, corporationID)
    if err != nil {
        log.Fatal(err)
    }
    
    for i, entry := range history {
        if i >= 3 { // Show only recent entries
            break
        }
        allianceInfo := "None"
        if entry.AllianceID > 0 {
            allianceInfo = fmt.Sprintf("Alliance %d", entry.AllianceID)
        }
        fmt.Printf("%s from %v\n", allianceInfo, entry.StartDate.Format("2006-01-02"))
    }
    
    // Example with cache information
    fmt.Println("\n=== Cache Information ===")
    result, err := client.Corporation.GetCorporationInfoWithCache(ctx, corporationID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Corporation: %s\n", result.Data.Name)
    fmt.Printf("From Cache: %v\n", result.Cache.Cached)
    if result.Cache.ExpiresAt != nil {
        fmt.Printf("Cache Expires: %v\n", result.Cache.ExpiresAt.Format("2006-01-02 15:04:05"))
        fmt.Printf("Time Until Expiry: %v\n", time.Until(*result.Cache.ExpiresAt).Round(time.Second))
    }
    
    // Authenticated endpoints would require a valid access token:
    /*
    accessToken := "your_eve_sso_access_token"
    
    members, err := client.Corporation.GetCorporationMembers(ctx, corporationID, accessToken)
    if err != nil {
        log.Printf("Members endpoint requires authentication: %v", err)
    } else {
        fmt.Printf("Corporation has %d members\n", len(members))
    }
    */
}
```

## Pagination Support

### Token-Based Pagination (Future Implementation)
**Note**: CCP has announced token-based pagination for select corporation endpoints, starting with Corporation Projects. This section documents the planned implementation for future development:

#### Key Concepts
- **Opaque Tokens**: `before` and `after` tokens are opaque strings - never attempt to parse or validate them
- **Time-Ordered Data**: Results sorted by "last modified" time for consistency
- **Bidirectional Navigation**: Navigate both forwards and backwards through datasets
- **Long-Term Validity**: Tokens remain valid for hours or weeks

#### Usage Example (Future Implementation)
```go
// When corporation endpoints support token-based pagination:
type CorporationProjectsParams struct {
    Before string `json:"before,omitempty"` // Get entries before this token
    After  string `json:"after,omitempty"`  // Get entries after this token  
    Limit  int    `json:"limit,omitempty"`  // Number of entries per page
}

type CorporationProjectsResult struct {
    Data   []CorporationProject `json:"data"`
    Before *string              `json:"before,omitempty"` // Token for previous page
    After  *string              `json:"after,omitempty"`  // Token for next page
    Cache  CacheInfo            `json:"cache"`
}

// Get most recent projects
result, err := client.Corporation.GetCorporationProjects(ctx, corporationID, token, CorporationProjectsParams{
    Limit: 50,
})

// Navigate to older projects using 'before' token
if result.Before != nil {
    olderResult, err := client.Corporation.GetCorporationProjects(ctx, corporationID, token, CorporationProjectsParams{
        Before: *result.Before,
        Limit:  50,
    })
}

// Check for newer projects using 'after' token
if result.After != nil {
    newerResult, err := client.Corporation.GetCorporationProjects(ctx, corporationID, token, CorporationProjectsParams{
        After: *result.After,
        Limit: 50,
    })
}
```

#### Best Practices for Token-Based Pagination
1. **Store Tokens**: Persist tokens to resume pagination sessions later
2. **Handle Duplicates**: Data modifications during pagination may cause duplicates
3. **Empty Results**: Empty response indicates reaching dataset boundary
4. **Monitor Changes**: Use `after` token to detect new/updated records
5. **Full Scans**: Use `before` token to crawl through entire dataset

### Current Corporation Endpoints
Most current corporation endpoints return complete datasets or use simple response formats:

| Endpoint | Pagination Type | Notes |
|----------|----------------|--------|
| `GetCorporationMembers` | None | Returns all members |
| `GetCorporationStructures` | None | Returns all structures |
| `GetCorporationWallets` | None | Returns all wallet divisions |
| `GetCorporationMemberTracking` | None | Returns all member tracking data |
| `GetCorporationMemberRoles` | None | Returns all member roles |
| `GetCorporationStandings` | None | Returns all standings |
| `GetCorporationAllianceHistory` | None | Returns complete history |
| Corporation Projects | Token-based (future) | Will use new pagination system |

### Planned Migration Timeline
- **Phase 1**: Corporation Projects endpoint (planned first implementation)
- **Phase 2**: High-volume endpoints (members, tracking, structures) - future
- **Phase 3**: Remaining endpoints as needed - future  
- **Legacy Support**: Current endpoints will continue working unchanged

## Testing

```bash
# Test corporation client compilation
go build ./pkg/evegateway/corporation

# Run with telemetry enabled
ENABLE_TELEMETRY=true ESI_USER_AGENT="test/1.0.0 test@example.com" go run your_app.go
```

## Contributing

1. Follow EVE Online ESI best practices
2. Include comprehensive error handling  
3. Add OpenTelemetry tracing to new endpoints
4. Update documentation for new features
5. Test with both cached and fresh data scenarios

## References

- [EVE Online ESI Documentation](https://esi.evetech.net/ui/)
- [EVE Online ESI Best Practices](https://developers.eveonline.com/docs/services/esi/best-practices/)
- [CCP Developer Resources](https://developers.eveonline.com/)