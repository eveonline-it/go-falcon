package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
	
	"go-falcon/pkg/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type TelemetryConfig struct {
	EnableTelemetry   bool
	ServiceName       string
	OTLPEndpoint      string
	LogLevel          string
	EnablePrettyLogs  bool
	DisableConsoleLog bool
	NodeEnv           string
}

type TelemetryManager struct {
	config        TelemetryConfig
	shutdownFuncs []func(context.Context) error
	logger        *slog.Logger
}

func NewTelemetryManager() *TelemetryManager {
	telemetryConfig := TelemetryConfig{
		EnableTelemetry:   config.GetBoolEnv("ENABLE_TELEMETRY", true),
		ServiceName:       config.GetEnv("SERVICE_NAME", "unknown-service"),
		OTLPEndpoint:      config.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318"),
		LogLevel:          config.GetEnv("LOG_LEVEL", "info"),
		EnablePrettyLogs:  config.GetBoolEnv("ENABLE_PRETTY_LOGS", false),
		DisableConsoleLog: config.GetBoolEnv("DISABLE_CONSOLE_LOG", false),
		NodeEnv:           config.GetEnv("NODE_ENV", "development"),
	}

	return &TelemetryManager{
		config: telemetryConfig,
	}
}

func (tm *TelemetryManager) Initialize(ctx context.Context) error {
	// Setup structured logging first (always needed)
	tm.setupLogger()

	// Only initialize telemetry if enabled
	if !tm.config.EnableTelemetry {
		slog.Info("Telemetry disabled",
			slog.String("service", tm.config.ServiceName),
			slog.Bool("telemetry_enabled", false),
		)
		return nil
	}

	// Create resource following OpenTelemetry 1.47.0 spec
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(tm.config.ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
			semconv.DeploymentEnvironmentKey.String(tm.config.NodeEnv),
		),
	)
	if err != nil {
		return err
	}

	// Initialize tracing
	if err := tm.initTracing(ctx, res); err != nil {
		slog.Warn("Failed to initialize tracing", "error", err)
	}

	// Initialize logging
	if err := tm.initLogging(ctx, res); err != nil {
		slog.Warn("Failed to initialize OpenTelemetry logging", "error", err)
	}

	slog.Info("Telemetry initialized",
		slog.String("service", tm.config.ServiceName),
		slog.Bool("telemetry_enabled", tm.config.EnableTelemetry),
		slog.String("log_level", tm.config.LogLevel),
		slog.Bool("pretty_logs", tm.config.EnablePrettyLogs),
	)

	return nil
}

func (tm *TelemetryManager) initTracing(ctx context.Context, res *resource.Resource) error {
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(tm.config.OTLPEndpoint),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithURLPath("/v1/traces"),
	)
	if err != nil {
		return err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tm.shutdownFuncs = append(tm.shutdownFuncs, tp.Shutdown)
	
	slog.Info("OpenTelemetry tracing initialized", 
		"endpoint", tm.config.OTLPEndpoint,
		"service", tm.config.ServiceName)
	return nil
}

func (tm *TelemetryManager) initLogging(ctx context.Context, res *resource.Resource) error {
	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(tm.config.OTLPEndpoint),
		otlploghttp.WithInsecure(),
		otlploghttp.WithURLPath("/v1/logs"),
	)
	if err != nil {
		return err
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	global.SetLoggerProvider(lp)
	tm.shutdownFuncs = append(tm.shutdownFuncs, lp.Shutdown)

	slog.Info("OpenTelemetry logging initialized", "endpoint", tm.config.OTLPEndpoint)
	return nil
}

func (tm *TelemetryManager) setupLogger() {
	var handler slog.Handler

	level := parseLogLevel(tm.config.LogLevel)

	if tm.config.EnablePrettyLogs {
		// Pretty console logging for development
		opts := &slog.HandlerOptions{
			Level: level,
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// JSON logging for production
		opts := &slog.HandlerOptions{
			Level: level,
		}
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	// Only wrap with OTel handler if telemetry is enabled
	if tm.config.EnableTelemetry {
		handler = NewOTelHandler(handler)
	}

	logger := slog.New(handler)
	
	// Set as default logger
	slog.SetDefault(logger)
	tm.logger = logger
}

func (tm *TelemetryManager) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down telemetry...")
	
	for _, shutdown := range tm.shutdownFuncs {
		if err := shutdown(ctx); err != nil {
			slog.Error("Error shutting down telemetry component", "error", err)
		}
	}
	
	return nil
}

func (tm *TelemetryManager) Logger() *slog.Logger {
	return tm.logger
}

// Helper function for log level parsing

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}