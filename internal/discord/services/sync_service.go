package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-falcon/internal/discord/models"
)

// GroupsServiceInterface defines the interface for interacting with the groups service
type GroupsServiceInterface interface {
	GetUserGroups(ctx context.Context, userID string) ([]GroupInfo, error)
	GetGroupInfo(ctx context.Context, groupID string) (*GroupInfo, error)
}

// CharacterServiceInterface defines the interface for interacting with the character service
type CharacterServiceInterface interface {
	GetCharacterProfile(ctx context.Context, characterID int) (*CharacterProfile, error)
}

// CorporationServiceInterface defines the interface for interacting with the corporation service
type CorporationServiceInterface interface {
	GetCorporationInfo(ctx context.Context, corporationID int) (*CorporationInfo, error)
}

// UserServiceInterface defines the interface for interacting with the user service
type UserServiceInterface interface {
	GetUserByUserID(ctx context.Context, userID string) (*UserProfile, error)
}

// CharacterProfile represents character information
type CharacterProfile struct {
	CharacterID   int    `json:"character_id"`
	Name          string `json:"name"`
	CorporationID int    `json:"corporation_id"`
}

// CorporationInfo represents corporation information
type CorporationInfo struct {
	CorporationID int    `json:"corporation_id"`
	Name          string `json:"name"`
	Ticker        string `json:"ticker"`
}

// UserProfile represents user profile information
type UserProfile struct {
	UserID        string `json:"user_id"`
	CharacterID   int    `json:"character_id"`
	CharacterName string `json:"character_name"`
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
	repo               *Repository
	botService         *BotService
	groupsService      GroupsServiceInterface
	characterService   CharacterServiceInterface
	corporationService CorporationServiceInterface
	userService        UserServiceInterface
}

// NewSyncService creates a new sync service
func NewSyncService(repo *Repository, botService *BotService, groupsService GroupsServiceInterface, characterService CharacterServiceInterface, corporationService CorporationServiceInterface, userService UserServiceInterface) *SyncService {
	return &SyncService{
		repo:               repo,
		botService:         botService,
		groupsService:      groupsService,
		characterService:   characterService,
		corporationService: corporationService,
		userService:        userService,
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

	// Update nickname with corporation ticker prefix if roles were applied
	if !dryRun && (len(rolesToAdd) > 0 || len(rolesToRemove) > 0) {
		if err := s.updateDiscordNicknameWithTicker(ctx, discordUser, guildID, member, botToken); err != nil {
			// Don't fail the entire sync for nickname update failures, just log the error
			slog.WarnContext(ctx, "Failed to update Discord nickname with corporation ticker",
				"user_id", discordUser.UserID,
				"discord_id", discordUser.DiscordID,
				"guild_id", guildID,
				"error", err)
		}
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

// getUserCorporationTicker gets the corporation ticker for a user's main character
func (s *SyncService) getUserCorporationTicker(ctx context.Context, userID string) (string, error) {
	// Skip if required services are not available
	if s.userService == nil || s.characterService == nil || s.corporationService == nil {
		return "", fmt.Errorf("required services not available")
	}

	// Get user profile to find main character ID
	userProfile, err := s.userService.GetUserByUserID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user profile: %w", err)
	}

	if userProfile == nil {
		return "", fmt.Errorf("user profile not found")
	}

	// Get character information to find corporation ID
	characterProfile, err := s.characterService.GetCharacterProfile(ctx, userProfile.CharacterID)
	if err != nil {
		return "", fmt.Errorf("failed to get character profile: %w", err)
	}

	if characterProfile == nil {
		return "", fmt.Errorf("character profile not found")
	}

	// Get corporation information to get ticker
	corporationInfo, err := s.corporationService.GetCorporationInfo(ctx, characterProfile.CorporationID)
	if err != nil {
		return "", fmt.Errorf("failed to get corporation info: %w", err)
	}

	if corporationInfo == nil {
		return "", fmt.Errorf("corporation info not found")
	}

	return corporationInfo.Ticker, nil
}

// buildNicknameWithTicker builds a Discord nickname with corporation ticker prefix
func (s *SyncService) buildNicknameWithTicker(originalNickname, ticker string) string {
	// Remove existing ticker prefix if present (format: [TICK] Name)
	cleanNickname := originalNickname
	if len(originalNickname) > 0 && originalNickname[0] == '[' {
		if endIndex := strings.Index(originalNickname, "] "); endIndex != -1 {
			cleanNickname = strings.TrimSpace(originalNickname[endIndex+2:])
		}
	}

	// If no clean nickname, use character name from user context
	if cleanNickname == "" {
		cleanNickname = originalNickname
	}

	// Build new nickname with ticker prefix
	return fmt.Sprintf("[%s] %s", ticker, cleanNickname)
}

// updateDiscordNicknameWithTicker updates a Discord user's nickname with corporation ticker prefix
func (s *SyncService) updateDiscordNicknameWithTicker(ctx context.Context, discordUser *models.DiscordUser, guildID string, member *DiscordGuildMember, botToken string) error {
	// Get user's corporation ticker
	ticker, err := s.getUserCorporationTicker(ctx, discordUser.UserID)
	if err != nil {
		return fmt.Errorf("failed to get corporation ticker: %w", err)
	}

	if ticker == "" {
		return fmt.Errorf("empty corporation ticker")
	}

	// Get current nickname (use global name or username as fallback)
	currentNickname := ""
	if member.Nick != nil {
		currentNickname = *member.Nick
	} else if member.User.GlobalName != nil {
		currentNickname = *member.User.GlobalName
	} else {
		currentNickname = member.User.Username
	}

	// Build new nickname with ticker
	newNickname := s.buildNicknameWithTicker(currentNickname, ticker)

	// Only update if the nickname would change
	if (member.Nick != nil && *member.Nick == newNickname) ||
		(member.Nick == nil && newNickname == currentNickname) {
		slog.DebugContext(ctx, "Discord nickname already has correct ticker, skipping update",
			"user_id", discordUser.UserID,
			"discord_id", discordUser.DiscordID,
			"guild_id", guildID,
			"current_nickname", currentNickname,
			"ticker", ticker)
		return nil
	}

	// Update the nickname
	if err := s.botService.UpdateGuildMemberNickname(ctx, guildID, discordUser.DiscordID, newNickname, botToken); err != nil {
		return fmt.Errorf("failed to update Discord nickname: %w", err)
	}

	slog.InfoContext(ctx, "Updated Discord nickname with corporation ticker",
		"user_id", discordUser.UserID,
		"discord_id", discordUser.DiscordID,
		"guild_id", guildID,
		"old_nickname", currentNickname,
		"new_nickname", newNickname,
		"ticker", ticker)

	return nil
}
