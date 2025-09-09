package routes

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/auth/models"
	"go-falcon/internal/structures/dto"
	"go-falcon/internal/structures/services"
	"go-falcon/pkg/middleware"
)

// AuthService interface for auth operations we need
type AuthService interface {
	GetUserProfileByCharacterID(ctx context.Context, characterID int) (*models.UserProfile, error)
}

// getTokenFromUserProfile retrieves the EVE SSO access token from user profile
func getTokenFromUserProfile(ctx context.Context, authService AuthService, characterID int) (string, error) {
	profile, err := authService.GetUserProfileByCharacterID(ctx, characterID)
	if err != nil {
		return "", huma.Error500InternalServerError("Failed to get user profile")
	}

	if profile.AccessToken == "" {
		return "", huma.Error401Unauthorized("No valid EVE SSO token found")
	}

	return profile.AccessToken, nil
}

// RegisterStructuresRoutes registers structure routes on a shared Huma API
func RegisterStructuresRoutes(api huma.API, basePath string, service *services.StructureService, structuresAdapter *middleware.PermissionMiddleware, authService AuthService) {
	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "structures-get-status",
		Method:      http.MethodGet,
		Path:        basePath + "/status",
		Summary:     "Get structures module status",
		Description: "Returns the health status of the structures module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		return &dto.StatusOutput{
			Body: dto.StructureModuleStatusResponse{
				Module:  "structures",
				Status:  "healthy",
				Message: "Structures module is operational",
			},
		}, nil
	})

	// Get structure by ID endpoint
	huma.Register(api, huma.Operation{
		OperationID: "structures-get-structure",
		Method:      http.MethodGet,
		Path:        basePath + "/{structure_id}",
		Summary:     "Get structure information",
		Description: "Retrieves detailed information about a specific structure",
		Tags:        []string{"Structures"},
	}, func(ctx context.Context, input *dto.GetStructureRequest) (*dto.StructureOutput, error) {
		// Authenticate user with JWT middleware
		user, err := structuresAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Get actual EVE SSO access token from user profile
		token, err := getTokenFromUserProfile(ctx, authService, user.CharacterID)
		if err != nil {
			return nil, err
		}

		// Get structure from service (convert int to int32 for characterID)
		structure, err := service.GetStructure(ctx, input.StructureID, int32(user.CharacterID), token)
		if err != nil {
			return nil, err
		}

		return &dto.StructureOutput{
			Body: dto.ToStructureResponse(structure),
		}, nil
	})

	// Get structures by solar system endpoint
	huma.Register(api, huma.Operation{
		OperationID: "structures-get-by-system",
		Method:      http.MethodGet,
		Path:        basePath + "/system/{solar_system_id}",
		Summary:     "Get structures by solar system",
		Description: "Retrieves all known structures in a solar system",
		Tags:        []string{"Structures"},
	}, func(ctx context.Context, input *dto.GetStructuresBySystemRequest) (*dto.StructureListOutput, error) {
		// Authenticate user with JWT middleware
		_, err := structuresAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Get structures from service (this uses database data, not ESI)
		structures, err := service.GetStructuresBySystem(ctx, input.SolarSystemID)
		if err != nil {
			return nil, err
		}

		// Convert to response DTOs
		structureResponses := make([]dto.StructureResponse, len(structures))
		for i, structure := range structures {
			structureResponses[i] = dto.ToStructureResponse(structure)
		}

		return &dto.StructureListOutput{
			Body: dto.StructureListResponse{
				Structures: structureResponses,
				Total:      len(structures),
			},
		}, nil
	})
}
