package services

import (
	"context"

	"go-falcon/internal/users/dto"
	"go-falcon/internal/users/models"
	"go-falcon/pkg/database"
)

// Service provides business logic for user operations
type Service struct {
	repository *Repository
}

// NewService creates a new service instance
func NewService(mongodb *database.MongoDB) *Service {
	return &Service{
		repository: NewRepository(mongodb),
	}
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