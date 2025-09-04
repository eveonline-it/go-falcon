package dto

import (
	"go-falcon/internal/websocket/models"
	"time"
)

// WebSocketConnectOutput represents the response for WebSocket connection
type WebSocketConnectOutput struct {
	Body struct {
		Success      bool     `json:"success" doc:"Whether the connection was successful"`
		ConnectionID string   `json:"connection_id,omitempty" doc:"Unique connection identifier"`
		UserID       string   `json:"user_id,omitempty" doc:"User UUID"`
		CharacterID  int64    `json:"character_id,omitempty" doc:"EVE character ID"`
		Rooms        []string `json:"rooms,omitempty" doc:"Initial rooms joined"`
		Message      string   `json:"message,omitempty" doc:"Status message"`
		WebSocketURL string   `json:"websocket_url,omitempty" doc:"WebSocket connection URL"`
	}
}

// SendMessageOutput represents the response for sending a message
type SendMessageOutput struct {
	Body struct {
		Success   bool      `json:"success" doc:"Whether the message was sent successfully"`
		MessageID string    `json:"message_id,omitempty" doc:"Unique message identifier"`
		Timestamp time.Time `json:"timestamp,omitempty" doc:"Message timestamp"`
		Message   string    `json:"message,omitempty" doc:"Status message"`
	}
}

// JoinRoomOutput represents the response for joining a room
type JoinRoomOutput struct {
	Body struct {
		Success  bool             `json:"success" doc:"Whether the room was joined successfully"`
		RoomID   string           `json:"room_id,omitempty" doc:"Room identifier"`
		RoomInfo *models.RoomInfo `json:"room_info,omitempty" doc:"Room information"`
		Message  string           `json:"message,omitempty" doc:"Status message"`
	}
}

// LeaveRoomOutput represents the response for leaving a room
type LeaveRoomOutput struct {
	Body struct {
		Success bool   `json:"success" doc:"Whether the room was left successfully"`
		RoomID  string `json:"room_id,omitempty" doc:"Room identifier"`
		Message string `json:"message,omitempty" doc:"Status message"`
	}
}

// BroadcastOutput represents the response for broadcasting a message
type BroadcastOutput struct {
	Body struct {
		Success         bool      `json:"success" doc:"Whether the broadcast was successful"`
		MessageID       string    `json:"message_id,omitempty" doc:"Unique message identifier"`
		RecipientsCount int       `json:"recipients_count,omitempty" doc:"Number of recipients"`
		Timestamp       time.Time `json:"timestamp,omitempty" doc:"Broadcast timestamp"`
		Message         string    `json:"message,omitempty" doc:"Status message"`
	}
}

// GetConnectionOutput represents the response for getting connection information
type GetConnectionOutput struct {
	Body struct {
		Connection *models.ConnectionInfo `json:"connection,omitempty" doc:"Connection information"`
	}
}

// GetRoomOutput represents the response for getting room information
type GetRoomOutput struct {
	Body struct {
		Room *models.RoomInfo `json:"room,omitempty" doc:"Room information"`
	}
}

// ListConnectionsOutput represents the response for listing connections
type ListConnectionsOutput struct {
	Body struct {
		Connections []models.ConnectionInfo `json:"connections" doc:"List of connections"`
		Total       int                     `json:"total" doc:"Total number of connections"`
	}
}

// ListRoomsOutput represents the response for listing rooms
type ListRoomsOutput struct {
	Body struct {
		Rooms []models.RoomInfo `json:"rooms" doc:"List of rooms"`
		Total int               `json:"total" doc:"Total number of rooms"`
	}
}

// WebSocketStatusOutput represents the WebSocket module status
type WebSocketStatusOutput struct {
	Body struct {
		Module  string                 `json:"module" doc:"Module name"`
		Status  string                 `json:"status" doc:"Module status (healthy/unhealthy)"`
		Stats   *models.WebSocketStats `json:"stats,omitempty" doc:"WebSocket statistics"`
		Message string                 `json:"message,omitempty" doc:"Status message"`
	}
}

// DirectMessageOutput represents the response for sending a direct message
type DirectMessageOutput struct {
	Body struct {
		Success      bool      `json:"success" doc:"Whether the message was sent successfully"`
		MessageID    string    `json:"message_id,omitempty" doc:"Unique message identifier"`
		ConnectionID string    `json:"connection_id,omitempty" doc:"Target connection ID"`
		Timestamp    time.Time `json:"timestamp,omitempty" doc:"Message timestamp"`
		Message      string    `json:"message,omitempty" doc:"Status message"`
	}
}

// UserMessageOutput represents the response for sending a message to user connections
type UserMessageOutput struct {
	Body struct {
		Success         bool      `json:"success" doc:"Whether the message was sent successfully"`
		MessageID       string    `json:"message_id,omitempty" doc:"Unique message identifier"`
		UserID          string    `json:"user_id,omitempty" doc:"Target user ID"`
		RecipientsCount int       `json:"recipients_count,omitempty" doc:"Number of user connections reached"`
		Timestamp       time.Time `json:"timestamp,omitempty" doc:"Message timestamp"`
		Message         string    `json:"message,omitempty" doc:"Status message"`
	}
}

// RoomMessageOutput represents the response for sending a message to a room
type RoomMessageOutput struct {
	Body struct {
		Success         bool      `json:"success" doc:"Whether the message was sent successfully"`
		MessageID       string    `json:"message_id,omitempty" doc:"Unique message identifier"`
		RoomID          string    `json:"room_id,omitempty" doc:"Target room ID"`
		RecipientsCount int       `json:"recipients_count,omitempty" doc:"Number of room members reached"`
		Timestamp       time.Time `json:"timestamp,omitempty" doc:"Message timestamp"`
		Message         string    `json:"message,omitempty" doc:"Status message"`
	}
}

// WebSocketErrorOutput represents an error response
type WebSocketErrorOutput struct {
	Body struct {
		Error   string `json:"error" doc:"Error code"`
		Message string `json:"message" doc:"Error message"`
		Details string `json:"details,omitempty" doc:"Additional error details"`
	}
}
