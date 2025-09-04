package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-falcon/internal/websocket/models"

	"github.com/redis/go-redis/v9"
)

// Repository provides Redis storage for WebSocket metadata
type Repository struct {
	client *redis.Client
}

// NewRepository creates a new repository
func NewRepository(redisClient *redis.Client) *Repository {
	return &Repository{
		client: redisClient,
	}
}

// Redis key prefixes
const (
	ConnectionPrefix = "websocket:connection:"
	RoomPrefix       = "websocket:room:"
	UserPrefix       = "websocket:user:"
	StatsPrefix      = "websocket:stats"
)

// StoreConnectionInfo stores connection information in Redis
func (r *Repository) StoreConnectionInfo(ctx context.Context, conn *models.Connection) error {
	key := ConnectionPrefix + conn.ID

	data := map[string]interface{}{
		"id":             conn.ID,
		"user_id":        conn.UserID,
		"character_id":   conn.CharacterID,
		"character_name": conn.CharacterName,
		"rooms":          conn.Rooms,
		"created_at":     conn.CreatedAt,
		"last_ping":      conn.LastPing,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal connection data: %w", err)
	}

	// Store with TTL of 1 hour
	if err := r.client.Set(ctx, key, jsonData, time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store connection info: %w", err)
	}

	// Add to user's connection set
	userKey := UserPrefix + conn.UserID
	if err := r.client.SAdd(ctx, userKey, conn.ID).Err(); err != nil {
		return fmt.Errorf("failed to add to user connection set: %w", err)
	}

	// Set TTL for user key
	r.client.Expire(ctx, userKey, time.Hour)

	return nil
}

// GetConnectionInfo retrieves connection information from Redis
func (r *Repository) GetConnectionInfo(ctx context.Context, connectionID string) (*models.ConnectionInfo, error) {
	key := ConnectionPrefix + connectionID

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("connection not found: %s", connectionID)
		}
		return nil, fmt.Errorf("failed to get connection info: %w", err)
	}

	var connData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &connData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal connection data: %w", err)
	}

	// Convert to ConnectionInfo
	connInfo := &models.ConnectionInfo{
		ID:            getString(connData, "id"),
		UserID:        getString(connData, "user_id"),
		CharacterID:   getInt64(connData, "character_id"),
		CharacterName: getString(connData, "character_name"),
		Rooms:         getStringSlice(connData, "rooms"),
		CreatedAt:     getTime(connData, "created_at"),
	}

	return connInfo, nil
}

// RemoveConnectionInfo removes connection information from Redis
func (r *Repository) RemoveConnectionInfo(ctx context.Context, connectionID string, userID string) error {
	key := ConnectionPrefix + connectionID

	// Remove connection data
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to remove connection info: %w", err)
	}

	// Remove from user's connection set
	userKey := UserPrefix + userID
	if err := r.client.SRem(ctx, userKey, connectionID).Err(); err != nil {
		return fmt.Errorf("failed to remove from user connection set: %w", err)
	}

	return nil
}

// GetUserConnections retrieves all connection IDs for a user
func (r *Repository) GetUserConnections(ctx context.Context, userID string) ([]string, error) {
	userKey := UserPrefix + userID

	connections, err := r.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user connections: %w", err)
	}

	return connections, nil
}

// StoreRoomInfo stores room information in Redis
func (r *Repository) StoreRoomInfo(ctx context.Context, room *models.Room) error {
	key := RoomPrefix + room.ID

	data := map[string]interface{}{
		"id":         room.ID,
		"type":       string(room.Type),
		"name":       room.Name,
		"members":    room.Members,
		"created_at": room.CreatedAt,
		"updated_at": room.UpdatedAt,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal room data: %w", err)
	}

	// Store with TTL of 2 hours
	if err := r.client.Set(ctx, key, jsonData, 2*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store room info: %w", err)
	}

	return nil
}

// GetRoomInfo retrieves room information from Redis
func (r *Repository) GetRoomInfo(ctx context.Context, roomID string) (*models.RoomInfo, error) {
	key := RoomPrefix + roomID

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("room not found: %s", roomID)
		}
		return nil, fmt.Errorf("failed to get room info: %w", err)
	}

	var roomData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &roomData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal room data: %w", err)
	}

	// Convert to RoomInfo
	roomInfo := &models.RoomInfo{
		ID:          getString(roomData, "id"),
		Type:        models.RoomType(getString(roomData, "type")),
		Name:        getString(roomData, "name"),
		MemberCount: len(getStringSlice(roomData, "members")),
	}

	return roomInfo, nil
}

// RemoveRoomInfo removes room information from Redis
func (r *Repository) RemoveRoomInfo(ctx context.Context, roomID string) error {
	key := RoomPrefix + roomID

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to remove room info: %w", err)
	}

	return nil
}

// UpdateStats updates WebSocket statistics in Redis
func (r *Repository) UpdateStats(ctx context.Context, stats *models.WebSocketStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	if err := r.client.Set(ctx, StatsPrefix, data, time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to update stats: %w", err)
	}

	return nil
}

// GetStats retrieves WebSocket statistics from Redis
func (r *Repository) GetStats(ctx context.Context) (*models.WebSocketStats, error) {
	data, err := r.client.Get(ctx, StatsPrefix).Result()
	if err != nil {
		if err == redis.Nil {
			// Return default stats if not found
			return &models.WebSocketStats{}, nil
		}
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	var stats models.WebSocketStats
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
	}

	return &stats, nil
}

// CleanupExpiredData removes expired data from Redis
func (r *Repository) CleanupExpiredData(ctx context.Context) error {
	// Redis handles TTL automatically, but we can do additional cleanup here if needed
	// For now, just return nil as Redis will handle expiration
	return nil
}

// Helper functions for type conversion
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getInt64(data map[string]interface{}, key string) int64 {
	if val, ok := data[key].(float64); ok {
		return int64(val)
	}
	if val, ok := data[key].(int64); ok {
		return val
	}
	return 0
}

func getStringSlice(data map[string]interface{}, key string) []string {
	if val, ok := data[key].([]interface{}); ok {
		result := make([]string, len(val))
		for i, v := range val {
			if str, ok := v.(string); ok {
				result[i] = str
			}
		}
		return result
	}
	return []string{}
}

func getTime(data map[string]interface{}, key string) time.Time {
	if val, ok := data[key].(string); ok {
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t
		}
	}
	return time.Time{}
}
