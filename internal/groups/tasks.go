package groups

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/pkg/config"
	"go-falcon/pkg/evegateway"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GroupTask represents a scheduled task related to group management
type GroupTask struct {
	groupService      *GroupService
	permissionService *PermissionService
	discordService    *DiscordService
	eveClient         *evegateway.Client
}

func NewGroupTask(groupService *GroupService, permissionService *PermissionService) *GroupTask {
	return &GroupTask{
		groupService:      groupService,
		permissionService: permissionService,
		discordService:    NewDiscordService(),
		eveClient:         evegateway.NewClient(),
	}
}

// ValidateCorporateMemberships validates all corporate and alliance group memberships against ESI data
func (gt *GroupTask) ValidateCorporateMemberships(ctx context.Context) error {
	slog.Info("Starting corporate and alliance membership validation task")
	
	// Validate corporate group
	err := gt.validateGroupMemberships(ctx, "corporate", "ENABLED_CORPORATION_IDS", func(charInfo map[string]any, enabledIDs []int) bool {
		corporationID, ok := charInfo["corporation_id"].(float64)
		if !ok {
			return false
		}
		for _, corpID := range enabledIDs {
			if int(corporationID) == corpID {
				return true
			}
		}
		return false
	})
	if err != nil {
		slog.Error("Failed to validate corporate memberships", slog.String("error", err.Error()))
	}

	// Validate alliance group
	err = gt.validateGroupMemberships(ctx, "alliance", "ENABLED_ALLIANCE_IDS", func(charInfo map[string]any, enabledIDs []int) bool {
		allianceIDVal, exists := charInfo["alliance_id"]
		if !exists {
			return false
		}
		allianceID, ok := allianceIDVal.(float64)
		if !ok {
			return false
		}
		for _, enabledAllianceID := range enabledIDs {
			if int(allianceID) == enabledAllianceID {
				return true
			}
		}
		return false
	})
	if err != nil {
		slog.Error("Failed to validate alliance memberships", slog.String("error", err.Error()))
	}

	return nil
}

// validateGroupMemberships validates memberships for a specific group type
func (gt *GroupTask) validateGroupMemberships(ctx context.Context, groupName, configKey string, validator func(map[string]any, []int) bool) error {
	// Get the group
	group, err := gt.groupService.GetGroupByName(ctx, groupName)
	if err != nil {
		return fmt.Errorf("failed to get %s group: %w", groupName, err)
	}

	members, err := gt.groupService.ListGroupMembers(ctx, group.ID.Hex())
	if err != nil {
		return fmt.Errorf("failed to get %s group members: %w", groupName, err)
	}

	slog.Info("Validating group memberships", 
		slog.String("group", groupName),
		slog.Int("member_count", len(members)))

	validatedCount := 0
	invalidCount := 0
	errorCount := 0

	// Get enabled IDs from configuration
	enabledIDs := config.GetEnvIntSlice(configKey)

	for _, member := range members {
		// Get character info from ESI
		charInfo, err := gt.eveClient.Character.GetCharacterInfo(ctx, member.CharacterID)
		if err != nil {
			slog.Warn("Failed to get character info from ESI",
				slog.Int("character_id", member.CharacterID),
				slog.String("error", err.Error()))
			errorCount++
			continue
		}

		valid := validator(charInfo, enabledIDs)

		// Update membership validation status
		err = gt.updateMembershipValidation(ctx, member.CharacterID, group.ID.Hex(), valid)
		if err != nil {
			slog.Error("Failed to update membership validation status",
				slog.Int("character_id", member.CharacterID),
				slog.String("error", err.Error()))
			errorCount++
			continue
		}

		if valid {
			validatedCount++
		} else {
			invalidCount++
			
			// Remove invalid members from group
			err = gt.groupService.RemoveGroupMember(ctx, group.ID.Hex(), member.CharacterID, 0) // System removal
			if err != nil {
				slog.Error("Failed to remove invalid member from group",
					slog.String("group", groupName),
					slog.Int("character_id", member.CharacterID),
					slog.String("error", err.Error()))
				errorCount++
			} else {
				slog.Info("Removed invalid member from group",
					slog.String("group", groupName),
					slog.Int("character_id", member.CharacterID))
			}
		}
	}

	slog.Info("Group membership validation completed",
		slog.String("group", groupName),
		slog.Int("validated", validatedCount),
		slog.Int("invalid", invalidCount),
		slog.Int("errors", errorCount))

	return nil
}


// updateMembershipValidation updates the validation status of a membership
func (gt *GroupTask) updateMembershipValidation(ctx context.Context, characterID int, groupID string, valid bool) error {
	collection := gt.groupService.mongodb.Database.Collection("group_memberships")
	
	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %w", err)
	}

	status := "valid"
	if !valid {
		status = "invalid"
	}

	filter := bson.M{
		"character_id": characterID,
		"group_id":     objectID,
	}
	
	update := bson.M{
		"$set": bson.M{
			"last_validated":    time.Now(),
			"validation_status": status,
		},
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update membership validation: %w", err)
	}

	return nil
}

// CleanupExpiredMemberships removes all expired group memberships
func (gt *GroupTask) CleanupExpiredMemberships(ctx context.Context) (int, error) {
	slog.Info("Starting expired membership cleanup task")

	count, err := gt.groupService.CleanupExpiredMemberships(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired memberships: %w", err)
	}

	// Invalidate all user permission caches after cleanup
	if err := gt.permissionService.InvalidateAllUserPermissions(ctx); err != nil {
		slog.Warn("Failed to invalidate permission cache after cleanup", 
			slog.String("error", err.Error()))
	}

	slog.Info("Expired membership cleanup completed", slog.Int("cleaned_up", count))
	return count, nil
}

// SyncDiscordRoles synchronizes group memberships with Discord roles
func (gt *GroupTask) SyncDiscordRoles(ctx context.Context) error {
	slog.Info("Starting Discord role synchronization task")

	if !gt.discordService.IsDiscordEnabled() {
		slog.Info("Discord service not configured, skipping Discord sync")
		return nil
	}

	// Get all groups with Discord roles configured
	groups, err := gt.groupService.ListGroups(ctx, true) // Include all groups
	if err != nil {
		return fmt.Errorf("failed to get groups for Discord sync: %w", err)
	}

	// Filter groups that have Discord roles
	var groupsWithDiscord []Group
	for _, group := range groups {
		if len(group.DiscordRoles) > 0 {
			groupsWithDiscord = append(groupsWithDiscord, group)
		}
	}

	if len(groupsWithDiscord) == 0 {
		slog.Info("No groups have Discord roles configured")
		return nil
	}

	// Use batch processing for efficiency
	err = gt.discordService.BatchProcessGroupRoles(ctx, groupsWithDiscord, gt.groupService)
	if err != nil {
		return fmt.Errorf("failed to batch process Discord roles: %w", err)
	}

	return nil
}


// AutoAssignNewUsers automatically assigns users to appropriate groups based on their profile
func (gt *GroupTask) AutoAssignNewUsers(ctx context.Context) error {
	slog.Info("Starting auto-assignment of new users")

	// Get all users who are only in the "full" group (newly authenticated users)
	// This would typically be called when a user first authenticates
	
	// For now, this is a placeholder - the actual implementation would be triggered
	// by the auth module when users authenticate
	
	slog.Info("Auto-assignment task completed")
	return nil
}

// GenerateGroupAnalytics generates analytics about group usage and permissions
func (gt *GroupTask) GenerateGroupAnalytics(ctx context.Context) error {
	slog.Info("Generating group analytics")

	analysis, err := gt.permissionService.AnalyzeGroupPermissions(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate group analytics: %w", err)
	}

	// Log summary analytics
	slog.Info("Group analytics summary",
		slog.Int("total_groups", analysis.TotalGroups),
		slog.Int("default_groups", analysis.DefaultGroups),
		slog.Int("custom_groups", analysis.CustomGroups))

	// Log detailed group information
	for _, summary := range analysis.GroupSummaries {
		slog.Info("Group summary",
			slog.String("group", summary.GroupName),
			slog.Bool("is_default", summary.IsDefault),
			slog.Int("members", summary.MemberCount),
			slog.Int("permissions", len(summary.Permissions)))
	}

	return nil
}

// ValidateGroupIntegrity checks for inconsistencies in group data
func (gt *GroupTask) ValidateGroupIntegrity(ctx context.Context) error {
	slog.Info("Starting group integrity validation")

	issues := 0

	// Check for orphaned memberships (memberships pointing to non-existent groups)
	issues += gt.checkOrphanedMemberships(ctx)

	// Check for invalid permission configurations
	issues += gt.checkInvalidPermissions(ctx)

	// Check for duplicate group memberships
	issues += gt.checkDuplicateMemberships(ctx)

	slog.Info("Group integrity validation completed", slog.Int("issues_found", issues))
	return nil
}

func (gt *GroupTask) checkOrphanedMemberships(ctx context.Context) int {
	collection := gt.groupService.mongodb.Database.Collection("group_memberships")
	
	// Use aggregation to find memberships with invalid group references
	pipeline := []bson.M{
		{"$lookup": bson.M{
			"from":         "groups",
			"localField":   "group_id",
			"foreignField": "_id",
			"as":           "group",
		}},
		{"$match": bson.M{
			"group": bson.M{"$size": 0}, // No matching group found
		}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		slog.Error("Failed to check orphaned memberships", slog.String("error", err.Error()))
		return 0
	}
	defer cursor.Close(ctx)

	orphanedCount := 0
	for cursor.Next(ctx) {
		var membership GroupMembership
		if err := cursor.Decode(&membership); err == nil {
			slog.Warn("Found orphaned membership",
				slog.String("membership_id", membership.ID.Hex()),
				slog.String("group_id", membership.GroupID.Hex()),
				slog.Int("character_id", membership.CharacterID))
			orphanedCount++
			
			// Remove orphaned membership
			_, err := collection.DeleteOne(ctx, bson.M{"_id": membership.ID})
			if err != nil {
				slog.Error("Failed to remove orphaned membership", slog.String("error", err.Error()))
			}
		}
	}

	if orphanedCount > 0 {
		slog.Info("Removed orphaned memberships", slog.Int("count", orphanedCount))
	}

	return orphanedCount
}

func (gt *GroupTask) checkInvalidPermissions(ctx context.Context) int {
	groups, err := gt.groupService.ListGroups(ctx, true)
	if err != nil {
		slog.Error("Failed to get groups for permission validation", slog.String("error", err.Error()))
		return 0
	}

	issueCount := 0
	for _, group := range groups {
		issues := gt.permissionService.ValidatePermissionStructure(group.Permissions)
		if len(issues) > 0 {
			slog.Warn("Group has invalid permissions",
				slog.String("group", group.Name),
				slog.Any("issues", issues))
			issueCount += len(issues)
		}
	}

	return issueCount
}

func (gt *GroupTask) checkDuplicateMemberships(ctx context.Context) int {
	collection := gt.groupService.mongodb.Database.Collection("group_memberships")
	
	// Find duplicate memberships (same character_id + group_id)
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id": bson.M{
				"character_id": "$character_id",
				"group_id":     "$group_id",
			},
			"count": bson.M{"$sum": 1},
			"docs":  bson.M{"$push": "$$ROOT"},
		}},
		{"$match": bson.M{
			"count": bson.M{"$gt": 1},
		}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		slog.Error("Failed to check duplicate memberships", slog.String("error", err.Error()))
		return 0
	}
	defer cursor.Close(ctx)

	duplicateCount := 0
	for cursor.Next(ctx) {
		var result struct {
			ID    bson.M              `bson:"_id"`
			Count int                 `bson:"count"`
			Docs  []GroupMembership   `bson:"docs"`
		}
		
		if err := cursor.Decode(&result); err == nil {
			slog.Warn("Found duplicate memberships",
				slog.Int("character_id", result.ID["character_id"].(int)),
				slog.String("group_id", result.ID["group_id"].(string)),
				slog.Int("duplicate_count", result.Count))
			
			// Keep the first membership, remove the rest
			for i := 1; i < len(result.Docs); i++ {
				_, err := collection.DeleteOne(ctx, bson.M{"_id": result.Docs[i].ID})
				if err != nil {
					slog.Error("Failed to remove duplicate membership", slog.String("error", err.Error()))
				} else {
					duplicateCount++
				}
			}
		}
	}

	if duplicateCount > 0 {
		slog.Info("Removed duplicate memberships", slog.Int("count", duplicateCount))
	}

	return duplicateCount
}

