package routes

import (
	"context"

	"go-falcon/internal/character/dto"
	"go-falcon/internal/character/services"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterCharacterRoutes registers character routes on a shared Huma API
func RegisterCharacterRoutes(api huma.API, basePath string, service *services.Service) {
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
}