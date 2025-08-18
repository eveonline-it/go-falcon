package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-falcon/internal/auth/models"
)

// Mock JWT validator for testing
type mockJWTValidator struct{}

func (m *mockJWTValidator) ValidateJWT(token string) (*models.AuthenticatedUser, error) {
	if token == "valid-token" {
		return &models.AuthenticatedUser{
			UserID:        "test-user-id",
			CharacterID:   123456789,
			CharacterName: "Test Character",
			Scopes:        "publicData",
		}, nil
	}
	return nil, &AuthError{message: "invalid token"}
}

// Mock user character resolver for testing
type mockUserCharacterResolver struct{}

func (m *mockUserCharacterResolver) GetUserWithCharacters(ctx context.Context, userID string) (*UserWithCharacters, error) {
	if userID == "test-user-id" {
		return &UserWithCharacters{
			ID: userID,
			Characters: []UserCharacter{
				{
					CharacterID:   123456789,
					Name:          "Test Character",
					CorporationID: 98000001,
					AllianceID:    99000001,
					IsPrimary:     true,
				},
				{
					CharacterID:   987654321,
					Name:          "Alt Character",
					CorporationID: 98000001,
					AllianceID:    99000001,
					IsPrimary:     false,
				},
			},
		}, nil
	}
	return nil, &AuthError{message: "user not found"}
}

func TestAuthenticationMiddleware(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	middleware := NewEnhancedAuthMiddleware(validator, resolver)

	// Test handler that checks for auth context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r.Context())
		if authCtx == nil {
			t.Error("Expected auth context to be set")
			return
		}
		
		if authCtx.UserID != "test-user-id" {
			t.Errorf("Expected user ID 'test-user-id', got '%s'", authCtx.UserID)
		}
		
		if authCtx.PrimaryCharID != 123456789 {
			t.Errorf("Expected character ID 123456789, got %d", authCtx.PrimaryCharID)
		}
		
		if authCtx.RequestType != "bearer" {
			t.Errorf("Expected request type 'bearer', got '%s'", authCtx.RequestType)
		}
		
		w.WriteHeader(http.StatusOK)
	})

	// Create request with Bearer token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	
	w := httptest.NewRecorder()

	// Apply middleware
	middlewareHandler := middleware.AuthenticationMiddleware()(handler)
	middlewareHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCharacterResolutionMiddleware(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	middleware := NewEnhancedAuthMiddleware(validator, resolver)

	// Test handler that checks for expanded auth context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expandedCtx := GetExpandedAuthContext(r.Context())
		if expandedCtx == nil {
			t.Error("Expected expanded auth context to be set")
			return
		}
		
		if len(expandedCtx.CharacterIDs) != 2 {
			t.Errorf("Expected 2 characters, got %d", len(expandedCtx.CharacterIDs))
		}
		
		if len(expandedCtx.CorporationIDs) != 1 {
			t.Errorf("Expected 1 corporation, got %d", len(expandedCtx.CorporationIDs))
		}
		
		if len(expandedCtx.AllianceIDs) != 1 {
			t.Errorf("Expected 1 alliance, got %d", len(expandedCtx.AllianceIDs))
		}
		
		if expandedCtx.PrimaryCharacter.ID != 123456789 {
			t.Errorf("Expected primary character ID 123456789, got %d", expandedCtx.PrimaryCharacter.ID)
		}
		
		w.WriteHeader(http.StatusOK)
	})

	// Create request with Bearer token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	
	w := httptest.NewRecorder()

	// Apply both middleware layers
	middlewareHandler := middleware.RequireExpandedAuth()(handler)
	middlewareHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestOptionalExpandedAuth(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	middleware := NewEnhancedAuthMiddleware(validator, resolver)

	// Test handler that checks for optional auth context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r.Context())
		expandedCtx := GetExpandedAuthContext(r.Context())
		
		// With valid token, both should be present
		if authCtx == nil {
			t.Error("Expected auth context to be set with valid token")
		}
		
		if expandedCtx == nil {
			t.Error("Expected expanded auth context to be set with valid token")
		}
		
		w.WriteHeader(http.StatusOK)
	})

	// Test with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	
	w := httptest.NewRecorder()
	middlewareHandler := middleware.OptionalExpandedAuth()(handler)
	middlewareHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestOptionalExpandedAuthWithoutToken(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	middleware := NewEnhancedAuthMiddleware(validator, resolver)

	// Test handler that checks for optional auth context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r.Context())
		expandedCtx := GetExpandedAuthContext(r.Context())
		
		// Without token, both should be nil
		if authCtx != nil {
			t.Error("Expected auth context to be nil without token")
		}
		
		if expandedCtx != nil {
			t.Error("Expected expanded auth context to be nil without token")
		}
		
		w.WriteHeader(http.StatusOK)
	})

	// Test without token
	req := httptest.NewRequest("GET", "/test", nil)
	
	w := httptest.NewRecorder()
	middlewareHandler := middleware.OptionalExpandedAuth()(handler)
	middlewareHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}