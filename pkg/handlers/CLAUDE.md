# Handlers Package (pkg/handlers)

## Overview
Shared HTTP handlers and utilities for consistent request handling, health checks, response formatting, and OpenTelemetry integration across all modules. Provides common patterns and middleware components.

## Core Features
- **Health Check Handlers**: Standardized health endpoints with module information
- **Response Utilities**: Consistent JSON response formatting and error handling
- **OpenTelemetry Integration**: HTTP span creation and tracing utilities
- **Standard Response Format**: Unified JSON response structures across all modules
- **Error Handling**: Common error response patterns with proper HTTP status codes

## Health Check System
- `HealthHandler(moduleName)`: Module-specific health checks
- `SimpleHealthHandler()`: Basic health check without module info
- **Logging Exclusion**: Health checks excluded from logs to reduce noise
- **Response Format**: Consistent JSON with status and module information

## OpenTelemetry Integration
```go
// Start HTTP span with attributes
span, r := handlers.StartHTTPSpan(r, "operation.name",
    attribute.String("service", "module"),
    attribute.String("operation", "action"),
)
defer span.End()
```

## Response Utilities

### Standard Response Structure
```go
type StandardResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Message string      `json:"message,omitempty"`
    Details interface{} `json:"details,omitempty"`
}
```

### Available Response Functions
```go
// Success responses
JSONResponse(w, data, statusCode)          // Generic JSON response
SuccessResponse(w, data, statusCode)       // Successful response wrapper
CreatedResponse(w, data)                   // 201 Created
NoContentResponse(w)                       // 204 No Content

// Error responses  
ErrorResponse(w, message, statusCode, details...)  // Generic error
ValidationErrorResponse(w, errors)                 // 400 Validation errors
BadRequestResponse(w, message)                      // 400 Bad Request
UnauthorizedResponse(w)                             // 401 Unauthorized
ForbiddenResponse(w, message)                       // 403 Forbidden
NotFoundResponse(w, resource)                       // 404 Not Found
InternalErrorResponse(w, message)                   // 500 Internal Error
MessageResponse(w, message, statusCode)             // Simple message
```

## Tracing Features
- **Automatic Span Creation**: HTTP request tracing
- **Attribute Management**: Rich metadata for observability
- **Error Recording**: Automatic error capture in spans
- **Context Propagation**: Proper trace context handling

## Usage Patterns
```go
// Health check registration
r.Get("/health", handlers.HealthHandler("module-name"))

// Tracing in handlers
span, r := handlers.StartHTTPSpan(r, "handler.operation")
span.SetAttributes(attribute.Bool("success", true))
```

## Response Standards
- Consistent JSON structure across all endpoints
- Proper HTTP status codes
- Error details with context
- Module identification in responses

## Integration
Used by all modules for consistent HTTP handling patterns and observability integration.