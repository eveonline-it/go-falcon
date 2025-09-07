package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// DiscordGuildMember represents a Discord guild member
type DiscordGuildMember struct {
	User         DiscordUser `json:"user"`
	Nick         *string     `json:"nick"`
	Avatar       *string     `json:"avatar"`
	Roles        []string    `json:"roles"`
	JoinedAt     time.Time   `json:"joined_at"`
	PremiumSince *time.Time  `json:"premium_since"`
	Deaf         bool        `json:"deaf"`
	Mute         bool        `json:"mute"`
	Pending      *bool       `json:"pending"`
}

// DiscordUser represents a Discord user in API responses
type DiscordUser struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Discriminator string  `json:"discriminator"`
	GlobalName    *string `json:"global_name"`
	Avatar        *string `json:"avatar"`
	Bot           *bool   `json:"bot"`
	System        *bool   `json:"system"`
}

// DiscordRole represents a Discord role
type DiscordRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Hoist       bool   `json:"hoist"`
	Position    int    `json:"position"`
	Permissions string `json:"permissions"`
	Managed     bool   `json:"managed"`
	Mentionable bool   `json:"mentionable"`
}

// DiscordGuild represents a Discord guild
type DiscordGuild struct {
	ID                          string        `json:"id"`
	Name                        string        `json:"name"`
	Icon                        *string       `json:"icon"`
	IconHash                    *string       `json:"icon_hash"`
	Splash                      *string       `json:"splash"`
	DiscoverySplash             *string       `json:"discovery_splash"`
	Owner                       *bool         `json:"owner"`
	OwnerID                     string        `json:"owner_id"`
	Permissions                 *string       `json:"permissions"`
	AfkChannelID                *string       `json:"afk_channel_id"`
	AfkTimeout                  int           `json:"afk_timeout"`
	WidgetEnabled               *bool         `json:"widget_enabled"`
	WidgetChannelID             *string       `json:"widget_channel_id"`
	VerificationLevel           int           `json:"verification_level"`
	DefaultMessageNotifications int           `json:"default_message_notifications"`
	ExplicitContentFilter       int           `json:"explicit_content_filter"`
	Roles                       []DiscordRole `json:"roles"`
	Features                    []string      `json:"features"`
	MfaLevel                    int           `json:"mfa_level"`
	ApplicationID               *string       `json:"application_id"`
	SystemChannelID             *string       `json:"system_channel_id"`
	SystemChannelFlags          int           `json:"system_channel_flags"`
	RulesChannelID              *string       `json:"rules_channel_id"`
	MaxPresences                *int          `json:"max_presences"`
	MaxMembers                  *int          `json:"max_members"`
	VanityURLCode               *string       `json:"vanity_url_code"`
	Description                 *string       `json:"description"`
	Banner                      *string       `json:"banner"`
	PremiumTier                 int           `json:"premium_tier"`
	PremiumSubscriptionCount    *int          `json:"premium_subscription_count"`
	PreferredLocale             string        `json:"preferred_locale"`
	PublicUpdatesChannelID      *string       `json:"public_updates_channel_id"`
	MaxVideoChannelUsers        *int          `json:"max_video_channel_users"`
	ApproximateMemberCount      *int          `json:"approximate_member_count"`
	ApproximatePresenceCount    *int          `json:"approximate_presence_count"`
}

// RateLimiter handles Discord API rate limiting
type RateLimiter struct {
	requests map[string]time.Time
	limits   map[string]int
}

// BotService handles Discord bot operations
type BotService struct {
	repo        *Repository
	client      *http.Client
	rateLimiter *RateLimiter
}

// NewBotService creates a new Discord bot service
func NewBotService(repo *Repository) *BotService {
	return &BotService{
		repo: repo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: &RateLimiter{
			requests: make(map[string]time.Time),
			limits:   make(map[string]int),
		},
	}
}

// GetGuildInfo gets information about a Discord guild
func (s *BotService) GetGuildInfo(ctx context.Context, guildID, botToken string) (*DiscordGuild, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s", guildID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create guild info request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("User-Agent", "go-falcon/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make guild info request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read guild info response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("guild info request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var guild DiscordGuild
	if err := json.Unmarshal(body, &guild); err != nil {
		return nil, fmt.Errorf("failed to parse guild info response: %w", err)
	}

	return &guild, nil
}

// GetGuildMember gets information about a guild member
func (s *BotService) GetGuildMember(ctx context.Context, guildID, userID, botToken string) (*DiscordGuildMember, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s", guildID, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create guild member request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("User-Agent", "go-falcon/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make guild member request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read guild member response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Member not found
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("guild member request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var member DiscordGuildMember
	if err := json.Unmarshal(body, &member); err != nil {
		return nil, fmt.Errorf("failed to parse guild member response: %w", err)
	}

	return &member, nil
}

// GetGuildRoles gets all roles in a Discord guild
func (s *BotService) GetGuildRoles(ctx context.Context, guildID, botToken string) ([]DiscordRole, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/roles", guildID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create guild roles request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("User-Agent", "go-falcon/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make guild roles request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read guild roles response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("guild not found: status %d: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("guild roles request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var roles []DiscordRole
	if err := json.Unmarshal(body, &roles); err != nil {
		return nil, fmt.Errorf("failed to parse guild roles response: %w", err)
	}

	return roles, nil
}

// AddGuildMemberRole adds a role to a guild member
func (s *BotService) AddGuildMemberRole(ctx context.Context, guildID, userID, roleID, botToken string) error {
	// Check rate limit
	if err := s.checkRateLimit(ctx, "role_modify"); err != nil {
		return err
	}

	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s/roles/%s", guildID, userID, roleID)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create add role request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("User-Agent", "go-falcon/1.0")
	req.Header.Set("X-Audit-Log-Reason", "Go Falcon role synchronization")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make add role request: %w", err)
	}
	defer resp.Body.Close()

	// Update rate limit
	s.updateRateLimit("role_modify", resp.Header)

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add role request failed with status %d: %s", resp.StatusCode, string(body))
	}

	slog.InfoContext(ctx, "Successfully added Discord role",
		"guild_id", guildID,
		"user_id", userID,
		"role_id", roleID)

	return nil
}

// RemoveGuildMemberRole removes a role from a guild member
func (s *BotService) RemoveGuildMemberRole(ctx context.Context, guildID, userID, roleID, botToken string) error {
	// Check rate limit
	if err := s.checkRateLimit(ctx, "role_modify"); err != nil {
		return err
	}

	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s/roles/%s", guildID, userID, roleID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create remove role request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("User-Agent", "go-falcon/1.0")
	req.Header.Set("X-Audit-Log-Reason", "Go Falcon role synchronization")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make remove role request: %w", err)
	}
	defer resp.Body.Close()

	// Update rate limit
	s.updateRateLimit("role_modify", resp.Header)

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove role request failed with status %d: %s", resp.StatusCode, string(body))
	}

	slog.InfoContext(ctx, "Successfully removed Discord role",
		"guild_id", guildID,
		"user_id", userID,
		"role_id", roleID)

	return nil
}

// ModifyGuildMemberRoles modifies all roles for a guild member in a single request
func (s *BotService) ModifyGuildMemberRoles(ctx context.Context, guildID, userID, botToken string, roleIDs []string) error {
	// Check rate limit
	if err := s.checkRateLimit(ctx, "member_modify"); err != nil {
		return err
	}

	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s", guildID, userID)

	payload := map[string]interface{}{
		"roles": roleIDs,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal modify roles payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create modify roles request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "go-falcon/1.0")
	req.Header.Set("X-Audit-Log-Reason", "Go Falcon role synchronization")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make modify roles request: %w", err)
	}
	defer resp.Body.Close()

	// Update rate limit
	s.updateRateLimit("member_modify", resp.Header)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("modify roles request failed with status %d: %s", resp.StatusCode, string(body))
	}

	slog.InfoContext(ctx, "Successfully modified Discord member roles",
		"guild_id", guildID,
		"user_id", userID,
		"role_count", len(roleIDs))

	return nil
}

// ValidateBotToken validates a Discord bot token by making a test API call
func (s *BotService) ValidateBotToken(ctx context.Context, botToken string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("User-Agent", "go-falcon/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make validation request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid bot token")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("validation request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AddGuildMember adds a user to a Discord guild with optional initial roles
func (s *BotService) AddGuildMember(ctx context.Context, guildID, userID, accessToken, botToken string, roleIDs []string) error {
	// Check rate limit
	if err := s.checkRateLimit(ctx, "guild_member_add"); err != nil {
		return err
	}

	url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s", guildID, userID)

	payload := map[string]interface{}{
		"access_token": accessToken, // User's OAuth token with guilds.join scope
	}

	// Add roles if specified
	if len(roleIDs) > 0 {
		payload["roles"] = roleIDs
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal add member payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create add member request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "go-falcon/1.0")
	req.Header.Set("X-Audit-Log-Reason", "Go Falcon auto-join")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make add member request: %w", err)
	}
	defer resp.Body.Close()

	// Update rate limit
	s.updateRateLimit("guild_member_add", resp.Header)

	body, _ := io.ReadAll(resp.Body)

	// DEBUG: Log the raw Discord API response
	slog.InfoContext(ctx, "DEBUG: Discord API Response",
		"status_code", resp.StatusCode,
		"response_body", string(body),
		"guild_id", guildID,
		"user_id", userID)

	switch resp.StatusCode {
	case http.StatusCreated:
		// User successfully added to guild
		slog.InfoContext(ctx, "Successfully added user to Discord guild",
			"guild_id", guildID,
			"user_id", userID,
			"roles_assigned", len(roleIDs))
		return nil
	case http.StatusNoContent:
		// User already in guild, roles updated
		slog.InfoContext(ctx, "User already in guild, roles updated",
			"guild_id", guildID,
			"user_id", userID,
			"roles_assigned", len(roleIDs))
		return nil
	case http.StatusForbidden:
		return fmt.Errorf("Discord API 403 Forbidden: %s", string(body))
	case http.StatusNotFound:
		return fmt.Errorf("Discord API 404 Not Found: %s", string(body))
	default:
		return fmt.Errorf("add member request failed with status %d: %s", resp.StatusCode, string(body))
	}
}

// CheckBotPermissions checks if the bot has required permissions in a guild
func (s *BotService) CheckBotPermissions(ctx context.Context, guildID, botToken string) (bool, error) {
	// Get bot's member info
	botUser, err := s.getBotUser(ctx, botToken)
	if err != nil {
		return false, fmt.Errorf("failed to get bot user: %w", err)
	}

	member, err := s.GetGuildMember(ctx, guildID, botUser.ID, botToken)
	if err != nil {
		return false, fmt.Errorf("failed to get bot member: %w", err)
	}

	if member == nil {
		return false, fmt.Errorf("bot is not a member of the guild")
	}

	// Get guild info to check roles
	guild, err := s.GetGuildInfo(ctx, guildID, botToken)
	if err != nil {
		return false, fmt.Errorf("failed to get guild info: %w", err)
	}

	// Check if bot has role management permissions
	hasManageRoles := false
	for _, roleID := range member.Roles {
		for _, role := range guild.Roles {
			if role.ID == roleID {
				// Check for manage roles permission (0x10000000)
				permissions, _ := strconv.ParseInt(role.Permissions, 10, 64)
				if permissions&0x10000000 != 0 {
					hasManageRoles = true
					break
				}
			}
		}
		if hasManageRoles {
			break
		}
	}

	return hasManageRoles, nil
}

// getBotUser gets information about the bot user
func (s *BotService) getBotUser(ctx context.Context, botToken string) (*DiscordUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot user request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("User-Agent", "go-falcon/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make bot user request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read bot user response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bot user request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var botUser DiscordUser
	if err := json.Unmarshal(body, &botUser); err != nil {
		return nil, fmt.Errorf("failed to parse bot user response: %w", err)
	}

	return &botUser, nil
}

// Rate limiting functions

// checkRateLimit checks if we can make a request without hitting rate limits
func (s *BotService) checkRateLimit(ctx context.Context, endpoint string) error {
	if lastRequest, exists := s.rateLimiter.requests[endpoint]; exists {
		timeSince := time.Since(lastRequest)
		if timeSince < time.Second {
			waitTime := time.Second - timeSince
			slog.WarnContext(ctx, "Rate limit hit, waiting", "endpoint", endpoint, "wait_time", waitTime)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
				// Continue after wait
			}
		}
	}
	return nil
}

// updateRateLimit updates rate limit information based on response headers
func (s *BotService) updateRateLimit(endpoint string, headers http.Header) {
	s.rateLimiter.requests[endpoint] = time.Now()

	// Parse rate limit headers if present
	if remaining := headers.Get("X-RateLimit-Remaining"); remaining != "" {
		if count, err := strconv.Atoi(remaining); err == nil {
			s.rateLimiter.limits[endpoint] = count
		}
	}
}

// EncryptBotToken encrypts a bot token for storage (simplified implementation)
// In production, this should use proper encryption with a secret key
func (s *BotService) EncryptBotToken(token string) string {
	// TODO: Implement proper encryption
	return token
}

// DecryptBotToken decrypts a stored bot token (simplified implementation)
// In production, this should use proper decryption with a secret key
func (s *BotService) DecryptBotToken(encryptedToken string) string {
	// TODO: Implement proper decryption
	return encryptedToken
}
