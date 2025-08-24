package logging

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
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
	// Add trace context to the console/JSON log output
	var attrs []slog.Attr

	// Copy existing attributes
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	// Add trace context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Create new record with trace attributes
	newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	newRecord.AddAttrs(attrs...)

	// Handle with the underlying handler (console/JSON) - now with trace info
	if err := h.handler.Handle(ctx, newRecord); err != nil {
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

	// Add trace context to OpenTelemetry logs
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		logRecord.AddAttributes(
			log.String("trace_id", spanCtx.TraceID().String()),
			log.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add attributes from slog record
	for _, attr := range attrs {
		logRecord.AddAttributes(log.String(attr.Key, attr.Value.String()))
	}

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
