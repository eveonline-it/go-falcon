package models

import "time"

// UserProfile represents a user profile in the database
type UserProfile struct {
	UserID              string            `bson:"user_id" json:"user_id"`
	CharacterID         int               `bson:"character_id" json:"character_id"`
	CharacterName       string            `bson:"character_name" json:"character_name"`
	CharacterOwnerHash  string            `bson:"character_owner_hash" json:"character_owner_hash"`
	CorporationID       int               `bson:"corporation_id" json:"corporation_id,omitempty"`
	CorporationName     string            `bson:"corporation_name" json:"corporation_name,omitempty"`
	AllianceID          int               `bson:"alliance_id" json:"alliance_id,omitempty"`
	AllianceName        string            `bson:"alliance_name" json:"alliance_name,omitempty"`
	SecurityStatus      float64           `bson:"security_status" json:"security_status,omitempty"`
	Birthday            time.Time         `bson:"birthday" json:"birthday,omitempty"`
	Scopes              string            `bson:"scopes" json:"scopes"`
	AccessToken         string            `bson:"access_token" json:"-"`               // Hidden from JSON
	RefreshToken        string            `bson:"refresh_token" json:"-"`             // Hidden from JSON
	TokenExpiry         time.Time         `bson:"token_expiry" json:"token_expiry"`
	LastLogin           time.Time         `bson:"last_login" json:"last_login"`
	ProfileUpdated      time.Time         `bson:"profile_updated" json:"profile_updated"`
	Valid               bool              `bson:"valid" json:"valid"`
	IsSuperAdmin        bool              `bson:"is_super_admin" json:"is_super_admin"`
	Metadata            map[string]string `bson:"metadata" json:"metadata,omitempty"`
	CreatedAt           time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt           time.Time         `bson:"updated_at" json:"updated_at"`
}

// AuthenticatedUser represents an authenticated user in context
type AuthenticatedUser struct {
	UserID        string `json:"user_id"`
	CharacterID   int    `json:"character_id"`
	CharacterName string `json:"character_name"`
	Scopes        string `json:"scopes"`
}

// EVETokenResponse represents the response from EVE's OAuth token endpoint
type EVETokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// EVECharacterInfo represents character info from EVE's verify endpoint
type EVECharacterInfo struct {
	CharacterID        int    `json:"CharacterID"`
	CharacterName      string `json:"CharacterName"`
	ExpiresOn          string `json:"ExpiresOn"`
	Scopes             string `json:"Scopes"`
	TokenType          string `json:"TokenType"`
	CharacterOwnerHash string `json:"CharacterOwnerHash"`
	IntellectualProperty string `json:"IntellectualProperty"`
}

// JWKSResponse represents the JSON Web Key Set response from EVE
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// EVELoginState represents OAuth state information
type EVELoginState struct {
	State     string    `bson:"state" json:"state"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
	UserID    string    `bson:"user_id,omitempty" json:"user_id,omitempty"`
}

// ESICharacterInfo represents character information from ESI
type ESICharacterInfo struct {
	CharacterID     int       `json:"character_id"`
	Name            string    `json:"name"`
	CorporationID   int       `json:"corporation_id"`
	AllianceID      int       `json:"alliance_id,omitempty"`
	Birthday        time.Time `json:"birthday"`
	SecurityStatus  float64   `json:"security_status"`
	Description     string    `json:"description,omitempty"`
	Gender          string    `json:"gender"`
	RaceID          int       `json:"race_id"`
	BloodlineID     int       `json:"bloodline_id"`
	AncestryID      int       `json:"ancestry_id,omitempty"`
}

// ESICorporationInfo represents corporation information from ESI
type ESICorporationInfo struct {
	CorporationID   int    `json:"corporation_id"`
	Name            string `json:"name"`
	Ticker          string `json:"ticker"`
	AllianceID      int    `json:"alliance_id,omitempty"`
	Description     string `json:"description,omitempty"`
	URL             string `json:"url,omitempty"`
	TaxRate         float32 `json:"tax_rate"`
	MemberCount     int    `json:"member_count"`
	CreationDate    time.Time `json:"creation_date"`
}

// ESIAllianceInfo represents alliance information from ESI
type ESIAllianceInfo struct {
	AllianceID      int       `json:"alliance_id"`
	Name            string    `json:"name"`
	Ticker          string    `json:"ticker"`
	ExecutorCorp    int       `json:"executor_corporation_id,omitempty"`
	DateFounded     time.Time `json:"date_founded"`
	CreatorCorp     int       `json:"creator_corporation_id"`
	CreatorCharacter int      `json:"creator_id"`
}

// TokenRefreshResult represents the result of a token refresh operation
type TokenRefreshResult struct {
	Success      bool      `json:"success"`
	CharacterID  int       `json:"character_id"`
	Error        string    `json:"error,omitempty"`
	RefreshedAt  time.Time `json:"refreshed_at"`
}

// BatchRefreshStats represents statistics from batch token refresh
type BatchRefreshStats struct {
	TotalProcessed int                   `json:"total_processed"`
	Successful     int                   `json:"successful"`
	Failed         int                   `json:"failed"`
	Results        []TokenRefreshResult  `json:"results,omitempty"`
	ProcessedAt    time.Time             `json:"processed_at"`
}