package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
	goredis "github.com/redis/go-redis/v9"
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
		marketGroups, _ := sdeService.GetAllMarketGroups()
		metaGroups, _ := sdeService.GetAllMetaGroups()
		npcCorporations, _ := sdeService.GetAllNPCCorporations()
		typeIDs, _ := sdeService.GetAllTypeIDs()
		types, _ := sdeService.GetAllTypes()

		stats = map[string]interface{}{
			"loaded":                true,
			"agents_count":          len(agents),
			"categories_count":      len(categories),
			"blueprints_count":      len(blueprints),
			"market_groups_count":   len(marketGroups),
			"meta_groups_count":     len(metaGroups),
			"npc_corporations_count": len(npcCorporations),
			"type_ids_count":        len(typeIDs),
			"types_count":           len(types),
		}

		span.SetAttributes(
			attribute.Bool("sde.loaded", true),
			attribute.Int("sde.agents_count", len(agents)),
			attribute.Int("sde.categories_count", len(categories)),
			attribute.Int("sde.blueprints_count", len(blueprints)),
			attribute.Int("sde.market_groups_count", len(marketGroups)),
			attribute.Int("sde.meta_groups_count", len(metaGroups)),
			attribute.Int("sde.npc_corporations_count", len(npcCorporations)),
			attribute.Int("sde.type_ids_count", len(typeIDs)),
			attribute.Int("sde.types_count", len(types)),
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

// sdeMarketGroupHandler gets market group information from SDE
func (m *Module) sdeMarketGroupHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeMarketGroupHandler",
		attribute.String("dev.operation", "sde_market_group"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	marketGroupID := chi.URLParam(r, "marketGroupID")
	slog.InfoContext(r.Context(), "Dev: SDE market group request", "market_group_id", marketGroupID)

	marketGroup, err := m.SDEService().GetMarketGroup(marketGroupID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE market group", "market_group_id", marketGroupID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Market group not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.market_group_id", marketGroupID),
		attribute.Bool("sde.market_group_has_types", marketGroup.HasTypes),
	)

	slog.InfoContext(r.Context(), "Dev: SDE market group retrieved", "market_group_id", marketGroupID, "has_types", marketGroup.HasTypes)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      marketGroup,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeMarketGroupsHandler gets all market groups from SDE
func (m *Module) sdeMarketGroupsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeMarketGroupsHandler",
		attribute.String("dev.operation", "sde_market_groups"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: SDE market groups request")

	marketGroups, err := m.SDEService().GetAllMarketGroups()
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE market groups", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve market groups","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("sde.market_groups_count", len(marketGroups)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE market groups retrieved", "count", len(marketGroups))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      marketGroups,
		"module":    m.Name(),
		"count":     len(marketGroups),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeMetaGroupHandler gets meta group information from SDE
func (m *Module) sdeMetaGroupHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeMetaGroupHandler",
		attribute.String("dev.operation", "sde_meta_group"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	metaGroupID := chi.URLParam(r, "metaGroupID")
	slog.InfoContext(r.Context(), "Dev: SDE meta group request", "meta_group_id", metaGroupID)

	metaGroup, err := m.SDEService().GetMetaGroup(metaGroupID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE meta group", "meta_group_id", metaGroupID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Meta group not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.meta_group_id", metaGroupID),
	)

	slog.InfoContext(r.Context(), "Dev: SDE meta group retrieved", "meta_group_id", metaGroupID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      metaGroup,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeMetaGroupsHandler gets all meta groups from SDE
func (m *Module) sdeMetaGroupsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeMetaGroupsHandler",
		attribute.String("dev.operation", "sde_meta_groups"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: SDE meta groups request")

	metaGroups, err := m.SDEService().GetAllMetaGroups()
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE meta groups", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve meta groups","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("sde.meta_groups_count", len(metaGroups)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE meta groups retrieved", "count", len(metaGroups))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      metaGroups,
		"module":    m.Name(),
		"count":     len(metaGroups),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeNPCCorpHandler gets NPC corporation information from SDE
func (m *Module) sdeNPCCorpHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeNPCCorpHandler",
		attribute.String("dev.operation", "sde_npc_corp"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	corpID := chi.URLParam(r, "corpID")
	slog.InfoContext(r.Context(), "Dev: SDE NPC corporation request", "corp_id", corpID)

	npcCorp, err := m.SDEService().GetNPCCorporation(corpID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE NPC corporation", "corp_id", corpID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"NPC corporation not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.npc_corp_id", corpID),
		attribute.String("sde.npc_corp_ticker", npcCorp.TickerName.String()),
	)

	slog.InfoContext(r.Context(), "Dev: SDE NPC corporation retrieved", "corp_id", corpID, "ticker", npcCorp.TickerName.String())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      npcCorp,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeNPCCorpsHandler gets all NPC corporations from SDE
func (m *Module) sdeNPCCorpsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeNPCCorpsHandler",
		attribute.String("dev.operation", "sde_npc_corps"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: SDE NPC corporations request")

	npcCorps, err := m.SDEService().GetAllNPCCorporations()
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE NPC corporations", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve NPC corporations","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("sde.npc_corps_count", len(npcCorps)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE NPC corporations retrieved", "count", len(npcCorps))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      npcCorps,
		"module":    m.Name(),
		"count":     len(npcCorps),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeNPCCorpsByFactionHandler gets NPC corporations by faction from SDE
func (m *Module) sdeNPCCorpsByFactionHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeNPCCorpsByFactionHandler",
		attribute.String("dev.operation", "sde_npc_corps_by_faction"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	factionIDStr := chi.URLParam(r, "factionID")
	factionID, err := strconv.Atoi(factionIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid faction ID", "faction_id", factionIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid faction ID"}`))
		return
	}

	slog.InfoContext(r.Context(), "Dev: SDE NPC corporations by faction request", "faction_id", factionID)

	npcCorps, err := m.SDEService().GetNPCCorporationsByFaction(factionID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE NPC corporations by faction", "faction_id", factionID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve NPC corporations by faction","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("sde.faction_id", factionID),
		attribute.Int("sde.npc_corps_count", len(npcCorps)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE NPC corporations by faction retrieved", "faction_id", factionID, "count", len(npcCorps))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      npcCorps,
		"module":    m.Name(),
		"count":     len(npcCorps),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeTypeIDHandler gets type ID information from SDE
func (m *Module) sdeTypeIDHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeTypeIDHandler",
		attribute.String("dev.operation", "sde_type_id"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	typeID := chi.URLParam(r, "typeID")
	slog.InfoContext(r.Context(), "Dev: SDE type ID request", "type_id", typeID)

	typeData, err := m.SDEService().GetTypeID(typeID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE type ID", "type_id", typeID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Type ID not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.type_id", typeID),
		attribute.Bool("sde.type_published", typeData.Published),
	)

	slog.InfoContext(r.Context(), "Dev: SDE type ID retrieved", "type_id", typeID, "published", typeData.Published)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      typeData,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}


// sdeTypeHandler gets type information from SDE
func (m *Module) sdeTypeHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeTypeHandler",
		attribute.String("dev.operation", "sde_type"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	typeID := chi.URLParam(r, "typeID")
	slog.InfoContext(r.Context(), "Dev: SDE type request", "type_id", typeID)

	typeData, err := m.SDEService().GetType(typeID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE type", "type_id", typeID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Type not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.type_id", typeID),
		attribute.Bool("sde.type_published", typeData.Published),
		attribute.Int("sde.type_group_id", typeData.GroupID),
	)

	slog.InfoContext(r.Context(), "Dev: SDE type retrieved", "type_id", typeID, "published", typeData.Published, "group_id", typeData.GroupID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      typeData,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeTypesHandler gets all types from SDE
func (m *Module) sdeTypesHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeTypesHandler",
		attribute.String("dev.operation", "sde_types"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: SDE types request")

	types, err := m.SDEService().GetAllTypes()
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE types", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve types","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("sde.types_count", len(types)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE types retrieved", "count", len(types))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      types,
		"module":    m.Name(),
		"count":     len(types),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdePublishedTypesHandler gets all published types from SDE
func (m *Module) sdePublishedTypesHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdePublishedTypesHandler",
		attribute.String("dev.operation", "sde_published_types"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: SDE published types request")

	types, err := m.SDEService().GetPublishedTypes()
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE published types", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve published types","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("sde.published_types_count", len(types)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE published types retrieved", "count", len(types))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      types,
		"module":    m.Name(),
		"count":     len(types),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeTypesByGroupHandler gets types by group ID from SDE
func (m *Module) sdeTypesByGroupHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeTypesByGroupHandler",
		attribute.String("dev.operation", "sde_types_by_group"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid group ID", "group_id", groupIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid group ID"}`))
		return
	}

	slog.InfoContext(r.Context(), "Dev: SDE types by group request", "group_id", groupID)

	types, err := m.SDEService().GetTypesByGroupID(groupID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE types by group", "group_id", groupID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve types by group","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("sde.group_id", groupID),
		attribute.Int("sde.types_count", len(types)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE types by group retrieved", "group_id", groupID, "count", len(types))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      types,
		"module":    m.Name(),
		"count":     len(types),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeTypeMaterialsHandler gets type materials from SDE
func (m *Module) sdeTypeMaterialsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeTypeMaterialsHandler",
		attribute.String("dev.operation", "sde_type_materials"),
		attribute.String("dev.service", "sde"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	typeID := chi.URLParam(r, "typeID")
	slog.InfoContext(r.Context(), "Dev: SDE type materials request", "type_id", typeID)

	materials, err := m.SDEService().GetTypeMaterials(typeID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE type materials", "type_id", typeID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Type materials not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.type_id", typeID),
		attribute.Int("sde.materials_count", len(materials)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE type materials retrieved", "type_id", typeID, "count", len(materials))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Service",
		"status":    "success",
		"data":      materials,
		"module":    m.Name(),
		"count":     len(materials),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// getSdeDataFromRedis retrieves SDE data directly from Redis JSON
func (m *Module) getSdeDataFromRedis(ctx context.Context, key string) (interface{}, error) {
	redis := m.Redis()
	
	result, err := redis.Client.Do(ctx, "JSON.GET", key, "$").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get SDE data from Redis: %w", err)
	}
	
	if result == nil {
		return nil, fmt.Errorf("SDE data not found for key: %s", key)
	}
	
	// Redis JSON.GET returns a JSON string, parse it
	var data []interface{}
	if err := json.Unmarshal([]byte(result.(string)), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SDE data: %w", err)
	}
	
	if len(data) == 0 {
		return nil, fmt.Errorf("empty SDE data for key: %s", key)
	}
	
	return data[0], nil
}

// getSdeDataByPattern retrieves multiple SDE keys matching a pattern
func (m *Module) getSdeDataByPattern(ctx context.Context, pattern string) (map[string]interface{}, error) {
	redis := m.Redis()
	
	// Get all keys matching the pattern
	keys, err := redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
	}
	
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}
	
	// Use pipeline to get all data
	pipe := redis.Client.Pipeline()
	for _, key := range keys {
		pipe.Do(ctx, "JSON.GET", key, "$")
	}
	
	results, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}
	
	data := make(map[string]interface{})
	for i, result := range results {
		if result.Err() != nil {
			slog.Warn("Failed to get SDE data", "key", keys[i], "error", result.Err())
			continue
		}
		
		// Parse JSON result
		cmd := result.(*goredis.Cmd)
		resultStr, err := cmd.Text()
		if err != nil {
			slog.Warn("Failed to get text from result", "key", keys[i], "error", err)
			continue
		}
		
		var jsonData []interface{}
		if err := json.Unmarshal([]byte(resultStr), &jsonData); err != nil {
			slog.Warn("Failed to unmarshal data", "key", keys[i], "error", err)
			continue
		}
		
		if len(jsonData) > 0 {
			data[keys[i]] = jsonData[0]
		}
	}
	
	return data, nil
}

// sdeRedisEntityHandler gets a specific SDE entity from Redis
func (m *Module) sdeRedisEntityHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeRedisEntityHandler",
		attribute.String("dev.operation", "sde_redis_entity"),
		attribute.String("dev.service", "redis"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	entityType := chi.URLParam(r, "type")
	entityID := chi.URLParam(r, "id")
	
	slog.InfoContext(r.Context(), "Dev: SDE Redis entity request", "type", entityType, "id", entityID)

	key := fmt.Sprintf("sde:%s:%s", entityType, entityID)
	data, err := m.getSdeDataFromRedis(r.Context(), key)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE entity from Redis", "key", key, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Entity not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.entity_type", entityType),
		attribute.String("sde.entity_id", entityID),
		attribute.String("sde.redis_key", key),
	)

	slog.InfoContext(r.Context(), "Dev: SDE entity retrieved from Redis", "type", entityType, "id", entityID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "Redis SDE",
		"status":    "success",
		"data":      data,
		"module":    m.Name(),
		"redis_key": key,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeRedisEntitiesByTypeHandler gets all entities of a specific type from Redis
func (m *Module) sdeRedisEntitiesByTypeHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeRedisEntitiesByTypeHandler",
		attribute.String("dev.operation", "sde_redis_entities_by_type"),
		attribute.String("dev.service", "redis"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	entityType := chi.URLParam(r, "type")
	
	slog.InfoContext(r.Context(), "Dev: SDE Redis entities by type request", "type", entityType)

	pattern := fmt.Sprintf("sde:%s:*", entityType)
	data, err := m.getSdeDataByPattern(r.Context(), pattern)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE entities from Redis", "pattern", pattern, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve entities","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.entity_type", entityType),
		attribute.String("sde.redis_pattern", pattern),
		attribute.Int("sde.entities_count", len(data)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE entities retrieved from Redis", "type", entityType, "count", len(data))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":       "Redis SDE",
		"status":       "success",
		"data":         data,
		"module":       m.Name(),
		"redis_pattern": pattern,
		"count":        len(data),
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeUniverseRegionSystemsHandler gets all solar systems from a region
func (m *Module) sdeUniverseRegionSystemsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeUniverseRegionSystemsHandler",
		attribute.String("dev.operation", "sde_universe_region_systems"),
		attribute.String("dev.service", "redis"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	universeType := chi.URLParam(r, "universeType")
	regionName := chi.URLParam(r, "regionName")
	
	slog.InfoContext(r.Context(), "Dev: SDE universe region systems request", "universe_type", universeType, "region", regionName)

	// Pattern to match all systems in the region: sde:universe:{type}:{region}:*:*
	pattern := fmt.Sprintf("sde:universe:%s:%s:*:*", universeType, regionName)
	data, err := m.getSdeDataByPattern(r.Context(), pattern)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get region systems from Redis", "pattern", pattern, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve region systems","details":"` + err.Error() + `"}`))
		return
	}

	// Filter to only include system-level data (6 segments total: sde:universe:type:region:constellation:system)
	systemData := make(map[string]interface{})
	for key, value := range data {
		keyParts := strings.Split(key, ":")
		if len(keyParts) == 6 && !strings.HasSuffix(key, ":constellation.yaml") { // sde:universe:type:region:constellation:system
			systemData[key] = value
		}
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.universe_type", universeType),
		attribute.String("sde.region_name", regionName),
		attribute.String("sde.redis_pattern", pattern),
		attribute.Int("sde.systems_count", len(systemData)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE region systems retrieved from Redis", 
		"universe_type", universeType, "region", regionName, "systems_count", len(systemData))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":        "Redis SDE",
		"status":        "success",
		"data":          systemData,
		"module":        m.Name(),
		"universe_type": universeType,
		"region_name":   regionName,
		"redis_pattern": pattern,
		"systems_count": len(systemData),
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeUniverseConstellationSystemsHandler gets all solar systems from a constellation
func (m *Module) sdeUniverseConstellationSystemsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeUniverseConstellationSystemsHandler",
		attribute.String("dev.operation", "sde_universe_constellation_systems"),
		attribute.String("dev.service", "redis"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	universeType := chi.URLParam(r, "universeType")
	regionName := chi.URLParam(r, "regionName")
	constellationName := chi.URLParam(r, "constellationName")
	
	slog.InfoContext(r.Context(), "Dev: SDE universe constellation systems request", 
		"universe_type", universeType, "region", regionName, "constellation", constellationName)

	// Pattern to match all systems in the constellation: sde:universe:{type}:{region}:{constellation}:*
	pattern := fmt.Sprintf("sde:universe:%s:%s:%s:*", universeType, regionName, constellationName)
	data, err := m.getSdeDataByPattern(r.Context(), pattern)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get constellation systems from Redis", "pattern", pattern, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve constellation systems","details":"` + err.Error() + `"}`))
		return
	}

	// Filter to only include system-level data (6 segments total: sde:universe:type:region:constellation:system)
	systemData := make(map[string]interface{})
	for key, value := range data {
		keyParts := strings.Split(key, ":")
		if len(keyParts) == 6 && !strings.HasSuffix(key, ":constellation.yaml") { // sde:universe:type:region:constellation:system
			systemData[key] = value
		}
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.universe_type", universeType),
		attribute.String("sde.region_name", regionName),
		attribute.String("sde.constellation_name", constellationName),
		attribute.String("sde.redis_pattern", pattern),
		attribute.Int("sde.systems_count", len(systemData)),
	)

	slog.InfoContext(r.Context(), "Dev: SDE constellation systems retrieved from Redis", 
		"universe_type", universeType, "region", regionName, "constellation", constellationName, "systems_count", len(systemData))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":            "Redis SDE",
		"status":            "success",
		"data":              systemData,
		"module":            m.Name(),
		"universe_type":     universeType,
		"region_name":       regionName,
		"constellation_name": constellationName,
		"redis_pattern":     pattern,
		"systems_count":     len(systemData),
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeUniverseDataHandler gets specific universe data (region, constellation, or system)
func (m *Module) sdeUniverseDataHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeUniverseDataHandler",
		attribute.String("dev.operation", "sde_universe_data"),
		attribute.String("dev.service", "redis"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	universeType := chi.URLParam(r, "universeType")
	regionName := chi.URLParam(r, "regionName")
	constellationName := chi.URLParam(r, "constellationName")
	systemName := chi.URLParam(r, "systemName")
	
	// Build Redis key based on provided parameters
	var key string
	var level string
	
	if systemName != "" {
		key = fmt.Sprintf("sde:universe:%s:%s:%s:%s", universeType, regionName, constellationName, systemName)
		level = "system"
	} else if constellationName != "" {
		key = fmt.Sprintf("sde:universe:%s:%s:%s:constellation.yaml", universeType, regionName, constellationName)
		level = "constellation"
	} else {
		key = fmt.Sprintf("sde:universe:%s:%s:region.yaml:region", universeType, regionName)
		level = "region"
	}
	
	slog.InfoContext(r.Context(), "Dev: SDE universe data request", 
		"universe_type", universeType, "region", regionName, "constellation", constellationName, "system", systemName, "level", level, "key", key)

	data, err := m.getSdeDataFromRedis(r.Context(), key)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get universe data from Redis", "key", key, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Universe data not found","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.universe_type", universeType),
		attribute.String("sde.region_name", regionName),
		attribute.String("sde.constellation_name", constellationName),
		attribute.String("sde.system_name", systemName),
		attribute.String("sde.level", level),
		attribute.String("sde.redis_key", key),
	)

	slog.InfoContext(r.Context(), "Dev: SDE universe data retrieved from Redis", 
		"universe_type", universeType, "level", level, "key", key)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":        "Redis SDE",
		"status":        "success",
		"data":          data,
		"module":        m.Name(),
		"universe_type": universeType,
		"region_name":   regionName,
		"constellation_name": constellationName,
		"system_name":   systemName,
		"level":         level,
		"redis_key":     key,
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeManagementStatusHandler gets the SDE management module status (like internal/sde module)
func (m *Module) sdeManagementStatusHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeManagementStatusHandler",
		attribute.String("dev.operation", "sde_management_status"),
		attribute.String("dev.service", "sde_management"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: SDE management status request", slog.String("remote_addr", r.RemoteAddr))

	// Get SDE management status from Redis (same keys as internal/sde module)
	redis := m.Redis()
	ctx := r.Context()

	// Get current hash
	currentHash, _ := redis.Client.Get(ctx, "sde:current_hash").Result()

	// Get stored status
	statusJSON, _ := redis.Client.Get(ctx, "sde:status").Result()
	var sdeStatus map[string]interface{}
	if statusJSON != "" {
		json.Unmarshal([]byte(statusJSON), &sdeStatus)
	} else {
		sdeStatus = make(map[string]interface{})
	}

	// Get progress
	progressJSON, _ := redis.Client.Get(ctx, "sde:progress").Result()
	var progressData map[string]interface{}
	if progressJSON != "" {
		json.Unmarshal([]byte(progressJSON), &progressData)
	} else {
		progressData = make(map[string]interface{})
	}

	// Build comprehensive status
	status := map[string]interface{}{
		"current_hash":     currentHash,
		"sde_status":       sdeStatus,
		"progress_info":    progressData,
		"management_type":  "web_based",
		"redis_keys": map[string]interface{}{
			"hash_key":     "sde:current_hash",
			"status_key":   "sde:status",
			"progress_key": "sde:progress",
		},
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.String("sde.current_hash", currentHash),
		attribute.Bool("sde.has_status", statusJSON != ""),
		attribute.Bool("sde.has_progress", progressJSON != ""),
	)

	slog.InfoContext(r.Context(), "Dev: SDE management status retrieved", "has_hash", currentHash != "", "has_status", statusJSON != "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "SDE Management",
		"status":    "success",
		"data":      status,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeComprehensiveStatusHandler provides comprehensive view of both SDE service and management
func (m *Module) sdeComprehensiveStatusHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeComprehensiveStatusHandler",
		attribute.String("dev.operation", "sde_comprehensive_status"),
		attribute.String("dev.service", "comprehensive"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: Comprehensive SDE status request")

	// Get in-memory SDE service status
	sdeService := m.SDEService()
	isLoaded := sdeService.IsLoaded()

	var memoryStats map[string]interface{}
	if isLoaded {
		// Get data counts from memory service
		agents, _ := sdeService.GetAllAgents()
		categories, _ := sdeService.GetAllCategories()
		blueprints, _ := sdeService.GetAllBlueprints()
		marketGroups, _ := sdeService.GetAllMarketGroups()
		metaGroups, _ := sdeService.GetAllMetaGroups()
		npcCorporations, _ := sdeService.GetAllNPCCorporations()
		typeIDs, _ := sdeService.GetAllTypeIDs()
		types, _ := sdeService.GetAllTypes()

		memoryStats = map[string]interface{}{
			"loaded":                true,
			"agents_count":          len(agents),
			"categories_count":      len(categories),
			"blueprints_count":      len(blueprints),
			"market_groups_count":   len(marketGroups),
			"meta_groups_count":     len(metaGroups),
			"npc_corporations_count": len(npcCorporations),
			"type_ids_count":        len(typeIDs),
			"types_count":           len(types),
			"data_source":           "memory",
		}
	} else {
		memoryStats = map[string]interface{}{
			"loaded":      false,
			"note":        "SDE data will be loaded on first access",
			"data_source": "memory",
		}
	}

	// Get Redis-based SDE management status
	redis := m.Redis()
	ctx := r.Context()

	currentHash, _ := redis.Client.Get(ctx, "sde:current_hash").Result()
	statusJSON, _ := redis.Client.Get(ctx, "sde:status").Result()
	progressJSON, _ := redis.Client.Get(ctx, "sde:progress").Result()

	var sdeStatus map[string]interface{}
	if statusJSON != "" {
		json.Unmarshal([]byte(statusJSON), &sdeStatus)
	} else {
		sdeStatus = make(map[string]interface{})
	}

	var progressData map[string]interface{}
	if progressJSON != "" {
		json.Unmarshal([]byte(progressJSON), &progressData)
	} else {
		progressData = make(map[string]interface{})
	}

	// Get sample Redis SDE entity counts for verification
	redisStats := map[string]interface{}{
		"current_hash": currentHash,
		"status":       sdeStatus,
		"progress":     progressData,
		"data_source":  "redis_individual_keys",
	}

	// Try to get entity counts from Redis pattern matching
	entityTypes := []string{"agents", "categories", "blueprints", "marketGroups", "metaGroups", "npcCorporations", "typeIDs", "types"}
	entityCounts := make(map[string]int)

	for _, entityType := range entityTypes {
		pattern := fmt.Sprintf("sde:%s:*", entityType)
		keys, err := redis.Client.Keys(ctx, pattern).Result()
		if err == nil {
			entityCounts[entityType+"_count"] = len(keys)
		}
	}

	if len(entityCounts) > 0 {
		redisStats["entity_counts"] = entityCounts
	}

	// Check for universe data
	universeKeys, err := redis.Client.Keys(ctx, "sde:universe:*").Result()
	if err == nil {
		redisStats["universe_entities_count"] = len(universeKeys)
	}

	comprehensiveStatus := map[string]interface{}{
		"memory_service": memoryStats,
		"redis_management": redisStats,
		"comparison": map[string]interface{}{
			"memory_loaded": isLoaded,
			"redis_has_data": currentHash != "",
			"sources_available": []string{"memory", "redis"},
		},
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Bool("sde.memory_loaded", isLoaded),
		attribute.Bool("sde.redis_has_data", currentHash != ""),
		attribute.Int("sde.universe_entities", len(universeKeys)),
	)

	slog.InfoContext(r.Context(), "Dev: Comprehensive SDE status retrieved", 
		"memory_loaded", isLoaded, "redis_has_data", currentHash != "", "universe_entities", len(universeKeys))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "Comprehensive SDE",
		"status":    "success",
		"data":      comprehensiveStatus,
		"module":    m.Name(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// sdeAllEntityTypesHandler lists all available SDE entity types from Redis
func (m *Module) sdeAllEntityTypesHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.sdeAllEntityTypesHandler",
		attribute.String("dev.operation", "sde_all_entity_types"),
		attribute.String("dev.service", "redis"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: SDE all entity types request")

	redis := m.Redis()
	ctx := r.Context()

	// Get all SDE keys
	keys, err := redis.Client.Keys(ctx, "sde:*").Result()
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Failed to get SDE keys", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve SDE keys","details":"` + err.Error() + `"}`))
		return
	}

	// Parse keys to extract entity types
	entityTypes := make(map[string]int)
	metadataKeys := []string{"sde:current_hash", "sde:status", "sde:progress"}
	metadataCount := 0

	for _, key := range keys {
		// Skip metadata keys
		isMetadata := false
		for _, metaKey := range metadataKeys {
			if key == metaKey {
				isMetadata = true
				metadataCount++
				break
			}
		}
		if isMetadata {
			continue
		}

		// Parse key pattern: sde:{type}:{id} or sde:universe:{type}:...
		parts := strings.Split(key, ":")
		if len(parts) >= 3 {
			if parts[1] == "universe" && len(parts) >= 4 {
				// Universe data: sde:universe:{type}:...
				entityType := fmt.Sprintf("universe_%s", parts[2])
				entityTypes[entityType]++
			} else {
				// Regular entity: sde:{type}:{id}
				entityType := parts[1]
				entityTypes[entityType]++
			}
		}
	}

	// Prepare response
	var typesList []string
	for entityType := range entityTypes {
		typesList = append(typesList, entityType)
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("sde.total_keys", len(keys)),
		attribute.Int("sde.entity_types_count", len(entityTypes)),
		attribute.Int("sde.metadata_keys", metadataCount),
	)

	slog.InfoContext(r.Context(), "Dev: SDE entity types retrieved", 
		"total_keys", len(keys), "entity_types", len(entityTypes), "metadata_keys", metadataCount)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":              "Redis SDE",
		"status":              "success",
		"data": map[string]interface{}{
			"entity_types":        typesList,
			"entity_counts":       entityTypes,
			"total_keys":          len(keys),
			"metadata_keys":       metadataCount,
			"data_keys":           len(keys) - metadataCount,
		},
		"module":              m.Name(),
		"timestamp":           time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}