package routes

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/structures/dto"
	"go-falcon/internal/structures/services"
	"go-falcon/pkg/middleware"
)

// StructureRoutes handles structure-related HTTP routes
type StructureRoutes struct {
	service    *services.StructureService
	middleware *middleware.PermissionMiddleware
}

// NewStructureRoutes creates a new structure routes handler
func NewStructureRoutes(service *services.StructureService, authMiddleware *middleware.PermissionMiddleware) *StructureRoutes {
	return &StructureRoutes{
		service:    service,
		middleware: authMiddleware,
	}
}

// RegisterRoutes registers all structure routes
func (r *StructureRoutes) RegisterRoutes(api huma.API) {
	// Public status endpoint
	huma.Register(api, huma.Operation{
		OperationID: "getStructuresStatus",
		Method:      http.MethodGet,
		Path:        "/structures/status",
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

	// TODO: Add remaining structure endpoints with proper authentication middleware
	// Structure info, bulk refresh, system/owner queries, etc.
}
