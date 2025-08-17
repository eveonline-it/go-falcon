package services

import (
	"context"
	"fmt"
	"log/slog"

	"go-falcon/internal/groups/dto"
	"go-falcon/internal/groups/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GroupService handles group management operations
type GroupService struct {
	repository *Repository
	mongodb    *database.MongoDB
	redis      *database.Redis
}

// NewGroupService creates a new group service
func NewGroupService(mongodb *database.MongoDB, redis *database.Redis) *GroupService {
	return &GroupService{
		repository: NewRepository(mongodb),
		mongodb:    mongodb,
		redis:      redis,
	}
}

// Group Management Operations

// CreateGroup creates a new group
func (gs *GroupService) CreateGroup(ctx context.Context, req *dto.GroupCreateRequest, createdBy int) (*models.Group, error) {
	// Check if group with this name already exists
	_, err := gs.repository.GetGroupByName(ctx, req.Name)
	if err == nil {
		return nil, fmt.Errorf("group with name '%s' already exists", req.Name)
	}
	if err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("failed to check existing group: %w", err)
	}

	// Create the group
	group := &models.Group{
		Name:                req.Name,
		Description:         req.Description,
		IsDefault:           req.IsDefault,
		DiscordRoles:        req.DiscordRoles,
		AutoAssignmentRules: req.AutoAssignmentRules,
		CreatedBy:           createdBy,
		MemberCount:         0,
	}

	if err := gs.repository.CreateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Action:      "create_group",
		GroupID:     &group.ID,
		PerformedBy: createdBy,
		Details: map[string]interface{}{
			"group_name":   req.Name,
			"description":  req.Description,
			"is_default":   req.IsDefault,
			"discord_roles": len(req.DiscordRoles),
		},
		Reason: "Group created",
	}

	if err := gs.repository.CreateAuditLog(ctx, auditLog); err != nil {
		slog.Error("Failed to create audit log", slog.String("error", err.Error()))
	}

	slog.Info("Group created", 
		slog.String("name", req.Name), 
		slog.Int("created_by", createdBy),
		slog.String("group_id", group.ID.Hex()))

	return group, nil
}

// GetGroup retrieves a group by ID
func (gs *GroupService) GetGroup(ctx context.Context, groupID string) (*models.Group, error) {
	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	group, err := gs.repository.GetGroup(ctx, objectID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("group not found")
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	return group, nil
}

// GetGroupByID retrieves a group by ID (used by other services)
func (gs *GroupService) GetGroupByID(ctx context.Context, groupID string) (*models.Group, error) {
	return gs.GetGroup(ctx, groupID)
}

// GetGroupByName retrieves a group by name
func (gs *GroupService) GetGroupByName(ctx context.Context, name string) (*models.Group, error) {
	group, err := gs.repository.GetGroupByName(ctx, name)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("group not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	return group, nil
}

// UpdateGroup updates an existing group
func (gs *GroupService) UpdateGroup(ctx context.Context, groupID string, req *dto.GroupUpdateRequest, updatedBy int) (*models.Group, error) {
	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Get existing group
	group, err := gs.repository.GetGroup(ctx, objectID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("group not found")
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	// Store old values for audit
	oldValues := map[string]interface{}{
		"name":                  group.Name,
		"description":           group.Description,
		"is_default":            group.IsDefault,
		"discord_roles":         group.DiscordRoles,
		"auto_assignment_rules": group.AutoAssignmentRules,
	}

	// Update fields if provided
	newValues := make(map[string]interface{})
	
	if req.Name != nil {
		if *req.Name != group.Name {
			// Check if new name conflicts with existing group
			existing, err := gs.repository.GetGroupByName(ctx, *req.Name)
			if err == nil && existing.ID != group.ID {
				return nil, fmt.Errorf("group with name '%s' already exists", *req.Name)
			}
			if err != nil && err != mongo.ErrNoDocuments {
				return nil, fmt.Errorf("failed to check existing group: %w", err)
			}
		}
		group.Name = *req.Name
		newValues["name"] = *req.Name
	}

	if req.Description != nil {
		group.Description = *req.Description
		newValues["description"] = *req.Description
	}

	if req.IsDefault != nil {
		group.IsDefault = *req.IsDefault
		newValues["is_default"] = *req.IsDefault
	}

	if req.DiscordRoles != nil {
		group.DiscordRoles = req.DiscordRoles
		newValues["discord_roles"] = req.DiscordRoles
	}

	if req.AutoAssignmentRules != nil {
		group.AutoAssignmentRules = req.AutoAssignmentRules
		newValues["auto_assignment_rules"] = req.AutoAssignmentRules
	}

	// Update the group
	if err := gs.repository.UpdateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Action:      "update_group",
		GroupID:     &group.ID,
		PerformedBy: updatedBy,
		Details: map[string]interface{}{
			"old_values": oldValues,
			"new_values": newValues,
		},
		Reason: "Group updated",
	}

	if err := gs.repository.CreateAuditLog(ctx, auditLog); err != nil {
		slog.Error("Failed to create audit log", slog.String("error", err.Error()))
	}

	slog.Info("Group updated", 
		slog.String("name", group.Name), 
		slog.Int("updated_by", updatedBy),
		slog.String("group_id", group.ID.Hex()))

	return group, nil
}

// DeleteGroup deletes a group and all its memberships
func (gs *GroupService) DeleteGroup(ctx context.Context, groupID string, deletedBy int) error {
	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %w", err)
	}

	// Get group for audit logging
	group, err := gs.repository.GetGroup(ctx, objectID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("group not found")
		}
		return fmt.Errorf("failed to get group: %w", err)
	}

	// Check if this is a system group that shouldn't be deleted
	if group.Name == "super_admin" || group.Name == "administrators" || group.Name == "members" {
		return fmt.Errorf("cannot delete system group: %s", group.Name)
	}

	// Delete all memberships for this group first
	memberships, _, err := gs.repository.GetGroupMemberships(ctx, objectID, 1, 1000) // Get up to 1000 for logging
	if err != nil {
		slog.Warn("Failed to get memberships for deletion audit", slog.String("error", err.Error()))
	}

	// Remove memberships (this will be handled by DeleteGroup in repository)
	// Delete the group - this will cascade delete memberships
	if err := gs.repository.DeleteGroup(ctx, objectID); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Action:      "delete_group",
		GroupID:     &group.ID,
		PerformedBy: deletedBy,
		Details: map[string]interface{}{
			"group_name":       group.Name,
			"member_count":     len(memberships),
			"was_default":      group.IsDefault,
			"discord_roles":    len(group.DiscordRoles),
		},
		Reason: "Group deleted",
	}

	if err := gs.repository.CreateAuditLog(ctx, auditLog); err != nil {
		slog.Error("Failed to create audit log", slog.String("error", err.Error()))
	}

	slog.Info("Group deleted", 
		slog.String("name", group.Name), 
		slog.Int("deleted_by", deletedBy),
		slog.String("group_id", group.ID.Hex()),
		slog.Int("memberships_removed", len(memberships)))

	return nil
}

// ListGroups lists groups with filtering and pagination
func (gs *GroupService) ListGroups(ctx context.Context, query *dto.GroupListQuery) (*dto.GroupListResponse, error) {
	// Build filter
	filter := bson.M{}
	
	if query.IsDefault != nil {
		filter["is_default"] = *query.IsDefault
	}

	if query.Search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": query.Search, "$options": "i"}},
			{"description": bson.M{"$regex": query.Search, "$options": "i"}},
		}
	}

	// Get groups with pagination
	groups, total, err := gs.repository.ListGroups(ctx, filter, query.Page, query.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	// Convert to response DTOs
	var groupResponses []dto.GroupResponse
	for _, group := range groups {
		groupResp := dto.GroupResponse{
			ID:                  group.ID.Hex(),
			Name:                group.Name,
			Description:         group.Description,
			IsDefault:           group.IsDefault,
			MemberCount:         group.MemberCount,
			DiscordRoles:        group.DiscordRoles,
			AutoAssignmentRules: group.AutoAssignmentRules,
			CreatedAt:           group.CreatedAt,
			UpdatedAt:           group.UpdatedAt,
			CreatedBy:           group.CreatedBy,
		}

		// Add members if requested
		if query.ShowMembers {
			memberships, _, err := gs.repository.GetGroupMemberships(ctx, group.ID, 1, 100) // Limit to 100 members
			if err != nil {
				slog.Warn("Failed to get group members", 
					slog.String("group_id", group.ID.Hex()),
					slog.String("error", err.Error()))
			} else {
				for _, membership := range memberships {
					groupResp.Members = append(groupResp.Members, dto.MembershipResponse{
						CharacterID:        membership.CharacterID,
						AssignedAt:         membership.AssignedAt,
						ExpiresAt:          membership.ExpiresAt,
						ValidationStatus:   membership.ValidationStatus,
						LastValidated:      membership.LastValidated,
						AssignmentSource:   membership.AssignmentSource,
						AssignmentMetadata: membership.AssignmentMetadata,
					})
				}
			}
		}

		groupResponses = append(groupResponses, groupResp)
	}

	return &dto.GroupListResponse{
		Groups: groupResponses,
		Pagination: dto.PaginationResponse{
			Page:       query.Page,
			PageSize:   query.PageSize,
			Total:      int(total),
			TotalPages: int((total + int64(query.PageSize) - 1) / int64(query.PageSize)),
		},
	}, nil
}

// GetDefaultGroups returns all default groups
func (gs *GroupService) GetDefaultGroups(ctx context.Context) ([]models.Group, error) {
	groups, err := gs.repository.GetDefaultGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default groups: %w", err)
	}

	return groups, nil
}

// Membership Management Operations

// AddMember adds a member to a group
func (gs *GroupService) AddMember(ctx context.Context, groupID string, req *dto.MembershipRequest, addedBy int) (*models.GroupMembership, error) {
	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Check if group exists
	group, err := gs.repository.GetGroup(ctx, objectID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("group not found")
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	// Check if membership already exists
	_, err = gs.repository.GetMembership(ctx, req.CharacterID, objectID)
	if err == nil {
		return nil, fmt.Errorf("character %d is already a member of group %s", req.CharacterID, group.Name)
	}
	if err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("failed to check existing membership: %w", err)
	}

	// Create membership
	membership := &models.GroupMembership{
		CharacterID:        req.CharacterID,
		GroupID:            objectID,
		ExpiresAt:          req.ExpiresAt,
		AssignmentSource:   "manual",
		AssignmentMetadata: req.AssignmentMetadata,
	}

	if err := gs.repository.CreateMembership(ctx, membership); err != nil {
		return nil, fmt.Errorf("failed to create membership: %w", err)
	}

	// Update group member count
	group.MemberCount++
	if err := gs.repository.UpdateGroup(ctx, group); err != nil {
		slog.Error("Failed to update group member count", slog.String("error", err.Error()))
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Action:      "add_member",
		GroupID:     &objectID,
		CharacterID: &req.CharacterID,
		PerformedBy: addedBy,
		Details: map[string]interface{}{
			"group_name":    group.Name,
			"character_id":  req.CharacterID,
			"expires_at":    req.ExpiresAt,
			"source":        "manual",
		},
		Reason: req.Reason,
	}

	if err := gs.repository.CreateAuditLog(ctx, auditLog); err != nil {
		slog.Error("Failed to create audit log", slog.String("error", err.Error()))
	}

	slog.Info("Member added to group", 
		slog.String("group_name", group.Name),
		slog.Int("character_id", req.CharacterID),
		slog.Int("added_by", addedBy))

	return membership, nil
}

// RemoveMember removes a member from a group
func (gs *GroupService) RemoveMember(ctx context.Context, groupID string, characterID int, removedBy int, reason string) error {
	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %w", err)
	}

	// Check if group exists
	group, err := gs.repository.GetGroup(ctx, objectID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("group not found")
		}
		return fmt.Errorf("failed to get group: %w", err)
	}

	// Check if membership exists
	_, err = gs.repository.GetMembership(ctx, characterID, objectID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("character %d is not a member of group %s", characterID, group.Name)
		}
		return fmt.Errorf("failed to check membership: %w", err)
	}

	// Remove membership
	if err := gs.repository.DeleteMembership(ctx, characterID, objectID); err != nil {
		return fmt.Errorf("failed to remove membership: %w", err)
	}

	// Update group member count
	if group.MemberCount > 0 {
		group.MemberCount--
		if err := gs.repository.UpdateGroup(ctx, group); err != nil {
			slog.Error("Failed to update group member count", slog.String("error", err.Error()))
		}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Action:      "remove_member",
		GroupID:     &objectID,
		CharacterID: &characterID,
		PerformedBy: removedBy,
		Details: map[string]interface{}{
			"group_name":   group.Name,
			"character_id": characterID,
		},
		Reason: reason,
	}

	if err := gs.repository.CreateAuditLog(ctx, auditLog); err != nil {
		slog.Error("Failed to create audit log", slog.String("error", err.Error()))
	}

	slog.Info("Member removed from group", 
		slog.String("group_name", group.Name),
		slog.Int("character_id", characterID),
		slog.Int("removed_by", removedBy))

	return nil
}

// GetGroupMembers returns members of a group with pagination
func (gs *GroupService) GetGroupMembers(ctx context.Context, groupID string, page, pageSize int) (*dto.MembershipListResponse, error) {
	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Get memberships
	memberships, total, err := gs.repository.GetGroupMemberships(ctx, objectID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get group memberships: %w", err)
	}

	// Convert to response DTOs
	var memberResponses []dto.MembershipResponse
	for _, membership := range memberships {
		memberResponses = append(memberResponses, dto.MembershipResponse{
			CharacterID:        membership.CharacterID,
			AssignedAt:         membership.AssignedAt,
			ExpiresAt:          membership.ExpiresAt,
			ValidationStatus:   membership.ValidationStatus,
			LastValidated:      membership.LastValidated,
			AssignmentSource:   membership.AssignmentSource,
			AssignmentMetadata: membership.AssignmentMetadata,
		})
	}

	return &dto.MembershipListResponse{
		Members: memberResponses,
		Pagination: dto.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			Total:      int(total),
			TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
		},
	}, nil
}

// GetUserGroups returns all groups a user is a member of
func (gs *GroupService) GetUserGroups(ctx context.Context, characterID int) ([]models.Group, error) {
	// Get user memberships
	memberships, err := gs.repository.GetUserMemberships(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user memberships: %w", err)
	}

	// Get groups for each membership
	var groups []models.Group
	for _, membership := range memberships {
		group, err := gs.repository.GetGroup(ctx, membership.GroupID)
		if err != nil {
			slog.Warn("Failed to get group for membership", 
				slog.String("group_id", membership.GroupID.Hex()),
				slog.String("error", err.Error()))
			continue
		}
		groups = append(groups, *group)
	}

	return groups, nil
}

// Utility Methods

// InitializeDefaultGroups creates the default system groups if they don't exist
func (gs *GroupService) InitializeDefaultGroups(ctx context.Context) error {
	slog.Info("Initializing default groups")

	defaultGroups := []struct {
		name        string
		description string
		isDefault   bool
	}{
		{
			name:        "super_admin",
			description: "Super administrators with full system access",
			isDefault:   false,
		},
		{
			name:        "administrators",
			description: "System administrators with elevated privileges",
			isDefault:   false,
		},
		{
			name:        "members",
			description: "Default group for all authenticated users",
			isDefault:   true,
		},
	}

	for _, defaultGroup := range defaultGroups {
		// Check if group already exists
		_, err := gs.repository.GetGroupByName(ctx, defaultGroup.name)
		if err == nil {
			slog.Info("Default group already exists, skipping", slog.String("group", defaultGroup.name))
			continue
		}

		if err != mongo.ErrNoDocuments {
			slog.Error("Failed to check if default group exists", 
				slog.String("group", defaultGroup.name),
				slog.String("error", err.Error()))
			continue
		}

		// Create the group
		group := &models.Group{
			Name:        defaultGroup.name,
			Description: defaultGroup.description,
			IsDefault:   defaultGroup.isDefault,
			CreatedBy:   0, // System created
			MemberCount: 0,
		}

		if err := gs.repository.CreateGroup(ctx, group); err != nil {
			slog.Error("Failed to create default group", 
				slog.String("group", defaultGroup.name),
				slog.String("error", err.Error()))
			continue
		}

		slog.Info("Created default group", slog.String("group", defaultGroup.name))
	}

	slog.Info("Default groups initialization complete")
	return nil
}

// CleanupExpiredMemberships removes expired group memberships
func (gs *GroupService) CleanupExpiredMemberships(ctx context.Context) (int64, error) {
	return gs.repository.CleanupExpiredMemberships(ctx)
}

// GetMembershipStats returns membership statistics
func (gs *GroupService) GetMembershipStats(ctx context.Context) (*models.MembershipStats, error) {
	return gs.repository.GetMembershipStats(ctx)
}