package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/assets/dto"
	"go-falcon/internal/assets/services"
	"go-falcon/internal/auth/models"
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

// RegisterAssetsRoutes registers asset routes on a shared Huma API
func RegisterAssetsRoutes(api huma.API, basePath string, service *services.AssetService, assetsAdapter *middleware.PermissionMiddleware, authService AuthService) {
	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "assets-get-status",
		Method:      http.MethodGet,
		Path:        basePath + "/status",
		Summary:     "Get assets module status",
		Description: "Returns the health status of the assets module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		return &dto.StatusOutput{
			Body: dto.AssetModuleStatusResponse{
				Module:  "assets",
				Status:  "healthy",
				Message: "Assets module is operational",
			},
		}, nil
	})

	// Character assets endpoint
	huma.Register(api, huma.Operation{
		OperationID: "assets-get-character-assets",
		Method:      http.MethodGet,
		Path:        basePath + "/character/{character_id}",
		Summary:     "Get character assets",
		Description: "Retrieves assets for a specific character",
		Tags:        []string{"Assets"},
	}, func(ctx context.Context, input *dto.GetCharacterAssetsRequest) (*dto.AssetListOutput, error) {
		// Require authentication
		user, err := assetsAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Get actual EVE SSO access token from user profile
		token, err := getTokenFromUserProfile(ctx, authService, user.CharacterID)
		if err != nil {
			return nil, err
		}

		// Handle optional location filter
		var locationID *int64
		if input.LocationID != 0 {
			locationID = &input.LocationID
		}

		// Get assets from service
		assets, total, err := service.GetCharacterAssets(ctx, input.CharacterID, token, locationID, input.Page, input.PageSize)
		if err != nil {
			return nil, err
		}

		// Convert to response DTOs
		assetResponses := make([]dto.AssetResponse, len(assets))
		totalValue := 0.0
		for i, asset := range assets {
			assetResponses[i] = dto.ToAssetResponse(asset)
			totalValue += asset.TotalValue
		}

		return &dto.AssetListOutput{
			Body: dto.AssetListResponse{
				Assets:      assetResponses,
				Total:       total,
				TotalValue:  totalValue,
				Page:        input.Page,
				PageSize:    input.PageSize,
				LastUpdated: time.Now(),
			},
		}, nil
	})

	// Corporation assets endpoint
	huma.Register(api, huma.Operation{
		OperationID: "assets-get-corporation-assets",
		Method:      http.MethodGet,
		Path:        basePath + "/corporation/{corporation_id}",
		Summary:     "Get corporation assets",
		Description: "Retrieves assets for a specific corporation (requires Director/Accountant roles)",
		Tags:        []string{"Assets"},
	}, func(ctx context.Context, input *dto.GetCorporationAssetsRequest) (*dto.AssetListOutput, error) {
		// Require authentication
		user, err := assetsAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Get actual EVE SSO access token from user profile
		token, err := getTokenFromUserProfile(ctx, authService, user.CharacterID)
		if err != nil {
			return nil, err
		}

		// Handle optional location filter
		var locationID *int64
		if input.LocationID != 0 {
			locationID = &input.LocationID
		}

		// Handle optional division filter
		var division *int
		if input.Division != 0 {
			division = &input.Division
		}

		// Get assets from service
		assets, total, err := service.GetCorporationAssets(ctx, input.CorporationID, int32(user.CharacterID), token, locationID, division, input.Page, input.PageSize)
		if err != nil {
			return nil, err
		}

		// Convert to response DTOs
		assetResponses := make([]dto.AssetResponse, len(assets))
		totalValue := 0.0
		for i, asset := range assets {
			assetResponses[i] = dto.ToAssetResponse(asset)
			totalValue += asset.TotalValue
		}

		return &dto.AssetListOutput{
			Body: dto.AssetListResponse{
				Assets:      assetResponses,
				Total:       total,
				TotalValue:  totalValue,
				Page:        input.Page,
				PageSize:    input.PageSize,
				LastUpdated: time.Now(),
			},
		}, nil
	})

	// Refresh character assets endpoint
	huma.Register(api, huma.Operation{
		OperationID: "assets-refresh-character-assets",
		Method:      http.MethodPost,
		Path:        basePath + "/character/{character_id}/refresh",
		Summary:     "Refresh character assets",
		Description: "Forces a refresh of character assets from ESI API",
		Tags:        []string{"Assets"},
	}, func(ctx context.Context, input *dto.RefreshCharacterAssetsRequest) (*dto.RefreshAssetsOutput, error) {
		// Require authentication
		user, err := assetsAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Get actual EVE SSO access token from user profile
		token, err := getTokenFromUserProfile(ctx, authService, user.CharacterID)
		if err != nil {
			return nil, err
		}

		// Force refresh assets from ESI
		updated, newItems, removedItems, err := service.RefreshCharacterAssets(ctx, input.CharacterID, token)
		if err != nil {
			return nil, err
		}

		// Get total value after refresh
		// TODO: Calculate total value from refreshed assets
		totalValue := 0.0

		return &dto.RefreshAssetsOutput{
			Body: dto.RefreshAssetsResponse{
				CharacterID:  input.CharacterID,
				ItemsUpdated: updated,
				NewItems:     newItems,
				RemovedItems: removedItems,
				TotalValue:   totalValue,
				UpdatedAt:    time.Now(),
			},
		}, nil
	})
}
