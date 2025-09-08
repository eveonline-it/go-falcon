package routes

import (
	"context"
	"net/http"

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

	// TODO: Add authenticated endpoints here
	// Character assets, corporation assets, tracking endpoints, etc.
	// Use assetsAdapter for authentication when implementing full endpoints
}
