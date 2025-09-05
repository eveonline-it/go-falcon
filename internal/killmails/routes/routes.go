package routes

import (
	"context"
	"net/http"
	"time"

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

	// Character Stats Endpoints

	// Get character killmail stats
	huma.Register(api, huma.Operation{
		OperationID:   "getCharacterKillmailStats",
		Method:        http.MethodGet,
		Path:          basePath + "/character/{character_id}/stats",
		Summary:       "Get character killmail statistics",
		Description:   "Returns killmail statistics for a character, including last ships used in tracked categories.",
		Tags:          []string{"Character Stats"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *dto.GetCharacterStatsInput) (*dto.CharacterStatsOutput, error) {
		stats, err := service.GetCharacterStats(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character stats", err)
		}

		if stats == nil {
			return nil, huma.Error404NotFound("Character stats not found")
		}

		return dto.ConvertCharacterStatsToResponse(stats), nil
	})

	// Get character's last ship by category
	huma.Register(api, huma.Operation{
		OperationID:   "getCharacterLastShipByCategory",
		Method:        http.MethodGet,
		Path:          basePath + "/character/{character_id}/last-ship/{category}",
		Summary:       "Get character's last ship in a category",
		Description:   "Returns the last ship used by a character in a specific category (interdictor, forcerecon, strategic, hic, monitor, blackops, marauders, fax, dread, carrier, super, titan, lancer).",
		Tags:          []string{"Character Stats"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *dto.GetCharacterLastShipByCategoryInput) (*dto.LastShipByCategoryOutput, error) {
		ship, err := service.GetCharacterLastShipByCategory(ctx, input.CharacterID, input.Category)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character's last ship", err)
		}

		if ship == nil {
			return nil, huma.Error404NotFound("No ship found in this category for this character")
		}

		return dto.ConvertLastShipToResponse(ship, input.Category), nil
	})

	// Get characters by ship category
	huma.Register(api, huma.Operation{
		OperationID:   "getCharactersByShipCategory",
		Method:        http.MethodGet,
		Path:          basePath + "/characters/by-category/{category}",
		Summary:       "Get characters by ship category",
		Description:   "Returns characters who have used ships in a specific category.",
		Tags:          []string{"Character Stats"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *dto.GetCharactersByShipCategoryInput) (*dto.CharactersByShipCategoryOutput, error) {
		characters, err := service.GetCharactersByShipCategory(ctx, input.Category, input.Limit)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get characters by category", err)
		}

		return &dto.CharactersByShipCategoryOutput{
			Body: dto.CharactersByShipCategoryResponse{
				Category:   input.Category,
				Characters: dto.ConvertCharacterStatsList(characters),
				Count:      len(characters),
			},
		}, nil
	})

	// Get characters by ship type
	huma.Register(api, huma.Operation{
		OperationID:   "getCharactersByShipType",
		Method:        http.MethodGet,
		Path:          basePath + "/characters/by-ship-type/{ship_type_id}",
		Summary:       "Get characters by ship type",
		Description:   "Returns characters who last used a specific ship type ID.",
		Tags:          []string{"Character Stats"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *dto.GetCharactersByShipTypeInput) (*dto.CharactersByShipTypeOutput, error) {
		characters, err := service.GetCharactersByShipType(ctx, input.ShipTypeID, input.Limit)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get characters by ship type", err)
		}

		return &dto.CharactersByShipTypeOutput{
			Body: dto.CharactersByShipTypeResponse{
				ShipTypeID: input.ShipTypeID,
				Characters: dto.ConvertCharacterStatsList(characters),
				Count:      len(characters),
			},
		}, nil
	})

	// Get recent character activity
	huma.Register(api, huma.Operation{
		OperationID:   "getRecentCharacterActivity",
		Method:        http.MethodGet,
		Path:          basePath + "/characters/recent-activity",
		Summary:       "Get recent character activity",
		Description:   "Returns characters with recent killmail activity in tracked ship categories.",
		Tags:          []string{"Character Stats"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *dto.GetRecentCharacterActivityInput) (*dto.RecentCharacterActivityOutput, error) {
		since := time.Now().Add(-time.Duration(input.Hours) * time.Hour)
		characters, err := service.GetRecentCharacterActivity(ctx, since, input.Limit)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get recent character activity", err)
		}

		return &dto.RecentCharacterActivityOutput{
			Body: dto.RecentCharacterActivityResponse{
				Hours:      input.Hours,
				Characters: dto.ConvertCharacterStatsList(characters),
				Count:      len(characters),
			},
		}, nil
	})

	// Get tracked categories
	huma.Register(api, huma.Operation{
		OperationID:   "getTrackedShipCategories",
		Method:        http.MethodGet,
		Path:          basePath + "/categories",
		Summary:       "Get tracked ship categories",
		Description:   "Returns the list of ship categories that are being tracked for character statistics.",
		Tags:          []string{"Character Stats"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *struct{}) (*dto.TrackedCategoriesOutput, error) {
		categories, err := service.GetTrackedCategories(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get tracked categories", err)
		}

		return &dto.TrackedCategoriesOutput{
			Body: dto.TrackedCategoriesResponse{
				Categories: categories,
				Count:      len(categories),
			},
		}, nil
	})

	// Get category statistics
	huma.Register(api, huma.Operation{
		OperationID:   "getCategoryStats",
		Method:        http.MethodGet,
		Path:          basePath + "/categories/stats",
		Summary:       "Get category statistics",
		Description:   "Returns statistics about how many characters have used ships in each tracked category.",
		Tags:          []string{"Character Stats"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, input *struct{}) (*dto.CategoryStatsOutput, error) {
		stats, err := service.GetCategoryStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get category stats", err)
		}

		var total int64
		for _, count := range stats {
			total += count
		}

		return &dto.CategoryStatsOutput{
			Body: dto.CategoryStatsResponse{
				Stats: stats,
				Total: total,
			},
		}, nil
	})

}
