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
	fmt.Printf("[DEBUG] NewUserCharacterResolver: Creating new user character resolver with MongoDB connection\n")
	if mongodb == nil {
		fmt.Printf("[DEBUG] NewUserCharacterResolver: WARNING - MongoDB connection is nil!\n")
	} else {
		fmt.Printf("[DEBUG] NewUserCharacterResolver: MongoDB connection established successfully\n")
	}
	
	resolver := &UserCharacterResolverImpl{
		mongodb: mongodb,
	}
	
	fmt.Printf("[DEBUG] NewUserCharacterResolver: User character resolver created successfully\n")
	return resolver
}

// GetUserWithCharacters implements UserCharacterResolver interface
func (r *UserCharacterResolverImpl) GetUserWithCharacters(ctx context.Context, userID string) (*UserWithCharacters, error) {
	fmt.Printf("[DEBUG] ===== UserCharacterResolver.GetUserWithCharacters START =====\n")
	fmt.Printf("[DEBUG] UserCharacterResolver: Getting characters for user %s\n", userID)
	
	if r.mongodb == nil {
		fmt.Printf("[DEBUG] UserCharacterResolver: ERROR - MongoDB connection is nil!\n")
		return nil, fmt.Errorf("mongodb connection is nil")
	}
	
	// Use the auth UserProfile collection since it has corporation/alliance data
	collection := r.mongodb.Collection("user_profiles")
	fmt.Printf("[DEBUG] UserCharacterResolver: Using collection 'user_profiles'\n")
	
	filter := bson.M{"user_id": userID}
	fmt.Printf("[DEBUG] UserCharacterResolver: Query filter: %+v\n", filter)
	
	// Project fields we need for character resolution
	projection := bson.M{
		"character_id":     1,
		"character_name":   1,
		"user_id":          1,
		"corporation_id":   1,
		"alliance_id":      1,
		"valid":            1,
	}
	fmt.Printf("[DEBUG] UserCharacterResolver: Projection fields: %+v\n", projection)
	
	findOptions := options.Find().
		SetProjection(projection).
		SetSort(bson.D{{Key: "character_name", Value: 1}})
	fmt.Printf("[DEBUG] UserCharacterResolver: Find options configured with projection and sort\n")

	fmt.Printf("[DEBUG] UserCharacterResolver: Executing MongoDB query...\n")
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		fmt.Printf("[DEBUG] UserCharacterResolver: Database query failed: %v\n", err)
		return nil, fmt.Errorf("failed to find characters for user %s: %w", userID, err)
	}
	defer cursor.Close(ctx)
	fmt.Printf("[DEBUG] UserCharacterResolver: MongoDB query executed successfully\n")

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
	fmt.Printf("[DEBUG] UserCharacterResolver: Converting %d profiles to UserCharacter format\n", len(profiles))
	
	for i, profile := range profiles {
		fmt.Printf("[DEBUG] UserCharacterResolver: Processing character %d: %s (ID: %d, Corp: %d, Alliance: %d, Valid: %t)\n", 
			i, profile.CharacterName, profile.CharacterID, profile.CorporationID, profile.AllianceID, profile.Valid)
		
		character := UserCharacter{
			CharacterID:   int64(profile.CharacterID),
			Name:          profile.CharacterName,
			CorporationID: int64(profile.CorporationID),
			AllianceID:    int64(profile.AllianceID),
			IsPrimary:     i == 0, // First character is considered primary for now
		}
		
		characters = append(characters, character)
		fmt.Printf("[DEBUG] UserCharacterResolver: Character %d converted successfully\n", i)
	}

	result := &UserWithCharacters{
		ID:         userID,
		Characters: characters,
	}

	fmt.Printf("[DEBUG] UserCharacterResolver: Successfully resolved %d characters for user %s\n", len(characters), userID)
	fmt.Printf("[DEBUG] ===== UserCharacterResolver.GetUserWithCharacters END =====\n")
	return result, nil
}