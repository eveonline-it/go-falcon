package middleware

import (
	"fmt"
	"net/http"

	"go-falcon/pkg/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TracingMiddleware(next http.Handler) http.Handler {
	// If telemetry is disabled, return the handler without tracing
	if !config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		fmt.Printf("[DEBUG] TracingMiddleware: Telemetry disabled, skipping tracing\n")
		return next
	}
	fmt.Printf("[DEBUG] TracingMiddleware: Telemetry enabled, setting up tracing\n")

	tracer := otel.Tracer("api-falcon")
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[DEBUG] TracingMiddleware: Processing request %s %s\n", r.Method, r.URL.Path)
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		
		ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path,
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.scheme", r.URL.Scheme),
				attribute.String("http.host", r.Host),
			),
		)
		defer span.End()

		r = r.WithContext(ctx)
		
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		
		fmt.Printf("[DEBUG] TracingMiddleware: Request completed with status %d\n", rw.statusCode)
		span.SetAttributes(
			attribute.Int("http.status_code", rw.statusCode),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}