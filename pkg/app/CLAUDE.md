# Application Package (pkg/app)

## Overview
Provides centralized application initialization and context management for the go-falcon monolith. Handles shared dependencies, configuration loading, and graceful shutdown.

## Core Features
- **Environment Loading**: Automatic .env file loading with godotenv
- **Dependency Injection**: Shared MongoDB, Redis, SDE service, and telemetry
- **Graceful Shutdown**: Coordinated cleanup of all application resources
- **Service Discovery**: Port configuration and environment detection

## Key Components
- `AppContext`: Central dependency container
- `InitializeApp()`: One-stop application initialization
- `Shutdown()`: Graceful cleanup with timeout
- Environment helpers: `GetPort()`, `IsProduction()`, `IsDevelopment()`

## Usage
```go
// Initialize all shared dependencies
appCtx, err := app.InitializeApp("service-name")
defer appCtx.Shutdown(ctx)

// Access shared resources
mongodb := appCtx.MongoDB
redis := appCtx.Redis
sdeService := appCtx.SDEService
```

## Dependencies
- MongoDB connection
- Redis connection  
- SDE service initialization
- OpenTelemetry telemetry manager
- Configuration management