package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/assets/dto"
	"go-falcon/internal/assets/services"
	"go-falcon/pkg/middleware"
)

// RegisterAssetsRoutes registers asset routes on a shared Huma API
func RegisterAssetsRoutes(api huma.API, basePath string, service *services.AssetService, assetsAdapter *middleware.PermissionMiddleware) {
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
		// Handle optional location filter
		var locationID *int64
		if input.LocationID != 0 {
			locationID = &input.LocationID
		}

		// Get assets from service
		assets, total, err := service.GetCharacterAssets(ctx, input.CharacterID, locationID, input.Page, input.PageSize)
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
		// TODO: Get authenticated character ID from context/middleware
		// For now, using a placeholder character ID
		characterID := int32(90000001)

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
		assets, total, err := service.GetCorporationAssets(ctx, input.CorporationID, characterID, locationID, division, input.Page, input.PageSize)
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
}
