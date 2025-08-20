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
	// Public endpoints
	huma.Get(api, basePath+"/stats", func(ctx context.Context, input *dto.UserStatsInput) (*dto.UserStatsOutput, error) {
		// TODO: Implement once service method is available
		return nil, huma.Error501NotImplemented("User statistics not yet implemented")
	})

	// Administrative endpoints (require authentication and permissions)

	huma.Get(api, basePath+"/users/{character_id}", func(ctx context.Context, input *dto.UserGetInput) (*dto.UserGetOutput, error) {
		// TODO: Implement once service method is available
		return nil, huma.Error501NotImplemented("User retrieval not yet implemented")
	})

	huma.Put(api, basePath+"/users/{character_id}", func(ctx context.Context, input *dto.UserUpdateInput) (*dto.UserUpdateOutput, error) {
		// TODO: Implement once service method is available
		return nil, huma.Error501NotImplemented("User update not yet implemented")
	})

	// User character management
	huma.Get(api, basePath+"/by-user-id/{user_id}/characters", func(ctx context.Context, input *dto.UserCharactersInput) (*dto.UserCharactersOutput, error) {
		// TODO: Implement once service method is available
		return nil, huma.Error501NotImplemented("User characters listing not yet implemented")
	})
}

// registerRoutes registers all Users module routes with Huma
func (hr *Routes) registerRoutes() {
	// Public endpoints
	huma.Get(hr.api, "/stats", hr.getUserStats)

	// Administrative endpoints (require authentication and permissions)
	huma.Get(hr.api, "/users/{character_id}", hr.getUser)
	huma.Put(hr.api, "/users/{character_id}", hr.updateUser)

	// User character management
	huma.Get(hr.api, "/by-user-id/{user_id}/characters", hr.getUserCharacters)
}

// Public endpoint handlers

func (hr *Routes) getUserStats(ctx context.Context, input *dto.UserStatsInput) (*dto.UserStatsOutput, error) {
	// TODO: Implement once service method is available
	return nil, huma.Error501NotImplemented("User statistics not yet implemented")
}

// Administrative endpoint handlers

func (hr *Routes) getUser(ctx context.Context, input *dto.UserGetInput) (*dto.UserGetOutput, error) {
	// TODO: Implement once service method is available
	return nil, huma.Error501NotImplemented("User retrieval not yet implemented")
}

func (hr *Routes) updateUser(ctx context.Context, input *dto.UserUpdateInput) (*dto.UserUpdateOutput, error) {
	// TODO: Implement once service method is available
	return nil, huma.Error501NotImplemented("User update not yet implemented")
}

// User management handlers

func (hr *Routes) getUserCharacters(ctx context.Context, input *dto.UserCharactersInput) (*dto.UserCharactersOutput, error) {
	// TODO: Implement once service method is available
	return nil, huma.Error501NotImplemented("User characters listing not yet implemented")
}