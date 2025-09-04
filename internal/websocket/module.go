package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go-falcon/internal/websocket/middleware"
	"go-falcon/internal/websocket/models"
	"go-falcon/internal/websocket/routes"
	"go-falcon/internal/websocket/services"
	"go-falcon/pkg/config"
	"go-falcon/pkg/database"
	pkgMiddleware "go-falcon/pkg/middleware"
	"go-falcon/pkg/module"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// Module represents the WebSocket module
type Module struct {
	*module.BaseModule
	service        *services.WebSocketService
	routes         *routes.WebSocketRoutes
	authMiddleware *middleware.WebSocketAuthMiddleware
	repository     *services.Repository
}

// NewModule creates a new WebSocket module
func NewModule(db *mongo.Database, redisClient *redis.Client, authService pkgMiddleware.JWTValidator) (*Module, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	if authService == nil {
		return nil, fmt.Errorf("auth service is required")
	}

	// Create base module with database wrappers
	mongodb := &database.MongoDB{Database: db}
	redis := &database.Redis{Client: redisClient}
	baseModule := module.NewBaseModule("websocket", mongodb, redis)

	// Create services
	service := services.NewWebSocketService(db, redisClient)
	repository := services.NewRepository(redisClient)

	// Create authentication middleware using auth service as JWT validator
	authMiddleware := pkgMiddleware.NewAuthMiddleware(authService)

	// Create WebSocket-specific auth middleware
	wsAuthMiddleware := middleware.NewWebSocketAuthMiddleware(authMiddleware)

	// Create routes
	wsRoutes := routes.NewWebSocketRoutes(service, wsAuthMiddleware, repository)

	return &Module{
		BaseModule:     baseModule,
		service:        service,
		routes:         wsRoutes,
		authMiddleware: wsAuthMiddleware,
		repository:     repository,
	}, nil
}

// Initialize starts the WebSocket service
func (m *Module) Initialize(ctx context.Context) error {
	// Start the WebSocket service
	if err := m.service.Start(); err != nil {
		return fmt.Errorf("failed to start WebSocket service: %w", err)
	}

	return nil
}

// StartBackgroundTasks implements module.Module interface
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	// Start base module background tasks
	go m.BaseModule.StartBackgroundTasks(ctx)

	// WebSocket module doesn't need additional background tasks
	// Connection management is handled by the service
}

// Stop implements module.Module interface
func (m *Module) Stop() {
	if m.service != nil {
		m.service.Stop()
	}
	m.BaseModule.Stop()
}

// Routes implements module.Module interface - registers traditional Chi routes
func (m *Module) Routes(r chi.Router) {
	// For now, WebSocket module doesn't need traditional Chi routes
	// All routes are handled via unified Huma API and HTTP handlers
}

// RegisterUnifiedRoutes registers all module routes with the API
func (m *Module) RegisterUnifiedRoutes(api huma.API) {
	m.routes.RegisterRoutes(api)
}

// RegisterHTTPHandler registers HTTP handlers for WebSocket upgrade
func (m *Module) RegisterHTTPHandler(mux interface{}) {
	websocketPath := config.GetWebSocketPath()

	// Handle both *http.ServeMux and *chi.Mux
	switch router := mux.(type) {
	case *http.ServeMux:
		router.HandleFunc(websocketPath, m.routes.HandleWebSocketUpgrade)
	case interface {
		HandleFunc(string, http.HandlerFunc)
	}:
		// This covers *chi.Mux and other routers with HandleFunc method
		router.HandleFunc(websocketPath, m.routes.HandleWebSocketUpgrade)
	default:
		panic("unsupported router type for WebSocket handler registration")
	}
}

// GetService returns the WebSocket service
func (m *Module) GetService() *services.WebSocketService {
	return m.service
}

// GetRepository returns the repository
func (m *Module) GetRepository() *services.Repository {
	return m.repository
}

// Shutdown gracefully shuts down the module
func (m *Module) Shutdown(ctx context.Context) error {
	if m.service != nil {
		return m.service.Stop()
	}
	return nil
}

// IsHealthy checks if the module is healthy
func (m *Module) IsHealthy(ctx context.Context) bool {
	return m.service.IsHealthy(ctx)
}

// GetStats returns module statistics
func (m *Module) GetStats() interface{} {
	return m.service.GetServiceInfo()
}

// BroadcastUserProfileUpdate broadcasts a user profile update
func (m *Module) BroadcastUserProfileUpdate(ctx context.Context, userID string, characterID int64, profileData map[string]interface{}) error {
	return m.service.BroadcastUserProfileUpdate(ctx, userID, characterID, profileData)
}

// BroadcastGroupMembershipChange broadcasts a group membership change
func (m *Module) BroadcastGroupMembershipChange(ctx context.Context, characterID int64, groupID string, groupName string, joined bool) error {
	return m.service.BroadcastGroupMembershipChange(ctx, characterID, groupID, groupName, joined)
}

// SendSystemMessage sends a system message to all users
func (m *Module) SendSystemMessage(ctx context.Context, message string, data map[string]interface{}) error {
	redisHub := m.service.GetRedisHub()

	systemMessage := &models.Message{
		Type:      models.MessageTypeSystemNotification,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": message,
			"data":    data,
		},
	}

	// Send to local connections first
	if err := m.service.SendMessage(systemMessage); err != nil {
		slog.Error("Failed to send system message to local connections", "error", err)
		// Continue to Redis broadcast even if local send fails
	}

	// Publish to Redis for other instances
	return redisHub.PublishSystemMessage(ctx, systemMessage)
}

// Interface compliance check
var _ module.Module = (*Module)(nil)
