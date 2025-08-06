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
	eveClient         *evegateway.Client
}

func NewGroupTask(groupService *GroupService, permissionService *PermissionService) *GroupTask {
	return &GroupTask{
		groupService:      groupService,
		permissionService: permissionService,
		eveClient:         evegateway.NewClient(),
	}
}

// ValidateCorporateMemberships validates all corporate group memberships against ESI data
func (gt *GroupTask) ValidateCorporateMemberships(ctx context.Context) error {
	slog.Info("Starting corporate membership validation task")
	
	// Get all members of the corporate group
	corporateGroup, err := gt.groupService.GetGroupByName(ctx, "corporate")
	if err != nil {
		return fmt.Errorf("failed to get corporate group: %w", err)
	}

	members, err := gt.groupService.ListGroupMembers(ctx, corporateGroup.ID.Hex())
	if err != nil {
		return fmt.Errorf("failed to get corporate group members: %w", err)
	}

	slog.Info("Validating corporate memberships", slog.Int("member_count", len(members)))

	validatedCount := 0
	invalidCount := 0
	errorCount := 0

	// Get enabled corporations and alliances from configuration
	enabledCorps := config.GetEnvIntSlice("ENABLED_CORPORATION_IDS")
	enabledAlliances := config.GetEnvIntSlice("ENABLED_ALLIANCE_IDS")

	for _, member := range members {
		valid, err := gt.validateMemberCorporateStatus(ctx, member.CharacterID, enabledCorps, enabledAlliances)
		if err != nil {
			slog.Warn("Failed to validate member corporate status",
				slog.Int("character_id", member.CharacterID),
				slog.String("error", err.Error()))
			errorCount++
			continue
		}

		// Update membership validation status
		err = gt.updateMembershipValidation(ctx, member.CharacterID, corporateGroup.ID.Hex(), valid)
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
			
			// Remove invalid members from corporate group
			err = gt.groupService.RemoveGroupMember(ctx, corporateGroup.ID.Hex(), member.CharacterID, 0) // System removal
			if err != nil {
				slog.Error("Failed to remove invalid corporate member",
					slog.Int("character_id", member.CharacterID),
					slog.String("error", err.Error()))
				errorCount++
			} else {
				slog.Info("Removed invalid corporate member",
					slog.Int("character_id", member.CharacterID))
			}
		}
	}

	slog.Info("Corporate membership validation completed",
		slog.Int("validated", validatedCount),
		slog.Int("invalid", invalidCount),
		slog.Int("errors", errorCount))

	return nil
}

// validateMemberCorporateStatus checks if a character is in an enabled corporation or alliance
func (gt *GroupTask) validateMemberCorporateStatus(ctx context.Context, characterID int, enabledCorps, enabledAlliances []int) (bool, error) {
	// Get character's current corporation and alliance from ESI
	charInfo, err := gt.eveClient.Character.GetCharacterInfo(ctx, characterID)
	if err != nil {
		return false, fmt.Errorf("failed to get character info from ESI: %w", err)
	}

	// The character info is returned as map[string]any, so we need to extract the fields
	corporationID, ok := charInfo["corporation_id"].(float64)
	if !ok {
		return false, fmt.Errorf("invalid corporation_id in character info")
	}

	// Check if character's corporation is in the enabled list
	for _, corpID := range enabledCorps {
		if int(corporationID) == corpID {
			return true, nil
		}
	}

	// Check if character's alliance is in the enabled list
	if allianceIDVal, exists := charInfo["alliance_id"]; exists {
		if allianceID, ok := allianceIDVal.(float64); ok {
			for _, enabledAllianceID := range enabledAlliances {
				if int(allianceID) == enabledAllianceID {
					return true, nil
				}
			}
		}
	}

	return false, nil
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
func (gt *GroupTask) CleanupExpiredMemberships(ctx context.Context) error {
	slog.Info("Starting expired membership cleanup task")

	count, err := gt.groupService.CleanupExpiredMemberships(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired memberships: %w", err)
	}

	// Invalidate all user permission caches after cleanup
	if err := gt.permissionService.InvalidateAllUserPermissions(ctx); err != nil {
		slog.Warn("Failed to invalidate permission cache after cleanup", 
			slog.String("error", err.Error()))
	}

	slog.Info("Expired membership cleanup completed", slog.Int("cleaned_up", count))
	return nil
}

// SyncDiscordRoles synchronizes group memberships with Discord roles
func (gt *GroupTask) SyncDiscordRoles(ctx context.Context) error {
	slog.Info("Starting Discord role synchronization task")

	discordServiceURL := config.GetEnv("DISCORD_SERVICE_URL", "")
	if discordServiceURL == "" {
		slog.Info("Discord service URL not configured, skipping Discord sync")
		return nil
	}

	// Get all groups with Discord roles configured
	groups, err := gt.groupService.ListGroups(ctx, true) // Include all groups
	if err != nil {
		return fmt.Errorf("failed to get groups for Discord sync: %w", err)
	}

	syncCount := 0
	errorCount := 0

	for _, group := range groups {
		if len(group.DiscordRoles) == 0 {
			continue // No Discord roles configured for this group
		}

		// Get members of this group
		members, err := gt.groupService.ListGroupMembers(ctx, group.ID.Hex())
		if err != nil {
			slog.Error("Failed to get group members for Discord sync",
				slog.String("group", group.Name),
				slog.String("error", err.Error()))
			errorCount++
			continue
		}

		// Sync roles for each member across all servers
		for _, member := range members {
			err := gt.syncMemberDiscordRoles(ctx, member.CharacterID, group.DiscordRoles, discordServiceURL)
			if err != nil {
				slog.Warn("Failed to sync Discord roles for member",
					slog.Int("character_id", member.CharacterID),
					slog.String("group", group.Name),
					slog.String("error", err.Error()))
				errorCount++
			} else {
				syncCount++
			}
		}
	}

	slog.Info("Discord role synchronization completed",
		slog.Int("synced", syncCount),
		slog.Int("errors", errorCount))

	return nil
}

// syncMemberDiscordRoles syncs Discord roles for a single member across multiple servers
func (gt *GroupTask) syncMemberDiscordRoles(ctx context.Context, characterID int, roles []DiscordRole, serviceURL string) error {
	// This would call the Discord service API to assign roles
	// The actual implementation would depend on the Discord service API structure
	
	slog.Debug("Syncing Discord roles for character",
		slog.Int("character_id", characterID),
		slog.Int("role_count", len(roles)))

	// TODO: Implement actual Discord API calls based on the Discord service API
	// For now, just simulate the sync
	
	for _, role := range roles {
		slog.Debug("Would assign Discord role",
			slog.Int("character_id", characterID),
			slog.String("server_id", role.ServerID),
			slog.String("role_name", role.RoleName))
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

