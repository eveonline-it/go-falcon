package handlers

import (
	"net/http"

	"go-falcon/pkg/config"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware creates HTTP tracing middleware using OpenTelemetry
func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	// If telemetry is disabled, return a no-op middleware
	if !config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return otelhttp.NewMiddleware(
		serviceName,
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
	)
}

// StartHTTPSpan starts a new span for HTTP operations
func StartHTTPSpan(r *http.Request, operationName string, attributes ...attribute.KeyValue) (trace.Span, *http.Request) {
	// Only create spans if telemetry is enabled
	if !config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		return trace.SpanFromContext(r.Context()), r
	}
	
	tracer := otel.Tracer("go-falcon/handlers")
	
	// Get the existing context (which may already have a span from middleware)
	ctx := r.Context()
	
	// Start a new span as a child of any existing span
	ctx, span := tracer.Start(ctx, operationName)
	
	// Add default HTTP attributes
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
		attribute.String("http.scheme", r.URL.Scheme),
		attribute.String("http.host", r.Host),
		attribute.String("http.target", r.URL.Path),
		attribute.String("user_agent.original", r.UserAgent()),
	)
	
	// Add custom attributes
	if len(attributes) > 0 {
		span.SetAttributes(attributes...)
	}
	
	return span, r.WithContext(ctx)
}