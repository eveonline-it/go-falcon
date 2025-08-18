package services

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"go-falcon/internal/users/dto"
	"go-falcon/internal/users/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// ListUsers retrieves users with pagination and filtering
func (r *Repository) ListUsers(ctx context.Context, req dto.UserSearchRequest) (*dto.UserListResponse, error) {
	collection := r.mongodb.Collection(models.User{}.CollectionName())
	
	// Build filter
	filter := bson.M{}
	
	// Search by character name or ID
	if req.Query != "" {
		// Try to parse as character ID first
		if characterID, err := strconv.Atoi(req.Query); err == nil {
			filter["character_id"] = characterID
		} else {
			// Search by character name (case-insensitive regex)
			filter["character_name"] = bson.M{
				"$regex": primitive.Regex{
					Pattern: req.Query,
					Options: "i",
				},
			}
		}
	}
	
	// Filter by enabled status
	if req.Enabled != nil {
		filter["enabled"] = *req.Enabled
	}
	
	// Filter by banned status
	if req.Banned != nil {
		filter["banned"] = *req.Banned
	}
	
	// Filter by invalid status
	if req.Invalid != nil {
		filter["invalid"] = *req.Invalid
	}
	
	// Filter by position
	if req.Position != nil {
		filter["position"] = *req.Position
	}

	// Build sort options
	sortOrder := 1
	if req.SortOrder == "desc" {
		sortOrder = -1
	}
	
	sortOptions := bson.D{{req.SortBy, sortOrder}}

	// Count total documents matching filter
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Calculate pagination
	skip := (req.Page - 1) * req.PageSize
	totalPages := int(math.Ceil(float64(total) / float64(req.PageSize)))

	// Find users with pagination
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize)).
		SetSort(sortOptions)

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	// Convert to response DTOs
	userResponses := make([]dto.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = dto.UserResponse{
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

	return &dto.UserListResponse{
		Users:      userResponses,
		Total:      int(total),
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateUser updates user status and administrative fields
func (r *Repository) UpdateUser(ctx context.Context, characterID int, req dto.UserUpdateRequest) (*models.User, error) {
	collection := r.mongodb.Collection(models.User{}.CollectionName())
	
	// Build update document
	update := bson.M{}
	if req.Enabled != nil {
		update["enabled"] = *req.Enabled
	}
	if req.Banned != nil {
		update["banned"] = *req.Banned
	}
	if req.Invalid != nil {
		update["invalid"] = *req.Invalid
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

// GetUserStats returns user statistics
func (r *Repository) GetUserStats(ctx context.Context) (*dto.UserStatsResponse, error) {
	collection := r.mongodb.Collection(models.User{}.CollectionName())

	// Use aggregation pipeline to get counts
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id": nil,
				"total_users": bson.M{"$sum": 1},
				"enabled_users": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$enabled", true}},
							1,
							0,
						},
					},
				},
				"disabled_users": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$enabled", false}},
							1,
							0,
						},
					},
				},
				"banned_users": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$banned", true}},
							1,
							0,
						},
					},
				},
				"invalid_users": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$invalid", true}},
							1,
							0,
						},
					},
				},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get user statistics: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode user statistics: %w", err)
	}

	stats := &dto.UserStatsResponse{}
	if len(results) > 0 {
		result := results[0]
		if val, ok := result["total_users"]; ok {
			if count, ok := val.(int32); ok {
				stats.TotalUsers = int(count)
			}
		}
		if val, ok := result["enabled_users"]; ok {
			if count, ok := val.(int32); ok {
				stats.EnabledUsers = int(count)
			}
		}
		if val, ok := result["disabled_users"]; ok {
			if count, ok := val.(int32); ok {
				stats.DisabledUsers = int(count)
			}
		}
		if val, ok := result["banned_users"]; ok {
			if count, ok := val.(int32); ok {
				stats.BannedUsers = int(count)
			}
		}
		if val, ok := result["invalid_users"]; ok {
			if count, ok := val.(int32); ok {
				stats.InvalidUsers = int(count)
			}
		}
	}

	return stats, nil
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
		"enabled":        1,
		"banned":         1,
		"position":       1,
		"last_login":     1,
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
			Enabled:       char.Enabled,
			Banned:        char.Banned,
			Position:      char.Position,
			LastLogin:     char.LastLogin,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return characters, nil
}

// Character Management Repository Methods (Phase 2: Character Resolution System)

// GetFullCharactersForUser retrieves complete character information for middleware resolution
func (r *Repository) GetFullCharactersForUser(ctx context.Context, userID string) ([]dto.FullCharacterResponse, error) {
	collection := r.mongodb.Collection(models.User{}.CollectionName())
	
	filter := bson.M{"user_id": userID}
	
	// Project all needed fields for middleware
	projection := bson.M{
		"character_id":     1,
		"character_name":   1,
		"user_id":          1,
		"corporation_id":   1,
		"corporation_name": 1,
		"alliance_id":      1,
		"alliance_name":    1,
		"enabled":          1,
		"banned":           1,
		"position":         1,
		"last_login":       1,
		"created_at":       1,
		"updated_at":       1,
	}
	
	findOptions := options.Find().
		SetProjection(projection).
		SetSort(bson.D{{"character_name", 1}})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to query characters: %w", err)
	}
	defer cursor.Close(ctx)

	var characters []dto.FullCharacterResponse
	for cursor.Next(ctx) {
		var char dto.FullCharacterResponse
		if err := cursor.Decode(&char); err != nil {
			return nil, fmt.Errorf("failed to decode character: %w", err)
		}
		characters = append(characters, char)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return characters, nil
}

// AddCharacterToUser adds a new character to an existing user (placeholder implementation)
func (r *Repository) AddCharacterToUser(ctx context.Context, userID string, character *models.UserCharacter) error {
	// This would typically involve creating a new user_profile document
	// For now, this is a placeholder that would need to be implemented based on
	// the specific requirements of how characters are added to users
	
	// In the current system, characters are added through the auth flow
	// This method could be used for admin operations or character transfers
	
	return fmt.Errorf("AddCharacterToUser not yet implemented - characters are added through auth flow")
}

// UpdateCharacterDetails updates corporation and alliance information for a character
func (r *Repository) UpdateCharacterDetails(ctx context.Context, characterID int64, corporationID, allianceID int64) error {
	collection := r.mongodb.Collection(models.User{}.CollectionName())
	
	filter := bson.M{"character_id": characterID}
	update := bson.M{
		"$set": bson.M{
			"corporation_id": corporationID,
			"alliance_id":    allianceID,
			"updated_at":     time.Now(),
		},
	}
	
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update character details: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("character not found: %d", characterID)
	}
	
	return nil
}

// RemoveCharacterFromUser removes a character from a user's account
func (r *Repository) RemoveCharacterFromUser(ctx context.Context, userID string, characterID int64) error {
	collection := r.mongodb.Collection(models.User{}.CollectionName())
	
	filter := bson.M{
		"user_id":      userID,
		"character_id": characterID,
	}
	
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to remove character from user: %w", err)
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("character not found for user: user_id=%s, character_id=%d", userID, characterID)
	}
	
	return nil
}