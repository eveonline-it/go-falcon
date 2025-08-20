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