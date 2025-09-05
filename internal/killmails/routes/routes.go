package routes

import (
	"context"
	"net/http"
	"time"

	"go-falcon/internal/killmails/dto"
	"go-falcon/internal/killmails/models"
	"go-falcon/internal/killmails/services"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterKillmailRoutes registers all killmail-related routes
func RegisterKillmailRoutes(api huma.API, basePath string, service *services.Service) {
	// Module status endpoint (public)
	huma.Register(api, huma.Operation{
		OperationID:   "getKillmailsStatus",
		Method:        http.MethodGet,
		Path:          basePath + "/status",
		Summary:       "Get killmails module status",
		Description:   "Returns the health status of the killmails module",
		Tags:          []string{"Module Status"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		if err := service.HealthCheck(ctx); err != nil {
			return &dto.StatusOutput{
				Body: dto.ModuleStatusResponse{
					Module:  "killmails",
					Status:  "unhealthy",
					Message: err.Error(),
				},
			}, nil
		}

		return &dto.StatusOutput{
			Body: dto.ModuleStatusResponse{
				Module: "killmails",
				Status: "healthy",
			},
		}, nil
	})

	// Get specific killmail by ID and hash (public)
	huma.Register(api, huma.Operation{
		OperationID:   "getKillmail",
		Method:        http.MethodGet,
		Path:          basePath + "/{killmail_id}/{hash}",
		Summary:       "Get killmail by ID and hash",
		Description:   "Retrieves a specific killmail using its ID and hash. Uses database-first approach with ESI fallback.",
		Tags:          []string{"Killmails"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *dto.GetKillmailInput) (*dto.KillmailOutput, error) {
		killmail, err := service.GetKillmail(ctx, input.KillmailID, input.Hash)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to fetch killmail", err)
		}

		if killmail == nil {
			return nil, huma.Error404NotFound("Killmail not found")
		}

		return dto.ConvertKillmailToResponse(killmail), nil
	})

	// Get recent killmails from database with filtering (public)
	huma.Register(api, huma.Operation{
		OperationID:   "getRecentKillmails",
		Method:        http.MethodGet,
		Path:          basePath + "/recent",
		Summary:       "Get recent killmails from database",
		Description:   "Retrieves recent killmails from the local database with optional filtering by character, corporation, alliance, or system.",
		Tags:          []string{"Killmails"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *dto.GetRecentKillmailsInput) (*dto.KillmailListOutput, error) {
		// Handle different filtering options
		var killmails []models.Killmail
		var err error

		if input.CharacterID != 0 {
			killmails, err = service.GetRecentKillmailsByCharacter(ctx, input.CharacterID, input.Limit)
		} else if input.CorporationID != 0 {
			killmails, err = service.GetRecentKillmailsByCorporation(ctx, input.CorporationID, input.Limit)
		} else if input.AllianceID != 0 {
			killmails, err = service.GetRecentKillmailsByAlliance(ctx, input.AllianceID, input.Limit)
		} else if input.SystemID != 0 {
			// Parse since time if provided
			var since time.Time
			if input.Since != "" {
				if parsedTime, parseErr := time.Parse(time.RFC3339, input.Since); parseErr == nil {
					since = parsedTime
				} else {
					return nil, huma.Error400BadRequest("Invalid 'since' timestamp format. Use RFC3339 format.", parseErr)
				}
			} else {
				// Default to last 24 hours
				since = time.Now().Add(-24 * time.Hour)
			}
			killmails, err = service.GetKillmailsBySystem(ctx, input.SystemID, since, input.Limit)
		} else {
			return nil, huma.Error400BadRequest("At least one filter parameter is required (character_id, corporation_id, alliance_id, or system_id)")
		}

		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to fetch recent killmails", err)
		}

		return dto.ConvertKillmailsToList(killmails, nil), nil
	})

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

	// Get character recent killmails from ESI (authenticated)
	huma.Register(api, huma.Operation{
		OperationID:   "getCharacterRecentKillmails",
		Method:        http.MethodGet,
		Path:          basePath + "/character/{character_id}/recent",
		Summary:       "Get character's recent killmails from ESI",
		Description:   "Fetches recent killmail references for a character from EVE ESI. Requires authentication.",
		Tags:          []string{"Killmails", "Authenticated"},
		DefaultStatus: http.StatusOK,
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.GetCharacterRecentKillmailsInput) (*dto.KillmailListOutput, error) {
		// In a real implementation, you would extract the token from the authenticated context
		// This is a placeholder for the authentication middleware
		token := getTokenFromContext(ctx)
		if token == "" {
			return nil, huma.Error401Unauthorized("Authentication required")
		}

		refs, err := service.GetCharacterRecentKillmails(ctx, input.CharacterID, token, input.Limit)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to fetch character recent killmails", err)
		}

		return &dto.KillmailListOutput{
			Body: dto.KillmailListResponse{
				Killmails: dto.ConvertKillmailRefsToResponse(refs),
				Count:     len(refs),
			},
		}, nil
	})

	// Get corporation recent killmails from ESI (authenticated)
	huma.Register(api, huma.Operation{
		OperationID:   "getCorporationRecentKillmails",
		Method:        http.MethodGet,
		Path:          basePath + "/corporation/{corporation_id}/recent",
		Summary:       "Get corporation's recent killmails from ESI",
		Description:   "Fetches recent killmail references for a corporation from EVE ESI. Requires authentication.",
		Tags:          []string{"Killmails", "Authenticated"},
		DefaultStatus: http.StatusOK,
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.GetCorporationRecentKillmailsInput) (*dto.KillmailListOutput, error) {
		// In a real implementation, you would extract the token from the authenticated context
		// This is a placeholder for the authentication middleware
		token := getTokenFromContext(ctx)
		if token == "" {
			return nil, huma.Error401Unauthorized("Authentication required")
		}

		refs, err := service.GetCorporationRecentKillmails(ctx, input.CorporationID, token, input.Limit)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to fetch corporation recent killmails", err)
		}

		return &dto.KillmailListOutput{
			Body: dto.KillmailListResponse{
				Killmails: dto.ConvertKillmailRefsToResponse(refs),
				Count:     len(refs),
			},
		}, nil
	})
}

// getTokenFromContext extracts the authentication token from the context
// This is a placeholder function - in the real implementation, this would be
// handled by the centralized authentication middleware
func getTokenFromContext(ctx context.Context) string {
	// TODO: Implement token extraction from authenticated context
	// This would typically be set by the authentication middleware
	if token := ctx.Value("token"); token != nil {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}
	return ""
}
