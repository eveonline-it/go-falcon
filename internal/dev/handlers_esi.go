package dev

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"go-falcon/pkg/evegateway/alliance"
	"go-falcon/pkg/evegateway/status"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
)

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
	result, err := m.characterClient.GetCharacterInfoWithCache(ctx, characterID)
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
		attribute.Bool("cache.hit", result.Cache.Cached),
	)

	slog.InfoContext(ctx, "Dev: Character info retrieved successfully", "character_id", characterID, "cached", result.Cache.Cached)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/characters/" + characterIDStr + "/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
	}

	// Add cache information
	if result.Cache.Cached && result.Cache.ExpiresAt != nil {
		response["cache"] = map[string]interface{}{
			"cached":     true,
			"expires_at": result.Cache.ExpiresAt.Format(time.RFC3339),
			"expires_in": int(time.Until(*result.Cache.ExpiresAt).Seconds()),
		}
	} else {
		response["cache"] = map[string]interface{}{
			"cached": false,
		}
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
	result, err := m.characterClient.GetCharacterPortraitWithCache(ctx, characterID)
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
		attribute.Bool("cache.hit", result.Cache.Cached),
	)

	slog.InfoContext(ctx, "Dev: Character portrait retrieved successfully", "character_id", characterID, "cached", result.Cache.Cached)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/characters/" + characterIDStr + "/portrait/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
	}

	// Add cache information
	if result.Cache.Cached && result.Cache.ExpiresAt != nil {
		response["cache"] = map[string]interface{}{
			"cached":     true,
			"expires_at": result.Cache.ExpiresAt.Format(time.RFC3339),
			"expires_in": int(time.Until(*result.Cache.ExpiresAt).Seconds()),
		}
	} else {
		response["cache"] = map[string]interface{}{
			"cached": false,
		}
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
	result, err := m.universeClient.GetSystemInfoWithCache(ctx, systemID)
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
		attribute.Bool("cache.hit", result.Cache.Cached),
	)

	slog.InfoContext(ctx, "Dev: System info retrieved successfully", "system_id", systemID, "cached", result.Cache.Cached)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/universe/systems/" + systemIDStr + "/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
	}

	// Add cache information
	if result.Cache.Cached && result.Cache.ExpiresAt != nil {
		response["cache"] = map[string]interface{}{
			"cached":     true,
			"expires_at": result.Cache.ExpiresAt.Format(time.RFC3339),
			"expires_in": int(time.Until(*result.Cache.ExpiresAt).Seconds()),
		}
	} else {
		response["cache"] = map[string]interface{}{
			"cached": false,
		}
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
	result, err := m.universeClient.GetStationInfoWithCache(ctx, stationID)
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
		attribute.Bool("cache.hit", result.Cache.Cached),
	)

	slog.InfoContext(ctx, "Dev: Station info retrieved successfully", "station_id", stationID, "cached", result.Cache.Cached)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/universe/stations/" + stationIDStr + "/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
	}

	// Add cache information
	if result.Cache.Cached && result.Cache.ExpiresAt != nil {
		response["cache"] = map[string]interface{}{
			"cached":     true,
			"expires_at": result.Cache.ExpiresAt.Format(time.RFC3339),
			"expires_in": int(time.Until(*result.Cache.ExpiresAt).Seconds()),
		}
	} else {
		response["cache"] = map[string]interface{}{
			"cached": false,
		}
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

// allianceContactsHandler gets alliance contacts from ESI (requires authentication)
func (m *Module) allianceContactsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.allianceContactsHandler",
		attribute.String("dev.operation", "alliance_contacts"),
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
		w.Write([]byte(`{"error":"Invalid alliance ID","details":"Alliance ID must be a valid integer"}`))
		return
	}

	// Extract token from Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Missing authorization token for alliance contacts", "alliance_id", allianceID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Authorization required","details":"This endpoint requires a valid EVE Online access token"}`))
		return
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	ctx := r.Context()
	contacts, err := m.allianceClient.GetAllianceContacts(ctx, allianceID, token)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get alliance contacts", "alliance_id", allianceID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		
		// Check if it's an auth error
		if err.Error() == "403" || err.Error() == "401" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Access denied","details":"Invalid token or insufficient permissions for alliance contacts"}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve alliance contacts","details":"` + err.Error() + `"}`))
		}
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int64("alliance.id", allianceID),
		attribute.Int("contacts.count", len(contacts)),
	)

	slog.InfoContext(ctx, "Dev: Alliance contacts retrieved successfully",
		"alliance_id", allianceID,
		"count", len(contacts))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/alliances/" + allianceIDStr + "/contacts",
		"status":   "success",
		"data":     contacts,
		"module":   m.Name(),
		"count":    len(contacts),
	}

	json.NewEncoder(w).Encode(response)
}

// allianceContactLabelsHandler gets alliance contact labels from ESI (requires authentication)
func (m *Module) allianceContactLabelsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.allianceContactLabelsHandler",
		attribute.String("dev.operation", "alliance_contact_labels"),
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
		w.Write([]byte(`{"error":"Invalid alliance ID","details":"Alliance ID must be a valid integer"}`))
		return
	}

	// Extract token from Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Missing authorization token for alliance contact labels", "alliance_id", allianceID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Authorization required","details":"This endpoint requires a valid EVE Online access token"}`))
		return
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	ctx := r.Context()
	labels, err := m.allianceClient.GetAllianceContactLabels(ctx, allianceID, token)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get alliance contact labels", "alliance_id", allianceID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		
		// Check if it's an auth error
		if err.Error() == "403" || err.Error() == "401" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Access denied","details":"Invalid token or insufficient permissions for alliance contact labels"}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve alliance contact labels","details":"` + err.Error() + `"}`))
		}
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int64("alliance.id", allianceID),
		attribute.Int("labels.count", len(labels)),
	)

	slog.InfoContext(ctx, "Dev: Alliance contact labels retrieved successfully",
		"alliance_id", allianceID,
		"count", len(labels))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/alliances/" + allianceIDStr + "/contacts/labels",
		"status":   "success",
		"data":     labels,
		"module":   m.Name(),
		"count":    len(labels),
	}

	json.NewEncoder(w).Encode(response)
}