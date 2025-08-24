package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go-falcon/internal/auth/models"
	"go-falcon/pkg/evegateway"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// ProfileService handles user profile operations
type ProfileService struct {
	repository *Repository
	eveService *EVEService
	esiClient  *evegateway.Client
}

// NewProfileService creates a new profile service
func NewProfileService(repository *Repository, eveService *EVEService, esiClient *evegateway.Client) *ProfileService {
	return &ProfileService{
		repository: repository,
		eveService: eveService,
		esiClient:  esiClient,
	}
}

// CreateOrUpdateProfile creates or updates a user profile
func (s *ProfileService) CreateOrUpdateProfile(ctx context.Context, charInfo *models.EVECharacterInfo, userID, accessToken, refreshToken string) (*models.UserProfile, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.profile_service.create_or_update")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "create_or_update_profile"),
		attribute.Int("character_id", charInfo.CharacterID),
		attribute.String("character_name", charInfo.CharacterName),
	)

	// If no userID provided, generate one
	if userID == "" {
		userID = uuid.New().String()
	}

	// Calculate token expiry from ExpiresOn
	var tokenExpiry time.Time
	if charInfo.ExpiresOn != "" {
		if parsed, err := time.Parse(time.RFC3339, charInfo.ExpiresOn); err == nil {
			tokenExpiry = parsed
		} else {
			// Fallback to 20 minutes from now
			tokenExpiry = time.Now().Add(20 * time.Minute)
		}
	} else {
		tokenExpiry = time.Now().Add(20 * time.Minute)
	}

	// Create profile model
	profile := &models.UserProfile{
		UserID:             userID,
		CharacterID:        charInfo.CharacterID,
		CharacterName:      charInfo.CharacterName,
		CharacterOwnerHash: charInfo.CharacterOwnerHash,
		Scopes:             charInfo.Scopes,
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		TokenExpiry:        tokenExpiry,
		LastLogin:          time.Now(),
		ProfileUpdated:     time.Now(),
		Valid:              true,
		Metadata:           make(map[string]string),
	}

	// Enhance profile with ESI data
	if err := s.enrichProfileWithESI(ctx, profile, accessToken); err != nil {
		slog.Warn("Failed to enrich profile with ESI data", "error", err, "character_id", charInfo.CharacterID)
		// Continue without ESI data - this is not a fatal error
	}

	// Save to database
	savedProfile, err := s.repository.CreateOrUpdateUserProfile(ctx, profile)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to save profile: %w", err)
	}

	slog.Info("Profile created/updated successfully",
		"character_id", charInfo.CharacterID,
		"character_name", charInfo.CharacterName,
		"user_id", userID,
	)

	return savedProfile, nil
}

// GetProfile retrieves a user profile by character ID
func (s *ProfileService) GetProfile(ctx context.Context, characterID int) (*models.UserProfile, error) {
	return s.repository.GetUserProfileByCharacterID(ctx, characterID)
}

// GetProfileByUserID retrieves a user profile by user ID
func (s *ProfileService) GetProfileByUserID(ctx context.Context, userID string) (*models.UserProfile, error) {
	return s.repository.GetUserProfileByUserID(ctx, userID)
}

// RefreshProfile refreshes profile data from ESI
func (s *ProfileService) RefreshProfile(ctx context.Context, characterID int) (*models.UserProfile, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.profile_service.refresh_profile")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "refresh_profile"),
		attribute.Int("character_id", characterID),
	)

	// Get existing profile
	profile, err := s.repository.GetUserProfileByCharacterID(ctx, characterID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	if profile == nil {
		return nil, fmt.Errorf("profile not found for character %d", characterID)
	}

	// Enrich with fresh ESI data
	if err := s.enrichProfileWithESI(ctx, profile, profile.AccessToken); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to refresh ESI data: %w", err)
	}

	// Update profile updated timestamp
	profile.ProfileUpdated = time.Now()

	// Save updated profile
	savedProfile, err := s.repository.CreateOrUpdateUserProfile(ctx, profile)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to save refreshed profile: %w", err)
	}

	return savedProfile, nil
}

// RefreshExpiringTokens refreshes tokens that are expiring soon
func (s *ProfileService) RefreshExpiringTokens(ctx context.Context, batchSize int) (int, int, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.profile_service.refresh_expiring_tokens")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "refresh_expiring_tokens"),
		attribute.Int("batch_size", batchSize),
	)

	// Get profiles with tokens expiring in the next hour
	expiringTime := time.Now().Add(1 * time.Hour)
	profiles, err := s.repository.GetExpiringTokens(ctx, expiringTime, batchSize)
	if err != nil {
		span.RecordError(err)
		return 0, 0, fmt.Errorf("failed to get expiring tokens: %w", err)
	}

	successCount := 0
	failureCount := 0

	for _, profile := range profiles {
		if err := s.refreshSingleToken(ctx, profile.CharacterID, profile.RefreshToken); err != nil {
			slog.Error("Failed to refresh token",
				"character_id", profile.CharacterID,
				"character_name", profile.CharacterName,
				"error", err,
			)
			failureCount++
		} else {
			slog.Info("Token refreshed successfully",
				"character_id", profile.CharacterID,
				"character_name", profile.CharacterName,
			)
			successCount++
		}
	}

	span.SetAttributes(
		attribute.Int("total_processed", len(profiles)),
		attribute.Int("success_count", successCount),
		attribute.Int("failure_count", failureCount),
	)

	return successCount, failureCount, nil
}

// enrichProfileWithESI enriches profile with ESI character data
func (s *ProfileService) enrichProfileWithESI(ctx context.Context, profile *models.UserProfile, accessToken string) error {
	// Get character information from ESI
	charInfo, err := s.getESICharacterInfo(ctx, profile.CharacterID, accessToken)
	if err != nil {
		return fmt.Errorf("failed to get character info: %w", err)
	}

	// Update profile with ESI data
	profile.CorporationID = charInfo.CorporationID
	profile.AllianceID = charInfo.AllianceID
	profile.SecurityStatus = charInfo.SecurityStatus
	profile.Birthday = charInfo.Birthday

	// Get corporation name
	if profile.CorporationID > 0 {
		if corpInfo, err := s.getESICorporationInfo(ctx, profile.CorporationID); err == nil {
			profile.CorporationName = corpInfo.Name
			profile.AllianceID = corpInfo.AllianceID // Update alliance from corp info
		}
	}

	// Get alliance name
	if profile.AllianceID > 0 {
		if allianceInfo, err := s.getESIAllianceInfo(ctx, profile.AllianceID); err == nil {
			profile.AllianceName = allianceInfo.Name
		}
	}

	return nil
}

// refreshSingleToken refreshes a single user's access token
func (s *ProfileService) refreshSingleToken(ctx context.Context, characterID int, refreshToken string) error {
	// Refresh the token
	tokenResp, err := s.eveService.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		// If refresh fails, mark profile as invalid
		s.repository.InvalidateProfile(ctx, characterID)
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Calculate new expiry time
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Update tokens in database
	err = s.repository.UpdateProfileTokens(ctx, characterID, tokenResp.AccessToken, tokenResp.RefreshToken, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to update tokens: %w", err)
	}

	return nil
}

// getESICharacterInfo retrieves character information from ESI
func (s *ProfileService) getESICharacterInfo(ctx context.Context, characterID int, accessToken string) (*models.ESICharacterInfo, error) {
	url := fmt.Sprintf("https://esi.evetech.net/latest/characters/%d/", characterID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "go-falcon/1.0.0 (contact@example.com)")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ESI character request failed: %s - %s", resp.Status, string(body))
	}

	var charInfo models.ESICharacterInfo
	if err := json.NewDecoder(resp.Body).Decode(&charInfo); err != nil {
		return nil, err
	}

	return &charInfo, nil
}

// getESICorporationInfo retrieves corporation information from ESI
func (s *ProfileService) getESICorporationInfo(ctx context.Context, corporationID int) (*models.ESICorporationInfo, error) {
	url := fmt.Sprintf("https://esi.evetech.net/latest/corporations/%d/", corporationID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "go-falcon/1.0.0 (contact@example.com)")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ESI corporation request failed: %s", resp.Status)
	}

	var corpInfo models.ESICorporationInfo
	if err := json.NewDecoder(resp.Body).Decode(&corpInfo); err != nil {
		return nil, err
	}

	return &corpInfo, nil
}

// getESIAllianceInfo retrieves alliance information from ESI
func (s *ProfileService) getESIAllianceInfo(ctx context.Context, allianceID int) (*models.ESIAllianceInfo, error) {
	url := fmt.Sprintf("https://esi.evetech.net/latest/alliances/%d/", allianceID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "go-falcon/1.0.0 (contact@example.com)")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ESI alliance request failed: %s", resp.Status)
	}

	var allianceInfo models.ESIAllianceInfo
	if err := json.NewDecoder(resp.Body).Decode(&allianceInfo); err != nil {
		return nil, err
	}

	return &allianceInfo, nil
}
