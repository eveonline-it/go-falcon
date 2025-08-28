package routes

import (
	"context"
	"fmt"
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
		Path:        fmt.Sprintf("%s/status", basePath),
		Summary:     "Get SDE Admin Module Status",
		Description: "Returns the health and status of the SDE admin module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		return &dto.StatusOutput{
			Body: dto.SDEStatusResponse{
				Module:  "sde_admin",
				Status:  "healthy",
				Message: "SDE admin module is operational for in-memory data management",
			},
		}, nil
	})

	// Get in-memory SDE data status (Super Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "getSDEMemoryStatus",
		Method:      http.MethodGet,
		Path:        fmt.Sprintf("%s/memory", basePath),
		Summary:     "Get SDE Memory Status",
		Description: "Returns detailed status of SDE data currently loaded in memory",
		Tags:        []string{"SDE Admin"},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
	}) (*dto.MemoryStatusOutput, error) {
		// Require super admin access
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.GetMemoryStatus(ctx)
		if err != nil {
			return nil, err
		}
		return &dto.MemoryStatusOutput{Body: *response}, nil
	})

	// Get SDE statistics (Super Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "getSDEStats",
		Method:      http.MethodGet,
		Path:        fmt.Sprintf("%s/stats", basePath),
		Summary:     "Get SDE Statistics",
		Description: "Returns detailed statistics about SDE data loaded in memory",
		Tags:        []string{"SDE Admin"},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
	}) (*dto.SDEStatsOutput, error) {
		// Require super admin access
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.GetStats(ctx)
		if err != nil {
			return nil, err
		}
		return &dto.SDEStatsOutput{Body: *response}, nil
	})

	// Reload SDE data from files (Super Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "reloadSDE",
		Method:      http.MethodPost,
		Path:        fmt.Sprintf("%s/reload", basePath),
		Summary:     "Reload SDE Data",
		Description: "Reload SDE data from files into memory. Can reload all data types or specific ones.",
		Tags:        []string{"SDE Admin"},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
		Body dto.ReloadSDERequest `json:"body"`
	}) (*dto.ReloadSDEOutput, error) {
		// Require super admin access
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.ReloadSDE(ctx, &input.Body)
		if err != nil {
			return nil, err
		}
		return &dto.ReloadSDEOutput{Body: *response}, nil
	})

	// Verify SDE data integrity (Super Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "verifySDEIntegrity",
		Method:      http.MethodGet,
		Path:        fmt.Sprintf("%s/verify", basePath),
		Summary:     "Verify SDE Data Integrity",
		Description: "Verify the integrity and completeness of loaded SDE data",
		Tags:        []string{"SDE Admin"},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
	}) (*dto.VerificationOutput, error) {
		// Require super admin access
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.VerifyIntegrity(ctx)
		if err != nil {
			return nil, err
		}
		return &dto.VerificationOutput{Body: *response}, nil
	})

	// Get system information (Super Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "getSDESystemInfo",
		Method:      http.MethodGet,
		Path:        fmt.Sprintf("%s/system", basePath),
		Summary:     "Get System Information",
		Description: "Get system information relevant to SDE data management including memory usage",
		Tags:        []string{"SDE Admin"},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
	}) (*dto.SystemInfoOutput, error) {
		// Require super admin access
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.GetSystemInfo(ctx)
		if err != nil {
			return nil, err
		}
		return &dto.SystemInfoOutput{Body: *response}, nil
	})

	// Check for SDE updates (Super Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "checkSDEUpdates",
		Method:      http.MethodPost,
		Path:        fmt.Sprintf("%s/check-updates", basePath),
		Summary:     "Check for SDE Updates",
		Description: "Check configured sources for available SDE updates",
		Tags:        []string{"SDE Admin"},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
		Body dto.CheckUpdatesRequest `json:"body"`
	}) (*dto.CheckUpdatesOutput, error) {
		// Require super admin access
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.CheckForUpdates(ctx, &input.Body)
		if err != nil {
			return nil, err
		}
		return &dto.CheckUpdatesOutput{Body: *response}, nil
	})

	// Update SDE data (Super Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "updateSDE",
		Method:      http.MethodPost,
		Path:        fmt.Sprintf("%s/update", basePath),
		Summary:     "Update SDE Data",
		Description: "Download and install SDE updates from configured sources",
		Tags:        []string{"SDE Admin"},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
		Body dto.UpdateSDERequest `json:"body"`
	}) (*dto.UpdateSDEOutput, error) {
		// Require super admin access
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.UpdateSDE(ctx, &input.Body)
		if err != nil {
			return nil, err
		}
		return &dto.UpdateSDEOutput{Body: *response}, nil
	})

	// List available backups (Super Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "listSDEBackups",
		Method:      http.MethodGet,
		Path:        fmt.Sprintf("%s/backups", basePath),
		Summary:     "List SDE Backups",
		Description: "List all available SDE backups",
		Tags:        []string{"SDE Admin"},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
	}) (*dto.ListBackupsOutput, error) {
		// Require super admin access
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.ListBackups(ctx)
		if err != nil {
			return nil, err
		}
		return &dto.ListBackupsOutput{Body: *response}, nil
	})

	// Restore from backup (Super Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "restoreSDEBackup",
		Method:      http.MethodPost,
		Path:        fmt.Sprintf("%s/restore", basePath),
		Summary:     "Restore SDE Backup",
		Description: "Restore SDE data from a backup",
		Tags:        []string{"SDE Admin"},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
		Body dto.RestoreBackupRequest `json:"body"`
	}) (*dto.RestoreBackupOutput, error) {
		// Require super admin access
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.RestoreBackup(ctx, &input.Body)
		if err != nil {
			return nil, err
		}
		return &dto.RestoreBackupOutput{Body: *response}, nil
	})

	// Get configured SDE sources (Super Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "getSDESources",
		Method:      http.MethodGet,
		Path:        fmt.Sprintf("%s/sources", basePath),
		Summary:     "Get SDE Sources",
		Description: "List configured SDE data sources",
		Tags:        []string{"SDE Admin"},
	}, func(ctx context.Context, input *struct {
		dto.AuthInput
	}) (*dto.GetSourcesOutput, error) {
		// Require super admin access
		_, err := middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		response, err := service.GetSources(ctx)
		if err != nil {
			return nil, err
		}
		return &dto.GetSourcesOutput{Body: *response}, nil
	})

	slog.Info("SDE admin routes registered successfully", "endpoints", 12)
}
