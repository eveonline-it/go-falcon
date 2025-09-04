package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go-falcon/internal/websocket/models"
	"log/slog"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ConnectionManager manages WebSocket connections
type ConnectionManager struct {
	connections map[string]*models.Connection // Map of connection ID to connection
	userConns   map[string][]string           // Map of user ID to connection IDs
	mu          sync.RWMutex
	roomManager *RoomManager
	redisHub    *RedisHub
	stats       models.WebSocketStats
	statsMu     sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(roomManager *RoomManager, redisHub *RedisHub) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*models.Connection),
		userConns:   make(map[string][]string),
		roomManager: roomManager,
		redisHub:    redisHub,
	}
}

// AddConnection adds a new connection
func (cm *ConnectionManager) AddConnection(conn *websocket.Conn, userID string, characterID int64, characterName string) (*models.Connection, error) {
	connectionID := uuid.New().String()

	connection := &models.Connection{
		ID:            connectionID,
		UserID:        userID,
		CharacterID:   characterID,
		CharacterName: characterName,
		Conn:          conn,
		Rooms:         []string{},
		CreatedAt:     time.Now(),
		LastPing:      time.Now(),
	}

	cm.mu.Lock()
	cm.connections[connectionID] = connection

	// Track user connections
	if cm.userConns[userID] == nil {
		cm.userConns[userID] = []string{}
	}
	cm.userConns[userID] = append(cm.userConns[userID], connectionID)
	cm.mu.Unlock()

	// Update stats
	cm.statsMu.Lock()
	cm.stats.TotalConnections++
	cm.stats.ActiveConnections++
	cm.stats.LastConnectionTime = time.Now()
	cm.statsMu.Unlock()

	slog.Info("WebSocket connection added", "connection_id", connectionID, "user_id", userID, "character_id", characterID)

	return connection, nil
}

// RemoveConnection removes a connection and cleans up
func (cm *ConnectionManager) RemoveConnection(connectionID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conn, exists := cm.connections[connectionID]
	if !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}

	// Remove from user connections
	if userConns, ok := cm.userConns[conn.UserID]; ok {
		newConns := []string{}
		for _, id := range userConns {
			if id != connectionID {
				newConns = append(newConns, id)
			}
		}
		if len(newConns) > 0 {
			cm.userConns[conn.UserID] = newConns
		} else {
			delete(cm.userConns, conn.UserID)
		}
	}

	// Leave all rooms
	for _, roomID := range conn.Rooms {
		if cm.roomManager != nil {
			cm.roomManager.RemoveConnectionFromRoom(roomID, connectionID)
		}
	}

	// Close the WebSocket connection
	if conn.Conn != nil {
		conn.Conn.Close()
	}

	delete(cm.connections, connectionID)

	// Update stats
	cm.statsMu.Lock()
	cm.stats.ActiveConnections--
	cm.statsMu.Unlock()

	slog.Info("WebSocket connection removed", "connection_id", connectionID, "user_id", conn.UserID)

	return nil
}

// GetConnection retrieves a connection by ID
func (cm *ConnectionManager) GetConnection(connectionID string) (*models.Connection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	conn, exists := cm.connections[connectionID]
	return conn, exists
}

// GetConnectionsByUser retrieves all connections for a user
func (cm *ConnectionManager) GetConnectionsByUser(userID string) []*models.Connection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	connIDs, exists := cm.userConns[userID]
	if !exists {
		return []*models.Connection{}
	}

	connections := make([]*models.Connection, 0, len(connIDs))
	for _, connID := range connIDs {
		if conn, ok := cm.connections[connID]; ok {
			connections = append(connections, conn)
		}
	}

	return connections
}

// GetConnectionsByCharacter retrieves all connections for a character
func (cm *ConnectionManager) GetConnectionsByCharacter(characterID int64) []*models.Connection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	connections := []*models.Connection{}
	for _, conn := range cm.connections {
		if conn.CharacterID == characterID {
			connections = append(connections, conn)
		}
	}

	return connections
}

// GetAllConnections retrieves all active connections
func (cm *ConnectionManager) GetAllConnections() []*models.Connection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	connections := make([]*models.Connection, 0, len(cm.connections))
	for _, conn := range cm.connections {
		connections = append(connections, conn)
	}

	return connections
}

// SendToConnection sends a message to a specific connection
func (cm *ConnectionManager) SendToConnection(connectionID string, message *models.Message) error {
	cm.mu.RLock()
	conn, exists := cm.connections[connectionID]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}

	// Update stats
	cm.statsMu.Lock()
	cm.stats.MessagesProcessed++
	cm.statsMu.Unlock()

	return cm.writeToConnection(conn, message)
}

// SendToUser sends a message to all connections of a user
func (cm *ConnectionManager) SendToUser(userID string, message *models.Message) error {
	connections := cm.GetConnectionsByUser(userID)

	if len(connections) == 0 {
		return fmt.Errorf("no connections found for user: %s", userID)
	}

	var lastErr error
	for _, conn := range connections {
		if err := cm.writeToConnection(conn, message); err != nil {
			lastErr = err
			slog.Error("Failed to send message to connection", "error", err, "connection_id", conn.ID)
		}
	}

	// Update stats
	cm.statsMu.Lock()
	cm.stats.MessagesProcessed++
	cm.stats.MessagesBroadcast++
	cm.statsMu.Unlock()

	return lastErr
}

// BroadcastToAll sends a message to all connections
func (cm *ConnectionManager) BroadcastToAll(message *models.Message) {
	connections := cm.GetAllConnections()

	for _, conn := range connections {
		if err := cm.writeToConnection(conn, message); err != nil {
			slog.Error("Failed to broadcast message to connection", "error", err, "connection_id", conn.ID)
		}
	}

	// Update stats
	cm.statsMu.Lock()
	cm.stats.MessagesProcessed++
	cm.stats.MessagesBroadcast += int64(len(connections))
	cm.statsMu.Unlock()
}

// writeToConnection writes a message to a WebSocket connection
func (cm *ConnectionManager) writeToConnection(conn *models.Connection, message *models.Message) error {
	// Set write deadline
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Marshal message to JSON
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send the message
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// HandleConnection handles a WebSocket connection lifecycle
func (cm *ConnectionManager) HandleConnection(ctx context.Context, conn *models.Connection) {
	defer func() {
		cm.RemoveConnection(conn.ID)
	}()

	// Send welcome message
	welcomeMsg := &models.Message{
		Type: models.MessageTypeSystemNotification,
		Data: map[string]interface{}{
			"message":       "Connected to WebSocket",
			"connection_id": conn.ID,
			"user_id":       conn.UserID,
		},
		Timestamp: time.Now(),
	}
	cm.SendToConnection(conn.ID, welcomeMsg)

	// Set up ping/pong to keep connection alive
	conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.Conn.SetPongHandler(func(string) error {
		conn.UpdateLastPing()
		conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Message reading goroutine
	messageChan := make(chan []byte, 256)
	errorChan := make(chan error, 1)

	go func() {
		for {
			_, message, err := conn.Conn.ReadMessage()
			if err != nil {
				errorChan <- err
				return
			}
			messageChan <- message
		}
	}()

	// Main loop
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("Failed to send ping", "error", err, "connection_id", conn.ID)
				return
			}

		case message := <-messageChan:
			// Process incoming message
			var msg models.Message
			if err := json.Unmarshal(message, &msg); err != nil {
				slog.Error("Failed to unmarshal message", "error", err, "connection_id", conn.ID)
				continue
			}

			// Handle the message based on its type
			cm.handleMessage(conn, &msg)

		case err := <-errorChan:
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("WebSocket error", "error", err, "connection_id", conn.ID)
			}
			return
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (cm *ConnectionManager) handleMessage(conn *models.Connection, message *models.Message) {
	message.From = conn.ID
	message.Timestamp = time.Now()

	switch message.Type {
	case models.MessageTypeHeartbeat:
		// Send heartbeat response
		response := &models.Message{
			Type: models.MessageTypeHeartbeat,
			Data: map[string]interface{}{
				"timestamp": time.Now(),
			},
			Timestamp: time.Now(),
		}
		cm.SendToConnection(conn.ID, response)

	case models.MessageTypeCustomEvent:
		// Handle custom events based on room
		if message.Room != "" {
			// Broadcast to room members
			if cm.roomManager != nil {
				cm.roomManager.BroadcastToRoom(message.Room, message)
			}
		} else if message.To != "" {
			// Direct message to specific connection
			cm.SendToConnection(message.To, message)
		}

	default:
		// Log unhandled message type
		slog.Warn("Unhandled message type", "type", message.Type, "connection_id", conn.ID)
	}

	// Update stats
	cm.statsMu.Lock()
	cm.stats.MessagesProcessed++
	cm.statsMu.Unlock()
}

// GetStats returns WebSocket statistics
func (cm *ConnectionManager) GetStats() models.WebSocketStats {
	cm.statsMu.RLock()
	defer cm.statsMu.RUnlock()

	stats := cm.stats
	stats.TotalRooms = cm.roomManager.GetRoomCount()

	return stats
}

// CleanupInactiveConnections removes inactive connections
func (cm *ConnectionManager) CleanupInactiveConnections() {
	cm.mu.RLock()
	inactiveConnections := []string{}
	for id, conn := range cm.connections {
		if !conn.IsAlive() {
			inactiveConnections = append(inactiveConnections, id)
		}
	}
	cm.mu.RUnlock()

	for _, id := range inactiveConnections {
		slog.Info("Removing inactive connection", "connection_id", id)
		cm.RemoveConnection(id)
	}
}
