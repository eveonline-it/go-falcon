package routes

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/structures/dto"
	"go-falcon/internal/structures/services"
	"go-falcon/pkg/middleware"
)

// RegisterStructuresRoutes registers structure routes on a shared Huma API
func RegisterStructuresRoutes(api huma.API, basePath string, service *services.StructureService, structuresAdapter *middleware.PermissionMiddleware) {
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

	// TODO: Add authenticated endpoints here
	// Structure info, bulk refresh, system/owner queries, etc.
	// Use structuresAdapter for authentication when implementing full endpoints
}
