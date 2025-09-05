# ZKillboard Module

## Overview

The ZKillboard module implements a real-time killmail feed consumer that connects to ZKillboard's RedisQ service. It provides a production-ready, long-running background service that continuously polls for new killmails, processes them through an enrichment pipeline, and aggregates statistics for analytics.

## Architecture

### Core Components

```
internal/zkillboard/
├── dto/                    # Data Transfer Objects
│   ├── redisq.go          # RedisQ API structures
│   └── outputs.go         # API response DTOs
├── models/                # Database Models
│   └── models.go          # ZKB metadata, timeseries, consumer state
├── services/              # Business Logic
│   ├── redisq_consumer.go # RedisQ polling service
│   ├── processor.go       # Killmail processing pipeline
│   ├── repository.go      # Database operations
│   ├── aggregator.go      # Timeseries aggregation
│   └── rate_limiter.go    # Rate limiting compliance
├── routes/                # HTTP Endpoints
│   └── routes.go          # API routes for service control
├── module.go              # Module initialization
└── CLAUDE.md              # This documentation
```

### Data Flow

```
ZKillboard RedisQ → Poll → Validate → Deduplicate → Process → Store → Aggregate → Notify
```

1. **RedisQ Polling**: HTTP polling with adaptive TTW (1-10 seconds)
2. **Validation**: Parse and validate incoming killmail data
3. **Deduplication**: Check against existing killmails to avoid duplicates
4. **Processing**: Convert ESI format to internal models
5. **Storage**: Batch insert to `killmails` and `zkb_metadata` collections
6. **Aggregation**: Update timeseries statistics (hourly/daily/monthly)
7. **Notification**: Emit WebSocket events for real-time updates

### Database Collections

#### `zkb_metadata`
Stores ZKillboard-specific metadata for each killmail:
- `killmail_id`: Reference to main killmail
- `location_id`: ZKB location data
- `hash`: ZKB hash
- `fitted_value`, `dropped_value`, `destroyed_value`, `total_value`: ISK breakdown
- `points`: ZKB point value
- `npc`, `solo`, `awox`: Kill flags
- `href`: ZKB URL

#### `killmail_timeseries`
Aggregated statistics over time:
- `period`: "hour", "day", "month"
- `timestamp`: Time bucket
- Dimensional breakdowns: system, region, alliance, corporation, ship type
- Metrics: kill count, total value, NPC/solo kills
- Top performers: victims, attackers by value/count

#### `zkb_consumer_state`
Consumer service state persistence:
- `queue_id`: Unique queue identifier
- `state`: Service status (stopped, running, throttled, draining)
- Performance metrics: polls, nulls, errors, rate limits
- Recovery data: last poll time, null streak, TTW value

## RedisQ Integration

### ZKillboard RedisQ Specifications

- **Endpoint**: `https://zkillredisq.stream/listen.php`
- **Method**: GET
- **Parameters**:
  - `queueID`: Unique identifier (auto-generated: `go-falcon-{hostname}-{timestamp}`)
  - `ttw`: Time to wait (1-10 seconds, adaptive)

### Rate Limiting Compliance

- **1 concurrent request per queueID**: Mutex-protected polling
- **2 requests per second per IP**: 500ms minimum interval
- **Exponential backoff**: 5s → 10s → 20s → 40s → 80s for rate limit hits

### 3-Hour Queue Recovery

ZKillboard maintains a 3-hour queue memory, allowing for:
- Service restarts without data loss
- Intermittent connectivity handling
- No local queue persistence required

### Adaptive Polling Strategy

```go
// TTW Algorithm
if nullStreak >= nullThreshold (5) {
    ttw = ttwMax (10 seconds)  // Quiet period
} else {
    ttw = ttwMin (1 second)    // Active period
}
```

## API Endpoints

### Service Control

#### `GET /zkillboard/status`
Returns current service status and metrics.

**Response:**
```json
{
  "status": "running",
  "queue_id": "go-falcon-prod-1",
  "last_poll": "2025-01-15T12:30:45Z",
  "last_killmail_id": 123456789,
  "metrics": {
    "total_polls": 86400,
    "null_responses": 43200,
    "killmails_found": 1250,
    "http_errors": 3,
    "parse_errors": 0,
    "store_errors": 1,
    "rate_limit_hits": 5,
    "current_ttw": 1,
    "null_streak": 0,
    "uptime": "24h30m15s"
  },
  "config": {
    "endpoint": "https://zkillredisq.stream/listen.php",
    "ttw_min": 1,
    "ttw_max": 10,
    "null_threshold": 5,
    "batch_size": 10
  }
}
```

#### `POST /zkillboard/control`
Control service operations (requires authentication).

**Request:**
```json
{
  "action": "start|stop|restart",
  "queue_id": "optional-override"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Service started successfully",
  "status": "running"
}
```

### Statistics

#### `GET /zkillboard/stats?period=day`
Returns aggregated killmail statistics.

**Parameters:**
- `period`: "hour", "day", "week", "month"

**Response:**
```json
{
  "period": "day",
  "total_killmails": 1250,
  "total_value": 15000000000.50,
  "npc_kills": 125,
  "solo_kills": 89,
  "top_systems": [
    {
      "system_id": 30000142,
      "system_name": "Jita",
      "kills": 45,
      "value": 5000000000.00
    }
  ],
  "top_alliances": [...],
  "top_ship_types": [...]
}
```

#### `GET /zkillboard/recent?limit=20`
Returns recently processed killmails.

**Response:**
```json
{
  "killmails": [
    {
      "killmail_id": 123456789,
      "timestamp": "2025-01-15T12:30:00Z",
      "solar_system_id": 30000142,
      "system_name": "Jita",
      "victim_id": 987654321,
      "victim_name": "Character Name",
      "ship_type_id": 670,
      "ship_type_name": "Capsule",
      "total_value": 1234567.89,
      "points": 1,
      "solo": false,
      "npc": false,
      "href": "https://zkillboard.com/kill/123456789/"
    }
  ],
  "count": 20
}
```

## Configuration

### Environment Variables

```bash
# ZKillboard Service Configuration
ZKB_ENABLED=true                               # Enable/disable service
ZKB_QUEUE_ID=go-falcon-prod-1                  # Unique queue ID (auto-generated if not set)
ZKB_ENDPOINT=https://zkillredisq.stream/listen.php  # RedisQ endpoint
ZKB_TTW_MIN=1                                  # Minimum time-to-wait (seconds)
ZKB_TTW_MAX=10                                 # Maximum time-to-wait (seconds)
ZKB_NULL_THRESHOLD=5                           # Nulls before increasing TTW
ZKB_HTTP_TIMEOUT=30s                           # HTTP request timeout
ZKB_RETRY_DELAY=5s                             # Delay after errors
ZKB_BATCH_SIZE=10                              # Database batch insert size
ZKB_TTL_DAYS=90                                # Timeseries data retention (days)
```

## Performance Characteristics

### CPU Efficiency
- **Single goroutine polling**: No concurrency complexity
- **Batch processing**: Configurable batch sizes (default: 10)
- **Adaptive polling**: Reduces CPU during quiet periods
- **Efficient JSON parsing**: Streaming with `json.RawMessage`

### Memory Management
- **Bounded batches**: Prevents memory growth
- **Connection pooling**: HTTP/2 keep-alive connections
- **Garbage collection friendly**: Minimal allocation patterns
- **TTL cleanup**: Automatic timeseries data expiration

### Network Optimization
- **HTTP/2 connection pooling**: Reuse connections
- **Compressed responses**: Reduced bandwidth usage
- **Minimal request overhead**: Simple GET requests
- **Rate limit compliance**: Avoids unnecessary retries

## Monitoring & Observability

### Metrics Dashboard

Key metrics for monitoring service health:

- **Polling Performance**: Polls/second, average response time, success rate
- **Data Processing**: Killmails/hour, processing time, batch efficiency
- **Error Tracking**: HTTP errors, parse failures, rate limit hits
- **Resource Usage**: Memory consumption, goroutine count, GC pressure

### Health Check Endpoints

The `/zkillboard/status` endpoint provides comprehensive health information:

- ✅ **Healthy**: Service running, low error rate, recent activity
- ⚠️ **Warning**: High error rate, rate limiting, or service stopped
- ❌ **Unhealthy**: Service crashed, persistent failures, or misconfiguration

### Logging Integration

Structured logging with OpenTelemetry correlation:

```go
logger.Info().
    Int64("killmail_id", km.KillmailID).
    Float64("value", zkb.TotalValue).
    Bool("solo", zkb.Solo).
    Bool("npc", zkb.NPC).
    Msg("Killmail processed")
```

## Real-Time Integration

### WebSocket Notifications

New killmails trigger real-time WebSocket events:

```javascript
// Event: killmail:new
{
  "killmail_id": 123456789,
  "timestamp": "2025-01-15T12:30:00Z",
  "solar_system_id": 30000142,
  "system_name": "Jita",
  "victim_name": "Character Name",
  "ship_type_name": "Capsule",
  "total_value": 1234567.89,
  "solo": false,
  "npc": false,
  "href": "https://zkillboard.com/kill/123456789/"
}
```

### Frontend Integration

Subscribe to killmail events:

```javascript
const websocket = new WebSocket('ws://localhost:3000/websocket');
websocket.on('killmail:new', (killmail) => {
    updateKillmailFeed(killmail);
    showNotification(`New ${killmail.ship_type_name} kill in ${killmail.system_name}`);
});
```

## Security Considerations

### Authentication & Authorization

- **Public Status**: `/status` endpoint is public for monitoring
- **Admin Control**: Service control requires authentication and `zkillboard:control` permission
- **Statistics**: Statistics endpoints are public by default

### Input Validation

- **Killmail Validation**: Comprehensive validation of RedisQ data
- **Duplicate Prevention**: Database constraints and application-level checks
- **Rate Limiting**: Built-in compliance with ZKillboard limits

## Deployment

### Production Deployment

1. **Environment Setup**: Configure all required environment variables
2. **Database Indexes**: Automatically created during module initialization
3. **Service Start**: Use API endpoints to start/stop the consumer service
4. **Monitoring**: Set up alerts on health check endpoints and error metrics

### Development Setup

```bash
# Basic configuration for development
export ZKB_ENABLED=true
export ZKB_QUEUE_ID=go-falcon-dev-$(whoami)
export ZKB_BATCH_SIZE=5  # Smaller batches for testing

# Start the service
curl -X POST http://localhost:3000/api/zkillboard/control \
  -H "Content-Type: application/json" \
  -d '{"action": "start"}'

# Monitor status
curl http://localhost:3000/api/zkillboard/status
```

### Scaling Considerations

- **Single Instance**: Each instance requires a unique `queueID`
- **Load Balancing**: Service control endpoints should hit the same instance
- **Database Sharding**: Consider sharding by time period for large datasets
- **Read Replicas**: Statistics queries can use read replicas

## Maintenance

### Regular Operations

- **Monitor Service Health**: Check status endpoint regularly
- **Review Error Rates**: Investigate persistent HTTP or parse errors
- **Capacity Planning**: Monitor killmail processing rate vs. arrival rate
- **Data Cleanup**: Timeseries data automatically expires based on TTL

### Troubleshooting

#### Service Won't Start
- Check environment variables
- Verify database connectivity
- Ensure unique `queueID`
- Review logs for specific errors

#### High Error Rate
- Check ZKillboard service status
- Verify network connectivity
- Review rate limiting metrics
- Check ESI client configuration

#### Missing Killmails
- Verify service is running and not throttled
- Check for duplicate filtering issues
- Review batch processing logs
- Investigate database write errors

## Integration with Scheduler Module

The ZKillboard module can be integrated with the scheduler for automated management:

```go
// Example task to restart consumer daily
scheduler.RegisterSystemTask("zkb_restart", &tasks.Task{
    Name:     "ZKillboard Consumer Restart",
    Schedule: "0 4 * * *", // Daily at 4 AM
    Handler: func(ctx context.Context) error {
        return zkbModule.GetConsumer().Stop()
        time.Sleep(5 * time.Second)
        return zkbModule.GetConsumer().Start(ctx)
    },
})
```

## Future Enhancements

### Planned Features

- **Advanced Filtering**: Corporation/alliance-specific killmail filtering
- **Machine Learning**: Anomaly detection for unusual killmail patterns
- **Historical Import**: Bulk import of historical killmail data
- **API Rate Optimization**: Dynamic TTW adjustment based on activity patterns

### Performance Improvements

- **Parallel Processing**: Multi-threaded processing for high-volume periods
- **Caching Layer**: Redis caching for frequently accessed statistics
- **Compression**: Database compression for timeseries data
- **Archival**: Cold storage for old killmail data

This module provides a robust, production-ready solution for real-time killmail processing that scales with EVE Online's activity levels while maintaining reliability and performance.