package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-falcon/internal/discord/dto"
	"go-falcon/internal/discord/models"
	"go-falcon/pkg/database"
)

// Service is the main Discord service that coordinates all operations
type Service struct {
	repo               *Repository
	oauthService       *OAuthService
	botService         *BotService
	syncService        *SyncService
	groupsService      GroupsServiceInterface
	characterService   CharacterServiceInterface
	corporationService CorporationServiceInterface
	userService        UserServiceInterface
}

// NewService creates a new Discord service
func NewService(db *database.MongoDB, groupsService GroupsServiceInterface, characterService CharacterServiceInterface, corporationService CorporationServiceInterface, userService UserServiceInterface) *Service {
	repo := NewRepository(db)
	oauthService := NewOAuthService(repo)
	botService := NewBotService(repo)
	syncService := NewSyncService(repo, botService, groupsService, characterService, corporationService, userService)

	return &Service{
		repo:               repo,
		oauthService:       oauthService,
		botService:         botService,
		syncService:        syncService,
		groupsService:      groupsService,
		characterService:   characterService,
		corporationService: corporationService,
		userService:        userService,
	}
}

// Initialize initializes the Discord service (creates indexes)
func (s *Service) Initialize(ctx context.Context) error {
	slog.InfoContext(ctx, "Initializing Discord service")

	// Create database indexes
	if err := s.repo.CreateIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	slog.InfoContext(ctx, "Discord service initialized successfully")
	return nil
}

// Authentication Methods

// GetAuthURL generates a Discord OAuth authorization URL
func (s *Service) GetAuthURL(ctx context.Context, input *dto.GetDiscordAuthURLInput, userID *string) (*dto.DiscordAuthURLOutput, error) {
	var linkUserID *string
	if input.LinkToUser && userID != nil {
		linkUserID = userID
	}

	authURL, state, err := s.oauthService.GenerateAuthURL(ctx, input.LinkToUser, linkUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth URL: %w", err)
	}

	return &dto.DiscordAuthURLOutput{
		Body: dto.DiscordAuthURLResponse{
			AuthURL: authURL,
			State:   state,
		},
	}, nil
}

// HandleCallback handles the Discord OAuth callback
func (s *Service) HandleCallback(ctx context.Context, input *dto.DiscordCallbackInput, currentUserID *string) (*dto.DiscordMessageOutput, string, error) {
	// Process OAuth callback
	userInfo, tokenResponse, storedUserID, err := s.oauthService.HandleCallback(ctx, input.Code, input.State)
	if err != nil {
		return nil, "", fmt.Errorf("failed to handle OAuth callback: %w", err)
	}

	// Determine which Go Falcon user to link to
	targetUserID := storedUserID
	if targetUserID == nil && currentUserID != nil {
		targetUserID = currentUserID
	}
	if targetUserID == nil {
		return nil, "", fmt.Errorf("Discord account linking requires authentication. Please log in with your Go Falcon account first, then restart the Discord OAuth flow")
	}

	// Create or update Discord user record
	_, err = s.oauthService.CreateOrUpdateDiscordUser(ctx, *targetUserID, userInfo, tokenResponse)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create/update Discord user: %w", err)
	}

	// Auto-join user to mapped guilds
	autoJoinResult, err := s.JoinUserToMappedGuilds(ctx, *targetUserID, userInfo.ID, tokenResponse.AccessToken)
	if err != nil {
		// Log error but don't fail the entire OAuth process
		slog.WarnContext(ctx, "Auto-join to guilds failed, but OAuth linking succeeded",
			"user_id", *targetUserID,
			"discord_id", userInfo.ID,
			"error", err)
	} else {
		slog.InfoContext(ctx, "Auto-join to guilds completed",
			"user_id", *targetUserID,
			"discord_id", userInfo.ID,
			"guilds_joined", autoJoinResult.SuccessCount,
			"guilds_failed", autoJoinResult.FailureCount)
	}

	message := "Discord account linked successfully"
	if storedUserID == nil {
		message = "Discord authentication completed successfully"
	}

	// Add guild join info to success message if applicable
	if autoJoinResult != nil && autoJoinResult.SuccessCount > 0 {
		if autoJoinResult.SuccessCount == 1 {
			message += fmt.Sprintf(" and automatically joined %d Discord server", autoJoinResult.SuccessCount)
		} else {
			message += fmt.Sprintf(" and automatically joined %d Discord servers", autoJoinResult.SuccessCount)
		}
	}

	slog.InfoContext(ctx, "Discord OAuth callback completed",
		"user_id", *targetUserID,
		"discord_id", userInfo.ID,
		"username", userInfo.Username)

	return &dto.DiscordMessageOutput{
		Body: dto.DiscordMessageResponse{
			Message: message,
		},
	}, *targetUserID, nil
}

// LinkAccount links a Discord account to an existing Go Falcon user
func (s *Service) LinkAccount(ctx context.Context, input *dto.LinkDiscordAccountInput, userID string) (*dto.DiscordMessageOutput, error) {
	// Validate Discord access token
	userInfo, err := s.oauthService.ValidateToken(ctx, input.Body.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to validate Discord token: %w", err)
	}

	// Create token response structure
	tokenResponse := &DiscordTokenResponse{
		AccessToken:  input.Body.AccessToken,
		RefreshToken: input.Body.RefreshToken,
		ExpiresIn:    3600, // Default 1 hour expiry if not specified
		TokenType:    "Bearer",
		Scope:        "identify guilds",
	}

	// Create or update Discord user record
	_, err = s.oauthService.CreateOrUpdateDiscordUser(ctx, userID, userInfo, tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to link Discord account: %w", err)
	}

	// Auto-join user to mapped guilds
	autoJoinResult, err := s.JoinUserToMappedGuilds(ctx, userID, userInfo.ID, tokenResponse.AccessToken)
	if err != nil {
		// Log error but don't fail the entire linking process
		slog.WarnContext(ctx, "Auto-join to guilds failed, but account linking succeeded",
			"user_id", userID,
			"discord_id", userInfo.ID,
			"error", err)
	} else {
		slog.InfoContext(ctx, "Auto-join to guilds completed",
			"user_id", userID,
			"discord_id", userInfo.ID,
			"guilds_joined", autoJoinResult.SuccessCount,
			"guilds_failed", autoJoinResult.FailureCount)
	}

	slog.InfoContext(ctx, "Discord account linked via API",
		"user_id", userID,
		"discord_id", userInfo.ID,
		"username", userInfo.Username)

	message := "Discord account linked successfully"
	// Add guild join info to success message if applicable
	if autoJoinResult != nil && autoJoinResult.SuccessCount > 0 {
		if autoJoinResult.SuccessCount == 1 {
			message += fmt.Sprintf(" and automatically joined %d Discord server", autoJoinResult.SuccessCount)
		} else {
			message += fmt.Sprintf(" and automatically joined %d Discord servers", autoJoinResult.SuccessCount)
		}
	}

	return &dto.DiscordMessageOutput{
		Body: dto.DiscordMessageResponse{
			Message: message,
		},
	}, nil
}

// UnlinkAccount unlinks a Discord account from a Go Falcon user
func (s *Service) UnlinkAccount(ctx context.Context, input *dto.UnlinkDiscordAccountInput, userID string) (*dto.DiscordMessageOutput, error) {
	// Get Discord user record
	discordUser, err := s.repo.GetDiscordUserByDiscordID(ctx, input.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Discord user: %w", err)
	}

	if discordUser == nil {
		return nil, fmt.Errorf("Discord account not found")
	}

	// Verify ownership
	if discordUser.UserID != userID {
		return nil, fmt.Errorf("Discord account belongs to different user")
	}

	// Revoke Discord token if possible
	accessToken := s.oauthService.decryptToken(discordUser.AccessToken)
	if accessToken != "" {
		if err := s.oauthService.RevokeToken(ctx, accessToken); err != nil {
			slog.WarnContext(ctx, "Failed to revoke Discord token", "error", err)
		}
	}

	// Delete Discord user record
	if err := s.repo.DeleteDiscordUser(ctx, discordUser.ID); err != nil {
		return nil, fmt.Errorf("failed to unlink Discord account: %w", err)
	}

	slog.InfoContext(ctx, "Discord account unlinked",
		"user_id", userID,
		"discord_id", input.DiscordID)

	return &dto.DiscordMessageOutput{
		Body: dto.DiscordMessageResponse{
			Message: "Discord account unlinked successfully",
		},
	}, nil
}

// GetAuthStatus gets the Discord authentication status for a user
func (s *Service) GetAuthStatus(ctx context.Context, userID *string) (*dto.DiscordAuthStatusOutput, error) {
	authenticated := userID != nil
	isLinked := false
	var discordUsers []dto.DiscordUserResponse

	if userID != nil {
		// Get linked Discord accounts
		users, err := s.repo.GetDiscordUsersByUserID(ctx, *userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get Discord users: %w", err)
		}

		isLinked = len(users) > 0

		// Convert to response format
		for _, user := range users {
			discordUsers = append(discordUsers, dto.DiscordUserResponse{
				ID:          user.ID.Hex(),
				UserID:      user.UserID,
				DiscordID:   user.DiscordID,
				Username:    user.Username,
				GlobalName:  user.GlobalName,
				Avatar:      user.Avatar,
				IsActive:    user.IsActive,
				LinkedAt:    user.LinkedAt,
				UpdatedAt:   user.UpdatedAt,
				TokenExpiry: user.TokenExpiry,
			})
		}
	}

	response := &dto.DiscordAuthStatusOutput{
		Body: dto.DiscordAuthStatusResponse{
			IsLinked:      isLinked,
			DiscordUsers:  discordUsers,
			Authenticated: authenticated,
		},
	}

	if userID != nil {
		response.Body.UserID = *userID
	}

	return response, nil
}

// User Management Methods

// GetDiscordUser gets a Discord user by Go Falcon user ID
func (s *Service) GetDiscordUser(ctx context.Context, input *dto.GetDiscordUserInput) (*dto.DiscordUserOutput, error) {
	users, err := s.repo.GetDiscordUsersByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Discord users: %w", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("no Discord accounts found for user")
	}

	// Return the first (primary) Discord account
	user := users[0]
	return &dto.DiscordUserOutput{
		Body: dto.DiscordUserResponse{
			ID:          user.ID.Hex(),
			UserID:      user.UserID,
			DiscordID:   user.DiscordID,
			Username:    user.Username,
			GlobalName:  user.GlobalName,
			Avatar:      user.Avatar,
			IsActive:    user.IsActive,
			LinkedAt:    user.LinkedAt,
			UpdatedAt:   user.UpdatedAt,
			TokenExpiry: user.TokenExpiry,
		},
	}, nil
}

// ListDiscordUsers lists Discord users with filtering and pagination
func (s *Service) ListDiscordUsers(ctx context.Context, input *dto.ListDiscordUsersInput) (*dto.ListDiscordUsersOutput, error) {
	// Build filter
	filter := bson.M{}
	if input.IsActive != "" {
		isActive := input.IsActive == "true"
		filter["is_active"] = isActive
	}
	if input.DiscordID != "" {
		filter["discord_id"] = input.DiscordID
	}

	// Set defaults
	page := input.Page
	if page == 0 {
		page = 1
	}
	limit := input.Limit
	if limit == 0 {
		limit = 20
	}

	users, total, err := s.repo.ListDiscordUsers(ctx, filter, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list Discord users: %w", err)
	}

	// Convert to response format
	userResponses := make([]dto.DiscordUserResponse, len(users))
	for i, user := range users {
		userResponses[i] = dto.DiscordUserResponse{
			ID:          user.ID.Hex(),
			UserID:      user.UserID,
			DiscordID:   user.DiscordID,
			Username:    user.Username,
			GlobalName:  user.GlobalName,
			Avatar:      user.Avatar,
			IsActive:    user.IsActive,
			LinkedAt:    user.LinkedAt,
			UpdatedAt:   user.UpdatedAt,
			TokenExpiry: user.TokenExpiry,
		}
	}

	return &dto.ListDiscordUsersOutput{
		Body: dto.ListDiscordUsersResponse{
			Users: userResponses,
			Total: total,
			Page:  page,
			Limit: limit,
		},
	}, nil
}

// Guild Management Methods

// CreateGuildConfig creates a new Discord guild configuration
func (s *Service) CreateGuildConfig(ctx context.Context, input *dto.CreateGuildConfigInput, createdBy int64) (*dto.DiscordGuildConfigOutput, error) {
	// Validate bot token
	if err := s.botService.ValidateBotToken(ctx, input.Body.BotToken); err != nil {
		return nil, fmt.Errorf("invalid bot token: %w", err)
	}

	// Check if guild already exists
	existing, err := s.repo.GetGuildConfigByGuildID(ctx, input.Body.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing guild: %w", err)
	}
	if existing != nil {
		return nil, huma.Error409Conflict("Guild configuration already exists for this Discord guild")
	}

	// Create guild configuration
	config := &models.DiscordGuildConfig{
		GuildID:   input.Body.GuildID,
		GuildName: input.Body.GuildName,
		BotToken:  s.botService.EncryptBotToken(input.Body.BotToken),
		IsEnabled: true,
		CreatedBy: &createdBy,
	}

	if err := s.repo.CreateGuildConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create guild configuration: %w", err)
	}

	return s.guildConfigToOutput(ctx, config), nil
}

// GetGuildConfig gets a guild configuration
func (s *Service) GetGuildConfig(ctx context.Context, input *dto.GetGuildConfigInput) (*dto.DiscordGuildConfigOutput, error) {
	config, err := s.repo.GetGuildConfigByGuildID(ctx, input.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild configuration: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("guild configuration not found")
	}

	return s.guildConfigToOutput(ctx, config), nil
}

// UpdateGuildConfig updates a guild configuration
func (s *Service) UpdateGuildConfig(ctx context.Context, input *dto.UpdateGuildConfigInput) (*dto.DiscordGuildConfigOutput, error) {
	// Check if guild exists
	existing, err := s.repo.GetGuildConfigByGuildID(ctx, input.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild configuration: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("guild configuration not found")
	}

	// Build update
	update := bson.M{}
	if input.Body.GuildName != nil {
		update["guild_name"] = *input.Body.GuildName
	}
	if input.Body.BotToken != nil {
		// Validate new bot token
		if err := s.botService.ValidateBotToken(ctx, *input.Body.BotToken); err != nil {
			return nil, fmt.Errorf("invalid bot token: %w", err)
		}
		update["bot_token"] = s.botService.EncryptBotToken(*input.Body.BotToken)
	}
	if input.Body.IsEnabled != nil {
		update["is_enabled"] = *input.Body.IsEnabled
	}

	if len(update) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	if err := s.repo.UpdateGuildConfig(ctx, input.GuildID, update); err != nil {
		return nil, fmt.Errorf("failed to update guild configuration: %w", err)
	}

	// Get updated configuration
	updated, err := s.repo.GetGuildConfigByGuildID(ctx, input.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated guild configuration: %w", err)
	}

	return s.guildConfigToOutput(ctx, updated), nil
}

// DeleteGuildConfig deletes a guild configuration
func (s *Service) DeleteGuildConfig(ctx context.Context, input *dto.DeleteGuildConfigInput) (*dto.DiscordSuccessOutput, error) {
	// Check if guild exists
	existing, err := s.repo.GetGuildConfigByGuildID(ctx, input.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild configuration: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("guild configuration not found")
	}

	if err := s.repo.DeleteGuildConfig(ctx, input.GuildID); err != nil {
		return nil, fmt.Errorf("failed to delete guild configuration: %w", err)
	}

	return &dto.DiscordSuccessOutput{
		Body: dto.DiscordSuccessResponse{
			Message: "Guild configuration deleted successfully",
		},
	}, nil
}

// ListGuildConfigs lists guild configurations
func (s *Service) ListGuildConfigs(ctx context.Context, input *dto.ListGuildConfigsInput) (*dto.ListDiscordGuildConfigsOutput, error) {
	// Build filter
	filter := bson.M{}
	if input.IsEnabled != "" {
		isEnabled := input.IsEnabled == "true"
		filter["is_enabled"] = isEnabled
	}

	// Set defaults
	page := input.Page
	if page == 0 {
		page = 1
	}
	limit := input.Limit
	if limit == 0 {
		limit = 20
	}

	configs, total, err := s.repo.ListGuildConfigs(ctx, filter, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list guild configurations: %w", err)
	}

	// Convert to response format
	configResponses := make([]dto.DiscordGuildConfigResponse, len(configs))
	for i, config := range configs {
		configResponses[i] = s.guildConfigToOutput(ctx, config).Body
	}

	return &dto.ListDiscordGuildConfigsOutput{
		Body: dto.ListDiscordGuildConfigsResponse{
			Guilds: configResponses,
			Total:  total,
			Page:   page,
			Limit:  limit,
		},
	}, nil
}

// GetGuildRoles gets all Discord roles from a guild
func (s *Service) GetGuildRoles(ctx context.Context, input *dto.GetGuildRolesInput) (*dto.DiscordGuildRolesOutput, error) {
	// Get guild configuration to retrieve bot token
	config, err := s.repo.GetGuildConfigByGuildID(ctx, input.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild configuration: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("guild configuration not found")
	}

	// Decrypt bot token
	botToken := s.botService.DecryptBotToken(config.BotToken)

	// Fetch roles from Discord API
	discordRoles, err := s.botService.GetGuildRoles(ctx, input.GuildID, botToken)
	if err != nil {
		// Check if this is a "guild not found" error from Discord API
		if strings.Contains(err.Error(), "guild not found") {
			return nil, huma.Error404NotFound("Discord guild not found. The guild may not exist or the bot may not have access to it.")
		}
		return nil, fmt.Errorf("failed to fetch guild roles from Discord: %w", err)
	}

	// Convert Discord roles to response format
	roles := make([]dto.DiscordRoleResponse, len(discordRoles))
	for i, role := range discordRoles {
		roles[i] = dto.DiscordRoleResponse{
			ID:          role.ID,
			Name:        role.Name,
			Color:       role.Color,
			Hoist:       role.Hoist,
			Position:    role.Position,
			Permissions: role.Permissions,
			Managed:     role.Managed,
			Mentionable: role.Mentionable,
		}
	}

	return &dto.DiscordGuildRolesOutput{
		Body: dto.DiscordGuildRolesResponse{
			GuildID: input.GuildID,
			Roles:   roles,
		},
	}, nil
}

// Role Mapping Methods

// CreateRoleMapping creates a new Discord role mapping
func (s *Service) CreateRoleMapping(ctx context.Context, input *dto.CreateRoleMappingInput, createdBy int64) (*dto.DiscordRoleMappingOutput, error) {
	// Check if guild exists
	guildConfig, err := s.repo.GetGuildConfigByGuildID(ctx, input.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild configuration: %w", err)
	}
	if guildConfig == nil {
		return nil, fmt.Errorf("guild configuration not found")
	}

	// Validate group ID format (MongoDB ObjectID)
	groupObjectID, err := primitive.ObjectIDFromHex(input.Body.GroupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID format: %w", err)
	}

	// Check if role mapping already exists (using ListRoleMappings with filter)
	filter := bson.M{
		"guild_id":  input.GuildID,
		"group_id":  groupObjectID,
		"is_active": true,
	}
	existingMappings, _, err := s.repo.ListRoleMappings(ctx, filter, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing role mapping: %w", err)
	}
	if len(existingMappings) > 0 {
		return nil, huma.Error409Conflict("A role mapping already exists for this guild and group combination. Each group can only have one role mapping per Discord guild.")
	}

	// Create role mapping
	mapping := &models.DiscordRoleMapping{
		GuildID:         input.GuildID,
		GroupID:         groupObjectID,
		GroupName:       "", // Will be resolved later from groups service if needed
		DiscordRoleID:   input.Body.DiscordRoleID,
		DiscordRoleName: input.Body.DiscordRoleName,
		IsActive:        true,
		CreatedBy:       &createdBy,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.CreateRoleMapping(ctx, mapping); err != nil {
		return nil, fmt.Errorf("failed to create role mapping: %w", err)
	}

	// Trigger guild sync to assign the new role to users who qualify
	go func() {
		syncCtx := context.Background() // Use background context for async operation
		slog.InfoContext(syncCtx, "Triggering guild sync after role mapping creation",
			"guild_id", mapping.GuildID,
			"mapping_id", mapping.ID.Hex(),
			"role_id", mapping.DiscordRoleID,
			"role_name", mapping.DiscordRoleName,
			"group_id", mapping.GroupID.Hex())

		if _, err := s.syncService.SyncGuild(syncCtx, mapping.GuildID, false); err != nil {
			slog.ErrorContext(syncCtx, "Failed to sync guild after role mapping creation",
				"guild_id", mapping.GuildID,
				"error", err)
		} else {
			slog.InfoContext(syncCtx, "Successfully synced guild after role mapping creation",
				"guild_id", mapping.GuildID)
		}
	}()

	return &dto.DiscordRoleMappingOutput{
		Body: s.roleMappingToResponse(ctx, mapping),
	}, nil
}

// GetRoleMapping gets a specific role mapping
func (s *Service) GetRoleMapping(ctx context.Context, input *dto.GetRoleMappingInput) (*dto.DiscordRoleMappingOutput, error) {
	mappingObjectID, err := primitive.ObjectIDFromHex(input.MappingID)
	if err != nil {
		return nil, fmt.Errorf("invalid mapping ID format: %w", err)
	}

	mapping, err := s.repo.GetRoleMappingByID(ctx, mappingObjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role mapping: %w", err)
	}
	if mapping == nil {
		return nil, fmt.Errorf("role mapping not found")
	}

	return &dto.DiscordRoleMappingOutput{
		Body: s.roleMappingToResponse(ctx, mapping),
	}, nil
}

// UpdateRoleMapping updates a role mapping
func (s *Service) UpdateRoleMapping(ctx context.Context, input *dto.UpdateRoleMappingInput) (*dto.DiscordRoleMappingOutput, error) {
	mappingObjectID, err := primitive.ObjectIDFromHex(input.MappingID)
	if err != nil {
		return nil, fmt.Errorf("invalid mapping ID format: %w", err)
	}

	// Check if mapping exists
	existing, err := s.repo.GetRoleMappingByID(ctx, mappingObjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role mapping: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("role mapping not found")
	}

	// Build update
	update := bson.M{"updated_at": time.Now()}
	if input.Body.DiscordRoleID != nil {
		update["discord_role_id"] = *input.Body.DiscordRoleID
	}
	if input.Body.DiscordRoleName != nil {
		update["discord_role_name"] = *input.Body.DiscordRoleName
	}
	if input.Body.IsActive != nil {
		update["is_active"] = *input.Body.IsActive
	}

	if len(update) <= 1 { // Only updated_at
		return nil, fmt.Errorf("no fields to update")
	}

	if err := s.repo.UpdateRoleMapping(ctx, mappingObjectID, update); err != nil {
		return nil, fmt.Errorf("failed to update role mapping: %w", err)
	}

	// Get updated mapping
	updated, err := s.repo.GetRoleMappingByID(ctx, mappingObjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated role mapping: %w", err)
	}

	// Check what changed to determine appropriate actions
	wasDisabled := existing.IsActive && !updated.IsActive
	roleChanged := existing.DiscordRoleID != updated.DiscordRoleID

	// Trigger role updates based on what changed
	go func() {
		syncCtx := context.Background() // Use background context for async operation

		if wasDisabled {
			// Mapping was disabled - explicitly remove this role from all users first
			slog.InfoContext(syncCtx, "Role mapping disabled - removing role from all users",
				"guild_id", updated.GuildID,
				"mapping_id", input.MappingID,
				"role_id", updated.DiscordRoleID,
				"role_name", updated.DiscordRoleName)

			// Step 1: Remove the disabled role from all users who have it
			if err := s.removeRoleFromAllUsers(syncCtx, updated.GuildID, updated.DiscordRoleID, updated.DiscordRoleName); err != nil {
				slog.ErrorContext(syncCtx, "Failed to remove disabled role from users",
					"guild_id", updated.GuildID,
					"role_id", updated.DiscordRoleID,
					"error", err)
			}
		} else if roleChanged && existing.IsActive {
			// Discord role ID changed for an active mapping - remove old role from all users
			slog.InfoContext(syncCtx, "Discord role changed - removing old role from all users",
				"guild_id", updated.GuildID,
				"mapping_id", input.MappingID,
				"old_role_id", existing.DiscordRoleID,
				"old_role_name", existing.DiscordRoleName,
				"new_role_id", updated.DiscordRoleID,
				"new_role_name", updated.DiscordRoleName)

			// Step 1: Remove the old role from all users who have it
			if err := s.removeRoleFromAllUsers(syncCtx, updated.GuildID, existing.DiscordRoleID, existing.DiscordRoleName); err != nil {
				slog.ErrorContext(syncCtx, "Failed to remove old role from users after role change",
					"guild_id", updated.GuildID,
					"old_role_id", existing.DiscordRoleID,
					"error", err)
			}
		}

		// Step 2: Run full guild sync to ensure everything is consistent
		slog.InfoContext(syncCtx, "Triggering guild sync after role mapping update",
			"guild_id", updated.GuildID,
			"mapping_id", input.MappingID,
			"role_id", updated.DiscordRoleID,
			"role_name", updated.DiscordRoleName,
			"is_active", updated.IsActive,
			"was_disabled", wasDisabled,
			"role_changed", roleChanged)

		if _, err := s.syncService.SyncGuild(syncCtx, updated.GuildID, false); err != nil {
			slog.ErrorContext(syncCtx, "Failed to sync guild after role mapping update",
				"guild_id", updated.GuildID,
				"error", err)
		} else {
			slog.InfoContext(syncCtx, "Successfully completed role update and guild sync",
				"guild_id", updated.GuildID,
				"was_disabled", wasDisabled,
				"role_changed", roleChanged)
		}
	}()

	return &dto.DiscordRoleMappingOutput{
		Body: s.roleMappingToResponse(ctx, updated),
	}, nil
}

// DeleteRoleMapping deletes a role mapping
func (s *Service) DeleteRoleMapping(ctx context.Context, input *dto.DeleteRoleMappingInput) (*dto.DiscordSuccessOutput, error) {
	mappingObjectID, err := primitive.ObjectIDFromHex(input.MappingID)
	if err != nil {
		return nil, fmt.Errorf("invalid mapping ID format: %w", err)
	}

	// Check if mapping exists
	existing, err := s.repo.GetRoleMappingByID(ctx, mappingObjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role mapping: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("role mapping not found")
	}

	// Store details for cleanup and syncing after deletion
	guildID := existing.GuildID
	deletedRoleID := existing.DiscordRoleID
	deletedRoleName := existing.DiscordRoleName

	if err := s.repo.DeleteRoleMapping(ctx, mappingObjectID); err != nil {
		return nil, fmt.Errorf("failed to delete role mapping: %w", err)
	}

	// Immediately remove the deleted role from all users in the guild, then trigger full sync
	go func() {
		syncCtx := context.Background() // Use background context for async operation
		slog.InfoContext(syncCtx, "Removing deleted role from all users and triggering guild sync",
			"guild_id", guildID,
			"deleted_mapping_id", input.MappingID,
			"deleted_role_id", deletedRoleID,
			"deleted_role_name", deletedRoleName)

		// Step 1: Remove the deleted role from all users who have it
		if err := s.removeRoleFromAllUsers(syncCtx, guildID, deletedRoleID, deletedRoleName); err != nil {
			slog.ErrorContext(syncCtx, "Failed to remove deleted role from users",
				"guild_id", guildID,
				"role_id", deletedRoleID,
				"error", err)
		}

		// Step 2: Run full guild sync to ensure everything is consistent
		if _, err := s.syncService.SyncGuild(syncCtx, guildID, false); err != nil {
			slog.ErrorContext(syncCtx, "Failed to sync guild after role mapping deletion",
				"guild_id", guildID,
				"error", err)
		} else {
			slog.InfoContext(syncCtx, "Successfully completed role removal and guild sync",
				"guild_id", guildID)
		}
	}()

	return &dto.DiscordSuccessOutput{
		Body: dto.DiscordSuccessResponse{
			Message: "Role mapping deleted successfully. Role synchronization initiated.",
		},
	}, nil
}

// ListRoleMappings lists role mappings for a guild with filtering
func (s *Service) ListRoleMappings(ctx context.Context, input *dto.ListRoleMappingsInput) (*dto.ListDiscordRoleMappingsOutput, error) {
	// Get guild configuration to get guild name
	guildConfig, err := s.repo.GetGuildConfigByGuildID(ctx, input.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild configuration: %w", err)
	}
	if guildConfig == nil {
		return nil, fmt.Errorf("guild configuration not found")
	}

	// Build filter
	filter := bson.M{"guild_id": input.GuildID}
	if input.IsActive != "" {
		isActive := input.IsActive == "true"
		filter["is_active"] = isActive
	}
	if input.GroupID != "" {
		groupObjectID, err := primitive.ObjectIDFromHex(input.GroupID)
		if err != nil {
			return nil, fmt.Errorf("invalid group ID format: %w", err)
		}
		filter["group_id"] = groupObjectID
	}

	// Set defaults
	page := input.Page
	if page == 0 {
		page = 1
	}
	limit := input.Limit
	if limit == 0 {
		limit = 20
	}

	mappings, total, err := s.repo.ListRoleMappings(ctx, filter, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list role mappings: %w", err)
	}

	// Convert to response format with enriched group information
	mappingResponses := make([]dto.DiscordRoleMappingResponse, len(mappings))
	for i, mapping := range mappings {
		mappingResponses[i] = s.roleMappingToResponse(ctx, mapping)
	}

	return &dto.ListDiscordRoleMappingsOutput{
		Body: dto.ListDiscordRoleMappingsResponse{
			GuildID:   input.GuildID,
			GuildName: guildConfig.GuildName,
			Mappings:  mappingResponses,
			Total:     total,
			Page:      page,
			Limit:     limit,
		},
	}, nil
}

// Sync Methods

// TriggerManualSync triggers manual synchronization
func (s *Service) TriggerManualSync(ctx context.Context, input *dto.ManualSyncInput) (*dto.ManualSyncOutput, error) {
	var stats *SyncStats
	var err error

	if input.Body.GuildID != "" {
		// Sync specific guild
		stats, err = s.syncService.SyncGuild(ctx, input.Body.GuildID, input.Body.DryRun)
	} else if len(input.Body.UserIDs) > 0 {
		// Sync specific users
		allStats := &SyncStats{
			Results: make([]SyncResult, 0),
			Errors:  make([]string, 0),
		}

		for _, userID := range input.Body.UserIDs {
			userStats, userErr := s.syncService.SyncUser(ctx, userID, nil, input.Body.DryRun)
			if userErr != nil {
				allStats.Errors = append(allStats.Errors, fmt.Sprintf("User %s: %v", userID, userErr))
				continue
			}

			// Aggregate stats
			allStats.TotalUsers += userStats.TotalUsers
			allStats.ProcessedUsers += userStats.ProcessedUsers
			allStats.SuccessfulUsers += userStats.SuccessfulUsers
			allStats.FailedUsers += userStats.FailedUsers
			allStats.TotalRolesAdded += userStats.TotalRolesAdded
			allStats.TotalRolesRemoved += userStats.TotalRolesRemoved
			allStats.ProcessingTime += userStats.ProcessingTime
			allStats.Results = append(allStats.Results, userStats.Results...)
			allStats.Errors = append(allStats.Errors, userStats.Errors...)
		}

		stats = allStats
	} else {
		// Sync all guilds
		stats, err = s.syncService.SyncAllGuilds(ctx, input.Body.DryRun)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to trigger sync: %w", err)
	}

	message := fmt.Sprintf("Sync completed: %d users processed, %d successful, %d failed, %d roles added, %d roles removed",
		stats.ProcessedUsers, stats.SuccessfulUsers, stats.FailedUsers, stats.TotalRolesAdded, stats.TotalRolesRemoved)

	return &dto.ManualSyncOutput{
		Body: dto.ManualSyncResponse{
			Message:           message,
			IsAsync:           false,
			EstimatedDuration: fmt.Sprintf("%dms", stats.ProcessingTime),
		},
	}, nil
}

// SyncUser synchronizes a specific user
func (s *Service) SyncUser(ctx context.Context, input *dto.SyncUserInput) (*dto.ManualSyncOutput, error) {
	guildID := input.Body.GuildID
	var guildPtr *string
	if guildID != "" {
		guildPtr = &guildID
	}

	stats, err := s.syncService.SyncUser(ctx, input.UserID, guildPtr, input.Body.DryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to sync user: %w", err)
	}

	message := fmt.Sprintf("User sync completed: %d accounts processed, %d successful, %d failed, %d roles added, %d roles removed",
		stats.ProcessedUsers, stats.SuccessfulUsers, stats.FailedUsers, stats.TotalRolesAdded, stats.TotalRolesRemoved)

	return &dto.ManualSyncOutput{
		Body: dto.ManualSyncResponse{
			Message:           message,
			IsAsync:           false,
			EstimatedDuration: fmt.Sprintf("%dms", stats.ProcessingTime),
		},
	}, nil
}

// GetSyncStatus gets synchronization status
func (s *Service) GetSyncStatus(ctx context.Context, input *dto.GetSyncStatusInput) (*dto.DiscordSyncStatusOutput, error) {
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}

	recentSyncs, err := s.syncService.GetSyncStatus(ctx, input.GuildID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	// Convert to response format
	syncResults := make([]dto.DiscordSyncResultResponse, len(recentSyncs))
	for i, sync := range recentSyncs {
		syncResults[i] = dto.DiscordSyncResultResponse{
			ID:             sync.ID.Hex(),
			GuildID:        sync.GuildID,
			Status:         sync.Status,
			UsersProcessed: sync.UsersProcessed,
			UsersSucceeded: sync.UsersSucceeded,
			UsersFailed:    sync.UsersFailed,
			RolesAdded:     sync.RolesAdded,
			RolesRemoved:   sync.RolesRemoved,
			Duration:       sync.Duration,
			Errors:         sync.Errors,
			LastSyncAt:     sync.LastSyncAt,
			CreatedAt:      sync.CreatedAt,
		}
	}

	currentStatus := "idle"
	if len(recentSyncs) > 0 && recentSyncs[0].Status == "running" {
		currentStatus = "running"
	}

	return &dto.DiscordSyncStatusOutput{
		Body: dto.DiscordSyncStatusResponse{
			CurrentStatus: currentStatus,
			RecentSyncs:   syncResults,
			TotalSyncs:    int64(len(recentSyncs)),
		},
	}, nil
}

// Status and Health Methods

// GetStatus returns the Discord module status
func (s *Service) GetStatus(ctx context.Context) *dto.DiscordStatusResponse {
	// Check database connectivity
	if err := s.repo.CheckHealth(ctx); err != nil {
		return &dto.DiscordStatusResponse{
			Module:  "discord",
			Status:  "unhealthy",
			Message: "Database connection failed: " + err.Error(),
		}
	}

	// Get some basic stats
	filter := map[string]interface{}{"is_enabled": true}
	guilds, _, _ := s.repo.ListGuildConfigs(ctx, filter, 1, 100)

	userFilter := map[string]interface{}{"is_active": true}
	users, _, _ := s.repo.ListDiscordUsers(ctx, userFilter, 1, 1)

	// Get last sync time
	recentSyncs, _ := s.syncService.GetSyncStatus(ctx, "", 1)
	var lastSyncAt *time.Time
	if len(recentSyncs) > 0 {
		lastSyncAt = &recentSyncs[0].LastSyncAt
	}

	return &dto.DiscordStatusResponse{
		Module:            "discord",
		Status:            "healthy",
		Message:           "All systems operational",
		ConfiguredGuilds:  int64(len(guilds)),
		ActiveUsers:       int64(len(users)),
		LastSyncAt:        lastSyncAt,
		DatabaseConnected: true,
		DiscordAPIHealthy: true, // Could add actual Discord API health check
	}
}

// Helper methods

// guildConfigToOutput converts a guild config model to output DTO
func (s *Service) guildConfigToOutput(ctx context.Context, config *models.DiscordGuildConfig) *dto.DiscordGuildConfigOutput {
	// Convert role mappings
	roleMappings := make([]dto.DiscordRoleMappingResponse, len(config.RoleMappings))
	for i, mapping := range config.RoleMappings {
		roleMappings[i] = s.roleMappingToResponse(ctx, &mapping)
	}

	return &dto.DiscordGuildConfigOutput{
		Body: dto.DiscordGuildConfigResponse{
			ID:           config.ID.Hex(),
			GuildID:      config.GuildID,
			GuildName:    config.GuildName,
			IsEnabled:    config.IsEnabled,
			RoleMappings: roleMappings,
			CreatedBy:    config.CreatedBy,
			CreatedAt:    config.CreatedAt,
			UpdatedAt:    config.UpdatedAt,
		},
	}
}

// RefreshExpiringTokens refreshes Discord tokens that are expiring soon (for scheduler)
func (s *Service) RefreshExpiringTokens(ctx context.Context, batchSize int) (int, int, error) {
	return s.oauthService.RefreshExpiringTokens(ctx, batchSize)
}

// PeriodicSync performs periodic role synchronization (for scheduler)
func (s *Service) PeriodicSync(ctx context.Context) error {
	slog.InfoContext(ctx, "Starting periodic Discord role synchronization")

	stats, err := s.syncService.SyncAllGuilds(ctx, false)
	if err != nil {
		return fmt.Errorf("periodic sync failed: %w", err)
	}

	slog.InfoContext(ctx, "Completed periodic Discord role synchronization",
		"processed_users", stats.ProcessedUsers,
		"successful_users", stats.SuccessfulUsers,
		"failed_users", stats.FailedUsers,
		"roles_added", stats.TotalRolesAdded,
		"roles_removed", stats.TotalRolesRemoved,
		"processing_time_ms", stats.ProcessingTime)

	return nil
}

// Auto-Join Methods

// JoinUserToMappedGuilds automatically joins user to all guilds where they have role mappings
func (s *Service) JoinUserToMappedGuilds(ctx context.Context, userID, discordUserID, accessToken string) (*dto.GuildAutoJoinResponse, error) {
	startTime := time.Now()

	slog.InfoContext(ctx, "Starting auto-join process for user",
		"user_id", userID,
		"discord_user_id", discordUserID)

	// 1. Get user's Go Falcon groups
	userGroups, err := s.groupsService.GetUserGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}

	slog.InfoContext(ctx, "Retrieved user groups",
		"user_id", userID,
		"group_count", len(userGroups))

	// 2. Convert group IDs to ObjectIDs
	groupIDs := make([]primitive.ObjectID, len(userGroups))
	for i, group := range userGroups {
		groupIDs[i] = group.ID
	}

	// 3. Get role mappings for user's groups
	roleMappings, err := s.repo.GetRoleMappingsByGroupIDs(ctx, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get role mappings: %w", err)
	}

	slog.InfoContext(ctx, "Found role mappings",
		"user_id", userID,
		"mapping_count", len(roleMappings))

	// 4. Group mappings by guild
	guildMappings := make(map[string][]string) // guildID -> roleIDs
	guildNames := make(map[string]string)      // guildID -> guildName

	for _, mapping := range roleMappings {
		guildMappings[mapping.GuildID] = append(guildMappings[mapping.GuildID], mapping.DiscordRoleID)
		// Store guild name from first mapping encountered for this guild
		if _, exists := guildNames[mapping.GuildID]; !exists {
			// Get guild name from config
			if guildConfig, err := s.repo.GetGuildConfigByGuildID(ctx, mapping.GuildID); err == nil && guildConfig != nil {
				guildNames[mapping.GuildID] = guildConfig.GuildName
			} else {
				guildNames[mapping.GuildID] = "Unknown Guild"
			}
		}
	}

	// 5. Process each guild
	var guildsJoined []dto.GuildJoinResult
	var guildsFailed []dto.GuildJoinResult

	for guildID, roleIDs := range guildMappings {
		result := s.processGuildJoin(ctx, guildID, guildNames[guildID], discordUserID, accessToken, roleIDs)

		if result.Status == "failed" {
			guildsFailed = append(guildsFailed, result)
		} else {
			guildsJoined = append(guildsJoined, result)
		}
	}

	slog.InfoContext(ctx, "Auto-join process completed",
		"user_id", userID,
		"total_guilds", len(guildMappings),
		"successful_joins", len(guildsJoined),
		"failed_joins", len(guildsFailed),
		"processing_time_ms", time.Since(startTime).Milliseconds())

	return &dto.GuildAutoJoinResponse{
		UserID:        userID,
		DiscordUserID: discordUserID,
		GuildsJoined:  guildsJoined,
		GuildsFailed:  guildsFailed,
		TotalGuilds:   len(guildMappings),
		SuccessCount:  len(guildsJoined),
		FailureCount:  len(guildsFailed),
		ProcessedAt:   time.Now(),
	}, nil
}

// processGuildJoin attempts to join a user to a specific guild with their roles
func (s *Service) processGuildJoin(ctx context.Context, guildID, guildName, discordUserID, accessToken string, roleIDs []string) dto.GuildJoinResult {
	// Get guild configuration for bot token
	guildConfig, err := s.repo.GetGuildConfigByGuildID(ctx, guildID)
	if err != nil {
		return dto.GuildJoinResult{
			GuildID:   guildID,
			GuildName: guildName,
			Status:    "failed",
			Error:     fmt.Sprintf("Failed to get guild config: %v", err),
		}
	}

	if guildConfig == nil {
		return dto.GuildJoinResult{
			GuildID:   guildID,
			GuildName: guildName,
			Status:    "failed",
			Error:     "Guild configuration not found",
		}
	}

	if !guildConfig.IsEnabled {
		return dto.GuildJoinResult{
			GuildID:   guildID,
			GuildName: guildName,
			Status:    "failed",
			Error:     "Guild sync is disabled",
		}
	}

	// Decrypt bot token
	botToken := s.botService.DecryptBotToken(guildConfig.BotToken)

	// Get role names for display
	roleNames := make([]string, 0, len(roleIDs))
	if roles, err := s.botService.GetGuildRoles(ctx, guildID, botToken); err == nil {
		roleNameMap := make(map[string]string)
		for _, role := range roles {
			roleNameMap[role.ID] = role.Name
		}
		for _, roleID := range roleIDs {
			if roleName, exists := roleNameMap[roleID]; exists {
				roleNames = append(roleNames, roleName)
			}
		}
	}

	// Attempt to add user to guild
	slog.InfoContext(ctx, "DEBUG: Attempting to add user to Discord guild",
		"guild_id", guildID,
		"guild_name", guildName,
		"discord_user_id", discordUserID,
		"role_ids", roleIDs,
		"has_access_token", accessToken != "",
		"has_bot_token", botToken != "")

	err = s.botService.AddGuildMember(ctx, guildID, discordUserID, accessToken, botToken, roleIDs)
	if err != nil {
		slog.ErrorContext(ctx, "DEBUG: Discord auto-join failed",
			"guild_id", guildID,
			"guild_name", guildName,
			"discord_user_id", discordUserID,
			"error", err.Error(),
			"role_ids", roleIDs)
		return dto.GuildJoinResult{
			GuildID:   guildID,
			GuildName: guildName,
			Status:    "failed",
			RoleIDs:   roleIDs,
			Error:     err.Error(),
		}
	}

	// Check if user was added or already a member based on the lack of error
	// The AddGuildMember method logs whether the user was added or already a member
	return dto.GuildJoinResult{
		GuildID:       guildID,
		GuildName:     guildName,
		Status:        "joined", // Could be "joined" or "already_member" based on HTTP status
		RolesAssigned: roleNames,
		RoleIDs:       roleIDs,
	}
}

// removeRoleFromAllUsers removes a specific Discord role from all users in a guild
func (s *Service) removeRoleFromAllUsers(ctx context.Context, guildID, roleID, roleName string) error {
	// Get guild configuration to get bot token
	guildConfig, err := s.repo.GetGuildConfigByGuildID(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild configuration: %w", err)
	}
	if guildConfig == nil || !guildConfig.IsEnabled {
		return fmt.Errorf("guild configuration not found or disabled")
	}

	// Decrypt bot token
	botToken := s.botService.DecryptBotToken(guildConfig.BotToken)
	if botToken == "" {
		return fmt.Errorf("failed to decrypt bot token")
	}

	// Get all active Discord users
	filter := map[string]interface{}{
		"is_active": true,
	}
	discordUsers, _, err := s.repo.ListDiscordUsers(ctx, filter, 1, 1000) // Process up to 1000 users
	if err != nil {
		return fmt.Errorf("failed to get Discord users: %w", err)
	}

	removedCount := 0
	errorCount := 0

	for _, discordUser := range discordUsers {
		// Get guild member to check current roles
		member, err := s.botService.GetGuildMember(ctx, guildID, discordUser.DiscordID, botToken)
		if err != nil {
			slog.WarnContext(ctx, "Failed to get guild member, skipping",
				"user_id", discordUser.UserID,
				"discord_id", discordUser.DiscordID,
				"guild_id", guildID,
				"error", err)
			continue
		}

		if member == nil {
			// User is not a member of this guild
			continue
		}

		// Check if user has the role to be removed
		hasRole := false
		for _, memberRoleID := range member.Roles {
			if memberRoleID == roleID {
				hasRole = true
				break
			}
		}

		if !hasRole {
			// User doesn't have this role, skip
			continue
		}

		// Remove the role
		if err := s.botService.RemoveGuildMemberRole(ctx, guildID, discordUser.DiscordID, roleID, botToken); err != nil {
			slog.ErrorContext(ctx, "Failed to remove role from user",
				"user_id", discordUser.UserID,
				"discord_id", discordUser.DiscordID,
				"guild_id", guildID,
				"role_id", roleID,
				"role_name", roleName,
				"error", err)
			errorCount++
			continue
		}

		removedCount++
		slog.InfoContext(ctx, "Removed deleted role from user",
			"user_id", discordUser.UserID,
			"discord_id", discordUser.DiscordID,
			"guild_id", guildID,
			"role_id", roleID,
			"role_name", roleName)

		// Rate limiting between operations
		time.Sleep(100 * time.Millisecond)
	}

	slog.InfoContext(ctx, "Completed role removal from all users",
		"guild_id", guildID,
		"role_id", roleID,
		"role_name", roleName,
		"users_processed", len(discordUsers),
		"roles_removed", removedCount,
		"errors", errorCount)

	if errorCount > 0 {
		return fmt.Errorf("removed role from %d users but encountered %d errors", removedCount, errorCount)
	}

	return nil
}

// Helper method to convert role mapping to response with enriched group data
func (s *Service) roleMappingToResponse(ctx context.Context, mapping *models.DiscordRoleMapping) dto.DiscordRoleMappingResponse {
	response := dto.DiscordRoleMappingResponse{
		ID:              mapping.ID.Hex(),
		GuildID:         mapping.GuildID,
		GroupID:         mapping.GroupID.Hex(),
		GroupName:       mapping.GroupName,
		DiscordRoleID:   mapping.DiscordRoleID,
		DiscordRoleName: mapping.DiscordRoleName,
		IsActive:        mapping.IsActive,
		CreatedBy:       mapping.CreatedBy,
		CreatedAt:       mapping.CreatedAt,
		UpdatedAt:       mapping.UpdatedAt,
	}

	// Try to get group details including description
	if s.groupsService != nil {
		if groupInfo, err := s.groupsService.GetGroupInfo(ctx, mapping.GroupID.Hex()); err == nil && groupInfo != nil {
			response.GroupName = groupInfo.Name
			response.GroupDescription = groupInfo.Description
		}
	}

	return response
}
