package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-falcon/internal/discord/models"
)

// GroupsServiceInterface defines the interface for interacting with the groups service
type GroupsServiceInterface interface {
	GetUserGroups(ctx context.Context, userID string) ([]GroupInfo, error)
	GetGroupInfo(ctx context.Context, groupID string) (*GroupInfo, error)
}

// GroupInfo represents group information from the groups service
type GroupInfo struct {
	ID          primitive.ObjectID `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Type        string             `json:"type"`
	IsActive    bool               `json:"is_active"`
}

// SyncResult represents the result of a role synchronization operation
type SyncResult struct {
	UserID         string   `json:"user_id"`
	DiscordID      string   `json:"discord_id"`
	GuildID        string   `json:"guild_id"`
	RolesAdded     []string `json:"roles_added"`
	RolesRemoved   []string `json:"roles_removed"`
	Success        bool     `json:"success"`
	Error          string   `json:"error,omitempty"`
	ProcessingTime int64    `json:"processing_time_ms"`
}

// SyncStats represents overall synchronization statistics
type SyncStats struct {
	TotalUsers        int64        `json:"total_users"`
	ProcessedUsers    int64        `json:"processed_users"`
	SuccessfulUsers   int64        `json:"successful_users"`
	FailedUsers       int64        `json:"failed_users"`
	TotalRolesAdded   int64        `json:"total_roles_added"`
	TotalRolesRemoved int64        `json:"total_roles_removed"`
	ProcessingTime    int64        `json:"processing_time_ms"`
	Results           []SyncResult `json:"results"`
	Errors            []string     `json:"errors"`
}

// SyncService handles Discord role synchronization
type SyncService struct {
	repo          *Repository
	botService    *BotService
	groupsService GroupsServiceInterface
}

// NewSyncService creates a new sync service
func NewSyncService(repo *Repository, botService *BotService, groupsService GroupsServiceInterface) *SyncService {
	return &SyncService{
		repo:          repo,
		botService:    botService,
		groupsService: groupsService,
	}
}

// SyncAllGuilds synchronizes all enabled guilds
func (s *SyncService) SyncAllGuilds(ctx context.Context, dryRun bool) (*SyncStats, error) {
	startTime := time.Now()
	stats := &SyncStats{
		Results: make([]SyncResult, 0),
		Errors:  make([]string, 0),
	}

	// Get all enabled guild configurations
	filter := map[string]interface{}{
		"is_enabled": true,
	}

	guilds, _, err := s.repo.ListGuildConfigs(ctx, filter, 1, 100) // Get all enabled guilds
	if err != nil {
		return nil, fmt.Errorf("failed to get guild configurations: %w", err)
	}

	slog.InfoContext(ctx, "Starting sync for all guilds",
		"guild_count", len(guilds),
		"dry_run", dryRun)

	for _, guild := range guilds {
		guildStats, err := s.SyncGuild(ctx, guild.GuildID, dryRun)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to sync guild %s: %v", guild.GuildName, err)
			stats.Errors = append(stats.Errors, errorMsg)
			slog.ErrorContext(ctx, errorMsg, "guild_id", guild.GuildID, "error", err)
			continue
		}

		// Aggregate stats
		stats.TotalUsers += guildStats.TotalUsers
		stats.ProcessedUsers += guildStats.ProcessedUsers
		stats.SuccessfulUsers += guildStats.SuccessfulUsers
		stats.FailedUsers += guildStats.FailedUsers
		stats.TotalRolesAdded += guildStats.TotalRolesAdded
		stats.TotalRolesRemoved += guildStats.TotalRolesRemoved
		stats.Results = append(stats.Results, guildStats.Results...)
		stats.Errors = append(stats.Errors, guildStats.Errors...)
	}

	stats.ProcessingTime = time.Since(startTime).Milliseconds()

	slog.InfoContext(ctx, "Completed sync for all guilds",
		"total_users", stats.TotalUsers,
		"successful_users", stats.SuccessfulUsers,
		"failed_users", stats.FailedUsers,
		"roles_added", stats.TotalRolesAdded,
		"roles_removed", stats.TotalRolesRemoved,
		"processing_time_ms", stats.ProcessingTime,
		"dry_run", dryRun)

	return stats, nil
}

// SyncGuild synchronizes a specific guild
func (s *SyncService) SyncGuild(ctx context.Context, guildID string, dryRun bool) (*SyncStats, error) {
	startTime := time.Now()
	stats := &SyncStats{
		Results: make([]SyncResult, 0),
		Errors:  make([]string, 0),
	}

	// Get guild configuration
	guildConfig, err := s.repo.GetGuildConfigByGuildID(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild configuration: %w", err)
	}
	if guildConfig == nil {
		return nil, fmt.Errorf("guild configuration not found")
	}
	if !guildConfig.IsEnabled {
		return nil, fmt.Errorf("guild sync is disabled")
	}

	// Decrypt bot token
	botToken := s.botService.DecryptBotToken(guildConfig.BotToken)

	// Validate bot token and permissions
	if err := s.botService.ValidateBotToken(ctx, botToken); err != nil {
		return nil, fmt.Errorf("invalid bot token: %w", err)
	}

	hasPermissions, err := s.botService.CheckBotPermissions(ctx, guildID, botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to check bot permissions: %w", err)
	}
	if !hasPermissions {
		return nil, fmt.Errorf("bot does not have required permissions in guild")
	}

	// Get role mappings for this guild
	roleMappings, err := s.repo.GetRoleMappingsByGuildID(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role mappings: %w", err)
	}

	if len(roleMappings) == 0 {
		slog.WarnContext(ctx, "No role mappings found for guild", "guild_id", guildID)
		return stats, nil
	}

	// Get all Discord users
	filter := map[string]interface{}{
		"is_active": true,
	}

	discordUsers, _, err := s.repo.ListDiscordUsers(ctx, filter, 1, 1000) // Get all active users
	if err != nil {
		return nil, fmt.Errorf("failed to get Discord users: %w", err)
	}

	stats.TotalUsers = int64(len(discordUsers))

	// Create sync status record
	syncStatus := &models.DiscordSyncStatus{
		GuildID:        guildID,
		LastSyncAt:     time.Now(),
		UsersProcessed: 0,
		UsersSucceeded: 0,
		UsersFailed:    0,
		RolesAdded:     0,
		RolesRemoved:   0,
		Status:         "running",
	}

	if !dryRun {
		if err := s.repo.CreateSyncStatus(ctx, syncStatus); err != nil {
			slog.WarnContext(ctx, "Failed to create sync status", "error", err)
		}
	}

	// Sync each user
	for _, user := range discordUsers {
		userResult := s.syncUser(ctx, user, guildID, roleMappings, botToken, dryRun)
		stats.Results = append(stats.Results, userResult)
		stats.ProcessedUsers++

		if userResult.Success {
			stats.SuccessfulUsers++
			stats.TotalRolesAdded += int64(len(userResult.RolesAdded))
			stats.TotalRolesRemoved += int64(len(userResult.RolesRemoved))
		} else {
			stats.FailedUsers++
			stats.Errors = append(stats.Errors, userResult.Error)
		}

		// Rate limiting - small delay between users
		time.Sleep(100 * time.Millisecond)
	}

	stats.ProcessingTime = time.Since(startTime).Milliseconds()

	// Update sync status
	if !dryRun && syncStatus.ID != primitive.NilObjectID {
		update := map[string]interface{}{
			"users_processed": stats.ProcessedUsers,
			"users_succeeded": stats.SuccessfulUsers,
			"users_failed":    stats.FailedUsers,
			"roles_added":     stats.TotalRolesAdded,
			"roles_removed":   stats.TotalRolesRemoved,
			"status":          "completed",
			"duration":        stats.ProcessingTime,
		}

		if len(stats.Errors) > 0 {
			update["errors"] = stats.Errors
			if stats.SuccessfulUsers == 0 {
				update["status"] = "failed"
			}
		}

		if err := s.repo.UpdateSyncStatus(ctx, syncStatus.ID, update); err != nil {
			slog.WarnContext(ctx, "Failed to update sync status", "error", err)
		}
	}

	slog.InfoContext(ctx, "Completed guild sync",
		"guild_id", guildID,
		"processed_users", stats.ProcessedUsers,
		"successful_users", stats.SuccessfulUsers,
		"failed_users", stats.FailedUsers,
		"roles_added", stats.TotalRolesAdded,
		"roles_removed", stats.TotalRolesRemoved,
		"processing_time_ms", stats.ProcessingTime,
		"dry_run", dryRun)

	return stats, nil
}

// SyncUser synchronizes a specific user's roles in all guilds or a specific guild
func (s *SyncService) SyncUser(ctx context.Context, userID string, guildID *string, dryRun bool) (*SyncStats, error) {
	startTime := time.Now()
	stats := &SyncStats{
		Results: make([]SyncResult, 0),
		Errors:  make([]string, 0),
	}

	// Get Discord users for this Go Falcon user
	discordUsers, err := s.repo.GetDiscordUsersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Discord users: %w", err)
	}

	if len(discordUsers) == 0 {
		return nil, fmt.Errorf("no Discord accounts linked to user")
	}

	// Get guilds to sync
	var guildsToSync []*models.DiscordGuildConfig

	if guildID != nil {
		// Sync specific guild
		guildConfig, err := s.repo.GetGuildConfigByGuildID(ctx, *guildID)
		if err != nil {
			return nil, fmt.Errorf("failed to get guild configuration: %w", err)
		}
		if guildConfig != nil && guildConfig.IsEnabled {
			guildsToSync = append(guildsToSync, guildConfig)
		}
	} else {
		// Sync all enabled guilds
		filter := map[string]interface{}{
			"is_enabled": true,
		}
		guilds, _, err := s.repo.ListGuildConfigs(ctx, filter, 1, 100)
		if err != nil {
			return nil, fmt.Errorf("failed to get guild configurations: %w", err)
		}
		guildsToSync = guilds
	}

	stats.TotalUsers = int64(len(discordUsers) * len(guildsToSync))

	// Sync user in each guild
	for _, guild := range guildsToSync {
		// Get role mappings for this guild
		roleMappings, err := s.repo.GetRoleMappingsByGuildID(ctx, guild.GuildID)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to get role mappings for guild %s: %v", guild.GuildName, err)
			stats.Errors = append(stats.Errors, errorMsg)
			continue
		}

		if len(roleMappings) == 0 {
			continue // Skip guild with no role mappings
		}

		// Decrypt bot token
		botToken := s.botService.DecryptBotToken(guild.BotToken)

		// Sync each Discord account
		for _, discordUser := range discordUsers {
			if !discordUser.IsActive {
				continue
			}

			userResult := s.syncUser(ctx, discordUser, guild.GuildID, roleMappings, botToken, dryRun)
			stats.Results = append(stats.Results, userResult)
			stats.ProcessedUsers++

			if userResult.Success {
				stats.SuccessfulUsers++
				stats.TotalRolesAdded += int64(len(userResult.RolesAdded))
				stats.TotalRolesRemoved += int64(len(userResult.RolesRemoved))
			} else {
				stats.FailedUsers++
				stats.Errors = append(stats.Errors, userResult.Error)
			}

			// Rate limiting
			time.Sleep(100 * time.Millisecond)
		}
	}

	stats.ProcessingTime = time.Since(startTime).Milliseconds()

	slog.InfoContext(ctx, "Completed user sync",
		"user_id", userID,
		"guild_id", guildID,
		"processed_users", stats.ProcessedUsers,
		"successful_users", stats.SuccessfulUsers,
		"failed_users", stats.FailedUsers,
		"roles_added", stats.TotalRolesAdded,
		"roles_removed", stats.TotalRolesRemoved,
		"processing_time_ms", stats.ProcessingTime,
		"dry_run", dryRun)

	return stats, nil
}

// syncUser synchronizes a single Discord user's roles in a specific guild
func (s *SyncService) syncUser(ctx context.Context, discordUser *models.DiscordUser, guildID string, roleMappings []*models.DiscordRoleMapping, botToken string, dryRun bool) SyncResult {
	startTime := time.Now()

	result := SyncResult{
		UserID:         discordUser.UserID,
		DiscordID:      discordUser.DiscordID,
		GuildID:        guildID,
		RolesAdded:     make([]string, 0),
		RolesRemoved:   make([]string, 0),
		Success:        false,
		ProcessingTime: 0,
	}

	defer func() {
		result.ProcessingTime = time.Since(startTime).Milliseconds()
	}()

	// Get user's current groups from groups service
	if s.groupsService == nil {
		result.Error = "groups service not available"
		return result
	}

	userGroups, err := s.groupsService.GetUserGroups(ctx, discordUser.UserID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get user groups: %v", err)
		return result
	}

	// Create map of user's group IDs for quick lookup
	userGroupIDs := make(map[string]bool)
	for _, group := range userGroups {
		userGroupIDs[group.ID.Hex()] = group.IsActive
	}

	// Determine which Discord roles the user should have
	requiredRoleIDs := make([]string, 0)
	for _, mapping := range roleMappings {
		if !mapping.IsActive {
			continue
		}

		// Check if user is in this group
		if hasGroup, exists := userGroupIDs[mapping.GroupID.Hex()]; exists && hasGroup {
			requiredRoleIDs = append(requiredRoleIDs, mapping.DiscordRoleID)
		}
	}

	// Get user's current Discord roles in this guild
	member, err := s.botService.GetGuildMember(ctx, guildID, discordUser.DiscordID, botToken)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get guild member: %v", err)
		return result
	}

	if member == nil {
		// User is not a member of this guild
		if len(requiredRoleIDs) > 0 {
			// User should have roles but isn't in guild - attempt to add them
			if !dryRun {
				// Decrypt user's access token for guild join
				accessToken := s.decryptAccessToken(discordUser.AccessToken)
				if accessToken == "" {
					result.Error = "user is not a member of the guild and access token unavailable for auto-join"
					return result
				}

				// Attempt to add user to guild with their required roles
				err := s.botService.AddGuildMember(ctx, guildID, discordUser.DiscordID, accessToken, botToken, requiredRoleIDs)
				if err != nil {
					result.Error = fmt.Sprintf("user is not a member of the guild and auto-join failed: %v", err)
					return result
				}

				// Record roles as added (since they were assigned during join)
				for _, roleID := range requiredRoleIDs {
					result.RolesAdded = append(result.RolesAdded, roleID)
				}

				slog.InfoContext(ctx, "Auto-joined user to guild during sync",
					"user_id", discordUser.UserID,
					"discord_id", discordUser.DiscordID,
					"guild_id", guildID,
					"roles_assigned", len(requiredRoleIDs))

				result.Success = true
				return result
			} else {
				// In dry run, just indicate what would happen
				result.RolesAdded = requiredRoleIDs
				result.Success = true
				return result
			}
		} else {
			result.Success = true // User not in guild and doesn't need roles
		}
		return result
	}

	// Create map of current roles for comparison
	currentRoles := make(map[string]bool)
	for _, roleID := range member.Roles {
		currentRoles[roleID] = true
	}

	// Create map of required roles
	requiredRoles := make(map[string]bool)
	for _, roleID := range requiredRoleIDs {
		requiredRoles[roleID] = true
	}

	// Determine roles to add and remove
	rolesToAdd := make([]string, 0)
	rolesToRemove := make([]string, 0)

	// Find roles to add (in required but not in current)
	for roleID := range requiredRoles {
		if !currentRoles[roleID] {
			rolesToAdd = append(rolesToAdd, roleID)
		}
	}

	// Find managed roles to remove (in current but not in required, and managed by us)
	managedRoleIDs := make(map[string]bool)
	for _, mapping := range roleMappings {
		if mapping.IsActive {
			managedRoleIDs[mapping.DiscordRoleID] = true
		}
	}

	for roleID := range currentRoles {
		if managedRoleIDs[roleID] && !requiredRoles[roleID] {
			rolesToRemove = append(rolesToRemove, roleID)
		}
	}

	// Apply changes if not dry run
	if !dryRun && (len(rolesToAdd) > 0 || len(rolesToRemove) > 0) {
		// Add roles
		for _, roleID := range rolesToAdd {
			if err := s.botService.AddGuildMemberRole(ctx, guildID, discordUser.DiscordID, roleID, botToken); err != nil {
				result.Error = fmt.Sprintf("failed to add role %s: %v", roleID, err)
				return result
			}
			result.RolesAdded = append(result.RolesAdded, roleID)
		}

		// Remove roles
		for _, roleID := range rolesToRemove {
			if err := s.botService.RemoveGuildMemberRole(ctx, guildID, discordUser.DiscordID, roleID, botToken); err != nil {
				result.Error = fmt.Sprintf("failed to remove role %s: %v", roleID, err)
				return result
			}
			result.RolesRemoved = append(result.RolesRemoved, roleID)
		}
	} else if dryRun {
		// In dry run mode, just record what would be changed
		result.RolesAdded = rolesToAdd
		result.RolesRemoved = rolesToRemove
	}

	result.Success = true

	if len(rolesToAdd) > 0 || len(rolesToRemove) > 0 {
		slog.InfoContext(ctx, "Synced user roles",
			"user_id", discordUser.UserID,
			"discord_id", discordUser.DiscordID,
			"guild_id", guildID,
			"roles_added", len(rolesToAdd),
			"roles_removed", len(rolesToRemove),
			"dry_run", dryRun)
	}

	return result
}

// decryptAccessToken decrypts a stored access token
// TODO: Implement proper decryption - currently returns token as-is
func (s *SyncService) decryptAccessToken(encryptedToken string) string {
	// For now, return as-is since encryption is not implemented
	// In production, this should use proper decryption
	return encryptedToken
}

// GetSyncStatus gets recent synchronization status
func (s *SyncService) GetSyncStatus(ctx context.Context, guildID string, limit int) ([]*models.DiscordSyncStatus, error) {
	return s.repo.GetRecentSyncStatus(ctx, guildID, limit)
}
