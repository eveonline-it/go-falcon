package routes

import (
	"context"
	"fmt"

	"go-falcon/internal/corporation/dto"
	"go-falcon/internal/corporation/middleware"
	"go-falcon/internal/corporation/services"

	"github.com/danielgtaylor/huma/v2"
)

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
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string, authMiddleware *middleware.AuthMiddleware) {
	// Search corporations by name endpoint (authenticated)
	huma.Register(api, huma.Operation{
		OperationID: "corporation-search-by-name",
		Method:      "GET",
		Path:        basePath + "/search",
		Summary:     "Search Corporations by Name",
		Description: "Search corporations by name or ticker with a minimum of 3 characters. Performs case-insensitive search in the database and supports partial matches. Requires authentication.",
		Tags:        []string{"Corporations"},
	}, func(ctx context.Context, input *dto.SearchCorporationsByNameAuthInput) (*dto.SearchCorporationsByNameOutput, error) {
		// Require authentication
		if authMiddleware != nil {
			_, err := authMiddleware.RequireCorporationAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
		}

		return m.searchCorporationsByName(ctx, input.Name)
	})

	// Corporation information endpoint (authenticated)
	huma.Register(api, huma.Operation{
		OperationID: "corporation-get-info",
		Method:      "GET",
		Path:        basePath + "/{corporation_id}",
		Summary:     "Get Corporation Information",
		Description: "Retrieve detailed information about a corporation from EVE Online ESI API. Data is cached locally for performance. Requires authentication.",
		Tags:        []string{"Corporations"},
	}, func(ctx context.Context, input *dto.GetCorporationAuthInput) (*dto.CorporationInfoOutput, error) {
		// Require authentication
		if authMiddleware != nil {
			_, err := authMiddleware.RequireCorporationAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
		}

		return m.getCorporationInfo(ctx, input.CorporationID)
	})

	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "corporation-get-status",
		Method:      "GET",
		Path:        basePath + "/status",
		Summary:     "Get corporation module status",
		Description: "Returns the health status of the corporation module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		status := m.service.GetStatus(ctx)
		return &dto.StatusOutput{Body: *status}, nil
	})
}

// getCorporationInfo handles the corporation information request
func (m *Module) getCorporationInfo(ctx context.Context, corporationID int) (*dto.CorporationInfoOutput, error) {
	if corporationID <= 0 {
		return nil, huma.Error400BadRequest("Corporation ID must be a positive integer")
	}

	// Call the service to get corporation information
	corpInfo, err := m.service.GetCorporationInfo(ctx, corporationID)
	if err != nil {
		// Check if it's a 404 from ESI (corporation not found)
		if isNotFoundError(err) {
			return nil, huma.Error404NotFound(fmt.Sprintf("Corporation with ID %d not found", corporationID))
		}

		// For other errors, return a 500
		return nil, huma.Error500InternalServerError("Failed to retrieve corporation information", err)
	}

	return corpInfo, nil
}

// searchCorporationsByName handles the corporation search request
func (m *Module) searchCorporationsByName(ctx context.Context, name string) (*dto.SearchCorporationsByNameOutput, error) {
	if len(name) < 3 {
		return nil, huma.Error400BadRequest("Search term must be at least 3 characters long")
	}

	// Call the service to search for corporations
	results, err := m.service.SearchCorporationsByName(ctx, name)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to search corporations", err)
	}

	return results, nil
}

// isNotFoundError checks if the error indicates a corporation was not found
func isNotFoundError(err error) bool {
	// This is a simple check - in a real implementation, you'd want to
	// examine the specific error type or HTTP status code from the ESI client
	return err != nil && (err.Error() == "ESI returned status 404" ||
		err.Error() == "corporation not found")
}
