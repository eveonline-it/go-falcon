package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/assets/dto"
	"go-falcon/internal/assets/services"
	models "go-falcon/internal/auth/models"
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

// AssetRoutes handles asset-related HTTP routes
type AssetRoutes struct {
	service        *services.AssetService
	middleware     *middleware.PermissionMiddleware
	authRepository AuthRepository
}

// NewAssetRoutes creates a new asset routes handler
func NewAssetRoutes(service *services.AssetService, authMiddleware *middleware.PermissionMiddleware, authRepository AuthRepository) *AssetRoutes {
	return &AssetRoutes{
		service:        service,
		middleware:     authMiddleware,
		authRepository: authRepository,
	}
}

// RegisterRoutes registers all asset routes
func (r *AssetRoutes) RegisterRoutes(api huma.API) {
	// Public status endpoint
	huma.Register(api, huma.Operation{
		OperationID: "getAssetsStatus",
		Method:      http.MethodGet,
		Path:        "/assets/status",
		Summary:     "Get assets module status",
		Description: "Returns the health status of the assets module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		return &dto.StatusOutput{
			Body: dto.AssetModuleStatusResponse{
				Module:  "assets",
				Status:  "healthy",
				Message: "Assets module is operational",
			},
		}, nil
	})

	// Character assets endpoint - requires authentication and ownership
	huma.Register(api, huma.Operation{
		OperationID: "getCharacterAssets",
		Method:      http.MethodGet,
		Path:        "/assets/character/{character_id}",
		Summary:     "Get character assets",
		Description: "Retrieves assets for a specific character including station/structure names",
		Tags:        []string{"Assets"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.GetCharacterAssetsRequest) (*dto.AssetListOutput, error) {
		// Authenticate user
		user, err := r.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Check ownership OR super admin permissions
		isSuperAdmin := false
		if int32(user.CharacterID) != input.CharacterID {
			// Check if user is super admin
			_, err := r.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, huma.Error403Forbidden("You can only view your own assets")
			}
			isSuperAdmin = true
		}

		// Get the appropriate user profile and token
		var profile *models.UserProfile
		if isSuperAdmin {
			// Use target character's token for ESI calls
			profile, err = r.authRepository.GetUserProfileByCharacterID(ctx, int(input.CharacterID))
		} else {
			// Use requester's token
			profile, err = r.authRepository.GetUserProfileByCharacterID(ctx, user.CharacterID)
		}
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

		// Parse location ID filter if provided
		var locationIDPtr *int64
		if input.LocationID > 0 {
			locationIDPtr = &input.LocationID
		}

		// Get assets from service
		assets, total, err := r.service.GetCharacterAssets(
			ctx,
			input.CharacterID,
			token,
			locationIDPtr,
		)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to retrieve assets", err)
		}

		// Convert assets to response DTOs
		responseAssets := make([]dto.AssetResponse, len(assets))
		var totalValue float64
		for i, asset := range assets {
			responseAssets[i] = dto.ToAssetResponse(asset)
			totalValue += asset.TotalValue
		}

		// Get last updated time (use first asset's update time if available)
		var lastUpdated time.Time
		if len(assets) > 0 {
			lastUpdated = assets[0].UpdatedAt
		} else {
			lastUpdated = time.Now()
		}

		return &dto.AssetListOutput{
			Body: dto.AssetListResponse{
				Assets:      responseAssets,
				Total:       total,
				TotalValue:  totalValue,
				LastUpdated: lastUpdated,
			},
		}, nil
	})

	// Refresh character assets endpoint
	huma.Register(api, huma.Operation{
		OperationID: "refreshCharacterAssets",
		Method:      http.MethodPost,
		Path:        "/assets/character/{character_id}/refresh",
		Summary:     "Refresh character assets",
		Description: "Forces a refresh of character assets from ESI",
		Tags:        []string{"Assets"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.RefreshCharacterAssetsRequest) (*dto.RefreshAssetsOutput, error) {
		// Authenticate user
		user, err := r.middleware.RequireAuth(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Check ownership OR super admin permissions
		isSuperAdmin := false
		if int32(user.CharacterID) != input.CharacterID {
			// Check if user is super admin
			_, err := r.middleware.RequireSuperAdmin(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, huma.Error403Forbidden("You can only refresh your own assets")
			}
			isSuperAdmin = true
		}

		// Get the appropriate user profile and token
		var profile *models.UserProfile
		if isSuperAdmin {
			// Use target character's token for ESI calls
			profile, err = r.authRepository.GetUserProfileByCharacterID(ctx, int(input.CharacterID))
		} else {
			// Use requester's token
			profile, err = r.authRepository.GetUserProfileByCharacterID(ctx, user.CharacterID)
		}
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

		// Refresh assets from ESI
		updated, newItems, removedItems, err := r.service.RefreshCharacterAssets(
			ctx,
			input.CharacterID,
			token,
		)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to refresh assets", err)
		}

		// Get total value after refresh
		assets, _, _ := r.service.GetCharacterAssets(ctx, input.CharacterID, token, nil)
		var totalValue float64
		for _, asset := range assets {
			totalValue += asset.TotalValue
		}

		return &dto.RefreshAssetsOutput{
			Body: dto.RefreshAssetsResponse{
				CharacterID:  input.CharacterID,
				ItemsUpdated: updated,
				NewItems:     newItems,
				RemovedItems: removedItems,
				TotalValue:   totalValue,
				UpdatedAt:    time.Now(),
			},
		}, nil
	})

	// Structure access monitoring endpoint
	huma.Register(api, huma.Operation{
		OperationID: "getStructureAccessStats",
		Method:      "GET",
		Path:        "/assets/structure-access-stats",
		Summary:     "Get structure access statistics",
		Description: "Returns statistics about failed structure access attempts for monitoring purposes",
		Tags:        []string{"Assets"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		CharacterID int32 `query:"character_id" doc:"Optional character ID to filter stats, 0 for global stats"`
	}) (*dto.StructureAccessStatsOutput, error) {
		// Get authenticated user from context (authentication is handled by API gateway)
		authData, ok := ctx.Value(AuthDataKey).(*AuthData)
		if !ok || authData == nil || authData.User == nil {
			return nil, huma.Error401Unauthorized("authentication required")
		}

		// Admin users can view all stats, regular users only their own
		var queryCharID *int32
		if input.CharacterID > 0 {
			// Check if user has permission to view this character's stats
			if input.CharacterID != int32(authData.User.CharacterID) {
				// TODO: Add admin check here
				return nil, huma.Error403Forbidden("not authorized to view this character's statistics")
			}
			queryCharID = &input.CharacterID
		}

		stats, err := r.service.GetStructureAccessStats(ctx, queryCharID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get structure access stats", err)
		}

		return &dto.StructureAccessStatsOutput{
			Body: stats,
		}, nil
	})

	// TODO: Add remaining asset endpoints
	// Corporation assets, tracking endpoints, etc.
}
