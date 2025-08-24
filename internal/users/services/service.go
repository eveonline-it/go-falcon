package services

import (
	"context"
	"fmt"

	"go-falcon/internal/groups/services"
	"go-falcon/internal/users/dto"
	"go-falcon/internal/users/models"
	"go-falcon/pkg/database"
)

// Service provides business logic for user operations
type Service struct {
	repository   *Repository
	groupService *services.Service
}

// NewService creates a new service instance
func NewService(mongodb *database.MongoDB) *Service {
	return &Service{
		repository:   NewRepository(mongodb),
		groupService: nil, // Will be set after initialization
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
		// Check if the character is a super admin
		isSuperAdmin, err := s.groupService.IsCharacterInGroup(ctx, int64(characterID), "Super Administrator")
		if err != nil {
			return fmt.Errorf("failed to check super admin status: %w", err)
		}

		if isSuperAdmin {
			return fmt.Errorf("cannot delete super admin character")
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
