package middleware

import (
	"context"
	"fmt"
	"net/http"

	"go-falcon/internal/auth/models"
	"go-falcon/pkg/middleware"

	"github.com/gorilla/websocket"
)

// WebSocketAuthMiddleware provides WebSocket-specific authentication
type WebSocketAuthMiddleware struct {
	authMiddleware *middleware.AuthMiddleware
}

// NewWebSocketAuthMiddleware creates a new WebSocket authentication middleware
func NewWebSocketAuthMiddleware(authMiddleware *middleware.AuthMiddleware) *WebSocketAuthMiddleware {
	return &WebSocketAuthMiddleware{
		authMiddleware: authMiddleware,
	}
}

// AuthenticateConnection validates authentication for WebSocket upgrade requests
func (m *WebSocketAuthMiddleware) AuthenticateConnection(r *http.Request) (*models.AuthenticatedUser, error) {
	// Extract auth header and cookie from the HTTP request
	authHeader := r.Header.Get("Authorization")
	cookieHeader := r.Header.Get("Cookie")

	// Extract token using low-level methods to avoid Huma errors
	token := m.authMiddleware.ExtractTokenFromHeaders(authHeader)
	if token == "" && cookieHeader != "" {
		token = m.authMiddleware.ExtractTokenFromCookie(cookieHeader)
	}

	if token == "" {
		return nil, fmt.Errorf("authentication required")
	}

	// Validate token using low-level method that returns standard Go errors
	user, err := m.authMiddleware.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid authentication token: %w", err)
	}

	return user, nil
}

// AuthenticateFromHeaders validates authentication from headers (for non-WebSocket endpoints)
func (m *WebSocketAuthMiddleware) AuthenticateFromHeaders(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	user, err := m.authMiddleware.ValidateAuthFromHeaders(authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// RequireWebSocketPermission checks if user has permission to use WebSocket
func (m *WebSocketAuthMiddleware) RequireWebSocketPermission(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	// For now, any authenticated user can use WebSocket
	// In the future, we can add specific permission checks here
	user, err := m.authMiddleware.ValidateAuthFromHeaders(authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}

	// Could add permission check like:
	// if !hasPermission(user, "websocket:connect") {
	//     return nil, huma.Error403Forbidden("Insufficient permissions for WebSocket")
	// }

	return user, nil
}

// RequireWebSocketAdmin checks if user has admin permission for WebSocket management
func (m *WebSocketAuthMiddleware) RequireWebSocketAdmin(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	user, err := m.authMiddleware.ValidateAuthFromHeaders(authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}

	// TODO: Implement proper admin check using groups service
	// For now, just return authenticated user - admin checks can be added later
	// Admin functionality in WebSocket is primarily for administrative API endpoints
	// not for connection establishment

	return user, nil
}

// UpgradeConnectionWithAuth performs WebSocket upgrade with authentication
func (m *WebSocketAuthMiddleware) UpgradeConnectionWithAuth(w http.ResponseWriter, r *http.Request) (*websocket.Conn, *models.AuthenticatedUser, error) {
	// First authenticate the request
	user, err := m.AuthenticateConnection(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, nil, err
	}

	// Configure the WebSocket upgrader
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// TEMPORARY: Allow all origins for bypass
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Upgrade the connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, nil, err
	}

	return conn, user, nil
}
