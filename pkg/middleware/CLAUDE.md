# Middleware Package (pkg/middleware)

## Overview
Shared middleware components for OpenTelemetry tracing, request correlation, and observability across all HTTP handlers in the monolith.

## Core Features
- **OpenTelemetry Tracing**: Automatic HTTP request tracing
- **Context Propagation**: Proper trace context handling
- **Span Management**: Request-scoped span lifecycle
- **Attribute Injection**: Rich metadata for observability

## Tracing Middleware
- **Automatic Spans**: Creates spans for all HTTP requests
- **URL Path Tracking**: Records request paths and methods
- **Error Capture**: Automatic error recording in spans
- **Response Status**: HTTP status code tracking

## Usage Examples
```go
// Apply tracing middleware
r.Use(middleware.TracingMiddleware)

// Custom tracing in handlers
span := trace.SpanFromContext(r.Context())
span.SetAttributes(attribute.String("custom", "value"))
```

## Integration Points
- **Chi Router**: Seamless integration with go-chi middleware stack
- **OpenTelemetry**: Works with existing telemetry infrastructure
- **Logging**: Correlates with structured logging system
- **Handlers**: Enhances shared handler utilities

## Configuration
- Respects global `ENABLE_TELEMETRY` setting
- No additional configuration required
- Automatic activation with telemetry system

## Performance
- **Minimal Overhead**: Optimized for production use
- **Conditional**: No impact when telemetry disabled
- **Efficient**: Reuses trace context across requests