package routes

import (
	"context"
	"net/http"

	"go-falcon/internal/killmails/dto"
	"go-falcon/internal/killmails/services"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterKillmailRoutes registers killmail-related routes
func RegisterKillmailRoutes(api huma.API, basePath string, service *services.Service) {

	// Import killmail by ID and hash (public)
	huma.Register(api, huma.Operation{
		OperationID:   "importKillmail",
		Method:        http.MethodPost,
		Path:          basePath + "/import",
		Summary:       "Import a killmail",
		Description:   "Imports a killmail by fetching it from ESI using the provided ID and hash, then stores it in the database.",
		Tags:          []string{"Killmails"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *dto.ImportKillmailInput) (*dto.KillmailOutput, error) {
		killmail, err := service.ImportKillmail(ctx, input.Body.KillmailID, input.Body.Hash)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to import killmail", err)
		}

		return dto.ConvertKillmailToResponse(killmail), nil
	})

	// Get killmail statistics (public)
	huma.Register(api, huma.Operation{
		OperationID:   "getKillmailStats",
		Method:        http.MethodGet,
		Path:          basePath + "/stats",
		Summary:       "Get killmail statistics",
		Description:   "Returns basic statistics about killmails stored in the database.",
		Tags:          []string{"Killmails"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *struct{}) (*dto.KillmailStatsOutput, error) {
		stats, err := service.GetKillmailStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get killmail statistics", err)
		}

		return &dto.KillmailStatsOutput{
			Body: dto.KillmailStatsResponse{
				TotalKillmails: stats["total_killmails"].(int64),
				Collection:     stats["collection"].(string),
			},
		}, nil
	})

}
