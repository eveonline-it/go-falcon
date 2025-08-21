package routes

import (
	"context"
	"fmt"
	"math"

	"go-falcon/internal/site_settings/dto"
	"go-falcon/internal/site_settings/middleware"
	"go-falcon/internal/site_settings/models"
	"go-falcon/internal/site_settings/services"

	"github.com/danielgtaylor/huma/v2"
)

// Module represents the site settings routes module
type Module struct {
	service    *services.Service
	middleware *middleware.AuthMiddleware
}

// NewModule creates a new routes module
func NewModule(service *services.Service, authMiddleware *middleware.AuthMiddleware) *Module {
	return &Module{
		service:    service,
		middleware: authMiddleware,
	}
}

// RegisterUnifiedRoutes registers all site settings routes with the Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API) {
	// Health check endpoint
	huma.Get(api, "/site-settings/health", m.healthHandler)

	// Public endpoints (no authentication required)
	huma.Get(api, "/site-settings/public", m.getPublicSettingsHandler)

	// Protected endpoints (super admin only)
	huma.Post(api, "/site-settings", m.createSettingHandler)
	huma.Get(api, "/site-settings", m.listSettingsHandler)
	huma.Get(api, "/site-settings/{key}", m.getSettingHandler)
	huma.Put(api, "/site-settings/{key}", m.updateSettingHandler)
	huma.Delete(api, "/site-settings/{key}", m.deleteSettingHandler)
}

// Health check handler
func (m *Module) healthHandler(ctx context.Context, input *struct{}) (*dto.HealthOutput, error) {
	status, err := m.service.GetHealth(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Site settings service is unhealthy", err)
	}

	return &dto.HealthOutput{
		Body: dto.SiteSettingsHealthResponse{
			Health: status,
		},
	}, nil
}

// Get public settings handler (no authentication required)
func (m *Module) getPublicSettingsHandler(ctx context.Context, input *dto.GetPublicSiteSettingsInput) (*dto.ListSiteSettingsOutput, error) {
	settings, total, err := m.service.GetPublicSettings(ctx, input)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve public settings", err)
	}

	// Convert to output format
	settingOutputs := make([]dto.SiteSettingOutput, len(settings))
	for i, setting := range settings {
		settingOutputs[i] = m.convertToOutput(setting)
	}

	totalPages := int(math.Ceil(float64(total) / float64(input.Limit)))

	return &dto.ListSiteSettingsOutput{
		Settings:   settingOutputs,
		Total:      total,
		Page:       input.Page,
		Limit:      input.Limit,
		TotalPages: totalPages,
	}, nil
}

// Create setting handler
func (m *Module) createSettingHandler(ctx context.Context, input *dto.CreateSiteSettingInput) (*dto.CreateSiteSettingOutput, error) {
	// Require super admin authentication
	user, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	setting, err := m.service.CreateSetting(ctx, input, int64(user.CharacterID))
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to create setting", err)
	}

	return &dto.CreateSiteSettingOutput{
		Setting: m.convertToOutput(setting),
		Message: fmt.Sprintf("Site setting '%s' created successfully", setting.Key),
	}, nil
}

// List settings handler
func (m *Module) listSettingsHandler(ctx context.Context, input *dto.ListSiteSettingsInput) (*dto.ListSiteSettingsOutput, error) {
	// Require super admin authentication
	_, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	settings, total, err := m.service.ListSettings(ctx, input)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve settings", err)
	}

	// Convert to output format
	settingOutputs := make([]dto.SiteSettingOutput, len(settings))
	for i, setting := range settings {
		settingOutputs[i] = m.convertToOutput(setting)
	}

	totalPages := int(math.Ceil(float64(total) / float64(input.Limit)))

	return &dto.ListSiteSettingsOutput{
		Settings:   settingOutputs,
		Total:      total,
		Page:       input.Page,
		Limit:      input.Limit,
		TotalPages: totalPages,
	}, nil
}

// Get setting handler
func (m *Module) getSettingHandler(ctx context.Context, input *dto.GetSiteSettingInput) (*dto.GetSiteSettingOutput, error) {
	// Require super admin authentication
	_, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	setting, err := m.service.GetSetting(ctx, input.Key)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve setting", err)
	}

	return &dto.GetSiteSettingOutput{
		Setting: m.convertToOutput(setting),
	}, nil
}

// Update setting handler
func (m *Module) updateSettingHandler(ctx context.Context, input *dto.UpdateSiteSettingInput) (*dto.UpdateSiteSettingOutput, error) {
	// Require super admin authentication
	user, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	setting, err := m.service.UpdateSetting(ctx, input.Key, input, int64(user.CharacterID))
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update setting", err)
	}

	return &dto.UpdateSiteSettingOutput{
		Setting: m.convertToOutput(setting),
		Message: fmt.Sprintf("Site setting '%s' updated successfully", setting.Key),
	}, nil
}

// Delete setting handler
func (m *Module) deleteSettingHandler(ctx context.Context, input *dto.DeleteSiteSettingInput) (*dto.DeleteSiteSettingOutput, error) {
	// Require super admin authentication
	_, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	err = m.service.DeleteSetting(ctx, input.Key)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete setting", err)
	}

	return &dto.DeleteSiteSettingOutput{
		Message: fmt.Sprintf("Site setting '%s' deleted successfully", input.Key),
	}, nil
}

// Helper function to convert model to output DTO
func (m *Module) convertToOutput(setting *models.SiteSetting) dto.SiteSettingOutput {
	return dto.SiteSettingOutput{
		Key:         setting.Key,
		Value:       setting.Value,
		Type:        string(setting.Type),
		Category:    setting.Category,
		Description: setting.Description,
		IsPublic:    setting.IsPublic,
		IsActive:    setting.IsActive,
		CreatedBy:   setting.CreatedBy,
		UpdatedBy:   setting.UpdatedBy,
		CreatedAt:   setting.CreatedAt,
		UpdatedAt:   setting.UpdatedAt,
	}
}