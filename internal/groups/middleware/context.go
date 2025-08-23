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
	
	// EVE Online scopes
	Scopes string `json:"scopes,omitempty"`
	
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
	
	// Auth service is required for character context resolution
	if m.authService == nil {
		return nil, fmt.Errorf("auth service not available")
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
	
	// Extract corporation and alliance information from profile
	if profile.CorporationID > 0 {
		corpID := int64(profile.CorporationID)
		charContext.CorporationID = &corpID
		
		if profile.CorporationName != "" {
			charContext.CorporationName = &profile.CorporationName
		}
	}
	
	if profile.AllianceID > 0 {
		allianceID := int64(profile.AllianceID)
		charContext.AllianceID = &allianceID
		
		if profile.AllianceName != "" {
			charContext.AllianceName = &profile.AllianceName
		}
	}
	
	// Add scopes information for group assignment
	charContext.Scopes = profile.Scopes
	
	slog.Debug("[CharacterContext] Profile enrichment completed", 
		"character_id", charContext.CharacterID,
		"corporation_id", charContext.CorporationID,
		"corporation_name", charContext.CorporationName,
		"alliance_id", charContext.AllianceID,
		"alliance_name", charContext.AllianceName)
	
	return nil
}

// enrichWithGroupMemberships resolves which groups this character belongs to
func (m *CharacterContextMiddleware) enrichWithGroupMemberships(ctx context.Context, charContext *CharacterContext) error {
	// Auto-join character to enabled corporation/alliance groups only
	if err := m.groupService.AutoJoinCharacterToEnabledGroups(ctx, charContext.CharacterID, charContext.CorporationID, charContext.AllianceID, charContext.Scopes); err != nil {
		slog.Warn("[CharacterContext] Failed to auto-join character to enabled groups", 
			"character_id", charContext.CharacterID, 
			"error", err)
		// Continue without auto-join - don't fail the entire request
	}
	
	// Get groups for this character
	groups, err := m.groupService.GetCharacterGroups(ctx, &dto.GetCharacterGroupsInput{
		CharacterID: fmt.Sprintf("%d", charContext.CharacterID),
	})
	if err != nil {
		return fmt.Errorf("failed to get character groups: %w", err)
	}
	
	// Extract group names
	groupNames := make([]string, 0, len(groups.Body.Groups))
	for _, group := range groups.Body.Groups {
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


// ResolveCharacterContextWithBypass resolves character context (bypass removed - now uses groups)
func (m *CharacterContextMiddleware) ResolveCharacterContextWithBypass(ctx context.Context, characterID int64, authHeader, cookieHeader string) (*CharacterContext, error) {
	// Use normal authentication flow - first user gets auto-assigned to super_admin group
	return m.ResolveCharacterContext(ctx, authHeader, cookieHeader)
}