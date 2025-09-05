package zkillboard

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	killmailsService "go-falcon/internal/killmails/services"
	websocketServices "go-falcon/internal/websocket/services"
	"go-falcon/internal/zkillboard/routes"
	"go-falcon/internal/zkillboard/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"
)

// Module represents the ZKillboard module
type Module struct {
	*module.BaseModule

	// Services
	consumer         *services.RedisQConsumer
	processor        *services.KillmailProcessor
	repository       *services.Repository
	aggregator       *services.Aggregator
	charStatsService *killmailsService.CharStatsService
	routes           *routes.Routes
}

// NewModule creates a new ZKillboard module instance
func NewModule(
	mongodb *database.MongoDB,
	redis *database.Redis,
	killmailRepo *killmailsService.Repository,
	eveGateway *evegateway.Client,
	websocketService *websocketServices.WebSocketService,
	sdeService sde.SDEService,
) (*Module, error) {
	// Create base module
	baseModule := module.NewBaseModule("zkillboard", mongodb, redis)

	// Create repository
	repository := services.NewRepository(mongodb.Database)

	// Create aggregator
	aggregator := services.NewAggregator(repository, sdeService)

	// Create character stats repository and service
	charStatsRepo := killmailsService.NewCharStatsRepository(mongodb)
	charStatsService := killmailsService.NewCharStatsService(charStatsRepo, sdeService)

	// Create processor
	processor := services.NewKillmailProcessor(
		killmailRepo,
		repository,
		aggregator,
		eveGateway.Killmails,
		websocketService,
		sdeService,
		charStatsService,
	)

	// Create RedisQ consumer
	consumer := services.NewRedisQConsumer(processor, repository)

	// Create routes
	routesHandler := routes.NewRoutes(consumer, repository, aggregator)

	return &Module{
		BaseModule:       baseModule,
		consumer:         consumer,
		processor:        processor,
		repository:       repository,
		aggregator:       aggregator,
		charStatsService: charStatsService,
		routes:           routesHandler,
	}, nil
}

// Initialize initializes the module
func (m *Module) Initialize(ctx context.Context) error {
	slog.Info("Initializing ZKillboard module")

	// Create database indexes
	if err := m.repository.CreateIndexes(ctx); err != nil {
		return err
	}

	// Create character stats indexes
	if err := m.charStatsService.CreateIndexes(ctx); err != nil {
		return err
	}

	slog.Info("ZKillboard module initialized successfully")
	return nil
}

// Routes implements the module.Module interface for chi router
func (m *Module) Routes(r chi.Router) {
	// This is a placeholder for chi.Router compatibility
	// The actual routes are registered via RegisterRoutes with Huma API
}

// RegisterRoutes registers the module's HTTP routes
func (m *Module) RegisterRoutes(api huma.API) error {
	slog.Info("Registering ZKillboard routes")
	m.routes.RegisterRoutes(api)
	return nil
}

// StartBackgroundTasks implements the module.Module interface
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting ZKillboard background tasks")

	// Check if ZKillboard service is enabled
	enabled := strings.ToLower(os.Getenv("ZKB_ENABLED")) == "true"

	if enabled {
		slog.Info("ZKB_ENABLED=true, auto-starting RedisQ consumer")
		if err := m.consumer.Start(ctx); err != nil {
			slog.Error("Failed to auto-start ZKillboard consumer", "error", err)
		} else {
			slog.Info("ZKillboard RedisQ consumer auto-started successfully")
		}
	} else {
		slog.Info("ZKB_ENABLED not set to true, consumer ready for manual start via API")
	}

	slog.Info("ZKillboard background tasks started")
}

// Start starts the module services (legacy method)
func (m *Module) Start(ctx context.Context) error {
	m.StartBackgroundTasks(ctx)
	return nil
}

// Stop implements the module.Module interface
func (m *Module) Stop() {
	slog.Info("Stopping ZKillboard module")

	// Stop consumer if running
	if err := m.consumer.Stop(); err != nil {
		slog.Warn("Failed to stop consumer gracefully", "error", err)
	}

	// Call the base module's Stop method
	m.BaseModule.Stop()

	slog.Info("ZKillboard module stopped")
}

// Health returns the health status of the module
func (m *Module) Health() map[string]interface{} {
	status := m.consumer.GetStatus()

	health := map[string]interface{}{
		"status":          status.Body.Status,
		"last_poll":       status.Body.LastPoll,
		"killmails_found": status.Body.Metrics.KillmailsFound,
		"errors": map[string]interface{}{
			"http":       status.Body.Metrics.HTTPErrors,
			"parse":      status.Body.Metrics.ParseErrors,
			"store":      status.Body.Metrics.StoreErrors,
			"rate_limit": status.Body.Metrics.RateLimitHits,
		},
	}

	// Determine overall health
	isHealthy := true
	if status.Body.Status == "stopped" {
		isHealthy = false // May or may not be an issue depending on configuration
	}
	if status.Body.Metrics.HTTPErrors > 10 || status.Body.Metrics.ParseErrors > 5 {
		isHealthy = false
	}

	health["healthy"] = isHealthy

	return health
}

// GetConsumer returns the RedisQ consumer for external access
func (m *Module) GetConsumer() *services.RedisQConsumer {
	return m.consumer
}

// GetProcessor returns the killmail processor for external access
func (m *Module) GetProcessor() *services.KillmailProcessor {
	return m.processor
}

// GetRepository returns the repository for external access
func (m *Module) GetRepository() *services.Repository {
	return m.repository
}

// GetAggregator returns the aggregator for external access
func (m *Module) GetAggregator() *services.Aggregator {
	return m.aggregator
}
