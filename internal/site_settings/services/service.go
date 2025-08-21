package services

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

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
	// Convert bool values to pointers for repository layer
	// For simplicity, we'll pass the values directly. The repository can handle nil checks.
	var isPublic, isActive *bool
	
	// Always set the pointers since we have concrete bool values
	isPublic = &input.IsPublic
	isActive = &input.IsActive
	
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
func (s *Service) GetHealth(ctx context.Context) (string, error) {
	// Test database connectivity by attempting to count documents
	_, err := s.repo.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return "unhealthy", err
	}
	return "healthy", nil
}