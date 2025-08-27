package services

import (
	"context"
	"fmt"
	"time"

	"go-falcon/internal/users/dto"
	"go-falcon/internal/users/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles database operations for users
type Repository struct {
	mongodb *database.MongoDB
}

// NewRepository creates a new repository instance
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb: mongodb,
	}
}

// GetUser retrieves a user by character ID
func (r *Repository) GetUser(ctx context.Context, characterID int) (*models.User, error) {
	collection := r.mongodb.Collection(models.User{}.CollectionName())

	var user models.User
	filter := bson.M{"character_id": characterID}

	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found for character ID %d", characterID)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByUserID retrieves a user by user ID
func (r *Repository) GetUserByUserID(ctx context.Context, userID string) (*models.User, error) {
	collection := r.mongodb.Collection(models.User{}.CollectionName())

	var user models.User
	filter := bson.M{"user_id": userID}

	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found for user ID %s", userID)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// UpdateUser updates user status and administrative fields
func (r *Repository) UpdateUser(ctx context.Context, characterID int, req dto.UserUpdateRequest) (*models.User, error) {
	collection := r.mongodb.Collection(models.User{}.CollectionName())

	// Build update document
	update := bson.M{}
	if req.Banned != nil {
		update["banned"] = *req.Banned
	}
	if req.Position != nil {
		update["position"] = *req.Position
	}
	if req.Notes != nil {
		update["notes"] = *req.Notes
	}

	// Always update the updated_at timestamp
	update["updated_at"] = time.Now()

	// Perform update
	filter := bson.M{"character_id": characterID}
	updateDoc := bson.M{"$set": update}

	result, err := collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("user not found for character ID %d", characterID)
	}

	// Return updated user
	return r.GetUser(ctx, characterID)
}

// ListCharacters retrieves character summaries for a specific user ID
func (r *Repository) ListCharacters(ctx context.Context, userID string) ([]dto.CharacterSummaryResponse, error) {
	collection := r.mongodb.Collection(models.CharacterSummary{}.CollectionName())

	filter := bson.M{"user_id": userID}

	// Project only needed fields for character summary
	projection := bson.M{
		"character_id":   1,
		"character_name": 1,
		"user_id":        1,
		"banned":         1,
		"position":       1,
		"last_login":     1,
		"valid":          1,
	}

	findOptions := options.Find().
		SetProjection(projection).
		SetSort(bson.D{{"character_name", 1}})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find characters: %w", err)
	}
	defer cursor.Close(ctx)

	var characters []dto.CharacterSummaryResponse
	for cursor.Next(ctx) {
		var char models.CharacterSummary
		if err := cursor.Decode(&char); err != nil {
			return nil, fmt.Errorf("failed to decode character: %w", err)
		}

		characters = append(characters, dto.CharacterSummaryResponse{
			CharacterID:   char.CharacterID,
			CharacterName: char.CharacterName,
			UserID:        char.UserID,
			Banned:        char.Banned,
			Position:      char.Position,
			LastLogin:     char.LastLogin,
			Valid:         char.Valid,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return characters, nil
}

// ListUsers retrieves paginated and filtered users
func (r *Repository) ListUsers(ctx context.Context, input dto.UserListInput) (*dto.UserListResponse, error) {
	collection := r.mongodb.Collection(models.User{}.CollectionName())

	// Build filter
	filter := bson.M{}

	if input.Query != "" {
		// Search by character name or character ID
		query := input.Query
		filter["$or"] = []bson.M{
			{"character_name": bson.M{"$regex": query, "$options": "i"}},
		}

		// If query is numeric, also search by character_id
		if characterID := parseInt(query); characterID > 0 {
			filter["$or"] = append(filter["$or"].([]bson.M), bson.M{"character_id": characterID})
		}
	}

	if input.Banned == "true" {
		filter["banned"] = true
	} else if input.Banned == "false" {
		filter["banned"] = false
	}

	if input.Position > 0 {
		filter["position"] = input.Position
	}

	// Build sort options
	sortField := "created_at"
	if input.SortBy != "" {
		switch input.SortBy {
		case "character_name", "created_at", "last_login", "position":
			sortField = input.SortBy
		}
	}

	sortOrder := -1 // desc by default
	if input.SortOrder == "asc" {
		sortOrder = 1
	}

	// Calculate pagination
	skip := (input.Page - 1) * input.PageSize

	// Get total count
	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(input.PageSize)).
		SetSort(bson.D{{sortField, sortOrder}})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []dto.UserResponse
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}

		users = append(users, dto.UserResponse{
			CharacterID:   user.CharacterID,
			UserID:        user.UserID,
			Banned:        user.Banned,
			Scopes:        user.Scopes,
			Position:      user.Position,
			Notes:         user.Notes,
			CreatedAt:     user.CreatedAt,
			UpdatedAt:     user.UpdatedAt,
			LastLogin:     user.LastLogin,
			CharacterName: user.CharacterName,
			Valid:         user.Valid,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	// Calculate total pages
	totalPages := int((totalCount + int64(input.PageSize) - 1) / int64(input.PageSize))

	return &dto.UserListResponse{
		Users:      users,
		Total:      int(totalCount),
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: totalPages,
	}, nil
}

// parseInt safely converts string to int, returns 0 if invalid
func parseInt(s string) int {
	// Simple numeric check for character ID search
	var result int
	for _, char := range s {
		if char < '0' || char > '9' {
			return 0
		}
		result = result*10 + int(char-'0')
		if result > 2147483647 { // int32 max
			return 0
		}
	}
	return result
}

// DeleteUser deletes a user by character ID
func (r *Repository) DeleteUser(ctx context.Context, characterID int) error {
	collection := r.mongodb.Collection(models.User{}.CollectionName())

	filter := bson.M{"character_id": characterID}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user not found for character ID %d", characterID)
	}

	return nil
}

// UpdateCharacterPositions updates positions for multiple characters
func (r *Repository) UpdateCharacterPositions(ctx context.Context, characters []dto.CharacterReorderRequest) error {
	collection := r.mongodb.Collection(models.User{}.CollectionName())

	for _, char := range characters {
		filter := bson.M{"character_id": char.CharacterID}
		update := bson.M{
			"$set": bson.M{
				"position":   char.Position,
				"updated_at": time.Now(),
			},
		}

		_, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("failed to update position for character %d: %w", char.CharacterID, err)
		}
	}

	return nil
}

// RecalculatePositions recalculates positions for all characters of a user to be consecutive
func (r *Repository) RecalculatePositions(ctx context.Context, userID string) error {
	collection := r.mongodb.Collection(models.User{}.CollectionName())

	// Get all characters for the user, sorted by current position
	filter := bson.M{"user_id": userID}
	findOptions := options.Find().SetSort(bson.D{{"position", 1}})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return fmt.Errorf("failed to find user characters: %w", err)
	}
	defer cursor.Close(ctx)

	var characters []models.User
	if err := cursor.All(ctx, &characters); err != nil {
		return fmt.Errorf("failed to decode user characters: %w", err)
	}

	// Update each character with consecutive positions starting from 0
	for i, char := range characters {
		newPosition := i
		filter := bson.M{"character_id": char.CharacterID}
		update := bson.M{
			"$set": bson.M{
				"position":   newPosition,
				"updated_at": time.Now(),
			},
		}

		_, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("failed to update position for character %d: %w", char.CharacterID, err)
		}
	}

	return nil
}

// CheckHealth verifies database connectivity
func (r *Repository) CheckHealth(ctx context.Context) error {
	// Perform a simple ping to check database connectivity
	return r.mongodb.Client.Ping(ctx, nil)
}
