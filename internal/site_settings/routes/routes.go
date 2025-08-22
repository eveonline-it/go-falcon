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
	huma.Register(api, huma.Operation{
		OperationID: "site-settings-health-check",
		Method:      "GET",
		Path:        "/site-settings/health",
		Summary:     "Site Settings module health check",
		Description: "Check if the site settings module is healthy",
		Tags:        []string{"Site Settings"},
	}, m.healthHandler)

	// Public endpoints (no authentication required)
	huma.Register(api, huma.Operation{
		OperationID: "site-settings-get-public",
		Method:      "GET",
		Path:        "/site-settings/public",
		Summary:     "Get public site settings",
		Description: "Retrieve public site settings that can be accessed without authentication",
		Tags:        []string{"Site Settings / Public"},
	}, m.getPublicSettingsHandler)


	// Protected endpoints (super admin only)
	huma.Register(api, huma.Operation{
		OperationID: "site-settings-create",
		Method:      "POST",
		Path:        "/site-settings",
		Summary:     "Create site setting",
		Description: "Create a new site configuration setting (super admin only)",
		Tags:        []string{"Site Settings / Management"},
	}, m.createSettingHandler)

	huma.Register(api, huma.Operation{
		OperationID: "site-settings-list",
		Method:      "GET",
		Path:        "/site-settings",
		Summary:     "List site settings",
		Description: "List all site settings with filtering and pagination (super admin only)",
		Tags:        []string{"Site Settings / Management"},
	}, m.listSettingsHandler)

	huma.Register(api, huma.Operation{
		OperationID: "site-settings-get",
		Method:      "GET",
		Path:        "/site-settings/{key}",
		Summary:     "Get site setting",
		Description: "Get a specific site setting by key (super admin only)",
		Tags:        []string{"Site Settings / Management"},
	}, m.getSettingHandler)

	huma.Register(api, huma.Operation{
		OperationID: "site-settings-update",
		Method:      "PUT",
		Path:        "/site-settings/{key}",
		Summary:     "Update site setting",
		Description: "Update an existing site setting (super admin only)",
		Tags:        []string{"Site Settings / Management"},
	}, m.updateSettingHandler)

	huma.Register(api, huma.Operation{
		OperationID: "site-settings-delete",
		Method:      "DELETE",
		Path:        "/site-settings/{key}",
		Summary:     "Delete site setting",
		Description: "Delete a site setting (super admin only)",
		Tags:        []string{"Site Settings / Management"},
	}, m.deleteSettingHandler)

	// Corporation Management endpoints (super admin only)
	huma.Register(api, huma.Operation{
		OperationID: "site-settings-add-corporation",
		Method:      "POST",
		Path:        "/site-settings/corporations",
		Summary:     "Add managed corporation",
		Description: "Add a new managed corporation with enable/disable status (super admin only)",
		Tags:        []string{"Site Settings / Corporations"},
	}, m.addCorporationHandler)

	huma.Register(api, huma.Operation{
		OperationID: "site-settings-list-corporations",
		Method:      "GET",
		Path:        "/site-settings/corporations",
		Summary:     "List managed corporations",
		Description: "List all managed corporations with optional filtering by enabled status (super admin only)",
		Tags:        []string{"Site Settings / Corporations"},
	}, m.listCorporationsHandler)

	huma.Register(api, huma.Operation{
		OperationID: "site-settings-get-corporation",
		Method:      "GET",
		Path:        "/site-settings/corporations/{corp_id}",
		Summary:     "Get managed corporation",
		Description: "Get a specific managed corporation by ID (super admin only)",
		Tags:        []string{"Site Settings / Corporations"},
	}, m.getCorporationHandler)

	huma.Register(api, huma.Operation{
		OperationID: "site-settings-update-corporation-status",
		Method:      "PUT",
		Path:        "/site-settings/corporations/{corp_id}/status",
		Summary:     "Update corporation status",
		Description: "Enable or disable a managed corporation (super admin only)",
		Tags:        []string{"Site Settings / Corporations"},
	}, m.updateCorporationStatusHandler)

	huma.Register(api, huma.Operation{
		OperationID: "site-settings-remove-corporation",
		Method:      "DELETE",
		Path:        "/site-settings/corporations/{corp_id}",
		Summary:     "Remove managed corporation",
		Description: "Remove a managed corporation completely (super admin only)",
		Tags:        []string{"Site Settings / Corporations"},
	}, m.removeCorporationHandler)

	huma.Register(api, huma.Operation{
		OperationID: "site-settings-bulk-update-corporations",
		Method:      "PUT",
		Path:        "/site-settings/corporations",
		Summary:     "Bulk update corporations",
		Description: "Bulk update or add multiple managed corporations (super admin only)",
		Tags:        []string{"Site Settings / Corporations"},
	}, m.bulkUpdateCorporationsHandler)
}

// Health check handler
func (m *Module) healthHandler(ctx context.Context, input *struct{}) (*dto.HealthOutput, error) {
	healthResponse, err := m.service.GetHealth(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Site settings service is unhealthy", err)
	}

	return &dto.HealthOutput{
		Body: *healthResponse,
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
		Body: dto.ListSiteSettingsResponseBody{
			Settings:   settingOutputs,
			Total:      total,
			Page:       input.Page,
			Limit:      input.Limit,
			TotalPages: totalPages,
		},
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
		Body: dto.CreateSiteSettingResponseBody{
			Setting: m.convertToOutput(setting),
			Message: fmt.Sprintf("Site setting '%s' created successfully", setting.Key),
		},
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
		Body: dto.ListSiteSettingsResponseBody{
			Settings:   settingOutputs,
			Total:      total,
			Page:       input.Page,
			Limit:      input.Limit,
			TotalPages: totalPages,
		},
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
		Body: dto.GetSiteSettingResponseBody{
			Setting: m.convertToOutput(setting),
		},
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
		Body: dto.UpdateSiteSettingResponseBody{
			Setting: m.convertToOutput(setting),
			Message: fmt.Sprintf("Site setting '%s' updated successfully", setting.Key),
		},
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
		Body: dto.DeleteSiteSettingResponseBody{
			Message: fmt.Sprintf("Site setting '%s' deleted successfully", input.Key),
		},
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

// Corporation Management Handlers

// Add corporation handler
func (m *Module) addCorporationHandler(ctx context.Context, input *dto.AddCorporationInput) (*dto.AddCorporationOutput, error) {
	// Require super admin authentication
	user, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	corporation, err := m.service.AddManagedCorporation(ctx, input, int64(user.CharacterID))
	if err != nil {
		if fmt.Sprintf("%s", err) == fmt.Sprintf("corporation with ID %d is already managed", input.Body.CorporationID) {
			return nil, huma.Error409Conflict("Corporation already exists", err)
		}
		return nil, huma.Error500InternalServerError("Failed to add corporation", err)
	}

	return &dto.AddCorporationOutput{
		Body: dto.AddCorporationResponseBody{
			Corporation: *corporation,
			Message:     fmt.Sprintf("Corporation '%s' (ID: %d) added successfully", corporation.Name, corporation.CorporationID),
		},
	}, nil
}

// List corporations handler
func (m *Module) listCorporationsHandler(ctx context.Context, input *dto.ListManagedCorporationsInput) (*dto.ListManagedCorporationsOutput, error) {
	// Require super admin authentication
	_, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	corporations, total, err := m.service.GetManagedCorporations(ctx, input.EnabledFilter, input.Page, input.Limit)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve corporations", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(input.Limit)))

	return &dto.ListManagedCorporationsOutput{
		Body: dto.ListManagedCorporationsResponseBody{
			Corporations: corporations,
			Total:        total,
			Page:         input.Page,
			Limit:        input.Limit,
			TotalPages:   totalPages,
		},
	}, nil
}

// Get corporation handler
func (m *Module) getCorporationHandler(ctx context.Context, input *dto.GetManagedCorporationInput) (*dto.GetManagedCorporationOutput, error) {
	// Require super admin authentication
	_, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	corporation, err := m.service.GetManagedCorporation(ctx, input.CorporationID)
	if err != nil {
		if fmt.Sprintf("%s", err) == fmt.Sprintf("corporation with ID %d not found", input.CorporationID) {
			return nil, huma.Error404NotFound("Corporation not found", err)
		}
		return nil, huma.Error500InternalServerError("Failed to retrieve corporation", err)
	}

	return &dto.GetManagedCorporationOutput{
		Body: dto.GetManagedCorporationResponseBody{
			Corporation: *corporation,
		},
	}, nil
}

// Update corporation status handler
func (m *Module) updateCorporationStatusHandler(ctx context.Context, input *dto.UpdateCorporationStatusInput) (*dto.UpdateCorporationStatusOutput, error) {
	// Require super admin authentication
	user, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	corporation, err := m.service.UpdateCorporationStatus(ctx, input.CorporationID, input.Body.Enabled, int64(user.CharacterID))
	if err != nil {
		if fmt.Sprintf("%s", err) == fmt.Sprintf("corporation with ID %d not found", input.CorporationID) {
			return nil, huma.Error404NotFound("Corporation not found", err)
		}
		return nil, huma.Error500InternalServerError("Failed to update corporation status", err)
	}

	status := "disabled"
	if corporation.Enabled {
		status = "enabled"
	}

	return &dto.UpdateCorporationStatusOutput{
		Body: dto.UpdateCorporationStatusResponseBody{
			Corporation: *corporation,
			Message:     fmt.Sprintf("Corporation '%s' (ID: %d) %s successfully", corporation.Name, corporation.CorporationID, status),
		},
	}, nil
}

// Remove corporation handler
func (m *Module) removeCorporationHandler(ctx context.Context, input *dto.RemoveCorporationInput) (*dto.RemoveCorporationOutput, error) {
	// Require super admin authentication
	user, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	// Get corporation details before removing for the response message
	corporation, err := m.service.GetManagedCorporation(ctx, input.CorporationID)
	if err != nil {
		if fmt.Sprintf("%s", err) == fmt.Sprintf("corporation with ID %d not found", input.CorporationID) {
			return nil, huma.Error404NotFound("Corporation not found", err)
		}
		return nil, huma.Error500InternalServerError("Failed to retrieve corporation", err)
	}

	err = m.service.RemoveManagedCorporation(ctx, input.CorporationID, int64(user.CharacterID))
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to remove corporation", err)
	}

	return &dto.RemoveCorporationOutput{
		Body: dto.RemoveCorporationResponseBody{
			Message: fmt.Sprintf("Corporation '%s' (ID: %d) removed successfully", corporation.Name, corporation.CorporationID),
		},
	}, nil
}

// Bulk update corporations handler
func (m *Module) bulkUpdateCorporationsHandler(ctx context.Context, input *dto.BulkUpdateCorporationsInput) (*dto.BulkUpdateCorporationsOutput, error) {
	// Require super admin authentication
	user, err := m.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	corporations, updated, added, err := m.service.BulkUpdateCorporations(ctx, input, int64(user.CharacterID))
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to bulk update corporations", err)
	}

	return &dto.BulkUpdateCorporationsOutput{
		Body: dto.BulkUpdateCorporationsResponseBody{
			Corporations: corporations,
			Updated:      updated,
			Added:        added,
			Message:      fmt.Sprintf("Bulk update completed: %d corporations updated, %d corporations added", updated, added),
		},
	}, nil
}

