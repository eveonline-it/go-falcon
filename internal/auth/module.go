package auth

import (
	"context"
	"log/slog"
	"time"

	"go-falcon/internal/auth/middleware"
	"go-falcon/internal/auth/routes"
	"go-falcon/internal/auth/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the standardized auth module
type Module struct {
	*module.BaseModule
	authService *services.AuthService
	middleware  *middleware.Middleware
	routes      *routes.Routes
}

// New creates a new auth module with standardized structure
func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService, esiClient *evegateway.Client) *Module {
	baseModule := module.NewBaseModule("auth", mongodb, redis, sdeService)
	
	// Create services
	authService := services.NewAuthService(mongodb, esiClient)
	
	// Create middleware with JWT validator
	middlewareLayer := middleware.New(authService)

	return &Module{
		BaseModule:  baseModule,
		authService: authService,
		middleware:  middlewareLayer,
		routes:      nil, // Will be created when needed
	}
}

// Routes implements module.Module interface - registers Huma v2 routes
func (m *Module) Routes(r chi.Router) {
	m.RegisterHumaRoutes(r)
}

// RegisterHumaRoutes registers the Huma v2 routes (legacy method)
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	if m.routes == nil {
		m.routes = routes.NewRoutes(m.authService, m.middleware, r)
	}
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	routes.RegisterAuthRoutes(api, basePath, m.authService, m.middleware)
}

// StartBackgroundTasks starts auth-specific background tasks
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting auth background tasks", "module", m.Name())
	
	// Start base module background tasks
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Start state cleanup routine
	go m.runStateCleanup(ctx)
	
	// Start token refresh routine (if needed)
	go m.runTokenRefresh(ctx)
}

// GetAuthService returns the auth service for other modules
func (m *Module) GetAuthService() *services.AuthService {
	return m.authService
}

// GetMiddleware returns the middleware for other modules
func (m *Module) GetMiddleware() *middleware.Middleware {
	return m.middleware
}

// ValidateJWT validates a JWT token (for integration with other modules)
func (m *Module) ValidateJWT(token string) (interface{}, error) {
	return m.authService.ValidateJWT(token)
}

// RefreshExpiringTokens refreshes tokens that are expiring soon (for scheduler integration)
func (m *Module) RefreshExpiringTokens(ctx context.Context, batchSize int) (int, int, error) {
	return m.authService.RefreshExpiringTokens(ctx, batchSize)
}

// runStateCleanup periodically cleans up expired OAuth states
func (m *Module) runStateCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("State cleanup routine stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("State cleanup routine stopped")
			return
		case <-ticker.C:
			if err := m.authService.CleanupExpiredStates(ctx); err != nil {
				slog.Error("Failed to cleanup expired states", "error", err)
			} else {
				slog.Debug("Cleaned up expired OAuth states")
			}
		}
	}
}

// runTokenRefresh periodically refreshes expiring tokens (optional background task)
func (m *Module) runTokenRefresh(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Token refresh routine stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Token refresh routine stopped")
			return
		case <-ticker.C:
			// This is optional - the scheduler can also handle token refresh
			success, failed, err := m.authService.RefreshExpiringTokens(ctx, 50)
			if err != nil {
				slog.Error("Failed to refresh expiring tokens", "error", err)
			} else if success > 0 || failed > 0 {
				slog.Info("Token refresh completed", 
					"success", success, 
					"failed", failed,
				)
			}
		}
	}
}