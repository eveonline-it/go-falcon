package dev

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go-falcon/internal/dev/dto"
	"go-falcon/internal/dev/middleware"
	"go-falcon/internal/dev/routes"
	"go-falcon/internal/dev/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/evegateway/alliance"
	"go-falcon/pkg/evegateway/character"
	"go-falcon/pkg/evegateway/corporation"
	"go-falcon/pkg/evegateway/status"
	"go-falcon/pkg/evegateway/universe"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

// Module represents the Dev module
type Module struct {
	service    *services.Service
	routes     *routes.Routes
	validation *middleware.ValidationMiddleware
}

// NewModule creates a new Dev module instance
func NewModule(
	mongodb *database.MongoDB,
	redis *database.Redis,
	sdeService sde.SDEService,
) (*Module, error) {
	// Create EVE gateway client and sub-clients
	evegateClient := evegateway.NewClient()
	
	// Create shared cache manager for consistency
	cacheManager := evegateway.NewDefaultCacheManager()
	
	// Create retry client
	retryClient := evegateway.NewDefaultRetryClient(evegateClient.HTTPClient(), &evegateway.ESIErrorLimits{}, &sync.RWMutex{})
	
	// Create clients with shared cache
	statusClient := status.NewStatusClient(evegateClient.HTTPClient(), "https://esi.evetech.net", "go-falcon/1.0.0", cacheManager, retryClient)
	characterClient := character.NewCharacterClient(evegateClient.HTTPClient(), "https://esi.evetech.net", "go-falcon/1.0.0", cacheManager, retryClient)
	universeClient := universe.NewUniverseClient(evegateClient.HTTPClient(), "https://esi.evetech.net", "go-falcon/1.0.0", cacheManager, retryClient)
	allianceClient := alliance.NewAllianceClient(evegateClient.HTTPClient(), "https://esi.evetech.net", "go-falcon/1.0.0", cacheManager, retryClient)
	corporationClient := corporation.NewCorporationClient(evegateClient.HTTPClient(), "https://esi.evetech.net", "go-falcon/1.0.0", cacheManager, retryClient)
	
	// Create repository
	repo := services.NewRepository(mongodb, redis)

	// Create service
	service := services.NewService(
		repo,
		evegateClient,
		statusClient,
		characterClient,
		universeClient,
		allianceClient,
		corporationClient,
		sdeService,
		cacheManager,
	)

	// Create validation middleware
	validation := middleware.NewValidationMiddleware()

	return &Module{
		service:    service,
		routes:     nil, // Will be created when needed
		validation: validation,
	}, nil
}

// GetInfo returns module information
func (m *Module) GetInfo() module.Info {
	return module.Info{
		Name:        "dev",
		Version:     "1.0.0",
		Description: "Development testing and debugging utilities for EVE Online ESI integration and SDE functionality",
		Author:      "Go Falcon Team",
		Endpoints: []module.EndpointInfo{
			// Public endpoints
			{
				Path:        "/health",
				Method:      "GET",
				Description: "Module health check",
				Permission:  "",
			},
			{
				Path:        "/status",
				Method:      "GET",
				Description: "Module status information",
				Permission:  "",
			},
			{
				Path:        "/services",
				Method:      "GET",
				Description: "Service discovery endpoint",
				Permission:  "",
			},
			
			// ESI testing endpoints (require dev.tools.read)
			{
				Path:        "/esi/status",
				Method:      "GET",
				Description: "Get EVE Online server status",
				Permission:  "dev.tools.read",
			},
			{
				Path:        "/character/{characterID}",
				Method:      "GET",
				Description: "Get character information",
				Permission:  "dev.tools.read",
			},
			{
				Path:        "/alliance/{allianceID}",
				Method:      "GET",
				Description: "Get alliance information",
				Permission:  "dev.tools.read",
			},
			{
				Path:        "/corporation/{corporationID}",
				Method:      "GET",
				Description: "Get corporation information",
				Permission:  "dev.tools.read",
			},
			{
				Path:        "/universe/system/{systemID}",
				Method:      "GET",
				Description: "Get solar system information",
				Permission:  "dev.tools.read",
			},
		},
		Permissions: []module.PermissionInfo{
			{
				Service:     "dev",
				Resource:    "tools",
				Action:      "read",
				Description: "Access to development tools, ESI testing endpoints, SDE validation, and debugging utilities",
			},
		},
	}
}

// RegisterHumaRoutes registers the Huma v2 routes
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	if m.routes == nil {
		m.routes = routes.NewRoutes(m.service, r)
	}
}

// Initialize initializes the module
func (m *Module) Initialize(ctx context.Context) error {
	// Verify service dependencies
	if m.service == nil {
		return fmt.Errorf("Dev service is not initialized")
	}

	// Test SDE service connectivity
	sdeStatus, err := m.service.GetSDEStatus(ctx)
	if err != nil || sdeStatus == nil {
		return fmt.Errorf("SDE service is not available: %v", err)
	}

	// Test ESI connectivity (optional - don't fail if ESI is down)
	_, err = m.service.GetESIStatus(ctx)
	if err != nil {
		// Log warning but don't fail initialization
		fmt.Printf("Warning: ESI connectivity test failed: %v\n", err)
	}

	return nil
}

// Shutdown gracefully shuts down the module
func (m *Module) Shutdown(ctx context.Context) error {
	// Any cleanup logic would go here
	// For now, we don't have any background processes to stop

	return nil
}

// Health returns the module health status
func (m *Module) Health(ctx context.Context) module.HealthStatus {
	// Test service connectivity
	if _, err := m.service.GetSDEStatus(ctx); err != nil {
		return module.HealthStatus{
			Status:  module.StatusUnhealthy,
			Message: fmt.Sprintf("Dev service unavailable: %v", err),
		}
	}

	return module.HealthStatus{
		Status:  module.StatusHealthy,
		Message: "Dev module is healthy",
	}
}

// GetService returns the Dev service for other modules to use
func (m *Module) GetService() *services.Service {
	return m.service
}

// Name returns the module name for logging and identification
func (m *Module) Name() string {
	return "dev"
}

// Routes sets up the HTTP routes for this module
// Routes is kept for compatibility - dev now uses Huma v2 routes only
func (m *Module) Routes(r chi.Router) {
	// Dev module now uses only Huma v2 routes - call RegisterHumaRoutes instead
	m.RegisterHumaRoutes(r)
}

// StartBackgroundTasks starts any background processing for this module
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	// Dev module doesn't need background tasks currently
	log.Println("Dev module: No background tasks to start")
}

// Stop gracefully stops the module and its background tasks
func (m *Module) Stop() {
	// Dev module doesn't have background tasks to stop currently
	log.Println("Dev module: Stopping")
}

// Testing helper functions for external use

// RunValidationTest runs a validation test
func (m *Module) RunValidationTest(ctx context.Context, req *dto.ValidationTestRequest) (*dto.ValidationTestResponse, error) {
	return m.service.RunValidationTest(ctx, req)
}

// RunPerformanceTest runs a performance test
func (m *Module) RunPerformanceTest(ctx context.Context, req *dto.PerformanceTestRequest) (*dto.PerformanceTestResponse, error) {
	return m.service.RunPerformanceTest(ctx, req)
}

// GenerateMockData generates mock data
func (m *Module) GenerateMockData(ctx context.Context, req *dto.MockDataRequest) (*dto.MockDataResponse, error) {
	return m.service.GenerateMockData(ctx, req)
}

// TestCache tests cache operations
func (m *Module) TestCache(ctx context.Context, req *dto.CacheTestRequest) (*dto.CacheTestResponse, error) {
	return m.service.TestCache(ctx, req)
}

// GetESIStatus gets ESI status for monitoring
func (m *Module) GetESIStatus(ctx context.Context) (*dto.ESIStatusResponse, error) {
	return m.service.GetESIStatus(ctx)
}

// GetSDEStatus gets SDE status for monitoring
func (m *Module) GetSDEStatus(ctx context.Context) (*dto.SDEStatusResponse, error) {
	return m.service.GetSDEStatus(ctx)
}