package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
	
	// Multi-character support - all characters under the same user_id
	AllUserCharacterIDs []int64  `json:"all_user_character_ids,omitempty"`
	AllCorporationIDs   []int64  `json:"all_corporation_ids,omitempty"`
	AllAllianceIDs      []int64  `json:"all_alliance_ids,omitempty"`
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
	user, err := m.authService.ValidateJWT(token)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT token: %w", err)
	}
	
	// Create character context with basic information
	charContext := &CharacterContext{
		UserID:              user.UserID,
		CharacterID:         int64(user.CharacterID),
		CharacterName:       user.CharacterName,
		IsSuperAdmin:        false,
		AllUserCharacterIDs: []int64{int64(user.CharacterID)},
		GroupMemberships:    []string{},
	}
	
	// Note: Full profile enrichment, group membership resolution, and multi-character
	// support are disabled for now to prevent potential MongoDB operation hangs.
	// This provides basic authentication without complex database operations.
	
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

// enrichWithAllUserCharacters fetches all characters for this user
func (m *CharacterContextMiddleware) enrichWithAllUserCharacters(ctx context.Context, charContext *CharacterContext) error {
	// Get all characters for this user
	allProfiles, err := m.authService.GetAllCharactersByUserID(ctx, charContext.UserID)
	if err != nil {
		return fmt.Errorf("failed to get all user characters: %w", err)
	}
	
	// Collect unique character, corporation, and alliance IDs
	characterIDSet := make(map[int64]bool)
	corpIDSet := make(map[int64]bool)
	allianceIDSet := make(map[int64]bool)
	
	for _, profile := range allProfiles {
		characterIDSet[int64(profile.CharacterID)] = true
		
		if profile.CorporationID > 0 {
			corpIDSet[int64(profile.CorporationID)] = true
		}
		
		if profile.AllianceID > 0 {
			allianceIDSet[int64(profile.AllianceID)] = true
		}
	}
	
	// Convert sets to slices
	charContext.AllUserCharacterIDs = make([]int64, 0, len(characterIDSet))
	for id := range characterIDSet {
		charContext.AllUserCharacterIDs = append(charContext.AllUserCharacterIDs, id)
	}
	
	charContext.AllCorporationIDs = make([]int64, 0, len(corpIDSet))
	for id := range corpIDSet {
		charContext.AllCorporationIDs = append(charContext.AllCorporationIDs, id)
	}
	
	charContext.AllAllianceIDs = make([]int64, 0, len(allianceIDSet))
	for id := range allianceIDSet {
		charContext.AllAllianceIDs = append(charContext.AllAllianceIDs, id)
	}
	
	slog.Debug("[CharacterContext] Enriched with all user characters",
		"user_id", charContext.UserID,
		"total_characters", len(charContext.AllUserCharacterIDs),
		"character_ids", charContext.AllUserCharacterIDs,
		"corporation_ids", charContext.AllCorporationIDs,
		"alliance_ids", charContext.AllAllianceIDs)
	
	return nil
}

// enrichWithGroupMemberships resolves which groups this character belongs to
func (m *CharacterContextMiddleware) enrichWithGroupMemberships(ctx context.Context, charContext *CharacterContext) error {
	// Auto-join character to enabled corporation/alliance groups only (with circuit breaker pattern)
	autoJoinDone := make(chan error, 1)
	go func() {
		autoJoinCtx, cancelAutoJoin := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancelAutoJoin()
		autoJoinDone <- m.groupService.AutoJoinCharacterToEnabledGroups(autoJoinCtx, charContext.CharacterID, charContext.CorporationID, charContext.AllianceID, charContext.Scopes)
	}()
	
	select {
	case err := <-autoJoinDone:
		if err != nil {
			slog.Warn("[CharacterContext] Failed to auto-join character to enabled groups", 
				"character_id", charContext.CharacterID, 
				"error", err)
		}
	case <-time.After(250*time.Millisecond):
		slog.Warn("[CharacterContext] Auto-join operation timed out, continuing without group assignment", 
			"character_id", charContext.CharacterID)
	}
	
	// Collect groups from ALL characters under this user
	groupNameSet := make(map[string]bool)
	
	// Get groups for all user's characters (with circuit breaker pattern)
	for _, characterID := range charContext.AllUserCharacterIDs {
		// Use circuit breaker pattern for each character to prevent hanging
		groupsDone := make(chan *dto.CharacterGroupsOutput, 1)
		errChan := make(chan error, 1)
		
		go func(charID int64) {
			charGroupCtx, cancelCharGroup := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancelCharGroup()
			groups, err := m.groupService.GetCharacterGroups(charGroupCtx, &dto.GetCharacterGroupsInput{
				CharacterID: fmt.Sprintf("%d", charID),
			})
			if err != nil {
				errChan <- err
			} else {
				groupsDone <- groups
			}
		}(characterID)
		
		select {
		case groups := <-groupsDone:
			// Add all groups to the set
			for _, group := range groups.Body.Groups {
				groupNameSet[group.Name] = true
			}
		case err := <-errChan:
			slog.Warn("[CharacterContext] Failed to get groups for character",
				"character_id", characterID,
				"error", err)
		case <-time.After(200*time.Millisecond):
			slog.Warn("[CharacterContext] Get groups operation timed out for character",
				"character_id", characterID)
		}
	}
	
	// Convert set to slice
	groupNames := make([]string, 0, len(groupNameSet))
	for name := range groupNameSet {
		groupNames = append(groupNames, name)
	}
	
	slog.Debug("[CharacterContext] Retrieved aggregate group memberships",
		"user_id", charContext.UserID,
		"current_character_id", charContext.CharacterID,
		"all_character_ids", charContext.AllUserCharacterIDs,
		"groups", groupNames,
		"group_count", len(groupNames))
	
	charContext.GroupMemberships = groupNames
	return nil
}

// checkSuperAdminStatus determines if character is super admin based on group membership
func (m *CharacterContextMiddleware) checkSuperAdminStatus(charContext *CharacterContext) {
	for _, groupName := range charContext.GroupMemberships {
		if groupName == "Super Administrator" {
			charContext.IsSuperAdmin = true
			slog.Info("[CharacterContext] User has super admin access via multi-character permissions",
				"user_id", charContext.UserID,
				"current_character_id", charContext.CharacterID,
				"all_character_ids", charContext.AllUserCharacterIDs)
			break
		}
	}
}


// ResolveCharacterContextWithBypass resolves character context (bypass removed - now uses groups)
func (m *CharacterContextMiddleware) ResolveCharacterContextWithBypass(ctx context.Context, characterID int64, authHeader, cookieHeader string) (*CharacterContext, error) {
	// Use normal authentication flow - first user gets auto-assigned to super_admin group
	return m.ResolveCharacterContext(ctx, authHeader, cookieHeader)
}