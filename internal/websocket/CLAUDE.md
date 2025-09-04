# WebSocket Module (internal/websocket)

## Overview

The WebSocket module provides real-time bidirectional communication capabilities for the go-falcon API gateway. It enables instant updates, notifications, and live data streaming to authenticated users through secure WebSocket connections with automatic room management based on user and group memberships.

**Status**: Production Ready - Complete Real-Time Communication System
**Authentication**: JWT-based authentication with automatic room assignment
**Multi-Instance**: Redis pub/sub for horizontal scaling across server instances

## Architecture

### Core Components

- **Connection Manager**: WebSocket connection lifecycle management with authentication
- **Room System**: Automatic personal and group room assignment based on user memberships
- **Message Broadcasting**: Real-time message distribution with Redis pub/sub
- **Integration Service**: Seamless integration with user and group modules
- **Health Monitoring**: Connection health tracking and automatic cleanup

### Files Structure

```
internal/websocket/
├── dto/
│   ├── inputs.go         # WebSocket request DTOs with Huma v2 validation
│   └── outputs.go        # WebSocket response DTOs
├── middleware/
│   └── auth.go          # WebSocket-specific authentication middleware
├── models/
│   └── models.go        # Connection, Room, and Message data structures
├── routes/
│   └── routes.go        # Huma v2 API endpoints and WebSocket upgrade handler
├── services/
│   ├── service.go       # Main WebSocket service orchestrator
│   ├── connection.go    # Connection management and lifecycle
│   ├── room.go         # Room management and membership
│   ├── redis.go        # Redis pub/sub for multi-instance broadcasting
│   ├── integration.go  # User/group module integration
│   └── repository.go   # Redis storage for connection metadata
├── module.go           # Module initialization and interface implementation
└── CLAUDE.md          # This documentation
```

## Core Features

### 🔌 WebSocket Connection Management
- **JWT Authentication**: Secure connection establishment using existing auth system
- **Connection Lifecycle**: Automatic connection tracking, health monitoring, and cleanup
- **Concurrent Support**: Thread-safe connection management with mutex protection
- **Heartbeat System**: Ping/pong mechanism to detect inactive connections

### 🏠 Automatic Room Assignment
- **Personal Rooms**: Every user gets a private room (`user:{user_id}`)
- **Group Rooms**: Automatic assignment to group rooms based on current memberships (`group:{group_id}`)
- **Dynamic Updates**: Real-time room assignment changes when group memberships change
- **Permission-Based**: Only users in groups can access corresponding group rooms

### 📨 Message Broadcasting System
- **Room Broadcasting**: Messages sent to all members of a specific room
- **Direct Messaging**: Point-to-point messages between specific connections
- **System Notifications**: Server-wide announcements and alerts
- **Multi-Instance**: Redis pub/sub ensures messages reach users across all server instances

### 🔄 Real-Time Integrations
- **User Profile Updates**: Live updates when user profiles change
- **Group Membership Changes**: Instant room assignment updates
- **Custom Events**: Extensible event system for application-specific messages

## Data Models

### Connection Model
```go
type Connection struct {
    ID            string          // Unique connection identifier
    UserID        string          // User UUID from auth system
    CharacterID   int64           // EVE character ID
    CharacterName string          // EVE character name
    Conn          *websocket.Conn // Underlying WebSocket connection
    Rooms         []string        // List of joined room IDs
    CreatedAt     time.Time       // Connection creation timestamp
    LastPing      time.Time       // Last heartbeat timestamp
}
```

### Room Model
```go
type Room struct {
    ID        string    // Room identifier (user:{id} or group:{id})
    Type      RoomType  // personal or group
    Name      string    // Human-readable room name
    Members   []string  // List of connection IDs in the room
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Message Model
```go
type Message struct {
    Type      MessageType            // Message type (user_update, group_change, etc.)
    Room      string                 // Target room ID (optional)
    From      string                 // Sender connection ID
    To        string                 // Target connection ID (for direct messages)
    Data      map[string]interface{} // Message payload
    Timestamp time.Time
}
```

## Message Types

### System Message Types
```go
const (
    // Standard websocket message types
    MessageTypeMessage               = "message"
    MessageTypeUserProfileUpdate     = "user_profile_update"
    MessageTypeGroupMembershipChange = "group_membership_change"
    MessageTypeSystemNotification    = "system_notification"
    MessageTypePresence              = "presence"
    MessageTypeNotification          = "notification"
    MessageTypeRoomUpdate            = "room_update"
    MessageTypeBackendStatus         = "backend_status"
    MessageTypeCriticalAlert         = "critical_alert"
    MessageTypeServiceRecovery       = "service_recovery"
)
```

**Standardized Message Types:**
- `message` - Generic messaging between users
- `user_profile_update` - User profile changes
- `group_membership_change` - Group membership changes  
- `system_notification` - System-wide notifications
- `presence` - User online/offline status (replaces heartbeat)
- `notification` - General user notifications
- `room_update` - Room changes (replaces room_joined/room_left)
- `backend_status` - Backend service status updates
- `critical_alert` - Critical system alerts
- `service_recovery` - Service recovery notifications

### Message Flow Examples

#### User Profile Update
```json
{
  "type": "user_profile_update",
  "data": {
    "user_id": "uuid-string",
    "character_id": 123456789,
    "profile": {
      "character_name": "Updated Name",
      "last_login": "2024-01-01T12:00:00Z"
    }
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### Group Membership Change
```json
{
  "type": "group_membership_change",
  "data": {
    "user_id": "uuid-string",
    "group_id": "group-object-id",
    "group_name": "Fleet Commanders",
    "joined": true
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## API Endpoints

### WebSocket Connection

#### Establish WebSocket Connection
```
GET /websocket/connect
Upgrade: websocket
Authorization: Bearer <token> | Cookie: falcon_auth_token
```
**Description**: Upgrades HTTP connection to WebSocket protocol
**Authentication**: Required - JWT token via header or cookie
**Response**: WebSocket connection with automatic room assignment

### Administrative Endpoints

All admin endpoints require super administrator privileges.

#### List Active Connections
```
GET /websocket/connections?user_id={uuid}&character_id={id}&room_id={room}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

#### Get Connection Details
```
GET /websocket/connections/{connection_id}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

#### List WebSocket Rooms
```
GET /websocket/rooms?type={personal|group}&member_id={connection_id}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

#### Get Room Details
```
GET /websocket/rooms/{room_id}
Authorization: Bearer <token> | Cookie: falcon_auth_token
```

#### Broadcast Message to All Connections
```
POST /websocket/broadcast
Authorization: Bearer <token> | Cookie: falcon_auth_token

{
  "type": "system_notification",
  "data": {
    "title": "System Maintenance",
    "message": "The system will be offline for maintenance from 02:00 to 04:00 UTC",
    "severity": "warning",
    "scheduled_time": "2025-09-05T02:00:00Z"
  }
}
```

#### Send Direct Message to Connection
```
POST /websocket/connections/{connection_id}/message
Authorization: Bearer <token> | Cookie: falcon_auth_token

{
  "type": "message",
  "data": {
    "text": "Hello from admin",
    "priority": "high"
  }
}
```

#### Send Message to User Connections
```
POST /websocket/users/{user_id}/message
Authorization: Bearer <token> | Cookie: falcon_auth_token

{
  "type": "notification",
  "data": {
    "title": "System Alert",
    "message": "Your account requires attention",
    "action_url": "/profile"
  }
}
```

#### Send Message to Room
```
POST /websocket/rooms/{room_id}/message
Authorization: Bearer <token> | Cookie: falcon_auth_token

{
  "type": "room_update",
  "data": {
    "action": "room_announcement",
    "title": "Room Maintenance",
    "message": "This room will be offline for maintenance at 3 PM UTC"
  }
}
```

**Message Body Structure:**
All admin messaging endpoints use the standardized `MessageBody` with:
- `type`: One of the 10 standardized message types (dropdown in OpenAPI docs)
- `data`: Flexible object containing message-specific data
- Complete OpenAPI 3.1.1 documentation with examples and validation

### Module Status Endpoint
```
GET /websocket/status
```
**Public endpoint** returning WebSocket module health and statistics

## Authentication Integration

### JWT Authentication Flow
1. **Connection Request**: Client requests WebSocket upgrade with JWT token
2. **Token Validation**: Existing `pkg/middleware/auth` system validates token
3. **User Context**: Extract user ID, character ID, and character name
4. **Room Assignment**: Automatic assignment to personal and group rooms
5. **Connection Tracking**: Connection added to manager with metadata

### Supported Authentication Methods
- **Bearer Token**: `Authorization: Bearer <jwt_token>`
- **Cookie Authentication**: `Cookie: falcon_auth_token=<jwt_token>`
- **Mixed Support**: Fallback from Bearer to Cookie authentication

## Room Management System

### Automatic Room Assignment

#### Personal Rooms
- **Format**: `user:{user_id}`
- **Purpose**: Private communication channel for user-specific updates
- **Membership**: Only connections from the same user
- **Creation**: Automatic on first connection

#### Group Rooms  
- **Format**: `group:{group_id}`
- **Purpose**: Communication channel for group members
- **Membership**: All active connections from group members
- **Creation**: Automatic when group members connect
- **Updates**: Real-time membership changes when group assignments change

### Room Assignment Logic
```go
// On connection establishment:
1. Join personal room: user:{user_id}
2. Query user's active group memberships from groups module
3. Join each group room: group:{group_id}
4. Store room assignments in connection metadata

// On group membership change:
1. Detect group membership change event
2. Update room assignments for all user's connections
3. Join/leave appropriate group rooms
4. Broadcast membership change notifications
```

## Redis Multi-Instance Support

### Redis Pub/Sub Channels
- **`websocket:messages`**: General message broadcasting
- **`websocket:rooms`**: Room-specific message broadcasting  
- **`websocket:users`**: User-specific message broadcasting
- **`websocket:system`**: System-wide notifications

### Message Distribution Flow
```
Instance A: User updates profile
    ↓
Instance A: Publishes to redis:websocket:users
    ↓
Instance B,C,D: Receive Redis message
    ↓
Instance B,C,D: Broadcast to local user connections
```

### Server Instance Identification
Each server instance gets a unique UUID to prevent message loops and enable distributed coordination.

## Integration Points

### User Module Integration
- **Profile Updates**: Real-time broadcasting when user profiles change
- **Character Management**: Connection tracking per character
- **Multi-Character Support**: Users with multiple characters get aggregated permissions

### Groups Module Integration  
- **Membership Queries**: Real-time querying of user group memberships
- **Room Assignment**: Automatic room assignment based on group membership
- **Change Notifications**: Live updates when group memberships change

### Auth Module Integration
- **JWT Validation**: Seamless integration with existing authentication system
- **Token Refresh**: Support for token refresh during long connections
- **Permission Checking**: Integration with permission system for admin operations

## Health Monitoring & Metrics

### Connection Health Tracking
- **Heartbeat Monitoring**: 30-second ping intervals with 60-second timeout
- **Connection Cleanup**: Automatic removal of inactive connections every 5 minutes
- **Health Checks**: Redis connectivity monitoring

### Statistics Tracking
```go
type WebSocketStats struct {
    TotalConnections   int       // Lifetime connection count
    ActiveConnections  int       // Current active connections
    TotalRooms         int       // Number of active rooms
    MessagesProcessed  int64     // Total messages processed
    MessagesBroadcast  int64     // Total messages broadcast
    LastConnectionTime time.Time // Most recent connection timestamp
}
```

### Module Status Endpoint
- **Health Status**: healthy/unhealthy based on Redis connectivity
- **Connection Statistics**: Real-time connection and room counts
- **Performance Metrics**: Message processing statistics

## Error Handling

### HTTP Status Codes
- **101 Switching Protocols**: Successful WebSocket upgrade
- **401 Unauthorized**: Invalid or missing authentication token
- **403 Forbidden**: Insufficient permissions (admin endpoints)
- **404 Not Found**: Connection or room not found
- **500 Internal Server Error**: Redis connectivity or system errors

### Connection Error Recovery
- **Automatic Reconnection**: Client-side reconnection logic recommended
- **Connection Recovery**: New connections automatically rejoin appropriate rooms
- **Message Queuing**: Critical messages can be queued in Redis for offline users

## Configuration

### Environment Variables

#### Required Configuration
- **`MONGODB_URI`**: MongoDB connection string for user/group data queries
- **`REDIS_URL`**: Redis connection string for pub/sub and connection metadata

#### WebSocket-Specific Configuration
```bash
# WebSocket Configuration
WEBSOCKET_URL=wss://localhost:3000/websocket/connect         # Full client connection URL (secure)
WEBSOCKET_PATH=/websocket/connect                            # Server routing path
WEBSOCKET_ALLOWED_ORIGINS=https://yourdomain.com,http://localhost:3000,https://localhost:3000  # Allowed origins
```

**Environment Variable Details:**
- **`WEBSOCKET_URL`**: Complete URL that clients use to connect (includes `ws://` or `wss://` protocol)
- **`WEBSOCKET_PATH`**: Internal server path for HTTP handler registration (path only)
- **`WEBSOCKET_ALLOWED_ORIGINS`**: Comma-separated list of allowed origins for CORS security

**Development vs Production Examples:**
```bash
# Development
WEBSOCKET_URL=wss://localhost:3000/websocket/connect
WEBSOCKET_PATH=/websocket/connect
WEBSOCKET_ALLOWED_ORIGINS=https://localhost:3000,http://localhost:3000

# Production (SSL)
WEBSOCKET_URL=wss://api.yourdomain.com/websocket/connect
WEBSOCKET_PATH=/websocket/connect
WEBSOCKET_ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com

# Production (Different Port)
WEBSOCKET_URL=wss://websockets.yourdomain.com:8443/connect
WEBSOCKET_PATH=/connect
WEBSOCKET_ALLOWED_ORIGINS=https://yourdomain.com
```

### Module Configuration
```go
// Module initialization
websocketModule, err := websocket.NewModule(db, redisClient, authMiddleware)
if err != nil {
    log.Fatal(err)
}

// Register API routes
websocketModule.RegisterUnifiedRoutes(api)

// Register WebSocket handler
websocketModule.RegisterHTTPHandler(mux)
```

## Performance Considerations

### Scalability Features
- **Horizontal Scaling**: Redis pub/sub enables unlimited server instances
- **Connection Pooling**: Efficient memory usage with connection recycling
- **Room Optimization**: Automatic cleanup of empty rooms
- **Message Batching**: Efficient broadcasting with minimal Redis calls

### Memory Management
- **TTL-Based Storage**: Redis keys expire automatically to prevent memory leaks
- **Concurrent Safety**: All operations use mutex protection for thread safety
- **Connection Limits**: Configurable connection limits per user/instance

## Security Considerations

### Authentication Security
- **JWT Validation**: Full token validation including expiration and signature
- **Origin Checking**: Configurable CORS origin validation with `WEBSOCKET_ALLOWED_ORIGINS`
- **Secure Connections**: WSS (WebSocket Secure) support for encrypted communication
- **Rate Limiting**: Connection establishment rate limiting (planned)

### Message Security
- **Room Isolation**: Users only receive messages from rooms they belong to
- **Admin Separation**: Administrative operations require super admin privileges
- **Input Validation**: All message payloads validated before processing

## Development Guidelines

### Adding New Message Types
1. **Define Type**: Add new `MessageType` constant in `models/models.go`
2. **Handler Logic**: Implement message handling in `services/connection.go`
3. **Broadcasting**: Add Redis pub/sub support in `services/redis.go`
4. **Documentation**: Update message type documentation

### Integration with Other Modules
1. **Event Publishing**: Use module's `BroadcastUserProfileUpdate` or `BroadcastGroupMembershipChange` methods
2. **Custom Events**: Create custom message types for module-specific events
3. **Room Management**: Leverage existing room system for module-specific channels

### Testing WebSocket Connections

#### Development Testing
```bash
# Using wscat (install with: npm install -g wscat)
wscat -c "wss://localhost:3000/websocket/connect" \
  -H "Authorization: Bearer your-jwt-token"

# Using environment-configured URL
wscat -c "$WEBSOCKET_URL" \
  -H "Authorization: Bearer your-jwt-token"

# Using curl for API endpoints
curl -H "Authorization: Bearer token" \
  http://localhost:3000/websocket/status
```

#### Client Connection Example
```javascript
const token = 'your-jwt-token';
// Get WebSocket URL from server configuration endpoint or environment
const websocketUrl = process.env.WEBSOCKET_URL || 'wss://localhost:3000/websocket/connect';

const ws = new WebSocket(websocketUrl, [], {
    headers: {
        'Authorization': `Bearer ${token}`
    }
});

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
};

// Send heartbeat
ws.send(JSON.stringify({
    type: 'heartbeat',
    timestamp: new Date().toISOString()
}));
```

## Future Enhancements

### Planned Features
- **Message Persistence**: Store offline messages in Redis for later delivery
- **Connection Rate Limiting**: Per-user connection limits and rate limiting
- **WebSocket Compression**: Protocol-level compression for large messages  
- **Advanced Admin Interface**: Web-based connection and room management
- **Message History**: Room message history storage and retrieval

### Advanced Features
- **Voice Channel Support**: Integration with voice communication systems
- **File Sharing**: Secure file sharing through WebSocket channels
- **Screen Sharing**: Real-time screen sharing capabilities
- **Mobile Push Integration**: Fallback to push notifications when WebSocket unavailable

## Troubleshooting

### Common Issues

#### Connection Refused
- **Check Authentication**: Verify JWT token is valid and not expired
- **Check Redis**: Ensure Redis is running and accessible
- **Check Ports**: Verify WebSocket port is not blocked by firewall

#### Messages Not Received
- **Check Room Membership**: Verify user is in the expected rooms
- **Check Redis Channels**: Monitor Redis pub/sub channels for message flow
- **Check Connection Health**: Verify connection is still alive with heartbeat

#### Performance Issues  
- **Monitor Connection Count**: Check active connection statistics
- **Redis Performance**: Monitor Redis memory usage and latency
- **Message Volume**: Check message processing statistics for bottlenecks

### Debug Logging
Enable debug logging to trace WebSocket operations:
```bash
# Environment variable
WEBSOCKET_DEBUG=true

# Or programmatically
logging.SetLevel("debug")
```

## Dependencies

### Internal Dependencies
- `go-falcon/internal/auth/models` (authentication models)
- `go-falcon/internal/groups/models` (group models for room assignment)
- `go-falcon/pkg/middleware` (authentication middleware)
- `go-falcon/pkg/logging` (structured logging)

### External Dependencies
- `github.com/gorilla/websocket` (WebSocket protocol implementation)
- `github.com/redis/go-redis/v9` (Redis client for pub/sub)
- `github.com/danielgtaylor/huma/v2` (API framework integration)
- `go.mongodb.org/mongo-driver` (MongoDB for user/group queries)

## Contributing

1. **Follow Module Patterns**: Use established go-falcon module structure
2. **Add Tests**: Include unit tests for new functionality
3. **Update Documentation**: Keep CLAUDE.md updated with changes
4. **Security Review**: Ensure new features maintain security standards
5. **Performance Testing**: Test with realistic connection loads

This WebSocket module provides a comprehensive real-time communication foundation for the go-falcon API gateway, with secure authentication, intelligent room management, and horizontal scalability built-in.