package middleware

import (
	"context"
	"fmt"

	authModels "go-falcon/internal/auth/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserCharacterResolverImpl implements UserCharacterResolver using MongoDB
type UserCharacterResolverImpl struct {
	mongodb *database.MongoDB
}

// NewUserCharacterResolver creates a new user character resolver
func NewUserCharacterResolver(mongodb *database.MongoDB) UserCharacterResolver {
	return &UserCharacterResolverImpl{
		mongodb: mongodb,
	}
}

// GetUserWithCharacters implements UserCharacterResolver interface
func (r *UserCharacterResolverImpl) GetUserWithCharacters(ctx context.Context, userID string) (*UserWithCharacters, error) {
	fmt.Printf("[DEBUG] UserCharacterResolver: Getting characters for user %s\n", userID)
	// Use the auth UserProfile collection since it has corporation/alliance data
	collection := r.mongodb.Collection("user_profiles")
	
	filter := bson.M{"user_id": userID}
	
	// Project fields we need for character resolution
	projection := bson.M{
		"character_id":     1,
		"character_name":   1,
		"user_id":          1,
		"corporation_id":   1,
		"alliance_id":      1,
		"valid":            1,
	}
	
	findOptions := options.Find().
		SetProjection(projection).
		SetSort(bson.D{{Key: "character_name", Value: 1}})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		fmt.Printf("[DEBUG] UserCharacterResolver: Database query failed: %v\n", err)
		return nil, fmt.Errorf("failed to find characters for user %s: %w", userID, err)
	}
	defer cursor.Close(ctx)

	var profiles []authModels.UserProfile
	if err := cursor.All(ctx, &profiles); err != nil {
		fmt.Printf("[DEBUG] UserCharacterResolver: Failed to decode profiles: %v\n", err)
		return nil, fmt.Errorf("failed to decode character profiles: %w", err)
	}
	fmt.Printf("[DEBUG] UserCharacterResolver: Found %d profiles for user %s\n", len(profiles), userID)

	if len(profiles) == 0 {
		fmt.Printf("[DEBUG] UserCharacterResolver: No profiles found for user %s\n", userID)
		return &UserWithCharacters{
			ID:         userID,
			Characters: []UserCharacter{},
		}, nil
	}

	// Convert auth UserProfile to middleware UserCharacter format
	characters := make([]UserCharacter, 0, len(profiles))
	for i, profile := range profiles {
		fmt.Printf("[DEBUG] UserCharacterResolver: Processing character %d: %s (ID: %d)\n", i, profile.CharacterName, profile.CharacterID)
		characters = append(characters, UserCharacter{
			CharacterID:   int64(profile.CharacterID),
			Name:          profile.CharacterName,
			CorporationID: int64(profile.CorporationID),
			AllianceID:    int64(profile.AllianceID),
			IsPrimary:     i == 0, // First character is considered primary for now
		})
	}

	fmt.Printf("[DEBUG] UserCharacterResolver: Successfully resolved %d characters for user %s\n", len(characters), userID)
	return &UserWithCharacters{
		ID:         userID,
		Characters: characters,
	}, nil
}