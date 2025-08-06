package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-falcon/pkg/database"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"
)

// mockSDEService provides a mock SDE service for testing
type mockSDEService struct{}

func (m *mockSDEService) IsReady() bool                                     { return true }
func (m *mockSDEService) GetAgent(agentID string) (interface{}, error)      { return nil, nil }
func (m *mockSDEService) GetCategory(categoryID string) (interface{}, error) { return nil, nil }
func (m *mockSDEService) GetBlueprint(blueprintID string) (interface{}, error) { return nil, nil }
func (m *mockSDEService) GetAgentsByLocation(locationID int) ([]interface{}, error) { return nil, nil }

// mockMongoDB provides a mock MongoDB for testing
type mockMongoDB struct{}

func (m *mockMongoDB) Database() *mongo.Database { return nil }
func (m *mockMongoDB) Collection(name string) *mongo.Collection { return nil }

// mockRedis provides a mock Redis for testing
type mockRedis struct{}

func (m *mockRedis) Close() error { return nil }

// setupTestModule creates a test auth module
func setupTestModule() *Module {
	return &Module{
		BaseModule: nil, // We'll mock this if needed
	}
}

// Test that RequirePermissions middleware blocks users without permissions
func TestRequirePermissions_MissingPermission(t *testing.T) {
	module := setupTestModule()
	
	// Create a test handler that should be blocked
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	
	// Apply the middleware
	middleware := module.RequirePermissions("users:admin")
	handler := middleware(testHandler)
	
	// Create a request with a user that has no permissions
	req := httptest.NewRequest("GET", "/test", nil)
	user := &AuthenticatedUser{
		UserID:        "test-user-123",
		CharacterID:   123456789,
		CharacterName: "Test User",
		Scopes:        "publicData",
		Permissions:   []string{"users:read"}, // Has read but not admin
	}
	ctx := context.WithValue(req.Context(), AuthContextKeyUser, user)
	req = req.WithContext(ctx)
	
	// Record the response
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	// Should be forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// Test that RequirePermissions middleware allows users with correct permissions
func TestRequirePermissions_ValidPermission(t *testing.T) {
	module := setupTestModule()
	
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	
	// Apply the middleware
	middleware := module.RequirePermissions("users:admin")
	handler := middleware(testHandler)
	
	// Create a request with a user that has the required permission
	req := httptest.NewRequest("GET", "/test", nil)
	user := &AuthenticatedUser{
		UserID:        "test-user-123",
		CharacterID:   123456789,
		CharacterName: "Test User",
		Scopes:        "publicData",
		Permissions:   []string{"users:read", "users:admin"}, // Has required permission
	}
	ctx := context.WithValue(req.Context(), AuthContextKeyUser, user)
	req = req.WithContext(ctx)
	
	// Record the response
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	// Should succeed
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	// Check response body
	body := w.Body.String()
	if body != "success" {
		t.Errorf("Expected body 'success', got '%s'", body)
	}
}

// Test that RequirePermissions middleware blocks unauthenticated users
func TestRequirePermissions_NoAuthenticatedUser(t *testing.T) {
	module := setupTestModule()
	
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	
	// Apply the middleware
	middleware := module.RequirePermissions("users:admin")
	handler := middleware(testHandler)
	
	// Create a request with no authenticated user
	req := httptest.NewRequest("GET", "/test", nil)
	
	// Record the response
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	// Should be unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// Test that RequireAnyPermission middleware allows users with one of the required permissions
func TestRequireAnyPermission_HasOnePermission(t *testing.T) {
	module := setupTestModule()
	
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	
	// Apply the middleware requiring any of multiple permissions
	middleware := module.RequireAnyPermission("users:admin", "users:write", "system:admin")
	handler := middleware(testHandler)
	
	// Create a request with a user that has one of the required permissions
	req := httptest.NewRequest("GET", "/test", nil)
	user := &AuthenticatedUser{
		UserID:        "test-user-123",
		CharacterID:   123456789,
		CharacterName: "Test User",
		Scopes:        "publicData",
		Permissions:   []string{"users:read", "users:write"}, // Has users:write but not admin
	}
	ctx := context.WithValue(req.Context(), AuthContextKeyUser, user)
	req = req.WithContext(ctx)
	
	// Record the response
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	// Should succeed
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test that RequireAnyPermission middleware blocks users with none of the required permissions
func TestRequireAnyPermission_HasNoPermissions(t *testing.T) {
	module := setupTestModule()
	
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	
	// Apply the middleware requiring any of multiple permissions
	middleware := module.RequireAnyPermission("users:admin", "system:admin")
	handler := middleware(testHandler)
	
	// Create a request with a user that has none of the required permissions
	req := httptest.NewRequest("GET", "/test", nil)
	user := &AuthenticatedUser{
		UserID:        "test-user-123",
		CharacterID:   123456789,
		CharacterName: "Test User",
		Scopes:        "publicData",
		Permissions:   []string{"users:read"}, // Only has read permission
	}
	ctx := context.WithValue(req.Context(), AuthContextKeyUser, user)
	req = req.WithContext(ctx)
	
	// Record the response
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	// Should be forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// Test GetAuthenticatedUser helper function
func TestGetAuthenticatedUser(t *testing.T) {
	// Create a request with an authenticated user
	req := httptest.NewRequest("GET", "/test", nil)
	user := &AuthenticatedUser{
		UserID:        "test-user-123",
		CharacterID:   123456789,
		CharacterName: "Test User",
		Scopes:        "publicData",
		Permissions:   []string{"users:read"},
	}
	ctx := context.WithValue(req.Context(), AuthContextKeyUser, user)
	req = req.WithContext(ctx)
	
	// Get the user from context
	retrievedUser, ok := GetAuthenticatedUser(req)
	
	// Should succeed
	if !ok {
		t.Error("Expected to find authenticated user in context")
	}
	
	if retrievedUser.UserID != user.UserID {
		t.Errorf("Expected UserID %s, got %s", user.UserID, retrievedUser.UserID)
	}
	
	if retrievedUser.CharacterID != user.CharacterID {
		t.Errorf("Expected CharacterID %d, got %d", user.CharacterID, retrievedUser.CharacterID)
	}
}

// Test GetAuthenticatedUser with no user in context
func TestGetAuthenticatedUser_NoUser(t *testing.T) {
	// Create a request with no authenticated user
	req := httptest.NewRequest("GET", "/test", nil)
	
	// Try to get the user from context
	_, ok := GetAuthenticatedUser(req)
	
	// Should fail
	if ok {
		t.Error("Expected not to find authenticated user in context")
	}
}

// Test GetUserPermissions helper function
func TestGetUserPermissions(t *testing.T) {
	// Create a request with an authenticated user
	req := httptest.NewRequest("GET", "/test", nil)
	expectedPermissions := []string{"users:read", "users:write"}
	user := &AuthenticatedUser{
		UserID:        "test-user-123",
		CharacterID:   123456789,
		CharacterName: "Test User",
		Scopes:        "publicData",
		Permissions:   expectedPermissions,
	}
	ctx := context.WithValue(req.Context(), AuthContextKeyUser, user)
	req = req.WithContext(ctx)
	
	// Get the permissions from context
	permissions, ok := GetUserPermissions(req)
	
	// Should succeed
	if !ok {
		t.Error("Expected to find user permissions in context")
	}
	
	if len(permissions) != len(expectedPermissions) {
		t.Errorf("Expected %d permissions, got %d", len(expectedPermissions), len(permissions))
	}
	
	for i, perm := range permissions {
		if perm != expectedPermissions[i] {
			t.Errorf("Expected permission %s, got %s", expectedPermissions[i], perm)
		}
	}
}

// Test hasPermission utility function
func TestHasPermission(t *testing.T) {
	userPermissions := []string{"users:read", "users:write", "scheduler:read"}
	
	// Test existing permission
	if !hasPermission(userPermissions, "users:write") {
		t.Error("Expected to find users:write permission")
	}
	
	// Test non-existing permission
	if hasPermission(userPermissions, "users:admin") {
		t.Error("Expected not to find users:admin permission")
	}
	
	// Test empty permission list
	if hasPermission([]string{}, "users:read") {
		t.Error("Expected not to find permission in empty list")
	}
	
	// Test permission with spaces (should be trimmed)
	userPermissionsWithSpaces := []string{" users:read ", "users:write", " scheduler:read"}
	if !hasPermission(userPermissionsWithSpaces, "users:read") {
		t.Error("Expected to find users:read permission even with spaces")
	}
}