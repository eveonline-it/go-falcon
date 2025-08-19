package scheduler

import (
	"context"
	"log/slog"
	"log"
	"time"

	"go-falcon/internal/scheduler/middleware"
	"go-falcon/internal/scheduler/routes"
	"go-falcon/internal/scheduler/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
	"github.com/danielgtaylor/huma/v2"
)

// Module represents the standardized scheduler module
type Module struct {
	*module.BaseModule
	schedulerService *services.SchedulerService
	middleware       *middleware.Middleware
	routes           *routes.Routes
	
	// Dependencies
	authModule   AuthModule
	groupsModule interface{}
}

// AuthModule interface defines the methods needed from the auth module
type AuthModule interface {
	RefreshExpiringTokens(ctx context.Context, batchSize int) (successCount, failureCount int, err error)
}



// New creates a new scheduler module with standardized structure
func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService, authModule AuthModule, casbinFactory interface{}) *Module {
	baseModule := module.NewBaseModule("scheduler", mongodb, redis, sdeService)
	
	// Create services
	schedulerService := services.NewSchedulerService(mongodb, redis, authModule, nil)
	
	// Create middleware
	middlewareLayer := middleware.New()
	
	return &Module{
		BaseModule:       baseModule,
		schedulerService: schedulerService,
		middleware:       middlewareLayer,
		routes:           nil, // Will be created when needed
		authModule:       authModule,
		groupsModule:     casbinFactory,
	}
}

// Routes registers all scheduler routes (traditional Chi)
func (m *Module) Routes(r chi.Router) {
	// Apply middleware first
	r.Use(m.middleware.RequestLogging)
	r.Use(m.middleware.SecurityHeaders)
	
	// Register health check route using base module
	m.RegisterHealthRoute(r)
	
	// Scheduler module now uses only Huma v2 routes - call RegisterHumaRoutes instead
	m.RegisterHumaRoutes(r)
}

// RegisterHumaRoutes registers the Huma v2 routes
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	if m.routes == nil {
		m.routes = routes.NewRoutes(m.schedulerService, m.middleware, r)
	}
}

// StartBackgroundTasks starts scheduler-specific background tasks
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting scheduler background tasks", "module", m.Name())
	
	// Start base module background tasks
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Initialize hardcoded system tasks
	go m.initializeSystemTasks(ctx)

	// Start the scheduler engine
	go m.startEngine(ctx)

	// Start task cleanup routine
	go m.runTaskCleanup(ctx)

	// Monitor scheduler health
	go m.runHealthMonitoring(ctx)
}

// GetSchedulerService returns the scheduler service for other modules
func (m *Module) GetSchedulerService() *services.SchedulerService {
	return m.schedulerService
}

// GetMiddleware returns the middleware for other modules
func (m *Module) GetMiddleware() *middleware.Middleware {
	return m.middleware
}

// initializeSystemTasks creates hardcoded system tasks if they don't exist
func (m *Module) initializeSystemTasks(ctx context.Context) {
	if err := m.schedulerService.InitializeSystemTasks(ctx); err != nil {
		slog.Error("Failed to initialize system tasks", "error", err)
	}
}

// startEngine starts the scheduler engine
func (m *Module) startEngine(ctx context.Context) {
	// Wait a moment for database connections to be ready
	time.Sleep(2 * time.Second)
	
	if err := m.schedulerService.StartEngine(ctx); err != nil {
		slog.Error("Failed to start scheduler engine", "error", err)
		return
	}
	
	slog.Info("Scheduler engine started successfully")
}

// runTaskCleanup performs periodic cleanup of task execution history
func (m *Module) runTaskCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Task cleanup stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Task cleanup stopped")
			return
		case <-ticker.C:
			// Cleanup logic would be implemented in the service layer
			// For now, just log that cleanup ran
			slog.Debug("Task cleanup cycle completed")
		}
	}
}

// runHealthMonitoring monitors scheduler health and performance
func (m *Module) runHealthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Monitor every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Health monitoring stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Health monitoring stopped")
			return
		case <-ticker.C:
			// Get scheduler stats for health monitoring
			stats, err := m.schedulerService.GetStats(ctx)
			if err != nil {
				slog.Error("Failed to get scheduler stats for health monitoring", "error", err)
				continue
			}

			// Log warning if failure rate is high
			if stats.FailedToday > 0 && stats.CompletedToday > 0 {
				failureRate := float64(stats.FailedToday) / float64(stats.CompletedToday+stats.FailedToday)
				if failureRate > 0.1 { // More than 10% failure rate
					slog.Warn("High task failure rate detected",
						"failure_rate", failureRate,
						"failed_today", stats.FailedToday,
						"completed_today", stats.CompletedToday)
				}
			}

			// Log info about scheduler health
			slog.Debug("Scheduler health check",
				"total_tasks", stats.TotalTasks,
				"enabled_tasks", stats.EnabledTasks,
				"running_tasks", stats.RunningTasks,
				"worker_count", stats.WorkerCount,
				"queue_size", stats.QueueSize)
		}
	}
}

// Stop implements the Module interface - gracefully stops the module
func (m *Module) Stop() {
	slog.Info("Stopping scheduler module", "module", m.Name())
	
	// Call base module stop first
	m.BaseModule.Stop()
	
	// Stop the scheduler engine
	if err := m.schedulerService.StopEngine(); err != nil {
		slog.Error("Error stopping scheduler engine", "error", err)
	}
}

// Shutdown gracefully shuts down the scheduler module (legacy method)
func (m *Module) Shutdown() error {
	// Use the new Stop method for consistency
	m.Stop()
	return nil
}
// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	routes.RegisterSchedulerRoutes(api, basePath, m.schedulerService, m.middleware, m.groupsModule)
	log.Printf("Scheduler module unified routes registered at %s", basePath)
}
