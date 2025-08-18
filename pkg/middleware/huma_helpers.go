package middleware

import (
	"context"
	"fmt"
	"strings"

	"go-falcon/internal/auth/models"

	"github.com/danielgtaylor/huma/v2"
)

// HumaAuthHelper provides HUMA-compatible authentication helpers following the codebase pattern
type HumaAuthHelper struct {
	jwtValidator      JWTValidator
	characterResolver UserCharacterResolver
}

// NewHumaAuthHelper creates a new HUMA authentication helper
func NewHumaAuthHelper(validator JWTValidator, resolver UserCharacterResolver) *HumaAuthHelper {
	return &HumaAuthHelper{
		jwtValidator:      validator,
		characterResolver: resolver,
	}
}

// ValidateAuthFromHeaders validates authentication from request headers (HUMA pattern)
func (h *HumaAuthHelper) ValidateAuthFromHeaders(authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	fmt.Printf("[DEBUG] HumaAuthHelper.ValidateAuthFromHeaders: authHeader=%q cookieHeader=%q\n", authHeader, cookieHeader)
	// Try to get token from Authorization header first
	token := h.ExtractTokenFromHeaders(authHeader)
	
	// If not found, try cookie
	if token == "" && cookieHeader != "" {
		token = h.ExtractTokenFromCookie(cookieHeader)
		fmt.Printf("[DEBUG] HumaAuthHelper: Extracted token from cookie: %q\n", token)
	} else if token != "" {
		fmt.Printf("[DEBUG] HumaAuthHelper: Extracted token from header: %q\n", token)
	}

	if token == "" {
		fmt.Printf("[DEBUG] HumaAuthHelper: No token found\n")
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	user, err := h.ValidateToken(token)
	if err != nil {
		fmt.Printf("[DEBUG] HumaAuthHelper: Token validation failed: %v\n", err)
		return nil, huma.Error401Unauthorized("Invalid authentication token", err)
	}

	fmt.Printf("[DEBUG] HumaAuthHelper: Token validation successful, userID=%s\n", user.UserID)
	return user, nil
}

// ValidateOptionalAuthFromHeaders validates optional authentication from request headers
func (h *HumaAuthHelper) ValidateOptionalAuthFromHeaders(authHeader, cookieHeader string) *models.AuthenticatedUser {
	user, _ := h.ValidateAuthFromHeaders(authHeader, cookieHeader)
	return user
}

// ValidateExpandedAuthFromHeaders validates authentication and resolves character context
func (h *HumaAuthHelper) ValidateExpandedAuthFromHeaders(ctx context.Context, authHeader, cookieHeader string) (*ExpandedAuthContext, error) {
	// First validate authentication
	user, err := h.ValidateAuthFromHeaders(authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}

	// Create base auth context
	authCtx := &AuthContext{
		UserID:          user.UserID,
		PrimaryCharID:   int64(user.CharacterID),
		RequestType:     h.getRequestType(authHeader, cookieHeader),
		IsAuthenticated: true,
	}

	// Resolve characters
	expandedCtx, err := h.resolveUserCharacters(ctx, authCtx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to resolve user characters", err)
	}

	return expandedCtx, nil
}

// ValidateOptionalExpandedAuthFromHeaders validates optional authentication and character resolution
func (h *HumaAuthHelper) ValidateOptionalExpandedAuthFromHeaders(ctx context.Context, authHeader, cookieHeader string) *ExpandedAuthContext {
	expandedCtx, _ := h.ValidateExpandedAuthFromHeaders(ctx, authHeader, cookieHeader)
	return expandedCtx
}

// ExtractTokenFromHeaders extracts JWT token from Authorization header string
func (h *HumaAuthHelper) ExtractTokenFromHeaders(authHeader string) string {
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

// ExtractTokenFromCookie extracts JWT token from cookie header string
func (h *HumaAuthHelper) ExtractTokenFromCookie(cookieHeader string) string {
	// Parse cookie header to find falcon_auth_token
	cookies := strings.Split(cookieHeader, ";")
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		if strings.HasPrefix(cookie, "falcon_auth_token=") {
			return strings.TrimPrefix(cookie, "falcon_auth_token=")
		}
	}
	return ""
}

// ValidateToken validates a JWT token string and returns the authenticated user
func (h *HumaAuthHelper) ValidateToken(token string) (*models.AuthenticatedUser, error) {
	if token == "" {
		return nil, huma.Error401Unauthorized("No authentication token provided")
	}

	// Validate JWT using the injected validator
	user, err := h.jwtValidator.ValidateJWT(token)
	if err != nil {
		return nil, huma.Error401Unauthorized("Invalid authentication token", err)
	}

	return user, nil
}

// getRequestType determines if authentication came from bearer token or cookie
func (h *HumaAuthHelper) getRequestType(authHeader, cookieHeader string) string {
	if h.ExtractTokenFromHeaders(authHeader) != "" {
		return "bearer"
	}
	if h.ExtractTokenFromCookie(cookieHeader) != "" {
		return "cookie"
	}
	return "unknown"
}

// resolveUserCharacters resolves all characters for a user and creates expanded context
func (h *HumaAuthHelper) resolveUserCharacters(ctx context.Context, authCtx *AuthContext) (*ExpandedAuthContext, error) {
	// Get user with all characters
	user, err := h.characterResolver.GetUserWithCharacters(ctx, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user characters: %w", err)
	}
	
	var characterIDs []int64
	var corporationIDs []int64
	var allianceIDs []int64
	
	// Extract unique IDs
	corpMap := make(map[int64]bool)
	allianceMap := make(map[int64]bool)
	
	for _, char := range user.Characters {
		characterIDs = append(characterIDs, char.CharacterID)
		
		if !corpMap[char.CorporationID] {
			corporationIDs = append(corporationIDs, char.CorporationID)
			corpMap[char.CorporationID] = true
		}
		
		if char.AllianceID > 0 && !allianceMap[char.AllianceID] {
			allianceIDs = append(allianceIDs, char.AllianceID)
			allianceMap[char.AllianceID] = true
		}
	}
	
	// Find primary character details
	var primaryChar UserCharacter
	for _, char := range user.Characters {
		if char.CharacterID == authCtx.PrimaryCharID {
			primaryChar = char
			break
		}
	}
	
	return &ExpandedAuthContext{
		AuthContext:    authCtx,
		CharacterIDs:   characterIDs,
		CorporationIDs: corporationIDs,
		AllianceIDs:    allianceIDs,
		PrimaryCharacter: struct {
			ID            int64  `json:"id"`
			Name          string `json:"name"`
			CorporationID int64  `json:"corporation_id"`
			AllianceID    int64  `json:"alliance_id,omitempty"`
		}{
			ID:            primaryChar.CharacterID,
			Name:          primaryChar.Name,
			CorporationID: primaryChar.CorporationID,
			AllianceID:    primaryChar.AllianceID,
		},
		Roles:       []string{}, // To be populated by CASBIN
		Permissions: []string{}, // To be populated by CASBIN
	}, nil
}

// HUMA Input Structures for consistent authentication across modules

// HumaAuthInput provides standard authentication headers for HUMA operations
type HumaAuthInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication" example:"Bearer eyJhbGciOiJIUzI1NiIs..."`
	Cookie        string `header:"Cookie" doc:"Authentication cookie" example:"falcon_auth_token=eyJhbGciOiJIUzI1NiIs..."`
}

// HumaAuthRequiredInput provides required authentication headers
type HumaAuthRequiredInput struct {
	Authorization string `header:"Authorization" required:"true" doc:"Bearer token for authentication" example:"Bearer eyJhbGciOiJIUzI1NiIs..."`
}

// Convenience functions for HUMA handlers

// RequireAuth validates authentication in a HUMA handler
func (h *HumaAuthHelper) RequireAuth(ctx context.Context, input *HumaAuthInput) (*models.AuthenticatedUser, error) {
	return h.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
}

// RequireExpandedAuth validates authentication and resolves characters in a HUMA handler
func (h *HumaAuthHelper) RequireExpandedAuth(ctx context.Context, input *HumaAuthInput) (*ExpandedAuthContext, error) {
	return h.ValidateExpandedAuthFromHeaders(ctx, input.Authorization, input.Cookie)
}

// OptionalAuth provides optional authentication in a HUMA handler
func (h *HumaAuthHelper) OptionalAuth(ctx context.Context, input *HumaAuthInput) *models.AuthenticatedUser {
	return h.ValidateOptionalAuthFromHeaders(input.Authorization, input.Cookie)
}

// OptionalExpandedAuth provides optional authentication and character resolution in a HUMA handler
func (h *HumaAuthHelper) OptionalExpandedAuth(ctx context.Context, input *HumaAuthInput) *ExpandedAuthContext {
	return h.ValidateOptionalExpandedAuthFromHeaders(ctx, input.Authorization, input.Cookie)
}

// CreateSubjectsForCASBIN creates CASBIN subject list from expanded context
func CreateSubjectsForCASBIN(expandedCtx *ExpandedAuthContext) []string {
	if expandedCtx == nil {
		return []string{}
	}

	var subjects []string
	
	// Add user subject
	subjects = append(subjects, fmt.Sprintf("user:%s", expandedCtx.UserID))
	
	// Add primary character subject
	subjects = append(subjects, fmt.Sprintf("character:%d", expandedCtx.PrimaryCharacter.ID))
	
	// Add all character subjects
	for _, charID := range expandedCtx.CharacterIDs {
		subjects = append(subjects, fmt.Sprintf("character:%d", charID))
	}
	
	// Add corporation subjects
	for _, corpID := range expandedCtx.CorporationIDs {
		subjects = append(subjects, fmt.Sprintf("corporation:%d", corpID))
	}
	
	// Add alliance subjects
	for _, allianceID := range expandedCtx.AllianceIDs {
		subjects = append(subjects, fmt.Sprintf("alliance:%d", allianceID))
	}
	
	return subjects
}