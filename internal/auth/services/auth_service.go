package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	}
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

	// Return authenticated response with user info
	return &dto.AuthStatusResponse{
		Authenticated: true,
		UserID:        &user.UserID,
		CharacterID:   &user.CharacterID,
		CharacterName: &user.CharacterName,
		Characters:    []string{user.CharacterName}, // For now, just include current character
		Permissions:   []string{},                   // TODO: Implement permissions system
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

// HandleEVECallback processes EVE SSO callback
func (s *AuthService) HandleEVECallback(ctx context.Context, code, state string) (string, *dto.UserInfoResponse, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.service.handle_eve_callback")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "handle_eve_callback"),
		attribute.String("state", state),
	)

	// Handle OAuth callback
	charInfo, tokenResp, existingUserID, err := s.eveService.HandleCallback(ctx, code, state)
	if err != nil {
		span.RecordError(err)
		return "", nil, fmt.Errorf("failed to handle callback: %w", err)
	}

	// Check if user already exists
	userID := existingUserID
	if userID == "" {
		// Check if character already has a profile
		existingProfile, err := s.profileService.GetProfile(ctx, charInfo.CharacterID)
		if err != nil {
			span.RecordError(err)
			return "", nil, fmt.Errorf("failed to check existing profile: %w", err)
		}
		
		if existingProfile != nil {
			userID = existingProfile.UserID
		} else {
			userID = uuid.New().String()
		}
	}

	// Create or update user profile
	profile, err := s.profileService.CreateOrUpdateProfile(ctx, charInfo, userID, tokenResp.AccessToken, tokenResp.RefreshToken)
	if err != nil {
		span.RecordError(err)
		return "", nil, fmt.Errorf("failed to create/update profile: %w", err)
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

	// Create or update user profile
	profile, err := s.profileService.CreateOrUpdateProfile(ctx, charInfo, "", req.AccessToken, req.RefreshToken)
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

	return s.profileToDTO(profile), nil
}

// RefreshUserProfile refreshes user profile from ESI
func (s *AuthService) RefreshUserProfile(ctx context.Context, characterID int) (*dto.ProfileResponse, error) {
	profile, err := s.profileService.RefreshProfile(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh profile: %w", err)
	}

	return s.profileToDTO(profile), nil
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

// profileToDTO converts a profile model to DTO
func (s *AuthService) profileToDTO(profile *models.UserProfile) *dto.ProfileResponse {
	return &dto.ProfileResponse{
		UserID:            profile.UserID,
		CharacterID:       profile.CharacterID,
		CharacterName:     profile.CharacterName,
		CorporationID:     profile.CorporationID,
		CorporationName:   profile.CorporationName,
		AllianceID:        profile.AllianceID,
		AllianceName:      profile.AllianceName,
		SecurityStatus:    profile.SecurityStatus,
		Birthday:          profile.Birthday,
		Scopes:            profile.Scopes,
		TokenExpiry:       profile.TokenExpiry,
		LastLogin:         profile.LastLogin,
		ProfileUpdated:    profile.ProfileUpdated,
		Valid:             profile.Valid,
		Metadata:          profile.Metadata,
	}
}