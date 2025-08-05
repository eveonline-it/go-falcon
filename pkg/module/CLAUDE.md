# Module Package (pkg/module)

## Overview
Base module interface and common functionality for the modular monolith architecture. Provides standardized patterns for module initialization, health checks, background tasks, and resource management.

## Core Features
- **Module Interface**: Standard contract for all application modules
- **Base Module**: Common functionality implementation
- **Health Check System**: Standardized health endpoint registration
- **Background Tasks**: Managed background processing with graceful shutdown
- **Resource Management**: Shared access to databases and services

## Module Interface
```go
type Module interface {
    Name() string
    Routes(chi.Router)
    StartBackgroundTasks(context.Context)
    Stop()
}
```

## Base Module Implementation
- **Shared Dependencies**: MongoDB, Redis, SDE service access
- **Health Endpoints**: Automatic health check registration
- **Background Task Management**: Coordinated task lifecycle
- **Graceful Shutdown**: Clean resource cleanup

## Functionality Provided
- **Database Access**: Consistent MongoDB and Redis access patterns
- **SDE Integration**: Shared Static Data Export service
- **Health Monitoring**: Standard health check implementation
- **Task Coordination**: Background process scheduling and cleanup

## Usage Pattern
```go
// Create module with base functionality
type MyModule struct {
    *module.BaseModule
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService) *MyModule {
    return &MyModule{
        BaseModule: module.NewBaseModule("mymodule", mongodb, redis, sdeService),
    }
}
```

## Features
- **Name Management**: Consistent module identification
- **Route Registration**: Standardized HTTP endpoint patterns
- **Resource Sharing**: Common database and service access
- **Lifecycle Management**: Coordinated startup and shutdown

## Integration
- Used by all internal modules (auth, dev, users, notifications)
- Provides consistency across the modular monolith
- Enables shared resource management and standardized patterns