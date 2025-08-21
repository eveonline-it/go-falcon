package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-falcon/internal/groups/dto"
	"go-falcon/internal/groups/models"
	"go-falcon/pkg/database"
)

// Service handles business logic for groups
type Service struct {
	repo *Repository
}

// NewService creates a new service instance
func NewService(db *database.MongoDB) *Service {
	return &Service{
		repo: NewRepository(db),
	}
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
		CreatedBy:   &createdBy,
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
	outputs := make([]dto.GroupOutput, len(groups))
	for i, group := range groups {
		outputs[i] = *s.modelToOutput(&group, nil)
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
		Message: "Group deleted successfully",
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
		Message: "Member removed successfully",
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

	// Convert to output
	outputs := make([]dto.GroupMembershipOutput, len(memberships))
	for i, membership := range memberships {
		outputs[i] = *s.membershipModelToOutput(&membership)
	}

	return &dto.ListMembersOutput{
		Members: outputs,
		Total:   total,
		Page:    page,
		Limit:   limit,
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
			IsMember: false,
			IsActive: false,
		}, nil
	}

	return &dto.MembershipCheckOutput{
		IsMember: true,
		IsActive: membership.IsActive,
		AddedAt:  &membership.AddedAt,
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
	outputs := make([]dto.GroupOutput, len(groups))
	for i, group := range groups {
		outputs[i] = *s.modelToOutput(&group, nil)
	}

	return &dto.CharacterGroupsOutput{
		Groups: outputs,
		Total:  int64(len(groups)),
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
		ID:          group.ID.Hex(),
		Name:        group.Name,
		Description: group.Description,
		Type:        string(group.Type),
		SystemName:  group.SystemName,
		EVEEntityID: group.EVEEntityID,
		IsActive:    group.IsActive,
		MemberCount: memberCount,
		CreatedBy:   group.CreatedBy,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}
}

func (s *Service) membershipModelToOutput(membership *models.GroupMembership) *dto.GroupMembershipOutput {
	return &dto.GroupMembershipOutput{
		ID:          membership.ID.Hex(),
		GroupID:     membership.GroupID.Hex(),
		CharacterID: membership.CharacterID,
		IsActive:    membership.IsActive,
		AddedBy:     membership.AddedBy,
		AddedAt:     membership.AddedAt,
		UpdatedAt:   membership.UpdatedAt,
	}
}

// SyncCharacterGroups synchronizes a character's corporation and alliance group memberships
func (s *Service) SyncCharacterGroups(ctx context.Context, characterID int64, corporationID, allianceID *int64) error {
	// Sync corporation groups
	if corporationID != nil {
		if err := s.syncCorporationGroups(ctx, characterID, *corporationID); err != nil {
			return fmt.Errorf("failed to sync corporation groups: %w", err)
		}
	}
	
	// Sync alliance groups  
	if allianceID != nil {
		if err := s.syncAllianceGroups(ctx, characterID, *allianceID); err != nil {
			return fmt.Errorf("failed to sync alliance groups: %w", err)
		}
	}
	
	return nil
}

// syncCorporationGroups ensures character is in corporation group and creates group if needed
func (s *Service) syncCorporationGroups(ctx context.Context, characterID int64, corporationID int64) error {
	// Find or create corporation group
	group, err := s.findOrCreateEVEEntityGroup(ctx, corporationID, models.GroupTypeCorporation)
	if err != nil {
		return fmt.Errorf("failed to find/create corporation group: %w", err)
	}
	
	// Check if character is already a member
	membership, err := s.repo.GetMembership(ctx, group.ID, characterID)
	if err != nil {
		return fmt.Errorf("failed to check corporation membership: %w", err)
	}
	
	// Add character to corporation group if not already a member
	if membership == nil || !membership.IsActive {
		err = s.repo.AddMembership(ctx, &models.GroupMembership{
			GroupID:     group.ID,
			CharacterID: characterID,
			IsActive:    true,
			AddedBy:     nil, // System-assigned
		})
		if err != nil {
			return fmt.Errorf("failed to add character to corporation group: %w", err)
		}
	}
	
	return nil
}

// syncAllianceGroups ensures character is in alliance group and creates group if needed
func (s *Service) syncAllianceGroups(ctx context.Context, characterID int64, allianceID int64) error {
	// Find or create alliance group
	group, err := s.findOrCreateEVEEntityGroup(ctx, allianceID, models.GroupTypeAlliance)
	if err != nil {
		return fmt.Errorf("failed to find/create alliance group: %w", err)
	}
	
	// Check if character is already a member
	membership, err := s.repo.GetMembership(ctx, group.ID, characterID)
	if err != nil {
		return fmt.Errorf("failed to check alliance membership: %w", err)
	}
	
	// Add character to alliance group if not already a member
	if membership == nil || !membership.IsActive {
		err = s.repo.AddMembership(ctx, &models.GroupMembership{
			GroupID:     group.ID,
			CharacterID: characterID,
			IsActive:    true,
			AddedBy:     nil, // System-assigned
		})
		if err != nil {
			return fmt.Errorf("failed to add character to alliance group: %w", err)
		}
	}
	
	return nil
}

// findOrCreateEVEEntityGroup finds existing or creates new corporation/alliance group
func (s *Service) findOrCreateEVEEntityGroup(ctx context.Context, entityID int64, groupType models.GroupType) (*models.Group, error) {
	// Try to find existing group
	group, err := s.repo.GetGroupByEVEEntityID(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to search for EVE entity group: %w", err)
	}
	
	if group != nil {
		return group, nil
	}
	
	// Create new group for this EVE entity
	var groupName string
	var description string
	
	switch groupType {
	case models.GroupTypeCorporation:
		groupName = fmt.Sprintf("Corp_%d", entityID)
		description = fmt.Sprintf("Corporation group for Corp ID: %d", entityID)
	case models.GroupTypeAlliance:
		groupName = fmt.Sprintf("Alliance_%d", entityID)
		description = fmt.Sprintf("Alliance group for Alliance ID: %d", entityID)
	default:
		return nil, fmt.Errorf("unsupported group type: %s", groupType)
	}
	
	newGroup := &models.Group{
		Name:        groupName,
		Description: description,
		Type:        groupType,
		EVEEntityID: &entityID,
		IsActive:    true,
		CreatedBy:   nil, // System-created
	}
	
	err = s.repo.CreateGroup(ctx, newGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to create EVE entity group: %w", err)
	}
	
	return newGroup, nil
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