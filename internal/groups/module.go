package groups

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	authModels "go-falcon/internal/auth/models"
	"go-falcon/internal/groups/middleware"
	"go-falcon/internal/groups/models"
	"go-falcon/internal/groups/routes"
	"go-falcon/internal/groups/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/handlers"
	"go-falcon/pkg/module"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// AuthModule interface for JWT validation
type AuthModule interface {
	ValidateJWT(token string) (interface{}, error)
}

// Module represents the Groups module
type Module struct {
	*module.BaseModule
	groupService              *services.GroupService
	granularPermissionService *services.GranularPermissionService
	middleware                *middleware.Middleware
	authModule                AuthModule
}

// New creates a new Groups module instance
func New(mongodb *database.MongoDB, redis *database.Redis) *Module {
	// Initialize services
	groupService := services.NewGroupService(mongodb, redis)
	granularPermissionService := services.NewGranularPermissionService(mongodb, redis, groupService)

	// Initialize middleware
	moduleMiddleware := middleware.New(granularPermissionService)

	return &Module{
		BaseModule:                module.NewBaseModule("groups", mongodb, redis, nil),
		groupService:              groupService,
		granularPermissionService: granularPermissionService,
		middleware:                moduleMiddleware,
		authModule:                nil, // Set later via SetAuthModule
	}
}

// SetAuthModule sets the auth module for JWT validation
func (m *Module) SetAuthModule(authModule AuthModule) {
	m.authModule = authModule
}

// authModuleWrapper wraps an AuthModule to implement JWTValidator interface
type authModuleWrapper struct {
	authModule AuthModule
}

// ValidateJWT validates a JWT token using the auth module
func (w *authModuleWrapper) ValidateJWT(token string) (*authModels.AuthenticatedUser, error) {
	result, err := w.authModule.ValidateJWT(token)
	if err != nil {
		return nil, err
	}
	
	// Type assert the result to AuthenticatedUser
	if user, ok := result.(*authModels.AuthenticatedUser); ok {
		return user, nil
	}
	
	return nil, fmt.Errorf("invalid user type returned from auth module")
}

// Initialize performs module initialization
func (m *Module) Initialize(ctx context.Context) error {
	slog.Info("Initializing Groups module")

	// Initialize default groups
	if err := m.groupService.InitializeDefaultGroups(ctx); err != nil {
		slog.Error("Failed to initialize default groups", slog.String("error", err.Error()))
		return err
	}

	// Initialize default services for granular permissions
	if err := m.granularPermissionService.InitializeDefaultServices(ctx); err != nil {
		slog.Error("Failed to initialize default services", slog.String("error", err.Error()))
		return err
	}

	// Initialize database indexes
	if err := m.granularPermissionService.InitializeIndexes(ctx); err != nil {
		slog.Error("Failed to initialize permission indexes", slog.String("error", err.Error()))
		return err
	}

	slog.Info("Groups module initialized successfully")
	return nil
}

// Routes registers the module's routes
func (m *Module) Routes(r chi.Router) {
	// For now, register a simple health endpoint
	r.Get("/groups/health", func(w http.ResponseWriter, r *http.Request) {
		handlers.JSONResponse(w, map[string]interface{}{
			"status":  "healthy",
			"module":  "groups",
			"message": "Groups module is operational",
		}, http.StatusOK)
	})
	
	// TODO: Full route registration will be implemented when auth service
	// integration is properly designed at the application level
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	// Create a JWT validator from the auth module
	if m.authModule == nil {
		// If no auth module available, create a simple health endpoint
		huma.Get(api, basePath+"/health", func(ctx context.Context, input *struct{}) (*struct {
			Body map[string]interface{} `json:"body"`
		}, error) {
			return &struct {
				Body map[string]interface{} `json:"body"`
			}{
				Body: map[string]interface{}{
					"status":  "healthy", 
					"module":  "groups",
					"message": "Groups module is operational (auth service not available)",
				},
			}, nil
		})
		return
	}

	// Create a JWT validator that wraps the auth module
	authService := &authModuleWrapper{authModule: m.authModule}

	// Register full groups routes using the shared API
	routes.RegisterGroupsRoutes(
		api,
		basePath,
		m.groupService,
		m.granularPermissionService,
		m.middleware,
		authService,
	)
}

// Public Interface Methods for other modules

// RequireGranularPermission creates middleware that requires specific granular permissions
func (m *Module) RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler {
	return m.middleware.RequireGranularPermission(service, resource, action)
}

// OptionalGranularPermission creates middleware that adds permission information to context without blocking
func (m *Module) OptionalGranularPermission(service, resource, action string) func(chi.Router) {
	return func(r chi.Router) {
		r.Use(m.middleware.OptionalGranularPermission(service, resource, action))
	}
}

// RequireSuperAdmin creates middleware that requires super admin privileges
func (m *Module) RequireSuperAdmin() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(m.middleware.RequireSuperAdmin())
	}
}

// CheckGranularPermission checks if a user has specific granular permissions from handler context
func (m *Module) CheckGranularPermission(ctx context.Context, characterID int, service, resource, action string) (bool, error) {
	result, err := m.granularPermissionService.CheckPermission(ctx, &models.GranularPermissionCheck{
		CharacterID: characterID,
		Service:     service,
		Resource:    resource,
		Action:      action,
	})
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}

// IsSuperAdmin checks if a user is a super admin
func (m *Module) IsSuperAdmin(ctx context.Context, characterID int) (bool, error) {
	// Check if the character ID matches the super admin configuration
	// This would need to be implemented to check against SUPER_ADMIN_CHARACTER_ID env var
	// For now, return false as a placeholder
	return false, nil
}

// GetUserGroups returns all groups a user is a member of
func (m *Module) GetUserGroups(ctx context.Context, characterID int) ([]models.Group, error) {
	return m.groupService.GetUserGroups(ctx, characterID)
}

// Cleanup and Maintenance

// CleanupExpiredMemberships removes expired group memberships
func (m *Module) CleanupExpiredMemberships(ctx context.Context) (int, error) {
	result, err := m.groupService.CleanupExpiredMemberships(ctx)
	return int(result), err
}

// CleanupExpiredPermissions removes expired permission assignments
func (m *Module) CleanupExpiredPermissions(ctx context.Context) (int64, error) {
	return m.granularPermissionService.CleanupExpiredPermissions(ctx)
}

// Service Accessors (for other modules that need direct access)

// GetGroupService returns the group service
func (m *Module) GetGroupService() *services.GroupService {
	return m.groupService
}

// GetGranularPermissionService returns the granular permission service
func (m *Module) GetGranularPermissionService() *services.GranularPermissionService {
	return m.granularPermissionService
}

// ValidateCorporateMemberships validates all corporate group memberships against ESI data
func (m *Module) ValidateCorporateMemberships(ctx context.Context) error {
	// This would be implemented to check corporation and alliance memberships
	// For now, return nil as a placeholder
	slog.Info("Validating corporate memberships")
	return nil
}

// SyncDiscordRoles synchronizes group memberships with Discord roles
func (m *Module) SyncDiscordRoles(ctx context.Context) error {
	// This would be implemented to sync with Discord service
	// For now, return nil as a placeholder
	slog.Info("Syncing Discord roles")
	return nil
}

// ValidateGroupIntegrity validates the integrity of all group assignments
func (m *Module) ValidateGroupIntegrity(ctx context.Context) error {
	// This would be implemented to check group integrity
	// For now, return nil as a placeholder
	slog.Info("Validating group integrity")
	return nil
}

// Health check method for monitoring
func (m *Module) HealthCheck(ctx context.Context) error {
	// Check database connectivity
	if err := m.MongoDB().HealthCheck(ctx); err != nil {
		return err
	}

	if err := m.Redis().HealthCheck(ctx); err != nil {
		return err
	}

	return nil
}