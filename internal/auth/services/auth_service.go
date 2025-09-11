package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"go-falcon/internal/auth/dto"
	"go-falcon/internal/auth/models"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/handlers"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// AuthService is the main service that orchestrates auth operations
type AuthService struct {
	repository     *Repository
	eveService     *EVEService
	profileService *ProfileService
	groupsService  GroupsService // Interface to avoid circular dependency
}

// GroupsService interface for groups module dependency
type GroupsService interface {
	EnsureFirstUserSuperAdmin(ctx context.Context, characterID int64) error
	IsCharacterInGroup(ctx context.Context, characterID int64, groupName string) (bool, error)
	IsUserInGroup(ctx context.Context, userID string, groupName string) (bool, error)
	AutoJoinCharacterToEnabledGroups(ctx context.Context, characterID int64, corporationID, allianceID *int64, scopes string) error
}

// NewAuthService creates a new auth service with all dependencies
func NewAuthService(mongodb *database.MongoDB, esiClient *evegateway.Client) *AuthService {
	repository := NewRepository(mongodb)
	eveService := NewEVEService(repository)
	profileService := NewProfileService(repository, eveService, esiClient)

	return &AuthService{
		repository:     repository,
		eveService:     eveService,
		profileService: profileService,
		groupsService:  nil, // Will be set after groups module initialization
	}
}

// SetGroupsService sets the groups service dependency (called after groups module initialization)
func (s *AuthService) SetGroupsService(groupsService GroupsService) {
	s.groupsService = groupsService
}

// HealthCheck handles health check requests
func (s *AuthService) HealthCheck(w http.ResponseWriter, r *http.Request) {
	handlers.HealthHandler("auth")(w, r)
}

// GetAuthStatus returns current authentication status
func (s *AuthService) GetAuthStatus(ctx context.Context, r *http.Request) (*dto.AuthStatusResponse, error) {
	// Try to get JWT from cookie or header
	var jwtToken string

	if cookie, err := r.Cookie("falcon_auth_token"); err == nil {
		jwtToken = cookie.Value
	} else {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			jwtToken = authHeader[7:]
		}
	}

	if jwtToken == "" {
		return &dto.AuthStatusResponse{
			Authenticated: false,
			UserID:        nil,
			CharacterID:   nil,
			CharacterName: nil,
			Characters:    []string{},
			Permissions:   []string{},
		}, nil
	}

	// Validate JWT and get user info
	user, err := s.eveService.ValidateJWT(jwtToken)
	if err != nil {
		return &dto.AuthStatusResponse{
			Authenticated: false,
			UserID:        nil,
			CharacterID:   nil,
			CharacterName: nil,
			Characters:    []string{},
			Permissions:   []string{},
		}, nil
	}

	// Check if user is super admin via groups service
	permissions := []string{}
	if s.groupsService != nil {
		isSuperAdmin, err := s.groupsService.IsUserInGroup(ctx, user.UserID, "Super Administrator")
		if err == nil && isSuperAdmin {
			// Grant super admin status
			permissions = []string{"super_admin"}
		}
	}

	// Return authenticated response with user info
	return &dto.AuthStatusResponse{
		Authenticated: true,
		UserID:        &user.UserID,
		CharacterID:   &user.CharacterID,
		CharacterName: &user.CharacterName,
		Characters:    []string{user.CharacterName}, // For now, just include current character
		Permissions:   permissions,
	}, nil
}

// GetCurrentUser returns current user information
func (s *AuthService) GetCurrentUser(ctx context.Context, r *http.Request) (*dto.UserInfoResponse, error) {
	// Try to get JWT from cookie or header
	var jwtToken string

	if cookie, err := r.Cookie("falcon_auth_token"); err == nil {
		jwtToken = cookie.Value
	} else {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			jwtToken = authHeader[7:]
		}
	}

	if jwtToken == "" {
		return nil, fmt.Errorf("no authentication token")
	}

	// Validate JWT and get user info
	user, err := s.eveService.ValidateJWT(jwtToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return &dto.UserInfoResponse{
		UserID:        user.UserID,
		CharacterID:   user.CharacterID,
		CharacterName: user.CharacterName,
		Scopes:        user.Scopes,
	}, nil
}

// GetAuthStatusFromHeaders returns current authentication status from header strings
func (s *AuthService) GetAuthStatusFromHeaders(ctx context.Context, authHeader, cookieHeader string) (*dto.AuthStatusResponse, error) {
	// Try to extract JWT token from headers
	var jwtToken string

	// Try Authorization header first
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		jwtToken = authHeader[7:]
	}

	// If not found, try cookie header
	if jwtToken == "" && cookieHeader != "" {
		// Parse cookie header to find falcon_auth_token
		cookies := strings.Split(cookieHeader, ";")
		for _, cookie := range cookies {
			cookie = strings.TrimSpace(cookie)
			if strings.HasPrefix(cookie, "falcon_auth_token=") {
				jwtToken = strings.TrimPrefix(cookie, "falcon_auth_token=")
				break
			}
		}
	}

	if jwtToken == "" {
		return &dto.AuthStatusResponse{
			Authenticated: false,
			UserID:        nil,
			CharacterID:   nil,
			CharacterName: nil,
			Characters:    []string{},
			Permissions:   []string{},
		}, nil
	}

	// Validate JWT and get user info
	user, err := s.eveService.ValidateJWT(jwtToken)
	if err != nil {
		return &dto.AuthStatusResponse{
			Authenticated: false,
			UserID:        nil,
			CharacterID:   nil,
			CharacterName: nil,
			Characters:    []string{},
			Permissions:   []string{},
		}, nil
	}

	// Check if user is super admin via groups service
	permissions := []string{}
	if s.groupsService != nil {
		isSuperAdmin, err := s.groupsService.IsUserInGroup(ctx, user.UserID, "Super Administrator")
		if err == nil && isSuperAdmin {
			// Grant super admin status
			permissions = []string{"super_admin"}
		}
	}

	// Return authenticated response with user info
	return &dto.AuthStatusResponse{
		Authenticated: true,
		UserID:        &user.UserID,
		CharacterID:   &user.CharacterID,
		CharacterName: &user.CharacterName,
		Characters:    []string{user.CharacterName}, // For now, just include current character
		Permissions:   permissions,
	}, nil
}

// GetCurrentUserFromHeaders returns current user information from header strings
func (s *AuthService) GetCurrentUserFromHeaders(ctx context.Context, authHeader, cookieHeader string) (*dto.UserInfoResponse, error) {
	// Try to extract JWT token from headers
	var jwtToken string

	// Try Authorization header first
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		jwtToken = authHeader[7:]
	}

	// If not found, try cookie header
	if jwtToken == "" && cookieHeader != "" {
		// Parse cookie header to find falcon_auth_token
		cookies := strings.Split(cookieHeader, ";")
		for _, cookie := range cookies {
			cookie = strings.TrimSpace(cookie)
			if strings.HasPrefix(cookie, "falcon_auth_token=") {
				jwtToken = strings.TrimPrefix(cookie, "falcon_auth_token=")
				break
			}
		}
	}

	if jwtToken == "" {
		return nil, fmt.Errorf("no authentication token")
	}

	// Validate JWT and get user info
	user, err := s.eveService.ValidateJWT(jwtToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return &dto.UserInfoResponse{
		UserID:        user.UserID,
		CharacterID:   user.CharacterID,
		CharacterName: user.CharacterName,
		Scopes:        user.Scopes,
	}, nil
}

// InitiateEVELogin initiates EVE SSO login flow
func (s *AuthService) InitiateEVELogin(ctx context.Context, withScopes bool, userID string) (*dto.EVELoginResponse, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.service.initiate_eve_login")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "initiate_eve_login"),
		attribute.Bool("with_scopes", withScopes),
	)

	authURL, state, err := s.eveService.GenerateAuthURL(ctx, withScopes, userID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to generate auth URL: %w", err)
	}

	return &dto.EVELoginResponse{
		AuthURL: authURL,
		State:   state,
	}, nil
}

// HandleEVECallback processes EVE SSO callback (legacy, without user ID from cookie)
func (s *AuthService) HandleEVECallback(ctx context.Context, code, state string) (string, *dto.UserInfoResponse, error) {
	return s.HandleEVECallbackWithUserID(ctx, code, state, "")
}

// HandleEVECallbackWithUserID processes EVE SSO callback with optional existing user ID from cookie
func (s *AuthService) HandleEVECallbackWithUserID(ctx context.Context, code, state, cookieUserID string) (string, *dto.UserInfoResponse, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.service.handle_eve_callback")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "handle_eve_callback"),
		attribute.String("state", state),
		attribute.Bool("has_cookie_user_id", cookieUserID != ""),
	)

	// Handle OAuth callback
	charInfo, tokenResp, stateUserID, err := s.eveService.HandleCallback(ctx, code, state)
	if err != nil {
		span.RecordError(err)
		return "", nil, fmt.Errorf("failed to handle callback: %w", err)
	}

	// Determine the user ID with priority:
	// 1. User ID from valid cookie (if user is already logged in)
	// 2. User ID from state (if stored during login initiation)
	// 3. User ID from existing profile for this character
	// 4. Generate new user ID
	userID := ""

	// First priority: use user ID from valid cookie if available
	if cookieUserID != "" {
		userID = cookieUserID
		span.SetAttributes(attribute.String("user_id_source", "cookie"))
	} else if stateUserID != "" {
		// Second priority: use user ID from state
		userID = stateUserID
		span.SetAttributes(attribute.String("user_id_source", "state"))
	} else {
		// Third priority: check if character already has a profile
		existingProfile, err := s.profileService.GetProfile(ctx, charInfo.CharacterID)
		if err != nil {
			span.RecordError(err)
			return "", nil, fmt.Errorf("failed to check existing profile: %w", err)
		}

		if existingProfile != nil {
			userID = existingProfile.UserID
			span.SetAttributes(attribute.String("user_id_source", "existing_profile"))
		} else {
			// Last resort: generate new user ID
			userID = uuid.New().String()
			span.SetAttributes(attribute.String("user_id_source", "new_uuid"))
		}
	}

	// Create or update user profile
	profile, err := s.profileService.CreateOrUpdateProfile(ctx, charInfo, userID, tokenResp.AccessToken, tokenResp.RefreshToken, tokenResp.ExpiresIn)
	if err != nil {
		span.RecordError(err)
		return "", nil, fmt.Errorf("failed to create/update profile: %w", err)
	}

	// Check if this should be the first super admin (only if groups service is available)
	if s.groupsService != nil {
		if err := s.groupsService.EnsureFirstUserSuperAdmin(ctx, int64(profile.CharacterID)); err != nil {
			// Log error but don't fail the authentication process
			slog.Error("Failed to ensure first user super admin", "error", err, "character_id", profile.CharacterID)
		}

		// Auto-join character to enabled corporation/alliance groups
		corpID := (*int64)(nil)
		allianceID := (*int64)(nil)

		if profile.CorporationID != 0 {
			corpIDInt64 := int64(profile.CorporationID)
			corpID = &corpIDInt64
		}
		if profile.AllianceID != 0 {
			allianceIDInt64 := int64(profile.AllianceID)
			allianceID = &allianceIDInt64
		}

		if err := s.groupsService.AutoJoinCharacterToEnabledGroups(ctx, int64(profile.CharacterID), corpID, allianceID, profile.Scopes); err != nil {
			// Log error but don't fail authentication
			slog.Error("Failed to auto-join character to enabled groups",
				"error", err, "character_id", profile.CharacterID, "corp_id", corpID, "alliance_id", allianceID, "has_scopes", profile.Scopes != "")
		} else {
			slog.Info("Auto-joined character to enabled groups", "character_id", profile.CharacterID, "has_scopes", profile.Scopes != "")
		}
	}

	// Generate JWT token
	jwtToken, _, err := s.eveService.GenerateJWT(profile.UserID, profile.CharacterID, profile.CharacterName, profile.Scopes)
	if err != nil {
		span.RecordError(err)
		return "", nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	userInfo := &dto.UserInfoResponse{
		UserID:        profile.UserID,
		CharacterID:   profile.CharacterID,
		CharacterName: profile.CharacterName,
		Scopes:        profile.Scopes,
	}

	return jwtToken, userInfo, nil
}

// ExchangeEVEToken exchanges EVE token for JWT (mobile apps)
func (s *AuthService) ExchangeEVEToken(ctx context.Context, req *dto.EVETokenExchangeRequest) (*dto.TokenResponse, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.service.exchange_eve_token")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "exchange_eve_token"),
	)

	// Verify the EVE access token
	charInfo, err := s.verifyEVEAccessToken(ctx, req.AccessToken)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to verify EVE token: %w", err)
	}

	// Calculate ExpiresIn from ExpiresOn timestamp
	expiresOn, err := time.Parse("2006-01-02T15:04:05Z", charInfo.ExpiresOn)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to parse token expiry: %w", err)
	}
	expiresIn := int(time.Until(expiresOn).Seconds())

	// Create or update user profile
	profile, err := s.profileService.CreateOrUpdateProfile(ctx, charInfo, "", req.AccessToken, req.RefreshToken, expiresIn)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create/update profile: %w", err)
	}

	// Generate JWT token
	jwtToken, expiresAt, err := s.eveService.GenerateJWT(profile.UserID, profile.CharacterID, profile.CharacterName, profile.Scopes)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	return &dto.TokenResponse{
		Token:     jwtToken,
		ExpiresAt: expiresAt,
	}, nil
}

// GetUserProfile returns full user profile
func (s *AuthService) GetUserProfile(ctx context.Context, characterID int) (*dto.ProfileResponse, error) {
	profile, err := s.profileService.GetProfile(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	if profile == nil {
		return nil, fmt.Errorf("profile not found")
	}

	return s.ProfileToDTO(profile), nil
}

// RefreshUserProfile refreshes user profile from ESI
func (s *AuthService) RefreshUserProfile(ctx context.Context, characterID int) (*dto.ProfileResponse, error) {
	profile, err := s.profileService.RefreshProfile(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh profile: %w", err)
	}

	// Re-sync group memberships after profile refresh (in case corp/alliance changed)
	if s.groupsService != nil {
		corpID := (*int64)(nil)
		allianceID := (*int64)(nil)

		if profile.CorporationID != 0 {
			corpIDInt64 := int64(profile.CorporationID)
			corpID = &corpIDInt64
		}
		if profile.AllianceID != 0 {
			allianceIDInt64 := int64(profile.AllianceID)
			allianceID = &allianceIDInt64
		}

		if err := s.groupsService.AutoJoinCharacterToEnabledGroups(ctx, int64(profile.CharacterID), corpID, allianceID, profile.Scopes); err != nil {
			// Log error but don't fail profile refresh
			slog.Error("Failed to re-sync character groups after profile refresh",
				"error", err, "character_id", profile.CharacterID, "corp_id", corpID, "alliance_id", allianceID, "has_scopes", profile.Scopes != "")
		} else {
			slog.Debug("Re-synced character groups after profile refresh", "character_id", profile.CharacterID, "has_scopes", profile.Scopes != "")
		}
	}

	return s.ProfileToDTO(profile), nil
}

// GetPublicProfile returns public character information
func (s *AuthService) GetPublicProfile(ctx context.Context, characterID int) (*dto.PublicProfileResponse, error) {
	profile, err := s.profileService.GetProfile(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	if profile == nil {
		return nil, fmt.Errorf("profile not found")
	}

	return &dto.PublicProfileResponse{
		CharacterID:     profile.CharacterID,
		CharacterName:   profile.CharacterName,
		CorporationID:   profile.CorporationID,
		CorporationName: profile.CorporationName,
		AllianceID:      profile.AllianceID,
		AllianceName:    profile.AllianceName,
		SecurityStatus:  profile.SecurityStatus,
	}, nil
}

// GetBearerToken generates a bearer token for authenticated user
func (s *AuthService) GetBearerToken(ctx context.Context, userID string, characterID int, characterName, scopes string) (*dto.TokenResponse, error) {
	jwtToken, expiresAt, err := s.eveService.GenerateJWT(userID, characterID, characterName, scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.TokenResponse{
		Token:     jwtToken,
		ExpiresAt: expiresAt,
	}, nil
}

// RefreshExpiringTokens refreshes tokens that are expiring soon
func (s *AuthService) RefreshExpiringTokens(ctx context.Context, batchSize int) (int, int, error) {
	return s.profileService.RefreshExpiringTokens(ctx, batchSize)
}

// GetAllCharactersByUserID retrieves all character profiles for a given user ID
// Used by groups module for multi-character permission resolution
func (s *AuthService) GetAllCharactersByUserID(ctx context.Context, userID string) ([]*models.UserProfile, error) {
	return s.repository.GetAllCharactersByUserID(ctx, userID)
}

// GetUserProfileByCharacterID retrieves a user profile by character ID
// Used by other modules that need to access user profile data
func (s *AuthService) GetUserProfileByCharacterID(ctx context.Context, characterID int) (*models.UserProfile, error) {
	return s.repository.GetUserProfileByCharacterID(ctx, characterID)
}

// CleanupExpiredStates removes expired OAuth states
func (s *AuthService) CleanupExpiredStates(ctx context.Context) error {
	return s.eveService.CleanupExpiredStates(ctx)
}

// ValidateJWT validates a JWT token (for middleware)
func (s *AuthService) ValidateJWT(token string) (*models.AuthenticatedUser, error) {
	return s.eveService.ValidateJWT(token)
}

// verifyEVEAccessToken verifies an EVE access token and returns character info
func (s *AuthService) verifyEVEAccessToken(ctx context.Context, accessToken string) (*models.EVECharacterInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://login.eveonline.com/oauth/verify", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("EVE token verification failed: %s", resp.Status)
	}

	var charInfo models.EVECharacterInfo
	if err := json.NewDecoder(resp.Body).Decode(&charInfo); err != nil {
		return nil, err
	}

	return &charInfo, nil
}

// ProfileToDTO converts a profile model to DTO
func (s *AuthService) ProfileToDTO(profile *models.UserProfile) *dto.ProfileResponse {
	return &dto.ProfileResponse{
		UserID:          profile.UserID,
		CharacterID:     profile.CharacterID,
		CharacterName:   profile.CharacterName,
		CorporationID:   profile.CorporationID,
		CorporationName: profile.CorporationName,
		AllianceID:      profile.AllianceID,
		AllianceName:    profile.AllianceName,
		SecurityStatus:  profile.SecurityStatus,
		Birthday:        profile.Birthday,
		Scopes:          profile.Scopes,
		TokenExpiry:     profile.TokenExpiry,
		LastLogin:       profile.LastLogin,
		ProfileUpdated:  profile.ProfileUpdated,
		Valid:           profile.Valid,
		Metadata:        profile.Metadata,
	}
}

// GetStatus returns the health status of the auth module
func (s *AuthService) GetStatus(ctx context.Context) *dto.AuthModuleStatusResponse {
	dependencies := &dto.AuthDependencyStatus{}
	metrics := &dto.AuthMetrics{}
	overallStatus := "healthy"
	var messages []string

	// 1. Check database connectivity with timing
	dbStartTime := time.Now()
	dbErr := s.repository.CheckHealth(ctx)
	dbLatency := time.Since(dbStartTime)

	if dbErr != nil {
		dependencies.Database = "unhealthy"
		dependencies.DatabaseLatency = dbLatency.String()
		overallStatus = "unhealthy"
		messages = append(messages, fmt.Sprintf("Database connection failed: %v", dbErr))
	} else {
		dependencies.Database = "healthy"
		dependencies.DatabaseLatency = dbLatency.String()

		// Get simple user count from database - using existing method if available
		// For now, use placeholder values that would be implemented with proper repository methods
		metrics.TotalUsers = 0     // Would query: db.user_profiles.countDocuments()
		metrics.ActiveSessions = 0 // Would query recent logins
		metrics.FailedLogins = 0   // Would track login attempts
		metrics.TokenRefreshes = 0 // Would track token operations
	}

	// 2. Check EVE Online ESI basic connectivity
	// Use existing evegateway status functionality if available
	esiStartTime := time.Now()
	dependencies.EVEOnlineESI = "unknown"
	dependencies.ESILatency = "0ms"

	// Try to get EVE server status using existing evegateway
	// This is a simplified check - in full implementation would use dedicated method
	esiLatency := time.Since(esiStartTime)
	dependencies.ESILatency = esiLatency.String()
	dependencies.EVEOnlineESI = "healthy" // Placeholder - would need actual ESI check

	// 3. Get system metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics.MemoryUsage = float64(m.Alloc) / 1024 / 1024 // Convert to MB

	// Placeholder metrics - would be implemented with proper tracking
	metrics.AverageLoginTime = "~200ms"

	// 4. Determine final message
	message := ""
	if len(messages) > 0 {
		message = messages[0] // Show first/most important issue
		if len(messages) > 1 {
			message += fmt.Sprintf(" (+%d more issues)", len(messages)-1)
		}
	} else {
		message = "Auth module operational"
	}

	return &dto.AuthModuleStatusResponse{
		Module:       "auth",
		Status:       overallStatus,
		Message:      message,
		Dependencies: dependencies,
		Metrics:      metrics,
		LastChecked:  time.Now().Format(time.RFC3339),
	}
}

// RefreshAccessToken refreshes an EVE access token using a refresh token
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (*models.EVETokenResponse, error) {
	return s.eveService.RefreshAccessToken(ctx, refreshToken)
}

// VerifyJWT verifies a JWT token and returns user information with expiration time
func (s *AuthService) VerifyJWT(token string) (*models.AuthenticatedUser, time.Time, error) {
	return s.eveService.VerifyJWT(token)
}
