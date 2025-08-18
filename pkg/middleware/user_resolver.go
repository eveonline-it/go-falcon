package middleware

import (
	"context"
	"fmt"
	"time"

	authModels "go-falcon/internal/auth/models"
	"go-falcon/pkg/database"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserCharacterResolverImpl implements UserCharacterResolver using MongoDB with Redis caching
type UserCharacterResolverImpl struct {
	mongodb *database.MongoDB
	redis   *database.Redis
}

// NewUserCharacterResolver creates a new user character resolver with optional Redis caching
func NewUserCharacterResolver(mongodb *database.MongoDB, redis ...*database.Redis) UserCharacterResolver {
	fmt.Printf("[DEBUG] NewUserCharacterResolver: Creating new user character resolver with MongoDB connection\n")
	if mongodb == nil {
		fmt.Printf("[DEBUG] NewUserCharacterResolver: WARNING - MongoDB connection is nil!\n")
	} else {
		fmt.Printf("[DEBUG] NewUserCharacterResolver: MongoDB connection established successfully\n")
	}
	
	var redisClient *database.Redis
	if len(redis) > 0 && redis[0] != nil {
		redisClient = redis[0]
		fmt.Printf("[DEBUG] NewUserCharacterResolver: Redis caching enabled\n")
	} else {
		fmt.Printf("[DEBUG] NewUserCharacterResolver: Redis caching disabled\n")
	}
	
	resolver := &UserCharacterResolverImpl{
		mongodb: mongodb,
		redis:   redisClient,
	}
	
	fmt.Printf("[DEBUG] NewUserCharacterResolver: User character resolver created successfully\n")
	return resolver
}

// GetUserWithCharacters implements UserCharacterResolver interface with Redis caching
func (r *UserCharacterResolverImpl) GetUserWithCharacters(ctx context.Context, userID string) (*UserWithCharacters, error) {
	fmt.Printf("[DEBUG] ===== UserCharacterResolver.GetUserWithCharacters START =====\n")
	fmt.Printf("[DEBUG] UserCharacterResolver: Getting characters for user %s\n", userID)
	
	if r.mongodb == nil {
		fmt.Printf("[DEBUG] UserCharacterResolver: ERROR - MongoDB connection is nil!\n")
		return nil, fmt.Errorf("mongodb connection is nil")
	}

	// Try Redis cache first if available
	cacheKey := fmt.Sprintf("user_characters:%s", userID)
	if r.redis != nil {
		fmt.Printf("[DEBUG] UserCharacterResolver: Checking Redis cache for key: %s\n", cacheKey)
		
		var cachedResult UserWithCharacters
		err := r.redis.GetJSON(ctx, cacheKey, &cachedResult)
		if err == nil {
			fmt.Printf("[DEBUG] UserCharacterResolver: Cache HIT - found %d characters in cache\n", len(cachedResult.Characters))
			fmt.Printf("[DEBUG] ===== UserCharacterResolver.GetUserWithCharacters END (from cache) =====\n")
			return &cachedResult, nil
		} else if err != redis.Nil {
			// Log cache error but continue with database lookup
			fmt.Printf("[DEBUG] UserCharacterResolver: Cache error (continuing with DB): %v\n", err)
		} else {
			fmt.Printf("[DEBUG] UserCharacterResolver: Cache MISS - fetching from database\n")
		}
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

	// Check MongoDB connection health before query
	fmt.Printf("[DEBUG] UserCharacterResolver: Checking MongoDB connection health...\n")
	err := r.mongodb.HealthCheck(ctx)
	if err != nil {
		fmt.Printf("[DEBUG] UserCharacterResolver: MongoDB health check failed: %v\n", err)
		return nil, fmt.Errorf("mongodb connection unhealthy: %w", err)
	}
	fmt.Printf("[DEBUG] UserCharacterResolver: MongoDB connection is healthy\n")

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

	// Cache the result in Redis if available
	if r.redis != nil {
		cacheExpiration := 15 * time.Minute // Cache for 15 minutes
		err := r.redis.SetJSON(ctx, cacheKey, result, cacheExpiration)
		if err != nil {
			fmt.Printf("[DEBUG] UserCharacterResolver: Failed to cache result: %v\n", err)
		} else {
			fmt.Printf("[DEBUG] UserCharacterResolver: Cached result for %d characters (TTL: %v)\n", len(characters), cacheExpiration)
		}
	}

	fmt.Printf("[DEBUG] UserCharacterResolver: Successfully resolved %d characters for user %s\n", len(characters), userID)
	fmt.Printf("[DEBUG] ===== UserCharacterResolver.GetUserWithCharacters END =====\n")
	return result, nil
}

// InvalidateUserCache removes cached character data for a specific user
func (r *UserCharacterResolverImpl) InvalidateUserCache(ctx context.Context, userID string) error {
	if r.redis == nil {
		fmt.Printf("[DEBUG] UserCharacterResolver: Redis not available, skipping cache invalidation for user %s\n", userID)
		return nil
	}

	cacheKey := fmt.Sprintf("user_characters:%s", userID)
	err := r.redis.Delete(ctx, cacheKey)
	if err != nil {
		fmt.Printf("[DEBUG] UserCharacterResolver: Failed to invalidate cache for user %s: %v\n", userID, err)
		return err
	}

	fmt.Printf("[DEBUG] UserCharacterResolver: Successfully invalidated cache for user %s\n", userID)
	return nil
}