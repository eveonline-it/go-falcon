package dev

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
)

// sdeStatusHandler provides SDE service status and statistics
func (m *Module) sdeStatusHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeStatusHandler",
		attribute.String("dev.operation", "sde_status"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: SDE status request", slog.String("remote_addr", r.RemoteAddr))

	sdeService := m.SDEService()
	isLoaded := sdeService.IsLoaded()

	var stats map[string]interface{}
	if isLoaded {
		// Get data counts
		agents, _ := sdeService.GetAllAgents()
		categories, _ := sdeService.GetAllCategories()
		blueprints, _ := sdeService.GetAllBlueprints()

		stats = map[string]interface{}{
			"loaded":           true,
			"agents_count":     len(agents),
			"categories_count": len(categories),
			"blueprints_count": len(blueprints),
		}

		span.SetAttributes(
			attribute.Bool("sde.loaded", true),
			attribute.Int("sde.agents_count", len(agents)),
			attribute.Int("sde.categories_count", len(categories)),
			attribute.Int("sde.blueprints_count", len(blueprints)),
		)
	} else {
		stats = map[string]interface{}{
			"loaded": false,
			"note":   "SDE data will be loaded on first access",
		}

		span.SetAttributes(attribute.Bool("sde.loaded", false))
	}

	span.SetAttributes(attribute.Bool("dev.success", true))
	slog.InfoContext(r.Context(), "Dev: SDE status retrieved", "loaded", isLoaded)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      stats,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeAgentHandler gets agent information from SDE
func (m *Module) sdeAgentHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeAgentHandler",
		attribute.String("dev.operation", "sde_agent"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	agentID := chi.URLParam(r, "agentID")
	slog.InfoContext(r.Context(), "Dev: SDE agent request", "agent_id", agentID)

	agent, err := m.SDEService().GetAgent(agentID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE agent", "agent_id", agentID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Agent not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.agent_id", agentID),
		attribute.Int("sde.agent_level", agent.Level),
	)

	slog.InfoContext(r.Context(), "Dev: SDE agent retrieved", "agent_id", agentID, "level", agent.Level)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      agent,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeCategoryHandler gets category information from SDE
func (m *Module) sdeCategoryHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeCategoryHandler",
		attribute.String("dev.operation", "sde_category"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	categoryID := chi.URLParam(r, "categoryID")
	slog.InfoContext(r.Context(), "Dev: SDE category request", "category_id", categoryID)

	category, err := m.SDEService().GetCategory(categoryID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE category", "category_id", categoryID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Category not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.category_id", categoryID),
		attribute.Bool("sde.category_published", category.Published),
	)

	slog.InfoContext(r.Context(), "Dev: SDE category retrieved", "category_id", categoryID, "published", category.Published)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      category,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeBlueprintHandler gets blueprint information from SDE
func (m *Module) sdeBlueprintHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeBlueprintHandler",
		attribute.String("dev.operation", "sde_blueprint"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	blueprintID := chi.URLParam(r, "blueprintID")
	slog.InfoContext(r.Context(), "Dev: SDE blueprint request", "blueprint_id", blueprintID)

	blueprint, err := m.SDEService().GetBlueprint(blueprintID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE blueprint", "blueprint_id", blueprintID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Blueprint not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.blueprint_id", blueprintID),
	)

	slog.InfoContext(r.Context(), "Dev: SDE blueprint retrieved", "blueprint_id", blueprintID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      blueprint,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeAgentsByLocationHandler gets agents by location from SDE
func (m *Module) sdeAgentsByLocationHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeAgentsByLocationHandler",
		attribute.String("dev.operation", "sde_agents_by_location"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	locationIDStr := chi.URLParam(r, "locationID")
	locationID, err := strconv.Atoi(locationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid location ID", "location_id", locationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid location ID"}`))
		return
	}

	slog.InfoContext(r.Context(), "Dev: SDE agents by location request", "location_id", locationID)

	agents, err := m.SDEService().GetAgentsByLocation(locationID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE agents by location", "location_id", locationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve agents by location","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("sde.location_id", locationID),
		attribute.Int("sde.agents_count", len(agents)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE agents by location retrieved", "location_id", locationID, "count", len(agents))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      agents,
		"module":    m.Name(),
		"count":     len(agents),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeBlueprintIdsHandler gets all available blueprint IDs from SDE
func (m *Module) sdeBlueprintIdsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeBlueprintIdsHandler",
		attribute.String("dev.operation", "sde_blueprint_ids"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: SDE blueprint IDs request")

	blueprints, err := m.SDEService().GetAllBlueprints()
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE blueprints", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve blueprint IDs","details":"` + err.Error() + `"}`))
		return
	}

	// Extract just the IDs
	var blueprintIDs []string
	for id := range blueprints {
		blueprintIDs = append(blueprintIDs, id)
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("sde.blueprint_ids_count", len(blueprintIDs)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE blueprint IDs retrieved", "count", len(blueprintIDs))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      blueprintIDs,
		"module":    m.Name(),
		"count":     len(blueprintIDs),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}