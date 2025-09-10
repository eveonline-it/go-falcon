package users

import (
	"context"
	"log"
	"log/slog"

	"go-falcon/internal/auth"
	"go-falcon/internal/groups/services"
	"go-falcon/internal/users/routes"
	usersServices "go-falcon/internal/users/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/middleware"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"
	"go-falcon/pkg/sde"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the users module
type Module struct {
	*module.BaseModule
	service      *usersServices.Service
	routes       *routes.Routes
	authModule   *auth.Module
	groupService *services.Service
	usersAdapter *middleware.UsersAdapter
}

// New creates a new users module instance
func New(mongodb *database.MongoDB, redis *database.Redis, authModule *auth.Module, eveGateway *evegateway.Client, sdeService sde.SDEService) *Module {
	service := usersServices.NewService(mongodb, redis, eveGateway, sdeService)

	return &Module{
		BaseModule:   module.NewBaseModule("users", mongodb, redis),
		service:      service,
		routes:       nil, // Will be created when needed
		authModule:   authModule,
		groupService: nil, // Will be set after groups module initialization
	}
}

// SetGroupService sets the groups service dependency
func (m *Module) SetGroupService(groupService *services.Service) {
	m.groupService = groupService
	m.service.SetGroupService(groupService)
}

// GetService returns the users service instance
func (m *Module) GetService() *usersServices.Service {
	return m.service
}

// Routes is kept for compatibility - users now uses Huma v2 routes only
func (m *Module) Routes(r chi.Router) {
	// Users module now uses only Huma v2 routes - call RegisterHumaRoutes instead
	m.RegisterHumaRoutes(r)
}

// RegisterHumaRoutes registers the Huma v2 routes
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	if m.routes == nil {
		m.routes = routes.NewRoutes(m.service, r)
	}
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	// Create centralized middleware if auth module is available
	if m.authModule != nil && m.usersAdapter == nil {
		authService := m.authModule.GetAuthService()
		if authService != nil {
			// Get permission manager from groups service if available
			var permissionManager *permissions.PermissionManager
			if m.groupService != nil {
				permissionManager = m.groupService.GetPermissionManager()
			}

			// Initialize centralized permission middleware with debug logging for migration
			permissionMiddleware := middleware.NewPermissionMiddleware(
				authService,
				permissionManager,
				middleware.WithDebugLogging(),
			)

			// Create users-specific adapter
			m.usersAdapter = middleware.NewUsersAdapter(permissionMiddleware)
		}
	}

	routes.RegisterUsersRoutes(api, basePath, m.service, m.usersAdapter)
	log.Printf("Users module unified routes registered at %s", basePath)
}

// StartBackgroundTasks starts any background processes for the module
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting users-specific background tasks")

	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)

	// Users module doesn't need specific background tasks currently
	// This could be extended in the future for user-specific maintenance tasks
	for {
		select {
		case <-ctx.Done():
			slog.Info("Users background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Users background tasks stopped")
			return
		default:
			// No specific background tasks for users module currently
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			}
		}
	}
}
