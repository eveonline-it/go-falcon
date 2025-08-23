package services

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"go-falcon/internal/site_settings/dto"
	"go-falcon/internal/site_settings/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Service handles business logic for site settings
type Service struct {
	repo *Repository
}

// NewService creates a new service instance
func NewService(db *mongo.Database) *Service {
	repo := NewRepository(db)
	return &Service{
		repo: repo,
	}
}

// InitializeModule initializes the site settings module
func (s *Service) InitializeModule(ctx context.Context) error {
	// Create database indexes
	if err := s.repo.CreateIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Initialize default settings
	if err := s.repo.InitializeDefaults(ctx); err != nil {
		return fmt.Errorf("failed to initialize default settings: %w", err)
	}

	return nil
}

// CreateSetting creates a new site setting
func (s *Service) CreateSetting(ctx context.Context, input *dto.CreateSiteSettingInput, createdBy int64) (*models.SiteSetting, error) {
	// Validate the value matches the specified type
	if err := s.validateValueType(input.Body.Value, input.Body.Type); err != nil {
		return nil, fmt.Errorf("invalid value for type '%s': %w", input.Body.Type, err)
	}
	

	setting := &models.SiteSetting{
		Key:         input.Body.Key,
		Value:       input.Body.Value,
		Type:        models.SettingType(input.Body.Type),
		Category:    input.Body.Category,
		Description: input.Body.Description,
		IsPublic:    input.Body.IsPublic,
		IsActive:    true, // New settings are active by default
		CreatedBy:   &createdBy,
	}

	if err := s.repo.Create(ctx, setting); err != nil {
		return nil, err
	}

	return setting, nil
}

// GetSetting retrieves a setting by key
func (s *Service) GetSetting(ctx context.Context, key string) (*models.SiteSetting, error) {
	return s.repo.GetByKey(ctx, key)
}

// UpdateSetting updates an existing site setting
func (s *Service) UpdateSetting(ctx context.Context, key string, input *dto.UpdateSiteSettingInput, updatedBy int64) (*models.SiteSetting, error) {
	updates := bson.M{}

	// Update value and validate type if both are provided
	if input.Body.Value != nil {
		settingType := ""
		if input.Body.Type != nil {
			settingType = *input.Body.Type
			updates["type"] = models.SettingType(*input.Body.Type)
		} else {
			// Get current setting to determine type
			current, err := s.repo.GetByKey(ctx, key)
			if err != nil {
				return nil, err
			}
			settingType = string(current.Type)
		}

		// Validate value against type
		if err := s.validateValueType(input.Body.Value, settingType); err != nil {
			return nil, fmt.Errorf("invalid value for type '%s': %w", settingType, err)
		}

		updates["value"] = input.Body.Value
	}

	// Update other fields if provided
	if input.Body.Type != nil && input.Body.Value == nil {
		updates["type"] = models.SettingType(*input.Body.Type)
	}
	if input.Body.Category != nil {
		updates["category"] = *input.Body.Category
	}
	if input.Body.Description != nil {
		updates["description"] = *input.Body.Description
	}
	if input.Body.IsPublic != nil {
		updates["is_public"] = *input.Body.IsPublic
	}
	if input.Body.IsActive != nil {
		updates["is_active"] = *input.Body.IsActive
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no valid updates provided")
	}

	return s.repo.Update(ctx, key, updates, updatedBy)
}

// DeleteSetting deletes a site setting
func (s *Service) DeleteSetting(ctx context.Context, key string) error {
	return s.repo.Delete(ctx, key)
}

// ListSettings returns a paginated list of settings with filters
func (s *Service) ListSettings(ctx context.Context, input *dto.ListSiteSettingsInput) ([]*models.SiteSetting, int, error) {
	// Convert string filters to boolean pointers
	var isPublic, isActive *bool
	
	// Parse IsPublicFilter
	if input.IsPublicFilter != "" {
		if input.IsPublicFilter == "true" {
			val := true
			isPublic = &val
		} else if input.IsPublicFilter == "false" {
			val := false
			isPublic = &val
		}
	}
	
	// Parse IsActiveFilter
	if input.IsActiveFilter != "" {
		if input.IsActiveFilter == "true" {
			val := true
			isActive = &val
		} else if input.IsActiveFilter == "false" {
			val := false
			isActive = &val
		}
	}
	
	return s.repo.ListSettings(
		ctx,
		input.Category,
		isPublic,
		isActive,
		input.Page,
		input.Limit,
	)
}

// GetPublicSettings returns public settings (no authentication required)
func (s *Service) GetPublicSettings(ctx context.Context, input *dto.GetPublicSiteSettingsInput) ([]*models.SiteSetting, int, error) {
	return s.repo.GetPublicSettings(ctx, input.Category, input.Page, input.Limit)
}

// SettingExists checks if a setting exists
func (s *Service) SettingExists(ctx context.Context, key string) (bool, error) {
	return s.repo.SettingExists(ctx, key)
}

// validateValueType validates that a value matches the expected type
func (s *Service) validateValueType(value interface{}, settingType string) error {
	if value == nil {
		return fmt.Errorf("value cannot be nil")
	}

	switch models.SettingType(settingType) {
	case models.SettingTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case models.SettingTypeNumber:
		// Accept both int and float values
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// Valid integer
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			// Valid unsigned integer
		case reflect.Float32, reflect.Float64:
			// Valid float
		default:
			// Try to parse as string if it's a string
			if str, ok := value.(string); ok {
				if _, err := strconv.ParseFloat(str, 64); err != nil {
					return fmt.Errorf("expected number, got string that cannot be converted to number: %s", str)
				}
			} else {
				return fmt.Errorf("expected number, got %T", value)
			}
		}
	case models.SettingTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case models.SettingTypeObject:
		// Accept maps, slices, or any complex type
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Map, reflect.Slice, reflect.Struct:
			// Valid object types
		default:
			return fmt.Errorf("expected object (map, slice, or struct), got %T", value)
		}
	default:
		return fmt.Errorf("unknown setting type: %s", settingType)
	}

	return nil
}

// GetHealth returns the health status of the site settings module
func (s *Service) GetHealth(ctx context.Context) (*dto.SiteSettingsHealthResponse, error) {
	// Test database connectivity by attempting to count documents
	totalCount, err := s.repo.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return &dto.SiteSettingsHealthResponse{
			Health:      "unhealthy",
			TotalCount:  0,
			PublicCount: 0,
		}, err
	}
	
	// Count public settings
	publicCount, err := s.repo.collection.CountDocuments(ctx, bson.M{"is_public": true})
	if err != nil {
		return &dto.SiteSettingsHealthResponse{
			Health:      "unhealthy",
			TotalCount:  int(totalCount),
			PublicCount: 0,
		}, err
	}
	
	return &dto.SiteSettingsHealthResponse{
		Health:      "healthy",
		TotalCount:  int(totalCount),
		PublicCount: int(publicCount),
	}, nil
}

// Corporation Management Methods

// AddManagedCorporation adds a new managed corporation
func (s *Service) AddManagedCorporation(ctx context.Context, input *dto.AddCorporationInput, addedBy int64) (*dto.ManagedCorporation, error) {
	corporations, err := s.getManagedCorporationsData(ctx)
	if err != nil {
		return nil, err
	}

	// Check if corporation already exists
	for _, corp := range corporations {
		if corp.CorporationID == input.Body.CorporationID {
			return nil, fmt.Errorf("corporation with ID %d is already managed", input.Body.CorporationID)
		}
	}

	// Default enabled to true if not specified
	enabled := true
	if input.Body.Enabled != nil {
		enabled = *input.Body.Enabled
	}

	// Handle position assignment
	position := 0
	if input.Body.Position != nil {
		position = *input.Body.Position
	} else {
		// Auto-assign next position
		var err error
		position, err = s.getNextPosition(ctx, corporations)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	newCorp := dto.ManagedCorporation{
		CorporationID: input.Body.CorporationID,
		Name:          input.Body.Name,
		Ticker:        input.Body.Ticker,
		Enabled:       enabled,
		Position:      position,
		AddedAt:       now,
		AddedBy:       &addedBy,
		UpdatedAt:     now,
		UpdatedBy:     &addedBy,
	}

	corporations = append(corporations, newCorp)

	// Update the setting
	if err := s.updateManagedCorporationsSetting(ctx, corporations, addedBy); err != nil {
		return nil, err
	}

	return &newCorp, nil
}

// UpdateCorporationStatus enables or disables a managed corporation
func (s *Service) UpdateCorporationStatus(ctx context.Context, corporationID int64, enabled bool, updatedBy int64) (*dto.ManagedCorporation, error) {
	corporations, err := s.getManagedCorporationsData(ctx)
	if err != nil {
		return nil, err
	}

	// Find and update the corporation
	for i, corp := range corporations {
		if corp.CorporationID == corporationID {
			corporations[i].Enabled = enabled
			corporations[i].UpdatedAt = time.Now()
			corporations[i].UpdatedBy = &updatedBy

			// Update the setting
			if err := s.updateManagedCorporationsSetting(ctx, corporations, updatedBy); err != nil {
				return nil, err
			}

			return &corporations[i], nil
		}
	}

	return nil, fmt.Errorf("corporation with ID %d not found", corporationID)
}

// RemoveManagedCorporation removes a managed corporation
func (s *Service) RemoveManagedCorporation(ctx context.Context, corporationID int64, removedBy int64) error {
	corporations, err := s.getManagedCorporationsData(ctx)
	if err != nil {
		return err
	}

	// Find and remove the corporation
	found := false
	newCorporations := make([]dto.ManagedCorporation, 0, len(corporations))
	for _, corp := range corporations {
		if corp.CorporationID != corporationID {
			newCorporations = append(newCorporations, corp)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("corporation with ID %d not found", corporationID)
	}

	// Update the setting
	return s.updateManagedCorporationsSetting(ctx, newCorporations, removedBy)
}

// GetManagedCorporations returns managed corporations with optional filtering
func (s *Service) GetManagedCorporations(ctx context.Context, enabledFilter string, page, limit int) ([]dto.ManagedCorporation, int, error) {
	corporations, err := s.getManagedCorporationsData(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Sort by position
	s.sortCorporationsByPosition(corporations)

	// Apply enabled filter
	if enabledFilter != "" {
		filtered := make([]dto.ManagedCorporation, 0, len(corporations))
		var filterEnabled bool
		if enabledFilter == "true" {
			filterEnabled = true
		} else if enabledFilter == "false" {
			filterEnabled = false
		} else {
			return nil, 0, fmt.Errorf("invalid enabled filter value: %s", enabledFilter)
		}

		for _, corp := range corporations {
			if corp.Enabled == filterEnabled {
				filtered = append(filtered, corp)
			}
		}
		corporations = filtered
	}

	// Apply pagination
	total := len(corporations)
	start := (page - 1) * limit
	end := start + limit

	if start >= len(corporations) {
		return []dto.ManagedCorporation{}, total, nil
	}

	if end > len(corporations) {
		end = len(corporations)
	}

	return corporations[start:end], total, nil
}

// GetManagedCorporation returns a specific managed corporation
func (s *Service) GetManagedCorporation(ctx context.Context, corporationID int64) (*dto.ManagedCorporation, error) {
	corporations, err := s.getManagedCorporationsData(ctx)
	if err != nil {
		return nil, err
	}

	for _, corp := range corporations {
		if corp.CorporationID == corporationID {
			return &corp, nil
		}
	}

	return nil, fmt.Errorf("corporation with ID %d not found", corporationID)
}

// BulkUpdateCorporations performs bulk update of managed corporations
func (s *Service) BulkUpdateCorporations(ctx context.Context, input *dto.BulkUpdateCorporationsInput, updatedBy int64) ([]dto.ManagedCorporation, int, int, error) {
	corporations, err := s.getManagedCorporationsData(ctx)
	if err != nil {
		return nil, 0, 0, err
	}

	// Create a map of existing corporations for quick lookup
	existingCorps := make(map[int64]*dto.ManagedCorporation)
	for i := range corporations {
		existingCorps[corporations[i].CorporationID] = &corporations[i]
	}

	updated := 0
	added := 0
	now := time.Now()

	// Process input corporations
	for _, inputCorp := range input.Body.Corporations {
		if existing, exists := existingCorps[inputCorp.CorporationID]; exists {
			// Update existing corporation
			existing.Name = inputCorp.Name
			existing.Enabled = inputCorp.Enabled
			if inputCorp.Position != nil {
				existing.Position = *inputCorp.Position
			}
			existing.UpdatedAt = now
			existing.UpdatedBy = &updatedBy
			updated++
		} else {
			// Handle position for new corporation
			position := 0
			if inputCorp.Position != nil {
				position = *inputCorp.Position
			} else {
				// Auto-assign next position
				var err error
				position, err = s.getNextPosition(ctx, corporations)
				if err != nil {
					return nil, 0, 0, err
				}
			}

			// Add new corporation
			newCorp := dto.ManagedCorporation{
				CorporationID: inputCorp.CorporationID,
				Name:          inputCorp.Name,
				Enabled:       inputCorp.Enabled,
				Position:      position,
				AddedAt:       now,
				AddedBy:       &updatedBy,
				UpdatedAt:     now,
				UpdatedBy:     &updatedBy,
			}
			corporations = append(corporations, newCorp)
			added++
		}
	}

	// Update the setting
	if err := s.updateManagedCorporationsSetting(ctx, corporations, updatedBy); err != nil {
		return nil, 0, 0, err
	}

	return corporations, updated, added, nil
}

// IsCorporationEnabled checks if a corporation is enabled
func (s *Service) IsCorporationEnabled(ctx context.Context, corporationID int64) (bool, error) {
	corp, err := s.GetManagedCorporation(ctx, corporationID)
	if err != nil {
		return false, err
	}
	return corp.Enabled, nil
}

// Helper methods

// getManagedCorporationsData retrieves the managed corporations from the setting
func (s *Service) getManagedCorporationsData(ctx context.Context) ([]dto.ManagedCorporation, error) {
	setting, err := s.repo.GetByKey(ctx, "managed_corporations")
	if err != nil {
		// If setting doesn't exist, return empty array
		if err == mongo.ErrNoDocuments {
			return []dto.ManagedCorporation{}, nil
		}
		// Handle the custom "not found" error from repository
		if err.Error() == "setting with key 'managed_corporations' not found" {
			return []dto.ManagedCorporation{}, nil
		}
		return nil, err
	}

	// Parse the setting value using proper BSON unmarshaling
	var managedCorpsValue models.ManagedCorporationsValue
	
	// Marshal the setting.Value to BSON and then unmarshal to our struct
	valueBytes, err := bson.Marshal(setting.Value)
	if err != nil {
		return []dto.ManagedCorporation{}, nil
	}
	
	if err := bson.Unmarshal(valueBytes, &managedCorpsValue); err != nil {
		return []dto.ManagedCorporation{}, nil
	}

	// Convert models to DTOs
	corporations := make([]dto.ManagedCorporation, len(managedCorpsValue.Corporations))
	for i, corp := range managedCorpsValue.Corporations {
		corporations[i] = dto.ManagedCorporation{
			CorporationID: corp.CorporationID,
			Name:          corp.Name,
			Ticker:        corp.Ticker,
			Enabled:       corp.Enabled,
			Position:      corp.Position,
			AddedAt:       corp.AddedAt,
			AddedBy:       corp.AddedBy,
			UpdatedAt:     corp.UpdatedAt,
			UpdatedBy:     corp.UpdatedBy,
		}
	}

	return corporations, nil
}

// updateManagedCorporationsSetting updates the managed_corporations setting
func (s *Service) updateManagedCorporationsSetting(ctx context.Context, corporations []dto.ManagedCorporation, updatedBy int64) error {
	// Convert DTOs to models
	modelCorps := make([]models.ManagedCorporation, len(corporations))
	for i, corp := range corporations {
		modelCorps[i] = models.ManagedCorporation{
			CorporationID: corp.CorporationID,
			Name:          corp.Name,
			Ticker:        corp.Ticker,
			Enabled:       corp.Enabled,
			Position:      corp.Position,
			AddedAt:       corp.AddedAt,
			AddedBy:       corp.AddedBy,
			UpdatedAt:     corp.UpdatedAt,
			UpdatedBy:     corp.UpdatedBy,
		}
	}
	
	settingValue := models.ManagedCorporationsValue{
		Corporations: modelCorps,
	}

	// Check if setting exists
	exists, err := s.repo.SettingExists(ctx, "managed_corporations")
	if err != nil {
		return err
	}

	if !exists {
		// Create new setting
		setting := &models.SiteSetting{
			Key:         "managed_corporations",
			Value:       settingValue,
			Type:        models.SettingTypeObject,
			Category:    "eve",
			Description: "Managed corporations with enable/disable status",
			IsPublic:    false,
			IsActive:    true,
			CreatedBy:   &updatedBy,
		}
		return s.repo.Create(ctx, setting)
	} else {
		// Update existing setting
		updates := bson.M{
			"value": settingValue,
		}
		_, err := s.repo.Update(ctx, "managed_corporations", updates, updatedBy)
		return err
	}
}

// ReorderCorporations reorders managed corporations based on provided positions
func (s *Service) ReorderCorporations(ctx context.Context, input *dto.ReorderCorporationsInput, updatedBy int64) ([]dto.ManagedCorporation, error) {
	corporations, err := s.getManagedCorporationsData(ctx)
	if err != nil {
		return nil, err
	}

	// Create map of existing corporations
	corpMap := make(map[int64]*dto.ManagedCorporation)
	for i := range corporations {
		corpMap[corporations[i].CorporationID] = &corporations[i]
	}

	// Validate all corporation IDs exist and positions are valid
	positionMap := make(map[int]bool)
	for _, order := range input.Body.CorporationOrders {
		if _, exists := corpMap[order.CorporationID]; !exists {
			return nil, fmt.Errorf("corporation with ID %d not found", order.CorporationID)
		}
		
		if order.Position < 1 {
			return nil, fmt.Errorf("position must be greater than 0")
		}
		
		if positionMap[order.Position] {
			return nil, fmt.Errorf("duplicate position %d", order.Position)
		}
		positionMap[order.Position] = true
	}

	// Update positions
	now := time.Now()
	for _, order := range input.Body.CorporationOrders {
		corp := corpMap[order.CorporationID]
		corp.Position = order.Position
		corp.UpdatedAt = now
		corp.UpdatedBy = &updatedBy
	}

	// Update the setting
	if err := s.updateManagedCorporationsSetting(ctx, corporations, updatedBy); err != nil {
		return nil, err
	}

	// Return sorted corporations
	s.sortCorporationsByPosition(corporations)
	return corporations, nil
}

// getNextPosition calculates the next available position
func (s *Service) getNextPosition(ctx context.Context, corporations []dto.ManagedCorporation) (int, error) {
	maxPosition := 0
	for _, corp := range corporations {
		if corp.Position > maxPosition {
			maxPosition = corp.Position
		}
	}
	return maxPosition + 1, nil
}

// sortCorporationsByPosition sorts corporations by position in ascending order
func (s *Service) sortCorporationsByPosition(corporations []dto.ManagedCorporation) {
	// Simple bubble sort by position
	n := len(corporations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if corporations[j].Position > corporations[j+1].Position {
				corporations[j], corporations[j+1] = corporations[j+1], corporations[j]
			}
		}
	}
}

// Alliance Management Methods

// AddManagedAlliance adds a new managed alliance
func (s *Service) AddManagedAlliance(ctx context.Context, input *dto.AddAllianceInput, addedBy int64) (*dto.ManagedAlliance, error) {
	alliances, err := s.getManagedAlliancesData(ctx)
	if err != nil {
		return nil, err
	}

	// Check if alliance already exists
	for _, alliance := range alliances {
		if alliance.AllianceID == input.Body.AllianceID {
			return nil, fmt.Errorf("alliance with ID %d is already managed", input.Body.AllianceID)
		}
	}

	// Default enabled to true if not specified
	enabled := true
	if input.Body.Enabled != nil {
		enabled = *input.Body.Enabled
	}

	// Handle position assignment
	position := 0
	if input.Body.Position != nil {
		position = *input.Body.Position
	} else {
		// Auto-assign next position
		var err error
		position, err = s.getNextAlliancePosition(ctx, alliances)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	newAlliance := dto.ManagedAlliance{
		AllianceID: input.Body.AllianceID,
		Name:       input.Body.Name,
		Ticker:     input.Body.Ticker,
		Enabled:    enabled,
		Position:   position,
		AddedAt:    now,
		AddedBy:    &addedBy,
		UpdatedAt:  now,
		UpdatedBy:  &addedBy,
	}

	alliances = append(alliances, newAlliance)

	// Update the setting
	if err := s.updateManagedAlliancesSetting(ctx, alliances, addedBy); err != nil {
		return nil, err
	}

	return &newAlliance, nil
}

// UpdateAllianceStatus enables or disables a managed alliance
func (s *Service) UpdateAllianceStatus(ctx context.Context, allianceID int64, enabled bool, updatedBy int64) (*dto.ManagedAlliance, error) {
	alliances, err := s.getManagedAlliancesData(ctx)
	if err != nil {
		return nil, err
	}

	// Find and update the alliance
	for i, alliance := range alliances {
		if alliance.AllianceID == allianceID {
			alliances[i].Enabled = enabled
			alliances[i].UpdatedAt = time.Now()
			alliances[i].UpdatedBy = &updatedBy

			// Update the setting
			if err := s.updateManagedAlliancesSetting(ctx, alliances, updatedBy); err != nil {
				return nil, err
			}

			return &alliances[i], nil
		}
	}

	return nil, fmt.Errorf("alliance with ID %d not found", allianceID)
}

// RemoveManagedAlliance removes a managed alliance
func (s *Service) RemoveManagedAlliance(ctx context.Context, allianceID int64, removedBy int64) error {
	alliances, err := s.getManagedAlliancesData(ctx)
	if err != nil {
		return err
	}

	// Find and remove the alliance
	found := false
	newAlliances := make([]dto.ManagedAlliance, 0, len(alliances))
	for _, alliance := range alliances {
		if alliance.AllianceID != allianceID {
			newAlliances = append(newAlliances, alliance)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("alliance with ID %d not found", allianceID)
	}

	// Update the setting
	return s.updateManagedAlliancesSetting(ctx, newAlliances, removedBy)
}

// GetManagedAlliances returns managed alliances with optional filtering
func (s *Service) GetManagedAlliances(ctx context.Context, enabledFilter string, page, limit int) ([]dto.ManagedAlliance, int, error) {
	alliances, err := s.getManagedAlliancesData(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Sort by position
	s.sortAlliancesByPosition(alliances)

	// Apply enabled filter
	if enabledFilter != "" {
		filtered := make([]dto.ManagedAlliance, 0, len(alliances))
		var filterEnabled bool
		if enabledFilter == "true" {
			filterEnabled = true
		} else if enabledFilter == "false" {
			filterEnabled = false
		} else {
			return nil, 0, fmt.Errorf("invalid enabled filter value: %s", enabledFilter)
		}

		for _, alliance := range alliances {
			if alliance.Enabled == filterEnabled {
				filtered = append(filtered, alliance)
			}
		}
		alliances = filtered
	}

	// Apply pagination
	total := len(alliances)
	start := (page - 1) * limit
	end := start + limit

	if start >= len(alliances) {
		return []dto.ManagedAlliance{}, total, nil
	}

	if end > len(alliances) {
		end = len(alliances)
	}

	return alliances[start:end], total, nil
}

// GetManagedAlliance returns a specific managed alliance
func (s *Service) GetManagedAlliance(ctx context.Context, allianceID int64) (*dto.ManagedAlliance, error) {
	alliances, err := s.getManagedAlliancesData(ctx)
	if err != nil {
		return nil, err
	}

	for _, alliance := range alliances {
		if alliance.AllianceID == allianceID {
			return &alliance, nil
		}
	}

	return nil, fmt.Errorf("alliance with ID %d not found", allianceID)
}

// BulkUpdateAlliances performs bulk update of managed alliances
func (s *Service) BulkUpdateAlliances(ctx context.Context, input *dto.BulkUpdateAlliancesInput, updatedBy int64) ([]dto.ManagedAlliance, int, int, error) {
	alliances, err := s.getManagedAlliancesData(ctx)
	if err != nil {
		return nil, 0, 0, err
	}

	// Create a map of existing alliances for quick lookup
	existingAlliances := make(map[int64]*dto.ManagedAlliance)
	for i := range alliances {
		existingAlliances[alliances[i].AllianceID] = &alliances[i]
	}

	updated := 0
	added := 0
	now := time.Now()

	// Process input alliances
	for _, inputAlliance := range input.Body.Alliances {
		if existing, exists := existingAlliances[inputAlliance.AllianceID]; exists {
			// Update existing alliance
			existing.Name = inputAlliance.Name
			existing.Enabled = inputAlliance.Enabled
			if inputAlliance.Position != nil {
				existing.Position = *inputAlliance.Position
			}
			existing.UpdatedAt = now
			existing.UpdatedBy = &updatedBy
			updated++
		} else {
			// Handle position for new alliance
			position := 0
			if inputAlliance.Position != nil {
				position = *inputAlliance.Position
			} else {
				// Auto-assign next position
				var err error
				position, err = s.getNextAlliancePosition(ctx, alliances)
				if err != nil {
					return nil, 0, 0, err
				}
			}

			// Add new alliance
			newAlliance := dto.ManagedAlliance{
				AllianceID: inputAlliance.AllianceID,
				Name:       inputAlliance.Name,
				Enabled:    inputAlliance.Enabled,
				Position:   position,
				AddedAt:    now,
				AddedBy:    &updatedBy,
				UpdatedAt:  now,
				UpdatedBy:  &updatedBy,
			}
			alliances = append(alliances, newAlliance)
			added++
		}
	}

	// Update the setting
	if err := s.updateManagedAlliancesSetting(ctx, alliances, updatedBy); err != nil {
		return nil, 0, 0, err
	}

	return alliances, updated, added, nil
}

// ReorderAlliances reorders managed alliances based on provided positions
func (s *Service) ReorderAlliances(ctx context.Context, input *dto.ReorderAlliancesInput, updatedBy int64) ([]dto.ManagedAlliance, error) {
	alliances, err := s.getManagedAlliancesData(ctx)
	if err != nil {
		return nil, err
	}

	// Create map of existing alliances
	allianceMap := make(map[int64]*dto.ManagedAlliance)
	for i := range alliances {
		allianceMap[alliances[i].AllianceID] = &alliances[i]
	}

	// Validate all alliance IDs exist and positions are valid
	positionMap := make(map[int]bool)
	for _, order := range input.Body.AllianceOrders {
		if _, exists := allianceMap[order.AllianceID]; !exists {
			return nil, fmt.Errorf("alliance with ID %d not found", order.AllianceID)
		}
		
		if order.Position < 1 {
			return nil, fmt.Errorf("position must be greater than 0")
		}
		
		if positionMap[order.Position] {
			return nil, fmt.Errorf("duplicate position %d", order.Position)
		}
		positionMap[order.Position] = true
	}

	// Update positions
	now := time.Now()
	for _, order := range input.Body.AllianceOrders {
		alliance := allianceMap[order.AllianceID]
		alliance.Position = order.Position
		alliance.UpdatedAt = now
		alliance.UpdatedBy = &updatedBy
	}

	// Update the setting
	if err := s.updateManagedAlliancesSetting(ctx, alliances, updatedBy); err != nil {
		return nil, err
	}

	// Return sorted alliances
	s.sortAlliancesByPosition(alliances)
	return alliances, nil
}

// IsAllianceEnabled checks if an alliance is enabled
func (s *Service) IsAllianceEnabled(ctx context.Context, allianceID int64) (bool, error) {
	alliance, err := s.GetManagedAlliance(ctx, allianceID)
	if err != nil {
		return false, err
	}
	return alliance.Enabled, nil
}

// Helper methods for alliance management

// getManagedAlliancesData retrieves the managed alliances from the setting
func (s *Service) getManagedAlliancesData(ctx context.Context) ([]dto.ManagedAlliance, error) {
	setting, err := s.repo.GetByKey(ctx, "managed_alliances")
	if err != nil {
		// If setting doesn't exist, return empty array
		if err == mongo.ErrNoDocuments {
			return []dto.ManagedAlliance{}, nil
		}
		// Handle the custom "not found" error from repository
		if err.Error() == "setting with key 'managed_alliances' not found" {
			return []dto.ManagedAlliance{}, nil
		}
		return nil, err
	}

	// Parse the setting value using proper BSON unmarshaling
	var managedAlliancesValue models.ManagedAlliancesValue
	
	// Marshal the setting.Value to BSON and then unmarshal to our struct
	valueBytes, err := bson.Marshal(setting.Value)
	if err != nil {
		return []dto.ManagedAlliance{}, nil
	}
	
	if err := bson.Unmarshal(valueBytes, &managedAlliancesValue); err != nil {
		return []dto.ManagedAlliance{}, nil
	}

	// Convert models to DTOs
	alliances := make([]dto.ManagedAlliance, len(managedAlliancesValue.Alliances))
	for i, alliance := range managedAlliancesValue.Alliances {
		alliances[i] = dto.ManagedAlliance{
			AllianceID: alliance.AllianceID,
			Name:       alliance.Name,
			Ticker:     alliance.Ticker,
			Enabled:    alliance.Enabled,
			Position:   alliance.Position,
			AddedAt:    alliance.AddedAt,
			AddedBy:    alliance.AddedBy,
			UpdatedAt:  alliance.UpdatedAt,
			UpdatedBy:  alliance.UpdatedBy,
		}
	}

	return alliances, nil
}

// updateManagedAlliancesSetting updates the managed_alliances setting
func (s *Service) updateManagedAlliancesSetting(ctx context.Context, alliances []dto.ManagedAlliance, updatedBy int64) error {
	// Convert DTOs to models
	modelAlliances := make([]models.ManagedAlliance, len(alliances))
	for i, alliance := range alliances {
		modelAlliances[i] = models.ManagedAlliance{
			AllianceID: alliance.AllianceID,
			Name:       alliance.Name,
			Ticker:     alliance.Ticker,
			Enabled:    alliance.Enabled,
			Position:   alliance.Position,
			AddedAt:    alliance.AddedAt,
			AddedBy:    alliance.AddedBy,
			UpdatedAt:  alliance.UpdatedAt,
			UpdatedBy:  alliance.UpdatedBy,
		}
	}
	
	settingValue := models.ManagedAlliancesValue{
		Alliances: modelAlliances,
	}

	// Check if setting exists
	exists, err := s.repo.SettingExists(ctx, "managed_alliances")
	if err != nil {
		return err
	}

	if !exists {
		// Create new setting
		setting := &models.SiteSetting{
			Key:         "managed_alliances",
			Value:       settingValue,
			Type:        models.SettingTypeObject,
			Category:    "eve",
			Description: "Managed alliances with enable/disable status",
			IsPublic:    false,
			IsActive:    true,
			CreatedBy:   &updatedBy,
		}
		return s.repo.Create(ctx, setting)
	} else {
		// Update existing setting
		updates := bson.M{
			"value": settingValue,
		}
		_, err := s.repo.Update(ctx, "managed_alliances", updates, updatedBy)
		return err
	}
}

// GetStatus returns the health status of the site settings module
func (s *Service) GetStatus(ctx context.Context) *dto.SiteSettingsStatusResponse {
	// Check database connectivity
	if err := s.repo.CheckHealth(ctx); err != nil {
		return &dto.SiteSettingsStatusResponse{
			Module:  "site_settings",
			Status:  "unhealthy",
			Message: "Database connection failed: " + err.Error(),
		}
	}

	// Check if default settings are initialized
	settings, err := s.repo.List(ctx, "", nil, nil, nil, 1, 1)
	if err != nil || len(settings) == 0 {
		return &dto.SiteSettingsStatusResponse{
			Module:  "site_settings",
			Status:  "unhealthy",
			Message: "Default settings not initialized",
		}
	}

	return &dto.SiteSettingsStatusResponse{
		Module: "site_settings",
		Status: "healthy",
	}
}

// getNextAlliancePosition calculates the next available position for alliances
func (s *Service) getNextAlliancePosition(ctx context.Context, alliances []dto.ManagedAlliance) (int, error) {
	maxPosition := 0
	for _, alliance := range alliances {
		if alliance.Position > maxPosition {
			maxPosition = alliance.Position
		}
	}
	return maxPosition + 1, nil
}

// sortAlliancesByPosition sorts alliances by position in ascending order
func (s *Service) sortAlliancesByPosition(alliances []dto.ManagedAlliance) {
	// Simple bubble sort by position
	n := len(alliances)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if alliances[j].Position > alliances[j+1].Position {
				alliances[j], alliances[j+1] = alliances[j+1], alliances[j]
			}
		}
	}
}

// GetEnabledCorporations returns only the enabled corporations for groups module integration
func (s *Service) GetEnabledCorporations(ctx context.Context) ([]models.ManagedCorporation, error) {
	setting, err := s.repo.GetByKey(ctx, "managed_corporations")
	if err != nil {
		return nil, fmt.Errorf("failed to get managed corporations: %w", err)
	}
	
	if setting == nil {
		return []models.ManagedCorporation{}, nil
	}
	
	var managedCorpsValue models.ManagedCorporationsValue
	
	// Marshal the setting.Value to BSON and then unmarshal to our struct
	valueBytes, err := bson.Marshal(setting.Value)
	if err != nil {
		return []models.ManagedCorporation{}, nil
	}
	
	if err := bson.Unmarshal(valueBytes, &managedCorpsValue); err != nil {
		return []models.ManagedCorporation{}, nil
	}
	
	// Return only enabled corporations
	var enabled []models.ManagedCorporation
	for _, corp := range managedCorpsValue.Corporations {
		if corp.Enabled {
			enabled = append(enabled, corp)
		}
	}
	
	return enabled, nil
}

// GetEnabledAlliances returns only the enabled alliances for groups module integration
func (s *Service) GetEnabledAlliances(ctx context.Context) ([]models.ManagedAlliance, error) {
	setting, err := s.repo.GetByKey(ctx, "managed_alliances")
	if err != nil {
		return nil, fmt.Errorf("failed to get managed alliances: %w", err)
	}
	
	if setting == nil {
		return []models.ManagedAlliance{}, nil
	}
	
	var managedAlliancesValue models.ManagedAlliancesValue
	
	// Marshal the setting.Value to BSON and then unmarshal to our struct
	valueBytes, err := bson.Marshal(setting.Value)
	if err != nil {
		return []models.ManagedAlliance{}, nil
	}
	
	if err := bson.Unmarshal(valueBytes, &managedAlliancesValue); err != nil {
		return []models.ManagedAlliance{}, nil
	}
	
	// Return only enabled alliances
	var enabled []models.ManagedAlliance
	for _, alliance := range managedAlliancesValue.Alliances {
		if alliance.Enabled {
			enabled = append(enabled, alliance)
		}
	}
	
	return enabled, nil
}


