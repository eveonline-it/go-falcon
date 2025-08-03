package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status string `json:"status"`
	Module string `json:"module,omitempty"`
}

// HealthHandler creates a generic health check handler for a given module
func HealthHandler(moduleName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Health check requested",
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.String("module", moduleName),
		)

		response := HealthResponse{
			Status: "healthy",
			Module: moduleName,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("Failed to encode health response", "error", err, "module", moduleName)
		}
	}
}

// SimpleHealthHandler creates a simple health check without module information
func SimpleHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Health check requested",
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
		)

		response := HealthResponse{
			Status: "healthy",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("Failed to encode health response", "error", err)
		}
	}
}