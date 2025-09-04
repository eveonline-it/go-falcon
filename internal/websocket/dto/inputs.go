package dto

import "go-falcon/internal/websocket/models"

// WebSocketConnectInput represents the input for WebSocket connection requests
type WebSocketConnectInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Cookie containing falcon_auth_token"`
}

// SendMessageInput represents the input for sending a message
type SendMessageInput struct {
	Type models.MessageType     `json:"type" required:"true" doc:"Message type"`
	Room string                 `json:"room,omitempty" doc:"Target room ID (optional)"`
	To   string                 `json:"to,omitempty" doc:"Target connection ID for direct messages (optional)"`
	Data map[string]interface{} `json:"data" required:"true" doc:"Message payload"`
}

// JoinRoomInput represents the input for joining a room
type JoinRoomInput struct {
	RoomID string `json:"room_id" required:"true" doc:"Room ID to join"`
}

// LeaveRoomInput represents the input for leaving a room
type LeaveRoomInput struct {
	RoomID string `json:"room_id" required:"true" doc:"Room ID to leave"`
}

// BroadcastInput represents the input for broadcasting a message
type BroadcastInput struct {
	Authorization string                 `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string                 `header:"Cookie" doc:"Cookie containing falcon_auth_token"`
	Type          models.MessageType     `json:"type" required:"true" doc:"Message type"`
	Room          string                 `json:"room,omitempty" doc:"Target room ID (if not provided, broadcasts to all)"`
	Data          map[string]interface{} `json:"data" required:"true" doc:"Message payload"`
}

// GetConnectionInput represents the input for getting connection information
type GetConnectionInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Cookie containing falcon_auth_token"`
	ConnectionID  string `path:"connection_id" doc:"Connection ID to retrieve"`
}

// GetRoomInput represents the input for getting room information
type GetRoomInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Cookie containing falcon_auth_token"`
	RoomID        string `path:"room_id" doc:"Room ID to retrieve"`
}

// ListConnectionsInput represents the input for listing connections
type ListConnectionsInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Cookie containing falcon_auth_token"`
	UserID        string `query:"user_id" doc:"Filter by user ID (optional)"`
	CharacterID   int64  `query:"character_id" doc:"Filter by character ID (optional)"`
	RoomID        string `query:"room_id" doc:"Filter by room ID (optional)"`
}

// ListRoomsInput represents the input for listing rooms
type ListRoomsInput struct {
	Authorization string          `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string          `header:"Cookie" doc:"Cookie containing falcon_auth_token"`
	Type          models.RoomType `query:"type" doc:"Filter by room type (optional)"`
	MemberID      string          `query:"member_id" doc:"Filter by member connection ID (optional)"`
}
