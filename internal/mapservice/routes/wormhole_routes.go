package routes

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/mapservice/dto"
	"go-falcon/internal/mapservice/models"
	"go-falcon/internal/mapservice/services"
	"go-falcon/pkg/middleware"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterWormholeRoutes registers protected wormhole management endpoints
func RegisterWormholeRoutes(api huma.API, basePath string, service *services.MapService, mapAdapter *middleware.MapAdapter) {
	// Create wormhole
	huma.Register(api, huma.Operation{
		OperationID: "map-create-wormhole",
		Method:      http.MethodPost,
		Path:        basePath + "/wormholes",
		Summary:     "Create wormhole",
		Description: "Create a new wormhole connection",
		Tags:        []string{"Map / Wormholes"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.CreateWormholeInputWithAuth) (*dto.WormholeResponseOutput, error) {
		// Validate authentication and wormhole management access
		user, err := mapAdapter.RequireWormholeManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		// Get user's default group for creating new wormholes
		groupID, err := service.GetUserDefaultGroupID(ctx, user.UserID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user group information", err)
		}

		wormhole, err := service.CreateWormhole(ctx, userID, user.CharacterName, groupID, input.CreateWormholeInput)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create wormhole", err)
		}

		// Get system names for output
		fromSystem, _ := service.GetSDE().GetSolarSystem(int(wormhole.FromSystemID))
		toSystem, _ := service.GetSDE().GetSolarSystem(int(wormhole.ToSystemID))
		fromSystemName := services.GetSystemName(service.GetSDE(), fromSystem)
		toSystemName := services.GetSystemName(service.GetSDE(), toSystem)

		// Get wormhole static info if available
		var staticInfo *models.WormholeStatic
		if wormhole.WormholeType != "" {
			staticInfo, _ = service.GetWormholeStaticInfo(ctx, wormhole.WormholeType)
		}

		result := dto.WormholeToOutput(wormhole, fromSystemName, toSystemName, staticInfo)
		return &dto.WormholeResponseOutput{Body: result}, nil
	})

	// List wormholes
	huma.Register(api, huma.Operation{
		OperationID: "map-list-wormholes",
		Method:      http.MethodGet,
		Path:        basePath + "/wormholes",
		Summary:     "List wormholes",
		Description: "List wormhole connections with filtering",
		Tags:        []string{"Map / Wormholes"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.GetWormholesInputWithAuth) (*dto.WormholeListResponseOutput, error) {
		// Validate authentication and map access
		user, err := mapAdapter.RequireMapAccess(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		// Get user's group IDs for access control
		groupIDs, err := service.GetUserGroupIDs(ctx, user.UserID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user group information", err)
		}

		wormholes, err := service.GetWormholes(ctx, userID, groupIDs, input.GetWormholesInput)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get wormholes", err)
		}

		// Convert to output format with system names
		var results []dto.WormholeOutput
		for _, wh := range wormholes {
			fromSystem, _ := service.GetSDE().GetSolarSystem(int(wh.FromSystemID))
			toSystem, _ := service.GetSDE().GetSolarSystem(int(wh.ToSystemID))
			fromSystemName := services.GetSystemName(service.GetSDE(), fromSystem)
			toSystemName := services.GetSystemName(service.GetSDE(), toSystem)

			// Get wormhole static info if available
			var staticInfo *models.WormholeStatic
			if wh.WormholeType != "" {
				staticInfo, _ = service.GetWormholeStaticInfo(ctx, wh.WormholeType)
			}

			result := dto.WormholeToOutput(&wh, fromSystemName, toSystemName, staticInfo)
			results = append(results, result)
		}

		return &dto.WormholeListResponseOutput{Body: results}, nil
	})

	// Get wormhole by ID
	huma.Register(api, huma.Operation{
		OperationID: "map-get-wormhole",
		Method:      http.MethodGet,
		Path:        basePath + "/wormholes/{wormhole_id}",
		Summary:     "Get wormhole",
		Description: "Get a specific wormhole by ID",
		Tags:        []string{"Map / Wormholes"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.GetWormholeInputWithAuth) (*dto.WormholeResponseOutput, error) {
		// Validate authentication and map access
		user, err := mapAdapter.RequireMapAccess(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert string ID to ObjectID
		wormholeID, err := primitive.ObjectIDFromHex(input.WormholeID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid wormhole ID", err)
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		// Get user's group IDs for access control
		groupIDs, err := service.GetUserGroupIDs(ctx, user.UserID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user group information", err)
		}

		wormhole, err := service.GetWormholeByID(ctx, wormholeID, userID, groupIDs)
		if err != nil {
			return nil, huma.Error404NotFound("Wormhole not found", err)
		}

		// Get system names for output
		fromSystem, _ := service.GetSDE().GetSolarSystem(int(wormhole.FromSystemID))
		toSystem, _ := service.GetSDE().GetSolarSystem(int(wormhole.ToSystemID))
		fromSystemName := services.GetSystemName(service.GetSDE(), fromSystem)
		toSystemName := services.GetSystemName(service.GetSDE(), toSystem)

		// Get wormhole static info if available
		var staticInfo *models.WormholeStatic
		if wormhole.WormholeType != "" {
			staticInfo, _ = service.GetWormholeStaticInfo(ctx, wormhole.WormholeType)
		}

		result := dto.WormholeToOutput(wormhole, fromSystemName, toSystemName, staticInfo)
		return &dto.WormholeResponseOutput{Body: result}, nil
	})

	// Update wormhole
	huma.Register(api, huma.Operation{
		OperationID: "map-update-wormhole",
		Method:      http.MethodPut,
		Path:        basePath + "/wormholes/{wormhole_id}",
		Summary:     "Update wormhole",
		Description: "Update an existing wormhole connection",
		Tags:        []string{"Map / Wormholes"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UpdateWormholeInputWithAuth) (*dto.WormholeResponseOutput, error) {
		// Validate authentication and wormhole management access
		user, err := mapAdapter.RequireWormholeManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert string ID to ObjectID
		wormholeID, err := primitive.ObjectIDFromHex(input.WormholeID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid wormhole ID", err)
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		wormhole, err := service.UpdateWormholeForRoute(ctx, wormholeID, userID, input.UpdateWormholeInput)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to update wormhole", err)
		}

		// Get system names for output
		fromSystem, _ := service.GetSDE().GetSolarSystem(int(wormhole.FromSystemID))
		toSystem, _ := service.GetSDE().GetSolarSystem(int(wormhole.ToSystemID))
		fromSystemName := services.GetSystemName(service.GetSDE(), fromSystem)
		toSystemName := services.GetSystemName(service.GetSDE(), toSystem)

		// Get wormhole static info if available
		var staticInfo *models.WormholeStatic
		if wormhole.WormholeType != "" {
			staticInfo, _ = service.GetWormholeStaticInfo(ctx, wormhole.WormholeType)
		}

		result := dto.WormholeToOutput(wormhole, fromSystemName, toSystemName, staticInfo)
		return &dto.WormholeResponseOutput{Body: result}, nil
	})

	// Delete wormhole
	huma.Register(api, huma.Operation{
		OperationID: "map-delete-wormhole",
		Method:      http.MethodDelete,
		Path:        basePath + "/wormholes/{wormhole_id}",
		Summary:     "Delete wormhole",
		Description: "Delete a wormhole connection",
		Tags:        []string{"Map / Wormholes"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.DeleteWormholeInputWithAuth) (*dto.DeleteWormholeResponseOutput, error) {
		// Validate authentication and wormhole management access
		user, err := mapAdapter.RequireWormholeManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert string ID to ObjectID
		wormholeID, err := primitive.ObjectIDFromHex(input.WormholeID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid wormhole ID", err)
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		err = service.DeleteWormholeForRoute(ctx, wormholeID, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to delete wormhole", err)
		}

		return &dto.DeleteWormholeResponseOutput{
			Body: struct {
				Success bool   `json:"success" doc:"Operation success"`
				Message string `json:"message" doc:"Response message"`
			}{
				Success: true,
				Message: "Wormhole deleted successfully",
			},
		}, nil
	})

	// Batch wormhole operations
	huma.Register(api, huma.Operation{
		OperationID: "map-batch-wormholes",
		Method:      http.MethodPost,
		Path:        basePath + "/wormholes/batch",
		Summary:     "Batch wormhole operations",
		Description: "Create, update, or delete multiple wormholes in one operation",
		Tags:        []string{"Map / Wormholes"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.BatchWormholeInputWithAuth) (*dto.BatchWormholeOutput, error) {
		// Validate authentication and wormhole management access
		user, err := mapAdapter.RequireWormholeManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		// Get user's default group for creating new wormholes
		groupID, err := service.GetUserDefaultGroupID(ctx, user.UserID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user group information", err)
		}

		result, err := service.BatchUpdateWormholes(ctx, userID, user.CharacterName, groupID, input.BatchWormholeInput)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to process batch operations", err)
		}

		return result, nil
	})
}
