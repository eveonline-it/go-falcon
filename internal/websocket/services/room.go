package services

import (
	"fmt"
	"sync"
	"time"

	"go-falcon/internal/websocket/models"
	"log/slog"
)

// RoomManager manages WebSocket rooms
type RoomManager struct {
	rooms           map[string]*models.Room // Map of room ID to room
	connectionRooms map[string][]string     // Map of connection ID to room IDs
	mu              sync.RWMutex
	connectionMgr   *ConnectionManager // Reference to connection manager (set later to avoid circular dependency)
}

// NewRoomManager creates a new room manager
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms:           make(map[string]*models.Room),
		connectionRooms: make(map[string][]string),
	}
}

// SetConnectionManager sets the connection manager reference
func (rm *RoomManager) SetConnectionManager(cm *ConnectionManager) {
	rm.connectionMgr = cm
}

// CreateRoom creates a new room
func (rm *RoomManager) CreateRoom(roomID string, roomType models.RoomType, name string) *models.Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if existingRoom, exists := rm.rooms[roomID]; exists {
		return existingRoom
	}

	room := &models.Room{
		ID:        roomID,
		Type:      roomType,
		Name:      name,
		Members:   []string{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	rm.rooms[roomID] = room
	slog.Info("WebSocket room created", "room_id", roomID, "type", roomType, "name", name)

	return room
}

// GetRoom retrieves a room by ID
func (rm *RoomManager) GetRoom(roomID string) (*models.Room, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	room, exists := rm.rooms[roomID]
	return room, exists
}

// GetAllRooms retrieves all rooms
func (rm *RoomManager) GetAllRooms() []*models.Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rooms := make([]*models.Room, 0, len(rm.rooms))
	for _, room := range rm.rooms {
		rooms = append(rooms, room)
	}

	return rooms
}

// GetRoomsByType retrieves rooms by type
func (rm *RoomManager) GetRoomsByType(roomType models.RoomType) []*models.Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rooms := []*models.Room{}
	for _, room := range rm.rooms {
		if room.Type == roomType {
			rooms = append(rooms, room)
		}
	}

	return rooms
}

// GetRoomCount returns the total number of rooms
func (rm *RoomManager) GetRoomCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.rooms)
}

// AddConnectionToRoom adds a connection to a room
func (rm *RoomManager) AddConnectionToRoom(roomID string, connectionID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	// Add connection to room
	room.AddMember(connectionID)

	// Track connection's rooms
	if rm.connectionRooms[connectionID] == nil {
		rm.connectionRooms[connectionID] = []string{}
	}

	// Check if already in room
	for _, existingRoomID := range rm.connectionRooms[connectionID] {
		if existingRoomID == roomID {
			return nil // Already in room
		}
	}

	rm.connectionRooms[connectionID] = append(rm.connectionRooms[connectionID], roomID)

	slog.Info("Connection added to room", "connection_id", connectionID, "room_id", roomID)

	// Notify other room members
	if rm.connectionMgr != nil {
		message := &models.Message{
			Type: models.MessageTypeRoomJoined,
			Room: roomID,
			Data: map[string]interface{}{
				"connection_id": connectionID,
				"room_id":       roomID,
				"member_count":  room.GetMemberCount(),
			},
			Timestamp: time.Now(),
		}
		rm.broadcastToRoomMembers(roomID, message, connectionID)
	}

	return nil
}

// RemoveConnectionFromRoom removes a connection from a room
func (rm *RoomManager) RemoveConnectionFromRoom(roomID string, connectionID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	// Remove connection from room
	room.RemoveMember(connectionID)

	// Update connection's rooms
	if rooms, ok := rm.connectionRooms[connectionID]; ok {
		newRooms := []string{}
		for _, existingRoomID := range rooms {
			if existingRoomID != roomID {
				newRooms = append(newRooms, existingRoomID)
			}
		}
		if len(newRooms) > 0 {
			rm.connectionRooms[connectionID] = newRooms
		} else {
			delete(rm.connectionRooms, connectionID)
		}
	}

	slog.Info("Connection removed from room", "connection_id", connectionID, "room_id", roomID)

	// Notify other room members
	if rm.connectionMgr != nil {
		message := &models.Message{
			Type: models.MessageTypeRoomLeft,
			Room: roomID,
			Data: map[string]interface{}{
				"connection_id": connectionID,
				"room_id":       roomID,
				"member_count":  room.GetMemberCount(),
			},
			Timestamp: time.Now(),
		}
		rm.broadcastToRoomMembers(roomID, message, connectionID)
	}

	// Clean up empty rooms (optional - you might want to keep them for history)
	if room.GetMemberCount() == 0 && room.Type != models.RoomTypePersonal {
		delete(rm.rooms, roomID)
		slog.Info("Empty room deleted", "room_id", roomID)
	}

	return nil
}

// GetConnectionRooms returns all rooms a connection is in
func (rm *RoomManager) GetConnectionRooms(connectionID string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if rooms, exists := rm.connectionRooms[connectionID]; exists {
		result := make([]string, len(rooms))
		copy(result, rooms)
		return result
	}

	return []string{}
}

// BroadcastToRoom broadcasts a message to all members of a room
func (rm *RoomManager) BroadcastToRoom(roomID string, message *models.Message) error {
	rm.mu.RLock()
	room, exists := rm.rooms[roomID]
	if !exists {
		rm.mu.RUnlock()
		return fmt.Errorf("room not found: %s", roomID)
	}

	// Get copy of members to avoid holding lock during broadcast
	members := room.GetMembersCopy()
	rm.mu.RUnlock()

	// Broadcast to all members
	for _, memberID := range members {
		if rm.connectionMgr != nil {
			if err := rm.connectionMgr.SendToConnection(memberID, message); err != nil {
				slog.Error("Failed to send message to room member", "error", err, "room_id", roomID, "connection_id", memberID)
			}
		}
	}

	return nil
}

// broadcastToRoomMembers broadcasts a message to all members except the excluded one
func (rm *RoomManager) broadcastToRoomMembers(roomID string, message *models.Message, excludeConnectionID string) {
	room, exists := rm.rooms[roomID]
	if !exists {
		return
	}

	members := room.GetMembersCopy()

	// Broadcast to all members except the excluded one
	for _, memberID := range members {
		if memberID != excludeConnectionID {
			if rm.connectionMgr != nil {
				if err := rm.connectionMgr.SendToConnection(memberID, message); err != nil {
					slog.Error("Failed to send message to room member", "error", err, "room_id", roomID, "connection_id", memberID)
				}
			}
		}
	}
}

// CreatePersonalRoom creates a personal room for a user
func (rm *RoomManager) CreatePersonalRoom(userID string) *models.Room {
	roomID := fmt.Sprintf("user:%s", userID)
	name := fmt.Sprintf("Personal Room")
	return rm.CreateRoom(roomID, models.RoomTypePersonal, name)
}

// CreateGroupRoom creates a group room
func (rm *RoomManager) CreateGroupRoom(groupID string, groupName string) *models.Room {
	roomID := fmt.Sprintf("group:%s", groupID)
	name := fmt.Sprintf("Group: %s", groupName)
	return rm.CreateRoom(roomID, models.RoomTypeGroup, name)
}

// JoinPersonalRoom joins a connection to their personal room
func (rm *RoomManager) JoinPersonalRoom(userID string, connectionID string) error {
	// Create personal room if it doesn't exist
	rm.CreatePersonalRoom(userID)

	roomID := fmt.Sprintf("user:%s", userID)
	return rm.AddConnectionToRoom(roomID, connectionID)
}

// JoinGroupRoom joins a connection to a group room
func (rm *RoomManager) JoinGroupRoom(groupID string, groupName string, connectionID string) error {
	// Create group room if it doesn't exist
	rm.CreateGroupRoom(groupID, groupName)

	roomID := fmt.Sprintf("group:%s", groupID)
	return rm.AddConnectionToRoom(roomID, connectionID)
}

// LeaveAllRooms removes a connection from all rooms
func (rm *RoomManager) LeaveAllRooms(connectionID string) {
	rooms := rm.GetConnectionRooms(connectionID)
	for _, roomID := range rooms {
		if err := rm.RemoveConnectionFromRoom(roomID, connectionID); err != nil {
			slog.Error("Failed to remove connection from room", "error", err, "connection_id", connectionID, "room_id", roomID)
		}
	}
}

// GetRoomMembers returns all members of a room
func (rm *RoomManager) GetRoomMembers(roomID string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return []string{}
	}

	return room.GetMembersCopy()
}

// IsConnectionInRoom checks if a connection is in a room
func (rm *RoomManager) IsConnectionInRoom(roomID string, connectionID string) bool {
	rooms := rm.GetConnectionRooms(connectionID)
	for _, room := range rooms {
		if room == roomID {
			return true
		}
	}
	return false
}
