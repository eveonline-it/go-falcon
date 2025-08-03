package logging

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

type OTelHandler struct {
	handler slog.Handler
	logger  log.Logger
}

func NewOTelHandler(handler slog.Handler) *OTelHandler {
	return &OTelHandler{
		handler: handler,
		logger:  global.GetLoggerProvider().Logger("microservice"),
	}
}

func (h *OTelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *OTelHandler) Handle(ctx context.Context, record slog.Record) error {
	// First, handle with the underlying handler (console/JSON)
	if err := h.handler.Handle(ctx, record); err != nil {
		return err
	}

	// Then send to OpenTelemetry
	logRecord := log.Record{}
	logRecord.SetTimestamp(record.Time)
	logRecord.SetBody(log.StringValue(record.Message))
	
	// Convert slog level to OpenTelemetry severity
	switch record.Level {
	case slog.LevelDebug:
		logRecord.SetSeverity(log.SeverityDebug)
	case slog.LevelInfo:
		logRecord.SetSeverity(log.SeverityInfo)
	case slog.LevelWarn:
		logRecord.SetSeverity(log.SeverityWarn)
	case slog.LevelError:
		logRecord.SetSeverity(log.SeverityError)
	}

	// Add trace context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		logRecord.AddAttributes(
			log.String("trace_id", spanCtx.TraceID().String()),
			log.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add attributes from slog record
	record.Attrs(func(attr slog.Attr) bool {
		logRecord.AddAttributes(log.String(attr.Key, attr.Value.String()))
		return true
	})

	h.logger.Emit(ctx, logRecord)
	return nil
}

func (h *OTelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &OTelHandler{
		handler: h.handler.WithAttrs(attrs),
		logger:  h.logger,
	}
}

func (h *OTelHandler) WithGroup(name string) slog.Handler {
	return &OTelHandler{
		handler: h.handler.WithGroup(name),
		logger:  h.logger,
	}
}