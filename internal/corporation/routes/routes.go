package routes

import (
	"context"
	"fmt"

	"go-falcon/internal/corporation/dto"
	"go-falcon/internal/corporation/services"
	"go-falcon/pkg/middleware"

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
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string, corporationAdapter *middleware.CorporationAdapter) {
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
		if corporationAdapter != nil {
			_, err := corporationAdapter.RequireCorporationAccess(ctx, input.Authorization, input.Cookie)
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
		if corporationAdapter != nil {
			_, err := corporationAdapter.RequireCorporationAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
		}

		return m.getCorporationInfo(ctx, input.CorporationID)
	})

	// Member tracking endpoint (authenticated, requires CEO ID)
	huma.Register(api, huma.Operation{
		OperationID: "corporation-member-tracking",
		Method:      "GET",
		Path:        basePath + "/{corporation_id}/membertracking",
		Summary:     "Track Corporation Members",
		Description: "Retrieves member tracking information for a corporation. Requires authentication and valid CEO ID that matches the corporation's CEO. Updates the tracking data in the database.",
		Tags:        []string{"Corporations"},
	}, func(ctx context.Context, input *dto.GetCorporationMemberTrackingInput) (*dto.CorporationMemberTrackingOutput, error) {
		// Require authentication
		if corporationAdapter != nil {
			_, err := corporationAdapter.RequireCorporationAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
		}

		// Call the service with the CEO ID
		return m.service.GetMemberTracking(ctx, input.CorporationID, input.CEOID)
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

	// CEO Token Validation endpoint (super_admin only)
	huma.Register(api, huma.Operation{
		OperationID: "corporation-validate-ceo-tokens",
		Method:      "POST",
		Path:        basePath + "/validate-ceo-tokens",
		Summary:     "Validate CEO Tokens",
		Description: "Validates all CEO tokens and returns detailed results about invalid or missing tokens. This endpoint requires super_admin privileges and may take a while to complete for large datasets.",
		Tags:        []string{"Corporations", "Administration"},
	}, func(ctx context.Context, input *dto.ValidateCEOTokensInput) (*dto.ValidateCEOTokensOutput, error) {
		// Require super_admin privileges
		if corporationAdapter != nil {
			_, err := corporationAdapter.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
		}

		// Run the validation and return results
		results, err := m.service.ValidateCEOTokensWithResults(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to validate CEO tokens", err)
		}

		return &dto.ValidateCEOTokensOutput{Body: *results}, nil
	})

	// Alliance history endpoint (authenticated)
	huma.Register(api, huma.Operation{
		OperationID: "corporation-alliance-history",
		Method:      "GET",
		Path:        basePath + "/{corporation_id}/alliancehistory",
		Summary:     "Get Corporation Alliance History",
		Description: "Retrieves the complete alliance history for a corporation from EVE Online ESI API. Shows when the corporation joined and left alliances. Requires authentication.",
		Tags:        []string{"Corporations"},
	}, func(ctx context.Context, input *dto.GetCorporationAllianceHistoryInput) (*dto.CorporationAllianceHistoryOutput, error) {
		// Require authentication
		if corporationAdapter != nil {
			_, err := corporationAdapter.RequireCorporationAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
		}

		return m.getCorporationAllianceHistory(ctx, input.CorporationID)
	})

	// Corporation members endpoint (authenticated, requires CEO ID)
	huma.Register(api, huma.Operation{
		OperationID: "corporation-get-members",
		Method:      "GET",
		Path:        basePath + "/{corporation_id}/members",
		Summary:     "Get Corporation Members",
		Description: "Retrieves the list of corporation members from EVE Online ESI API. Requires authentication and valid CEO ID that matches the corporation's CEO. Returns a list of character IDs for all corporation members.",
		Tags:        []string{"Corporations"},
	}, func(ctx context.Context, input *dto.GetCorporationMembersInput) (*dto.CorporationMembersOutput, error) {
		// Require authentication
		if corporationAdapter != nil {
			_, err := corporationAdapter.RequireCorporationAccess(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}
		}

		return m.getCorporationMembers(ctx, input.CorporationID, input.CEOID)
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

// getCorporationAllianceHistory handles the corporation alliance history request
func (m *Module) getCorporationAllianceHistory(ctx context.Context, corporationID int) (*dto.CorporationAllianceHistoryOutput, error) {
	if corporationID <= 0 {
		return nil, huma.Error400BadRequest("Corporation ID must be a positive integer")
	}

	// Call the service to get alliance history
	history, err := m.service.GetCorporationAllianceHistory(ctx, corporationID)
	if err != nil {
		// Check if it's a 404 from ESI (corporation not found)
		if isNotFoundError(err) {
			return nil, huma.Error404NotFound(fmt.Sprintf("Corporation with ID %d not found", corporationID))
		}

		// For other errors, return a 500
		return nil, huma.Error500InternalServerError("Failed to retrieve corporation alliance history", err)
	}

	return history, nil
}

// getCorporationMembers handles the corporation members request
func (m *Module) getCorporationMembers(ctx context.Context, corporationID int, ceoID int) (*dto.CorporationMembersOutput, error) {
	if corporationID <= 0 {
		return nil, huma.Error400BadRequest("Corporation ID must be a positive integer")
	}

	if ceoID <= 0 {
		return nil, huma.Error400BadRequest("CEO ID must be a positive integer")
	}

	// Call the service to get corporation members
	members, err := m.service.GetCorporationMembers(ctx, corporationID, ceoID)
	if err != nil {
		// Check if it's a 404 from ESI (corporation not found)
		if isNotFoundError(err) {
			return nil, huma.Error404NotFound(fmt.Sprintf("Corporation with ID %d not found", corporationID))
		}

		// Check for CEO validation errors
		if err.Error() == fmt.Sprintf("invalid CEO ID for corporation %d", corporationID) {
			return nil, huma.Error403Forbidden("The provided CEO ID does not match the corporation's CEO")
		}

		// Check for missing CEO profile
		if err.Error() == fmt.Sprintf("CEO profile not found for character ID %d", ceoID) {
			return nil, huma.Error404NotFound("CEO profile not found")
		}

		// Check for invalid token
		if err.Error() == "CEO does not have a valid access token" {
			return nil, huma.Error403Forbidden("CEO does not have a valid access token")
		}

		// For other errors, return a 500
		return nil, huma.Error500InternalServerError("Failed to retrieve corporation members", err)
	}

	return members, nil
}

// isNotFoundError checks if the error indicates a corporation was not found
func isNotFoundError(err error) bool {
	// This is a simple check - in a real implementation, you'd want to
	// examine the specific error type or HTTP status code from the ESI client
	return err != nil && (err.Error() == "ESI returned status 404" ||
		err.Error() == "corporation not found")
}
