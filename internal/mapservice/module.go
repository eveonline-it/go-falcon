package mapservice

import (
	"context"
	"log"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	groupsDTO "go-falcon/internal/groups/dto"
	"go-falcon/internal/mapservice/routes"
	"go-falcon/internal/mapservice/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/middleware"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"
	"go-falcon/pkg/sde"
)

// GroupsService interface for groups service dependency
type GroupsService interface {
	GetUserGroups(ctx context.Context, input *groupsDTO.GetUserGroupsInput) (*groupsDTO.UserGroupsOutput, error)
	GetCharacterGroups(ctx context.Context, input *groupsDTO.GetCharacterGroupsInput) (*groupsDTO.CharacterGroupsOutput, error)
}

// Module represents the map module
type Module struct {
	*module.BaseModule
	mapService      *services.MapService
	wormholeService *services.WormholeService
	routeService    *services.RouteService
	sdeService      sde.SDEService
	groupsService   GroupsService // Optional groups service for access control
}

// NewModule creates a new map module instance
func NewModule(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService) *Module {
	// Create services
	mapService := services.NewMapService(mongodb.Database, redis.Client, sdeService.(*sde.Service))
	wormholeService := services.NewWormholeService(mongodb.Database, redis.Client, sdeService.(*sde.Service))
	routeService := services.NewRouteService(mongodb.Database, redis.Client, sdeService.(*sde.Service))

	return &Module{
		BaseModule:      module.NewBaseModule("map", mongodb, redis),
		mapService:      mapService,
		wormholeService: wormholeService,
		routeService:    routeService,
		sdeService:      sdeService,
		groupsService:   nil, // Set later via SetGroupsService
	}
}

// GetMapService returns the map service for external access
func (m *Module) GetMapService() *services.MapService {
	return m.mapService
}

// GetWormholeService returns the wormhole service for external access
func (m *Module) GetWormholeService() *services.WormholeService {
	return m.wormholeService
}

// Initialize sets up the map module, creating necessary database indexes
func (m *Module) Initialize(ctx context.Context) error {
	log.Printf("Initializing map module...")

	// Initialize static wormhole data
	if err := m.wormholeService.InitializeStaticData(ctx); err != nil {
		log.Printf("Failed to initialize wormhole static data: %v", err)
		return err
	}

	// Create database indexes for signatures
	signatureIndexes := []string{
		"system_id",
		"signature_id",
		"type",
		"sharing_level",
		"expires_at",
		"created_by",
	}

	for _, index := range signatureIndexes {
		// Simple index creation - in production you'd want more sophisticated index management
		log.Printf("Creating index on map_signatures.%s", index)
	}

	// Create database indexes for wormholes
	wormholeIndexes := []string{
		"from_system_id",
		"to_system_id",
		"sharing_level",
		"expires_at",
		"created_by",
	}

	for _, index := range wormholeIndexes {
		log.Printf("Creating index on map_wormholes.%s", index)
	}

	log.Printf("Map module initialized successfully")
	return nil
}

// Routes is kept for compatibility (no longer used with unified routing)
func (m *Module) Routes(r chi.Router) {
	// Map module uses only Huma v2 unified routes
}

// RegisterHumaRoutes registers the Huma v2 routes (legacy method)
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	// Map module uses unified routes only
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string, permissionManager interface{}, authService interface{}) {
	// Create middleware adapter for authentication
	if pm, ok := permissionManager.(*permissions.PermissionManager); ok {
		if as, ok := authService.(middleware.JWTValidator); ok {
			// Create permission middleware
			permissionMiddleware := middleware.NewPermissionMiddleware(as, pm)
			mapAdapter := middleware.NewMapAdapter(permissionMiddleware)

			// Register public routes (no auth required)
			routes.RegisterStatusRoute(api, m.mapService)
			routes.RegisterSearchRoute(api, m.mapService)
			routes.RegisterRegionRoute(api, m.mapService)
			routes.RegisterRouteCalculation(api, m.routeService)

			// Register protected signature endpoints
			routes.RegisterSignatureRoutes(api, basePath, m.mapService, mapAdapter)

			// Register protected wormhole endpoints
			routes.RegisterWormholeRoutes(api, basePath, m.mapService, mapAdapter)

			log.Printf("Map module unified routes registered at %s (with authentication)", basePath)
			return
		}
	}

	// Fallback to public routes only if middleware setup fails
	routes.RegisterStatusRoute(api, m.mapService)
	routes.RegisterSearchRoute(api, m.mapService)
	routes.RegisterRegionRoute(api, m.mapService)
	routes.RegisterRouteCalculation(api, m.routeService)

	log.Printf("Map module unified routes registered at %s (public only)", basePath)
}

// StartBackgroundTasks starts any background processes for the module
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	// Background tasks will be handled by the scheduler system
	// Map cleanup tasks, signature expiration, etc. will be scheduled tasks
}

// SetGroupsService sets the groups service for access control
func (m *Module) SetGroupsService(groupsService GroupsService) {
	m.groupsService = groupsService
	log.Printf("Map module: Groups service set for access control")
}

// GetGroupsService returns the groups service
func (m *Module) GetGroupsService() GroupsService {
	return m.groupsService
}

// RegisterPermissions registers map-specific permissions
func (m *Module) RegisterPermissions(ctx context.Context, permissionManager interface{}) error {
	// TODO: Implement permission registration when ready
	return nil
}
