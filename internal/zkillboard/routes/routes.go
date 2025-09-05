package routes

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/zkillboard/dto"
	"go-falcon/internal/zkillboard/services"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Routes handles HTTP endpoints for the ZKillboard module
type Routes struct {
	consumer   *services.RedisQConsumer
	repository *services.Repository
	aggregator *services.Aggregator
}

// NewRoutes creates a new Routes instance
func NewRoutes(
	consumer *services.RedisQConsumer,
	repository *services.Repository,
	aggregator *services.Aggregator,
) *Routes {
	return &Routes{
		consumer:   consumer,
		repository: repository,
		aggregator: aggregator,
	}
}

// RegisterRoutes registers all ZKillboard routes
func (r *Routes) RegisterRoutes(api huma.API) {
	// Module status endpoint (public)
	huma.Register(api, huma.Operation{
		OperationID: "getZKillboardStatus",
		Method:      http.MethodGet,
		Path:        "/zkillboard/status",
		Summary:     "Get ZKillboard service status",
		Description: "Returns the current status of the ZKillboard RedisQ consumer service",
		Tags:        []string{"Module Status", "ZKillboard"},
		Security:    []map[string][]string{}, // Public endpoint
	}, r.GetStatus)

	// Service control endpoints (require authentication)
	huma.Register(api, huma.Operation{
		OperationID: "controlZKillboardService",
		Method:      http.MethodPost,
		Path:        "/zkillboard/control",
		Summary:     "Control ZKillboard service",
		Description: "Start, stop, or restart the ZKillboard RedisQ consumer service",
		Tags:        []string{"ZKillboard"},
		Security:    []map[string][]string{{"bearer": {}}, {"cookie": {}}},
		// TODO: Add proper authentication middleware
		// Middlewares: huma.Middlewares{
		//     middleware.RequireAuthentication(),
		//     middleware.RequirePermission("zkillboard:control"),
		// },
	}, r.ControlService)

	// Statistics endpoints
	huma.Register(api, huma.Operation{
		OperationID: "getZKillboardStats",
		Method:      http.MethodGet,
		Path:        "/zkillboard/stats",
		Summary:     "Get killmail statistics",
		Description: "Returns aggregated statistics for processed killmails",
		Tags:        []string{"ZKillboard"},
		Security:    []map[string][]string{}, // Public endpoint
	}, r.GetStats)

	// Recent killmails endpoint
	huma.Register(api, huma.Operation{
		OperationID: "getRecentKillmails",
		Method:      http.MethodGet,
		Path:        "/zkillboard/recent",
		Summary:     "Get recent killmails",
		Description: "Returns recently processed killmails from ZKillboard",
		Tags:        []string{"ZKillboard"},
		Security:    []map[string][]string{}, // Public endpoint
	}, r.GetRecentKillmails)
}

// GetStatusInput represents query parameters for status endpoint
type GetStatusInput struct{}

// GetStatus returns the current service status
func (r *Routes) GetStatus(ctx context.Context, input *GetStatusInput) (*dto.ServiceStatusOutput, error) {
	status := r.consumer.GetStatus()
	return status, nil
}

// ControlServiceBody represents the request body for service control
type ControlServiceBody struct {
	Body dto.ServiceControlInput `json:"body" required:"true"`
}

// ControlService handles service control operations
func (r *Routes) ControlService(ctx context.Context, input *ControlServiceBody) (*dto.ServiceControlOutput, error) {
	var message string
	var success bool

	switch input.Body.Action {
	case "start":
		err := r.consumer.Start(ctx)
		if err != nil {
			message = "Failed to start service: " + err.Error()
			success = false
		} else {
			message = "Service started successfully"
			success = true
		}

	case "stop":
		err := r.consumer.Stop()
		if err != nil {
			message = "Failed to stop service: " + err.Error()
			success = false
		} else {
			message = "Service stopped successfully"
			success = true
		}

	case "restart":
		// Stop if running
		_ = r.consumer.Stop()
		time.Sleep(1 * time.Second)

		// Start again
		err := r.consumer.Start(ctx)
		if err != nil {
			message = "Failed to restart service: " + err.Error()
			success = false
		} else {
			message = "Service restarted successfully"
			success = true
		}

	default:
		return nil, huma.Error400BadRequest("invalid action: " + input.Body.Action)
	}

	status := r.consumer.GetStatus()

	return &dto.ServiceControlOutput{
		Body: dto.ServiceControlResponse{
			Success: success,
			Message: message,
			Status:  status.Body.Status,
		},
	}, nil
}

// GetStatsInput represents query parameters for stats endpoint
type GetStatsInput struct {
	Period string `query:"period" enum:"hour,day,week,month" default:"day" doc:"Time period for statistics"`
}

// GetStats returns killmail statistics
func (r *Routes) GetStats(ctx context.Context, input *GetStatsInput) (*dto.ServiceStatsOutput, error) {
	// Calculate time range based on period
	end := time.Now()
	var start time.Time

	switch input.Period {
	case "hour":
		start = end.Add(-1 * time.Hour)
	case "day":
		start = end.Add(-24 * time.Hour)
	case "week":
		start = end.Add(-7 * 24 * time.Hour)
	case "month":
		start = end.Add(-30 * 24 * time.Hour)
	default:
		start = end.Add(-24 * time.Hour)
	}

	// Get basic statistics
	stats, err := r.repository.GetStats(ctx, input.Period, start, end)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get stats: " + err.Error())
	}

	// Get top systems
	topSystems, err := r.aggregator.GetTopSystems(ctx, input.Period, start, end, 10)
	if err != nil {
		slog.Warn("Failed to get top systems", "error", err)
		topSystems = []bson.M{}
	}

	// Get top alliances
	topAlliances, err := r.aggregator.GetTopAlliances(ctx, input.Period, start, end, 10)
	if err != nil {
		slog.Warn("Failed to get top alliances", "error", err)
		topAlliances = []bson.M{}
	}

	// Get top ship types
	topShipTypes, err := r.aggregator.GetTopShipTypes(ctx, input.Period, start, end, 10)
	if err != nil {
		slog.Warn("Failed to get top ship types", "error", err)
		topShipTypes = []bson.M{}
	}

	// Convert to output format
	output := &dto.ServiceStatsOutput{
		Body: dto.ServiceStatsResponse{
			Period:         input.Period,
			TotalKillmails: getInt64(stats, "total_killmails"),
			TotalValue:     getFloat64(stats, "total_value"),
			NPCKills:       getInt64(stats, "npc_kills"),
			SoloKills:      getInt64(stats, "solo_kills"),
			TopSystems:     convertSystemStats(topSystems),
			TopAlliances:   convertAllianceStats(topAlliances),
			TopShipTypes:   convertShipTypeStats(topShipTypes),
		},
	}

	return output, nil
}

// GetRecentKillmailsInput represents query parameters for recent killmails
type GetRecentKillmailsInput struct {
	Limit int `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Number of killmails to return"`
}

// GetRecentKillmails returns recently processed killmails
func (r *Routes) GetRecentKillmails(ctx context.Context, input *GetRecentKillmailsInput) (*dto.RecentKillmailsOutput, error) {
	// Get recent killmails from repository
	killmails, err := r.repository.GetRecentKillmails(ctx, input.Limit)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get recent killmails: " + err.Error())
	}

	// Convert to output format
	summaries := make([]dto.KillmailSummary, 0, len(killmails))
	for _, km := range killmails {
		summary := dto.KillmailSummary{
			KillmailID:    getInt64(km, "killmail_id"),
			Timestamp:     getTime(km, "timestamp"),
			SolarSystemID: getInt32(km, "solar_system_id"),
			TotalValue:    getFloat64(km, "total_value"),
			Points:        getInt(km, "points"),
			Solo:          getBool(km, "solo"),
			NPC:           getBool(km, "npc"),
			Href:          getString(km, "href"),
		}

		// Extract victim information
		if victim, ok := km["victim"].(map[string]interface{}); ok {
			if charID := getInt32Ptr(victim, "character_id"); charID != nil {
				summary.VictimID = charID
			}
			summary.ShipTypeID = getInt32(victim, "ship_type_id")
		}

		// TODO: Resolve names from database or SDE
		summary.SystemName = "Unknown System"
		summary.VictimName = "Unknown"
		summary.ShipTypeName = "Unknown Ship"

		summaries = append(summaries, summary)
	}

	return &dto.RecentKillmailsOutput{
		Body: dto.RecentKillmailsResponse{
			Killmails: summaries,
			Count:     len(summaries),
		},
	}, nil
}

// Helper functions for extracting values from bson.M
func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int64:
			return val
		case int32:
			return int64(val)
		case int:
			return int64(val)
		case float64:
			return int64(val)
		}
	}
	return 0
}

func getInt32(m map[string]interface{}, key string) int32 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int32:
			return val
		case int64:
			return int32(val)
		case int:
			return int32(val)
		case float64:
			return int32(val)
		}
	}
	return 0
}

func getInt32Ptr(m map[string]interface{}, key string) *int32 {
	if v, ok := m[key]; ok && v != nil {
		val := getInt32(m, key)
		return &val
	}
	return nil
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int32:
			return int(val)
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return 0
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getTime(m map[string]interface{}, key string) time.Time {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case time.Time:
			return val
		case primitive.DateTime:
			return val.Time()
		}
	}
	return time.Time{}
}

// Convert aggregation results to output DTOs
func convertSystemStats(systems []bson.M) []dto.SystemStats {
	result := make([]dto.SystemStats, 0, len(systems))
	for _, sys := range systems {
		result = append(result, dto.SystemStats{
			SystemID:   getInt32(sys, "system_id"),
			SystemName: getString(sys, "system_name"),
			Kills:      getInt64(sys, "kills"),
			Value:      getFloat64(sys, "value"),
		})
	}
	return result
}

func convertAllianceStats(alliances []bson.M) []dto.AllianceStats {
	result := make([]dto.AllianceStats, 0, len(alliances))
	for _, all := range alliances {
		result = append(result, dto.AllianceStats{
			AllianceID:   getInt32(all, "alliance_id"),
			AllianceName: getString(all, "alliance_name"),
			Kills:        getInt64(all, "kills"),
			Losses:       getInt64(all, "losses"),
			Value:        getFloat64(all, "value"),
		})
	}
	return result
}

func convertShipTypeStats(shipTypes []bson.M) []dto.ShipTypeStats {
	result := make([]dto.ShipTypeStats, 0, len(shipTypes))
	for _, ship := range shipTypes {
		result = append(result, dto.ShipTypeStats{
			ShipTypeID:   getInt32(ship, "ship_type_id"),
			ShipTypeName: getString(ship, "ship_type_name"),
			Destroyed:    getInt64(ship, "destroyed"),
			Value:        getFloat64(ship, "value"),
		})
	}
	return result
}
