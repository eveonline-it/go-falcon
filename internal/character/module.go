package character

import (
	"context"
	"log"
	"time"

	"go-falcon/internal/character/routes"
	"go-falcon/internal/character/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the character module
type Module struct {
	*module.BaseModule
	service       *services.Service
	updateService *services.UpdateService
	eveGateway    *evegateway.Client
}

// New creates a new character module instance
func New(mongodb *database.MongoDB, redis *database.Redis, eveGateway *evegateway.Client) *Module {
	service := services.NewService(mongodb, eveGateway)
	updateService := services.NewUpdateService(mongodb, eveGateway)

	return &Module{
		BaseModule:    module.NewBaseModule("character", mongodb, redis),
		service:       service,
		updateService: updateService,
		eveGateway:    eveGateway,
	}
}

// GetUpdateService returns the update service for scheduler integration
func (m *Module) GetUpdateService() *services.UpdateService {
	return m.updateService
}

// UpdateAllAffiliations implements the CharacterModule interface for scheduler integration
func (m *Module) UpdateAllAffiliations(ctx context.Context) (updated, failed, skipped int, err error) {
	stats, err := m.updateService.UpdateAllAffiliations(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	return stats.UpdatedCharacters, stats.FailedCharacters, stats.SkippedCharacters, nil
}

// Initialize sets up the character module, creating necessary database indexes
func (m *Module) Initialize(ctx context.Context) error {
	log.Printf("Initializing character module...")
	
	// Create database indexes for optimal performance
	if err := m.service.CreateIndexes(ctx); err != nil {
		log.Printf("Failed to create character indexes: %v", err)
		return err
	}
	
	log.Printf("Character module initialized successfully")
	return nil
}

// Routes is kept for compatibility
func (m *Module) Routes(r chi.Router) {
	// Character module uses only Huma v2 routes
}

// RegisterHumaRoutes registers the Huma v2 routes (legacy)
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	// Character module uses unified routes only
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	routes.RegisterCharacterRoutes(api, basePath, m.service)
	log.Printf("Character module unified routes registered at %s", basePath)
}

// StartBackgroundTasks starts any background processes for the module
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	// No background tasks needed for now
}

// RegisterPermissions registers character-specific permissions
func (m *Module) RegisterPermissions(ctx context.Context, permissionManager *permissions.PermissionManager) error {
	characterPermissions := []permissions.Permission{
		{
			ID:          "character:profiles:view",
			Service:     "character",
			Resource:    "profiles",
			Action:      "view",
			IsStatic:    false,
			Name:        "View Character Profiles",
			Description: "View detailed EVE character profiles and information",
			Category:    "Content Management",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "character:search:access",
			Service:     "character",
			Resource:    "search",
			Action:      "access",
			IsStatic:    false,
			Name:        "Search Characters",
			Description: "Search for characters by name and access character listings",
			Category:    "Content Management",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "character:affiliations:manage",
			Service:     "character",
			Resource:    "affiliations",
			Action:      "manage",
			IsStatic:    false,
			Name:        "Manage Character Affiliations",
			Description: "Trigger character affiliation updates and manage character data",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
	}
	
	return permissionManager.RegisterServicePermissions(ctx, characterPermissions)
}