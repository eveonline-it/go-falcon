package evegate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go-falcon/pkg/config"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Client represents an EVE Online ESI client
type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
}

// ESIStatusResponse represents the EVE Online server status
type ESIStatusResponse struct {
	Players       int       `json:"players"`
	ServerVersion string    `json:"server_version"`
	StartTime     time.Time `json:"start_time"`
}

// NewClient creates a new EVE Online ESI client
func NewClient() *Client {
	var transport http.RoundTripper = http.DefaultTransport
	
	// Only add OpenTelemetry instrumentation if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		transport = otelhttp.NewTransport(http.DefaultTransport, 
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Host)
			}),
		)
	}
	
	return &Client{
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		baseURL:   "https://esi.evetech.net",
		userAgent: "go-falcon/1.0.0",
	}
}

// GetServerStatus retrieves EVE Online server status from ESI
func (c *Client) GetServerStatus(ctx context.Context) (*ESIStatusResponse, error) {
	var span trace.Span
	
	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		tracer := otel.Tracer("go-falcon/evegate")
		ctx, span = tracer.Start(ctx, "evegate.GetServerStatus")
		defer span.End()
		
		span.SetAttributes(
			attribute.String("esi.endpoint", "status"),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("http.user_agent", c.userAgent),
		)
	}
	
	slog.InfoContext(ctx, "Requesting server status from ESI")
	
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/status", nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create ESI status request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	
	if span != nil {
		span.SetAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", req.URL.String()),
		)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to call ESI")
		}
		slog.ErrorContext(ctx, "Failed to call ESI status endpoint", "error", err)
		return nil, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()
	
	if span != nil {
		span.SetAttributes(
			attribute.Int("http.status_code", resp.StatusCode),
			attribute.String("http.status_text", resp.Status),
		)
	}
	
	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI status endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read ESI status response", "error", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if span != nil {
		span.SetAttributes(attribute.Int("http.response_size", len(body)))
	}
	
	var status ESIStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse ESI status response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if span != nil {
		span.SetAttributes(
			attribute.Int("esi.players", status.Players),
			attribute.String("esi.server_version", status.ServerVersion),
			attribute.String("esi.start_time", status.StartTime.Format(time.RFC3339)),
		)
		span.SetStatus(codes.Ok, "successfully retrieved ESI status")
	}
	
	slog.InfoContext(ctx, "Successfully retrieved ESI status", 
		slog.Int("players", status.Players),
		slog.String("server_version", status.ServerVersion),
		slog.Time("start_time", status.StartTime))
	
	return &status, nil
}

// GetCharacterInfo retrieves character information from EVE ESI
func (c *Client) GetCharacterInfo(ctx context.Context, characterID int) (map[string]any, error) {
	slog.Info("Requesting character info from ESI", slog.Int("character_id", characterID))
	
	// TODO: Implement actual ESI call
	return map[string]any{"character_id": characterID, "name": "placeholder"}, nil
}

// GetCorporationInfo retrieves corporation information from EVE ESI
func (c *Client) GetCorporationInfo(ctx context.Context, corporationID int) (map[string]any, error) {
	slog.Info("Requesting corporation info from ESI", slog.Int("corporation_id", corporationID))
	
	// TODO: Implement actual ESI call
	return map[string]any{"corporation_id": corporationID, "name": "placeholder"}, nil
}

// GetAllianceInfo retrieves alliance information from EVE ESI
func (c *Client) GetAllianceInfo(ctx context.Context, allianceID int) (map[string]any, error) {
	slog.Info("Requesting alliance info from ESI", slog.Int("alliance_id", allianceID))
	
	// TODO: Implement actual ESI call
	return map[string]any{"alliance_id": allianceID, "name": "placeholder"}, nil
}

// RefreshToken refreshes an EVE SSO access token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (map[string]any, error) {
	slog.Info("Refreshing EVE SSO token")
	
	// TODO: Implement actual token refresh
	return map[string]any{"access_token": "placeholder", "expires_in": 1200}, nil
}