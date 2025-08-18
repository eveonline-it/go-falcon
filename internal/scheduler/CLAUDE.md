# Scheduler Module (internal/scheduler)

## Overview

The scheduler module provides a comprehensive task scheduling and management system for the Go-Falcon monolith. It supports both hardcoded system tasks and dynamic user-defined tasks with cron-like scheduling, worker pool execution, distributed locking, and complete lifecycle management.

## Architecture

### Core Components

- **Module**: Main scheduler module implementing the base module interface
- **Engine**: Task execution engine with worker pool and cron scheduling
- **Repository**: Database operations for tasks and execution history
- **Hardcoded Tasks**: Predefined system tasks for critical operations
- **Task Types**: Support for HTTP, function, system, and custom tasks
- **Distributed Locking**: Redis-based locking to prevent duplicate executions
- **Module Integration**: Direct integration with auth module for token refresh operations

### Files Structure

```
internal/scheduler/
├── scheduler.go      # Main module with API handlers
├── engine.go         # Task execution engine and worker pool
├── repository.go     # Database operations and persistence
├── types.go          # Type definitions and data structures
├── executors.go      # Task execution implementations
├── hardcoded.go      # Hardcoded system tasks definitions
└── CLAUDE.md         # This documentation file
```

## Features

### Task Management
- **CRUD Operations**: Complete task lifecycle management
- **Task Types**: HTTP requests, function calls, system tasks, and custom executors
- **Flexible Scheduling**: Cron expressions for complex scheduling needs
- **Priority System**: Low, normal, high, and critical priority levels
- **Status Management**: Pending, running, completed, failed, paused, disabled states

### Execution Engine
- **Worker Pool**: Configurable concurrent task execution
- **Cron Scheduling**: Standard cron expressions (6-field format with seconds)
- **Distributed Safe**: Redis-based locking prevents duplicate runs
- **Retry Logic**: Configurable retry attempts with exponential backoff
- **Timeout Handling**: Per-task execution timeouts

### Monitoring & History
- **Execution History**: Complete audit trail of task runs
- **Statistics**: Success/failure rates, average runtime, execution counts
- **Health Monitoring**: Stale task detection and cleanup
- **Performance Metrics**: Worker utilization and queue statistics

## Task Types

### System Tasks (Hardcoded)
Predefined tasks that are automatically created and managed by the scheduler. These tasks are defined in `hardcoded.go` and handle critical system operations:

- **EVE Token Refresh** (`system-token-refresh`)
  - Schedule: Every 15 minutes
  - Refreshes expired EVE Online access tokens using the auth module
  - High priority with 3 retry attempts
  - Processes tokens in configurable batches (default: 100 users)
  - Uses `AuthModule.RefreshExpiringTokens()` for actual implementation

- **State Cleanup** (`system-state-cleanup`)
  - Schedule: Every 2 hours
  - Cleans up expired states and temporary data
  - Normal priority with configurable retention

- **Health Check** (`system-health-check`)
  - Schedule: Every 5 minutes
  - Monitors system health (MongoDB, Redis, ESI)
  - Normal priority with quick timeout

- **Task History Cleanup** (`system-task-cleanup`)
  - Schedule: Daily at 2 AM
  - Removes old task execution records
  - Low priority with 30-day retention

#### Managing System Tasks
System tasks are defined in `hardcoded.go` and include:
- **Task Definitions**: Complete task configuration with schedules, priorities, and metadata
- **Task Metadata**: Comprehensive information about each system task via `SystemTaskDefinitions`
- **Modification**: To add, modify, or remove system tasks, edit the `getSystemTasks()` function
- **Protection**: System tasks cannot be modified or deleted via the API for security reasons

### HTTP Tasks
Execute HTTP requests with full configuration:

```json
{
  "type": "http",
  "config": {
    "url": "https://api.example.com/endpoint",
    "method": "POST",
    "headers": {
      "Content-Type": "application/json",
      "Authorization": "Bearer token"
    },
    "body": "{\"key\": \"value\"}",
    "expected_code": 200,
    "timeout": "30s",
    "follow_redirect": true,
    "validate_ssl": true
  }
}
```

### Function Tasks
Execute internal Go functions:

```json
{
  "type": "function",
  "config": {
    "function_name": "processData",
    "module": "analytics",
    "parameters": {
      "batch_size": 1000,
      "timeout": "5m"
    }
  }
}
```

### Custom Tasks
User-defined task executors with flexible configuration:

```json
{
  "type": "custom",
  "config": {
    "executor": "custom_processor",
    "parameters": {
      "custom_param": "value"
    }
  }
}
```

## API Endpoints

| Endpoint | Method | Description | Permission Required |
|----------|--------|-------------|-------------------|
| `/scheduler/status` | GET | Get scheduler status | `scheduler.read` (CASBIN protected) |
| `/scheduler/stats` | GET | Get scheduler statistics | None (public) |
| `/scheduler/tasks` | GET | List tasks with filtering and pagination | `scheduler.tasks.read` |
| `/scheduler/tasks` | POST | Create new task | `scheduler.tasks.write` |
| `/scheduler/tasks/{id}` | GET | Get specific task details | `scheduler.tasks.read` |
| `/scheduler/tasks/{id}` | PUT | Update task configuration | `scheduler.tasks.write` |
| `/scheduler/tasks/{id}` | DELETE | Delete task (system tasks protected) | `scheduler.tasks.delete` |
| `/scheduler/tasks/{id}/start` | POST | Manually execute task immediately | `scheduler.tasks.execute` |
| `/scheduler/tasks/{id}/stop` | POST | Stop currently running task | `scheduler.tasks.execute` |
| `/scheduler/tasks/{id}/pause` | POST | Pause task scheduling | `scheduler.tasks.write` |
| `/scheduler/tasks/{id}/resume` | POST | Resume paused task | `scheduler.tasks.write` |
| `/scheduler/tasks/{id}/history` | GET | Get task execution history | `scheduler.executions.read` |
| `/scheduler/tasks/{id}/executions/{exec_id}` | GET | Get specific execution details | `scheduler.executions.read` |
| `/scheduler/reload` | POST | Reload tasks from database | `scheduler.tasks.admin` |

### API Examples

#### Create HTTP Task
```bash
curl -X POST /scheduler/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "API Health Check",
    "description": "Check external API health",
    "type": "http",
    "schedule": "0 */10 * * * *",
    "priority": "normal",
    "enabled": true,
    "config": {
      "url": "https://api.example.com/health",
      "method": "GET",
      "expected_code": 200,
      "timeout": "30s"
    },
    "tags": ["monitoring", "health"]
  }'
```

#### List Tasks with Filtering
```bash
curl "/scheduler/tasks?status=running&type=http&page=1&page_size=20"
```

#### Get Task Execution History
```bash
curl "/scheduler/tasks/task-id-123/history?page=1&page_size=50"
```

#### Manually Execute Task
```bash
curl -X POST "/scheduler/tasks/task-id-123/start"
```

## Configuration

### Environment Variables
```bash
# Redis Configuration (for distributed locking)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# MongoDB Configuration (for task storage)
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=go_falcon

# Scheduler Configuration
SCHEDULER_WORKER_COUNT=10          # Number of concurrent workers
SCHEDULER_QUEUE_SIZE=1000          # Task queue buffer size
SCHEDULER_CLEANUP_INTERVAL=1h      # How often to run cleanup
SCHEDULER_STALE_TIMEOUT=2h         # When to mark running tasks as stale
```

### Task Scheduling Format
Uses standard 6-field cron expressions with seconds:

```
┌───────────── second (0 - 59)
│ ┌───────────── minute (0 - 59)
│ │ ┌───────────── hour (0 - 23)
│ │ │ ┌───────────── day of month (1 - 31)
│ │ │ │ ┌───────────── month (1 - 12)
│ │ │ │ │ ┌───────────── day of week (0 - 6) (Sunday to Saturday)
│ │ │ │ │ │
* * * * * *
```

**Examples:**
- `0 */5 * * * *` - Every 5 minutes (at second 0)
- `0 0 */2 * * *` - Every 2 hours (at minute 0, second 0)
- `0 0 9 * * 1-5` - 9 AM on weekdays (at minute 0, second 0)
- `0 30 14 1 * *` - 2:30 PM on the 1st of every month (at second 0)

## Database Schema

### Tasks Collection (`scheduler_tasks`)
```javascript
{
  "_id": "task-uuid",
  "name": "Task Name",
  "description": "Task description",
  "type": "http|function|system|custom",
  "schedule": "0 */2 * * *",
  "status": "pending|running|completed|failed|paused|disabled",
  "priority": "low|normal|high|critical",
  "enabled": true,
  "config": {
    // Type-specific configuration
  },
  "metadata": {
    "max_retries": 3,
    "retry_interval": "2m",
    "timeout": "5m",
    "tags": ["monitoring", "system"],
    "is_system": false,
    "source": "api|system|import",
    "version": 1,
    "last_error": "Error message",
    "success_count": 150,
    "failure_count": 5,
    "total_runs": 155,
    "average_runtime": "1.5s"
  },
  "last_run": "2024-01-15T10:30:00Z",
  "next_run": "2024-01-15T12:30:00Z",
  "created_at": "2024-01-15T09:15:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "created_by": "system|user-id",
  "updated_by": "system|user-id"
}
```

### Executions Collection (`scheduler_executions`)
```javascript
{
  "_id": "execution-uuid",
  "task_id": "task-uuid",
  "status": "pending|running|completed|failed",
  "started_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:30:45Z",
  "duration": "45s",
  "output": "Task output or response",
  "error": "Error message if failed",
  "metadata": {
    "worker_id": "worker-1",
    "retry_count": 0,
    "lock_key": "lock:task-uuid"
  },
  "worker_id": "worker-1",
  "retry_count": 0
}
```

## Integration Examples

### Adding Custom Task Executor
```go
// Register custom executor
func (e *Engine) RegisterExecutor(taskType TaskType, executor TaskExecutor) {
    e.executors[taskType] = executor
}

// Custom executor implementation
type MyCustomExecutor struct{}

func (e *MyCustomExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    // Custom task logic here
    return &TaskResult{
        Success: true,
        Output:  "Custom task completed",
    }, nil
}

// Register during initialization
engine.RegisterExecutor("my_custom_type", &MyCustomExecutor{})
```

### Module Dependencies
The scheduler module integrates with other modules through interfaces to maintain loose coupling:

```go
// AuthModule interface for token refresh operations
type AuthModule interface {
    RefreshExpiringTokens(ctx context.Context, batchSize int) (successCount, failureCount int, err error)
}

// GroupsModule interface for permission checking
type GroupsModule interface {
    RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler
}

// SystemExecutor with auth module dependency
type SystemExecutor struct {
    authModule AuthModule
}
```

### Adding New System Tasks
To add a new hardcoded system task, edit `hardcoded.go`:

```go
// Add to getSystemTasks() function
{
    ID:          "system-my-new-task",
    Name:        "My New System Task",
    Description: "Description of what this task does",
    Type:        TaskTypeSystem,
    Schedule:    "0 0 6 * * *", // Daily at 6 AM
    Status:      TaskStatusPending,
    Priority:    TaskPriorityNormal,
    Enabled:     true,
    Config: map[string]interface{}{
        "task_name": "my_new_task",
        "parameters": map[string]interface{}{
            "param1": "value1",
            "param2": 42,
        },
    },
    Metadata: TaskMetadata{
        MaxRetries:    2,
        RetryInterval: 5 * time.Minute,
        Timeout:       15 * time.Minute,
        Tags:          []string{"system", "custom"},
        IsSystem:      true,
        Source:        "system",
        Version:       1,
    },
    CreatedAt: now,
    UpdatedAt: now,
    CreatedBy: "system",
}

// Add to SystemTaskDefinitions for documentation
"system-my-new-task": {
    Name:        "My New System Task",
    Description: "Description of what this task does",
    Schedule:    "Daily at 6 AM",
    Purpose:     "Explain why this task is needed",
    Priority:    "Normal",
},
```


### Monitoring Integration
```go
// Get scheduler statistics
stats, err := repository.GetSchedulerStats(ctx)
if err != nil {
    return err
}

// Check for failing tasks
failedTasks := stats.FailedToday
if failedTasks > threshold {
    // Send alert
    alerting.SendAlert("High task failure rate", stats)
}
```

## Error Handling

### Common Error Scenarios
- **Task Not Found**: Returns 404 for non-existent tasks
- **System Task Protection**: Returns 403 when trying to modify system tasks
- **Invalid Configuration**: Returns 400 for malformed task configs
- **Scheduling Conflicts**: Handles overlapping task executions
- **Database Failures**: Graceful degradation and retry logic

### Retry Logic
```go
type TaskMetadata struct {
    MaxRetries    int           `json:"max_retries"`     // 0-10
    RetryInterval time.Duration `json:"retry_interval"`  // 1s-1h
    Timeout       time.Duration `json:"timeout"`         // 30s-24h
}
```

### Distributed Locking
- Uses Redis for coordination across multiple instances
- Lock keys: `scheduler:lock:{task_id}`
- Automatic lock expiration based on task timeout
- Handles Redis failures gracefully

## Performance Considerations

### Worker Pool Sizing
```bash
# Conservative (low resource usage)
SCHEDULER_WORKER_COUNT=5

# Balanced (recommended)
SCHEDULER_WORKER_COUNT=10

# High throughput (more resources)
SCHEDULER_WORKER_COUNT=20
```

### Database Optimization
- **Indexes**: Created automatically on key fields
- **Cleanup**: Automatic removal of old execution records
- **Statistics**: Periodic aggregation of task metrics
- **Pagination**: Efficient pagination for large task lists

### Memory Management
- **Task Caching**: Active tasks cached in memory
- **Execution Cleanup**: Automatic cleanup of completed executions
- **Connection Pooling**: Efficient database connection usage

## Security Considerations

### Task Execution
- **Input Validation**: All task configurations validated
- **Resource Limits**: Timeout and retry limits enforced
- **System Task Protection**: System tasks cannot be modified via API
- **Error Sanitization**: Sensitive data removed from error messages

### API Security
- **Authentication**: Integration with auth module and JWT middleware
- **Authorization**: Granular permission-based access control
- **Input Sanitization**: All API inputs validated and sanitized
- **System Task Protection**: System tasks cannot be modified via API

### Granular Permission System

The scheduler module implements a comprehensive granular permission system with the following permissions:

#### Service: `scheduler`

##### Global Permissions
- **read**: View scheduler status and basic information (required for `/scheduler/status`)

##### Resource: `tasks`
- **read**: View task details and list tasks
- **write**: Create, update, pause, and resume tasks
- **delete**: Delete tasks (system tasks always protected)
- **execute**: Manually trigger task execution
- **admin**: Reload tasks from database and advanced operations

##### Resource: `executions`
- **read**: View task execution history and details

### CASBIN Authorization Integration

The `/scheduler/status` endpoint is now protected with CASBIN authorization middleware:

**Debug Logging Example:**
```bash
[DEBUG] SchedulerRoutes: /status endpoint called
[DEBUG] CasbinAuthMiddleware.RequirePermission: Checking scheduler.read for GET /scheduler/status
[DEBUG] CasbinAuthMiddleware: Found authenticated user (user:test-user)
[DEBUG] CasbinAuthMiddleware: Checking permission 'scheduler.read' for subjects: [user:test-user, character:123456]
[DEBUG] CasbinAuthMiddleware: Permission denied for subject user:test-user
[DEBUG] CasbinAuthMiddleware: Permission denied for subject character:123456
[DEBUG] CasbinAuthMiddleware: No explicit allow found, defaulting to deny
[DEBUG] CasbinAuthMiddleware: Permission denied for user test-user
```

**Required Policy Setup:**
```bash
# Grant scheduler.read permission to appropriate roles
casbin.AddPolicy("role:admin", "scheduler", "read", "allow")
casbin.AddPolicy("role:monitoring", "scheduler", "read", "allow")

# Assign roles to users
casbin.AddRoleForUser("user:12345", "role:admin")
```

### Required Group Configuration

To use the scheduler module, the following groups should be configured:

#### Administrators Group
```json
{
  "name": "administrators",
  "permissions": {
    "scheduler": {
      "global": ["read"],
      "tasks": ["read", "write", "delete", "execute", "admin"],
      "executions": ["read"]
    }
  }
}
```

#### Task Managers Group
```json
{
  "name": "task_managers", 
  "permissions": {
    "scheduler": {
      "global": ["read"],
      "tasks": ["read", "write", "execute"],
      "executions": ["read"]
    }
  }
}
```

#### Monitoring Group
```json
{
  "name": "monitoring",
  "permissions": {
    "scheduler": {
      "global": ["read"],
      "tasks": ["read"],
      "executions": ["read"]
    }
  }
}
```

## Monitoring & Alerting

### Metrics Available
- Total tasks, enabled tasks, running tasks
- Daily completion and failure counts
- Average execution times
- Worker utilization and queue sizes
- Next scheduled run times

### Health Checks
```bash
# Check scheduler status (requires authentication and scheduler.read permission)
curl -H "Authorization: Bearer <token>" /scheduler/status
# OR with cookie authentication
curl -H "Cookie: falcon_auth_token=<token>" /scheduler/status

# Get comprehensive statistics
curl /scheduler/stats

# Verify specific task
curl /scheduler/tasks/task-id/history
```

### Alerting Integration
- High failure rates detection
- Stale task monitoring
- Resource utilization alerts
- System task failure notifications

## Best Practices

### Task Design
1. **Idempotent Tasks**: Design tasks to be safely re-runnable
2. **Timeout Setting**: Set appropriate timeouts for task complexity
3. **Error Handling**: Implement proper error handling and logging
4. **Resource Cleanup**: Ensure tasks clean up resources properly

### Scheduling
1. **Avoid Overlaps**: Use appropriate intervals to prevent task overlaps
2. **Distribute Load**: Spread tasks across different times
3. **Priority Usage**: Use priorities appropriately (critical sparingly)
4. **Testing**: Test cron expressions before deployment

### Monitoring
1. **Regular Review**: Monitor task success/failure rates
2. **Performance Tracking**: Track execution times and optimize
3. **Cleanup**: Regularly review and remove obsolete tasks
4. **Alerting**: Set up alerts for critical task failures

## Troubleshooting

### Common Issues

**Tasks Not Running**
- Check if task is enabled
- Verify cron schedule syntax
- Check worker availability
- Review execution history for errors

**High Failure Rates**
- Review task configurations
- Check external dependencies
- Verify timeout settings
- Examine error logs

**Performance Issues**
- Monitor worker utilization
- Review database performance
- Check Redis connectivity
- Analyze task execution times

### Debugging Commands
```bash
# Get task details
curl /scheduler/tasks/task-id

# Check execution history
curl /scheduler/tasks/task-id/history

# View scheduler statistics
curl /scheduler/stats

# Manually trigger task
curl -X POST /scheduler/tasks/task-id/start
```

## Dependencies

### External Services
- **MongoDB**: Task definitions and execution history
- **Redis**: Distributed locking and coordination
- **EVE Online ESI**: For EVE-related system tasks (via auth module)

### Go Packages
- `github.com/robfig/cron/v3` - Cron expression parsing and scheduling
- `github.com/go-chi/chi/v5` - HTTP routing
- `go.mongodb.org/mongo-driver` - MongoDB client
- `github.com/go-redis/redis/v8` - Redis client
- `github.com/google/uuid` - UUID generation

### Internal Dependencies
- `go-falcon/pkg/module` - Base module interface
- `go-falcon/pkg/database` - Database connections
- `go-falcon/pkg/handlers` - HTTP utilities
- `go-falcon/pkg/config` - Configuration management
- `go-falcon/internal/auth` - Auth module interface for token refresh operations

## Future Enhancements

### Planned Features
- **Web UI**: Browser-based task management interface
- **Task Templates**: Predefined task templates for common operations
- **Workflow Support**: Multi-task workflows with dependencies
- **Advanced Scheduling**: More complex scheduling rules
- **Notification Integration**: Task completion notifications
- **Metrics Export**: Prometheus metrics integration
- **Task Versioning**: Task configuration versioning and rollback

### API Extensions
- **Bulk Operations**: Batch task creation and updates
- **Task Import/Export**: Configuration backup and restore
- **Advanced Filtering**: More sophisticated task filtering
- **Webhook Support**: HTTP callbacks for task events