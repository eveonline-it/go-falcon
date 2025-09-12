package market

import (
	"context"
	"log"

	"go-falcon/internal/market/routes"
	"go-falcon/internal/market/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"
	"go-falcon/pkg/sde"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"time"
)

// Module represents the market module
type Module struct {
	*module.BaseModule
	service      *services.Service
	fetchService *services.FetchService
	repository   *services.Repository
	eveGateway   *evegateway.Client
	sdeService   sde.SDEService
}

// New creates a new market module instance
func New(mongodb *database.MongoDB, redis *database.Redis, eveGateway *evegateway.Client, sdeService sde.SDEService) *Module {
	// Create repository
	repository := services.NewRepository(mongodb)

	// Create main service
	service := services.NewService(repository, eveGateway, sdeService)

	// Create fetch service
	fetchService := services.NewFetchService(repository, eveGateway, sdeService)

	return &Module{
		BaseModule:   module.NewBaseModule("market", mongodb, redis),
		service:      service,
		fetchService: fetchService,
		repository:   repository,
		eveGateway:   eveGateway,
		sdeService:   sdeService,
	}
}

// GetService returns the market service for external access (scheduler integration)
func (m *Module) GetService() *services.Service {
	return m.service
}

// GetFetchService returns the fetch service for external access (scheduler integration)
func (m *Module) GetFetchService() *services.FetchService {
	return m.fetchService
}

// FetchAllRegionalOrders implements the MarketModule interface for scheduler integration
func (m *Module) FetchAllRegionalOrders(ctx context.Context, force bool) error {
	return m.fetchService.FetchAllRegionalOrders(ctx, force)
}

// GetMarketStatus implements the MarketModule interface for scheduler integration
func (m *Module) GetMarketStatus(ctx context.Context) (string, error) {
	status, err := m.service.GetMarketStatus(ctx)
	if err != nil {
		return "", err
	}

	// Return the pagination mode for monitoring
	return status.Body.PaginationInfo.CurrentMode, nil
}

// Initialize sets up the market module, creating necessary database indexes
func (m *Module) Initialize(ctx context.Context) error {
	log.Printf("Initializing market module...")

	// Create database indexes for optimal performance
	if err := m.service.CreateIndexes(ctx); err != nil {
		log.Printf("Failed to create market indexes: %v", err)
		return err
	}

	log.Printf("Market module initialized successfully")
	return nil
}

// Routes is kept for compatibility (no longer used with unified routing)
func (m *Module) Routes(r chi.Router) {
	// Market module uses only Huma v2 unified routes
}

// RegisterHumaRoutes registers the Huma v2 routes (legacy method)
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	// Market module uses unified routes only
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	routes.RegisterMarketRoutes(api, basePath, m.service)
	log.Printf("Market module unified routes registered at %s", basePath)
}

// StartBackgroundTasks starts any background processes for the module
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	// Background tasks are handled by the scheduler system
	// The market fetch system task will call the fetch service periodically
}

// RegisterPermissions registers market-specific permissions
func (m *Module) RegisterPermissions(ctx context.Context, permissionManager *permissions.PermissionManager) error {
	marketPermissions := []permissions.Permission{
		{
			ID:          "market:orders:read",
			Service:     "market",
			Resource:    "orders",
			Action:      "read",
			IsStatic:    false,
			Name:        "Read Market Orders",
			Description: "View market order data for stations, regions, and items",
			Category:    "Market Data",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "market:summary:read",
			Service:     "market",
			Resource:    "summary",
			Action:      "read",
			IsStatic:    false,
			Name:        "Read Market Summaries",
			Description: "View market summary statistics and aggregate data",
			Category:    "Market Data",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "market:status:read",
			Service:     "market",
			Resource:    "status",
			Action:      "read",
			IsStatic:    false,
			Name:        "Read Market Status",
			Description: "View market module status and health information",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "market:fetch:trigger",
			Service:     "market",
			Resource:    "fetch",
			Action:      "trigger",
			IsStatic:    false,
			Name:        "Trigger Market Fetch",
			Description: "Manually trigger market data fetching operations",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "market:administration:manage",
			Service:     "market",
			Resource:    "administration",
			Action:      "manage",
			IsStatic:    false,
			Name:        "Manage Market Administration",
			Description: "Full administrative access to market module operations",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
	}

	return permissionManager.RegisterServicePermissions(ctx, marketPermissions)
}
