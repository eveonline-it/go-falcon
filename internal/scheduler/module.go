package scheduler

import (
	"context"
	"log"
	"log/slog"
	"time"

	"go-falcon/internal/alliance/dto"
	"go-falcon/internal/auth"
	groupsServices "go-falcon/internal/groups/services"
	"go-falcon/internal/scheduler/routes"
	"go-falcon/internal/scheduler/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/middleware"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the standardized scheduler module
type Module struct {
	*module.BaseModule
	schedulerService *services.SchedulerService
	schedulerAdapter *middleware.SchedulerAdapter
	routes           *routes.Routes

	// Dependencies
	authModule        *auth.Module
	characterModule   CharacterModule
	allianceModule    AllianceModule
	corporationModule CorporationModule
	groupService      *groupsServices.Service
}

// AuthModule interface defines the methods needed from the auth module
type AuthModule interface {
	RefreshExpiringTokens(ctx context.Context, batchSize int) (successCount, failureCount int, err error)
}

// CharacterModule interface defines the methods needed from the character module
type CharacterModule interface {
	UpdateAllAffiliations(ctx context.Context) (updated, failed, skipped int, err error)
}

// AllianceModule interface defines the methods needed from the alliance module
type AllianceModule interface {
	BulkImportAlliances(ctx context.Context) (*dto.BulkImportAlliancesOutput, error)
}

// CorporationModule interface defines the methods needed from the corporation module
type CorporationModule interface {
	UpdateAllCorporations(ctx context.Context, concurrentWorkers int) error
	ValidateCEOTokens(ctx context.Context) error
}

// New creates a new scheduler module with standardized structure
func New(mongodb *database.MongoDB, redis *database.Redis, authModule *auth.Module, characterModule CharacterModule, allianceModule AllianceModule, corporationModule CorporationModule) *Module {
	baseModule := module.NewBaseModule("scheduler", mongodb, redis)

	// Create services (note: groups module will be set later via SetGroupService)
	schedulerService := services.NewSchedulerService(mongodb, redis, authModule, characterModule, allianceModule, corporationModule, nil)

	// Note: SchedulerAdapter will be created in SetGroupService when PermissionManager becomes available
	var schedulerAdapter *middleware.SchedulerAdapter

	return &Module{
		BaseModule:        baseModule,
		schedulerService:  schedulerService,
		schedulerAdapter:  schedulerAdapter,
		routes:            nil, // Will be created when needed
		authModule:        authModule,
		characterModule:   characterModule,
		allianceModule:    allianceModule,
		corporationModule: corporationModule,
		groupService:      nil, // Will be set after groups module initialization
	}
}

// SetGroupService sets the groups service dependency
func (m *Module) SetGroupService(groupService *groupsServices.Service) {
	m.groupService = groupService

	// Recreate scheduler service with groups module dependency
	if groupService != nil {
		m.schedulerService = services.NewSchedulerService(
			m.BaseModule.MongoDB(), m.BaseModule.Redis(),
			m.authModule, m.characterModule, m.allianceModule, m.corporationModule,
			groupService,
		)
		slog.Info("Scheduler service recreated with groups module dependency")
	}

	// Create scheduler adapter with permission manager
	if m.authModule != nil && groupService != nil {
		authService := m.authModule.GetAuthService()
		if authService != nil {
			permissionManager := groupService.GetPermissionManager()
			// Create permission middleware with both auth service and permission manager
			permissionMiddleware := middleware.NewPermissionMiddleware(
				authService,
				permissionManager,
				middleware.WithDebugLogging(), // Enable debug logging for migration
			)
			m.schedulerAdapter = middleware.NewSchedulerAdapter(permissionMiddleware)
		}
	}
}

// Routes registers all scheduler routes (traditional Chi)
func (m *Module) Routes(r chi.Router) {
	// Apply centralized middleware
	r.Use(middleware.TracingMiddleware)

	// Register health check route using base module
	m.RegisterHealthRoute(r)

	// NOTE: RegisterHumaRoutes() call DISABLED for security - it registered endpoints without authentication
	// All secure routes are now registered via RegisterUnifiedRoutes() called from main.go
	// m.RegisterHumaRoutes(r) // DISABLED: This registered unprotected endpoints
}

// RegisterHumaRoutes registers the Huma v2 routes
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	if m.routes == nil {
		m.routes = routes.NewRoutes(m.schedulerService, m.schedulerAdapter, r)
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

// GetSchedulerAdapter returns the scheduler adapter for other modules
func (m *Module) GetSchedulerAdapter() *middleware.SchedulerAdapter {
	return m.schedulerAdapter
}

// RegisterPermissions registers scheduler-specific permissions
func (m *Module) RegisterPermissions(ctx context.Context, permissionManager *permissions.PermissionManager) error {
	schedulerPermissions := []permissions.Permission{
		{
			ID:          "scheduler:tasks:create",
			Service:     "scheduler",
			Resource:    "tasks",
			Action:      "create",
			IsStatic:    false,
			Name:        "Create Scheduled Tasks",
			Description: "Create new scheduled tasks and system automation",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "scheduler:tasks:read",
			Service:     "scheduler",
			Resource:    "tasks",
			Action:      "read",
			IsStatic:    false,
			Name:        "View Scheduled Tasks",
			Description: "View scheduled tasks and their execution history",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "scheduler:tasks:update",
			Service:     "scheduler",
			Resource:    "tasks",
			Action:      "update",
			IsStatic:    false,
			Name:        "Update Scheduled Tasks",
			Description: "Modify existing scheduled tasks and their configuration",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "scheduler:tasks:delete",
			Service:     "scheduler",
			Resource:    "tasks",
			Action:      "delete",
			IsStatic:    false,
			Name:        "Delete Scheduled Tasks",
			Description: "Delete scheduled tasks (system tasks are protected)",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "scheduler:tasks:execute",
			Service:     "scheduler",
			Resource:    "tasks",
			Action:      "execute",
			IsStatic:    false,
			Name:        "Execute Tasks Manually",
			Description: "Manually trigger task execution outside of scheduled times",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "scheduler:tasks:control",
			Service:     "scheduler",
			Resource:    "tasks",
			Action:      "control",
			IsStatic:    false,
			Name:        "Control Task Execution",
			Description: "Start, stop, pause, resume, enable, and disable scheduled tasks",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "scheduler:system:manage",
			Service:     "scheduler",
			Resource:    "system",
			Action:      "manage",
			IsStatic:    false,
			Name:        "Manage Scheduler System",
			Description: "Reload scheduler, view system statistics, and manage scheduler configuration",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
	}

	return permissionManager.RegisterServicePermissions(ctx, schedulerPermissions)
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
	routes.RegisterSchedulerRoutes(api, basePath, m.schedulerService, m.schedulerAdapter)
	log.Printf("Scheduler module unified routes registered at %s", basePath)
}
