package models

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// RoomType represents the type of room
type RoomType string

const (
	RoomTypePersonal RoomType = "personal" // Personal user room (user:{user_id})
	RoomTypeGroup    RoomType = "group"    // Group room (group:{group_id})
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeUserProfileUpdate     MessageType = "user_profile_update"
	MessageTypeGroupMembershipChange MessageType = "group_membership_change"
	MessageTypeSystemNotification    MessageType = "system_notification"
	MessageTypeCustomEvent           MessageType = "custom_event"
	MessageTypeHeartbeat             MessageType = "heartbeat"
	MessageTypeError                 MessageType = "error"
	MessageTypeRoomJoined            MessageType = "room_joined"
	MessageTypeRoomLeft              MessageType = "room_left"
)

// Connection represents a WebSocket connection
type Connection struct {
	ID            string          `json:"id"`             // Unique connection ID
	UserID        string          `json:"user_id"`        // User UUID
	CharacterID   int64           `json:"character_id"`   // EVE Character ID
	CharacterName string          `json:"character_name"` // EVE Character Name
	Conn          *websocket.Conn `json:"-"`              // WebSocket connection (not serialized)
	Rooms         []string        `json:"rooms"`          // List of room IDs the connection is in
	CreatedAt     time.Time       `json:"created_at"`
	LastPing      time.Time       `json:"last_ping"`
	mu            sync.RWMutex    // Protects concurrent access
}

// Room represents a WebSocket room
type Room struct {
	ID        string    `json:"id"`      // Room ID (user:{id} or group:{id})
	Type      RoomType  `json:"type"`    // Room type (personal or group)
	Name      string    `json:"name"`    // Human-readable room name
	Members   []string  `json:"members"` // List of connection IDs
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	mu        sync.RWMutex
}

// Message represents a WebSocket message
type Message struct {
	ID        string                 `json:"id,omitempty"`
	Type      MessageType            `json:"type"`
	Room      string                 `json:"room,omitempty"` // Target room (if applicable)
	From      string                 `json:"from,omitempty"` // Connection ID of sender
	To        string                 `json:"to,omitempty"`   // Target connection ID (for direct messages)
	Data      map[string]interface{} `json:"data,omitempty"` // Message payload
	Timestamp time.Time              `json:"timestamp"`
}

// RedisMessage represents a message for Redis pub/sub
type RedisMessage struct {
	ServerID  string    `json:"server_id"` // ID of the server that published the message
	Message   Message   `json:"message"`   // The actual message
	Timestamp time.Time `json:"timestamp"`
}

// ConnectionInfo represents public connection information
type ConnectionInfo struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	CharacterID   int64     `json:"character_id"`
	CharacterName string    `json:"character_name"`
	Rooms         []string  `json:"rooms"`
	CreatedAt     time.Time `json:"created_at"`
}

// RoomInfo represents public room information
type RoomInfo struct {
	ID          string   `json:"id"`
	Type        RoomType `json:"type"`
	Name        string   `json:"name"`
	MemberCount int      `json:"member_count"`
}

// WebSocketStats represents WebSocket module statistics
type WebSocketStats struct {
	TotalConnections   int       `json:"total_connections"`
	ActiveConnections  int       `json:"active_connections"`
	TotalRooms         int       `json:"total_rooms"`
	MessagesProcessed  int64     `json:"messages_processed"`
	MessagesBroadcast  int64     `json:"messages_broadcast"`
	LastConnectionTime time.Time `json:"last_connection_time,omitempty"`
}

// IsAlive checks if the connection is still alive
func (c *Connection) IsAlive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.LastPing) < 60*time.Second
}

// UpdateLastPing updates the last ping timestamp
func (c *Connection) UpdateLastPing() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastPing = time.Now()
}

// AddRoom adds a room to the connection's room list
func (c *Connection) AddRoom(roomID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, room := range c.Rooms {
		if room == roomID {
			return // Already in room
		}
	}
	c.Rooms = append(c.Rooms, roomID)
}

// RemoveRoom removes a room from the connection's room list
func (c *Connection) RemoveRoom(roomID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	newRooms := []string{}
	for _, room := range c.Rooms {
		if room != roomID {
			newRooms = append(newRooms, room)
		}
	}
	c.Rooms = newRooms
}

// ToConnectionInfo converts Connection to ConnectionInfo (public representation)
func (c *Connection) ToConnectionInfo() ConnectionInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return ConnectionInfo{
		ID:            c.ID,
		UserID:        c.UserID,
		CharacterID:   c.CharacterID,
		CharacterName: c.CharacterName,
		Rooms:         c.Rooms,
		CreatedAt:     c.CreatedAt,
	}
}

// WriteMessage writes a message to the WebSocket connection safely
func (c *Connection) WriteMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Conn == nil {
		return fmt.Errorf("connection is nil")
	}

	return c.Conn.WriteMessage(messageType, data)
}

// WriteJSON writes a JSON message to the WebSocket connection safely
func (c *Connection) WriteJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Conn == nil {
		return fmt.Errorf("connection is nil")
	}

	return c.Conn.WriteJSON(v)
}

// SetWriteDeadline sets write deadline safely
func (c *Connection) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Conn == nil {
		return fmt.Errorf("connection is nil")
	}

	return c.Conn.SetWriteDeadline(t)
}

// AddMember adds a member to the room
func (r *Room) AddMember(connectionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, member := range r.Members {
		if member == connectionID {
			return // Already a member
		}
	}
	r.Members = append(r.Members, connectionID)
	r.UpdatedAt = time.Now()
}

// RemoveMember removes a member from the room
func (r *Room) RemoveMember(connectionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	newMembers := []string{}
	for _, member := range r.Members {
		if member != connectionID {
			newMembers = append(newMembers, member)
		}
	}
	r.Members = newMembers
	r.UpdatedAt = time.Now()
}

// GetMemberCount returns the number of members in the room
func (r *Room) GetMemberCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Members)
}

// ToRoomInfo converts Room to RoomInfo (public representation)
func (r *Room) ToRoomInfo() RoomInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return RoomInfo{
		ID:          r.ID,
		Type:        r.Type,
		Name:        r.Name,
		MemberCount: len(r.Members),
	}
}

// GetMembersCopy returns a copy of the members list
func (r *Room) GetMembersCopy() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	members := make([]string, len(r.Members))
	copy(members, r.Members)
	return members
}

// HasMember checks if a connection is a member of the room
func (r *Room) HasMember(connectionID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, member := range r.Members {
		if member == connectionID {
			return true
		}
	}
	return false
}
