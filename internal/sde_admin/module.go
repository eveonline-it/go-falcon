package sde_admin

import (
	"context"
	"log"
	"log/slog"

	"go-falcon/internal/auth"
	"go-falcon/internal/sde_admin/routes"
	"go-falcon/internal/sde_admin/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/middleware"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the SDE admin module
type Module struct {
	*module.BaseModule
	service         *services.Service
	routes          *routes.Routes
	authModule      *auth.Module
	sdeAdminAdapter *middleware.SDEAdminAdapter
}

// New creates a new SDE admin module instance
func New(mongodb *database.MongoDB, redis *database.Redis, authModule *auth.Module, sdeService sde.SDEService) *Module {
	service := services.NewService(mongodb, redis, sdeService)

	return &Module{
		BaseModule: module.NewBaseModule("sde_admin", mongodb, redis),
		service:    service,
		routes:     routes.NewRoutes(service),
		authModule: authModule,
	}
}

// Routes is kept for compatibility - SDE admin now uses Huma v2 routes only
func (m *Module) Routes(r chi.Router) {
	// SDE admin module uses only Huma v2 routes - call RegisterHumaRoutes instead
	m.RegisterHumaRoutes(r)
}

// RegisterHumaRoutes registers the Huma v2 routes (compatibility method)
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	// This module only uses unified routes - no separate Huma routes needed
	slog.Info("SDE admin module uses unified routes only")
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	// Create centralized middleware if auth module is available
	if m.authModule != nil && m.sdeAdminAdapter == nil {
		authService := m.authModule.GetAuthService()
		if authService != nil {
			// Initialize centralized permission middleware
			permissionMiddleware := middleware.NewPermissionMiddleware(
				authService,
				nil, // No specific permission manager needed for SDE admin
				middleware.WithDebugLogging(),
			)

			// Create SDE admin specific adapter
			m.sdeAdminAdapter = middleware.NewSDEAdminAdapter(permissionMiddleware)
		}
	}

	// Register routes
	routes.RegisterSDEAdminRoutes(api, basePath, m.service, m.sdeAdminAdapter)
	log.Printf("SDE admin module unified routes registered at %s", basePath)
}

// StartBackgroundTasks starts any background processes for the module
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting SDE admin background tasks")

	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)

	// SDE admin module doesn't need specific background tasks currently
	// This could be extended in the future for periodic import monitoring
	for {
		select {
		case <-ctx.Done():
			slog.Info("SDE admin background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("SDE admin background tasks stopped")
			return
		default:
			// No specific background tasks for SDE admin module currently
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			}
		}
	}
}
