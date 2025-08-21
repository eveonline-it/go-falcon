package routes

import (
	"context"
	"fmt"

	"go-falcon/internal/alliance/dto"
	"go-falcon/internal/alliance/services"

	"github.com/danielgtaylor/huma/v2"
)

// AllianceHealthCheck represents a health check response data
type AllianceHealthCheck struct {
	Healthy bool   `json:"healthy" description:"Whether the module is healthy"`
	Module  string `json:"module" description:"Module name"`
}

// AllianceHealthResponse represents a health check response (Huma wrapper)
type AllianceHealthResponse struct {
	Body AllianceHealthCheck `json:"body"`
}

// Module represents the alliance routes module
type Module struct {
	service *services.Service
}

// NewModule creates a new alliance routes module
func NewModule(service *services.Service) *Module {
	return &Module{
		service: service,
	}
}

// RegisterUnifiedRoutes registers all alliance routes with the provided Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	// Alliance information endpoint
	huma.Register(api, huma.Operation{
		OperationID: "alliance-get-info",
		Method:      "GET",
		Path:        basePath + "/{alliance_id}",
		Summary:     "Get Alliance Information",
		Description: "Retrieve detailed information about an alliance from EVE Online ESI API. Data is cached locally for performance.",
		Tags:        []string{"Alliances"},
	}, func(ctx context.Context, input *dto.GetAllianceInput) (*dto.AllianceInfoOutput, error) {
		return m.getAllianceInfo(ctx, input)
	})
	
	// Health check endpoint for the alliance module
	huma.Register(api, huma.Operation{
		OperationID: "alliance-health-check",
		Method:      "GET",
		Path:        basePath + "/health",
		Summary:     "Alliance Module Health Check",
		Description: "Check if the alliance module is functioning properly",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, input *struct{}) (*AllianceHealthResponse, error) {
		return &AllianceHealthResponse{
			Body: AllianceHealthCheck{
				Healthy: true,
				Module:  "alliance",
			},
		}, nil
	})
}

// getAllianceInfo handles the alliance information request
func (m *Module) getAllianceInfo(ctx context.Context, input *dto.GetAllianceInput) (*dto.AllianceInfoOutput, error) {
	if input.AllianceID <= 0 {
		return nil, huma.Error400BadRequest("Alliance ID must be a positive integer")
	}
	
	// Call the service to get alliance information
	allianceInfo, err := m.service.GetAllianceInfo(ctx, input.AllianceID)
	if err != nil {
		// Check if it's a 404 from ESI (alliance not found)
		if isNotFoundError(err) {
			return nil, huma.Error404NotFound(fmt.Sprintf("Alliance with ID %d not found", input.AllianceID))
		}
		
		// For other errors, return a 500
		return nil, huma.Error500InternalServerError("Failed to retrieve alliance information", err)
	}
	
	return allianceInfo, nil
}

// isNotFoundError checks if the error indicates an alliance was not found
func isNotFoundError(err error) bool {
	// This is a simple check - in a real implementation, you'd want to 
	// examine the specific error type or HTTP status code from the ESI client
	return err != nil && (
		err.Error() == "ESI returned status 404" ||
		err.Error() == "alliance not found")
}