package routes

import (
	"context"

	"go-falcon/internal/character/dto"
	"go-falcon/internal/character/services"
	"go-falcon/pkg/middleware"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterCharacterRoutes registers character routes on a shared Huma API
func RegisterCharacterRoutes(api huma.API, basePath string, service *services.Service, characterAdapter *middleware.CharacterAdapter) {
	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "character-get-status",
		Method:      "GET",
		Path:        basePath + "/status",
		Summary:     "Get character module status",
		Description: "Returns the health status of the character module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		status := service.GetStatus(ctx)
		return &dto.StatusOutput{Body: *status}, nil
	})

	// Get character profile endpoint (authenticated)
	huma.Register(api, huma.Operation{
		OperationID: "character-get-profile",
		Method:      "GET",
		Path:        basePath + "/{character_id}",
		Summary:     "Get character profile",
		Description: "Get character profile from database or fetch from EVE ESI if not found. Requires authentication.",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.GetCharacterProfileAuthInput) (*dto.CharacterProfileOutput, error) {
		// Require authentication
		if characterAdapter != nil {
			_, err := characterAdapter.RequireCharacterAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
		}

		profile, err := service.GetCharacterProfile(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character profile", err)
		}
		if profile == nil {
			return nil, huma.Error404NotFound("Character not found")
		}
		return profile, nil
	})

	// Search characters by name endpoint (authenticated)
	huma.Register(api, huma.Operation{
		OperationID: "character-search-by-name",
		Method:      "GET",
		Path:        basePath + "/search",
		Summary:     "Search characters by name",
		Description: "Search characters by name with a minimum of 3 characters. Performs case-insensitive search in the database. Requires authentication.",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.SearchCharactersByNameAuthInput) (*dto.SearchCharactersByNameOutput, error) {
		// Require authentication
		if characterAdapter != nil {
			_, err := characterAdapter.RequireCharacterAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
		}

		result, err := service.SearchCharactersByName(ctx, input.Name)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to search characters", err)
		}
		return result, nil
	})
}
