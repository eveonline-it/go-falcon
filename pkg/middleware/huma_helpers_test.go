package middleware

import (
	"context"
	"testing"
)

func TestHumaAuthHelper_ValidateAuthFromHeaders(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	helper := NewHumaAuthHelper(validator, resolver)

	tests := []struct {
		name           string
		authHeader     string
		cookieHeader   string
		expectedUserID string
		expectError    bool
	}{
		{
			name:           "Valid bearer token",
			authHeader:     "Bearer valid-token",
			cookieHeader:   "",
			expectedUserID: "test-user-id",
			expectError:    false,
		},
		{
			name:           "Valid cookie",
			authHeader:     "",
			cookieHeader:   "falcon_auth_token=valid-token; path=/",
			expectedUserID: "test-user-id",
			expectError:    false,
		},
		{
			name:         "No authentication",
			authHeader:   "",
			cookieHeader: "",
			expectError:  true,
		},
		{
			name:         "Invalid token",
			authHeader:   "Bearer invalid-token",
			cookieHeader: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := helper.ValidateAuthFromHeaders(tt.authHeader, tt.cookieHeader)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if user == nil {
				t.Error("Expected user but got nil")
				return
			}

			if user.UserID != tt.expectedUserID {
				t.Errorf("Expected user ID %s, got %s", tt.expectedUserID, user.UserID)
			}
		})
	}
}

func TestHumaAuthHelper_ValidateOptionalAuthFromHeaders(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	helper := NewHumaAuthHelper(validator, resolver)

	// Test with valid token
	user := helper.ValidateOptionalAuthFromHeaders("Bearer valid-token", "")
	if user == nil {
		t.Error("Expected user with valid token")
	}

	// Test without token (should not error)
	user = helper.ValidateOptionalAuthFromHeaders("", "")
	if user != nil {
		t.Error("Expected nil user without token")
	}

	// Test with invalid token (should not error)
	user = helper.ValidateOptionalAuthFromHeaders("Bearer invalid-token", "")
	if user != nil {
		t.Error("Expected nil user with invalid token")
	}
}

func TestHumaAuthHelper_ValidateExpandedAuthFromHeaders(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	helper := NewHumaAuthHelper(validator, resolver)

	ctx := context.Background()

	// Test with valid authentication
	expandedCtx, err := helper.ValidateExpandedAuthFromHeaders(ctx, "Bearer valid-token", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if expandedCtx == nil {
		t.Fatal("Expected expanded context but got nil")
	}

	if expandedCtx.UserID != "test-user-id" {
		t.Errorf("Expected user ID 'test-user-id', got '%s'", expandedCtx.UserID)
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

	// Test without authentication
	_, err = helper.ValidateExpandedAuthFromHeaders(ctx, "", "")
	if err == nil {
		t.Error("Expected error without authentication")
	}
}

func TestHumaAuthHelper_RequireAuth(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	helper := NewHumaAuthHelper(validator, resolver)

	ctx := context.Background()

	// Test with valid input
	input := &HumaAuthInput{
		Authorization: "Bearer valid-token",
		Cookie:        "",
	}

	user, err := helper.RequireAuth(ctx, input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if user.UserID != "test-user-id" {
		t.Errorf("Expected user ID 'test-user-id', got '%s'", user.UserID)
	}

	// Test without authentication
	input = &HumaAuthInput{
		Authorization: "",
		Cookie:        "",
	}

	_, err = helper.RequireAuth(ctx, input)
	if err == nil {
		t.Error("Expected error without authentication")
	}
}

func TestHumaAuthHelper_RequireExpandedAuth(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	helper := NewHumaAuthHelper(validator, resolver)

	ctx := context.Background()

	// Test with valid input
	input := &HumaAuthInput{
		Authorization: "Bearer valid-token",
		Cookie:        "",
	}

	expandedCtx, err := helper.RequireExpandedAuth(ctx, input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if expandedCtx.UserID != "test-user-id" {
		t.Errorf("Expected user ID 'test-user-id', got '%s'", expandedCtx.UserID)
	}

	if expandedCtx.RequestType != "bearer" {
		t.Errorf("Expected request type 'bearer', got '%s'", expandedCtx.RequestType)
	}
}

func TestHumaAuthHelper_OptionalAuth(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	helper := NewHumaAuthHelper(validator, resolver)

	ctx := context.Background()

	// Test with valid authentication
	input := &HumaAuthInput{
		Authorization: "Bearer valid-token",
		Cookie:        "",
	}

	user := helper.OptionalAuth(ctx, input)
	if user == nil {
		t.Error("Expected user with valid authentication")
	}

	// Test without authentication (should not error)
	input = &HumaAuthInput{
		Authorization: "",
		Cookie:        "",
	}

	user = helper.OptionalAuth(ctx, input)
	if user != nil {
		t.Error("Expected nil user without authentication")
	}
}

func TestHumaAuthHelper_OptionalExpandedAuth(t *testing.T) {
	validator := &mockJWTValidator{}
	resolver := &mockUserCharacterResolver{}
	helper := NewHumaAuthHelper(validator, resolver)

	ctx := context.Background()

	// Test with valid authentication
	input := &HumaAuthInput{
		Authorization: "Bearer valid-token",
		Cookie:        "",
	}

	expandedCtx := helper.OptionalExpandedAuth(ctx, input)
	if expandedCtx == nil {
		t.Error("Expected expanded context with valid authentication")
	}

	// Test without authentication (should not error)
	input = &HumaAuthInput{
		Authorization: "",
		Cookie:        "",
	}

	expandedCtx = helper.OptionalExpandedAuth(ctx, input)
	if expandedCtx != nil {
		t.Error("Expected nil expanded context without authentication")
	}
}

func TestCreateSubjectsForCASBIN(t *testing.T) {
	// Test with nil context
	subjects := CreateSubjectsForCASBIN(nil)
	if len(subjects) != 0 {
		t.Errorf("Expected empty subjects for nil context, got %d", len(subjects))
	}

	// Test with valid expanded context
	expandedCtx := &ExpandedAuthContext{
		AuthContext: &AuthContext{
			UserID: "test-user-id",
		},
		CharacterIDs:   []int64{123456789, 987654321},
		CorporationIDs: []int64{98000001},
		AllianceIDs:    []int64{99000001},
		PrimaryCharacter: struct {
			ID            int64  `json:"id"`
			Name          string `json:"name"`
			CorporationID int64  `json:"corporation_id"`
			AllianceID    int64  `json:"alliance_id,omitempty"`
		}{
			ID: 123456789,
		},
	}

	subjects = CreateSubjectsForCASBIN(expandedCtx)

	expectedSubjects := []string{
		"user:test-user-id",
		"character:123456789",
		"character:123456789",
		"character:987654321",
		"corporation:98000001",
		"alliance:99000001",
	}

	if len(subjects) != len(expectedSubjects) {
		t.Errorf("Expected %d subjects, got %d", len(expectedSubjects), len(subjects))
	}

	// Check if all expected subjects are present
	subjectMap := make(map[string]bool)
	for _, subject := range subjects {
		subjectMap[subject] = true
	}

	for _, expected := range expectedSubjects {
		if !subjectMap[expected] {
			t.Errorf("Expected subject %s not found", expected)
		}
	}
}

func TestHumaAuthHelper_ExtractTokenFromHeaders(t *testing.T) {
	helper := &HumaAuthHelper{}

	tests := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{
			name:       "Valid bearer token",
			authHeader: "Bearer abc123",
			expected:   "abc123",
		},
		{
			name:       "Bearer with extra spaces",
			authHeader: "Bearer   xyz789",
			expected:   "  xyz789", // Should preserve spaces after "Bearer "
		},
		{
			name:       "No bearer prefix",
			authHeader: "abc123",
			expected:   "",
		},
		{
			name:       "Empty header",
			authHeader: "",
			expected:   "",
		},
		{
			name:       "Just Bearer",
			authHeader: "Bearer",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.ExtractTokenFromHeaders(tt.authHeader)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHumaAuthHelper_getRequestType(t *testing.T) {
	helper := &HumaAuthHelper{}

	tests := []struct {
		name         string
		authHeader   string
		cookieHeader string
		expected     string
	}{
		{
			name:         "Bearer token",
			authHeader:   "Bearer abc123",
			cookieHeader: "",
			expected:     "bearer",
		},
		{
			name:         "Cookie token",
			authHeader:   "",
			cookieHeader: "falcon_auth_token=abc123; path=/",
			expected:     "cookie",
		},
		{
			name:         "No token",
			authHeader:   "",
			cookieHeader: "",
			expected:     "unknown",
		},
		{
			name:         "Both tokens (bearer wins)",
			authHeader:   "Bearer abc123",
			cookieHeader: "falcon_auth_token=xyz789; path=/",
			expected:     "bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.getRequestType(tt.authHeader, tt.cookieHeader)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}