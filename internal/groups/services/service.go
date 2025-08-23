package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-falcon/internal/groups/dto"
	"go-falcon/internal/groups/models"
	siteSettingsModels "go-falcon/internal/site_settings/models"
	"go-falcon/pkg/database"
	"go-falcon/pkg/permissions"
)

// Service handles business logic for groups
type Service struct {
	repo                *Repository
	siteSettingsService SiteSettingsServiceInterface
	permissionManager   *permissions.PermissionManager
}

// Interface to access site settings without circular dependency
type SiteSettingsServiceInterface interface {
	GetEnabledCorporations(ctx context.Context) ([]siteSettingsModels.ManagedCorporation, error)
	GetEnabledAlliances(ctx context.Context) ([]siteSettingsModels.ManagedAlliance, error)
}

// NewService creates a new service instance
func NewService(db *database.MongoDB, siteSettingsService SiteSettingsServiceInterface) *Service {
	return &Service{
		repo:                NewRepository(db),
		siteSettingsService: siteSettingsService,
		permissionManager:   nil, // Will be set later
	}
}

// SetPermissionManager sets the permission manager for the service
func (s *Service) SetPermissionManager(permissionManager *permissions.PermissionManager) {
	s.permissionManager = permissionManager
}

// InitializeService sets up the service (creates indexes and system groups)
func (s *Service) InitializeService(ctx context.Context) error {
	// Create database indexes
	if err := s.repo.CreateIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Create system groups if they don't exist
	if err := s.createSystemGroups(ctx); err != nil {
		return fmt.Errorf("failed to create system groups: %w", err)
	}

	return nil
}

// createSystemGroups creates the predefined system groups
func (s *Service) createSystemGroups(ctx context.Context) error {
	for systemName, name := range models.SystemGroups {
		// Check if system group already exists
		existing, err := s.repo.GetGroupBySystemName(ctx, systemName)
		if err != nil {
			return fmt.Errorf("failed to check system group %s: %w", systemName, err)
		}

		if existing == nil {
			// Create system group
			group := &models.Group{
				Name:       name,
				Type:       models.GroupTypeSystem,
				SystemName: &systemName,
				IsActive:   true,
			}

			if err := s.repo.CreateGroup(ctx, group); err != nil {
				return fmt.Errorf("failed to create system group %s: %w", systemName, err)
			}
		}
	}

	return nil
}

// CreateGroup creates a new custom group
func (s *Service) CreateGroup(ctx context.Context, input *dto.CreateGroupInput, createdBy int64) (*dto.GroupOutput, error) {
	// Validate group type (only custom allowed for manual creation)
	if input.Body.Type != string(models.GroupTypeCustom) {
		return nil, fmt.Errorf("only custom groups can be created manually")
	}

	// Check if group name already exists
	existing, err := s.repo.GetGroupByName(ctx, input.Body.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing group: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("group with name '%s' already exists", input.Body.Name)
	}

	// Create group
	group := &models.Group{
		Name:        input.Body.Name,
		Description: input.Body.Description,
		Type:        models.GroupType(input.Body.Type),
		IsActive:    true,
	}

	if err := s.repo.CreateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return s.modelToOutput(group, nil), nil
}

// GetGroup retrieves a group by ID
func (s *Service) GetGroup(ctx context.Context, input *dto.GetGroupInput) (*dto.GroupOutput, error) {
	id, err := primitive.ObjectIDFromHex(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	group, err := s.repo.GetGroupByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	// Get member count
	memberCount, err := s.repo.GetGroupMemberCount(ctx, group.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member count: %w", err)
	}

	return s.modelToOutput(group, &memberCount), nil
}

// ListGroups retrieves groups with filtering and pagination
func (s *Service) ListGroups(ctx context.Context, input *dto.ListGroupsInput) (*dto.ListGroupsOutput, error) {
	// Build filter
	filter := bson.M{}
	if input.Type != "" {
		filter["type"] = input.Type
	}
	// Only show active groups by default for Phase 1
	filter["is_active"] = true

	// Set defaults if zero values provided
	page := input.Page
	if page == 0 {
		page = 1
	}
	limit := input.Limit
	if limit == 0 {
		limit = 20
	}

	groups, total, err := s.repo.ListGroups(ctx, filter, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	// Convert to output
	outputs := make([]dto.GroupResponse, len(groups))
	for i, group := range groups {
		outputs[i] = *s.modelToGroupResponse(&group, nil)
	}

	return &dto.ListGroupsOutput{
		Body: dto.ListGroupsResponse{
			Groups: outputs,
			Total:  total,
			Page:   page,
			Limit:  limit,
		},
	}, nil
}

// UpdateGroup updates a group
func (s *Service) UpdateGroup(ctx context.Context, input *dto.UpdateGroupInput) (*dto.GroupOutput, error) {
	id, err := primitive.ObjectIDFromHex(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Get existing group
	group, err := s.repo.GetGroupByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	// System groups cannot be updated
	if group.Type == models.GroupTypeSystem {
		return nil, fmt.Errorf("system groups cannot be updated")
	}

	// Build update
	update := bson.M{}
	if input.Body.Name != nil {
		// Check if new name already exists (excluding current group)
		existing, err := s.repo.GetGroupByName(ctx, *input.Body.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing group: %w", err)
		}
		if existing != nil && existing.ID != group.ID {
			return nil, fmt.Errorf("group with name '%s' already exists", *input.Body.Name)
		}
		update["name"] = *input.Body.Name
	}
	if input.Body.Description != nil {
		update["description"] = *input.Body.Description
	}
	if input.Body.IsActive != nil {
		update["is_active"] = *input.Body.IsActive
	}

	if len(update) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	if err := s.repo.UpdateGroup(ctx, id, update); err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	// Get updated group
	updatedGroup, err := s.repo.GetGroupByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated group: %w", err)
	}

	// Get member count
	memberCount, err := s.repo.GetGroupMemberCount(ctx, updatedGroup.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member count: %w", err)
	}

	return s.modelToOutput(updatedGroup, &memberCount), nil
}

// DeleteGroup deletes a group
func (s *Service) DeleteGroup(ctx context.Context, input *dto.DeleteGroupInput) (*dto.SuccessOutput, error) {
	id, err := primitive.ObjectIDFromHex(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Get existing group
	group, err := s.repo.GetGroupByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	// System groups cannot be deleted
	if group.Type == models.GroupTypeSystem {
		return nil, fmt.Errorf("system groups cannot be deleted")
	}

	if err := s.repo.DeleteGroup(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to delete group: %w", err)
	}

	return &dto.SuccessOutput{
		Body: dto.SuccessResponse{
			Message: "Group deleted successfully",
		},
	}, nil
}

// AddMember adds a character to a group
func (s *Service) AddMember(ctx context.Context, input *dto.AddMemberInput, addedBy int64) (*dto.GroupMembershipOutput, error) {
	groupID, err := primitive.ObjectIDFromHex(input.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Verify group exists
	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	// Check if membership already exists
	existing, err := s.repo.GetMembership(ctx, groupID, input.Body.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing membership: %w", err)
	}
	if existing != nil && existing.IsActive {
		return nil, fmt.Errorf("character is already a member of this group")
	}

	// Create membership
	membership := &models.GroupMembership{
		GroupID:     groupID,
		CharacterID: input.Body.CharacterID,
		IsActive:    true,
		AddedBy:     &addedBy,
	}

	if err := s.repo.AddMembership(ctx, membership); err != nil {
		return nil, fmt.Errorf("failed to add membership: %w", err)
	}

	return s.membershipModelToOutput(membership), nil
}

// RemoveMember removes a character from a group
func (s *Service) RemoveMember(ctx context.Context, input *dto.RemoveMemberInput) (*dto.SuccessOutput, error) {
	groupID, err := primitive.ObjectIDFromHex(input.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	characterID, err := strconv.ParseInt(input.CharacterID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid character ID: %w", err)
	}

	// Verify group exists
	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	if err := s.repo.RemoveMembership(ctx, groupID, characterID); err != nil {
		return nil, fmt.Errorf("failed to remove membership: %w", err)
	}

	return &dto.SuccessOutput{
		Body: dto.SuccessResponse{
			Message: "Member removed successfully",
		},
	}, nil
}

// ListMembers lists members of a group
func (s *Service) ListMembers(ctx context.Context, input *dto.ListMembersInput) (*dto.ListMembersOutput, error) {
	groupID, err := primitive.ObjectIDFromHex(input.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Verify group exists
	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("group not found")
	}

	// Build filter - only show active memberships by default for Phase 1
	filter := bson.M{}
	filter["is_active"] = true

	// Set defaults if zero values provided
	page := input.Page
	if page == 0 {
		page = 1
	}
	limit := input.Limit
	if limit == 0 {
		limit = 20
	}

	memberships, total, err := s.repo.ListMemberships(ctx, groupID, filter, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list memberships: %w", err)
	}

	// Collect character IDs to fetch names
	characterIDs := make([]int64, len(memberships))
	for i, membership := range memberships {
		characterIDs[i] = membership.CharacterID
	}

	// Fetch character names
	characterNames, err := s.repo.GetCharacterNames(ctx, characterIDs)
	if err != nil {
		// Log error but continue without names
		slog.ErrorContext(ctx, "Failed to fetch character names", "error", err, "character_ids", characterIDs)
		characterNames = make(map[int64]string)
	} else {
		slog.InfoContext(ctx, "Fetched character names", "count", len(characterNames), "names", characterNames)
	}

	// Convert to output with character names
	outputs := make([]dto.GroupMembershipResponse, len(memberships))
	for i, membership := range memberships {
		response := s.membershipModelToGroupMembershipResponse(&membership)
		// Add character name if found
		if name, ok := characterNames[membership.CharacterID]; ok {
			response.CharacterName = name
			slog.InfoContext(ctx, "Added character name", "character_id", membership.CharacterID, "name", name)
		} else {
			response.CharacterName = fmt.Sprintf("Unknown (%d)", membership.CharacterID)
			slog.WarnContext(ctx, "Character name not found", "character_id", membership.CharacterID)
		}
		outputs[i] = *response
	}

	return &dto.ListMembersOutput{
		Body: dto.ListMembersResponse{
			Members: outputs,
			Total:   total,
			Page:    page,
			Limit:   limit,
		},
	}, nil
}

// CheckMembership checks if a character is a member of a group
func (s *Service) CheckMembership(ctx context.Context, input *dto.CheckMembershipInput) (*dto.MembershipCheckOutput, error) {
	groupID, err := primitive.ObjectIDFromHex(input.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	characterID, err := strconv.ParseInt(input.CharacterID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid character ID: %w", err)
	}

	membership, err := s.repo.GetMembership(ctx, groupID, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}

	if membership == nil {
		return &dto.MembershipCheckOutput{
			Body: dto.MembershipCheckResponse{
				IsMember: false,
				IsActive: false,
			},
		}, nil
	}

	return &dto.MembershipCheckOutput{
		Body: dto.MembershipCheckResponse{
			IsMember: true,
			IsActive: membership.IsActive,
			AddedAt:  &membership.AddedAt,
		},
	}, nil
}

// GetCharacterGroups gets all groups a character belongs to
func (s *Service) GetCharacterGroups(ctx context.Context, input *dto.GetCharacterGroupsInput) (*dto.CharacterGroupsOutput, error) {
	characterID, err := strconv.ParseInt(input.CharacterID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid character ID: %w", err)
	}

	// Build filter
	filter := bson.M{}
	if input.Type != "" {
		filter["type"] = input.Type
	}
	// Only show active groups by default for Phase 1
	filter["is_active"] = true

	groups, err := s.repo.GetCharacterGroups(ctx, characterID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get character groups: %w", err)
	}

	// Convert to output
	outputs := make([]dto.GroupResponse, len(groups))
	for i, group := range groups {
		outputs[i] = *s.modelToGroupResponse(&group, nil)
	}

	return &dto.CharacterGroupsOutput{
		Body: dto.CharacterGroupsResponse{
			Groups: outputs,
			Total:  int64(len(groups)),
		},
	}, nil
}

// GetMyGroups gets all groups the current authenticated user belongs to
func (s *Service) GetMyGroups(ctx context.Context, characterID int64, input *dto.GetMyGroupsInput) (*dto.CharacterGroupsOutput, error) {
	// Build filter
	filter := bson.M{}
	if input.Type != "" {
		filter["type"] = input.Type
	}
	// Only show active groups by default for Phase 1
	filter["is_active"] = true

	groups, err := s.repo.GetCharacterGroups(ctx, characterID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user's groups: %w", err)
	}

	// Convert to output
	outputs := make([]dto.GroupResponse, len(groups))
	for i, group := range groups {
		outputs[i] = *s.modelToGroupResponse(&group, nil)
	}

	return &dto.CharacterGroupsOutput{
		Body: dto.CharacterGroupsResponse{
			Groups: outputs,
			Total:  int64(len(groups)),
		},
	}, nil
}

// GetUserGroups gets all unique groups that any character belonging to a user_id belongs to
func (s *Service) GetUserGroups(ctx context.Context, input *dto.GetUserGroupsInput) (*dto.UserGroupsOutput, error) {
	// Get all character IDs for this user_id
	characterIDs, err := s.repo.GetCharacterIDsByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character IDs for user: %w", err)
	}

	if len(characterIDs) == 0 {
		// User has no characters
		return &dto.UserGroupsOutput{
			Body: dto.UserGroupsResponse{
				UserID:     input.UserID,
				Characters: []int64{},
				Groups:     []dto.GroupResponse{},
				Total:      0,
			},
		}, nil
	}

	// Build filter
	filter := bson.M{}
	if input.Type != "" {
		filter["type"] = input.Type
	}
	// Only show active groups by default
	filter["is_active"] = true

	// Get groups for all characters and deduplicate
	groupMap := make(map[string]*models.Group)
	
	for _, characterID := range characterIDs {
		groups, err := s.repo.GetCharacterGroups(ctx, characterID, filter)
		if err != nil {
			// Log error but continue with other characters
			slog.ErrorContext(ctx, "Failed to get groups for character", "character_id", characterID, "error", err)
			continue
		}

		// Add groups to map to deduplicate
		for _, group := range groups {
			groupMap[group.ID.Hex()] = &group
		}
	}

	// Convert map to slice
	uniqueGroups := make([]dto.GroupResponse, 0, len(groupMap))
	for _, group := range groupMap {
		uniqueGroups = append(uniqueGroups, *s.modelToGroupResponse(group, nil))
	}

	// Sort by group name for consistent output
	sort.Slice(uniqueGroups, func(i, j int) bool {
		return uniqueGroups[i].Name < uniqueGroups[j].Name
	})

	return &dto.UserGroupsOutput{
		Body: dto.UserGroupsResponse{
			UserID:     input.UserID,
			Characters: characterIDs,
			Groups:     uniqueGroups,
			Total:      int64(len(uniqueGroups)),
		},
	}, nil
}

// IsCharacterInGroup checks if a character is in a specific group (by group name)
func (s *Service) IsCharacterInGroup(ctx context.Context, characterID int64, groupName string) (bool, error) {
	// Get group by name
	group, err := s.repo.GetGroupByName(ctx, groupName)
	if err != nil {
		return false, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return false, nil
	}

	// Check membership
	membership, err := s.repo.GetMembership(ctx, group.ID, characterID)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return membership != nil && membership.IsActive, nil
}

// Helper methods

func (s *Service) modelToOutput(group *models.Group, memberCount *int64) *dto.GroupOutput {
	return &dto.GroupOutput{
		Body: dto.GroupResponse{
			ID:          group.ID.Hex(),
			Name:        group.Name,
			Description: group.Description,
			Type:        string(group.Type),
			SystemName:  group.SystemName,
			EVEEntityID: group.EVEEntityID,
			IsActive:    group.IsActive,
			MemberCount: memberCount,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		},
	}
}

func (s *Service) modelToGroupResponse(group *models.Group, memberCount *int64) *dto.GroupResponse {
	return &dto.GroupResponse{
		ID:          group.ID.Hex(),
		Name:        group.Name,
		Description: group.Description,
		Type:        string(group.Type),
		SystemName:  group.SystemName,
		EVEEntityID: group.EVEEntityID,
		IsActive:    group.IsActive,
		MemberCount: memberCount,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}
}

func (s *Service) membershipModelToOutput(membership *models.GroupMembership) *dto.GroupMembershipOutput {
	return &dto.GroupMembershipOutput{
		Body: dto.GroupMembershipResponse{
			ID:            membership.ID.Hex(),
			GroupID:       membership.GroupID.Hex(),
			CharacterID:   membership.CharacterID,
			CharacterName: "", // Will be populated by the caller if needed
			IsActive:      membership.IsActive,
			AddedBy:       membership.AddedBy,
			AddedAt:       membership.AddedAt,
			UpdatedAt:     membership.UpdatedAt,
		},
	}
}

func (s *Service) membershipModelToGroupMembershipResponse(membership *models.GroupMembership) *dto.GroupMembershipResponse {
	return &dto.GroupMembershipResponse{
		ID:            membership.ID.Hex(),
		GroupID:       membership.GroupID.Hex(),
		CharacterID:   membership.CharacterID,
		CharacterName: "", // Will be populated by the caller
		IsActive:      membership.IsActive,
		AddedBy:       membership.AddedBy,
		AddedAt:       membership.AddedAt,
		UpdatedAt:     membership.UpdatedAt,
	}
}



// ValidateCharacterMemberships validates that a character is still in their corporation and alliance via ESI
func (s *Service) ValidateCharacterMemberships(ctx context.Context, characterID int64) error {
	// This is a placeholder for ESI validation - would need to integrate with evegateway
	// For now, we'll just log that validation would happen here
	slog.Debug("[Groups] ESI membership validation", 
		"character_id", characterID,
		"note", "ESI validation not yet implemented - Phase 2")
	
	// TODO: Implement actual ESI validation:
	// 1. Get character info from ESI to get current corp/alliance
	// 2. Compare with stored group memberships
	// 3. Remove character from groups if they've left corp/alliance
	// 4. Add character to new groups if they've joined different corp/alliance
	
	return nil
}

// CleanupInvalidMemberships removes characters from corp/alliance groups if they're no longer members
func (s *Service) CleanupInvalidMemberships(ctx context.Context) error {
	// This would be called by a scheduler task to periodically clean up group memberships
	slog.Debug("[Groups] Cleaning up invalid memberships", 
		"note", "ESI cleanup not yet implemented - Phase 2")
	
	// TODO: Implement batch cleanup:
	// 1. Get all corp/alliance groups
	// 2. For each group, validate all members via ESI
	// 3. Remove invalid memberships
	// 4. Log cleanup results
	
	return nil
}

// EnsureFirstUserSuperAdmin checks if this is the first user and adds them to super_admin group
func (s *Service) EnsureFirstUserSuperAdmin(ctx context.Context, characterID int64) error {
	// Check if super_admin group has any members
	superAdminGroup, err := s.repo.GetGroupBySystemName(ctx, "super_admin")
	if err != nil {
		return fmt.Errorf("failed to get super_admin group: %w", err)
	}
	if superAdminGroup == nil {
		return fmt.Errorf("super_admin group not found")
	}

	// Check if super_admin group has any active members
	memberCount, err := s.repo.GetGroupMemberCount(ctx, superAdminGroup.ID)
	if err != nil {
		return fmt.Errorf("failed to check super_admin member count: %w", err)
	}

	// If no super admins exist, make this user the first super admin
	if memberCount == 0 {
		membership := &models.GroupMembership{
			GroupID:     superAdminGroup.ID,
			CharacterID: characterID,
			IsActive:    true,
			AddedBy:     nil, // System-assigned
		}

		if err := s.repo.AddMembership(ctx, membership); err != nil {
			return fmt.Errorf("failed to add first user to super_admin group: %w", err)
		}

		slog.Info("[Groups] First user assigned to super_admin group", "character_id", characterID)
	}

	return nil
}

// GetStatus returns the health status of the groups module
func (s *Service) GetStatus(ctx context.Context) *dto.GroupsStatusResponse {
	// Check database connectivity
	if err := s.repo.CheckHealth(ctx); err != nil {
		return &dto.GroupsStatusResponse{
			Module:  "groups",
			Status:  "unhealthy",
			Message: "Database connection failed: " + err.Error(),
		}
	}

	return &dto.GroupsStatusResponse{
		Module: "groups",
		Status: "healthy",
	}
}

// AutoJoinCharacterToEnabledGroups automatically joins character to corporation/alliance groups if enabled
func (s *Service) AutoJoinCharacterToEnabledGroups(ctx context.Context, characterID int64, corporationID, allianceID *int64, scopes string) error {
	slog.Debug("Auto-joining character to enabled groups", 
		"character_id", characterID, 
		"corporation_id", corporationID, 
		"alliance_id", allianceID,
		"has_scopes", scopes != "")

	// Determine which system group to join based on scopes
	if scopes != "" {
		// User has scopes - add to "Authenticated Users" group
		if err := s.ensureCharacterInSystemGroup(ctx, characterID, "authenticated", "Authenticated Users"); err != nil {
			slog.Error("Failed to add character to Authenticated Users group", "character_id", characterID, "error", err)
			// Continue anyway - don't fail the whole operation
		}
	} else {
		// User has no scopes - add to "Guest Users" group
		if err := s.ensureCharacterInSystemGroup(ctx, characterID, "guest", "Guest Users"); err != nil {
			slog.Error("Failed to add character to Guest Users group", "character_id", characterID, "error", err)
			// Continue anyway - don't fail the whole operation
		}
	}

	// Remove character from all existing corp/alliance groups first (clean slate approach)
	if err := s.removeCharacterFromEntityGroups(ctx, characterID); err != nil {
		slog.Error("Failed to remove character from existing entity groups", "character_id", characterID, "error", err)
		// Continue anyway - don't fail the whole operation
	}

	// Get enabled entities from site settings
	enabledCorps, err := s.siteSettingsService.GetEnabledCorporations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get enabled corporations: %w", err)
	}

	enabledAlliances, err := s.siteSettingsService.GetEnabledAlliances(ctx)
	if err != nil {
		return fmt.Errorf("failed to get enabled alliances: %w", err)
	}
	
	slog.Debug("Auto-join: Retrieved enabled entities", 
		"character_id", characterID, 
		"enabled_corps_count", len(enabledCorps),
		"enabled_alliances_count", len(enabledAlliances))

	// Join corporation group if character's corp is enabled
	if corporationID != nil {
		for _, corp := range enabledCorps {
			if corp.CorporationID == *corporationID {
				if err := s.ensureCharacterInEntityGroup(ctx, characterID, "corp", corp.CorporationID, corp.Ticker, corp.Name); err != nil {
					slog.Error("Failed to join character to corp group", 
						"character_id", characterID, "corp_id", *corporationID, "error", err)
				} else {
					slog.Info("Auto-joined character to corporation group", 
						"character_id", characterID, "corp_id", *corporationID, "ticker", corp.Ticker)
				}
				break
			}
		}
	}

	// Join alliance group if character's alliance is enabled
	if allianceID != nil {
		for _, alliance := range enabledAlliances {
			if alliance.AllianceID == *allianceID {
				if err := s.ensureCharacterInEntityGroup(ctx, characterID, "alliance", alliance.AllianceID, alliance.Ticker, alliance.Name); err != nil {
					slog.Error("Failed to join character to alliance group", 
						"character_id", characterID, "alliance_id", *allianceID, "error", err)
				} else {
					slog.Info("Auto-joined character to alliance group", 
						"character_id", characterID, "alliance_id", *allianceID, "ticker", alliance.Ticker)
				}
				break
			}
		}
	}

	return nil
}

// ensureCharacterInEntityGroup creates group if needed and adds character
func (s *Service) ensureCharacterInEntityGroup(ctx context.Context, characterID int64, entityType string, entityID int64, ticker, name string) error {
	// Validate that ticker is not empty - this prevents creating malformed groups
	if ticker == "" {
		slog.Error("Cannot create entity group with empty ticker", 
			"character_id", characterID, "entity_type", entityType, "entity_id", entityID, "name", name)
		return fmt.Errorf("ticker is required for entity group creation")
	}
	
	// Create or get group with new naming convention: corp_TICKER or alliance_TICKER
	groupName := fmt.Sprintf("%s_%s", entityType, strings.ToUpper(ticker))
	
	group, err := s.createOrUpdateEntityGroup(ctx, groupName, entityType, entityID, ticker, name)
	if err != nil {
		return fmt.Errorf("failed to create/update entity group: %w", err)
	}

	// Add character to group (upsert - won't duplicate if already exists)
	return s.addMemberToGroup(ctx, group.ID, characterID)
}

// createOrUpdateEntityGroup creates or updates entity group with new naming convention
func (s *Service) createOrUpdateEntityGroup(ctx context.Context, groupName, entityType string, entityID int64, ticker, name string) (*models.Group, error) {
	// Check if group exists by EVE entity ID
	existing, err := s.repo.GetGroupByEVEEntityID(ctx, entityID)
	if err != nil {
		return nil, err
	}

	groupType := models.GroupTypeCorporation
	if entityType == "alliance" {
		groupType = models.GroupTypeAlliance
	}

	if existing != nil {
		// Update existing group if name/ticker changed
		needsUpdate := false
		if existing.Name != groupName {
			existing.Name = groupName
			needsUpdate = true
		}
		if existing.EVEEntityTicker == nil || *existing.EVEEntityTicker != ticker {
			existing.EVEEntityTicker = &ticker
			needsUpdate = true
		}
		if existing.EVEEntityName == nil || *existing.EVEEntityName != name {
			existing.EVEEntityName = &name
			needsUpdate = true
		}

		if needsUpdate {
			update := bson.M{
				"name":               existing.Name,
				"eve_entity_ticker":  existing.EVEEntityTicker,
				"eve_entity_name":    existing.EVEEntityName,
			}
			if err := s.repo.UpdateGroup(ctx, existing.ID, update); err != nil {
				return nil, fmt.Errorf("failed to update group: %w", err)
			}
			slog.Info("Updated entity group", "group_name", groupName, "entity_id", entityID)
		}

		return existing, nil
	}

	// Create new group
	group := &models.Group{
		Name:            groupName,
		Description:     name,
		Type:            groupType,
		EVEEntityID:     &entityID,
		EVEEntityTicker: &ticker,
		EVEEntityName:   &name,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.CreateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	slog.Info("Created new entity group", "group_name", groupName, "entity_id", entityID)
	return group, nil
}

// removeCharacterFromEntityGroups removes character from all corp/alliance groups
func (s *Service) removeCharacterFromEntityGroups(ctx context.Context, characterID int64) error {
	// Get all corp/alliance groups the character belongs to
	filter := bson.M{
		"type": bson.M{"$in": []string{string(models.GroupTypeCorporation), string(models.GroupTypeAlliance)}},
	}
	groups, err := s.repo.GetCharacterGroups(ctx, characterID, filter)
	if err != nil {
		return fmt.Errorf("failed to get character entity groups: %w", err)
	}

	// Remove from each corp/alliance group
	for _, group := range groups {
		if err := s.repo.RemoveMembership(ctx, group.ID, characterID); err != nil {
			slog.Error("Failed to remove character from entity group", 
				"character_id", characterID, "group_id", group.ID.Hex(), "error", err)
			// Continue with other groups
		}
	}

	return nil
}

// ensureCharacterInSystemGroup ensures character is in the specified system group
func (s *Service) ensureCharacterInSystemGroup(ctx context.Context, characterID int64, systemName string, groupDisplayName string) error {
	// Get the system group by system name
	group, err := s.repo.GetGroupBySystemName(ctx, systemName)
	if err != nil {
		return fmt.Errorf("failed to get %s group: %w", groupDisplayName, err)
	}
	if group == nil {
		return fmt.Errorf("%s system group not found", groupDisplayName)
	}

	// First, remove character from other system groups (Guest/Authenticated are mutually exclusive)
	if systemName == "authenticated" {
		// Remove from Guest Users if adding to Authenticated
		if err := s.removeCharacterFromSystemGroup(ctx, characterID, "guest"); err != nil {
			slog.Debug("Failed to remove character from Guest Users group", "character_id", characterID, "error", err)
		}
	} else if systemName == "guest" {
		// Remove from Authenticated Users if adding to Guest
		if err := s.removeCharacterFromSystemGroup(ctx, characterID, "authenticated"); err != nil {
			slog.Debug("Failed to remove character from Authenticated Users group", "character_id", characterID, "error", err)
		}
	}

	// Add or reactivate membership (upsert logic in AddMembership handles duplicates)
	membership := &models.GroupMembership{
		GroupID:     group.ID,
		CharacterID: characterID,
		IsActive:    true,
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if err := s.repo.AddMembership(ctx, membership); err != nil {
		return fmt.Errorf("failed to add/update membership to %s group: %w", groupDisplayName, err)
	}
	
	slog.Info("Ensured character is in system group", "character_id", characterID, "group_name", groupDisplayName, "group_id", group.ID.Hex())
	return nil
}

// removeCharacterFromSystemGroup removes character from a specific system group
func (s *Service) removeCharacterFromSystemGroup(ctx context.Context, characterID int64, systemName string) error {
	group, err := s.repo.GetGroupBySystemName(ctx, systemName)
	if err != nil {
		return fmt.Errorf("failed to get system group %s: %w", systemName, err)
	}
	if group == nil {
		// Group doesn't exist, nothing to remove
		return nil
	}

	if err := s.repo.RemoveMembership(ctx, group.ID, characterID); err != nil {
		return fmt.Errorf("failed to remove membership from system group %s: %w", systemName, err)
	}

	return nil
}

// addMemberToGroup adds character to group (helper method)
func (s *Service) addMemberToGroup(ctx context.Context, groupID primitive.ObjectID, characterID int64) error {
	membership := &models.GroupMembership{
		GroupID:     groupID,
		CharacterID: characterID,
		IsActive:    true,
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.repo.AddMembership(ctx, membership)
}

// RemoveCharacterFromAllGroups removes a character from all groups (for user deletion cleanup)
func (s *Service) RemoveCharacterFromAllGroups(ctx context.Context, characterID int64) error {
	// Get all groups the character belongs to
	groups, err := s.repo.GetCharacterGroups(ctx, characterID, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to get character groups: %w", err)
	}

	// Remove character from each group
	for _, group := range groups {
		if err := s.repo.RemoveMembership(ctx, group.ID, characterID); err != nil {
			slog.Error("Failed to remove character from group during user deletion", 
				"character_id", characterID, "group_id", group.ID.Hex(), "group_name", group.Name, "error", err)
			// Continue with other groups - don't fail the entire operation
		} else {
			slog.Info("Removed character from group during user deletion", 
				"character_id", characterID, "group_id", group.ID.Hex(), "group_name", group.Name)
		}
	}

	slog.Info("Completed group membership cleanup for deleted user", "character_id", characterID, "groups_count", len(groups))
	return nil
}

// Permission Management Methods

// ListPermissions returns all available permissions
func (s *Service) ListPermissions(ctx context.Context, input *dto.ListPermissionsInput) (*dto.ListPermissionsOutput, error) {
	if s.permissionManager == nil {
		return nil, fmt.Errorf("permission manager not available")
	}
	
	// Get all permissions
	allPermissions := s.permissionManager.GetAllPermissions()
	
	// Filter permissions if requested
	var filteredPerms []permissions.Permission
	for _, perm := range allPermissions {
		if input.Service != "" && perm.Service != input.Service {
			continue
		}
		if input.Category != "" && perm.Category != input.Category {
			continue
		}
		if input.IsStatic != "" {
			isStatic := input.IsStatic == "true"
			if perm.IsStatic != isStatic {
				continue
			}
		}
		filteredPerms = append(filteredPerms, perm)
	}
	
	// Convert to response DTOs
	var permResponses []dto.PermissionResponse
	for _, perm := range filteredPerms {
		permResponses = append(permResponses, dto.PermissionResponse{
			ID:          perm.ID,
			Service:     perm.Service,
			Resource:    perm.Resource,
			Action:      perm.Action,
			IsStatic:    perm.IsStatic,
			Name:        perm.Name,
			Description: perm.Description,
			Category:    perm.Category,
			CreatedAt:   perm.CreatedAt,
		})
	}
	
	// Get permission categories
	var categories []dto.PermissionCategory
	for _, cat := range permissions.PermissionCategories {
		categories = append(categories, dto.PermissionCategory{
			Name:        cat.Name,
			Description: cat.Description,
			Order:       cat.Order,
		})
	}
	
	return &dto.ListPermissionsOutput{
		Body: dto.ListPermissionsResponse{
			Permissions: permResponses,
			Categories:  categories,
			Total:       int64(len(permResponses)),
		},
	}, nil
}

// GetPermission returns a specific permission by ID
func (s *Service) GetPermission(ctx context.Context, input *dto.GetPermissionInput) (*dto.PermissionOutput, error) {
	if s.permissionManager == nil {
		return nil, fmt.Errorf("permission manager not available")
	}
	
	perm, exists := s.permissionManager.GetPermission(input.PermissionID)
	if !exists {
		return nil, fmt.Errorf("permission not found: %s", input.PermissionID)
	}
	
	return &dto.PermissionOutput{
		Body: dto.PermissionResponse{
			ID:          perm.ID,
			Service:     perm.Service,
			Resource:    perm.Resource,
			Action:      perm.Action,
			IsStatic:    perm.IsStatic,
			Name:        perm.Name,
			Description: perm.Description,
			Category:    perm.Category,
			CreatedAt:   perm.CreatedAt,
		},
	}, nil
}

// GrantPermissionToGroup assigns a permission to a group
func (s *Service) GrantPermissionToGroup(ctx context.Context, input *dto.GrantPermissionToGroupInput, grantedBy int64) (*dto.GroupPermissionOutput, error) {
	if s.permissionManager == nil {
		return nil, fmt.Errorf("permission manager not available")
	}
	
	// Parse group ID
	groupID, err := primitive.ObjectIDFromHex(input.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}
	
	// Verify group exists
	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}
	
	// Grant permission
	err = s.permissionManager.GrantPermissionToGroup(ctx, groupID, input.PermissionID, grantedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to grant permission: %w", err)
	}
	
	// Get permission details for response
	perm, exists := s.permissionManager.GetPermission(input.PermissionID)
	if !exists {
		return nil, fmt.Errorf("permission not found: %s", input.PermissionID)
	}
	
	return &dto.GroupPermissionOutput{
		Body: dto.GroupPermissionResponse{
			GroupID:      groupID.Hex(),
			GroupName:    group.Name,
			PermissionID: input.PermissionID,
			Permission: dto.PermissionResponse{
				ID:          perm.ID,
				Service:     perm.Service,
				Resource:    perm.Resource,
				Action:      perm.Action,
				IsStatic:    perm.IsStatic,
				Name:        perm.Name,
				Description: perm.Description,
				Category:    perm.Category,
				CreatedAt:   perm.CreatedAt,
			},
			GrantedBy: &grantedBy,
			GrantedAt: time.Now(),
			IsActive:  true,
			UpdatedAt: time.Now(),
		},
	}, nil
}

// RevokePermissionFromGroup removes a permission from a group
func (s *Service) RevokePermissionFromGroup(ctx context.Context, input *dto.RevokePermissionFromGroupInput) (*dto.MessageOutput, error) {
	if s.permissionManager == nil {
		return nil, fmt.Errorf("permission manager not available")
	}
	
	// Parse group ID
	groupID, err := primitive.ObjectIDFromHex(input.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}
	
	// Revoke permission
	err = s.permissionManager.RevokePermissionFromGroup(ctx, groupID, input.PermissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke permission: %w", err)
	}
	
	return &dto.MessageOutput{
		Body: dto.MessageResponse{
			Message: fmt.Sprintf("Permission %s revoked from group %s", input.PermissionID, input.GroupID),
		},
	}, nil
}

// ListGroupPermissions returns all permissions assigned to a group
func (s *Service) ListGroupPermissions(ctx context.Context, input *dto.ListGroupPermissionsInput) (*dto.ListGroupPermissionsOutput, error) {
	if s.permissionManager == nil {
		return nil, fmt.Errorf("permission manager not available")
	}
	
	// Parse group ID
	groupID, err := primitive.ObjectIDFromHex(input.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}
	
	// Get group details
	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}
	
	// Build aggregation pipeline to get group permissions with permission details
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"group_id": groupID,
			},
		},
	}
	
	// Add active filter if specified
	if input.IsActive != "" {
		isActive := input.IsActive == "true"
		pipeline[0]["$match"].(bson.M)["is_active"] = isActive
	}
	
	// Execute aggregation
	cursor, err := s.repo.db.Collection("group_permissions").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to query group permissions: %w", err)
	}
	defer cursor.Close(ctx)
	
	var groupPermissions []dto.GroupPermissionResponse
	for cursor.Next(ctx) {
		var gp permissions.GroupPermission
		if err := cursor.Decode(&gp); err != nil {
			slog.Warn("Failed to decode group permission", "error", err)
			continue
		}
		
		// Get permission details
		perm, exists := s.permissionManager.GetPermission(gp.PermissionID)
		if !exists {
			slog.Warn("Permission not found for group permission", "permission_id", gp.PermissionID)
			continue
		}
		
		groupPermissions = append(groupPermissions, dto.GroupPermissionResponse{
			ID:           gp.ID.Hex(),
			GroupID:      gp.GroupID.Hex(),
			GroupName:    group.Name,
			PermissionID: gp.PermissionID,
			Permission: dto.PermissionResponse{
				ID:          perm.ID,
				Service:     perm.Service,
				Resource:    perm.Resource,
				Action:      perm.Action,
				IsStatic:    perm.IsStatic,
				Name:        perm.Name,
				Description: perm.Description,
				Category:    perm.Category,
				CreatedAt:   perm.CreatedAt,
			},
			GrantedBy: gp.GrantedBy,
			GrantedAt: gp.GrantedAt,
			IsActive:  gp.IsActive,
			UpdatedAt: gp.UpdatedAt,
		})
	}
	
	return &dto.ListGroupPermissionsOutput{
		Body: dto.ListGroupPermissionsResponse{
			GroupID:     groupID.Hex(),
			GroupName:   group.Name,
			Permissions: groupPermissions,
			Total:       int64(len(groupPermissions)),
		},
	}, nil
}

// CheckPermission checks if a character has a specific permission
func (s *Service) CheckPermission(ctx context.Context, input *dto.CheckPermissionInput, authenticatedCharacterID int64) (*dto.PermissionCheckOutput, error) {
	if s.permissionManager == nil {
		return nil, fmt.Errorf("permission manager not available")
	}
	
	// Use provided character ID or default to authenticated user
	characterID := authenticatedCharacterID
	if input.CharacterID != "" {
		parsedCharacterID, err := strconv.ParseInt(input.CharacterID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid character ID: %w", err)
		}
		characterID = parsedCharacterID
	}
	
	// Check permission
	permCheck, err := s.permissionManager.CheckPermission(ctx, characterID, input.PermissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}
	
	return &dto.PermissionCheckOutput{
		Body: dto.PermissionCheckResponse{
			CharacterID:  permCheck.CharacterID,
			PermissionID: permCheck.PermissionID,
			Granted:      permCheck.Granted,
			GrantedVia:   permCheck.GrantedVia,
		},
	}, nil
}