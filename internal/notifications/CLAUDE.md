# Notifications Module (internal/notifications)

## Overview

The notifications module provides a comprehensive notification system for the Go Falcon application. It enables users to receive, manage, and track notifications across different channels with support for real-time delivery and persistence.

## Architecture

### Core Components

- **Module**: Main notifications module implementing the base module interface
- **Message Management**: Create, read, update, and delete notifications
- **Delivery Systems**: Support for multiple notification channels
- **Permission Integration**: Granular permission control for notification access
- **Background Processing**: Automated notification cleanup and delivery

### Files Structure

```
internal/notifications/
├── notifications.go      # Main module with API handlers and route registration
├── handlers.go          # HTTP handlers for notification operations (planned)
├── models.go            # Notification data structures and database models (planned)
├── service.go           # Business logic for notification operations (planned)
└── CLAUDE.md            # This documentation file
```

## Features

### Notification Management
- **Message Creation**: Create notifications with different types and priorities
- **Message Retrieval**: Fetch notifications with filtering and pagination
- **Message Updates**: Mark notifications as read/unread, update status
- **Message Deletion**: Remove notifications with proper authorization

### Delivery Channels (Planned)
- **In-App Notifications**: Real-time web notifications via WebSocket
- **Email Notifications**: SMTP-based email delivery
- **Discord Integration**: Discord webhook and bot notifications
- **Push Notifications**: Mobile push notification support

### Notification Types (Planned)
- **System Notifications**: Application updates, maintenance notices
- **User Notifications**: Personal messages, account updates
- **Alert Notifications**: Security alerts, error notifications
- **Event Notifications**: Scheduled task completions, system events

## API Endpoints

| Endpoint | Method | Description | Permission Required |
|----------|--------|-------------|-------------------|
| `/notifications` | GET | Get user's notifications with filtering | `notifications.messages.read` |
| `/notifications` | POST | Send a new notification | `notifications.messages.write` |
| `/notifications/{id}` | PUT | Mark notification as read/update status | `notifications.messages.write` |
| `/notifications/{id}` | DELETE | Delete specific notification | `notifications.messages.write` |
| `/notifications/bulk` | POST | Bulk operations on notifications | `notifications.messages.write` |
| `/notifications/stats` | GET | Get notification statistics | `notifications.messages.read` |
| `/health` | GET | Module health check | None (public) |

### Example API Calls

#### Get Notifications
```bash
curl -H "Authorization: Bearer <jwt_token>" \
     http://localhost:8080/notifications?page=1&page_size=20&unread_only=true
```

**Response:**
```json
{
  "notifications": [
    {
      "id": "notification_id",
      "type": "system",
      "title": "System Maintenance",
      "message": "Scheduled maintenance will occur...",
      "priority": "high",
      "read": false,
      "created_at": "2024-01-15T10:30:00Z",
      "expires_at": "2024-01-22T10:30:00Z"
    }
  ],
  "total": 15,
  "unread_count": 3,
  "page": 1,
  "page_size": 20
}
```

#### Send Notification
```bash
curl -X POST \
     -H "Authorization: Bearer <jwt_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "type": "alert",
       "title": "Security Alert",
       "message": "Unusual login activity detected",
       "priority": "high",
       "recipients": ["character_id_1", "character_id_2"],
       "expires_at": "2024-01-22T10:30:00Z"
     }' \
     http://localhost:8080/notifications
```

#### Mark as Read
```bash
curl -X PUT \
     -H "Authorization: Bearer <jwt_token>" \
     -H "Content-Type: application/json" \
     -d '{"read": true}' \
     http://localhost:8080/notifications/notification_id_123
```

## Database Schema (Planned)

### Notifications Collection
```javascript
{
  "_id": "ObjectId",
  "notification_id": "uuid-string",
  "type": "system|user|alert|event",
  "title": "Notification title",
  "message": "Notification content",
  "priority": "low|normal|high|critical",
  "sender_id": "character_id (optional)",
  "recipients": [
    {
      "character_id": "number",
      "read": "boolean",
      "read_at": "timestamp (optional)"
    }
  ],
  "channels": ["in_app", "email", "discord"],
  "metadata": {
    "source": "module_name",
    "category": "category_name",
    "action_url": "url (optional)",
    "action_text": "button_text (optional)"
  },
  "delivery_status": {
    "in_app": "sent|delivered|failed",
    "email": "pending|sent|delivered|failed",
    "discord": "pending|sent|delivered|failed"
  },
  "created_at": "timestamp",
  "expires_at": "timestamp (optional)",
  "deleted_at": "timestamp (optional)"
}
```

### Notification Preferences Collection (Planned)
```javascript
{
  "_id": "ObjectId",
  "character_id": "number",
  "preferences": {
    "email_notifications": "boolean",
    "discord_notifications": "boolean",
    "push_notifications": "boolean",
    "notification_types": {
      "system": "boolean",
      "alerts": "boolean",
      "events": "boolean"
    },
    "quiet_hours": {
      "enabled": "boolean",
      "start_time": "HH:MM",
      "end_time": "HH:MM",
      "timezone": "string"
    }
  },
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

## Security and Permissions

### Granular Permission System

The notifications module implements comprehensive permission control for secure notification management:

#### Service: `notifications`

##### Resource: `messages`
- **read**: View and retrieve notifications
- **write**: Send notifications, mark as read, and manage notification status

### Required Group Configuration

To use the notifications module, configure the following groups:

#### Administrators Group
```json
{
  "name": "administrators",
  "permissions": {
    "notifications": {
      "messages": ["read", "write"]
    }
  }
}
```

#### General Users Group
```json
{
  "name": "general_users",
  "permissions": {
    "notifications": {
      "messages": ["read", "write"]
    }
  }
}
```

#### Notification Managers Group (Optional)
```json
{
  "name": "notification_managers",
  "permissions": {
    "notifications": {
      "messages": ["read", "write"]
    }
  }
}
```

### Permission Requirements by Endpoint

| Endpoint Category | Permission | Description |
|------------------|------------|-------------|
| Get Notifications | `notifications.messages.read` | View user's own notifications |
| Send Notifications | `notifications.messages.write` | Create and send notifications |
| Update Notifications | `notifications.messages.write` | Mark as read, update status |
| Delete Notifications | `notifications.messages.write` | Remove notifications |

### Security Features

- **User Isolation**: Users can only access their own notifications
- **Permission-Based Sending**: Sending notifications requires write permissions
- **Data Privacy**: Notification content protected by authentication
- **Audit Trail**: All notification actions logged with timestamps

## Integration Examples

### Module Dependencies
The notifications module integrates with other modules through interfaces:

```go
// GroupsModule interface for permission checking
type GroupsModule interface {
    RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler
}

// Module structure
type Module struct {
    *module.BaseModule
    groupsModule GroupsModule
}
```

### Sending Notifications from Other Modules (Planned)
```go
// Interface for sending notifications
type NotificationService interface {
    SendNotification(ctx context.Context, notification *Notification) error
    SendBulkNotifications(ctx context.Context, notifications []*Notification) error
    GetUserNotifications(ctx context.Context, characterID int, options *NotificationOptions) (*NotificationResponse, error)
}

// Example usage from scheduler module
func (s *SchedulerModule) notifyTaskCompletion(taskID string, success bool) error {
    notification := &Notification{
        Type:      "event",
        Title:     "Task Completed",
        Message:   fmt.Sprintf("Task %s has completed", taskID),
        Priority:  "normal",
        Recipients: []string{"admin_character_id"},
        Metadata: map[string]interface{}{
            "source": "scheduler",
            "task_id": taskID,
            "success": success,
        },
    }
    
    return s.notificationService.SendNotification(ctx, notification)
}
```

## Background Tasks

### Module Background Processing
- **Cleanup Tasks**: Remove expired notifications and maintain database size
- **Delivery Retry**: Retry failed notification deliveries
- **Statistics**: Generate notification usage statistics
- **Preference Sync**: Synchronize user preferences across channels

### Cleanup Operations (Planned)
```go
// Daily cleanup of expired notifications
Schedule: "0 2 * * *"  // 2 AM daily

// Retry failed deliveries
Schedule: "*/15 * * * *"  // Every 15 minutes

// Generate daily statistics
Schedule: "0 1 * * *"  // 1 AM daily
```

## Error Handling

### Common Error Scenarios
- **Permission Denied**: User lacks required notification permissions
- **Notification Not Found**: Invalid notification ID or deleted notification
- **Invalid Recipients**: Specified recipients don't exist
- **Delivery Failures**: External service failures (email, Discord)
- **Rate Limiting**: Too many notifications sent in short period

### Error Response Format
```json
{
  "error": "permission_denied",
  "message": "Insufficient permissions to send notifications",
  "code": 403
}
```

## Configuration (Planned)

### Environment Variables
```bash
# Notification settings
NOTIFICATIONS_ENABLED=true
NOTIFICATIONS_DEFAULT_EXPIRY=7d
NOTIFICATIONS_MAX_RECIPIENTS=100
NOTIFICATIONS_RATE_LIMIT=60  # per hour per user

# Email configuration
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=notifications@example.com
SMTP_PASSWORD=password
SMTP_FROM_ADDRESS=notifications@example.com

# Discord webhook
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...

# Push notification service
PUSH_SERVICE_URL=https://push.example.com
PUSH_SERVICE_TOKEN=token
```

## Performance Considerations

### Database Optimization
- **Indexes**: Created on character_id, created_at, read status
- **Pagination**: Efficient pagination for large notification lists
- **Cleanup**: Automatic removal of expired notifications
- **Archival**: Move old notifications to archive collection

### Caching Strategy (Planned)
- **Unread Counts**: Cache unread notification counts in Redis
- **Recent Notifications**: Cache recent notifications for faster access
- **User Preferences**: Cache notification preferences
- **Delivery Status**: Cache delivery attempts and status

## Future Enhancements

### Planned Features
- **Real-time Notifications**: WebSocket support for instant delivery
- **Notification Templates**: Predefined templates for common notifications
- **Scheduling**: Schedule notifications for future delivery
- **Rich Content**: Support for HTML content, images, and attachments
- **Push Notifications**: Mobile push notification support
- **Analytics**: Detailed notification analytics and reporting

### API Extensions
- **Subscription Management**: Subscribe/unsubscribe from notification types
- **Delivery Preferences**: User-configurable delivery preferences
- **Notification History**: Complete notification history with search
- **Export Functionality**: Export notifications for archival

### Integration Improvements
- **Webhook Support**: Incoming webhooks for external notifications
- **API Integration**: REST API for external services to send notifications
- **Batch Processing**: Improved bulk notification processing
- **Multi-tenant Support**: Namespace notifications by organization

## Development Notes

### Current Implementation Status
The notifications module is currently in basic implementation with:
- ✅ Basic module structure and routes
- ✅ Granular permission integration
- ✅ Placeholder handlers for development
- ❌ Full CRUD operations (planned)
- ❌ Database models and persistence (planned)
- ❌ Multi-channel delivery (planned)
- ❌ Real-time features (planned)

### Implementation Priority
1. **Database Models**: Define notification and preference schemas
2. **CRUD Operations**: Complete notification management functionality
3. **User Interface**: Basic notification list and management
4. **Email Delivery**: SMTP-based email notifications
5. **Real-time Updates**: WebSocket integration
6. **Mobile Support**: Push notification implementation

## Testing Strategy

### Unit Tests
- Notification creation and validation
- Permission checking and authorization
- Database operations and queries
- Background task processing

### Integration Tests
- End-to-end notification workflows
- Multi-channel delivery testing
- Performance testing with large notification volumes
- Error handling and recovery scenarios

### Load Testing
- High-volume notification sending
- Concurrent user notification retrieval
- Database performance under load
- Real-time delivery performance

This module provides a solid foundation for notification management in the Go Falcon application, with comprehensive permission control and extensive expansion capabilities for future requirements.