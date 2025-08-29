package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	authModels "go-falcon/internal/auth/models"
	characterServices "go-falcon/internal/character/services"
	"go-falcon/internal/corporation/dto"
	"go-falcon/internal/corporation/models"
	"go-falcon/pkg/evegateway"
	evegatewayTypes "go-falcon/pkg/evegateway/corporation"
	"go-falcon/pkg/sde"

	"go.mongodb.org/mongo-driver/mongo"
)

// Service handles corporation business logic
type Service struct {
	repository       *Repository
	eveClient        *evegateway.Client
	characterService *characterServices.Service
	sdeService       sde.SDEService
	authService      AuthService
}

// AuthService interface for auth operations we need
type AuthService interface {
	GetUserProfileByCharacterID(ctx context.Context, characterID int) (*authModels.UserProfile, error)
}

// NewService creates a new corporation service
func NewService(repository *Repository, eveClient *evegateway.Client, characterService *characterServices.Service, sdeService sde.SDEService, authService AuthService) *Service {
	return &Service{
		repository:       repository,
		eveClient:        eveClient,
		characterService: characterService,
		sdeService:       sdeService,
		authService:      authService,
	}
}

// GetCorporationInfo retrieves corporation information, first checking the database,
// then falling back to EVE ESI if not found or data is stale
func (s *Service) GetCorporationInfo(ctx context.Context, corporationID int) (*dto.CorporationInfoOutput, error) {
	slog.InfoContext(ctx, "Getting corporation info", "corporation_id", corporationID)

	// Try to get from database first
	corporation, err := s.repository.GetCorporationByID(ctx, corporationID)
	if err != nil && err != mongo.ErrNoDocuments {
		slog.ErrorContext(ctx, "Failed to get corporation from database", "error", err)
		// Continue to fetch from ESI
	}

	// If not found in database or data might be stale, fetch from ESI
	if corporation == nil || err == mongo.ErrNoDocuments {
		slog.InfoContext(ctx, "Corporation not found in database, fetching from ESI", "corporation_id", corporationID)

		// Get corporation info from EVE ESI
		esiData, err := s.eveClient.GetCorporationInfo(ctx, corporationID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get corporation from ESI", "error", err)
			return nil, fmt.Errorf("failed to get corporation information: %w", err)
		}

		// Convert ESI data to our model
		corporation = s.convertESIDataToModel(esiData, corporationID)

		// Save to database for future use
		if err := s.repository.UpdateCorporation(ctx, corporation); err != nil {
			slog.WarnContext(ctx, "Failed to save corporation to database", "error", err)
			// Don't fail the request if we can't save to DB
		}
	}

	// Convert model to output DTO
	return s.convertModelToOutput(ctx, corporation), nil
}

// convertESIDataToModel converts ESI response data to our corporation model
func (s *Service) convertESIDataToModel(esiData map[string]any, corporationID int) *models.Corporation {
	now := time.Now().UTC()
	corporation := &models.Corporation{
		CorporationID: corporationID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if name, ok := esiData["name"].(string); ok {
		corporation.Name = name
	}
	if ticker, ok := esiData["ticker"].(string); ok {
		corporation.Ticker = ticker
	}
	if description, ok := esiData["description"].(string); ok {
		corporation.Description = description
	}
	if url, ok := esiData["url"].(string); ok && url != "" {
		corporation.URL = &url
	}
	// Handle date_founded
	if dateFounded, ok := esiData["date_founded"].(string); ok {
		if parsedTime, err := time.Parse(time.RFC3339, dateFounded); err == nil {
			corporation.DateFounded = parsedTime
		}
	}

	// Handle numeric fields - can be int, float64, or string depending on source
	if allianceID, ok := esiData["alliance_id"].(int); ok {
		corporation.AllianceID = &allianceID
	} else if allianceIDFloat, ok := esiData["alliance_id"].(float64); ok {
		allianceIDInt := int(allianceIDFloat)
		corporation.AllianceID = &allianceIDInt
	}

	if creatorID, ok := esiData["creator_id"].(int); ok {
		corporation.CreatorID = creatorID
	} else if creatorIDFloat, ok := esiData["creator_id"].(float64); ok {
		corporation.CreatorID = int(creatorIDFloat)
	}
	if factionID, ok := esiData["faction_id"].(int); ok {
		corporation.FactionID = &factionID
	} else if factionIDFloat, ok := esiData["faction_id"].(float64); ok {
		factionIDInt := int(factionIDFloat)
		corporation.FactionID = &factionIDInt
	}
	if homeStationID, ok := esiData["home_station_id"].(int); ok {
		corporation.HomeStationID = &homeStationID
	} else if homeStationIDFloat, ok := esiData["home_station_id"].(float64); ok {
		homeStationIDInt := int(homeStationIDFloat)
		corporation.HomeStationID = &homeStationIDInt
	}
	if memberCount, ok := esiData["member_count"].(int); ok {
		corporation.MemberCount = memberCount
	} else if memberCountFloat, ok := esiData["member_count"].(float64); ok {
		corporation.MemberCount = int(memberCountFloat)
	}
	if shares, ok := esiData["shares"].(int64); ok {
		corporation.Shares = &shares
	} else if sharesInt, ok := esiData["shares"].(int); ok {
		sharesInt64 := int64(sharesInt)
		corporation.Shares = &sharesInt64
	} else if sharesFloat, ok := esiData["shares"].(float64); ok {
		sharesInt64 := int64(sharesFloat)
		corporation.Shares = &sharesInt64
	}
	if taxRate, ok := esiData["tax_rate"].(float64); ok {
		corporation.TaxRate = taxRate
	}
	if warEligible, ok := esiData["war_eligible"].(bool); ok {
		corporation.WarEligible = &warEligible
	}

	return corporation
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// convertModelToOutput converts corporation model to output DTO
func (s *Service) convertModelToOutput(ctx context.Context, corporation *models.Corporation) *dto.CorporationInfoOutput {
	corporationInfo := dto.CorporationInfo{
		AllianceID:     corporation.AllianceID,
		CEOCharacterID: corporation.CEOID,
		CreatorID:      corporation.CreatorID,
		DateFounded:    corporation.DateFounded,
		Description:    corporation.Description,
		FactionID:      corporation.FactionID,
		HomeStationID:  corporation.HomeStationID,
		MemberCount:    corporation.MemberCount,
		Name:           corporation.Name,
		Shares:         corporation.Shares,
		TaxRate:        corporation.TaxRate,
		Ticker:         corporation.Ticker,
		URL:            corporation.URL,
		WarEligible:    corporation.WarEligible,
	}

	// Fetch CEO character information
	if corporation.CEOID > 0 {
		ceoProfile, err := s.characterService.GetCharacterProfile(ctx, corporation.CEOID)
		if err != nil {
			slog.WarnContext(ctx, "Failed to get CEO character info", "ceo_id", corporation.CEOID, "error", err)
		} else if ceoProfile != nil && ceoProfile.Body.CharacterID > 0 {
			corporationInfo.CEO = &dto.CharacterInfo{
				CharacterID: ceoProfile.Body.CharacterID,
				Name:        ceoProfile.Body.Name,
			}
		}
	}

	// Fetch Creator character information
	if corporation.CreatorID > 0 {
		creatorProfile, err := s.characterService.GetCharacterProfile(ctx, corporation.CreatorID)
		if err != nil {
			slog.WarnContext(ctx, "Failed to get creator character info", "creator_id", corporation.CreatorID, "error", err)
		} else if creatorProfile != nil && creatorProfile.Body.CharacterID > 0 {
			corporationInfo.Creator = &dto.CharacterInfo{
				CharacterID: creatorProfile.Body.CharacterID,
				Name:        creatorProfile.Body.Name,
			}
		}
	}

	// Fetch home station information from SDE
	if corporation.HomeStationID != nil && *corporation.HomeStationID > 0 {
		station, err := s.sdeService.GetStaStation(*corporation.HomeStationID)
		if err != nil {
			slog.WarnContext(ctx, "Failed to get home station info from SDE", "station_id", *corporation.HomeStationID, "error", err)
		} else if station != nil {
			corporationInfo.HomeStation = &dto.StationInfo{
				StationID:                station.StationID,
				ConstellationID:          station.ConstellationID,
				SolarSystemID:            station.SolarSystemID,
				RegionID:                 station.RegionID,
				CorporationID:            station.CorporationID,
				DockingCostPerVolume:     station.DockingCostPerVolume,
				MaxShipVolumeDockable:    station.MaxShipVolumeDockable,
				OfficeRentalCost:         station.OfficeRentalCost,
				ReprocessingEfficiency:   station.ReprocessingEfficiency,
				ReprocessingStationsTake: station.ReprocessingStationsTake,
				Security:                 station.Security,
			}
		}
	}

	return &dto.CorporationInfoOutput{
		Body: corporationInfo,
	}
}

// SearchCorporationsByName searches corporations by name or ticker
func (s *Service) SearchCorporationsByName(ctx context.Context, name string) (*dto.SearchCorporationsByNameOutput, error) {
	slog.InfoContext(ctx, "Searching corporations by name", "name", name)

	corporations, err := s.repository.SearchCorporationsByName(ctx, name)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to search corporations", "error", err)
		return nil, fmt.Errorf("failed to search corporations: %w", err)
	}

	// Convert models to search DTOs
	searchResults := make([]dto.CorporationSearchInfo, len(corporations))
	for i, corp := range corporations {
		searchResults[i] = dto.CorporationSearchInfo{
			CorporationID:  corp.CorporationID,
			Name:           corp.Name,
			Ticker:         corp.Ticker,
			CEOCharacterID: 0, // CEO ID not available in search results to avoid ESI calls
			MemberCount:    corp.MemberCount,
			AllianceID:     corp.AllianceID,
			UpdatedAt:      corp.UpdatedAt,
		}
	}

	result := &dto.SearchCorporationsByNameOutput{
		Body: dto.SearchCorporationsResult{
			Corporations: searchResults,
			Count:        len(searchResults),
		},
	}

	slog.InfoContext(ctx, "Corporation search completed", "count", len(searchResults))
	return result, nil
}

// UpdateAllCorporations updates all corporations in the database by fetching fresh data from ESI
func (s *Service) UpdateAllCorporations(ctx context.Context, concurrentWorkers int) error {
	slog.InfoContext(ctx, "Starting update of all corporations", "concurrent_workers", concurrentWorkers)

	// Get all corporation IDs from database
	corporationIDs, err := s.repository.GetAllCorporationIDs(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get corporation IDs from database", "error", err)
		return fmt.Errorf("failed to get corporation IDs: %w", err)
	}

	totalCount := len(corporationIDs)
	if totalCount == 0 {
		slog.InfoContext(ctx, "No corporations found in database to update")
		return nil
	}

	slog.InfoContext(ctx, "Found corporations to update", "total_count", totalCount)

	// Create channels for work distribution
	type updateResult struct {
		corporationID int
		success       bool
		err           error
	}

	jobs := make(chan int, totalCount)
	results := make(chan updateResult, totalCount)

	// Start workers
	for w := 1; w <= concurrentWorkers; w++ {
		go func(workerID int) {
			for corporationID := range jobs {
				// Create a new context for each update to avoid context cancellation issues
				updateCtx := context.Background()

				// Get corporation info from EVE ESI
				esiData, err := s.eveClient.GetCorporationInfo(updateCtx, corporationID)
				if err != nil {
					slog.WarnContext(updateCtx, "Failed to get corporation from ESI",
						"worker_id", workerID,
						"corporation_id", corporationID,
						"error", err)
					results <- updateResult{
						corporationID: corporationID,
						success:       false,
						err:           err,
					}
					continue
				}

				// Convert ESI data to our model
				corporation := s.convertESIDataToModel(esiData, corporationID)

				// Update in database
				if err := s.repository.UpdateCorporation(updateCtx, corporation); err != nil {
					slog.WarnContext(updateCtx, "Failed to update corporation in database",
						"worker_id", workerID,
						"corporation_id", corporationID,
						"error", err)
					results <- updateResult{
						corporationID: corporationID,
						success:       false,
						err:           err,
					}
					continue
				}

				slog.DebugContext(updateCtx, "Successfully updated corporation",
					"worker_id", workerID,
					"corporation_id", corporationID,
					"corporation_name", corporation.Name)

				results <- updateResult{
					corporationID: corporationID,
					success:       true,
					err:           nil,
				}

				// Small delay to respect ESI rate limits
				time.Sleep(50 * time.Millisecond)
			}
		}(w)
	}

	// Send all jobs
	for _, corporationID := range corporationIDs {
		jobs <- corporationID
	}
	close(jobs)

	// Collect results
	successCount := 0
	failureCount := 0

	for i := 0; i < totalCount; i++ {
		result := <-results
		if result.success {
			successCount++
		} else {
			failureCount++
		}

		// Log progress every 100 corporations
		if (i+1)%100 == 0 || (i+1) == totalCount {
			slog.InfoContext(ctx, "Corporation update progress",
				"processed", i+1,
				"total", totalCount,
				"success", successCount,
				"failures", failureCount,
				"progress_percent", fmt.Sprintf("%.1f%%", float64(i+1)/float64(totalCount)*100))
		}
	}

	slog.InfoContext(ctx, "Completed updating all corporations",
		"total_processed", totalCount,
		"successful", successCount,
		"failed", failureCount)

	if failureCount > 0 {
		return fmt.Errorf("failed to update %d out of %d corporations", failureCount, totalCount)
	}

	return nil
}

// ValidateCEOTokens checks if all CEO characters have valid tokens
// TODO: This will be enhanced with a notification system for invalid tokens
func (s *Service) ValidateCEOTokens(ctx context.Context) error {
	slog.InfoContext(ctx, "Starting CEO token validation")

	// Check if auth service is available
	if s.authService == nil {
		slog.WarnContext(ctx, "Auth service not available, skipping CEO token validation")
		return nil
	}

	// Get all CEO IDs from enabled corporations
	ceoIDs, err := s.repository.GetCEOIDsFromEnabledCorporations(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get CEO IDs from enabled corporations", "error", err)
		return fmt.Errorf("failed to get CEO IDs: %w", err)
	}

	if len(ceoIDs) == 0 {
		slog.InfoContext(ctx, "No CEOs found in enabled corporations")
		return nil
	}

	slog.InfoContext(ctx, "Found CEOs to validate", "count", len(ceoIDs))

	invalidTokenCount := 0
	validTokenCount := 0
	noProfileCount := 0

	// Check each CEO's token validity
	for _, ceoID := range ceoIDs {
		// Get the user profile for this CEO from the auth service
		profile, err := s.authService.GetUserProfileByCharacterID(ctx, ceoID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				// CEO doesn't have a profile/token at all
				noProfileCount++
				slog.WarnContext(ctx, "CEO has no user profile/token",
					"ceo_character_id", ceoID)
			} else {
				// Other error occurred
				slog.ErrorContext(ctx, "Failed to get CEO profile",
					"ceo_character_id", ceoID,
					"error", err)
			}
			continue
		}

		// Check if profile is nil (can happen when no document exists but no error is returned)
		if profile == nil {
			noProfileCount++
			slog.WarnContext(ctx, "CEO has no user profile/token",
				"ceo_character_id", ceoID)
			continue
		}

		// Check if the token is valid
		if profile.Valid {
			validTokenCount++
			slog.DebugContext(ctx, "CEO has valid token",
				"ceo_character_id", ceoID,
				"character_name", profile.CharacterName,
				"corporation_id", profile.CorporationID)
		} else {
			invalidTokenCount++
			slog.WarnContext(ctx, "CEO has invalid token",
				"ceo_character_id", ceoID,
				"character_name", profile.CharacterName,
				"corporation_id", profile.CorporationID,
				"last_login", profile.LastLogin,
				"token_expiry", profile.TokenExpiry)
		}
	}

	slog.InfoContext(ctx, "CEO token validation completed",
		"total_ceos", len(ceoIDs),
		"valid_tokens", validTokenCount,
		"invalid_tokens", invalidTokenCount,
		"no_profile", noProfileCount)

	if invalidTokenCount > 0 {
		slog.WarnContext(ctx, "Found CEOs with invalid tokens", "count", invalidTokenCount)
		// TODO: Implement notification system integration here
	}

	if noProfileCount > 0 {
		slog.WarnContext(ctx, "Found CEOs with no user profile", "count", noProfileCount)
		// TODO: These CEOs need to authenticate with the system
	}

	return nil
}

// ValidateCEOTokensWithResults checks if all CEO characters have valid tokens and returns detailed results
func (s *Service) ValidateCEOTokensWithResults(ctx context.Context) (*dto.CEOTokenValidationResult, error) {
	slog.InfoContext(ctx, "Starting CEO token validation with results")

	result := &dto.CEOTokenValidationResult{
		InvalidCEOs: []dto.CEOTokenInfo{},
		MissingCEOs: []int{},
		ExecutedAt:  time.Now().UTC(),
	}

	// Check if auth service is available
	if s.authService == nil {
		slog.WarnContext(ctx, "Auth service not available, skipping CEO token validation")
		return result, fmt.Errorf("auth service not available")
	}

	// Get all CEO IDs from enabled corporations
	ceoIDs, err := s.repository.GetCEOIDsFromEnabledCorporations(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get CEO IDs from enabled corporations", "error", err)
		return nil, fmt.Errorf("failed to get CEO IDs: %w", err)
	}

	result.TotalCEOs = len(ceoIDs)

	if len(ceoIDs) == 0 {
		slog.InfoContext(ctx, "No CEOs found in enabled corporations")
		return result, nil
	}

	slog.InfoContext(ctx, "Found CEOs to validate", "count", len(ceoIDs))

	// Check each CEO's token validity
	for _, ceoID := range ceoIDs {
		// Get the user profile for this CEO from the auth service
		profile, err := s.authService.GetUserProfileByCharacterID(ctx, ceoID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				// CEO doesn't have a profile/token at all
				result.NoProfile++
				result.MissingCEOs = append(result.MissingCEOs, ceoID)
				slog.WarnContext(ctx, "CEO has no user profile/token",
					"ceo_character_id", ceoID)
			} else {
				// Other error occurred
				slog.ErrorContext(ctx, "Failed to get CEO profile",
					"ceo_character_id", ceoID,
					"error", err)
			}
			continue
		}

		// Check if profile is nil (can happen when no document exists but no error is returned)
		if profile == nil {
			result.NoProfile++
			result.MissingCEOs = append(result.MissingCEOs, ceoID)
			slog.WarnContext(ctx, "CEO has no user profile/token",
				"ceo_character_id", ceoID)
			continue
		}

		// Check if the token is valid
		if profile.Valid {
			result.ValidTokens++
			slog.DebugContext(ctx, "CEO has valid token",
				"ceo_character_id", ceoID,
				"character_name", profile.CharacterName,
				"corporation_id", profile.CorporationID)
		} else {
			result.InvalidTokens++
			// Add to invalid CEOs list
			ceoInfo := dto.CEOTokenInfo{
				CharacterID:     ceoID,
				CharacterName:   profile.CharacterName,
				CorporationID:   profile.CorporationID,
				CorporationName: profile.CorporationName,
				Valid:           false,
				TokenExpiry:     &profile.TokenExpiry,
				LastLogin:       &profile.LastLogin,
			}
			result.InvalidCEOs = append(result.InvalidCEOs, ceoInfo)

			slog.WarnContext(ctx, "CEO has invalid token",
				"ceo_character_id", ceoID,
				"character_name", profile.CharacterName,
				"corporation_id", profile.CorporationID,
				"last_login", profile.LastLogin,
				"token_expiry", profile.TokenExpiry)
		}
	}

	slog.InfoContext(ctx, "CEO token validation completed",
		"total_ceos", result.TotalCEOs,
		"valid_tokens", result.ValidTokens,
		"invalid_tokens", result.InvalidTokens,
		"no_profile", result.NoProfile)

	if result.InvalidTokens > 0 {
		slog.WarnContext(ctx, "Found CEOs with invalid tokens", "count", result.InvalidTokens)
	}

	if result.NoProfile > 0 {
		slog.WarnContext(ctx, "Found CEOs with no user profile", "count", result.NoProfile)
	}

	return result, nil
}

// GetMemberTracking retrieves member tracking information for a corporation
func (s *Service) GetMemberTracking(ctx context.Context, corporationID int, ceoID int) (*dto.CorporationMemberTrackingOutput, error) {
	slog.InfoContext(ctx, "Getting member tracking", "corporation_id", corporationID, "ceo_id", ceoID)

	// First verify that the CEO ID matches the corporation's CEO
	corporation, err := s.repository.GetCorporationByID(ctx, corporationID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("corporation not found: %d", corporationID)
		}
		slog.ErrorContext(ctx, "Failed to get corporation from database", "error", err)
		return nil, fmt.Errorf("failed to get corporation: %w", err)
	}

	// Check if the provided CEO ID matches the corporation's CEO
	if corporation.CEOID != ceoID {
		slog.WarnContext(ctx, "CEO ID mismatch",
			"provided_ceo_id", ceoID,
			"actual_ceo_id", corporation.CEOID,
			"corporation_id", corporationID)
		return nil, fmt.Errorf("invalid CEO ID for corporation %d", corporationID)
	}

	// Get CEO's profile to get their access token
	ceoProfile, err := s.authService.GetUserProfileByCharacterID(ctx, ceoID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get CEO profile", "ceo_id", ceoID, "error", err)
		return nil, fmt.Errorf("failed to get CEO profile: %w", err)
	}
	if ceoProfile == nil {
		return nil, fmt.Errorf("CEO profile not found for character ID %d", ceoID)
	}

	// Check if the CEO's token is valid
	if ceoProfile.AccessToken == "" {
		return nil, fmt.Errorf("CEO does not have a valid access token")
	}

	// Get member tracking from EVE ESI using the CEO's token
	trackingData, err := s.eveClient.Corporation.GetCorporationMemberTracking(ctx, corporationID, ceoProfile.AccessToken)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get member tracking from ESI", "error", err)
		return nil, fmt.Errorf("failed to get member tracking: %w", err)
	}

	// Convert ESI data to our models and save to database
	trackingModels := make([]*models.TrackCorporationMember, len(trackingData))
	for i, member := range trackingData {
		trackingModels[i] = s.convertESIMemberTrackingToModel(member, corporationID)
	}

	// Save all tracking data to database
	if err := s.repository.UpdateMemberTracking(ctx, corporationID, trackingModels); err != nil {
		slog.WarnContext(ctx, "Failed to save member tracking to database", "error", err)
		// Don't fail the request if we can't save to DB
	}

	// Convert to output DTO
	members := make([]dto.MemberTrackingInfo, len(trackingData))
	for i, member := range trackingData {
		// Look up location name based on location ID type
		var locationName *string
		if member.LocationID != 0 {
			locationName = s.getLocationName(ctx, member.LocationID, member.CharacterID)
		}

		members[i] = dto.MemberTrackingInfo{
			BaseID:       convertIntToPointer(member.BaseID),
			CharacterID:  member.CharacterID,
			LocationID:   convertInt64ToPointer(int64(member.LocationID)),
			LocationName: locationName,
			LogoffDate:   convertTimeToPointer(member.LogoffDate),
			LogonDate:    convertTimeToPointer(member.LogonDate),
			ShipTypeID:   convertIntToPointer(member.ShipTypeID),
			StartDate:    convertTimeToPointer(member.StartDate),
		}
	}

	result := &dto.CorporationMemberTrackingOutput{
		Body: dto.MemberTrackingResult{
			CorporationID: corporationID,
			Members:       members,
			Count:         len(members),
		},
	}

	slog.InfoContext(ctx, "Member tracking completed", "corporation_id", corporationID, "member_count", len(members))
	return result, nil
}

// convertESIMemberTrackingToModel converts ESI member tracking data to our model
func (s *Service) convertESIMemberTrackingToModel(esiData evegatewayTypes.CorporationMemberTracking, corporationID int) *models.TrackCorporationMember {
	now := time.Now().UTC()
	tracking := &models.TrackCorporationMember{
		CorporationID: corporationID,
		CharacterID:   esiData.CharacterID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Handle optional fields
	if esiData.BaseID != 0 {
		tracking.BaseID = &esiData.BaseID
	}
	if esiData.LocationID != 0 {
		locationID := int64(esiData.LocationID)
		tracking.LocationID = &locationID
	}
	if !esiData.LogoffDate.IsZero() {
		tracking.LogoffDate = &esiData.LogoffDate
	}
	if !esiData.LogonDate.IsZero() {
		tracking.LogonDate = &esiData.LogonDate
	}
	if esiData.ShipTypeID != 0 {
		tracking.ShipTypeID = &esiData.ShipTypeID
	}
	if !esiData.StartDate.IsZero() {
		tracking.StartDate = &esiData.StartDate
	}

	return tracking
}

// getLocationName retrieves the location name for a given location ID
// Stations (60,000,000 - 69,999,999) are looked up from SDE
// Structures (outside that range) are looked up from database
func (s *Service) getLocationName(ctx context.Context, locationID int64, characterID int) *string {
	// Check if it's a station (NPC station range: 60,000,000 - 69,999,999)
	if locationID >= 60000000 && locationID <= 69999999 {
		// It's a station - look up from SDE
		if invName, err := s.sdeService.GetInvName(int(locationID)); err == nil && invName != nil {
			// Handle the interface{} type for ItemName
			if nameStr, ok := invName.ItemName.(string); ok && nameStr != "" {
				return &nameStr
			}
		} else {
			// Log the station lookup failure
			slog.DebugContext(ctx, "Failed to lookup station name from SDE",
				"location_id", locationID,
				"character_id", characterID,
				"location_type", "station",
				"error", err)
		}
	} else {
		// It's a structure - look up from database
		if structure, err := s.repository.GetStructureByID(ctx, locationID); err == nil && structure != nil {
			if structure.Name != "" {
				return &structure.Name
			}
		} else if err != mongo.ErrNoDocuments {
			// Log database errors (but not "not found" errors)
			slog.DebugContext(ctx, "Failed to lookup structure from database",
				"location_id", locationID,
				"character_id", characterID,
				"location_type", "structure",
				"error", err)
		} else {
			// Structure not found in database - this is expected for many structures
			slog.DebugContext(ctx, "Structure not found in database",
				"location_id", locationID,
				"character_id", characterID,
				"location_type", "structure")
		}
	}

	// Return null if no name could be found
	return nil
}

// Helper functions to convert values to pointers
func convertIntToPointer(val int) *int {
	if val == 0 {
		return nil
	}
	return &val
}

func convertInt64ToPointer(val int64) *int64 {
	if val == 0 {
		return nil
	}
	return &val
}

func convertTimeToPointer(val time.Time) *time.Time {
	if val.IsZero() {
		return nil
	}
	return &val
}

// fetchStructureFromESI fetches structure information from ESI and saves it to database
// TODO: Implement when universe structure endpoint is added to EVE Gateway
// ESI endpoint: GET /universe/structures/{structure_id}
// Requires: esi-universe.read_structures.v1 scope
func (s *Service) fetchStructureFromESI(ctx context.Context, structureID int64, token string) (*models.Structure, error) {
	// TODO: Implement ESI structure lookup using:
	// GET /universe/structures/{structure_id}
	//
	// Example implementation structure:
	// 1. Call s.eveClient.Universe.GetStructure(ctx, structureID, token)
	// 2. Convert ESI response to models.Structure
	// 3. Save to database using s.repository.UpdateStructure(ctx, structure)
	// 4. Return the structure

	slog.DebugContext(ctx, "Structure ESI lookup not yet implemented",
		"structure_id", structureID)

	return nil, fmt.Errorf("structure ESI lookup not yet implemented")
}

// GetCorporationAllianceHistory retrieves the alliance history for a corporation from EVE ESI
func (s *Service) GetCorporationAllianceHistory(ctx context.Context, corporationID int) (*dto.CorporationAllianceHistoryOutput, error) {
	slog.InfoContext(ctx, "Getting corporation alliance history", "corporation_id", corporationID)

	// Fetch alliance history from ESI
	history, err := s.eveClient.Corporation.GetCorporationAllianceHistory(ctx, corporationID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get corporation alliance history from ESI", "error", err, "corporation_id", corporationID)
		return nil, fmt.Errorf("failed to get corporation alliance history: %w", err)
	}

	// Convert ESI data to our DTO format
	result := &dto.CorporationAllianceHistoryResult{
		CorporationID: corporationID,
		History:       make([]dto.AllianceHistoryEntry, 0, len(history)),
		Count:         len(history),
	}

	for _, entry := range history {
		historyEntry := dto.AllianceHistoryEntry{
			RecordID:  entry.RecordID,
			StartDate: entry.StartDate,
			IsDeleted: entry.IsDeleted,
		}

		// AllianceID is optional (null when corporation left all alliances)
		if entry.AllianceID != 0 {
			historyEntry.AllianceID = &entry.AllianceID
		}

		result.History = append(result.History, historyEntry)
	}

	slog.InfoContext(ctx, "Successfully retrieved corporation alliance history",
		"corporation_id", corporationID,
		"history_count", result.Count)

	return &dto.CorporationAllianceHistoryOutput{
		Body: *result,
	}, nil
}

// GetCorporationMembers retrieves corporation members from ESI
func (s *Service) GetCorporationMembers(ctx context.Context, corporationID int, ceoID int) (*dto.CorporationMembersOutput, error) {
	slog.InfoContext(ctx, "Getting corporation members", "corporation_id", corporationID, "ceo_id", ceoID)

	// First verify that the CEO ID matches the corporation's CEO
	corporation, err := s.repository.GetCorporationByID(ctx, corporationID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("corporation not found: %d", corporationID)
		}
		slog.ErrorContext(ctx, "Failed to get corporation from database", "error", err)
		return nil, fmt.Errorf("failed to get corporation: %w", err)
	}

	// Check if the provided CEO ID matches the corporation's CEO
	if corporation.CEOID != ceoID {
		slog.WarnContext(ctx, "CEO ID mismatch",
			"provided_ceo_id", ceoID,
			"actual_ceo_id", corporation.CEOID,
			"corporation_id", corporationID)
		return nil, fmt.Errorf("invalid CEO ID for corporation %d", corporationID)
	}

	// Get CEO's profile to get their access token
	ceoProfile, err := s.authService.GetUserProfileByCharacterID(ctx, ceoID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get CEO profile", "ceo_id", ceoID, "error", err)
		return nil, fmt.Errorf("failed to get CEO profile: %w", err)
	}
	if ceoProfile == nil {
		return nil, fmt.Errorf("CEO profile not found for character ID %d", ceoID)
	}

	// Check if the CEO's token is valid
	if ceoProfile.AccessToken == "" {
		return nil, fmt.Errorf("CEO does not have a valid access token")
	}

	// Get members from ESI
	members, err := s.eveClient.Corporation.GetCorporationMembers(ctx, corporationID, ceoProfile.AccessToken)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get corporation members from ESI", "error", err, "corporation_id", corporationID)
		return nil, fmt.Errorf("failed to get corporation members: %w", err)
	}

	// Convert ESI data to our DTO format
	result := &dto.CorporationMembersResult{
		CorporationID: corporationID,
		Members:       make([]dto.CorporationMemberInfo, 0, len(members)),
		Count:         len(members),
	}

	for _, member := range members {
		memberInfo := dto.CorporationMemberInfo{
			CharacterID: member.CharacterID,
		}
		result.Members = append(result.Members, memberInfo)
	}

	slog.InfoContext(ctx, "Successfully retrieved corporation members",
		"corporation_id", corporationID,
		"member_count", result.Count)

	return &dto.CorporationMembersOutput{
		Body: *result,
	}, nil
}

// GetStatus returns the health status of the corporation module
func (s *Service) GetStatus(ctx context.Context) *dto.CorporationStatusResponse {
	// Check database connectivity
	if err := s.repository.CheckHealth(ctx); err != nil {
		return &dto.CorporationStatusResponse{
			Module:  "corporation",
			Status:  "unhealthy",
			Message: "Database connection failed: " + err.Error(),
		}
	}

	return &dto.CorporationStatusResponse{
		Module: "corporation",
		Status: "healthy",
	}
}
