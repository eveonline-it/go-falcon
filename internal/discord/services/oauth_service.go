package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-falcon/internal/discord/models"
	"go-falcon/pkg/config"
)

// DiscordOAuthConfig holds Discord OAuth configuration
type DiscordOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

// DiscordUserInfo represents user information from Discord API
type DiscordUserInfo struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	GlobalName *string `json:"global_name"`
	Avatar     *string `json:"avatar"`
	Email      *string `json:"email"`
}

// DiscordTokenResponse represents Discord OAuth token response
type DiscordTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// OAuthService handles Discord OAuth operations
type OAuthService struct {
	config *DiscordOAuthConfig
	repo   *Repository
	client *http.Client
}

// NewOAuthService creates a new Discord OAuth service
func NewOAuthService(repo *Repository) *OAuthService {
	// Load configuration from environment
	cfg := &DiscordOAuthConfig{
		ClientID:     config.GetEnv("DISCORD_CLIENT_ID", ""),
		ClientSecret: config.GetEnv("DISCORD_CLIENT_SECRET", ""),
		RedirectURI:  config.GetEnv("DISCORD_REDIRECT_URI", "http://localhost:3000/api/discord/auth/callback"),
		Scopes:       []string{"identify", "guilds"},
	}

	// Override scopes if specified in environment
	if scopes := config.GetEnv("DISCORD_SCOPES", ""); scopes != "" {
		cfg.Scopes = strings.Split(scopes, " ")
	}

	return &OAuthService{
		config: cfg,
		repo:   repo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateAuthURL creates a Discord OAuth authorization URL
func (s *OAuthService) GenerateAuthURL(ctx context.Context, linkToUser bool, userID *string) (string, string, error) {
	// Generate cryptographically secure state parameter
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate state parameter: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Store state in database for CSRF protection
	oauthState := &models.DiscordOAuthState{
		State:     state,
		UserID:    userID,
		ExpiresAt: time.Now().Add(15 * time.Minute), // 15 minute expiry
	}

	if err := s.repo.CreateOAuthState(ctx, oauthState); err != nil {
		return "", "", fmt.Errorf("failed to store OAuth state: %w", err)
	}

	// Build authorization URL
	params := url.Values{}
	params.Set("client_id", s.config.ClientID)
	params.Set("redirect_uri", s.config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(s.config.Scopes, " "))
	params.Set("state", state)

	authURL := fmt.Sprintf("https://discord.com/api/oauth2/authorize?%s", params.Encode())

	slog.InfoContext(ctx, "Generated Discord OAuth URL",
		"link_to_user", linkToUser,
		"user_id", userID,
		"scopes", s.config.Scopes)

	return authURL, state, nil
}

// HandleCallback processes the OAuth callback and exchanges code for tokens
func (s *OAuthService) HandleCallback(ctx context.Context, code, state string) (*DiscordUserInfo, *DiscordTokenResponse, *string, error) {
	// Validate state parameter
	storedState, err := s.repo.GetOAuthState(ctx, state)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get OAuth state: %w", err)
	}
	if storedState == nil {
		return nil, nil, nil, fmt.Errorf("invalid or expired state parameter")
	}

	// Exchange authorization code for access token
	tokenResponse, err := s.exchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user information from Discord
	userInfo, err := s.getUserInfo(ctx, tokenResponse.AccessToken)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var userID string
	if storedState.UserID != nil {
		userID = *storedState.UserID
	}

	slog.InfoContext(ctx, "Discord OAuth callback completed",
		"user_id", userID,
		"discord_id", userInfo.ID,
		"username", userInfo.Username)

	return userInfo, tokenResponse, storedState.UserID, nil
}

// exchangeCodeForToken exchanges authorization code for access token
func (s *OAuthService) exchangeCodeForToken(ctx context.Context, code string) (*DiscordTokenResponse, error) {
	// Prepare token request
	data := url.Values{}
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", s.config.RedirectURI)

	// Make token request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "go-falcon/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse DiscordTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResponse, nil
}

// getUserInfo gets user information from Discord API
func (s *OAuthService) getUserInfo(ctx context.Context, accessToken string) (*DiscordUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "go-falcon/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make user info request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo DiscordUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info response: %w", err)
	}

	return &userInfo, nil
}

// RefreshToken refreshes an expired Discord access token
func (s *OAuthService) RefreshToken(ctx context.Context, refreshToken string) (*DiscordTokenResponse, error) {
	// Prepare refresh request
	data := url.Values{}
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	// Make refresh request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "go-falcon/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse DiscordTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	slog.InfoContext(ctx, "Discord token refreshed successfully")
	return &tokenResponse, nil
}

// ValidateToken validates a Discord access token by making a test API call
func (s *OAuthService) ValidateToken(ctx context.Context, accessToken string) (*DiscordUserInfo, error) {
	return s.getUserInfo(ctx, accessToken)
}

// RevokeToken revokes a Discord access token
func (s *OAuthService) RevokeToken(ctx context.Context, accessToken string) error {
	// Prepare revocation request
	data := url.Values{}
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)
	data.Set("token", accessToken)

	// Make revocation request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://discord.com/api/oauth2/token/revoke", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "go-falcon/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make revoke request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("revoke request failed with status %d: %s", resp.StatusCode, string(body))
	}

	slog.InfoContext(ctx, "Discord token revoked successfully")
	return nil
}

// CreateOrUpdateDiscordUser creates or updates a Discord user record
func (s *OAuthService) CreateOrUpdateDiscordUser(ctx context.Context, userID string, userInfo *DiscordUserInfo, tokenResponse *DiscordTokenResponse) (*models.DiscordUser, error) {
	// Check if Discord user already exists
	existing, err := s.repo.GetDiscordUserByDiscordID(ctx, userInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing Discord user: %w", err)
	}

	// Calculate token expiry
	tokenExpiry := time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)

	if existing != nil {
		// Update existing user
		update := map[string]interface{}{
			"user_id":       userID,
			"username":      userInfo.Username,
			"global_name":   userInfo.GlobalName,
			"avatar":        userInfo.Avatar,
			"access_token":  s.encryptToken(tokenResponse.AccessToken),
			"refresh_token": s.encryptToken(tokenResponse.RefreshToken),
			"token_expiry":  tokenExpiry,
			"is_active":     true,
		}

		if err := s.repo.UpdateDiscordUser(ctx, existing.ID, update); err != nil {
			return nil, fmt.Errorf("failed to update Discord user: %w", err)
		}

		// Return updated user
		return s.repo.GetDiscordUserByDiscordID(ctx, userInfo.ID)
	}

	// Create new user
	discordUser := &models.DiscordUser{
		UserID:       userID,
		DiscordID:    userInfo.ID,
		Username:     userInfo.Username,
		GlobalName:   userInfo.GlobalName,
		Avatar:       userInfo.Avatar,
		AccessToken:  s.encryptToken(tokenResponse.AccessToken),
		RefreshToken: s.encryptToken(tokenResponse.RefreshToken),
		TokenExpiry:  tokenExpiry,
		IsActive:     true,
	}

	if err := s.repo.CreateDiscordUser(ctx, discordUser); err != nil {
		return nil, fmt.Errorf("failed to create Discord user: %w", err)
	}

	return discordUser, nil
}

// GetUserGuilds gets the Discord guilds a user belongs to
func (s *OAuthService) GetUserGuilds(ctx context.Context, accessToken string) ([]DiscordGuildInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/users/@me/guilds", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create guilds request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "go-falcon/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make guilds request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read guilds response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("guilds request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var guilds []DiscordGuildInfo
	if err := json.Unmarshal(body, &guilds); err != nil {
		return nil, fmt.Errorf("failed to parse guilds response: %w", err)
	}

	return guilds, nil
}

// DiscordGuildInfo represents guild information from Discord API
type DiscordGuildInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Owner       bool   `json:"owner"`
	Permissions string `json:"permissions"`
}

// RefreshExpiringTokens refreshes Discord tokens that are expiring soon
func (s *OAuthService) RefreshExpiringTokens(ctx context.Context, batchSize int) (int, int, error) {
	// Find users with tokens expiring in the next hour
	filter := map[string]interface{}{
		"token_expiry": map[string]interface{}{
			"$lt": time.Now().Add(time.Hour),
		},
		"is_active": true,
	}

	users, _, err := s.repo.ListDiscordUsers(ctx, filter, 1, batchSize)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get users with expiring tokens: %w", err)
	}

	successCount := 0
	failureCount := 0

	for _, user := range users {
		// Decrypt refresh token
		refreshToken := s.decryptToken(user.RefreshToken)
		if refreshToken == "" {
			slog.WarnContext(ctx, "Failed to decrypt refresh token", "user_id", user.UserID, "discord_id", user.DiscordID)
			failureCount++
			continue
		}

		// Attempt to refresh token
		tokenResponse, err := s.RefreshToken(ctx, refreshToken)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to refresh Discord token", "user_id", user.UserID, "discord_id", user.DiscordID, "error", err)
			failureCount++
			continue
		}

		// Update user with new tokens
		tokenExpiry := time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
		update := map[string]interface{}{
			"access_token":  s.encryptToken(tokenResponse.AccessToken),
			"refresh_token": s.encryptToken(tokenResponse.RefreshToken),
			"token_expiry":  tokenExpiry,
		}

		if err := s.repo.UpdateDiscordUser(ctx, user.ID, update); err != nil {
			slog.ErrorContext(ctx, "Failed to update user tokens", "user_id", user.UserID, "discord_id", user.DiscordID, "error", err)
			failureCount++
			continue
		}

		slog.InfoContext(ctx, "Successfully refreshed Discord token", "user_id", user.UserID, "discord_id", user.DiscordID)
		successCount++
	}

	return successCount, failureCount, nil
}

// encryptToken encrypts a token for storage (simplified implementation)
// In production, this should use proper encryption with a secret key
func (s *OAuthService) encryptToken(token string) string {
	// TODO: Implement proper encryption
	return token
}

// decryptToken decrypts a stored token (simplified implementation)
// In production, this should use proper decryption with a secret key
func (s *OAuthService) decryptToken(encryptedToken string) string {
	// TODO: Implement proper decryption
	return encryptedToken
}
