package routes

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"go-falcon/internal/sde_admin/dto"
	"go-falcon/internal/sde_admin/services"
	"go-falcon/pkg/middleware"

	"github.com/danielgtaylor/huma/v2"
)

// Routes handles SDE admin route registration
type Routes struct {
	service *services.Service
}

// NewRoutes creates a new Routes instance
func NewRoutes(service *services.Service) *Routes {
	return &Routes{
		service: service,
	}
}

// RegisterSDEAdminRoutes registers all SDE admin routes on the unified Huma API
func RegisterSDEAdminRoutes(api huma.API, basePath string, service *services.Service, middleware *middleware.SDEAdminAdapter) {
	slog.Info("Registering SDE admin routes", "base_path", basePath)

	// Module status endpoint (public)
	huma.Register(api, huma.Operation{
		OperationID: "getSDEAdminStatus",
		Method:      http.MethodGet,
		Path:        fmt.Sprintf("%s/sde_admin/status", basePath),
		Summary:     "Get SDE Admin Module Status",
		Description: "Returns the health and status of the SDE admin module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		return &dto.StatusOutput{
			Body: dto.SDEStatusResponse{
				Module:  "sde_admin",
				Status:  "healthy",
				Message: "SDE admin module is operational",
			},
		}, nil
	})

	// Import SDE data endpoint (admin only)
	huma.Register(api, huma.Operation{
		OperationID: "importSDEData",
		Method:      http.MethodPost,
		Path:        fmt.Sprintf("%s/sde_admin/import", basePath),
		Summary:     "Import SDE Data to Redis",
		Description: "Start an import operation to load SDE data from files into Redis for fast access",
		Tags:        []string{"SDE Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}}, {"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
		Body dto.ImportSDERequest `json:"body"`
	}) (*dto.ImportSDEOutput, error) {
		// Validate authentication and super admin permissions
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		result, err := service.StartImport(ctx, &input.Body)
		if err != nil {
			slog.Error("Failed to start SDE import", "error", err)
			return nil, huma.Error500InternalServerError("Failed to start SDE import", err)
		}

		return &dto.ImportSDEOutput{Body: *result}, nil
	})

	// Get import status endpoint (admin only)
	huma.Register(api, huma.Operation{
		OperationID: "getSDEImportStatus",
		Method:      http.MethodGet,
		Path:        fmt.Sprintf("%s/sde_admin/import/{import_id}/status", basePath),
		Summary:     "Get SDE Import Status",
		Description: "Get the current status and progress of an SDE import operation",
		Tags:        []string{"SDE Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}}, {"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
		ImportID string `path:"import_id" doc:"The ID of the import operation to check"`
	}) (*dto.ImportStatusOutput, error) {
		// Validate authentication and super admin permissions
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		result, err := service.GetImportStatus(ctx, input.ImportID)
		if err != nil {
			if err.Error() == fmt.Sprintf("import not found: %s", input.ImportID) {
				return nil, huma.Error404NotFound("Import operation not found")
			}
			slog.Error("Failed to get import status", "import_id", input.ImportID, "error", err)
			return nil, huma.Error500InternalServerError("Failed to get import status", err)
		}

		return &dto.ImportStatusOutput{Body: *result}, nil
	})

	// Get SDE statistics endpoint (admin only)
	huma.Register(api, huma.Operation{
		OperationID: "getSDEStats",
		Method:      http.MethodGet,
		Path:        fmt.Sprintf("%s/sde_admin/stats", basePath),
		Summary:     "Get SDE Statistics",
		Description: "Get statistics about SDE data currently stored in Redis",
		Tags:        []string{"SDE Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}}, {"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
	}) (*dto.SDEStatsOutput, error) {
		// Validate authentication and super admin permissions
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		result, err := service.GetSDEStats(ctx)
		if err != nil {
			slog.Error("Failed to get SDE stats", "error", err)
			return nil, huma.Error500InternalServerError("Failed to get SDE statistics", err)
		}

		return &dto.SDEStatsOutput{Body: *result}, nil
	})

	// Clear SDE data endpoint (admin only)
	huma.Register(api, huma.Operation{
		OperationID: "clearSDEData",
		Method:      http.MethodDelete,
		Path:        fmt.Sprintf("%s/sde_admin/clear", basePath),
		Summary:     "Clear SDE Data from Redis",
		Description: "Remove all SDE data from Redis. Use with caution - this cannot be undone.",
		Tags:        []string{"SDE Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}}, {"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
	}) (*dto.ClearSDEOutput, error) {
		// Validate authentication and super admin permissions
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		result, err := service.ClearSDE(ctx)
		if err != nil {
			slog.Error("Failed to clear SDE data", "error", err)
			return nil, huma.Error500InternalServerError("Failed to clear SDE data", err)
		}

		return &dto.ClearSDEOutput{Body: *result}, nil
	})

	log.Printf("SDE admin routes registered at %s", basePath)
}
