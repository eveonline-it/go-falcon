package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// UserProfile represents a user's profile with EVE character information
type UserProfile struct {
	CharacterID   int    `json:"character_id" bson:"character_id"`
	CharacterName string `json:"character_name" bson:"character_name"`
	Scopes        string `json:"scopes" bson:"scopes"`
	UserID        string `json:"user_id" bson:"user_id"`                         // UUID linking characters to user
	Valid         bool   `json:"valid" bson:"valid"`                           // Character profile validity status
	AccessToken   string `json:"-" bson:"access_token"`                        // Hidden from JSON for security
	RefreshToken  string `json:"-" bson:"refresh_token"`                       // Hidden from JSON for security
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
	
	profile := &UserProfile{
		CharacterID:   charInfo.CharacterID,
		CharacterName: charInfo.CharacterName,
		Scopes:        charInfo.Scopes,
		UserID:        userID,
		Valid:         true, // Set valid as true as specified in docs
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
	}

	// Replace the entire document to ensure all fields are present
	filter := bson.M{"character_id": charInfo.CharacterID}
	
	slog.Info("MongoDB replace operation", 
		slog.Any("filter", filter),
		slog.Any("profile", profile))

	opts := options.Replace().SetUpsert(true)
	_, err := collection.ReplaceOne(ctx, filter, profile, opts)
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