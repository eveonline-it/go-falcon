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

// corporationInfoHandler gets corporation information from ESI
func (m *Module) corporationInfoHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.corporationInfoHandler",
		attribute.String("dev.operation", "corporation_info"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	corporationIDStr := chi.URLParam(r, "corporationID")
	corporationID, err := strconv.Atoi(corporationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid corporation ID", "corporation_id", corporationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid corporation ID","details":"Corporation ID must be a valid integer"}`))
		return
	}

	ctx := r.Context()
	result, err := m.corporationClient.GetCorporationInfoWithCache(ctx, corporationID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get corporation info", "corporation_id", corporationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve corporation information","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("corporation.id", corporationID),
		attribute.Bool("cache.hit", result.Cache.Cached),
	)

	slog.InfoContext(ctx, "Dev: Corporation info retrieved successfully", "corporation_id", corporationID, "cached", result.Cache.Cached)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/corporations/" + corporationIDStr + "/",
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

// corporationIconsHandler gets corporation icons from ESI
func (m *Module) corporationIconsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.corporationIconsHandler",
		attribute.String("dev.operation", "corporation_icons"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	corporationIDStr := chi.URLParam(r, "corporationID")
	corporationID, err := strconv.Atoi(corporationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid corporation ID", "corporation_id", corporationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid corporation ID","details":"Corporation ID must be a valid integer"}`))
		return
	}

	ctx := r.Context()
	result, err := m.corporationClient.GetCorporationIconsWithCache(ctx, corporationID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get corporation icons", "corporation_id", corporationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve corporation icons","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("corporation.id", corporationID),
		attribute.Bool("cache.hit", result.Cache.Cached),
	)

	slog.InfoContext(ctx, "Dev: Corporation icons retrieved successfully", "corporation_id", corporationID, "cached", result.Cache.Cached)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/corporations/" + corporationIDStr + "/icons/",
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

// corporationAllianceHistoryHandler gets corporation alliance history from ESI
func (m *Module) corporationAllianceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.corporationAllianceHistoryHandler",
		attribute.String("dev.operation", "corporation_alliance_history"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	corporationIDStr := chi.URLParam(r, "corporationID")
	corporationID, err := strconv.Atoi(corporationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid corporation ID", "corporation_id", corporationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid corporation ID","details":"Corporation ID must be a valid integer"}`))
		return
	}

	ctx := r.Context()
	result, err := m.corporationClient.GetCorporationAllianceHistoryWithCache(ctx, corporationID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get corporation alliance history", "corporation_id", corporationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to retrieve corporation alliance history","details":"` + err.Error() + `"}`))
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("corporation.id", corporationID),
		attribute.Bool("cache.hit", result.Cache.Cached),
		attribute.Int("history.count", len(result.Data)),
	)

	slog.InfoContext(ctx, "Dev: Corporation alliance history retrieved successfully", "corporation_id", corporationID, "cached", result.Cache.Cached, "count", len(result.Data))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/corporations/" + corporationIDStr + "/alliancehistory/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
		"count":    len(result.Data),
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

// corporationMembersHandler gets corporation members from ESI (requires authentication)
func (m *Module) corporationMembersHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.corporationMembersHandler",
		attribute.String("dev.operation", "corporation_members"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	corporationIDStr := chi.URLParam(r, "corporationID")
	corporationID, err := strconv.Atoi(corporationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid corporation ID", "corporation_id", corporationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid corporation ID","details":"Corporation ID must be a valid integer"}`))
		return
	}

	// Extract token from Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Missing authorization token for corporation members", "corporation_id", corporationID)
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
	result, err := m.corporationClient.GetCorporationMembersWithCache(ctx, corporationID, token)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get corporation members", "corporation_id", corporationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		
		// Check if it's an auth error
		if err.Error() == "403" || err.Error() == "401" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Access denied","details":"Invalid token or insufficient permissions for corporation members"}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve corporation members","details":"` + err.Error() + `"}`))
		}
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("corporation.id", corporationID),
		attribute.Bool("cache.hit", result.Cache.Cached),
		attribute.Int("members.count", len(result.Data)),
	)

	slog.InfoContext(ctx, "Dev: Corporation members retrieved successfully",
		"corporation_id", corporationID,
		"cached", result.Cache.Cached,
		"count", len(result.Data))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/corporations/" + corporationIDStr + "/members/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
		"count":    len(result.Data),
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

// corporationMemberTrackingHandler gets corporation member tracking from ESI (requires authentication)
func (m *Module) corporationMemberTrackingHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.corporationMemberTrackingHandler",
		attribute.String("dev.operation", "corporation_member_tracking"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	corporationIDStr := chi.URLParam(r, "corporationID")
	corporationID, err := strconv.Atoi(corporationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid corporation ID", "corporation_id", corporationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid corporation ID","details":"Corporation ID must be a valid integer"}`))
		return
	}

	// Extract token from Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Missing authorization token for corporation member tracking", "corporation_id", corporationID)
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
	result, err := m.corporationClient.GetCorporationMemberTrackingWithCache(ctx, corporationID, token)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get corporation member tracking", "corporation_id", corporationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		
		// Check if it's an auth error
		if err.Error() == "403" || err.Error() == "401" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Access denied","details":"Invalid token or insufficient permissions for corporation member tracking"}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve corporation member tracking","details":"` + err.Error() + `"}`))
		}
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("corporation.id", corporationID),
		attribute.Bool("cache.hit", result.Cache.Cached),
		attribute.Int("tracking.count", len(result.Data)),
	)

	slog.InfoContext(ctx, "Dev: Corporation member tracking retrieved successfully",
		"corporation_id", corporationID,
		"cached", result.Cache.Cached,
		"count", len(result.Data))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/corporations/" + corporationIDStr + "/membertracking/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
		"count":    len(result.Data),
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

// corporationMemberRolesHandler gets corporation member roles from ESI (requires authentication)
func (m *Module) corporationMemberRolesHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.corporationMemberRolesHandler",
		attribute.String("dev.operation", "corporation_member_roles"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	corporationIDStr := chi.URLParam(r, "corporationID")
	corporationID, err := strconv.Atoi(corporationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid corporation ID", "corporation_id", corporationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid corporation ID","details":"Corporation ID must be a valid integer"}`))
		return
	}

	// Extract token from Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Missing authorization token for corporation member roles", "corporation_id", corporationID)
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
	result, err := m.corporationClient.GetCorporationMemberRolesWithCache(ctx, corporationID, token)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get corporation member roles", "corporation_id", corporationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		
		// Check if it's an auth error
		if err.Error() == "403" || err.Error() == "401" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Access denied","details":"Invalid token or insufficient permissions for corporation member roles"}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve corporation member roles","details":"` + err.Error() + `"}`))
		}
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("corporation.id", corporationID),
		attribute.Bool("cache.hit", result.Cache.Cached),
		attribute.Int("roles.count", len(result.Data)),
	)

	slog.InfoContext(ctx, "Dev: Corporation member roles retrieved successfully",
		"corporation_id", corporationID,
		"cached", result.Cache.Cached,
		"count", len(result.Data))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/corporations/" + corporationIDStr + "/roles/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
		"count":    len(result.Data),
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

// corporationStructuresHandler gets corporation structures from ESI (requires authentication)
func (m *Module) corporationStructuresHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.corporationStructuresHandler",
		attribute.String("dev.operation", "corporation_structures"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	corporationIDStr := chi.URLParam(r, "corporationID")
	corporationID, err := strconv.Atoi(corporationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid corporation ID", "corporation_id", corporationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid corporation ID","details":"Corporation ID must be a valid integer"}`))
		return
	}

	// Extract token from Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Missing authorization token for corporation structures", "corporation_id", corporationID)
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
	result, err := m.corporationClient.GetCorporationStructuresWithCache(ctx, corporationID, token)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get corporation structures", "corporation_id", corporationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		
		// Check if it's an auth error
		if err.Error() == "403" || err.Error() == "401" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Access denied","details":"Invalid token or insufficient permissions for corporation structures"}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve corporation structures","details":"` + err.Error() + `"}`))
		}
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("corporation.id", corporationID),
		attribute.Bool("cache.hit", result.Cache.Cached),
		attribute.Int("structures.count", len(result.Data)),
	)

	slog.InfoContext(ctx, "Dev: Corporation structures retrieved successfully",
		"corporation_id", corporationID,
		"cached", result.Cache.Cached,
		"count", len(result.Data))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/corporations/" + corporationIDStr + "/structures/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
		"count":    len(result.Data),
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

// corporationStandingsHandler gets corporation standings from ESI (requires authentication)
func (m *Module) corporationStandingsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.corporationStandingsHandler",
		attribute.String("dev.operation", "corporation_standings"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	corporationIDStr := chi.URLParam(r, "corporationID")
	corporationID, err := strconv.Atoi(corporationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid corporation ID", "corporation_id", corporationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid corporation ID","details":"Corporation ID must be a valid integer"}`))
		return
	}

	// Extract token from Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Missing authorization token for corporation standings", "corporation_id", corporationID)
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
	result, err := m.corporationClient.GetCorporationStandingsWithCache(ctx, corporationID, token)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get corporation standings", "corporation_id", corporationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		
		// Check if it's an auth error
		if err.Error() == "403" || err.Error() == "401" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Access denied","details":"Invalid token or insufficient permissions for corporation standings"}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve corporation standings","details":"` + err.Error() + `"}`))
		}
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("corporation.id", corporationID),
		attribute.Bool("cache.hit", result.Cache.Cached),
		attribute.Int("standings.count", len(result.Data)),
	)

	slog.InfoContext(ctx, "Dev: Corporation standings retrieved successfully",
		"corporation_id", corporationID,
		"cached", result.Cache.Cached,
		"count", len(result.Data))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/corporations/" + corporationIDStr + "/standings/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
		"count":    len(result.Data),
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

// corporationWalletsHandler gets corporation wallets from ESI (requires authentication)
func (m *Module) corporationWalletsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "dev.corporationWalletsHandler",
		attribute.String("dev.operation", "corporation_wallets"),
		attribute.String("dev.service", "evegate"),
		attribute.String("http.route", r.URL.Path),
		attribute.String("http.method", r.Method),
	)
	defer span.End()

	corporationIDStr := chi.URLParam(r, "corporationID")
	corporationID, err := strconv.Atoi(corporationIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Invalid corporation ID", "corporation_id", corporationIDStr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid corporation ID","details":"Corporation ID must be a valid integer"}`))
		return
	}

	// Extract token from Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(r.Context(), "Dev: Missing authorization token for corporation wallets", "corporation_id", corporationID)
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
	result, err := m.corporationClient.GetCorporationWalletsWithCache(ctx, corporationID, token)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("dev.success", false))
		slog.ErrorContext(ctx, "Dev: Failed to get corporation wallets", "corporation_id", corporationID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		
		// Check if it's an auth error
		if err.Error() == "403" || err.Error() == "401" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Access denied","details":"Invalid token or insufficient permissions for corporation wallets"}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Failed to retrieve corporation wallets","details":"` + err.Error() + `"}`))
		}
		return
	}

	span.SetAttributes(
		attribute.Bool("dev.success", true),
		attribute.Int("corporation.id", corporationID),
		attribute.Bool("cache.hit", result.Cache.Cached),
		attribute.Int("wallets.count", len(result.Data)),
	)

	slog.InfoContext(ctx, "Dev: Corporation wallets retrieved successfully",
		"corporation_id", corporationID,
		"cached", result.Cache.Cached,
		"count", len(result.Data))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"source":   "EVE Online ESI",
		"endpoint": "/corporations/" + corporationIDStr + "/wallets/",
		"status":   "success",
		"data":     result.Data,
		"module":   m.Name(),
		"count":    len(result.Data),
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