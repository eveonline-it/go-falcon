package routes

import (
	"context"
	"time"

	"go-falcon/internal/auth/models"
	"go-falcon/internal/character/dto"
	"go-falcon/internal/character/services"
	"go-falcon/pkg/middleware"

	"github.com/danielgtaylor/huma/v2"
)

// AuthRepository interface for auth operations
type AuthRepository interface {
	GetUserProfileByCharacterID(ctx context.Context, characterID int) (*models.UserProfile, error)
}

// RegisterCharacterRoutes registers character routes on a shared Huma API
func RegisterCharacterRoutes(api huma.API, basePath string, service *services.Service, characterAdapter *middleware.CharacterAdapter, authRepository AuthRepository) {
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

	// Get character attributes endpoint (authenticated, requires ESI token)
	huma.Register(api, huma.Operation{
		OperationID: "character-get-attributes",
		Method:      "GET",
		Path:        basePath + "/{character_id}/attributes",
		Summary:     "Get character attributes",
		Description: "Get character attributes from EVE ESI. Requires authentication and esi-skills.read_skills.v1 scope for the character.",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.GetCharacterAttributesInput) (*dto.CharacterAttributesOutput, error) {
		// Require authentication
		var user *models.AuthenticatedUser
		if characterAdapter != nil {
			authUser, err := characterAdapter.RequireCharacterAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
			user = authUser
		}

		// Check if the user is requesting their own attributes or if they have permission
		if user == nil || user.CharacterID != input.CharacterID {
			return nil, huma.Error403Forbidden("You can only view your own character attributes")
		}

		// Get the user profile to retrieve the ESI access token
		if authRepository == nil {
			return nil, huma.Error500InternalServerError("Authentication service not available")
		}

		profile, err := authRepository.GetUserProfileByCharacterID(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to retrieve user profile", err)
		}
		if profile == nil {
			return nil, huma.Error404NotFound("User profile not found")
		}

		// Check if access token is expired
		if time.Now().After(profile.TokenExpiry) {
			return nil, huma.Error401Unauthorized("EVE access token expired, please re-authenticate")
		}

		// Use the ESI access token from the profile
		token := profile.AccessToken
		if token == "" {
			return nil, huma.Error401Unauthorized("No EVE access token available")
		}

		result, err := service.GetCharacterAttributes(ctx, input.CharacterID, token)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character attributes", err)
		}
		return result, nil
	})

	// Get character skill queue endpoint (authenticated, requires ESI token)
	huma.Register(api, huma.Operation{
		OperationID: "character-get-skill-queue",
		Method:      "GET",
		Path:        basePath + "/{character_id}/skillqueue",
		Summary:     "Get character skill queue",
		Description: "Get character skill queue from EVE ESI. Requires authentication and esi-skills.read_skillqueue.v1 scope for the character.",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.GetCharacterSkillQueueInput) (*dto.CharacterSkillQueueOutput, error) {
		// Require authentication
		var user *models.AuthenticatedUser
		if characterAdapter != nil {
			authUser, err := characterAdapter.RequireCharacterAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
			user = authUser
		}

		// Check if the user is requesting their own skill queue or if they have permission
		if user == nil || user.CharacterID != input.CharacterID {
			return nil, huma.Error403Forbidden("You can only view your own character skill queue")
		}

		// Get the user profile to retrieve the ESI access token
		if authRepository == nil {
			return nil, huma.Error500InternalServerError("Authentication service not available")
		}

		profile, err := authRepository.GetUserProfileByCharacterID(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to retrieve user profile", err)
		}
		if profile == nil {
			return nil, huma.Error404NotFound("User profile not found")
		}

		// Check if access token is expired
		if time.Now().After(profile.TokenExpiry) {
			return nil, huma.Error401Unauthorized("EVE access token expired, please re-authenticate")
		}

		// Use the ESI access token from the profile
		token := profile.AccessToken
		if token == "" {
			return nil, huma.Error401Unauthorized("No EVE access token available")
		}

		result, err := service.GetCharacterSkillQueue(ctx, input.CharacterID, token)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character skill queue", err)
		}
		return result, nil
	})

	// Get character skills endpoint (authenticated, requires ESI token)
	huma.Register(api, huma.Operation{
		OperationID: "character-get-skills",
		Method:      "GET",
		Path:        basePath + "/{character_id}/skills",
		Summary:     "Get character skills",
		Description: "Get character skills from EVE ESI. Requires authentication and esi-skills.read_skills.v1 scope for the character.",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.GetCharacterSkillsInput) (*dto.CharacterSkillsOutput, error) {
		// Require authentication
		var user *models.AuthenticatedUser
		if characterAdapter != nil {
			authUser, err := characterAdapter.RequireCharacterAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
			user = authUser
		}

		// Check if the user is requesting their own skills or if they have permission
		if user == nil || user.CharacterID != input.CharacterID {
			return nil, huma.Error403Forbidden("You can only view your own character skills")
		}

		// Get the user profile to retrieve the ESI access token
		if authRepository == nil {
			return nil, huma.Error500InternalServerError("Authentication service not available")
		}

		profile, err := authRepository.GetUserProfileByCharacterID(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to retrieve user profile", err)
		}
		if profile == nil {
			return nil, huma.Error404NotFound("User profile not found")
		}

		// Check if access token is expired
		if time.Now().After(profile.TokenExpiry) {
			return nil, huma.Error401Unauthorized("EVE access token expired, please re-authenticate")
		}

		// Use the ESI access token from the profile
		token := profile.AccessToken
		if token == "" {
			return nil, huma.Error401Unauthorized("No EVE access token available")
		}

		result, err := service.GetCharacterSkills(ctx, input.CharacterID, token)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character skills", err)
		}
		return result, nil
	})

	// Get character corporation history endpoint (public, no token required)
	huma.Register(api, huma.Operation{
		OperationID: "character-get-corporation-history",
		Method:      "GET",
		Path:        basePath + "/{character_id}/corporationhistory",
		Summary:     "Get character corporation history",
		Description: "Get character corporation history from database or fetch from EVE ESI if not found. This endpoint does not require authentication as corporation history is public information.",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.GetCharacterCorporationHistoryInput) (*dto.CharacterCorporationHistoryOutput, error) {
		// Require authentication for consistency with other endpoints
		if characterAdapter != nil {
			_, err := characterAdapter.RequireCharacterAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
		}

		result, err := service.GetCharacterCorporationHistory(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character corporation history", err)
		}
		if result == nil {
			return nil, huma.Error404NotFound("Character corporation history not found")
		}
		return result, nil
	})

	// Get character clones endpoint (authenticated, requires ESI token)
	huma.Register(api, huma.Operation{
		OperationID: "character-get-clones",
		Method:      "GET",
		Path:        basePath + "/{character_id}/clones",
		Summary:     "Get character clones",
		Description: "Get character clones from database or fetch from EVE ESI if not found. Requires authentication and esi-clones.read_clones.v1 scope for the character.",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.GetCharacterClonesInput) (*dto.CharacterClonesOutput, error) {
		// Require authentication
		var user *models.AuthenticatedUser
		if characterAdapter != nil {
			authUser, err := characterAdapter.RequireCharacterAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
			user = authUser
		}

		// Check if the user is requesting their own clones or if they have permission
		if user == nil || user.CharacterID != input.CharacterID {
			return nil, huma.Error403Forbidden("You can only view your own character clones")
		}

		// Get the user profile to retrieve the ESI access token
		if authRepository == nil {
			return nil, huma.Error500InternalServerError("Authentication service not available")
		}

		profile, err := authRepository.GetUserProfileByCharacterID(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to retrieve user profile", err)
		}
		if profile == nil {
			return nil, huma.Error404NotFound("User profile not found")
		}

		// Check if access token is expired
		if time.Now().After(profile.TokenExpiry) {
			return nil, huma.Error401Unauthorized("EVE access token expired, please re-authenticate")
		}

		// Use the ESI access token from the profile
		token := profile.AccessToken
		if token == "" {
			return nil, huma.Error401Unauthorized("No EVE access token available")
		}

		result, err := service.GetCharacterClones(ctx, input.CharacterID, token)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character clones", err)
		}
		return result, nil
	})

	// Get character implants endpoint (authenticated, requires ESI token)
	huma.Register(api, huma.Operation{
		OperationID: "character-get-implants",
		Method:      "GET",
		Path:        basePath + "/{character_id}/implants",
		Summary:     "Get character implants",
		Description: "Get character implants from database or fetch from EVE ESI if not found. Requires authentication and esi-clones.read_implants.v1 scope for the character.",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.GetCharacterImplantsInput) (*dto.CharacterImplantsOutput, error) {
		// Require authentication
		var user *models.AuthenticatedUser
		if characterAdapter != nil {
			authUser, err := characterAdapter.RequireCharacterAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
			user = authUser
		}

		// Check if the user is requesting their own implants or if they have permission
		if user == nil || user.CharacterID != input.CharacterID {
			return nil, huma.Error403Forbidden("You can only view your own character implants")
		}

		// Get the user profile to retrieve the ESI access token
		if authRepository == nil {
			return nil, huma.Error500InternalServerError("Authentication service not available")
		}

		profile, err := authRepository.GetUserProfileByCharacterID(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to retrieve user profile", err)
		}
		if profile == nil {
			return nil, huma.Error404NotFound("User profile not found")
		}

		// Check if access token is expired
		if time.Now().After(profile.TokenExpiry) {
			return nil, huma.Error401Unauthorized("EVE access token expired, please re-authenticate")
		}

		// Use the ESI access token from the profile
		token := profile.AccessToken
		if token == "" {
			return nil, huma.Error401Unauthorized("No EVE access token available")
		}

		result, err := service.GetCharacterImplants(ctx, input.CharacterID, token)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character implants", err)
		}
		return result, nil
	})

	// Get character location endpoint (authenticated, requires ESI token)
	huma.Register(api, huma.Operation{
		OperationID: "character-get-location",
		Method:      "GET",
		Path:        basePath + "/{character_id}/location",
		Summary:     "Get character location",
		Description: "Get character's current location including solar system, station or structure. Requires authentication and esi-location.read_location.v1 scope for the character.",
		Tags:        []string{"Character"},
	}, func(ctx context.Context, input *dto.GetCharacterLocationInput) (*dto.CharacterLocationOutput, error) {
		// Require authentication
		var user *models.AuthenticatedUser
		if characterAdapter != nil {
			authUser, err := characterAdapter.RequireCharacterAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
			user = authUser
		}

		// Check if the user is requesting their own location or if they have permission
		if user == nil || user.CharacterID != input.CharacterID {
			return nil, huma.Error403Forbidden("You can only view your own character location")
		}

		// Get the user profile to retrieve the ESI access token
		if authRepository == nil {
			return nil, huma.Error500InternalServerError("Authentication service not available")
		}

		profile, err := authRepository.GetUserProfileByCharacterID(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to retrieve user profile", err)
		}
		if profile == nil {
			return nil, huma.Error404NotFound("User profile not found")
		}

		// Check if access token is expired
		if time.Now().After(profile.TokenExpiry) {
			return nil, huma.Error401Unauthorized("EVE access token expired, please re-authenticate")
		}

		// Use the ESI access token from the profile
		token := profile.AccessToken
		if token == "" {
			return nil, huma.Error401Unauthorized("No EVE access token available")
		}

		result, err := service.GetCharacterLocation(ctx, input.CharacterID, token)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get character location", err)
		}
		return result, nil
	})
}
