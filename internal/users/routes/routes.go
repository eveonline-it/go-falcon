package routes

import (
	"context"

	"go-falcon/internal/users/dto"
	"go-falcon/internal/users/services"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Routes handles Huma-based HTTP routing for the Users module
type Routes struct {
	service *services.Service
	api     huma.API
}

// NewRoutes creates a new Huma Users routes handler
func NewRoutes(service *services.Service, router chi.Router) *Routes {
	// Create Huma API with Chi adapter
	config := huma.DefaultConfig("Go Falcon Users Module", "1.0.0")
	config.Info.Description = "User management and character administration"
	
	api := humachi.New(router, config)

	hr := &Routes{
		service: service,
		api:     api,
	}

	// Register all routes
	hr.registerRoutes()

	return hr
}

// RegisterUsersRoutes registers users routes on a shared Huma API
func RegisterUsersRoutes(api huma.API, basePath string, service *services.Service) {
	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "users-get-status",
		Method:      "GET",
		Path:        basePath + "/status",
		Summary:     "Get users module status",
		Description: "Returns the health status of the users module",
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		status := service.GetStatus(ctx)
		return &dto.StatusOutput{Body: *status}, nil
	})

	// Public endpoints
	huma.Register(api, huma.Operation{
		OperationID: "users-get-stats",
		Method:      "GET",
		Path:        basePath + "/stats",
		Summary:     "Get user statistics",
		Description: "Get aggregate statistics about users in the system",
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *dto.UserStatsInput) (*dto.UserStatsOutput, error) {
		stats, err := service.GetUserStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user statistics", err)
		}
		return &dto.UserStatsOutput{Body: *stats}, nil
	})

	// Administrative endpoints (require authentication and permissions)

	huma.Register(api, huma.Operation{
		OperationID: "users-get-user",
		Method:      "GET",
		Path:        basePath + "/mgt/{character_id}",
		Summary:     "Get user details",
		Description: "Get detailed information about a specific user by character ID",
		Tags:        []string{"Users / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UserGetInput) (*dto.UserGetOutput, error) {
		user, err := service.GetUser(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user", err)
		}
		if user == nil {
			return nil, huma.Error404NotFound("User not found")
		}
		
		userResponse := service.UserToResponse(user)
		return &dto.UserGetOutput{Body: *userResponse}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "users-update-user",
		Method:      "PUT",
		Path:        basePath + "/mgt/{character_id}",
		Summary:     "Update user",
		Description: "Update user status and settings",
		Tags:        []string{"Users / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UserUpdateInput) (*dto.UserUpdateOutput, error) {
		user, err := service.UpdateUser(ctx, input.CharacterID, input.Body)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to update user", err)
		}
		if user == nil {
			return nil, huma.Error404NotFound("User not found")
		}
		
		userResponse := service.UserToResponse(user)
		return &dto.UserUpdateOutput{Body: *userResponse}, nil
	})

	// User character management
	huma.Register(api, huma.Operation{
		OperationID: "users-get-user-characters",
		Method:      "GET",
		Path:        basePath + "/{user_id}/characters",
		Summary:     "List user characters",
		Description: "List all characters associated with a user ID",
		Tags:        []string{"Users / Characters"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UserCharactersInput) (*dto.UserCharactersOutput, error) {
		characters, err := service.ListCharacters(ctx, input.UserID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user characters", err)
		}
		
		response := dto.CharacterListResponse{
			UserID:     input.UserID,
			Characters: characters,
			Count:      len(characters),
		}
		
		return &dto.UserCharactersOutput{Body: response}, nil
	})
}

// registerRoutes registers all Users module routes with Huma
func (hr *Routes) registerRoutes() {
	// Status endpoint (public, no auth required)
	huma.Get(hr.api, "/status", hr.getStatus)
	
	// Public endpoints
	huma.Get(hr.api, "/stats", hr.getUserStats)

	// Administrative endpoints (require authentication and permissions)
	huma.Get(hr.api, "/mgt/{character_id}", hr.getUser)
	huma.Put(hr.api, "/mgt/{character_id}", hr.updateUser)

	// User character management
	huma.Get(hr.api, "/{user_id}/characters", hr.getUserCharacters)
}

// Public endpoint handlers

func (hr *Routes) getStatus(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
	status := hr.service.GetStatus(ctx)
	return &dto.StatusOutput{Body: *status}, nil
}

func (hr *Routes) getUserStats(ctx context.Context, input *dto.UserStatsInput) (*dto.UserStatsOutput, error) {
	stats, err := hr.service.GetUserStats(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get user statistics", err)
	}
	return &dto.UserStatsOutput{Body: *stats}, nil
}

// Administrative endpoint handlers

func (hr *Routes) getUser(ctx context.Context, input *dto.UserGetInput) (*dto.UserGetOutput, error) {
	user, err := hr.service.GetUser(ctx, input.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get user", err)
	}
	if user == nil {
		return nil, huma.Error404NotFound("User not found")
	}
	
	userResponse := hr.service.UserToResponse(user)
	return &dto.UserGetOutput{Body: *userResponse}, nil
}

func (hr *Routes) updateUser(ctx context.Context, input *dto.UserUpdateInput) (*dto.UserUpdateOutput, error) {
	user, err := hr.service.UpdateUser(ctx, input.CharacterID, input.Body)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update user", err)
	}
	if user == nil {
		return nil, huma.Error404NotFound("User not found")
	}
	
	userResponse := hr.service.UserToResponse(user)
	return &dto.UserUpdateOutput{Body: *userResponse}, nil
}

// User management handlers

func (hr *Routes) getUserCharacters(ctx context.Context, input *dto.UserCharactersInput) (*dto.UserCharactersOutput, error) {
	characters, err := hr.service.ListCharacters(ctx, input.UserID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get user characters", err)
	}
	
	response := dto.CharacterListResponse{
		UserID:     input.UserID,
		Characters: characters,
		Count:      len(characters),
	}
	
	return &dto.UserCharactersOutput{Body: response}, nil
}