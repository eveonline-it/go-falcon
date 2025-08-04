package dev

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go-falcon/pkg/database"
	"go-falcon/pkg/evegate"
	"go-falcon/pkg/evegate/alliance"
	"go-falcon/pkg/evegate/character"
	"go-falcon/pkg/evegate/status"
	"go-falcon/pkg/evegate/universe"
	"go-falcon/pkg/handlers"
	"go-falcon/pkg/module"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
)

type Module struct {
	*module.BaseModule
	evegateClient    *evegate.Client
	statusClient     status.Client
	characterClient  character.Client
	universeClient   universe.Client
	allianceClient   alliance.Client
	cacheManager     evegate.CacheManager
}

func New(mongodb *database.MongoDB, redis *database.Redis) *Module {
	evegateClient := evegate.NewClient()
	
	// Create shared cache manager for consistency
	cacheManager := evegate.NewDefaultCacheManager()
	httpClient := &http.Client{Timeout: 30 * time.Second}
	baseURL := "https://esi.evetech.net"
	userAgent := "go-falcon/1.0.0 contact@example.com"
	
	errorLimits := &evegate.ESIErrorLimits{}
	limitsMutex := &sync.RWMutex{}
	retryClient := evegate.NewDefaultRetryClient(httpClient, errorLimits, limitsMutex)
	
	statusClient := status.NewStatusClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	characterClient := character.NewCharacterClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	universeClient := universe.NewUniverseClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	allianceClient := alliance.NewAllianceClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	
	return &Module{
		BaseModule:       module.NewBaseModule("dev", mongodb, redis),
		evegateClient:    evegateClient,
		statusClient:     statusClient,
		characterClient:  characterClient,
		universeClient:   universeClient,
		allianceClient:   allianceClient,
		cacheManager:     cacheManager,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	r.Get("/esi-status", m.esiStatusHandler)
	r.Get("/character/{characterID}", m.characterInfoHandler)
	r.Get("/character/{characterID}/portrait", m.characterPortraitHandler)
	r.Get("/universe/system/{systemID}", m.systemInfoHandler)
	r.Get("/universe/station/{stationID}", m.stationInfoHandler)
	r.Get("/alliances", m.alliancesHandler)
	r.Get("/alliance/{allianceID}", m.allianceInfoHandler)
	r.Get("/alliance/{allianceID}/corporations", m.allianceCorporationsHandler)
	r.Get("/alliance/{allianceID}/icons", m.allianceIconsHandler)
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
	
	// Check for cached data first using the shared cache manager
	cacheKey := "https://esi.evetech.net/status"
	cachedData, cached, expiry, _ := m.cacheManager.GetWithExpiry(cacheKey)
	
	var statusResponse *status.ServerStatusResponse
	var err error
	
	if cached {
		// Use cached data if available
		if err := json.Unmarshal(cachedData, &statusResponse); err == nil {
			span.SetAttributes(
				attribute.Bool("dev.success", true),
				attribute.Bool("cache.hit", true),
				attribute.Int("esi.players", statusResponse.Players),
				attribute.String("esi.server_version", statusResponse.ServerVersion),
			)
			slog.InfoContext(ctx, "Dev: ESI status retrieved from cache", 
				slog.Int("players", statusResponse.Players),
				slog.String("server_version", statusResponse.ServerVersion))
		} else {
			cached = false // Fall back to API call if unmarshal fails
		}
	}
	
	if !cached {
		// Make API call if not cached using the status client
		statusResponse, err = m.statusClient.GetServerStatus(ctx)
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
			attribute.Bool("cache.hit", false),
			attribute.Int("esi.players", statusResponse.Players),
			attribute.String("esi.server_version", statusResponse.ServerVersion),
		)
		
		slog.InfoContext(ctx, "Dev: ESI status retrieved from ESI", 
			slog.Int("players", statusResponse.Players),
			slog.String("server_version", statusResponse.ServerVersion))
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"source":         "EVE Online ESI",
		"endpoint":       "https://esi.evetech.net/status",
		"status":         "success",
		"data":           statusResponse,
		"timestamp":      statusResponse.StartTime,
		"module":         m.Name(),
	}
	
	// Add cache information
	addCacheInfoToResponse(response, cached, expiry)
	
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
			"/character/{characterID} - Get character information",
			"/character/{characterID}/portrait - Get character portrait URLs",
			"/universe/system/{systemID} - Get solar system information",
			"/universe/station/{stationID} - Get station information",
			"/alliances - Get all active alliances",
			"/alliance/{allianceID} - Get alliance information",
			"/alliance/{allianceID}/corporations - Get alliance member corporations",
			"/alliance/{allianceID}/icons - Get alliance icon URLs",
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

// characterInfoHandler gets character information from ESI
func (m *Module) characterInfoHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.characterInfoHandler",
		attribute.String("dev.operation", "character_info"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	characterIDStr := chi.URLParam(r, "characterID")
	characterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid character ID", "character_id", characterIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid character ID"}`))
		return
	}

	ctx := r.Context()
	characterInfo, err := m.characterClient.GetCharacterInfo(ctx, characterID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get character info", "character_id", characterID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve character information","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("character.id", characterID),
	)

	slog.InfoContext(ctx, "Dev: Character info retrieved successfully", "character_id", characterID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "EVE Online ESI",
		"endpoint":  "/characters/" + characterIDStr + "/",
		"status":    "success",
		"data":      characterInfo,
		"module":    m.Name(),
		"cache": map[string]interface{}{
			"cached": false,
			"note":   "Character endpoints not yet implemented with caching",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// characterPortraitHandler gets character portrait URLs from ESI
func (m *Module) characterPortraitHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.characterPortraitHandler",
		attribute.String("dev.operation", "character_portrait"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	characterIDStr := chi.URLParam(r, "characterID")
	characterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid character ID", "character_id", characterIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid character ID"}`))
		return
	}

	ctx := r.Context()
	portraitData, err := m.characterClient.GetCharacterPortrait(ctx, characterID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get character portrait", "character_id", characterID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve character portrait","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("character.id", characterID),
	)

	slog.InfoContext(ctx, "Dev: Character portrait retrieved successfully", "character_id", characterID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "EVE Online ESI",
		"endpoint":  "/characters/" + characterIDStr + "/portrait/",
		"status":    "success",
		"data":      portraitData,
		"module":    m.Name(),
		"cache": map[string]interface{}{
			"cached": false,
			"note":   "Character endpoints not yet implemented with caching",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// systemInfoHandler gets solar system information from ESI
func (m *Module) systemInfoHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.systemInfoHandler",
		attribute.String("dev.operation", "system_info"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	systemIDStr := chi.URLParam(r, "systemID")
	systemID, err := strconv.Atoi(systemIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid system ID", "system_id", systemIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid system ID"}`))
		return
	}

	ctx := r.Context()
	systemInfo, err := m.universeClient.GetSystemInfo(ctx, systemID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get system info", "system_id", systemID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve system information","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("system.id", systemID),
	)

	slog.InfoContext(ctx, "Dev: System info retrieved successfully", "system_id", systemID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "EVE Online ESI",
		"endpoint":  "/universe/systems/" + systemIDStr + "/",
		"status":    "success",
		"data":      systemInfo,
		"module":    m.Name(),
		"cache": map[string]interface{}{
			"cached": false,
			"note":   "Universe endpoints not yet implemented with caching",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// stationInfoHandler gets station information from ESI
func (m *Module) stationInfoHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.stationInfoHandler",
		attribute.String("dev.operation", "station_info"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	stationIDStr := chi.URLParam(r, "stationID")
	stationID, err := strconv.Atoi(stationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid station ID", "station_id", stationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid station ID"}`))
		return
	}

	ctx := r.Context()
	stationInfo, err := m.universeClient.GetStationInfo(ctx, stationID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get station info", "station_id", stationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve station information","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("station.id", stationID),
	)

	slog.InfoContext(ctx, "Dev: Station info retrieved successfully", "station_id", stationID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "EVE Online ESI",
		"endpoint":  "/universe/stations/" + stationIDStr + "/",
		"status":    "success",
		"data":      stationInfo,
		"module":    m.Name(),
		"cache": map[string]interface{}{
			"cached": false,
			"note":   "Universe endpoints not yet implemented with caching",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// alliancesHandler gets all active alliances from ESI
func (m *Module) alliancesHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.alliancesHandler",
		attribute.String("dev.operation", "alliances_list"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	slog.InfoContext(r.Context(), "Dev: Alliances list request", slog.String("remote_addr", r.RemoteAddr))

	ctx := r.Context()
	cacheKey := "https://esi.evetech.net/alliances"
	cachedData, cached, expiry, _ := m.cacheManager.GetWithExpiry(cacheKey)

	var alliances []int64
	var err error

	if cached {
		// Use cached data if available
		if err := json.Unmarshal(cachedData, &alliances); err == nil {
			span.SetAttributes(
				attribute.Bool("dev.success", true),
				attribute.Bool("cache.hit", true),
				attribute.Int("alliance.count", len(alliances)),
			)
			slog.InfoContext(ctx, "Dev: Alliances retrieved from cache",
				slog.Int("count", len(alliances)))
		} else {
			cached = false // Fall back to API call if unmarshal fails
		}
	}

	if !cached {
		// Make API call if not cached
		alliances, err = m.allianceClient.GetAlliances(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.Bool("dev.success", false))
			slog.ErrorContext(ctx, "Dev: Failed to get alliances", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve alliances","details":"` + err.Error() + `"}`))
			return
		}

		span.SetAttributes(
			attribute.Bool("dev.success", true),
			attribute.Bool("cache.hit", false),
			attribute.Int("alliance.count", len(alliances)),
		)

		slog.InfoContext(ctx, "Dev: Alliances retrieved from ESI",
			slog.Int("count", len(alliances)))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "EVE Online ESI",
		"endpoint":  "/alliances",
		"status":    "success",
		"data":      alliances,
		"module":    m.Name(),
		"count":     len(alliances),
	}

	// Add cache information
	addCacheInfoToResponse(response, cached, expiry)

	json.NewEncoder(w).Encode(response)
}

// allianceInfoHandler gets alliance information from ESI
func (m *Module) allianceInfoHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.allianceInfoHandler",
		attribute.String("dev.operation", "alliance_info"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	allianceIDStr := chi.URLParam(r, "allianceID")
	allianceID, err := strconv.ParseInt(allianceIDStr, 10, 64)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid alliance ID", "alliance_id", allianceIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid alliance ID"}`))
		return
	}

	ctx := r.Context()
	cacheKey := "https://esi.evetech.net/alliances/" + allianceIDStr
	cachedData, cached, expiry, _ := m.cacheManager.GetWithExpiry(cacheKey)

	var allianceInfo *alliance.AllianceInfoResponse
	if cached {
		// Use cached data if available
		if err := json.Unmarshal(cachedData, &allianceInfo); err == nil {
			span.SetAttributes(
				attribute.Bool("dev.success", true),
				attribute.Bool("cache.hit", true),
				attribute.Int64("alliance.id", allianceID),
			)
			slog.InfoContext(ctx, "Dev: Alliance info retrieved from cache", "alliance_id", allianceID)
		} else {
			cached = false // Fall back to API call if unmarshal fails
		}
	}

	if !cached {
		// Make API call if not cached
		allianceInfo, err = m.allianceClient.GetAllianceInfo(ctx, allianceID)
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.Bool("dev.success", false))
			slog.ErrorContext(ctx, "Dev: Failed to get alliance info", "alliance_id", allianceID, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve alliance information","details":"` + err.Error() + `"}`))
			return
		}

		span.SetAttributes(
			attribute.Bool("dev.success", true),
			attribute.Bool("cache.hit", false),
			attribute.Int64("alliance.id", allianceID),
			attribute.String("alliance.name", allianceInfo.Name),
		)

		slog.InfoContext(ctx, "Dev: Alliance info retrieved from ESI",
			"alliance_id", allianceID,
			"name", allianceInfo.Name)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "EVE Online ESI",
		"endpoint":  "/alliances/" + allianceIDStr,
		"status":    "success",
		"data":      allianceInfo,
		"module":    m.Name(),
	}

	// Add cache information
	addCacheInfoToResponse(response, cached, expiry)

	json.NewEncoder(w).Encode(response)
}

// allianceCorporationsHandler gets alliance member corporations from ESI
func (m *Module) allianceCorporationsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.allianceCorporationsHandler",
		attribute.String("dev.operation", "alliance_corporations"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	allianceIDStr := chi.URLParam(r, "allianceID")
	allianceID, err := strconv.ParseInt(allianceIDStr, 10, 64)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid alliance ID", "alliance_id", allianceIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid alliance ID"}`))
		return
	}

	ctx := r.Context()
	cacheKey := "https://esi.evetech.net/alliances/" + allianceIDStr + "/corporations"
	cachedData, cached, expiry, _ := m.cacheManager.GetWithExpiry(cacheKey)

	var corporations []int64
	if cached {
		// Use cached data if available
		if err := json.Unmarshal(cachedData, &corporations); err == nil {
			span.SetAttributes(
				attribute.Bool("dev.success", true),
				attribute.Bool("cache.hit", true),
				attribute.Int64("alliance.id", allianceID),
				attribute.Int("corporations.count", len(corporations)),
			)
			slog.InfoContext(ctx, "Dev: Alliance corporations retrieved from cache",
				"alliance_id", allianceID,
				"count", len(corporations))
		} else {
			cached = false // Fall back to API call if unmarshal fails
		}
	}

	if !cached {
		// Make API call if not cached
		corporations, err = m.allianceClient.GetAllianceCorporations(ctx, allianceID)
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.Bool("dev.success", false))
			slog.ErrorContext(ctx, "Dev: Failed to get alliance corporations", "alliance_id", allianceID, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve alliance corporations","details":"` + err.Error() + `"}`))
			return
		}

		span.SetAttributes(
			attribute.Bool("dev.success", true),
			attribute.Bool("cache.hit", false),
			attribute.Int64("alliance.id", allianceID),
			attribute.Int("corporations.count", len(corporations)),
		)

		slog.InfoContext(ctx, "Dev: Alliance corporations retrieved from ESI",
			"alliance_id", allianceID,
			"count", len(corporations))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "EVE Online ESI",
		"endpoint":  "/alliances/" + allianceIDStr + "/corporations",
		"status":    "success",
		"data":      corporations,
		"module":    m.Name(),
		"count":     len(corporations),
	}

	// Add cache information
	addCacheInfoToResponse(response, cached, expiry)

	json.NewEncoder(w).Encode(response)
}

// allianceIconsHandler gets alliance icon URLs from ESI
func (m *Module) allianceIconsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.allianceIconsHandler",
		attribute.String("dev.operation", "alliance_icons"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	allianceIDStr := chi.URLParam(r, "allianceID")
	allianceID, err := strconv.ParseInt(allianceIDStr, 10, 64)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid alliance ID", "alliance_id", allianceIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid alliance ID"}`))
		return
	}

	ctx := r.Context()
	cacheKey := "https://esi.evetech.net/alliances/" + allianceIDStr + "/icons"
	cachedData, cached, expiry, _ := m.cacheManager.GetWithExpiry(cacheKey)

	var icons *alliance.AllianceIconsResponse
	if cached {
		// Use cached data if available
		if err := json.Unmarshal(cachedData, &icons); err == nil {
			span.SetAttributes(
				attribute.Bool("dev.success", true),
				attribute.Bool("cache.hit", true),
				attribute.Int64("alliance.id", allianceID),
			)
			slog.InfoContext(ctx, "Dev: Alliance icons retrieved from cache", "alliance_id", allianceID)
		} else {
			cached = false // Fall back to API call if unmarshal fails
		}
	}

	if !cached {
		// Make API call if not cached
		icons, err = m.allianceClient.GetAllianceIcons(ctx, allianceID)
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.Bool("dev.success", false))
			slog.ErrorContext(ctx, "Dev: Failed to get alliance icons", "alliance_id", allianceID, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve alliance icons","details":"` + err.Error() + `"}`))
			return
		}

		span.SetAttributes(
			attribute.Bool("dev.success", true),
			attribute.Bool("cache.hit", false),
			attribute.Int64("alliance.id", allianceID),
		)

		slog.InfoContext(ctx, "Dev: Alliance icons retrieved from ESI", "alliance_id", allianceID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":    "EVE Online ESI",
		"endpoint":  "/alliances/" + allianceIDStr + "/icons",
		"status":    "success",
		"data":      icons,
		"module":    m.Name(),
	}

	// Add cache information
	addCacheInfoToResponse(response, cached, expiry)

	json.NewEncoder(w).Encode(response)
}