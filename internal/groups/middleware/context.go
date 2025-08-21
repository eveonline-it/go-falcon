package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go-falcon/internal/auth/models"
	authServices "go-falcon/internal/auth/services"
	"go-falcon/internal/groups/dto"
	"go-falcon/internal/groups/services"
)

// CharacterContext contains resolved character information
type CharacterContext struct {
	UserID        string `json:"user_id"`
	CharacterID   int64  `json:"character_id"`
	CharacterName string `json:"character_name"`
	IsSuperAdmin  bool   `json:"is_super_admin"`
	
	// Corporation and Alliance info (for Phase 2)
	CorporationID   *int64  `json:"corporation_id,omitempty"`
	CorporationName *string `json:"corporation_name,omitempty"`
	AllianceID      *int64  `json:"alliance_id,omitempty"`
	AllianceName    *string `json:"alliance_name,omitempty"`
	
	// Groups this character belongs to (resolved)
	GroupMemberships []string `json:"group_memberships,omitempty"`
}

// CharacterContextMiddleware provides character context resolution
type CharacterContextMiddleware struct {
	authService  *authServices.AuthService
	groupService *services.Service
}

// NewCharacterContextMiddleware creates a new character context middleware
func NewCharacterContextMiddleware(authService *authServices.AuthService, groupService *services.Service) *CharacterContextMiddleware {
	return &CharacterContextMiddleware{
		authService:  authService,
		groupService: groupService,
	}
}

// ResolveCharacterContext extracts and enriches character context from JWT token
func (m *CharacterContextMiddleware) ResolveCharacterContext(ctx context.Context, authHeader, cookieHeader string) (*CharacterContext, error) {
	// Extract JWT token from headers
	token := m.extractToken(authHeader, cookieHeader)
	if token == "" {
		return nil, fmt.Errorf("no authentication token provided")
	}
	
	// If no auth service available, return dummy context (Phase 1 fallback)
	if m.authService == nil {
		return m.getDummyContext(), nil
	}
	
	// Validate JWT token and get basic user info
	var user *models.AuthenticatedUser
	user, err := m.authService.ValidateJWT(token)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT token: %w", err)
	}
	
	// Create character context
	charContext := &CharacterContext{
		UserID:        user.UserID,
		CharacterID:   int64(user.CharacterID),
		CharacterName: user.CharacterName,
		IsSuperAdmin:  false, // Will be determined from profile or group membership
	}
	
	// Load user profile to get additional character information
	if err := m.enrichWithProfile(ctx, charContext); err != nil {
		slog.Warn("Failed to enrich character context with profile", "character_id", charContext.CharacterID, "error", err)
		// Continue without profile enrichment
	}
	
	// Resolve group memberships
	if err := m.enrichWithGroupMemberships(ctx, charContext); err != nil {
		slog.Warn("Failed to enrich character context with group memberships", "character_id", charContext.CharacterID, "error", err)
		// Continue without group membership enrichment
	}
	
	// Check if character is super admin based on group membership
	m.checkSuperAdminStatus(charContext)
	
	return charContext, nil
}

// extractToken extracts JWT token from Authorization header or Cookie
func (m *CharacterContextMiddleware) extractToken(authHeader, cookieHeader string) string {
	// Try Bearer token first
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	
	// Try cookie (falcon_auth_token)
	if cookieHeader != "" {
		// Simple cookie parsing - look for falcon_auth_token
		cookies := strings.Split(cookieHeader, ";")
		for _, cookie := range cookies {
			cookie = strings.TrimSpace(cookie)
			if strings.HasPrefix(cookie, "falcon_auth_token=") {
				return strings.TrimPrefix(cookie, "falcon_auth_token=")
			}
		}
	}
	
	return ""
}

// enrichWithProfile loads user profile and adds corp/alliance info
func (m *CharacterContextMiddleware) enrichWithProfile(ctx context.Context, charContext *CharacterContext) error {
	// Get user profile from auth service
	profile, err := m.authService.GetUserProfile(ctx, int(charContext.CharacterID))
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}
	
	// TODO: Extract corporation and alliance information from profile
	// This will be implemented in Phase 2 when we have ESI integration
	_ = profile
	
	return nil
}

// enrichWithGroupMemberships resolves which groups this character belongs to
func (m *CharacterContextMiddleware) enrichWithGroupMemberships(ctx context.Context, charContext *CharacterContext) error {
	// Get groups for this character
	groups, err := m.groupService.GetCharacterGroups(ctx, &dto.GetCharacterGroupsInput{
		CharacterID: fmt.Sprintf("%d", charContext.CharacterID),
	})
	if err != nil {
		return fmt.Errorf("failed to get character groups: %w", err)
	}
	
	// Extract group names
	groupNames := make([]string, 0, len(groups.Groups))
	for _, group := range groups.Groups {
		groupNames = append(groupNames, group.Name)
	}
	
	charContext.GroupMemberships = groupNames
	return nil
}

// checkSuperAdminStatus determines if character is super admin based on group membership
func (m *CharacterContextMiddleware) checkSuperAdminStatus(charContext *CharacterContext) {
	for _, groupName := range charContext.GroupMemberships {
		if groupName == "Super Administrator" {
			charContext.IsSuperAdmin = true
			break
		}
	}
}

// getDummyContext returns a dummy super admin context for Phase 1 testing
func (m *CharacterContextMiddleware) getDummyContext() *CharacterContext {
	return &CharacterContext{
		UserID:           "00000000-0000-0000-0000-000000000000",
		CharacterID:      99999999,
		CharacterName:    "Test SuperAdmin",
		IsSuperAdmin:     true,
		GroupMemberships: []string{"Super Administrator"},
	}
}

// ResolveCharacterContextWithBypass resolves character context with super admin bypass
func (m *CharacterContextMiddleware) ResolveCharacterContextWithBypass(ctx context.Context, characterID int64, authHeader, cookieHeader string) (*CharacterContext, error) {
	// Check if this is the super admin character (661916654)
	if characterID == 661916654 {
		return &CharacterContext{
			UserID:           "super-admin-user-id",
			CharacterID:      661916654,
			CharacterName:    "Black Dharma",
			IsSuperAdmin:     true,
			GroupMemberships: []string{"Super Administrator"},
		}, nil
	}
	
	// For non-super admin characters, use normal authentication flow
	return m.ResolveCharacterContext(ctx, authHeader, cookieHeader)
}