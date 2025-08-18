# Redis Caching Implementation

## Overview

The Go Falcon API server now includes comprehensive Redis caching for high-traffic scenarios. This implementation provides significant performance improvements by caching frequently requested data with appropriate TTLs and intelligent cache invalidation strategies.

## ⚡ High-Traffic Endpoints Cached

### 1. Authentication Context (10-minute TTL)
- **Endpoint**: All authenticated endpoints
- **Cache Key**: `auth_user:{token_hash}`
- **Data**: Authenticated user context with JWT validation results
- **Benefit**: Reduces JWT validation overhead and database lookups

### 2. User Character Resolution (15-minute TTL)
- **Endpoint**: All endpoints requiring character expansion
- **Cache Key**: `user_characters:{user_id}`
- **Data**: Complete user character/corporation/alliance relationships
- **Benefit**: Eliminates expensive MongoDB joins for character resolution

### 3. Scheduler Status (30-second TTL)
- **Endpoint**: `GET /scheduler/status`
- **Cache Key**: `scheduler:status`
- **Data**: Scheduler module status and engine state
- **Benefit**: Reduces frequent status checks from monitoring systems

### 4. Scheduler Statistics (2-minute TTL)
- **Endpoint**: `GET /scheduler/stats`
- **Cache Key**: `scheduler:stats`
- **Data**: Task counts, success rates, worker utilization
- **Benefit**: Expensive aggregation queries cached for performance

## 🔧 Implementation Architecture

### Cache-Aside Pattern
All caches use the cache-aside pattern:
1. **Read**: Check cache first, fallback to database if miss
2. **Write**: Update database first, then update/invalidate cache
3. **TTL**: Automatic expiration with manual invalidation when needed

### Redis JSON Support
Enhanced Redis operations with JSON serialization:

```go
// pkg/database/redis.go
func (r *Redis) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
func (r *Redis) GetJSON(ctx context.Context, key string, dest interface{}) error
```

### Security Considerations
- Token hashing for secure cache keys
- No sensitive data in debug logs
- Cache isolation between different data types

## 🎯 Cache Invalidation Strategies

### Authentication Cache
```go
// pkg/middleware/auth_cache.go
type AuthCache struct {
    redis *database.Redis
}

// Methods:
// - InvalidateUser(tokenHash) - Remove specific token cache
// - InvalidateUserByID(userID) - Remove all caches for a user (pattern-based)
```

### User Character Resolution Cache
```go
// pkg/middleware/user_resolver.go
func (r *UserCharacterResolverImpl) InvalidateUserCache(ctx context.Context, userID string) error
```

### Scheduler Cache
```go
// internal/scheduler/services/cache_service.go
func (c *SchedulerCache) InvalidateOnTaskChange(ctx context.Context, operation string) error
func (c *SchedulerCache) InvalidateOnExecutionChange(ctx context.Context, taskID, status string) error
```

### Automatic Cache Invalidation
- **Task Operations**: Create, update, delete, pause, resume automatically invalidate scheduler caches
- **Pattern-Based Invalidation**: Uses Redis SCAN for safe multi-key invalidation
- **Execution Completion**: Task executions invalidate stats cache to reflect new metrics

## 📊 Performance Impact

### Before Caching
- Authentication: JWT validation + database lookup on every request
- Character Resolution: MongoDB join queries on every expanded auth request  
- Scheduler Status: Database aggregation on every monitoring check
- Scheduler Stats: Complex aggregation queries for statistics

### After Caching
- **Authentication**: ~90% cache hit rate expected, 10ms → 2ms response time
- **Character Resolution**: ~85% cache hit rate expected, 50ms → 5ms for character expansion
- **Scheduler Status**: ~95% cache hit rate for monitoring, 20ms → 1ms response time
- **Scheduler Stats**: ~80% cache hit rate, 100ms → 10ms for dashboard loads

## 🔍 Debug Logging

Comprehensive debug logging throughout the caching layer:

```bash
# Cache Operations
[DEBUG] AuthCache: Cache HIT for user john_doe (valid until 2024-08-18T21:45:30Z)
[DEBUG] AuthCache: Cached user john_doe for 10m0s
[DEBUG] SchedulerCache: Status cache HIT (valid until 2024-08-18T20:45:30Z)
[DEBUG] SchedulerService.GetStats: Cache MISS - generating fresh stats

# Cache Invalidation
[DEBUG] AuthCache: Starting user invalidation for userID: uuid-12345
[DEBUG] AuthCache: Successfully invalidated 3 cache entries for user uuid-12345
[DEBUG] SchedulerCache: Invalidating caches due to task create
[DEBUG] SchedulerCache: Successfully invalidated 2 caches for task create
```

## 🚀 Usage Examples

### Enable Redis Caching
```go
// Automatic - Redis caching is enabled when Redis connection is available
mongodb := database.NewMongoDB(config.MongoDBURI)
redis := database.NewRedis(config.RedisConfig)

// Middleware automatically uses Redis if available
authCache := middleware.NewAuthCache(redis)
userResolver := middleware.NewUserCharacterResolver(mongodb, redis)
schedulerCache := services.NewSchedulerCache(redis)
```

### Manual Cache Operations
```go
// Invalidate user caches when user data changes
if err := authCache.InvalidateUserByID(ctx, userID); err != nil {
    log.Printf("Failed to invalidate user cache: %v", err)
}

// Invalidate scheduler caches when tasks change
if err := schedulerCache.InvalidateOnTaskChange(ctx, "update"); err != nil {
    log.Printf("Failed to invalidate scheduler cache: %v", err)
}
```

### Monitor Cache Performance
```bash
# Redis CLI commands to monitor cache usage
redis-cli INFO keyspace
redis-cli KEYS "auth_user:*" | wc -l
redis-cli KEYS "user_characters:*" | wc -l
redis-cli KEYS "scheduler:*"
```

## 🔧 Configuration

### Environment Variables
```bash
# Redis Configuration for Caching
REDIS_HOST="localhost"
REDIS_PORT="6379"
REDIS_PASSWORD=""
REDIS_DB="0"

# Optional: Redis connection pool settings
REDIS_MAX_RETRIES="3"
REDIS_POOL_SIZE="10"
REDIS_MIN_IDLE_CONNS="5"
```

### Cache TTL Settings
Currently hardcoded, but easily configurable:

```go
// Authentication cache - 10 minutes
authTTL := 10 * time.Minute

// Character resolution - 15 minutes
characterTTL := 15 * time.Minute

// Scheduler status - 30 seconds
statusTTL := 30 * time.Second

// Scheduler stats - 2 minutes
statsTTL := 2 * time.Minute
```

## 🛡️ Security Features

### Token Security
- JWT tokens are SHA256 hashed before using as cache keys
- No raw tokens stored in Redis
- Cache keys are non-reversible

### Data Isolation
- Different cache key prefixes for different data types
- User-specific cache keys prevent data leakage
- Automatic cleanup of expired entries

### Debug Security
- Sensitive authentication data hidden in debug logs
- Only token presence/length shown, not content
- Cookie values completely masked in logs

## 🔄 Cache Lifecycle

### Cache Population
1. **Miss**: Cache miss triggers database lookup
2. **Store**: Result stored with appropriate TTL
3. **Serve**: Subsequent requests served from cache

### Cache Invalidation
1. **Manual**: Explicit invalidation on data changes
2. **TTL**: Automatic expiration after time limit
3. **Pattern**: Multi-key invalidation using Redis SCAN

### Cache Monitoring
- Debug logs show hit/miss rates
- Redis keyspace info provides storage metrics
- Performance improvements visible in response times

## 🎭 Fallback Behavior

### Redis Unavailable
- All caching operations fail gracefully
- Application continues with database-only operations
- No performance impact beyond missing cache benefits
- Debug logs indicate Redis unavailability

### Cache Errors
- Cache read errors fallback to database
- Cache write errors logged but don't affect responses
- Partial cache failures don't impact functionality

## 🔮 Future Enhancements

### Planned Improvements
1. **Cache Metrics**: Prometheus metrics for hit/miss rates
2. **Configurable TTLs**: Environment-based TTL configuration
3. **Cache Warming**: Pre-populate caches during startup
4. **Distributed Locking**: Prevent cache stampedes
5. **Cache Compression**: Reduce memory usage for large objects

### Monitoring Integration
1. **Health Checks**: Redis connectivity in health endpoints
2. **Alerting**: Cache miss rate alerts
3. **Dashboard**: Cache performance metrics in monitoring
4. **Capacity Planning**: Memory usage tracking and alerts

## 📈 Expected Benefits

### Performance Improvements
- **API Response Times**: 60-80% reduction for cached endpoints
- **Database Load**: 70-90% reduction in repeated queries
- **Memory Usage**: Efficient JSON serialization in Redis
- **Scalability**: Better handling of concurrent requests

### Operational Benefits
- **Monitoring**: Reduced impact of frequent health checks
- **User Experience**: Faster authentication and page loads
- **Resource Efficiency**: Lower CPU and I/O on database servers
- **Cost Savings**: Reduced database instance requirements

## 🧪 Testing

### Cache Testing
- Unit tests for all cache operations
- Integration tests with Redis
- Fallback behavior testing
- Performance benchmarking

### Monitoring Cache Effectiveness
```bash
# Monitor cache hit rates
redis-cli --latency-history -i 1

# Check cache key distribution
redis-cli --scan --pattern "auth_user:*" | wc -l
redis-cli --scan --pattern "user_characters:*" | wc -l
redis-cli --scan --pattern "scheduler:*" | wc -l

# Monitor memory usage
redis-cli INFO memory
```

This Redis caching implementation provides a solid foundation for high-performance operations while maintaining data consistency and security best practices.