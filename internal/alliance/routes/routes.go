package routes

import (
	"context"
	"fmt"

	"go-falcon/internal/alliance/dto"
	"go-falcon/internal/alliance/services"

	"github.com/danielgtaylor/huma/v2"
)


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
	// Search alliances by name endpoint
	huma.Register(api, huma.Operation{
		OperationID: "alliance-search-by-name",
		Method:      "GET",
		Path:        basePath + "/search",
		Summary:     "Search Alliances by Name",
		Description: "Search alliances by name or ticker with a minimum of 3 characters. Performs case-insensitive search in the database and supports partial matches.",
		Tags:        []string{"Alliances"},
	}, func(ctx context.Context, input *dto.SearchAlliancesByNameInput) (*dto.SearchAlliancesByNameOutput, error) {
		return m.searchAlliancesByName(ctx, input)
	})

	// List all alliances endpoint
	huma.Register(api, huma.Operation{
		OperationID: "alliance-list-all",
		Method:      "GET",
		Path:        basePath,
		Summary:     "List All Alliances",
		Description: "Retrieve a list of all active alliance IDs from EVE Online ESI API. Returns an array of alliance IDs.",
		Tags:        []string{"Alliances"},
	}, func(ctx context.Context, input *dto.ListAlliancesInput) (*dto.AllianceListOutput, error) {
		return m.listAlliances(ctx, input)
	})
	
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
	
	// Alliance member corporations endpoint
	huma.Register(api, huma.Operation{
		OperationID: "alliance-get-corporations",
		Method:      "GET",
		Path:        basePath + "/{alliance_id}/corporations",
		Summary:     "List Alliance Member Corporations",
		Description: "Retrieve a list of corporation IDs that are members of the specified alliance from EVE Online ESI API.",
		Tags:        []string{"Alliances"},
	}, func(ctx context.Context, input *dto.GetAllianceCorporationsInput) (*dto.AllianceCorporationsOutput, error) {
		return m.getAllianceCorporations(ctx, input)
	})
	
	// Bulk import alliances endpoint
	huma.Register(api, huma.Operation{
		OperationID: "alliance-bulk-import",
		Method:      "POST",
		Path:        basePath + "/bulk-import",
		Summary:     "Bulk Import All Alliances",
		Description: "Retrieve all alliance IDs from ESI and import detailed information for each alliance into the database. This operation respects ESI rate limits and provides progress statistics.",
		Tags:        []string{"Alliances", "Import"},
	}, func(ctx context.Context, input *struct{}) (*dto.BulkImportAlliancesOutput, error) {
		return m.bulkImportAlliances(ctx, input)
	})

	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "alliance-get-status",
		Method:      "GET",
		Path:        basePath + "/status",
		Summary:     "Get alliance module status",
		Description: "Returns the health status of the alliance module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		status := m.service.GetStatus(ctx)
		return &dto.StatusOutput{Body: *status}, nil
	})
}

// listAlliances handles the list all alliances request
func (m *Module) listAlliances(ctx context.Context, input *dto.ListAlliancesInput) (*dto.AllianceListOutput, error) {
	// Call the service to get all alliance IDs
	alliances, err := m.service.GetAllAlliances(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve alliances list", err)
	}
	
	return alliances, nil
}

// getAllianceCorporations handles the alliance member corporations request
func (m *Module) getAllianceCorporations(ctx context.Context, input *dto.GetAllianceCorporationsInput) (*dto.AllianceCorporationsOutput, error) {
	if input.AllianceID <= 0 {
		return nil, huma.Error400BadRequest("Alliance ID must be a positive integer")
	}
	
	// Call the service to get alliance member corporations
	corporations, err := m.service.GetAllianceCorporations(ctx, input.AllianceID)
	if err != nil {
		// Check if it's a 404 from ESI (alliance not found)
		if isNotFoundError(err) {
			return nil, huma.Error404NotFound(fmt.Sprintf("Alliance with ID %d not found", input.AllianceID))
		}
		
		// For other errors, return a 500
		return nil, huma.Error500InternalServerError("Failed to retrieve alliance corporations", err)
	}
	
	return corporations, nil
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

// searchAlliancesByName handles the alliance search request
func (m *Module) searchAlliancesByName(ctx context.Context, input *dto.SearchAlliancesByNameInput) (*dto.SearchAlliancesByNameOutput, error) {
	if len(input.Name) < 3 {
		return nil, huma.Error400BadRequest("Search term must be at least 3 characters long")
	}
	
	// Call the service to search for alliances
	results, err := m.service.SearchAlliancesByName(ctx, input.Name)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to search alliances", err)
	}
	
	return results, nil
}

// bulkImportAlliances handles the bulk alliance import request
func (m *Module) bulkImportAlliances(ctx context.Context, input *struct{}) (*dto.BulkImportAlliancesOutput, error) {
	// Call the service to perform bulk import
	result, err := m.service.BulkImportAlliances(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to bulk import alliances", err)
	}
	
	return result, nil
}

// isNotFoundError checks if the error indicates an alliance was not found
func isNotFoundError(err error) bool {
	// This is a simple check - in a real implementation, you'd want to 
	// examine the specific error type or HTTP status code from the ESI client
	return err != nil && (
		err.Error() == "ESI returned status 404" ||
		err.Error() == "alliance not found")
}