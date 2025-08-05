package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"go-falcon/pkg/evegateway"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserProfile represents a user's profile with EVE character information
type UserProfile struct {
	CharacterID       int       `json:"character_id" bson:"character_id"`
	CharacterName     string    `json:"character_name" bson:"character_name"`
	CorporationID     int       `json:"corporation_id" bson:"corporation_id"`
	CorporationName   string    `json:"corporation_name" bson:"corporation_name"`
	AllianceID        int       `json:"alliance_id,omitempty" bson:"alliance_id,omitempty"`
	AllianceName      string    `json:"alliance_name,omitempty" bson:"alliance_name,omitempty"`
	SecurityStatus    float64   `json:"security_status" bson:"security_status"`
	Birthday          time.Time `json:"birthday" bson:"birthday"`
	Scopes            string    `json:"scopes" bson:"scopes"`
	LastLogin         time.Time `json:"last_login" bson:"last_login"`
	RefreshToken      string    `json:"-" bson:"refresh_token"` // Hidden from JSON
	CreatedAt         time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" bson:"updated_at"`
}

// CreateOrUpdateUserProfile creates or updates a user profile with EVE character data
func (m *Module) CreateOrUpdateUserProfile(ctx context.Context, charInfo *EVECharacterInfo, refreshToken string) (*UserProfile, error) {
	collection := m.MongoDB().Collection("user_profiles")
	
	// Create EVE ESI client to get additional character information
	client := evegateway.NewClient()
	
	// Get character public information
	charPublicInfo, err := client.GetCharacterInfo(ctx, charInfo.CharacterID)
	if err != nil {
		slog.Error("Failed to get character public info", 
			slog.Int("character_id", charInfo.CharacterID),
			slog.String("error", err.Error()))
		// Continue with basic info if ESI call fails
	}

	// Get corporation information
	var corpName string
	var corpID int
	if charPublicInfo != nil {
		if id, ok := charPublicInfo["corporation_id"].(float64); ok {
			corpID = int(id)
			if corpInfo, err := client.GetCorporationInfo(ctx, corpID); err == nil {
				if name, ok := corpInfo["name"].(string); ok {
					corpName = name
				}
			}
		}
	}

	// Get alliance information if character is in an alliance
	var allianceID int
	var allianceName string
	if charPublicInfo != nil {
		if id, ok := charPublicInfo["alliance_id"].(float64); ok {
			allianceID = int(id)
			if allianceInfo, err := client.GetAllianceInfo(ctx, allianceID); err == nil {
				if name, ok := allianceInfo["name"].(string); ok {
					allianceName = name
				}
			}
		}
	}

	now := time.Now()
	profile := &UserProfile{
		CharacterID:     charInfo.CharacterID,
		CharacterName:   charInfo.CharacterName,
		Scopes:          charInfo.Scopes,
		LastLogin:       now,
		RefreshToken:    refreshToken,
		UpdatedAt:       now,
	}

	// Add additional info if available
	if charPublicInfo != nil {
		profile.CorporationID = corpID
		profile.CorporationName = corpName
		profile.AllianceID = allianceID
		profile.AllianceName = allianceName
		
		// Extract security status and birthday if available
		if secStatus, ok := charPublicInfo["security_status"].(float64); ok {
			profile.SecurityStatus = secStatus
		}
		if birthdayStr, ok := charPublicInfo["birthday"].(string); ok {
			if birthday, err := time.Parse(time.RFC3339, birthdayStr); err == nil {
				profile.Birthday = birthday
			}
		}
	}

	// Try to update existing profile or create new one
	filter := bson.M{"character_id": charInfo.CharacterID}
	update := bson.M{
		"$set": bson.M{
			"character_name":   profile.CharacterName,
			"corporation_id":   profile.CorporationID,
			"corporation_name": profile.CorporationName,
			"alliance_id":      profile.AllianceID,
			"alliance_name":    profile.AllianceName,
			"security_status":  profile.SecurityStatus,
			"birthday":         profile.Birthday,
			"scopes":           profile.Scopes,
			"last_login":       profile.LastLogin,
			"refresh_token":    profile.RefreshToken,
			"updated_at":       profile.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user profile: %w", err)
	}

	// Fetch the updated document
	result := collection.FindOne(ctx, filter)
	
	if err := result.Decode(profile); err != nil {
		return nil, fmt.Errorf("failed to upsert user profile: %w", err)
	}

	slog.Info("User profile updated", 
		slog.Int("character_id", profile.CharacterID),
		slog.String("character_name", profile.CharacterName))

	return profile, nil
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
	tokenResp, err := m.eveSSOHandler.RefreshToken(ctx, profile.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh access token: %w", err)
	}

	// Verify the new token and get updated character info
	charInfo, err := m.eveSSOHandler.VerifyToken(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify refreshed token: %w", err)
	}

	// Update profile with fresh data
	return m.CreateOrUpdateUserProfile(ctx, charInfo, tokenResp.RefreshToken)
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
		"character_id":     profile.CharacterID,
		"character_name":   profile.CharacterName,
		"corporation_id":   profile.CorporationID,
		"corporation_name": profile.CorporationName,
		"alliance_id":      profile.AllianceID,
		"alliance_name":    profile.AllianceName,
		"security_status":  profile.SecurityStatus,
		"birthday":         profile.Birthday,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(publicProfile)
}