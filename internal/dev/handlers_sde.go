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
		attribute.String("sde.npc_corp_ticker", npcCorp.TickerName),
	)

	slog.InfoContext(r.Context(), "Dev: SDE NPC corporation retrieved", "corp_id", corpID, "ticker", npcCorp.TickerName)

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