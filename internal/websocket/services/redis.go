package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-falcon/internal/websocket/models"
	"log/slog"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisHub handles Redis pub/sub for WebSocket message broadcasting
type RedisHub struct {
	client        *redis.Client
	serverID      string
	connectionMgr *ConnectionManager // Reference to connection manager (set later)
	roomMgr       *RoomManager       // Reference to room manager (set later)
	pubsub        *redis.PubSub
	channels      []string
}

const (
	// Redis channel for WebSocket messages
	WebSocketChannel = "websocket:messages"

	// Redis channel for room messages
	WebSocketRoomChannel = "websocket:rooms"

	// Redis channel for user messages
	WebSocketUserChannel = "websocket:users"

	// Redis channel for system messages
	WebSocketSystemChannel = "websocket:system"
)

// NewRedisHub creates a new Redis hub
func NewRedisHub(redisClient *redis.Client) *RedisHub {
	return &RedisHub{
		client:   redisClient,
		serverID: uuid.New().String(),
		channels: []string{
			WebSocketChannel,
			WebSocketRoomChannel,
			WebSocketUserChannel,
			WebSocketSystemChannel,
		},
	}
}

// SetConnectionManager sets the connection manager reference
func (rh *RedisHub) SetConnectionManager(cm *ConnectionManager) {
	rh.connectionMgr = cm
}

// SetRoomManager sets the room manager reference
func (rh *RedisHub) SetRoomManager(rm *RoomManager) {
	rh.roomMgr = rm
}

// Start begins listening for Redis pub/sub messages
func (rh *RedisHub) Start(ctx context.Context) error {
	rh.pubsub = rh.client.Subscribe(ctx, rh.channels...)

	slog.Info("WebSocket Redis hub started", "server_id", rh.serverID, "channels", rh.channels)

	go rh.listen(ctx)

	return nil
}

// Stop stops the Redis hub
func (rh *RedisHub) Stop() error {
	if rh.pubsub != nil {
		return rh.pubsub.Close()
	}
	return nil
}

// listen listens for Redis pub/sub messages
func (rh *RedisHub) listen(ctx context.Context) {
	ch := rh.pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			rh.handleRedisMessage(msg)
		}
	}
}

// handleRedisMessage handles incoming Redis messages
func (rh *RedisHub) handleRedisMessage(msg *redis.Message) {
	var redisMsg models.RedisMessage
	if err := json.Unmarshal([]byte(msg.Payload), &redisMsg); err != nil {
		slog.Error("Failed to unmarshal Redis message",
			"error", err,
			"channel", msg.Channel,
			"payload", msg.Payload)
		return
	}

	// Ignore messages from this server instance
	if redisMsg.ServerID == rh.serverID {
		return
	}

	// Handle the message based on the channel
	switch msg.Channel {
	case WebSocketChannel:
		rh.handleGeneralMessage(&redisMsg.Message)
	case WebSocketRoomChannel:
		rh.handleRoomMessage(&redisMsg.Message)
	case WebSocketUserChannel:
		rh.handleUserMessage(&redisMsg.Message)
	case WebSocketSystemChannel:
		rh.handleSystemMessage(&redisMsg.Message)
	}
}

// handleGeneralMessage handles general WebSocket messages
func (rh *RedisHub) handleGeneralMessage(message *models.Message) {
	if message.To != "" {
		// Direct message to specific connection
		if rh.connectionMgr != nil {
			rh.connectionMgr.SendToConnection(message.To, message)
		}
	} else {
		// Broadcast to all connections
		if rh.connectionMgr != nil {
			rh.connectionMgr.BroadcastToAll(message)
		}
	}
}

// handleRoomMessage handles room-specific messages
func (rh *RedisHub) handleRoomMessage(message *models.Message) {
	if message.Room != "" && rh.roomMgr != nil {
		rh.roomMgr.BroadcastToRoom(message.Room, message)
	}
}

// handleUserMessage handles user-specific messages
func (rh *RedisHub) handleUserMessage(message *models.Message) {
	if userID, ok := message.Data["user_id"].(string); ok && rh.connectionMgr != nil {
		rh.connectionMgr.SendToUser(userID, message)
	}
}

// handleSystemMessage handles system messages
func (rh *RedisHub) handleSystemMessage(message *models.Message) {
	// System messages are broadcast to all connections
	if rh.connectionMgr != nil {
		rh.connectionMgr.BroadcastToAll(message)
	}
}

// PublishMessage publishes a message to Redis
func (rh *RedisHub) PublishMessage(ctx context.Context, channel string, message *models.Message) error {
	redisMsg := models.RedisMessage{
		ServerID:  rh.serverID,
		Message:   *message,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(redisMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal Redis message: %w", err)
	}

	if err := rh.client.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish to Redis: %w", err)
	}

	slog.Debug("Message published to Redis", "channel", channel, "server_id", rh.serverID, "message_type", message.Type)
	return nil
}

// PublishToRoom publishes a message to a room across all instances
func (rh *RedisHub) PublishToRoom(ctx context.Context, roomID string, message *models.Message) error {
	message.Room = roomID
	return rh.PublishMessage(ctx, WebSocketRoomChannel, message)
}

// PublishToUser publishes a message to a user across all instances
func (rh *RedisHub) PublishToUser(ctx context.Context, userID string, message *models.Message) error {
	if message.Data == nil {
		message.Data = make(map[string]interface{})
	}
	message.Data["user_id"] = userID
	return rh.PublishMessage(ctx, WebSocketUserChannel, message)
}

// PublishSystemMessage publishes a system message across all instances
func (rh *RedisHub) PublishSystemMessage(ctx context.Context, message *models.Message) error {
	message.Type = models.MessageTypeSystemNotification
	return rh.PublishMessage(ctx, WebSocketSystemChannel, message)
}

// BroadcastUserProfileUpdate broadcasts a user profile update
func (rh *RedisHub) BroadcastUserProfileUpdate(ctx context.Context, userID string, characterID int64, profileData map[string]interface{}) error {
	message := &models.Message{
		Type: models.MessageTypeUserProfileUpdate,
		Data: map[string]interface{}{
			"user_id":      userID,
			"character_id": characterID,
			"profile":      profileData,
			"timestamp":    time.Now().Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	}

	return rh.PublishToUser(ctx, userID, message)
}

// BroadcastGroupMembershipChange broadcasts a group membership change
func (rh *RedisHub) BroadcastGroupMembershipChange(ctx context.Context, userID string, groupID string, groupName string, joined bool) error {
	message := &models.Message{
		Type: models.MessageTypeGroupMembershipChange,
		Data: map[string]interface{}{
			"user_id":    userID,
			"group_id":   groupID,
			"group_name": groupName,
			"joined":     joined,
			"timestamp":  time.Now().Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	}

	return rh.PublishToUser(ctx, userID, message)
}

// BroadcastToAllInstances broadcasts a message to all server instances
func (rh *RedisHub) BroadcastToAllInstances(ctx context.Context, message *models.Message) error {
	return rh.PublishMessage(ctx, WebSocketChannel, message)
}

// GetServerID returns the server ID
func (rh *RedisHub) GetServerID() string {
	return rh.serverID
}

// IsHealthy checks if Redis connection is healthy
func (rh *RedisHub) IsHealthy(ctx context.Context) bool {
	if rh.client == nil {
		return false
	}

	// Try a simple ping
	if err := rh.client.Ping(ctx).Err(); err != nil {
		slog.Error("Redis health check failed", "error", err)
		return false
	}

	return true
}

// GetStats returns Redis hub statistics
func (rh *RedisHub) GetStats(ctx context.Context) map[string]interface{} {
	stats := map[string]interface{}{
		"server_id": rh.serverID,
		"channels":  rh.channels,
		"healthy":   rh.IsHealthy(ctx),
	}

	// Try to get Redis info
	if rh.client != nil {
		info, err := rh.client.Info(ctx, "memory", "clients").Result()
		if err == nil {
			stats["redis_info"] = info
		}
	}

	return stats
}
