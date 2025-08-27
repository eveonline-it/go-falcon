package routes

import (
	"context"
	"fmt"

	"go-falcon/internal/users/dto"
	"go-falcon/internal/users/services"
	"go-falcon/pkg/middleware"

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

	// Note: registerRoutes() call removed for security - it exposed stats endpoint without authentication
	// All secure routes are now registered via RegisterUsersRoutes() called by RegisterUnifiedRoutes()
	// hr.registerRoutes() // REMOVED: This exposed unprotected endpoints

	return hr
}

// RegisterUsersRoutes registers users routes on a shared Huma API
func RegisterUsersRoutes(api huma.API, basePath string, service *services.Service, usersAdapter *middleware.UsersAdapter) {
	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "users-get-status",
		Method:      "GET",
		Path:        basePath + "/status",
		Summary:     "Get users module status",
		Description: "Returns the health status of the users module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		status := service.GetStatus(ctx)
		return &dto.StatusOutput{Body: *status}, nil
	})

	// Administrative endpoints require authentication and permissions
	huma.Register(api, huma.Operation{
		OperationID: "users-get-stats",
		Method:      "GET",
		Path:        basePath + "/stats",
		Summary:     "Get user statistics",
		Description: "Get aggregate statistics about users in the system",
		Tags:        []string{"Users / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UserStatsInput) (*dto.UserStatsOutput, error) {
		// Validate authentication and user management access
		_, err := usersAdapter.RequireUserManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		stats, err := service.GetUserStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user statistics", err)
		}
		return &dto.UserStatsOutput{Body: *stats}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "users-list-users",
		Method:      "GET",
		Path:        basePath,
		Summary:     "List users",
		Description: "List and search users with pagination and filtering",
		Tags:        []string{"Users / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UserListInput) (*dto.UserListOutput, error) {
		// Validate authentication and user management access
		_, err := usersAdapter.RequireUserManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.ListUsers(ctx, *input)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list users", err)
		}
		return &dto.UserListOutput{Body: *response}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "users-get-user",
		Method:      "GET",
		Path:        basePath + "/mgt/{character_id}",
		Summary:     "Get user details",
		Description: "Get detailed information about a specific user by character ID",
		Tags:        []string{"Users / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UserGetInput) (*dto.UserGetOutput, error) {
		// Validate authentication and user management access
		_, err := usersAdapter.RequireUserManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

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
		// Validate authentication and user management access
		_, err := usersAdapter.RequireUserManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

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

	// User character management with enriched profile data
	huma.Register(api, huma.Operation{
		OperationID: "users-get-user-characters",
		Method:      "GET",
		Path:        basePath + "/{user_id}/characters",
		Summary:     "List user characters",
		Description: "List all characters associated with a user ID with enriched profile data including corporation, alliance, security status, and character details",
		Tags:        []string{"Users / Characters"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UserCharactersInput) (*dto.EnrichedUserCharactersOutput, error) {
		// Validate authentication and user access (self or admin)
		_, err := usersAdapter.RequireUserAccess(ctx, input.Authorization, input.Cookie, input.UserID)
		if err != nil {
			return nil, err
		}

		// Use enriched character data with profile information
		enrichedCharacters, err := service.ListEnrichedCharacters(ctx, input.UserID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user characters", err)
		}

		response := dto.EnrichedCharacterListResponse{
			UserID:     input.UserID,
			Characters: enrichedCharacters,
			Count:      len(enrichedCharacters),
		}

		return &dto.EnrichedUserCharactersOutput{Body: response}, nil
	})

	// Character deletion
	huma.Register(api, huma.Operation{
		OperationID: "users-delete-user-character",
		Method:      "DELETE",
		Path:        basePath + "/mgt/{character_id}",
		Summary:     "Delete user character",
		Description: "Delete a user character. Super administrators cannot be deleted.",
		Tags:        []string{"Users / Management"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UserDeleteInput) (*dto.UserDeleteOutput, error) {
		// Validate authentication and user management access
		_, err := usersAdapter.RequireUserManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		err = service.DeleteUser(ctx, input.CharacterID)
		if err != nil {
			// Check for specific error types
			if err.Error() == "cannot delete super admin character" {
				return nil, huma.Error403Forbidden("Cannot delete super administrator character")
			}
			if err.Error() == "user not found" || err.Error() == fmt.Sprintf("user not found for character ID %d", input.CharacterID) {
				return nil, huma.Error404NotFound("User not found")
			}
			return nil, huma.Error500InternalServerError("Failed to delete user", err)
		}

		return &dto.UserDeleteOutput{
			Body: dto.UserDeleteResponse{
				Success: true,
				Message: "User character deleted successfully",
			},
		}, nil
	})

	// Character reordering
	huma.Register(api, huma.Operation{
		OperationID: "users-reorder-user-characters",
		Method:      "PUT",
		Path:        basePath + "/{user_id}/characters/reorder",
		Summary:     "Reorder user characters",
		Description: "Reorder characters for a user by updating their positions",
		Tags:        []string{"Users / Characters"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UserReorderCharactersInput) (*dto.UserReorderCharactersOutput, error) {
		// Validate authentication and user access (self or admin)
		_, err := usersAdapter.RequireUserAccess(ctx, input.Authorization, input.Cookie, input.UserID)
		if err != nil {
			return nil, err
		}

		response, err := service.ReorderUserCharacters(ctx, input.UserID, input.Body)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to reorder characters", err)
		}

		return &dto.UserReorderCharactersOutput{Body: *response}, nil
	})
}

// registerRoutes registers all Users module routes with Huma
func (hr *Routes) registerRoutes() {
	// Status endpoint (public, no auth required)
	huma.Get(hr.api, "/status", hr.getStatus)

	// Public endpoints
	huma.Get(hr.api, "/stats", hr.getUserStats)

	// Administrative endpoints (require authentication and permissions)
	huma.Get(hr.api, "", hr.listUsers)
	huma.Get(hr.api, "/mgt/{character_id}", hr.getUser)
	huma.Put(hr.api, "/mgt/{character_id}", hr.updateUser)
	huma.Delete(hr.api, "/mgt/{character_id}", hr.deleteUser)

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

func (hr *Routes) listUsers(ctx context.Context, input *dto.UserListInput) (*dto.UserListOutput, error) {
	response, err := hr.service.ListUsers(ctx, *input)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list users", err)
	}
	return &dto.UserListOutput{Body: *response}, nil
}

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

func (hr *Routes) deleteUser(ctx context.Context, input *dto.UserDeleteInput) (*dto.UserDeleteOutput, error) {
	err := hr.service.DeleteUser(ctx, input.CharacterID)
	if err != nil {
		// Check for specific error types
		if err.Error() == "cannot delete super admin character" {
			return nil, huma.Error403Forbidden("Cannot delete super administrator character")
		}
		if err.Error() == "user not found" || err.Error() == fmt.Sprintf("user not found for character ID %d", input.CharacterID) {
			return nil, huma.Error404NotFound("User not found")
		}
		return nil, huma.Error500InternalServerError("Failed to delete user", err)
	}

	return &dto.UserDeleteOutput{
		Body: dto.UserDeleteResponse{
			Success: true,
			Message: "User character deleted successfully",
		},
	}, nil
}
