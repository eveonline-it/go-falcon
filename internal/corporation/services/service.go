package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/internal/corporation/dto"
	"go-falcon/internal/corporation/models"
	"go-falcon/pkg/evegateway"

	"go.mongodb.org/mongo-driver/mongo"
)

// Service handles corporation business logic
type Service struct {
	repository *Repository
	eveClient  *evegateway.Client
}

// NewService creates a new corporation service
func NewService(repository *Repository, eveClient *evegateway.Client) *Service {
	return &Service{
		repository: repository,
		eveClient:  eveClient,
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
	return s.convertModelToOutput(corporation), nil
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
	
	if ceoID, ok := esiData["ceo_id"].(int); ok {
		corporation.CEOCharacterID = ceoID
	} else if ceoIDFloat, ok := esiData["ceo_id"].(float64); ok {
		corporation.CEOCharacterID = int(ceoIDFloat)
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
func (s *Service) convertModelToOutput(corporation *models.Corporation) *dto.CorporationInfoOutput {
	corporationInfo := dto.CorporationInfo{
		AllianceID:      corporation.AllianceID,
		CEOCharacterID:  corporation.CEOCharacterID,
		CreatorID:       corporation.CreatorID,
		DateFounded:     corporation.DateFounded,
		Description:     corporation.Description,
		FactionID:       corporation.FactionID,
		HomeStationID:   corporation.HomeStationID,
		MemberCount:     corporation.MemberCount,
		Name:            corporation.Name,
		Shares:          corporation.Shares,
		TaxRate:         corporation.TaxRate,
		Ticker:          corporation.Ticker,
		URL:             corporation.URL,
		WarEligible:     corporation.WarEligible,
	}
	
	return &dto.CorporationInfoOutput{
		Body: corporationInfo,
	}
}