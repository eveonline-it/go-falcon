package routes

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	models "go-falcon/internal/auth/models"
	"go-falcon/internal/structures/dto"
	"go-falcon/internal/structures/services"
	"go-falcon/pkg/middleware"
)

// contextKey type for context keys
type contextKey string

const (
	// AuthDataKey is the context key for auth data
	AuthDataKey = contextKey("auth_data")
)

// AuthData stores authentication data
type AuthData struct {
	User  *models.AuthenticatedUser
	Token string
}

// AuthRepository interface for auth operations
type AuthRepository interface {
	GetUserProfileByCharacterID(ctx context.Context, characterID int) (*models.UserProfile, error)
}

// StructureRoutes handles structure-related HTTP routes
type StructureRoutes struct {
	service        *services.StructureService
	middleware     *middleware.PermissionMiddleware
	authRepository AuthRepository
}

// NewStructureRoutes creates a new structure routes handler
func NewStructureRoutes(service *services.StructureService, authMiddleware *middleware.PermissionMiddleware, authRepository AuthRepository) *StructureRoutes {
	return &StructureRoutes{
		service:        service,
		middleware:     authMiddleware,
		authRepository: authRepository,
	}
}

// RegisterRoutes registers all structure routes
func (r *StructureRoutes) RegisterRoutes(api huma.API) {
	// Public status endpoint
	huma.Register(api, huma.Operation{
		OperationID: "getStructuresStatus",
		Method:      http.MethodGet,
		Path:        "/structures/status",
		Summary:     "Get structures module status",
		Description: "Returns the health status of the structures module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		return &dto.StatusOutput{
			Body: dto.StructureModuleStatusResponse{
				Module:  "structures",
				Status:  "healthy",
				Message: "Structures module is operational",
			},
		}, nil
	})

	// Authenticated endpoint to get structure information
	huma.Register(api, huma.Operation{
		OperationID: "getStructure",
		Method:      http.MethodGet,
		Path:        "/structures/{structure_id}",
		Summary:     "Get structure information",
		Description: "Retrieves detailed information about a specific structure. Requires authentication and access to the structure.",
		Tags:        []string{"Structures"},
	}, r.getStructure)

	// TODO: Add remaining structure endpoints with proper authentication middleware
	// Bulk refresh, system/owner queries, etc.
}

// getStructure retrieves structure information
func (r *StructureRoutes) getStructure(ctx context.Context, input *dto.GetStructureRequest) (*dto.StructureOutput, error) {
	// Authenticate user
	user, err := r.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	// Get the user's full profile to access token information
	profile, err := r.authRepository.GetUserProfileByCharacterID(ctx, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to retrieve user profile", err)
	}
	if profile == nil {
		return nil, huma.Error500InternalServerError("user profile not found")
	}

	// Check if access token is expired and needs refresh
	if time.Now().After(profile.TokenExpiry) {
		return nil, huma.Error401Unauthorized("EVE access token expired, please re-authenticate")
	}

	// Use the EVE access token from the profile for ESI calls
	token := profile.AccessToken
	if token == "" {
		return nil, huma.Error401Unauthorized("No EVE access token available")
	}

	// Get structure information using the authenticated character ID (convert int to int32)
	structure, err := r.service.GetStructure(ctx, input.StructureID, int32(user.CharacterID), token)
	if err != nil {
		// Handle specific error cases
		if strings.Contains(err.Error(), "structure not found") {
			return nil, huma.Error404NotFound("Structure not found")
		}
		if strings.Contains(err.Error(), "access denied") {
			return nil, huma.Error403Forbidden("Access denied to structure")
		}
		return nil, huma.Error500InternalServerError("Failed to retrieve structure information", err)
	}

	// Convert to response DTO
	response := dto.ToStructureResponse(structure)

	return &dto.StructureOutput{
		Body: response,
	}, nil
}
