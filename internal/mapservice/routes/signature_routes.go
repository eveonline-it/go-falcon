package routes

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/mapservice/dto"
	"go-falcon/internal/mapservice/services"
	"go-falcon/pkg/middleware"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterSignatureRoutes registers protected signature management endpoints
func RegisterSignatureRoutes(api huma.API, basePath string, service *services.MapService, mapAdapter *middleware.MapAdapter) {
	// Create signature
	huma.Register(api, huma.Operation{
		OperationID: "map-create-signature",
		Method:      http.MethodPost,
		Path:        basePath + "/signatures",
		Summary:     "Create signature",
		Description: "Create a new map signature",
		Tags:        []string{"Map / Signatures"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.CreateSignatureInputWithAuth) (*dto.SignatureResponseOutput, error) {
		// Validate authentication and signature management access
		user, err := mapAdapter.RequireSignatureManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		// Get user's default group for creating new signatures
		groupID, err := service.GetUserDefaultGroupID(ctx, user.UserID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user group information", err)
		}

		signature, err := service.CreateSignature(ctx, userID, user.CharacterName, groupID, input.CreateSignatureInput)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create signature", err)
		}

		// Get system name for output
		system, _ := service.GetSDE().GetSolarSystem(int(signature.SystemID))
		systemName := services.GetSystemName(service.GetSDE(), system)

		result := dto.SignatureToOutput(signature, systemName)
		return &dto.SignatureResponseOutput{Body: result}, nil
	})

	// List signatures
	huma.Register(api, huma.Operation{
		OperationID: "map-list-signatures",
		Method:      http.MethodGet,
		Path:        basePath + "/signatures",
		Summary:     "List signatures",
		Description: "List map signatures with filtering",
		Tags:        []string{"Map / Signatures"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.GetSignaturesInputWithAuth) (*dto.SignatureListResponseOutput, error) {
		// Validate authentication and map access
		user, err := mapAdapter.RequireMapAccess(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		// Get user's group IDs for access control
		groupIDs, err := service.GetUserGroupIDs(ctx, user.UserID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user group information", err)
		}

		signatures, err := service.GetSignatures(ctx, userID, groupIDs, input.GetSignaturesInput)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get signatures", err)
		}

		// Convert to output format with system names
		var results []dto.SignatureOutput
		for _, sig := range signatures {
			system, _ := service.GetSDE().GetSolarSystem(int(sig.SystemID))
			systemName := services.GetSystemName(service.GetSDE(), system)
			result := dto.SignatureToOutput(&sig, systemName)
			results = append(results, result)
		}

		return &dto.SignatureListResponseOutput{Body: results}, nil
	})

	// Get signature by ID
	huma.Register(api, huma.Operation{
		OperationID: "map-get-signature",
		Method:      http.MethodGet,
		Path:        basePath + "/signatures/{signature_id}",
		Summary:     "Get signature",
		Description: "Get a specific signature by ID",
		Tags:        []string{"Map / Signatures"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.GetSignatureInputWithAuth) (*dto.SignatureResponseOutput, error) {
		// Validate authentication and map access
		user, err := mapAdapter.RequireMapAccess(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert string ID to ObjectID
		signatureID, err := primitive.ObjectIDFromHex(input.SignatureID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid signature ID", err)
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		// Get user's group IDs for access control
		groupIDs, err := service.GetUserGroupIDs(ctx, user.UserID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user group information", err)
		}

		signature, err := service.GetSignatureByID(ctx, signatureID, userID, groupIDs)
		if err != nil {
			return nil, huma.Error404NotFound("Signature not found", err)
		}

		// Get system name for output
		system, _ := service.GetSDE().GetSolarSystem(int(signature.SystemID))
		systemName := services.GetSystemName(service.GetSDE(), system)

		result := dto.SignatureToOutput(signature, systemName)
		return &dto.SignatureResponseOutput{Body: result}, nil
	})

	// Update signature
	huma.Register(api, huma.Operation{
		OperationID: "map-update-signature",
		Method:      http.MethodPut,
		Path:        basePath + "/signatures/{signature_id}",
		Summary:     "Update signature",
		Description: "Update an existing signature",
		Tags:        []string{"Map / Signatures"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.UpdateSignatureInputWithAuth) (*dto.SignatureResponseOutput, error) {
		// Validate authentication and signature management access
		user, err := mapAdapter.RequireSignatureManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert string ID to ObjectID
		signatureID, err := primitive.ObjectIDFromHex(input.SignatureID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid signature ID", err)
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		signature, err := service.UpdateSignatureForRoute(ctx, signatureID, userID, input.UpdateSignatureInput)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to update signature", err)
		}

		// Get system name for output
		system, _ := service.GetSDE().GetSolarSystem(int(signature.SystemID))
		systemName := services.GetSystemName(service.GetSDE(), system)

		result := dto.SignatureToOutput(signature, systemName)
		return &dto.SignatureResponseOutput{Body: result}, nil
	})

	// Delete signature
	huma.Register(api, huma.Operation{
		OperationID: "map-delete-signature",
		Method:      http.MethodDelete,
		Path:        basePath + "/signatures/{signature_id}",
		Summary:     "Delete signature",
		Description: "Delete a signature",
		Tags:        []string{"Map / Signatures"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.DeleteSignatureInputWithAuth) (*dto.DeleteSignatureResponseOutput, error) {
		// Validate authentication and signature management access
		user, err := mapAdapter.RequireSignatureManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert string ID to ObjectID
		signatureID, err := primitive.ObjectIDFromHex(input.SignatureID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid signature ID", err)
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		err = service.DeleteSignatureForRoute(ctx, signatureID, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to delete signature", err)
		}

		return &dto.DeleteSignatureResponseOutput{
			Body: struct {
				Success bool   `json:"success" doc:"Operation success"`
				Message string `json:"message" doc:"Response message"`
			}{
				Success: true,
				Message: "Signature deleted successfully",
			},
		}, nil
	})

	// Batch signature operations
	huma.Register(api, huma.Operation{
		OperationID: "map-batch-signatures",
		Method:      http.MethodPost,
		Path:        basePath + "/signatures/batch",
		Summary:     "Batch signature operations",
		Description: "Create, update, or delete multiple signatures in one operation",
		Tags:        []string{"Map / Signatures"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.BatchSignatureInputWithAuth) (*dto.BatchSignatureOutput, error) {
		// Validate authentication and signature management access
		user, err := mapAdapter.RequireSignatureManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(user.UserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid user ID", err)
		}

		// Get user's default group for creating new signatures
		groupID, err := service.GetUserDefaultGroupID(ctx, user.UserID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user group information", err)
		}

		result, err := service.BatchUpdateSignatures(ctx, userID, user.CharacterName, groupID, input.BatchSignatureInput)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to process batch operations", err)
		}

		return result, nil
	})
}
