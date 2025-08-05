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

## Redis Features  
- **URL Parsing**: Redis connection string support
- **Connection Pooling**: Efficient connection reuse
- **Ping Testing**: Health check capabilities
- **Error Recovery**: Automatic reconnection handling

## Usage Examples
```go
// MongoDB
mongodb, err := database.NewMongoDB(ctx, "database_name")
collection := mongodb.Collection("users")

// Redis  
redis, err := database.NewRedis(ctx)
err = redis.Set(ctx, "key", "value", 0)
```

## Configuration
- `MONGODB_URI`: Full MongoDB connection string
- `REDIS_URL`: Redis connection URL

## Integration
Used by all modules through the base module interface for consistent database access patterns.