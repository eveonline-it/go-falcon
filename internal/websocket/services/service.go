package services

import (
	"context"
	"time"

	"go-falcon/internal/websocket/models"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// WebSocketService orchestrates all WebSocket services
type WebSocketService struct {
	connectionMgr  *ConnectionManager
	roomMgr        *RoomManager
	redisHub       *RedisHub
	integrationSvc *IntegrationService
	cleanupTicker  *time.Ticker
	ctx            context.Context
	cancelFunc     context.CancelFunc
}

// NewWebSocketService creates a new WebSocket service
func NewWebSocketService(db *mongo.Database, redisClient *redis.Client) *WebSocketService {
	// Create services
	roomMgr := NewRoomManager()
	redisHub := NewRedisHub(redisClient)
	connectionMgr := NewConnectionManager(roomMgr, redisHub)
	integrationSvc := NewIntegrationService(db, roomMgr, redisHub)

	// Set circular references
	roomMgr.SetConnectionManager(connectionMgr)
	redisHub.SetConnectionManager(connectionMgr)
	redisHub.SetRoomManager(roomMgr)

	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketService{
		connectionMgr:  connectionMgr,
		roomMgr:        roomMgr,
		redisHub:       redisHub,
		integrationSvc: integrationSvc,
		ctx:            ctx,
		cancelFunc:     cancel,
	}
}

// Start initializes the WebSocket service
func (ws *WebSocketService) Start() error {
	slog.Info("Starting WebSocket service")

	// Start Redis hub
	if err := ws.redisHub.Start(ws.ctx); err != nil {
		return err
	}

	// Start cleanup routine
	ws.cleanupTicker = time.NewTicker(5 * time.Minute)
	go ws.cleanupRoutine()

	slog.Info("WebSocket service started successfully")
	return nil
}

// Stop shuts down the WebSocket service
func (ws *WebSocketService) Stop() error {
	slog.Info("Stopping WebSocket service")

	// Cancel context
	ws.cancelFunc()

	// Stop cleanup ticker
	if ws.cleanupTicker != nil {
		ws.cleanupTicker.Stop()
	}

	// Stop Redis hub
	if err := ws.redisHub.Stop(); err != nil {
		slog.Error("Failed to stop Redis hub", "error", err)
	}

	// Clean up all connections
	connections := ws.connectionMgr.GetAllConnections()
	for _, conn := range connections {
		ws.connectionMgr.RemoveConnection(conn.ID)
	}

	slog.Info("WebSocket service stopped")
	return nil
}

// cleanupRoutine performs periodic cleanup
func (ws *WebSocketService) cleanupRoutine() {
	for {
		select {
		case <-ws.ctx.Done():
			return
		case <-ws.cleanupTicker.C:
			ws.connectionMgr.CleanupInactiveConnections()
		}
	}
}

// GetConnectionManager returns the connection manager
func (ws *WebSocketService) GetConnectionManager() *ConnectionManager {
	return ws.connectionMgr
}

// GetRoomManager returns the room manager
func (ws *WebSocketService) GetRoomManager() *RoomManager {
	return ws.roomMgr
}

// GetRedisHub returns the Redis hub
func (ws *WebSocketService) GetRedisHub() *RedisHub {
	return ws.redisHub
}

// GetIntegrationService returns the integration service
func (ws *WebSocketService) GetIntegrationService() *IntegrationService {
	return ws.integrationSvc
}

// CreateConnection creates a new WebSocket connection with automatic room assignment
func (ws *WebSocketService) CreateConnection(conn *models.Connection) error {
	// Add connection to manager first (fast operation)
	actualConn, err := ws.connectionMgr.AddConnection(conn.Conn, conn.UserID, conn.CharacterID, conn.CharacterName)
	if err != nil {
		return err
	}

	// Update the connection reference to use the one from the manager
	*conn = *actualConn

	// Assign to appropriate rooms asynchronously to prevent blocking the WebSocket upgrade
	go func() {
		// Use a timeout context to prevent indefinite blocking
		ctx, cancel := context.WithTimeout(ws.ctx, 30*time.Second)
		defer cancel()

		if err := ws.integrationSvc.AssignUserToRooms(ctx, conn); err != nil {
			slog.Error("Failed to assign user to rooms", "error", err, "connection_id", conn.ID, "user_id", conn.UserID)
			// Room assignment failed but connection is still valid - user just won't get group messages initially
		} else {
			slog.Info("Successfully assigned user to rooms", "connection_id", conn.ID, "user_id", conn.UserID)
		}
	}()

	return nil
}

// SendMessage sends a message through the WebSocket system
func (ws *WebSocketService) SendMessage(message *models.Message) error {
	if message.Room != "" {
		// Send to room
		return ws.roomMgr.BroadcastToRoom(message.Room, message)
	} else if message.To != "" {
		// Send to specific connection
		return ws.connectionMgr.SendToConnection(message.To, message)
	} else {
		// Broadcast to all
		ws.connectionMgr.BroadcastToAll(message)
		return nil
	}
}

// PublishMessage publishes a message via Redis for multi-instance support
func (ws *WebSocketService) PublishMessage(ctx context.Context, channel string, message *models.Message) error {
	return ws.redisHub.PublishMessage(ctx, channel, message)
}

// BroadcastUserProfileUpdate broadcasts a user profile update
func (ws *WebSocketService) BroadcastUserProfileUpdate(ctx context.Context, userID string, characterID int64, profileData map[string]interface{}) error {
	// Handle locally
	ws.integrationSvc.HandleUserProfileUpdate(ctx, userID, characterID, profileData)

	// Broadcast to other instances
	return ws.redisHub.BroadcastUserProfileUpdate(ctx, userID, characterID, profileData)
}

// BroadcastGroupMembershipChange broadcasts a group membership change
func (ws *WebSocketService) BroadcastGroupMembershipChange(ctx context.Context, characterID int64, groupID string, groupName string, joined bool) error {
	// Handle locally
	if err := ws.integrationSvc.HandleGroupMembershipChange(ctx, characterID, groupID, groupName, joined); err != nil {
		return err
	}

	// Get user ID for broadcasting
	userID, err := ws.integrationSvc.getUserIDFromCharacter(ctx, characterID)
	if err != nil {
		return err
	}

	// Broadcast to other instances
	return ws.redisHub.BroadcastGroupMembershipChange(ctx, userID, groupID, groupName, joined)
}

// GetStats returns comprehensive WebSocket statistics
func (ws *WebSocketService) GetStats() models.WebSocketStats {
	return ws.connectionMgr.GetStats()
}

// IsHealthy checks if the WebSocket service is healthy
func (ws *WebSocketService) IsHealthy(ctx context.Context) bool {
	// Check Redis connectivity
	if !ws.redisHub.IsHealthy(ctx) {
		return false
	}

	// Service is healthy if Redis is connected
	return true
}

// GetServiceInfo returns service information
func (ws *WebSocketService) GetServiceInfo() map[string]interface{} {
	stats := ws.GetStats()

	return map[string]interface{}{
		"service":            "websocket",
		"status":             "healthy",
		"server_id":          ws.redisHub.GetServerID(),
		"total_connections":  stats.TotalConnections,
		"active_connections": stats.ActiveConnections,
		"total_rooms":        stats.TotalRooms,
		"messages_processed": stats.MessagesProcessed,
		"redis_channels":     []string{WebSocketChannel, WebSocketRoomChannel, WebSocketUserChannel, WebSocketSystemChannel},
	}
}
