package routes

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/assets/dto"
	"go-falcon/internal/assets/services"
	"go-falcon/pkg/middleware"
)

// AssetRoutes handles asset-related HTTP routes
type AssetRoutes struct {
	service    *services.AssetService
	middleware *middleware.PermissionMiddleware
}

// NewAssetRoutes creates a new asset routes handler
func NewAssetRoutes(service *services.AssetService, authMiddleware *middleware.PermissionMiddleware) *AssetRoutes {
	return &AssetRoutes{
		service:    service,
		middleware: authMiddleware,
	}
}

// RegisterRoutes registers all asset routes
func (r *AssetRoutes) RegisterRoutes(api huma.API) {
	// Public status endpoint
	huma.Register(api, huma.Operation{
		OperationID: "getAssetsStatus",
		Method:      http.MethodGet,
		Path:        "/assets/status",
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

	// TODO: Add remaining asset endpoints with proper authentication middleware
	// Character assets, corporation assets, tracking endpoints, etc.
}
