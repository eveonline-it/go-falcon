package services

import (
	"context"
	"time"

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

// ListUsers retrieves users with pagination and filtering
func (s *Service) ListUsers(ctx context.Context, req dto.UserSearchRequest) (*dto.UserListResponse, error) {
	// Set defaults and validate
	req.SetDefaults()
	if err := dto.ValidateUserSearchRequest(&req); err != nil {
		return nil, err
	}

	return s.repository.ListUsers(ctx, req)
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

// Character Management Methods (Phase 2: Character Resolution System)

// GetUserWithCharacters retrieves a user with all their characters for middleware resolution
func (s *Service) GetUserWithCharacters(ctx context.Context, userID string) (*models.UserWithCharacters, error) {
	// Get full character data needed for middleware (not just summaries)
	fullCharacters, err := s.repository.GetFullCharactersForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	// Convert to the format expected by middleware
	userChars := make([]models.UserCharacter, len(fullCharacters))
	for i, char := range fullCharacters {
		// Handle potential nil LastLogin
		lastActive := time.Now() // Default to now if no last login
		if char.LastLogin != nil {
			lastActive = *char.LastLogin
		}
		
		userChars[i] = models.UserCharacter{
			CharacterID:   int64(char.CharacterID),
			Name:          char.CharacterName,
			CorporationID: int64(char.CorporationID),
			AllianceID:    int64(char.AllianceID),
			IsPrimary:     i == 0, // First character is considered primary
			AddedAt:       char.CreatedAt,
			LastActive:    lastActive,
		}
	}
	
	return &models.UserWithCharacters{
		ID:         userID,
		Characters: userChars,
		CreatedAt:  time.Now(), // We could track this separately if needed
		UpdatedAt:  time.Now(),
	}, nil
}

// AddCharacterToUser adds a new character to an existing user
func (s *Service) AddCharacterToUser(ctx context.Context, userID string, character *models.UserCharacter) error {
	return s.repository.AddCharacterToUser(ctx, userID, character)
}

// UpdateCharacterDetails updates corporation and alliance information for a character
func (s *Service) UpdateCharacterDetails(ctx context.Context, characterID int64, corporationID, allianceID int64) error {
	return s.repository.UpdateCharacterDetails(ctx, characterID, corporationID, allianceID)
}

// RemoveCharacterFromUser removes a character from a user's account
func (s *Service) RemoveCharacterFromUser(ctx context.Context, userID string, characterID int64) error {
	return s.repository.RemoveCharacterFromUser(ctx, userID, characterID)
}