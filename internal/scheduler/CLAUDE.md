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
- **Execution Cancellation**: Context-based cancellation system for running tasks
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

### Execution Cancellation System

The scheduler provides **real-time execution cancellation** capability:

#### Architecture
- **Context-based Cancellation**: Each execution runs with a cancellable Go context
- **Execution Tracking**: Running executions are tracked in memory with cancellation functions
- **Thread-safe Operations**: Concurrent access to execution tracking is protected by mutexes
- **Graceful Termination**: Tasks can check context cancellation and terminate cleanly

#### Cancellation Process
1. **Stop Request**: `POST /tasks/{id}/stop` endpoint is called
2. **Find Running Executions**: System identifies all active executions for the task
3. **Cancel Contexts**: Each execution's context is cancelled using `context.Cancel()`
4. **Status Updates**: Cancelled executions are marked as "failed" with descriptive messages
5. **Task Pausing**: Task scheduling is paused to prevent new executions

#### Execution States
- **Active Executions**: Tracked in `runningExecutions` map with cancellation contexts
- **Cancelled Executions**: Status = "failed", Error = "Task execution was cancelled"
- **Cleanup**: Execution tracking is automatically removed when tasks complete

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
- **Execution Cancellation**: Real-time cancellation of running task executions

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

- **Character Affiliation Update** (`system-character-affiliation-update`)
  - Schedule: Every 30 minutes
  - Updates character corporation and alliance affiliations from ESI
  - Normal priority with 3 retry attempts and 5-minute retry intervals
  - Processes all characters in database using EVE ESI affiliation endpoint
  - Uses character module's UpdateService for batch processing with parallel workers

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

- **Alliance Bulk Import** (`system-alliance-bulk-import`)
  - Schedule: Weekly on Sunday at 3 AM
  - Retrieves all alliance IDs from ESI and imports detailed information
  - Normal priority with 2 retry attempts and 15-minute retry intervals
  - Processes alliances with configurable batch size and request delays
  - Uses alliance module's BulkImportAlliances for implementation

- **Corporation Data Update** (`system-corporation-update`)
  - Schedule: Daily at 4 AM
  - Updates all corporation information from EVE ESI for corporations in the database
  - Normal priority with 2 retry attempts and 15-minute retry intervals
  - Processes corporations with 10 concurrent workers by default
  - Uses corporation module's UpdateAllCorporations for parallel ESI updates
  - Includes rate limit compliance with 50ms delays between requests

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
| `/scheduler/status` | GET | Get scheduler status | None (public) |
| `/scheduler/stats` | GET | Get scheduler statistics | None (public) |
| `/scheduler/tasks` | GET | List tasks with filtering and pagination | Authentication required |
| `/scheduler/tasks` | POST | Create new task | Authentication required |
| `/scheduler/tasks/{id}` | GET | Get specific task details | Authentication required |
| `/scheduler/tasks/{id}` | PUT | Update task configuration | Authentication required |
| `/scheduler/tasks/{id}` | DELETE | Delete task (system tasks protected) | Authentication required |
| `/scheduler/tasks/{id}/execute` | POST | Manually execute task immediately | Authentication required |
| `/scheduler/tasks/{id}/stop` | POST | Stop currently running task and cancel active executions | Authentication required |
| `/scheduler/tasks/{id}/pause` | POST | Pause task scheduling | Authentication required |
| `/scheduler/tasks/{id}/resume` | POST | Resume paused task | Authentication required |
| `/scheduler/tasks/{id}/enable` | POST | Enable a disabled task | Authentication required |
| `/scheduler/tasks/{id}/disable` | POST | Disable a task without deleting it | Authentication required |
| `/scheduler/tasks/{id}/history` | GET | Get task execution history | Authentication required |
| `/scheduler/tasks/{id}/executions/{exec_id}` | GET | Get specific execution details | Authentication required |
| `/scheduler/reload` | POST | Reload tasks from database | Authentication required |

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
curl -X POST "/scheduler/tasks/task-id-123/execute"
```

#### Stop Running Task
```bash
curl -X POST "/scheduler/tasks/task-id-123/stop"
```

**Execution Cancellation**: The stop endpoint now provides **true execution cancellation** capability:
- **Cancels Active Executions**: Immediately interrupts any currently running executions for the specified task
- **Context-based Cancellation**: Uses Go's context cancellation for graceful termination
- **Status Tracking**: Cancelled executions are marked as "failed" with descriptive error messages
- **Prevents Future Scheduling**: Task status is updated to "paused" to prevent new executions

**Response for Cancelled Executions**:
```json
{
  "status": "failed",
  "error": "Task execution was cancelled",
  "output": "Execution stopped by user request",
  "completed_at": "2025-08-21T20:39:40.473Z"
}
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

// CharacterModule interface for affiliation update operations
type CharacterModule interface {
    UpdateAllAffiliations(ctx context.Context) (updated, failed, skipped int, err error)
}

// AllianceModule interface for alliance operations
type AllianceModule interface {
    BulkImportAlliances(ctx context.Context) (*dto.BulkImportAlliancesOutput, error)
}

// CorporationModule interface for corporation operations
type CorporationModule interface {
    UpdateAllCorporations(ctx context.Context, concurrentWorkers int) error
}

// GroupsModule interface for permission checking
type GroupsModule interface {
    RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler
}

// SystemExecutor with module dependencies
type SystemExecutor struct {
    authModule        AuthModule
    characterModule   CharacterModule
    allianceModule    AllianceModule
    corporationModule CorporationModule
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

#### Character Affiliation Update Integration

The character affiliation update system task demonstrates how the scheduler integrates with other modules:

```go
// System task definition in hardcoded.go
{
    ID:          "system-character-affiliation-update",
    Name:        "Character Affiliation Update",
    Description: "Updates character corporation and alliance affiliations from ESI",
    Type:        TaskTypeSystem,
    Schedule:    "0 */30 * * * *", // Every 30 minutes
    Status:      TaskStatusPending,
    Priority:    TaskPriorityNormal,
    Enabled:     true,
    Config: map[string]interface{}{
        "task_name": "character_affiliation_update",
    },
    Metadata: TaskMetadata{
        MaxRetries:    3,
        RetryInterval: 5 * time.Minute,
        Timeout:       30 * time.Minute,
        Tags:          []string{"system", "character", "esi"},
        IsSystem:      true,
        Source:        "system",
        Version:       1,
    },
}

// Execution in SystemExecutor
case "character_affiliation_update":
    if se.characterModule == nil {
        return &TaskResult{
            Success: false,
            Output:  "Character module not available",
        }, fmt.Errorf("character module not initialized")
    }

    stats, err := se.characterModule.UpdateAllAffiliations(ctx)
    if err != nil {
        return &TaskResult{
            Success: false,
            Output:  fmt.Sprintf("Failed to update affiliations: %v", err),
        }, err
    }

    return &TaskResult{
        Success: true,
        Output: fmt.Sprintf("Updated %d characters (%d failed, %d skipped) in %d seconds",
            stats.UpdatedCharacters, stats.FailedCharacters, stats.SkippedCharacters, stats.Duration),
    }, nil
```

**Key Integration Features**:
- **Module Interface**: Uses CharacterModule interface for loose coupling
- **Dependency Injection**: Character module injected during scheduler initialization
- **Error Handling**: Proper error propagation and reporting
- **Statistics**: Detailed execution statistics returned to scheduler
- **Timeout Management**: 30-minute timeout for large-scale updates
- **Retry Logic**: 3 retry attempts with 5-minute intervals on failure

#### Corporation Data Update Integration

The corporation data update system task demonstrates parallel processing integration:

```go
// System task definition in hardcoded.go
{
    ID:          "system-corporation-update",
    Name:        "Corporation Data Update",
    Description: "Updates all corporation information from EVE ESI for corporations in the database",
    Type:        TaskTypeSystem,
    Schedule:    "0 0 4 * * *", // Daily at 4 AM
    Status:      TaskStatusPending,
    Priority:    TaskPriorityNormal,
    Enabled:     true,
    Config: map[string]interface{}{
        "task_name": "corporation_update",
        "parameters": map[string]interface{}{
            "concurrent_workers": 10,
            "timeout":           "60m",
        },
    },
    Metadata: TaskMetadata{
        MaxRetries:    2,
        RetryInterval: 15 * time.Minute,
        Timeout:       60 * time.Minute,
        Tags:          []string{"system", "corporation", "esi", "update"},
        IsSystem:      true,
        Source:        "system",
        Version:       1,
    },
}

// Execution in SystemExecutor
case "corporation_update":
    if se.corporationModule == nil {
        return &TaskResult{
            Success: false,
            Output:  "Corporation module not available",
        }, fmt.Errorf("corporation module not initialized")
    }

    concurrentWorkers := 10 // default
    if params, ok := config.Parameters["concurrent_workers"].(int); ok {
        concurrentWorkers = params
    }

    err := se.corporationModule.UpdateAllCorporations(ctx, concurrentWorkers)
    if err != nil {
        return &TaskResult{
            Success: false,
            Output:  fmt.Sprintf("Corporation update failed: %v", err),
        }, err
    }

    return &TaskResult{
        Success: true,
        Output:  fmt.Sprintf("Successfully updated all corporations with %d concurrent workers", concurrentWorkers),
    }, nil
```

**Key Features**:
- **Parallel Processing**: Configurable concurrent workers (default: 10)
- **Rate Limiting**: Built-in ESI rate limit compliance with 50ms delays
- **Progress Tracking**: Logs progress every 100 corporations processed
- **Error Resilience**: Continues processing on individual failures
- **Memory Efficient**: Only loads corporation IDs, not full documents

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

### Permission System

The scheduler module uses a simplified permission model:

- **super_admin**: Full administrative access to all scheduler functionality including task creation, modification, deletion, and execution
- **authenticated**: Access to view scheduler status and statistics
- **public**: Access to basic scheduler status only

System tasks are always protected from modification or deletion regardless of permission level.

## Monitoring & Alerting

### Metrics Available
- Total tasks, enabled tasks, running tasks
- Daily completion and failure counts
- Average execution times
- Worker utilization and queue sizes
- Next scheduled run times

### Health Checks
```bash
# Check scheduler status
curl /scheduler/status

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
5. **Context Awareness**: Tasks should check context cancellation for graceful termination
6. **Cancellation Handling**: Design tasks to handle cancellation gracefully and save state

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

**Long-Running or Stuck Tasks**
- Use the stop endpoint to cancel running executions: `POST /tasks/{id}/stop`
- Check execution history for cancelled tasks with error "Task execution was cancelled"
- Monitor task execution duration and set appropriate timeouts
- Review system task logs for EVE ESI rate limiting issues

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
curl -X POST /scheduler/tasks/task-id/execute

# Stop running task and cancel executions
curl -X POST /scheduler/tasks/task-id/stop

# Check for cancelled executions
curl "/scheduler/executions?task_id=task-id&status=failed" | jq '.executions[] | select(.error == "Task execution was cancelled")'
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
- `go-falcon/internal/character` - Character module interface for affiliation updates

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