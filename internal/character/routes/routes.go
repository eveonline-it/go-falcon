package routes

import (
	"context"

	"go-falcon/internal/character/dto"
	"go-falcon/internal/character/services"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterCharacterRoutes registers character routes on a shared Huma API
func RegisterCharacterRoutes(api huma.API, basePath string, service *services.Service) {
	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "character-get-status",
		Method:      "GET",
		Path:        basePath + "/status",
		Summary:     "Get character module status",
		Description: "Returns the health status of the character module",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		status := service.GetStatus(ctx)
		return &dto.StatusOutput{Body: *status}, nil
	})

	// Get character profile endpoint
	huma.Register(api, huma.Operation{
		OperationID: "character-get-profile",
		Method:      "GET",
		Path:        basePath + "/{character_id}",
		Summary:     "Get character profile",
		Description: "Get character profile from database or fetch from EVE ESI if not found",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.GetCharacterProfileInput) (*dto.CharacterProfileOutput, error) {
		profile, err := service.GetCharacterProfile(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character profile", err)
		}
		if profile == nil {
			return nil, huma.Error404NotFound("Character not found")
		}
		return profile, nil
	})

	// Search characters by name endpoint
	huma.Register(api, huma.Operation{
		OperationID: "character-search-by-name",
		Method:      "GET",
		Path:        basePath + "/search",
		Summary:     "Search characters by name",
		Description: "Search characters by name with a minimum of 3 characters. Performs case-insensitive search in the database.",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.SearchCharactersByNameInput) (*dto.SearchCharactersByNameOutput, error) {
		result, err := service.SearchCharactersByName(ctx, input.Name)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to search characters", err)
		}
		return result, nil
	})
}