package discord

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"go-falcon/internal/discord/routes"
	"go-falcon/internal/discord/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
)

// Module represents the Discord module
type Module struct {
	*module.BaseModule
	service *services.Service
	routes  *routes.Module
}

// GroupsService interface for groups module dependency
type GroupsService interface {
	GetUserGroups(ctx context.Context, userID string) ([]services.GroupInfo, error)
}

// NewModule creates a new Discord module
func NewModule(db *database.MongoDB, redis *database.Redis, groupsService GroupsService) *Module {
	baseModule := module.NewBaseModule("discord", db, redis)

	// Create service with groups service dependency
	service := services.NewService(db, groupsService)

	// Create routes module (middleware will be set later)
	routesModule := routes.NewModule(service, nil)

	return &Module{
		BaseModule: baseModule,
		service:    service,
		routes:     routesModule,
	}
}

// Initialize implements the module.Module interface
func (m *Module) Initialize(ctx context.Context) error {
	slog.InfoContext(ctx, "Initializing Discord module")

	// Initialize service (create indexes)
	if err := m.service.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize Discord service: %w", err)
	}

	slog.InfoContext(ctx, "Discord module initialized successfully")
	return nil
}

// Routes implements module.Module interface - registers Chi routes (legacy)
func (m *Module) Routes(r chi.Router) {
	slog.InfoContext(context.Background(), "Registering Discord Chi routes")
	m.routes.RegisterRoutes(r)
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API) {
	slog.InfoContext(context.Background(), "Registering Discord unified routes")
	m.routes.RegisterUnifiedRoutes(api)
}

// StartBackgroundTasks implements module.Module interface
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.InfoContext(ctx, "Starting Discord background tasks")

	// Start base module background tasks
	go m.BaseModule.StartBackgroundTasks(ctx)

	// Start Discord-specific background tasks
	go m.runTokenRefresh(ctx)
}

// Stop implements module.Module interface
func (m *Module) Stop() {
	slog.InfoContext(context.Background(), "Stopping Discord module")
	m.BaseModule.Stop()
}

// Close implements the module.Module interface
func (m *Module) Close() error {
	slog.InfoContext(context.Background(), "Closing Discord module")
	return nil
}

// GetService returns the Discord service for use by other modules
func (m *Module) GetService() *services.Service {
	return m.service
}

// SetMiddleware updates the Discord module with middleware dependencies
func (m *Module) SetMiddleware(middleware routes.MiddlewareInterface) {
	// Recreate routes with the new middleware
	m.routes = routes.NewModule(m.service, middleware)
	slog.InfoContext(context.Background(), "Discord module updated with middleware dependencies")
}

// RefreshExpiringTokens refreshes Discord tokens that are expiring soon (for scheduler integration)
func (m *Module) RefreshExpiringTokens(ctx context.Context, batchSize int) (int, int, error) {
	return m.service.RefreshExpiringTokens(ctx, batchSize)
}

// PeriodicSync performs periodic role synchronization (for scheduler integration)
func (m *Module) PeriodicSync(ctx context.Context) error {
	return m.service.PeriodicSync(ctx)
}

// runTokenRefresh periodically refreshes expiring Discord tokens
func (m *Module) runTokenRefresh(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute) // Check every 30 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "Discord token refresh routine stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.InfoContext(ctx, "Discord token refresh routine stopped")
			return
		case <-ticker.C:
			success, failed, err := m.service.RefreshExpiringTokens(ctx, 50)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to refresh expiring Discord tokens", "error", err)
			} else if success > 0 || failed > 0 {
				slog.InfoContext(ctx, "Discord token refresh completed",
					"success", success,
					"failed", failed,
				)
			}
		}
	}
}

// Ensure Module implements the module.Module interface
var _ module.Module = (*Module)(nil)
