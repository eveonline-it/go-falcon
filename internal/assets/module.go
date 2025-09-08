package assets

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go-falcon/internal/assets/models"
	"go-falcon/internal/assets/routes"
	"go-falcon/internal/assets/services"
	authModels "go-falcon/internal/auth/models"
	schedulerServices "go-falcon/internal/scheduler/services"
	structureServices "go-falcon/internal/structures/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/middleware"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Module implements the assets module
type Module struct {
	*module.BaseModule
	service          *services.AssetService
	routes           *routes.AssetRoutes
	schedulerService *schedulerServices.SchedulerService
	authMiddleware   *middleware.PermissionMiddleware
	authService      AuthService
}

// AuthService interface for auth operations we need
type AuthService interface {
	GetUserProfileByCharacterID(ctx context.Context, characterID int) (*authModels.UserProfile, error)
}

// NewModule creates a new assets module
func NewModule(
	db *database.MongoDB,
	eveGateway *evegateway.Client,
	sdeService sde.SDEService,
	structureService *structureServices.StructureService,
	authMiddleware *middleware.PermissionMiddleware,
	schedulerService *schedulerServices.SchedulerService,
	authService AuthService,
) *Module {
	// Create service
	service := services.NewAssetService(db.Database, eveGateway, sdeService, structureService)

	// Create module
	m := &Module{
		BaseModule:       module.NewBaseModule("assets", db, nil),
		service:          service,
		routes:           nil,
		schedulerService: schedulerService,
		authMiddleware:   authMiddleware,
		authService:      authService,
	}

	// Register scheduled tasks if scheduler is available
	// Note: This will be called during module initialization with proper context
	if schedulerService != nil {
		// Tasks will be registered during Initialize() method with proper context
		m.schedulerService = schedulerService
	}

	return m
}

// Routes registers traditional Chi routes (required for Module interface)
func (m *Module) Routes(r chi.Router) {
	// Assets module doesn't register traditional Chi routes
	// All routes are registered via RegisterUnifiedRoutes
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	routes.RegisterAssetsRoutes(api, basePath, m.service, m.authMiddleware, m.authService)
}

// GetService returns the asset service
func (m *Module) GetService() *services.AssetService {
	return m.service
}

// Initialize initializes the module
func (m *Module) Initialize(ctx context.Context) error {
	// Create indexes
	if err := m.createIndexes(ctx); err != nil {
		return err
	}

	// Register scheduled tasks if scheduler is available
	if m.schedulerService != nil {
		if err := m.service.RegisterScheduledTasks(ctx, m.schedulerService); err != nil {
			return err
		}
	}

	return nil
}

// createIndexes creates database indexes for optimal performance
func (m *Module) createIndexes(ctx context.Context) error {
	// Assets collection indexes
	assetsIndexes := []mongo.IndexModel{
		// Compound index for character assets
		{
			Keys: bson.D{
				{Key: "character_id", Value: 1},
				{Key: "location_id", Value: 1},
			},
			Options: options.Index().SetName("idx_character_location"),
		},
		// Compound index for corporation assets
		{
			Keys: bson.D{
				{Key: "corporation_id", Value: 1},
				{Key: "location_id", Value: 1},
			},
			Options: options.Index().SetName("idx_corporation_location"),
		},
		// Item ID index for quick lookups
		{
			Keys:    bson.D{{Key: "item_id", Value: 1}},
			Options: options.Index().SetName("idx_item_id").SetUnique(false),
		},
		// Type ID index for filtering by type
		{
			Keys:    bson.D{{Key: "type_id", Value: 1}},
			Options: options.Index().SetName("idx_type_id"),
		},
		// Location flag index for division filtering
		{
			Keys:    bson.D{{Key: "location_flag", Value: 1}},
			Options: options.Index().SetName("idx_location_flag"),
		},
		// Updated at index for stale asset detection
		{
			Keys:    bson.D{{Key: "updated_at", Value: -1}},
			Options: options.Index().SetName("idx_updated_at"),
		},
	}

	// Create assets collection indexes
	if _, err := m.BaseModule.MongoDB().Database.Collection(models.AssetsCollection).Indexes().CreateMany(ctx, assetsIndexes); err != nil {
		return err
	}

	// Asset snapshots collection indexes
	snapshotIndexes := []mongo.IndexModel{
		// Compound index for character snapshots
		{
			Keys: bson.D{
				{Key: "character_id", Value: 1},
				{Key: "snapshot_time", Value: -1},
			},
			Options: options.Index().SetName("idx_character_snapshot"),
		},
		// Compound index for corporation snapshots
		{
			Keys: bson.D{
				{Key: "corporation_id", Value: 1},
				{Key: "snapshot_time", Value: -1},
			},
			Options: options.Index().SetName("idx_corporation_snapshot"),
		},
		// Location ID index
		{
			Keys:    bson.D{{Key: "location_id", Value: 1}},
			Options: options.Index().SetName("idx_location"),
		},
	}

	// Create snapshot collection indexes
	if _, err := m.BaseModule.MongoDB().Database.Collection(models.AssetSnapshotsCollection).Indexes().CreateMany(ctx, snapshotIndexes); err != nil {
		return err
	}

	// Asset tracking collection indexes
	trackingIndexes := []mongo.IndexModel{
		// User ID index
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_user"),
		},
		// Character ID index
		{
			Keys:    bson.D{{Key: "character_id", Value: 1}},
			Options: options.Index().SetName("idx_character"),
		},
		// Corporation ID index
		{
			Keys:    bson.D{{Key: "corporation_id", Value: 1}},
			Options: options.Index().SetName("idx_corporation"),
		},
		// Enabled index for active tracking
		{
			Keys:    bson.D{{Key: "enabled", Value: 1}},
			Options: options.Index().SetName("idx_enabled"),
		},
	}

	// Create tracking collection indexes
	if _, err := m.BaseModule.MongoDB().Database.Collection(models.AssetTrackingCollection).Indexes().CreateMany(ctx, trackingIndexes); err != nil {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the module
func (m *Module) Shutdown(ctx context.Context) error {
	// Any cleanup needed
	return nil
}
