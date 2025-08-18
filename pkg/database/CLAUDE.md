# Database Package (pkg/database)

## Overview
Database connection utilities for MongoDB and Redis with proper connection pooling, error handling, and graceful shutdown. Provides consistent database access patterns across all modules.

## Core Features
- **MongoDB Integration**: Connection management with proper authentication
- **Redis Integration**: Connection pooling and configuration
- **Connection Health**: Ping testing and connection validation
- **Graceful Shutdown**: Proper cleanup and connection closing
- **Error Handling**: Comprehensive error reporting and recovery

## MongoDB Features
- **Connection String Parsing**: Full MongoDB URI support
- **Database Selection**: Automatic database switching
- **Collection Access**: Simplified collection retrieval
- **Authentication**: Built-in auth source handling
- **Connection Pooling**: Automatic connection management
- **Health Monitoring**: Connection health checks with automatic reconnection
- **Connection Recovery**: Automatic reconnection on "client is disconnected" errors
- **Stale Connection Handling**: Detects and recovers from stale connections

## Redis Features  
- **URL Parsing**: Redis connection string support
- **Connection Pooling**: Efficient connection reuse
- **Ping Testing**: Health check capabilities
- **Error Recovery**: Automatic reconnection handling

## Usage Examples
```go
// MongoDB with automatic health checks
mongodb, err := database.NewMongoDB(ctx, "database_name")
collection := mongodb.Collection("users")

// Health check before database operations
err = mongodb.HealthCheck(ctx)  // Automatically reconnects if disconnected
if err != nil {
    // Connection is unhealthy
}

// Redis  
redis, err := database.NewRedis(ctx)
err = redis.Set(ctx, "key", "value", 0)
```

## Connection Recovery
The MongoDB connection includes automatic recovery mechanisms:

```go
// Health check with automatic reconnection
func (m *MongoDB) HealthCheck(ctx context.Context) error {
    // Ping MongoDB
    err := m.Client.Ping(ctx, nil)
    if err != nil {
        // Automatically attempt reconnection
        if reconnErr := m.reconnect(ctx); reconnErr != nil {
            return fmt.Errorf("ping failed and reconnect failed: %v", reconnErr)
        }
        // Verify reconnection
        return m.Client.Ping(ctx, nil)
    }
    return nil
}
```

## Best Practices
- Always call `HealthCheck()` before critical database operations
- The health check automatically handles "client is disconnected" errors
- Connection recovery is transparent to the application
- Failed health checks indicate persistent connection issues

## Configuration
- `MONGODB_URI`: Full MongoDB connection string
- `REDIS_URL`: Redis connection URL

## Integration
Used by all modules through the base module interface for consistent database access patterns.