# Logging Package (pkg/logging)

## Overview
OpenTelemetry-based logging and telemetry management with conditional activation, structured logging, and proper trace correlation. Follows OpenTelemetry Specification 1.47.0.

## Core Features
- **Conditional Telemetry**: Only active when `ENABLE_TELEMETRY=true`
- **Structured Logging**: JSON-formatted logs with trace correlation
- **OpenTelemetry Integration**: OTLP HTTP transport support
- **Graceful Shutdown**: Proper cleanup of telemetry resources
- **Context Propagation**: Service-specific logging contexts

## Telemetry Manager
- **Initialization**: Configure OTLP exporters and processors
- **Shutdown**: Clean resource cleanup with timeout
- **Environment Awareness**: Respects ENABLE_TELEMETRY flag
- **Multi-Transport**: Console and OTLP HTTP support

## Configuration
```bash
# Required for telemetry activation
ENABLE_TELEMETRY=true

# Optional configuration
SERVICE_NAME=falcon-dev
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

## Usage Examples
```go
// Initialize telemetry
telemetryManager := logging.NewTelemetryManager()
err := telemetryManager.Initialize(ctx)
defer telemetryManager.Shutdown(ctx)

// Structured logging with context
slog.InfoContext(ctx, "Operation completed", 
    slog.String("operation", "data_fetch"),
    slog.Int("count", 42))
```

## Features
- **Trace Correlation**: Automatic trace and span ID injection
- **Service Identification**: Service name in all telemetry data
- **Performance Safe**: No overhead when telemetry disabled
- **Production Ready**: Safe for deployment in any environment

## Integration
- Initialized in `pkg/app` for shared telemetry management
- Used by all modules for consistent observability
- Integrates with OpenTelemetry collectors and backends