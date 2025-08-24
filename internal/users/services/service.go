package services

import (
	"context"
	"fmt"

	characterServices "go-falcon/internal/character/services"
	"go-falcon/internal/groups/services"
	"go-falcon/internal/users/dto"
	"go-falcon/internal/users/models"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
)

// Service provides business logic for user operations
type Service struct {
	repository       *Repository
	groupService     *services.Service
	characterService *characterServices.Service
}

// NewService creates a new service instance
func NewService(mongodb *database.MongoDB, eveGateway *evegateway.Client) *Service {
	return &Service{
		repository:       NewRepository(mongodb),
		groupService:     nil, // Will be set after initialization
		characterService: characterServices.NewService(mongodb, eveGateway),
	}
}

// SetGroupService sets the groups service dependency
func (s *Service) SetGroupService(groupService *services.Service) {
	s.groupService = groupService
}

// GetUser retrieves a user by character ID
func (s *Service) GetUser(ctx context.Context, characterID int) (*models.User, error) {
	return s.repository.GetUser(ctx, characterID)
}

// GetUserByUserID retrieves a user by user ID
func (s *Service) GetUserByUserID(ctx context.Context, userID string) (*models.User, error) {
	return s.repository.GetUserByUserID(ctx, userID)
}

// UpdateUser updates user status and administrative fields
func (s *Service) UpdateUser(ctx context.Context, characterID int, req dto.UserUpdateRequest) (*models.User, error) {
	// Validate the update request
	if err := dto.ValidateUserUpdateRequest(&req); err != nil {
		return nil, err
	}

	return s.repository.UpdateUser(ctx, characterID, req)
}

// GetUserStats returns user statistics
func (s *Service) GetUserStats(ctx context.Context) (*dto.UserStatsResponse, error) {
	return s.repository.GetUserStats(ctx)
}

// ListCharacters retrieves character summaries for a specific user ID
func (s *Service) ListCharacters(ctx context.Context, userID string) ([]dto.CharacterSummaryResponse, error) {
	return s.repository.ListCharacters(ctx, userID)
}

// ListEnrichedCharacters retrieves enriched character summaries with profile data for a specific user ID
func (s *Service) ListEnrichedCharacters(ctx context.Context, userID string) ([]dto.EnrichedCharacterSummaryResponse, error) {
	// Get basic character summaries from repository
	basicCharacters, err := s.repository.ListCharacters(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert basic summaries to enriched summaries with additional profile data
	enrichedCharacters := make([]dto.EnrichedCharacterSummaryResponse, len(basicCharacters))

	for i, basicChar := range basicCharacters {
		enriched := dto.EnrichedCharacterSummaryResponse{
			// Copy basic user management data
			CharacterID:   basicChar.CharacterID,
			CharacterName: basicChar.CharacterName,
			UserID:        basicChar.UserID,
			Enabled:       basicChar.Enabled,
			Banned:        basicChar.Banned,
			Position:      basicChar.Position,
			LastLogin:     basicChar.LastLogin,
		}

		// Try to fetch additional character profile data (optional enhancement)
		if s.characterService != nil {
			profileOutput, profileErr := s.characterService.GetCharacterProfile(ctx, basicChar.CharacterID)
			if profileErr == nil && profileOutput != nil {
				profile := profileOutput.Body
				enriched.CorporationID = &profile.CorporationID
				enriched.SecurityStatus = &profile.SecurityStatus
				enriched.Birthday = &profile.Birthday
				enriched.Gender = &profile.Gender
				enriched.RaceID = &profile.RaceID
				enriched.BloodlineID = &profile.BloodlineID

				if profile.AllianceID != 0 {
					enriched.AllianceID = &profile.AllianceID
				}
				if profile.AncestryID != 0 {
					enriched.AncestryID = &profile.AncestryID
				}
				if profile.FactionID != 0 {
					enriched.FactionID = &profile.FactionID
				}
				if profile.Description != "" {
					enriched.Description = &profile.Description
				}
			}
			// Note: Portrait fetching is commented out to avoid potential rate limiting issues
			// In production, consider implementing portrait caching or batch fetching
			// if portraitData, portraitErr := eveGateway.GetCharacterPortrait(ctx, basicChar.CharacterID); portraitErr == nil {
			//     enriched.Portraits = &dto.CharacterPortraits{
			//         Px64x64:   portraitData.Px64x64,
			//         Px128x128: portraitData.Px128x128,
			//         Px256x256: portraitData.Px256x256,
			//         Px512x512: portraitData.Px512x512,
			//     }
			// }
		}

		enrichedCharacters[i] = enriched
	}

	return enrichedCharacters, nil
}

// ListUsers retrieves paginated and filtered users
func (s *Service) ListUsers(ctx context.Context, input dto.UserListInput) (*dto.UserListResponse, error) {
	return s.repository.ListUsers(ctx, input)
}

// UserToResponse converts a User model to UserResponse DTO
func (s *Service) UserToResponse(user *models.User) *dto.UserResponse {
	return &dto.UserResponse{
		CharacterID:   user.CharacterID,
		UserID:        user.UserID,
		Enabled:       user.Enabled,
		Banned:        user.Banned,
		Invalid:       user.Invalid,
		Scopes:        user.Scopes,
		Position:      user.Position,
		Notes:         user.Notes,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
		LastLogin:     user.LastLogin,
		CharacterName: user.CharacterName,
		Valid:         user.Valid,
	}
}

// DeleteUser deletes a user character with super admin protection
func (s *Service) DeleteUser(ctx context.Context, characterID int) error {
	// Check if user exists before deletion
	user, err := s.repository.GetUser(ctx, characterID)
	if err != nil {
		if err.Error() == fmt.Sprintf("user not found for character ID %d", characterID) {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Check if groups service is available for super admin validation
	if s.groupService != nil {
		// Check if ANY character belonging to this user is a super admin (user-based check)
		isSuperAdmin, err := s.groupService.IsUserInGroup(ctx, user.UserID, "Super Administrator")
		if err != nil {
			return fmt.Errorf("failed to check super admin status: %w", err)
		}

		if isSuperAdmin {
			return fmt.Errorf("cannot delete super administrator character")
		}
	}

	// Remove character from all groups before deleting the user
	if s.groupService != nil {
		if err := s.groupService.RemoveCharacterFromAllGroups(ctx, int64(characterID)); err != nil {
			// Log the error but don't fail the deletion - this is cleanup
			fmt.Printf("Warning: Failed to cleanup group memberships for character %d: %v\n", characterID, err)
		}
	}

	// Delete the user
	return s.repository.DeleteUser(ctx, characterID)
}

// GetStatus returns the health status of the users module
func (s *Service) GetStatus(ctx context.Context) *dto.UsersStatusResponse {
	// Check database connectivity
	if err := s.repository.CheckHealth(ctx); err != nil {
		return &dto.UsersStatusResponse{
			Module:  "users",
			Status:  "unhealthy",
			Message: "Database connection failed: " + err.Error(),
		}
	}

	return &dto.UsersStatusResponse{
		Module: "users",
		Status: "healthy",
	}
}
