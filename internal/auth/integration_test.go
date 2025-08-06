package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// mockAuthModule provides a minimal auth module for integration testing
type mockAuthModule struct {
	permissions map[int][]string // characterID -> permissions
}

func (m *mockAuthModule) RequirePermissions(requiredPermissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Mock authenticated user for testing
			user := &AuthenticatedUser{
				UserID:        "test-admin-user",
				CharacterID:   999999999,
				CharacterName: "Admin User",
				Scopes:        "publicData",
				Permissions:   []string{"system:admin", "permissions:write", "permissions:read"},
			}
			
			// Add user to context
			ctx := r.Context()
			ctx = ctx.WithValue(AuthContextKeyUser, user)
			r = r.WithContext(ctx)
			
			next.ServeHTTP(w, r)
		})
	}
}

func (m *mockAuthModule) GetUserPermissions(characterID int) ([]string, error) {
	if perms, exists := m.permissions[characterID]; exists {
		return perms, nil
	}
	return []string{}, nil
}

func (m *mockAuthModule) AddUserPermission(characterID int, permission string) error {
	if m.permissions == nil {
		m.permissions = make(map[int][]string)
	}
	
	// Check if permission already exists
	for _, p := range m.permissions[characterID] {
		if p == permission {
			return nil // Already exists
		}
	}
	
	m.permissions[characterID] = append(m.permissions[characterID], permission)
	return nil
}

func (m *mockAuthModule) RemoveUserPermission(characterID int, permission string) error {
	if perms, exists := m.permissions[characterID]; exists {
		for i, p := range perms {
			if p == permission {
				m.permissions[characterID] = append(perms[:i], perms[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (m *mockAuthModule) SetUserPermissions(characterID int, permissions []string) error {
	if m.permissions == nil {
		m.permissions = make(map[int][]string)
	}
	m.permissions[characterID] = permissions
	return nil
}

// setupTestRouter creates a test router with permission management endpoints
func setupTestRouter() (*chi.Mux, *mockAuthModule) {
	mockAuth := &mockAuthModule{
		permissions: map[int][]string{
			123456789: {"users:read", "users:write"},
			987654321: {"scheduler:read"},
		},
	}
	
	// Create a minimal module with the mock methods
	module := &Module{}
	
	// Override the methods to use our mock
	module.GetUserPermissions = func(characterID int) ([]string, error) {
		return mockAuth.GetUserPermissions(characterID)
	}
	module.AddUserPermission = func(characterID int, permission string) error {
		return mockAuth.AddUserPermission(characterID, permission)
	}
	module.RemoveUserPermission = func(characterID int, permission string) error {
		return mockAuth.RemoveUserPermission(characterID, permission)
	}
	module.SetUserPermissions = func(characterID int, permissions []string) error {
		return mockAuth.SetUserPermissions(characterID, permissions)
	}
	
	r := chi.NewRouter()
	
	// Add permission management routes with mock auth
	r.With(mockAuth.RequirePermissions(PermissionPermissionsRead)).Get("/permissions", module.listAvailablePermissionsHandler)
	r.With(mockAuth.RequirePermissions(PermissionPermissionsRead)).Get("/permissions/user/{characterID}", module.listUserPermissionsHandler)
	r.With(mockAuth.RequirePermissions(PermissionPermissionsWrite)).Post("/permissions/user/{characterID}/grant", module.grantPermissionHandler)
	r.With(mockAuth.RequirePermissions(PermissionPermissionsWrite)).Post("/permissions/user/{characterID}/revoke", module.revokePermissionHandler)
	r.With(mockAuth.RequirePermissions(PermissionPermissionsWrite)).Put("/permissions/user/{characterID}", module.setUserPermissionsHandler)
	
	return r, mockAuth
}

// Test listing available permissions
func TestListAvailablePermissions_Integration(t *testing.T) {
	router, _ := setupTestRouter()
	
	req := httptest.NewRequest("GET", "/permissions", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	permissions, ok := response["permissions"].([]interface{})
	if !ok {
		t.Error("Expected 'permissions' field to be an array")
	}
	
	if len(permissions) == 0 {
		t.Error("Expected non-empty permissions array")
	}
	
	// Check that we have the expected total count
	totalCount, ok := response["total"].(float64)
	if !ok {
		t.Error("Expected 'total' field to be a number")
	}
	
	if int(totalCount) != len(permissions) {
		t.Errorf("Expected total count %d to match permissions array length %d", 
			int(totalCount), len(permissions))
	}
}

// Test listing user permissions
func TestListUserPermissions_Integration(t *testing.T) {
	router, _ := setupTestRouter()
	
	req := httptest.NewRequest("GET", "/permissions/user/123456789", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	characterID, ok := response["character_id"].(float64)
	if !ok || int(characterID) != 123456789 {
		t.Errorf("Expected character_id 123456789, got %v", characterID)
	}
	
	permissions, ok := response["permissions"].([]interface{})
	if !ok {
		t.Error("Expected 'permissions' field to be an array")
	}
	
	// Should have 2 permissions: users:read, users:write
	if len(permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(permissions))
	}
	
	// Check total count
	total, ok := response["total"].(float64)
	if !ok || int(total) != 2 {
		t.Errorf("Expected total 2, got %v", total)
	}
}

// Test granting permission to user
func TestGrantPermission_Integration(t *testing.T) {
	router, mockAuth := setupTestRouter()
	
	// Grant users:admin permission to user 123456789
	requestBody := map[string]string{
		"permission": "users:admin",
	}
	body, _ := json.Marshal(requestBody)
	
	req := httptest.NewRequest("POST", "/permissions/user/123456789/grant", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Error("Expected successful permission grant")
	}
	
	// Verify permission was actually added
	permissions, _ := mockAuth.GetUserPermissions(123456789)
	hasPermission := false
	for _, p := range permissions {
		if p == "users:admin" {
			hasPermission = true
			break
		}
	}
	
	if !hasPermission {
		t.Error("Expected user to have users:admin permission after grant")
	}
}

// Test revoking permission from user  
func TestRevokePermission_Integration(t *testing.T) {
	router, mockAuth := setupTestRouter()
	
	// Revoke users:write permission from user 123456789
	requestBody := map[string]string{
		"permission": "users:write",
	}
	body, _ := json.Marshal(requestBody)
	
	req := httptest.NewRequest("POST", "/permissions/user/123456789/revoke", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Error("Expected successful permission revocation")
	}
	
	// Verify permission was actually removed
	permissions, _ := mockAuth.GetUserPermissions(123456789)
	for _, p := range permissions {
		if p == "users:write" {
			t.Error("Expected users:write permission to be removed")
		}
	}
}

// Test setting user permissions (replace all)
func TestSetUserPermissions_Integration(t *testing.T) {
	router, mockAuth := setupTestRouter()
	
	// Set new permissions for user 123456789
	requestBody := map[string][]string{
		"permissions": {"users:admin", "scheduler:read", "notifications:write"},
	}
	body, _ := json.Marshal(requestBody)
	
	req := httptest.NewRequest("PUT", "/permissions/user/123456789", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Error("Expected successful permission update")
	}
	
	// Verify permissions were replaced
	permissions, _ := mockAuth.GetUserPermissions(123456789)
	expectedPermissions := []string{"users:admin", "scheduler:read", "notifications:write"}
	
	if len(permissions) != len(expectedPermissions) {
		t.Errorf("Expected %d permissions, got %d", len(expectedPermissions), len(permissions))
	}
	
	for _, expected := range expectedPermissions {
		found := false
		for _, actual := range permissions {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find permission '%s'", expected)
		}
	}
}

// Test granting invalid permission
func TestGrantInvalidPermission_Integration(t *testing.T) {
	router, _ := setupTestRouter()
	
	// Try to grant invalid permission
	requestBody := map[string]string{
		"permission": "invalid:permission",
	}
	body, _ := json.Marshal(requestBody)
	
	req := httptest.NewRequest("POST", "/permissions/user/123456789/grant", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid permission, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test invalid character ID
func TestInvalidCharacterID_Integration(t *testing.T) {
	router, _ := setupTestRouter()
	
	req := httptest.NewRequest("GET", "/permissions/user/invalid", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid character ID, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test setting permissions with invalid permission in list
func TestSetPermissions_WithInvalidPermission_Integration(t *testing.T) {
	router, _ := setupTestRouter()
	
	// Try to set permissions including an invalid one
	requestBody := map[string][]string{
		"permissions": {"users:read", "invalid:permission", "scheduler:read"},
	}
	body, _ := json.Marshal(requestBody)
	
	req := httptest.NewRequest("PUT", "/permissions/user/123456789", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid permission in list, got %d", http.StatusBadRequest, w.Code)
	}
}