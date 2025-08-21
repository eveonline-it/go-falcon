package routes

import (
	"context"
	"fmt"

	"go-falcon/internal/corporation/dto"
	"go-falcon/internal/corporation/services"

	"github.com/danielgtaylor/huma/v2"
)

// HealthResponse represents a health check response
type HealthResponse struct {
	Healthy bool   `json:"healthy" description:"Whether the module is healthy"`
	Module  string `json:"module" description:"Module name"`
}

// Module represents the corporation routes module
type Module struct {
	service *services.Service
}

// NewModule creates a new corporation routes module
func NewModule(service *services.Service) *Module {
	return &Module{
		service: service,
	}
}

// RegisterUnifiedRoutes registers all corporation routes with the provided Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API) {
	// Corporation information endpoint
	huma.Register(api, huma.Operation{
		OperationID: "corporation-get-info",
		Method:      "GET",
		Path:        "/corporations/{corporation_id}",
		Summary:     "Get Corporation Information",
		Description: "Retrieve detailed information about a corporation from EVE Online ESI API. Data is cached locally for performance.",
		Tags:        []string{"Corporations"},
	}, func(ctx context.Context, input *dto.GetCorporationInput) (*dto.CorporationInfoOutput, error) {
		return m.getCorporationInfo(ctx, input)
	})
	
	// Health check endpoint for the corporation module
	huma.Register(api, huma.Operation{
		OperationID: "corporation-health-check",
		Method:      "GET",
		Path:        "/corporations/health",
		Summary:     "Corporation Module Health Check",
		Description: "Check if the corporation module is functioning properly",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, input *struct{}) (*HealthResponse, error) {
		return &HealthResponse{
			Healthy: true,
			Module:  "corporation",
		}, nil
	})
}

// getCorporationInfo handles the corporation information request
func (m *Module) getCorporationInfo(ctx context.Context, input *dto.GetCorporationInput) (*dto.CorporationInfoOutput, error) {
	if input.CorporationID <= 0 {
		return nil, huma.Error400BadRequest("Corporation ID must be a positive integer")
	}
	
	// Call the service to get corporation information
	corpInfo, err := m.service.GetCorporationInfo(ctx, input.CorporationID)
	if err != nil {
		// Check if it's a 404 from ESI (corporation not found)
		if isNotFoundError(err) {
			return nil, huma.Error404NotFound(fmt.Sprintf("Corporation with ID %d not found", input.CorporationID))
		}
		
		// For other errors, return a 500
		return nil, huma.Error500InternalServerError("Failed to retrieve corporation information", err)
	}
	
	return corpInfo, nil
}

// isNotFoundError checks if the error indicates a corporation was not found
func isNotFoundError(err error) bool {
	// This is a simple check - in a real implementation, you'd want to 
	// examine the specific error type or HTTP status code from the ESI client
	return err != nil && (
		err.Error() == "ESI returned status 404" ||
		err.Error() == "corporation not found")
}