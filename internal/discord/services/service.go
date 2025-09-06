package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-falcon/internal/discord/dto"
	"go-falcon/internal/discord/models"
	"go-falcon/pkg/database"
)

// Service is the main Discord service that coordinates all operations
type Service struct {
	repo         *Repository
	oauthService *OAuthService
	botService   *BotService
	syncService  *SyncService
}

// NewService creates a new Discord service
func NewService(db *database.MongoDB, groupsService GroupsServiceInterface) *Service {
	repo := NewRepository(db)
	oauthService := NewOAuthService(repo)
	botService := NewBotService(repo)
	syncService := NewSyncService(repo, botService, groupsService)

	return &Service{
		repo:         repo,
		oauthService: oauthService,
		botService:   botService,
		syncService:  syncService,
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

	message := "Discord account linked successfully"
	if storedUserID == nil {
		message = "Discord authentication completed successfully"
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

	slog.InfoContext(ctx, "Discord account linked via API",
		"user_id", userID,
		"discord_id", userInfo.ID,
		"username", userInfo.Username)

	return &dto.DiscordMessageOutput{
		Body: dto.DiscordMessageResponse{
			Message: "Discord account linked successfully",
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
		return nil, fmt.Errorf("guild configuration already exists")
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

	return s.guildConfigToOutput(config), nil
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

	return s.guildConfigToOutput(config), nil
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

	return s.guildConfigToOutput(updated), nil
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
		configResponses[i] = s.guildConfigToOutput(config).Body
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
		return nil, fmt.Errorf("role mapping already exists for this guild and group")
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

	return &dto.DiscordRoleMappingOutput{
		Body: dto.DiscordRoleMappingResponse{
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
		},
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
		Body: dto.DiscordRoleMappingResponse{
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
		},
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

	return &dto.DiscordRoleMappingOutput{
		Body: dto.DiscordRoleMappingResponse{
			ID:              updated.ID.Hex(),
			GuildID:         updated.GuildID,
			GroupID:         updated.GroupID.Hex(),
			GroupName:       updated.GroupName,
			DiscordRoleID:   updated.DiscordRoleID,
			DiscordRoleName: updated.DiscordRoleName,
			IsActive:        updated.IsActive,
			CreatedBy:       updated.CreatedBy,
			CreatedAt:       updated.CreatedAt,
			UpdatedAt:       updated.UpdatedAt,
		},
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

	if err := s.repo.DeleteRoleMapping(ctx, mappingObjectID); err != nil {
		return nil, fmt.Errorf("failed to delete role mapping: %w", err)
	}

	return &dto.DiscordSuccessOutput{
		Body: dto.DiscordSuccessResponse{
			Message: "Role mapping deleted successfully",
		},
	}, nil
}

// ListRoleMappings lists role mappings for a guild with filtering
func (s *Service) ListRoleMappings(ctx context.Context, input *dto.ListRoleMappingsInput) (*dto.ListDiscordRoleMappingsOutput, error) {
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

	// Convert to response format
	mappingResponses := make([]dto.DiscordRoleMappingResponse, len(mappings))
	for i, mapping := range mappings {
		mappingResponses[i] = dto.DiscordRoleMappingResponse{
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
	}

	return &dto.ListDiscordRoleMappingsOutput{
		Body: dto.ListDiscordRoleMappingsResponse{
			GuildID:  input.GuildID,
			Mappings: mappingResponses,
			Total:    total,
			Page:     page,
			Limit:    limit,
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
func (s *Service) guildConfigToOutput(config *models.DiscordGuildConfig) *dto.DiscordGuildConfigOutput {
	// Convert role mappings
	roleMappings := make([]dto.DiscordRoleMappingResponse, len(config.RoleMappings))
	for i, mapping := range config.RoleMappings {
		roleMappings[i] = dto.DiscordRoleMappingResponse{
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
