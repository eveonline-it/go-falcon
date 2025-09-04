package routes

import (
	"context"
	"net/http"
	"time"

	"go-falcon/internal/websocket/dto"
	"go-falcon/internal/websocket/middleware"
	"go-falcon/internal/websocket/models"
	"go-falcon/internal/websocket/services"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// WebSocketRoutes handles WebSocket API endpoints
type WebSocketRoutes struct {
	service    *services.WebSocketService
	authMw     *middleware.WebSocketAuthMiddleware
	repository *services.Repository
}

// NewWebSocketRoutes creates a new WebSocket routes handler
func NewWebSocketRoutes(service *services.WebSocketService, authMw *middleware.WebSocketAuthMiddleware, repository *services.Repository) *WebSocketRoutes {
	return &WebSocketRoutes{
		service:    service,
		authMw:     authMw,
		repository: repository,
	}
}

// RegisterRoutes registers all WebSocket routes
func (wr *WebSocketRoutes) RegisterRoutes(api huma.API) {
	// NOTE: WebSocket connection endpoint is registered directly with HTTP router
	// via RegisterHTTPHandler method, not through Huma API, to properly handle
	// WebSocket protocol upgrade which requires direct HTTP response control.

	// Administrative endpoints
	huma.Register(api, huma.Operation{
		OperationID: "websocket-list-connections",
		Method:      http.MethodGet,
		Path:        "/websocket/connections",
		Summary:     "List active WebSocket connections",
		Description: "Retrieve list of active WebSocket connections (admin only)",
		Tags:        []string{"WebSocket Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}, wr.handleListConnections)

	huma.Register(api, huma.Operation{
		OperationID: "websocket-get-connection",
		Method:      http.MethodGet,
		Path:        "/websocket/connections/{connection_id}",
		Summary:     "Get WebSocket connection details",
		Description: "Retrieve details of a specific WebSocket connection (admin only)",
		Tags:        []string{"WebSocket Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}, wr.handleGetConnection)

	huma.Register(api, huma.Operation{
		OperationID: "websocket-list-rooms",
		Method:      http.MethodGet,
		Path:        "/websocket/rooms",
		Summary:     "List WebSocket rooms",
		Description: "Retrieve list of WebSocket rooms (admin only)",
		Tags:        []string{"WebSocket Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}, wr.handleListRooms)

	huma.Register(api, huma.Operation{
		OperationID: "websocket-get-room",
		Method:      http.MethodGet,
		Path:        "/websocket/rooms/{room_id}",
		Summary:     "Get WebSocket room details",
		Description: "Retrieve details of a specific WebSocket room (admin only)",
		Tags:        []string{"WebSocket Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}, wr.handleGetRoom)

	huma.Register(api, huma.Operation{
		OperationID: "websocket-broadcast",
		Method:      http.MethodPost,
		Path:        "/websocket/broadcast",
		Summary:     "Broadcast message",
		Description: "Broadcast a message to WebSocket connections (admin only)",
		Tags:        []string{"WebSocket Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}, wr.handleBroadcast)

	// Status endpoint
	huma.Register(api, huma.Operation{
		OperationID: "websocket-status",
		Method:      http.MethodGet,
		Path:        "/websocket/status",
		Summary:     "WebSocket module status",
		Description: "Get WebSocket module health and statistics",
		Tags:        []string{"Module Status"},
	}, wr.handleStatus)
}

// HandleWebSocketUpgrade handles actual WebSocket upgrade (called by HTTP handler)
func (wr *WebSocketRoutes) HandleWebSocketUpgrade(w http.ResponseWriter, r *http.Request) {
	// Authenticate and upgrade connection
	conn, user, err := wr.authMw.UpgradeConnectionWithAuth(w, r)
	if err != nil {
		slog.Error("Failed to upgrade WebSocket connection", "error", err)
		return
	}

	// Create connection model
	connection := &models.Connection{
		ID:            uuid.New().String(),
		UserID:        user.UserID,
		CharacterID:   int64(user.CharacterID),
		CharacterName: user.CharacterName,
		Conn:          conn,
		Rooms:         []string{},
		CreatedAt:     time.Now(),
		LastPing:      time.Now(),
	}

	// Add connection to service
	if err := wr.service.CreateConnection(connection); err != nil {
		slog.Error("Failed to create WebSocket connection", "error", err)
		conn.Close()
		return
	}

	// Handle connection lifecycle
	connectionMgr := wr.service.GetConnectionManager()
	connectionMgr.HandleConnection(r.Context(), connection)
}

// handleListConnections lists active WebSocket connections
func (wr *WebSocketRoutes) handleListConnections(ctx context.Context, input *dto.ListConnectionsInput) (*dto.ListConnectionsOutput, error) {
	// Require admin access
	_, err := wr.authMw.RequireWebSocketAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	connectionMgr := wr.service.GetConnectionManager()
	var connections []*models.Connection

	if input.UserID != "" {
		connections = connectionMgr.GetConnectionsByUser(input.UserID)
	} else if input.CharacterID != 0 {
		connections = connectionMgr.GetConnectionsByCharacter(input.CharacterID)
	} else {
		connections = connectionMgr.GetAllConnections()
	}

	// Filter by room if specified
	if input.RoomID != "" {
		roomMgr := wr.service.GetRoomManager()
		filteredConns := []*models.Connection{}
		for _, conn := range connections {
			if roomMgr.IsConnectionInRoom(input.RoomID, conn.ID) {
				filteredConns = append(filteredConns, conn)
			}
		}
		connections = filteredConns
	}

	// Convert to ConnectionInfo
	connectionInfos := make([]models.ConnectionInfo, len(connections))
	for i, conn := range connections {
		connectionInfos[i] = conn.ToConnectionInfo()
	}

	return &dto.ListConnectionsOutput{
		Body: struct {
			Connections []models.ConnectionInfo `json:"connections" doc:"List of connections"`
			Total       int                     `json:"total" doc:"Total number of connections"`
		}{
			Connections: connectionInfos,
			Total:       len(connectionInfos),
		},
	}, nil
}

// handleGetConnection gets specific connection details
func (wr *WebSocketRoutes) handleGetConnection(ctx context.Context, input *dto.GetConnectionInput) (*dto.GetConnectionOutput, error) {
	// Require admin access
	_, err := wr.authMw.RequireWebSocketAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	connectionMgr := wr.service.GetConnectionManager()
	conn, exists := connectionMgr.GetConnection(input.ConnectionID)
	if !exists {
		return nil, huma.Error404NotFound("Connection not found")
	}

	connInfo := conn.ToConnectionInfo()
	return &dto.GetConnectionOutput{
		Body: struct {
			Connection *models.ConnectionInfo `json:"connection,omitempty" doc:"Connection information"`
		}{
			Connection: &connInfo,
		},
	}, nil
}

// handleListRooms lists WebSocket rooms
func (wr *WebSocketRoutes) handleListRooms(ctx context.Context, input *dto.ListRoomsInput) (*dto.ListRoomsOutput, error) {
	// Require admin access
	_, err := wr.authMw.RequireWebSocketAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	roomMgr := wr.service.GetRoomManager()
	var rooms []*models.Room

	if input.Type != "" {
		rooms = roomMgr.GetRoomsByType(input.Type)
	} else {
		rooms = roomMgr.GetAllRooms()
	}

	// Filter by member if specified
	if input.MemberID != "" {
		filteredRooms := []*models.Room{}
		for _, room := range rooms {
			if roomMgr.IsConnectionInRoom(room.ID, input.MemberID) {
				filteredRooms = append(filteredRooms, room)
			}
		}
		rooms = filteredRooms
	}

	// Convert to RoomInfo
	roomInfos := make([]models.RoomInfo, len(rooms))
	for i, room := range rooms {
		roomInfos[i] = room.ToRoomInfo()
	}

	return &dto.ListRoomsOutput{
		Body: struct {
			Rooms []models.RoomInfo `json:"rooms" doc:"List of rooms"`
			Total int               `json:"total" doc:"Total number of rooms"`
		}{
			Rooms: roomInfos,
			Total: len(roomInfos),
		},
	}, nil
}

// handleGetRoom gets specific room details
func (wr *WebSocketRoutes) handleGetRoom(ctx context.Context, input *dto.GetRoomInput) (*dto.GetRoomOutput, error) {
	// Require admin access
	_, err := wr.authMw.RequireWebSocketAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	roomMgr := wr.service.GetRoomManager()
	room, exists := roomMgr.GetRoom(input.RoomID)
	if !exists {
		return nil, huma.Error404NotFound("Room not found")
	}

	roomInfo := room.ToRoomInfo()
	return &dto.GetRoomOutput{
		Body: struct {
			Room *models.RoomInfo `json:"room,omitempty" doc:"Room information"`
		}{
			Room: &roomInfo,
		},
	}, nil
}

// handleBroadcast handles message broadcasting
func (wr *WebSocketRoutes) handleBroadcast(ctx context.Context, input *dto.BroadcastInput) (*dto.BroadcastOutput, error) {
	// Require admin access
	_, err := wr.authMw.RequireWebSocketAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	message := &models.Message{
		ID:        uuid.New().String(),
		Type:      input.Type,
		Room:      input.Room,
		Data:      input.Data,
		Timestamp: time.Now(),
	}

	var recipientsCount int

	if input.Room != "" {
		// Broadcast to specific room
		roomMgr := wr.service.GetRoomManager()
		members := roomMgr.GetRoomMembers(input.Room)
		recipientsCount = len(members)

		if err := wr.service.SendMessage(message); err != nil {
			return nil, huma.Error500InternalServerError("Failed to broadcast message", err)
		}

		// Also publish via Redis for other instances
		redisHub := wr.service.GetRedisHub()
		redisHub.PublishToRoom(ctx, input.Room, message)
	} else {
		// Broadcast to all connections
		connectionMgr := wr.service.GetConnectionManager()
		connections := connectionMgr.GetAllConnections()
		recipientsCount = len(connections)

		wr.service.SendMessage(message)

		// Also publish via Redis for other instances
		redisHub := wr.service.GetRedisHub()
		redisHub.BroadcastToAllInstances(ctx, message)
	}

	return &dto.BroadcastOutput{
		Body: struct {
			Success         bool      `json:"success" doc:"Whether the broadcast was successful"`
			MessageID       string    `json:"message_id,omitempty" doc:"Unique message identifier"`
			RecipientsCount int       `json:"recipients_count,omitempty" doc:"Number of recipients"`
			Timestamp       time.Time `json:"timestamp,omitempty" doc:"Broadcast timestamp"`
			Message         string    `json:"message,omitempty" doc:"Status message"`
		}{
			Success:         true,
			MessageID:       message.ID,
			RecipientsCount: recipientsCount,
			Timestamp:       message.Timestamp,
			Message:         "Message broadcast successfully",
		},
	}, nil
}

// handleStatus returns WebSocket module status
func (wr *WebSocketRoutes) handleStatus(ctx context.Context, input *struct{}) (*dto.WebSocketStatusOutput, error) {
	stats := wr.service.GetStats()
	isHealthy := wr.service.IsHealthy(ctx)

	status := "healthy"
	message := "WebSocket service is running normally"

	if !isHealthy {
		status = "unhealthy"
		message = "WebSocket service has connectivity issues"
	}

	return &dto.WebSocketStatusOutput{
		Body: struct {
			Module  string                 `json:"module" doc:"Module name"`
			Status  string                 `json:"status" doc:"Module status (healthy/unhealthy)"`
			Stats   *models.WebSocketStats `json:"stats,omitempty" doc:"WebSocket statistics"`
			Message string                 `json:"message,omitempty" doc:"Status message"`
		}{
			Module:  "websocket",
			Status:  status,
			Stats:   &stats,
			Message: message,
		},
	}, nil
}
