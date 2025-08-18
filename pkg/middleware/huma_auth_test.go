package middleware

import (
	"testing"

	"go-falcon/internal/auth/models"
)

// MockJWTValidator implements JWTValidator for testing
type MockJWTValidator struct {
	ValidUser   *models.AuthenticatedUser
	ShouldError bool
}

func (m *MockJWTValidator) ValidateJWT(token string) (*models.AuthenticatedUser, error) {
	if m.ShouldError {
		return nil, &AuthError{message: "invalid token"}
	}
	if token == "valid_token" {
		return m.ValidUser, nil
	}
	return nil, &AuthError{message: "invalid token"}
}

func TestHumaAuthMiddleware(t *testing.T) {
	// Create test user
	testUser := &models.AuthenticatedUser{
		CharacterID:   123456789,
		CharacterName: "Test Character",
		Scopes:        "esi-characters.read_contacts.v1 publicData",
	}

	// Create middleware with mock validator
	mockValidator := &MockJWTValidator{
		ValidUser: testUser,
	}
	middleware := NewHumaAuthMiddleware(mockValidator)

	t.Run("ValidateAuthFromHeaders_ValidBearerToken", func(t *testing.T) {
		authHeader := "Bearer valid_token"
		cookieHeader := ""

		user, err := middleware.ValidateAuthFromHeaders(authHeader, cookieHeader)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if user.CharacterID != testUser.CharacterID {
			t.Errorf("Expected character ID %d, got %d", testUser.CharacterID, user.CharacterID)
		}
	})

	t.Run("ValidateAuthFromHeaders_ValidCookie", func(t *testing.T) {
		authHeader := ""
		cookieHeader := "falcon_auth_token=valid_token; other_cookie=value"

		user, err := middleware.ValidateAuthFromHeaders(authHeader, cookieHeader)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if user.CharacterName != testUser.CharacterName {
			t.Errorf("Expected character name %s, got %s", testUser.CharacterName, user.CharacterName)
		}
	})

	t.Run("ValidateAuthFromHeaders_InvalidToken", func(t *testing.T) {
		authHeader := "Bearer invalid_token"
		cookieHeader := ""

		user, err := middleware.ValidateAuthFromHeaders(authHeader, cookieHeader)
		if err == nil {
			t.Fatal("Expected error for invalid token")
		}

		if user != nil {
			t.Error("Expected nil user for invalid token")
		}
	})

	t.Run("ValidateAuthFromHeaders_NoToken", func(t *testing.T) {
		authHeader := ""
		cookieHeader := ""

		user, err := middleware.ValidateAuthFromHeaders(authHeader, cookieHeader)
		if err == nil {
			t.Fatal("Expected error when no token provided")
		}

		if user != nil {
			t.Error("Expected nil user when no token provided")
		}
	})

	t.Run("ValidateScopesFromHeaders_ValidScopes", func(t *testing.T) {
		authHeader := "Bearer valid_token"
		cookieHeader := ""
		requiredScopes := []string{"esi-characters.read_contacts.v1", "publicData"}

		user, err := middleware.ValidateScopesFromHeaders(authHeader, cookieHeader, requiredScopes...)
		if err != nil {
			t.Fatalf("Expected no error for valid scopes, got %v", err)
		}

		if user.CharacterID != testUser.CharacterID {
			t.Errorf("Expected character ID %d, got %d", testUser.CharacterID, user.CharacterID)
		}
	})

	t.Run("ValidateScopesFromHeaders_MissingScopes", func(t *testing.T) {
		authHeader := "Bearer valid_token"
		cookieHeader := ""
		requiredScopes := []string{"esi-characters.read_contacts.v1", "missing_scope"}

		user, err := middleware.ValidateScopesFromHeaders(authHeader, cookieHeader, requiredScopes...)
		if err == nil {
			t.Fatal("Expected error for missing scopes")
		}

		if user != nil {
			t.Error("Expected nil user for missing scopes")
		}
	})

	t.Run("ValidateOptionalAuthFromHeaders_ValidToken", func(t *testing.T) {
		authHeader := "Bearer valid_token"
		cookieHeader := ""

		user := middleware.ValidateOptionalAuthFromHeaders(authHeader, cookieHeader)
		if user == nil {
			t.Fatal("Expected user for valid optional auth")
		}

		if user.CharacterID != testUser.CharacterID {
			t.Errorf("Expected character ID %d, got %d", testUser.CharacterID, user.CharacterID)
		}
	})

	t.Run("ValidateOptionalAuthFromHeaders_InvalidToken", func(t *testing.T) {
		authHeader := "Bearer invalid_token"
		cookieHeader := ""

		user := middleware.ValidateOptionalAuthFromHeaders(authHeader, cookieHeader)
		if user != nil {
			t.Error("Expected nil user for invalid optional auth")
		}
	})
}

func TestTokenExtraction(t *testing.T) {
	middleware := NewHumaAuthMiddleware(&MockJWTValidator{})

	t.Run("ExtractTokenFromHeaders_BearerToken", func(t *testing.T) {
		authHeader := "Bearer test_token_123"
		token := middleware.ExtractTokenFromHeaders(authHeader)

		if token != "test_token_123" {
			t.Errorf("Expected token 'test_token_123', got '%s'", token)
		}
	})

	t.Run("ExtractTokenFromHeaders_NoBearerPrefix", func(t *testing.T) {
		authHeader := "test_token_123"
		token := middleware.ExtractTokenFromHeaders(authHeader)

		if token != "" {
			t.Errorf("Expected empty token, got '%s'", token)
		}
	})

	t.Run("ExtractTokenFromCookie_ValidCookie", func(t *testing.T) {
		cookieHeader := "session_id=abc123; falcon_auth_token=test_token_456; other_cookie=xyz"
		token := middleware.ExtractTokenFromCookie(cookieHeader)

		if token != "test_token_456" {
			t.Errorf("Expected token 'test_token_456', got '%s'", token)
		}
	})

	t.Run("ExtractTokenFromCookie_NoCookie", func(t *testing.T) {
		cookieHeader := "session_id=abc123; other_cookie=xyz"
		token := middleware.ExtractTokenFromCookie(cookieHeader)

		if token != "" {
			t.Errorf("Expected empty token, got '%s'", token)
		}
	})
}

func TestCookieHeaderGeneration(t *testing.T) {
	t.Run("CreateAuthCookieHeader", func(t *testing.T) {
		token := "test_jwt_token_123"
		cookieHeader := CreateAuthCookieHeader(token)

		expected := "falcon_auth_token=test_jwt_token_123; Path=/; Domain=.eveonline.it; Max-Age=86400; HttpOnly; Secure; SameSite=Lax"
		if cookieHeader != expected {
			t.Errorf("Expected cookie header '%s', got '%s'", expected, cookieHeader)
		}
	})

	t.Run("CreateClearCookieHeader", func(t *testing.T) {
		cookieHeader := CreateClearCookieHeader()

		expected := "falcon_auth_token=; Path=/; Domain=.eveonline.it; Max-Age=0; HttpOnly; Secure; SameSite=Lax"
		if cookieHeader != expected {
			t.Errorf("Expected clear cookie header '%s', got '%s'", expected, cookieHeader)
		}
	})

	t.Run("CookieRoundTrip", func(t *testing.T) {
		// Test that a cookie created with CreateAuthCookieHeader can be extracted
		token := "round_trip_token_456"
		cookieHeader := CreateAuthCookieHeader(token)
		
		// Simulate the cookie being sent back in a request
		middleware := NewHumaAuthMiddleware(&MockJWTValidator{})
		extractedToken := middleware.ExtractTokenFromCookie(cookieHeader)
		
		if extractedToken != token {
			t.Errorf("Expected extracted token '%s', got '%s'", token, extractedToken)
		}
	})
}