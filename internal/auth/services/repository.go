package services

import (
	"context"
	"time"

	"go-falcon/internal/auth/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// Repository handles database operations for auth module
type Repository struct {
	mongodb *database.MongoDB
}

// NewRepository creates a new auth repository
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb: mongodb,
	}
}

// CreateOrUpdateUserProfile creates or updates a user profile
func (r *Repository) CreateOrUpdateUserProfile(ctx context.Context, profile *models.UserProfile) (*models.UserProfile, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.repository.create_or_update_profile")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "create_or_update_profile"),
		attribute.Int("character_id", profile.CharacterID),
	)

	collection := r.mongodb.Collection("user_profiles")

	now := time.Now()
	profile.UpdatedAt = now

	// Check if this is a new character or an existing one
	filter := bson.M{"character_id": profile.CharacterID}
	var existingProfile models.UserProfile
	err := collection.FindOne(ctx, filter).Decode(&existingProfile)
	isNewCharacter := err == mongo.ErrNoDocuments

	// If this is a new character, assign position
	if isNewCharacter {
		position, err := r.assignPositionForNewCharacter(ctx, profile.UserID)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		profile.Position = position
	} else {
		// For existing characters, keep the existing position
		profile.Position = existingProfile.Position
	}

	// Prepare update document - exclude created_at from $set to avoid conflict
	updateFields := bson.M{
		"user_id":              profile.UserID,
		"character_id":         profile.CharacterID,
		"character_name":       profile.CharacterName,
		"character_owner_hash": profile.CharacterOwnerHash,
		"corporation_id":       profile.CorporationID,
		"corporation_name":     profile.CorporationName,
		"alliance_id":          profile.AllianceID,
		"alliance_name":        profile.AllianceName,
		"security_status":      profile.SecurityStatus,
		"birthday":             profile.Birthday,
		"scopes":               profile.Scopes,
		"access_token":         profile.AccessToken,
		"refresh_token":        profile.RefreshToken,
		"token_expiry":         profile.TokenExpiry,
		"last_login":           profile.LastLogin,
		"profile_updated":      profile.ProfileUpdated,
		"valid":                profile.Valid,
		"position":             profile.Position,
		"metadata":             profile.Metadata,
		"updated_at":           profile.UpdatedAt,
	}

	update := bson.M{
		"$set": updateFields,
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Retrieve the updated document
	var updatedProfile models.UserProfile
	err = collection.FindOne(ctx, filter).Decode(&updatedProfile)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return &updatedProfile, nil
}

// assignPositionForNewCharacter assigns the correct position for a new character
func (r *Repository) assignPositionForNewCharacter(ctx context.Context, userID string) (int, error) {
	collection := r.mongodb.Collection("user_profiles")

	// Find the highest position for this user
	filter := bson.M{"user_id": userID}
	opts := options.FindOne().SetSort(bson.M{"position": -1})

	var highestProfile models.UserProfile
	err := collection.FindOne(ctx, filter, opts).Decode(&highestProfile)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This is the first character for this user, position should be 0
			return 0, nil
		}
		return 0, err
	}

	// This is an additional character, position should be last position + 1
	return highestProfile.Position + 1, nil
}

// GetUserProfileByCharacterID retrieves a user profile by character ID
func (r *Repository) GetUserProfileByCharacterID(ctx context.Context, characterID int) (*models.UserProfile, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.repository.get_profile_by_character_id")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "get_profile_by_character_id"),
		attribute.Int("character_id", characterID),
	)

	collection := r.mongodb.Collection("user_profiles")

	var profile models.UserProfile
	err := collection.FindOne(ctx, bson.M{"character_id": characterID}).Decode(&profile)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Profile not found
		}
		span.RecordError(err)
		return nil, err
	}

	return &profile, nil
}

// GetUserProfileByUserID retrieves a user profile by user ID
func (r *Repository) GetUserProfileByUserID(ctx context.Context, userID string) (*models.UserProfile, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.repository.get_profile_by_user_id")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "get_profile_by_user_id"),
		attribute.String("user_id", userID),
	)

	collection := r.mongodb.Collection("user_profiles")

	var profile models.UserProfile
	err := collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&profile)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Profile not found
		}
		span.RecordError(err)
		return nil, err
	}

	return &profile, nil
}

// GetAllCharactersByUserID retrieves all user profiles for a given user ID
func (r *Repository) GetAllCharactersByUserID(ctx context.Context, userID string) ([]*models.UserProfile, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.repository.get_all_characters_by_user_id")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "get_all_characters_by_user_id"),
		attribute.String("user_id", userID),
	)

	collection := r.mongodb.Collection("user_profiles")

	// Find all profiles with this user_id
	cursor, err := collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var profiles []*models.UserProfile
	for cursor.Next(ctx) {
		var profile models.UserProfile
		if err := cursor.Decode(&profile); err != nil {
			span.RecordError(err)
			return nil, err
		}
		profiles = append(profiles, &profile)
	}

	if err := cursor.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	return profiles, nil
}

// GetExpiringTokens retrieves profiles with tokens expiring soon
func (r *Repository) GetExpiringTokens(ctx context.Context, beforeTime time.Time, limit int) ([]models.UserProfile, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.repository.get_expiring_tokens")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "get_expiring_tokens"),
		attribute.String("before_time", beforeTime.Format(time.RFC3339)),
		attribute.Int("limit", limit),
	)

	collection := r.mongodb.Collection("user_profiles")

	filter := bson.M{
		"token_expiry":  bson.M{"$lt": beforeTime},
		"refresh_token": bson.M{"$ne": ""},
	}

	opts := options.Find().SetLimit(int64(limit))
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var profiles []models.UserProfile
	if err := cursor.All(ctx, &profiles); err != nil {
		span.RecordError(err)
		return nil, err
	}

	return profiles, nil
}

// UpdateProfileTokens updates access and refresh tokens for a profile
func (r *Repository) UpdateProfileTokens(ctx context.Context, characterID int, accessToken, refreshToken string, expiresAt time.Time) error {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.repository.update_profile_tokens")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "update_profile_tokens"),
		attribute.Int("character_id", characterID),
	)

	collection := r.mongodb.Collection("user_profiles")

	filter := bson.M{"character_id": characterID}
	update := bson.M{
		"$set": bson.M{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_expiry":  expiresAt,
			"valid":         true,
			"updated_at":    time.Now(),
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

// InvalidateProfile marks a profile as invalid
func (r *Repository) InvalidateProfile(ctx context.Context, characterID int) error {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.repository.invalidate_profile")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "invalidate_profile"),
		attribute.Int("character_id", characterID),
	)

	collection := r.mongodb.Collection("user_profiles")

	filter := bson.M{"character_id": characterID}
	update := bson.M{
		"$set": bson.M{
			"valid":      false,
			"updated_at": time.Now(),
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

// StoreLoginState stores OAuth login state
func (r *Repository) StoreLoginState(ctx context.Context, state *models.EVELoginState) error {
	collection := r.mongodb.Collection("auth_states")

	state.CreatedAt = time.Now()
	state.ExpiresAt = state.CreatedAt.Add(15 * time.Minute) // States expire in 15 minutes

	_, err := collection.InsertOne(ctx, state)
	return err
}

// GetLoginState retrieves and validates OAuth login state
func (r *Repository) GetLoginState(ctx context.Context, state string) (*models.EVELoginState, error) {
	collection := r.mongodb.Collection("auth_states")

	var loginState models.EVELoginState
	err := collection.FindOne(ctx, bson.M{
		"state":      state,
		"expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&loginState)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // State not found or expired
		}
		return nil, err
	}

	return &loginState, nil
}

// CleanupExpiredStates removes expired OAuth states
func (r *Repository) CleanupExpiredStates(ctx context.Context) error {
	collection := r.mongodb.Collection("auth_states")

	filter := bson.M{"expires_at": bson.M{"$lt": time.Now()}}
	_, err := collection.DeleteMany(ctx, filter)
	return err
}

// CheckHealth verifies database connectivity
func (r *Repository) CheckHealth(ctx context.Context) error {
	// Perform a simple ping to check database connectivity
	return r.mongodb.Client.Ping(ctx, nil)
}
