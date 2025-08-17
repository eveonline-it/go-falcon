package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// UserProfile represents a user's profile with EVE character information
type UserProfile struct {
	CharacterID   int       `json:"character_id" bson:"character_id"`
	CharacterName string    `json:"character_name" bson:"character_name"`
	Scopes        string    `json:"scopes" bson:"scopes"`
	UserID        string    `json:"user_id" bson:"user_id"`                         // UUID linking characters to user
	Valid         bool      `json:"valid" bson:"valid"`                           // Character profile validity status
	AccessToken   string    `json:"-" bson:"access_token"`                        // Hidden from JSON for security
	RefreshToken  string    `json:"-" bson:"refresh_token"`                       // Hidden from JSON for security
	CreatedAt     time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" bson:"updated_at"`
}

// CreateOrUpdateUserProfile creates or updates a user profile with EVE character data
func (m *Module) CreateOrUpdateUserProfile(ctx context.Context, charInfo *EVECharacterInfo, userID, accessToken, refreshToken string) (*UserProfile, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.profile.create_or_update")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "create_or_update_profile"),
		attribute.Int("eve.character_id", charInfo.CharacterID),
		attribute.String("eve.character_name", charInfo.CharacterName),
		attribute.String("auth.user_id", userID),
	)

	collection := m.MongoDB().Collection("user_profiles")
	
	slog.Info("CreateOrUpdateUserProfile called", 
		slog.Int("character_id", charInfo.CharacterID),
		slog.String("character_name", charInfo.CharacterName),
		slog.String("user_id", userID))
	
	now := time.Now()
	
	// Check if profile already exists to determine if this is creation or update
	existingProfile, err := m.GetUserProfile(ctx, charInfo.CharacterID)
	isCreate := err != nil // If error, profile doesn't exist, so this is a create
	
	profile := &UserProfile{
		CharacterID:   charInfo.CharacterID,
		CharacterName: charInfo.CharacterName,
		Scopes:        charInfo.Scopes,
		UserID:        userID,
		Valid:         true, // Set valid as true as specified in docs
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		UpdatedAt:     now,
	}
	
	if isCreate {
		profile.CreatedAt = now
	} else {
		// Preserve original creation date for updates
		profile.CreatedAt = existingProfile.CreatedAt
	}

	// Replace the entire document to ensure all fields are present
	filter := bson.M{"character_id": charInfo.CharacterID}
	
	slog.Info("MongoDB replace operation", 
		slog.Any("filter", filter),
		slog.String("character_name", profile.CharacterName),
		slog.Int("character_id", profile.CharacterID))

	opts := options.Replace().SetUpsert(true)
	_, err = collection.ReplaceOne(ctx, filter, profile, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to upsert user profile")
		span.SetAttributes(attribute.String("error.type", "database_upsert_failed"))
		return nil, fmt.Errorf("failed to upsert user profile: %w", err)
	}

	span.SetAttributes(attribute.Bool("database.upsert_success", true))

	// Fetch the updated document
	var updatedProfile UserProfile
	if err := collection.FindOne(ctx, filter).Decode(&updatedProfile); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to decode user profile")
		span.SetAttributes(attribute.String("error.type", "database_decode_failed"))
		return nil, fmt.Errorf("failed to decode user profile after upsert: %w", err)
	}

	span.SetAttributes(
		attribute.Bool("database.fetch_success", true),
		attribute.Bool("profile.created", true),
		attribute.Bool("profile.valid", updatedProfile.Valid),
	)

	slog.Info("User profile updated", 
		slog.Int("character_id", updatedProfile.CharacterID),
		slog.String("character_name", updatedProfile.CharacterName),
		slog.String("user_id", updatedProfile.UserID))

	// Automatically assign user to appropriate groups if this is a new character
	if isCreate && m.groupsModule != nil {
		if err := m.assignUserToGroups(ctx, &updatedProfile, charInfo); err != nil {
			// Log error but don't fail the profile creation
			slog.Warn("Failed to assign user to default groups", 
				slog.String("error", err.Error()),
				slog.Int("character_id", updatedProfile.CharacterID))
		}
	}

	return &updatedProfile, nil
}

// GetUserProfile retrieves a user profile by character ID
func (m *Module) GetUserProfile(ctx context.Context, characterID int) (*UserProfile, error) {
	collection := m.MongoDB().Collection("user_profiles")
	
	var profile UserProfile
	filter := bson.M{"character_id": characterID}
	
	err := collection.FindOne(ctx, filter).Decode(&profile)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user profile not found for character ID %d", characterID)
		}
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	return &profile, nil
}

// GetAllUserCharacters retrieves all characters for a given user_id
func (m *Module) GetAllUserCharacters(ctx context.Context, userID string) ([]UserProfile, error) {
	collection := m.MongoDB().Collection("user_profiles")
	
	var characters []UserProfile
	filter := bson.M{"user_id": userID}
	
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find characters for user_id %s: %w", userID, err)
	}
	defer cursor.Close(ctx)
	
	err = cursor.All(ctx, &characters)
	if err != nil {
		return nil, fmt.Errorf("failed to decode characters for user_id %s: %w", userID, err)
	}
	
	return characters, nil
}

// RefreshUserProfile updates character information from EVE ESI
func (m *Module) RefreshUserProfile(ctx context.Context, characterID int) (*UserProfile, error) {
	profile, err := m.GetUserProfile(ctx, characterID)
	if err != nil {
		return nil, err
	}

	// Use stored refresh token to get new access token
	if profile.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available for character %d", characterID)
	}

	// Get EVE SSO handler from the module
	tokenResp, err := m.eveSSOHandler.RefreshToken(ctx, profile.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh access token: %w", err)
	}

	// Verify the new token and get updated character info
	charInfo, err := m.eveSSOHandler.VerifyToken(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify refreshed token: %w", err)
	}

	// Update profile with fresh data and new tokens
	return m.CreateOrUpdateUserProfile(ctx, charInfo, profile.UserID, tokenResp.AccessToken, tokenResp.RefreshToken)
}

// RefreshExpiringTokens refreshes tokens for users with expiring access tokens
func (m *Module) RefreshExpiringTokens(ctx context.Context, batchSize int) (successCount, failureCount int, err error) {
	// Find users with tokens expiring within the next hour
	expirationThreshold := time.Now().Add(1 * time.Hour)
	
	// Create aggregation pipeline to find users needing token refresh
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"refresh_token": bson.M{"$ne": ""},
				"$or": []bson.M{
					{"token_expires_at": bson.M{"$lt": expirationThreshold}},
					{"token_expires_at": bson.M{"$exists": false}}, // Handle missing expiration
				},
			},
		},
		{"$limit": batchSize},
		{
			"$project": bson.M{
				"character_id":  1,
				"refresh_token": 1,
				"user_id":       1,
			},
		},
	}

	collection := m.MongoDB().Collection("user_profiles")
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query users for token refresh: %w", err)
	}
	defer cursor.Close(ctx)

	var users []struct {
		CharacterID  int    `bson:"character_id"`
		RefreshToken string `bson:"refresh_token"`
		UserID       string `bson:"user_id"`
	}

	if err := cursor.All(ctx, &users); err != nil {
		return 0, 0, fmt.Errorf("failed to decode users: %w", err)
	}

	slog.Info("Found users for token refresh", 
		slog.Int("count", len(users)),
		slog.Int("batch_size", batchSize))

	// Refresh tokens for each user
	for _, user := range users {
		select {
		case <-ctx.Done():
			return successCount, failureCount, ctx.Err()
		default:
		}

		// Use existing RefreshToken method
		tokenResp, err := m.eveSSOHandler.RefreshToken(ctx, user.RefreshToken)
		if err != nil {
			failureCount++
			slog.Warn("Token refresh failed", 
				slog.Int("character_id", user.CharacterID),
				slog.String("error", err.Error()))
			continue
		}

		// Verify the new token to get character info
		charInfo, err := m.eveSSOHandler.VerifyToken(ctx, tokenResp.AccessToken)
		if err != nil {
			failureCount++
			slog.Warn("Token verification failed after refresh", 
				slog.Int("character_id", user.CharacterID),
				slog.String("error", err.Error()))
			continue
		}

		// Update the user profile with new tokens
		_, err = m.CreateOrUpdateUserProfile(ctx, charInfo, user.UserID, tokenResp.AccessToken, tokenResp.RefreshToken)
		if err != nil {
			failureCount++
			slog.Warn("Failed to update user profile after token refresh", 
				slog.Int("character_id", user.CharacterID),
				slog.String("error", err.Error()))
			continue
		}

		successCount++
		slog.Debug("Token refreshed successfully", 
			slog.Int("character_id", user.CharacterID))
	}

	slog.Info("Token refresh batch completed", 
		slog.Int("success_count", successCount),
		slog.Int("failure_count", failureCount),
		slog.Int("total_users", len(users)))

	return successCount, failureCount, nil
}

// assignUserToGroups assigns a user to appropriate default groups based on their profile
func (m *Module) assignUserToGroups(ctx context.Context, profile *UserProfile, charInfo *EVECharacterInfo) error {
	if m.groupsModule == nil {
		return fmt.Errorf("groups module not available")
	}

	// Use reflection to call the groups module method
	// This avoids circular imports between auth and groups modules
	groupsModuleValue := reflect.ValueOf(m.groupsModule)
	
	// Check if user has meaningful scopes (not empty or just publicData)
	hasScopes := charInfo.Scopes != "" && charInfo.Scopes != "publicData"
	
	var method reflect.Value
	var methodName string
	
	if hasScopes {
		// User has scopes, assign to full groups
		method = groupsModuleValue.MethodByName("AssignUserToDefaultGroups")
		methodName = "AssignUserToDefaultGroups"
	} else {
		// User has no meaningful scopes, assign to basic login group
		method = groupsModuleValue.MethodByName("AssignUserToBasicLoginGroup")
		methodName = "AssignUserToBasicLoginGroup"
	}
	
	if !method.IsValid() {
		return fmt.Errorf("%s method not found in groups module", methodName)
	}

	var args []reflect.Value
	if hasScopes {
		// For now, we'll pass nil for corporation and alliance IDs since they're not in charInfo
		// The groups module can fetch this information via ESI if needed
		var corporationID, allianceID *int
		
		// Prepare arguments for AssignUserToDefaultGroups
		args = []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(profile.CharacterID),
			reflect.ValueOf(corporationID),
			reflect.ValueOf(allianceID),
		}
	} else {
		// Prepare arguments for AssignUserToBasicLoginGroup
		args = []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(profile.CharacterID),
		}
	}

	// Call the method
	results := method.Call(args)
	
	// Check if there was an error
	if len(results) > 0 && !results[0].IsNil() {
		err := results[0].Interface().(error)
		return fmt.Errorf("groups assignment failed: %w", err)
	}

	slog.Info("Successfully assigned user to groups",
		slog.Int("character_id", profile.CharacterID),
		slog.String("character_name", profile.CharacterName),
		slog.String("scopes", charInfo.Scopes),
		slog.Bool("has_scopes", hasScopes),
		slog.String("group_type", func() string {
			if hasScopes {
				return "full"
			}
			return "basic_login"
		}()))

	return nil
}

// Profile handler endpoints

func (m *Module) profileHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := GetAuthenticatedUser(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	profile, err := m.GetUserProfile(r.Context(), user.CharacterID)
	if err != nil {
		slog.Error("Failed to get user profile", 
			slog.Int("character_id", user.CharacterID),
			slog.String("error", err.Error()))
		http.Error(w, "Failed to get profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (m *Module) refreshProfileHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := GetAuthenticatedUser(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	profile, err := m.RefreshUserProfile(r.Context(), user.CharacterID)
	if err != nil {
		slog.Error("Failed to refresh user profile", 
			slog.Int("character_id", user.CharacterID),
			slog.String("error", err.Error()))
		http.Error(w, "Failed to refresh profile", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"profile": profile,
		"message": "Profile refreshed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *Module) publicProfileHandler(w http.ResponseWriter, r *http.Request) {
	// Get character ID from URL path
	characterIDStr := r.URL.Query().Get("character_id")
	if characterIDStr == "" {
		http.Error(w, "Missing character_id parameter", http.StatusBadRequest)
		return
	}

	characterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		http.Error(w, "Invalid character_id parameter", http.StatusBadRequest)
		return
	}

	profile, err := m.GetUserProfile(r.Context(), characterID)
	if err != nil {
		slog.Warn("Profile not found", 
			slog.Int("character_id", characterID),
			slog.String("error", err.Error()))
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	// Return only public information
	publicProfile := map[string]interface{}{
		"character_id":   profile.CharacterID,
		"character_name": profile.CharacterName,
		"scopes":         profile.Scopes,
		"user_id":        profile.UserID,
		"valid":          profile.Valid,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(publicProfile)
}