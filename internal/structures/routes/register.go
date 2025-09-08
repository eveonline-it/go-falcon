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

	// Get structure by ID endpoint
	huma.Register(api, huma.Operation{
		OperationID: "structures-get-structure",
		Method:      http.MethodGet,
		Path:        basePath + "/{structure_id}",
		Summary:     "Get structure information",
		Description: "Retrieves detailed information about a specific structure",
		Tags:        []string{"Structures"},
	}, func(ctx context.Context, input *dto.GetStructureRequest) (*dto.StructureOutput, error) {
		// TODO: Get authenticated character ID from context/middleware
		// For now, using a placeholder character ID
		characterID := int32(90000001)

		// Get structure from service
		structure, err := service.GetStructure(ctx, input.StructureID, characterID)
		if err != nil {
			return nil, err
		}

		return &dto.StructureOutput{
			Body: dto.ToStructureResponse(structure),
		}, nil
	})

	// Get structures by solar system endpoint
	huma.Register(api, huma.Operation{
		OperationID: "structures-get-by-system",
		Method:      http.MethodGet,
		Path:        basePath + "/system/{solar_system_id}",
		Summary:     "Get structures by solar system",
		Description: "Retrieves all known structures in a solar system",
		Tags:        []string{"Structures"},
	}, func(ctx context.Context, input *dto.GetStructuresBySystemRequest) (*dto.StructureListOutput, error) {
		// Get structures from service
		structures, err := service.GetStructuresBySystem(ctx, input.SolarSystemID)
		if err != nil {
			return nil, err
		}

		// Convert to response DTOs
		structureResponses := make([]dto.StructureResponse, len(structures))
		for i, structure := range structures {
			structureResponses[i] = dto.ToStructureResponse(structure)
		}

		return &dto.StructureListOutput{
			Body: dto.StructureListResponse{
				Structures: structureResponses,
				Total:      len(structures),
			},
		}, nil
	})
}
