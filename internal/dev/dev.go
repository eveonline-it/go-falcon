package dev

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"go-falcon/pkg/database"
	"go-falcon/pkg/evegate"
	"go-falcon/pkg/handlers"
	"go-falcon/pkg/module"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
)

type Module struct {
	*module.BaseModule
	evegateClient *evegate.Client
}

func New(mongodb *database.MongoDB, redis *database.Redis) *Module {
	return &Module{
		BaseModule:    module.NewBaseModule("dev", mongodb, redis),
		evegateClient: evegate.NewClient(),
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	r.Get("/esi-status", m.esiStatusHandler)
	r.Get("/services", m.servicesHandler)
	r.Get("/status", m.statusHandler)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting dev module background tasks")
	
	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Add dev-specific background processing here
	for {
		select {
		case <-ctx.Done():
			slog.Info("Dev background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Dev background tasks stopped")
			return
		default:
			// Dev-specific background work would go here
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			}
		}
	}
}

// esiStatusHandler calls the EVE Online ESI status endpoint
func (m *Module) esiStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Create the main span for this handler operation
	span, r := handlers.StartHTTPSpan(r, "dev.esiStatusHandler",
		attribute.String("dev.operation", "esi_status"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()
	
	slog.InfoContext(r.Context(), "Dev: ESI status request", slog.String("remote_addr", r.RemoteAddr))
	
	ctx := r.Context()
	status, err := m.evegateClient.GetServerStatus(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get ESI status", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve ESI status","details":"` + err.Error() + `"}`))
		return
	}
	
	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("esi.players", status.Players),
		attribute.String("esi.server_version", status.ServerVersion),
	)
	
	slog.InfoContext(ctx, "Dev: ESI status retrieved successfully", 
		slog.Int("players", status.Players),
		slog.String("server_version", status.ServerVersion))
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"source":         "EVE Online ESI",
		"endpoint":       "https://esi.evetech.net/status",
		"status":         "success",
		"data":           status,
		"timestamp":      status.StartTime,
		"module":         m.Name(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// servicesHandler lists available development services
func (m *Module) servicesHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Dev: Services list request", slog.String("remote_addr", r.RemoteAddr))
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	services := map[string]interface{}{
		"module": m.Name(),
		"description": "Development module for testing and calling other services",
		"available_endpoints": []string{
			"/esi-status - Get EVE Online server status",
			"/services - List available development services",
			"/status - Get module status",
			"/health - Health check",
		},
		"evegate_client": "Available for EVE Online ESI calls",
	}
	
	json.NewEncoder(w).Encode(services)
}

func (m *Module) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"module":"dev","status":"running","version":"1.0.0","purpose":"development_testing"}`))
}