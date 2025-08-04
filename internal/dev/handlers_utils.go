package dev

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

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
			"/character/{characterID} - Get character information",
			"/character/{characterID}/portrait - Get character portrait URLs",
			"/universe/system/{systemID} - Get solar system information",
			"/universe/station/{stationID} - Get station information",
			"/alliances - Get all active alliances",
			"/alliance/{allianceID} - Get alliance information",
			"/alliance/{allianceID}/corporations - Get alliance member corporations",
			"/alliance/{allianceID}/icons - Get alliance icon URLs",
			"/sde/status - Get SDE service status and stats",
			"/sde/agent/{agentID} - Get agent information from SDE",
			"/sde/category/{categoryID} - Get category information from SDE",
			"/sde/blueprint/{blueprintID} - Get blueprint information from SDE",
			"/sde/agents/location/{locationID} - Get agents by location from SDE",
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

// addCacheInfoToResponse adds cache information to the response if data was cached
func addCacheInfoToResponse(response map[string]interface{}, cached bool, expiry *time.Time) {
	if cached && expiry != nil {
		response["cache"] = map[string]interface{}{
			"cached":     true,
			"expires_at": expiry.Format(time.RFC3339),
			"expires_in": int(time.Until(*expiry).Seconds()),
		}
	} else {
		response["cache"] = map[string]interface{}{
			"cached": false,
		}
	}
}