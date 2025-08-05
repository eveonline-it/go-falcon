package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-falcon/pkg/config"

	"github.com/golang-jwt/jwt/v5"
)

const (
	EVEAuthURL      = "https://login.eveonline.com/v2/oauth/authorize"
	EVETokenURL     = "https://login.eveonline.com/v2/oauth/token"
	EVEVerifyURL    = "https://login.eveonline.com/oauth/verify"
	EVEJWKSEndpoint = "https://login.eveonline.com/oauth/jwks"
)

type EVETokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type EVECharacterInfo struct {
	CharacterID       int    `json:"CharacterID"`
	CharacterName     string `json:"CharacterName"`
	ExpiresOn         string `json:"ExpiresOn"`
	Scopes            string `json:"Scopes"`
	TokenType         string `json:"TokenType"`
	CharacterOwnerHash string `json:"CharacterOwnerHash"`
	IntellectualProperty string `json:"IntellectualProperty"`
}

type EVEAuthState struct {
	State     string
	CreatedAt time.Time
}

type EVESSOHandler struct {
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       string
	jwtSecret    []byte
	states       map[string]*EVEAuthState // In production, use Redis for this
}

func NewEVESSOHandler() *EVESSOHandler {
	return &EVESSOHandler{
		clientID:     config.GetEVEClientID(),
		clientSecret: config.GetEVEClientSecret(),
		redirectURI:  config.GetEVERedirectURI(),
		scopes:       config.GetEVEScopes(),
		jwtSecret:    []byte(config.GetJWTSecret()),
		states:       make(map[string]*EVEAuthState),
	}
}

// GenerateAuthURL creates the EVE Online SSO authorization URL
func (h *EVESSOHandler) GenerateAuthURL() (string, string, error) {
	state, err := h.generateState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state for validation (in production, use Redis with TTL)
	h.states[state] = &EVEAuthState{
		State:     state,
		CreatedAt: time.Now(),
	}

	params := url.Values{
		"response_type": {"code"},
		"redirect_uri":  {h.redirectURI},
		"client_id":     {h.clientID},
		"scope":         {h.scopes},
		"state":         {state},
	}

	authURL := fmt.Sprintf("%s?%s", EVEAuthURL, params.Encode())
	return authURL, state, nil
}

// ExchangeCodeForToken exchanges authorization code for access token
func (h *EVESSOHandler) ExchangeCodeForToken(ctx context.Context, code, state string) (*EVETokenResponse, error) {
	// Validate state
	if !h.validateState(state) {
		return nil, errors.New("invalid or expired state parameter")
	}

	// Prepare token request
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {h.redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", EVETokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "go-falcon/1.0.0 (EVE Online SSO integration)")
	
	// Basic authentication with client credentials
	auth := base64.StdEncoding.EncodeToString([]byte(h.clientID + ":" + h.clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp EVETokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Clean up state
	delete(h.states, state)
	
	return &tokenResp, nil
}

// VerifyToken verifies and extracts character information from access token
func (h *EVESSOHandler) VerifyToken(ctx context.Context, accessToken string) (*EVECharacterInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", EVEVerifyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create verify request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "go-falcon/1.0.0 (EVE Online SSO integration)")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token verification failed with status %d: %s", resp.StatusCode, string(body))
	}

	var charInfo EVECharacterInfo
	if err := json.NewDecoder(resp.Body).Decode(&charInfo); err != nil {
		return nil, fmt.Errorf("failed to decode character info: %w", err)
	}

	return &charInfo, nil
}

// GenerateJWT creates a JWT token for the authenticated user
func (h *EVESSOHandler) GenerateJWT(charInfo *EVECharacterInfo) (string, error) {
	claims := jwt.MapClaims{
		"character_id":   charInfo.CharacterID,
		"character_name": charInfo.CharacterName,
		"scopes":         charInfo.Scopes,
		"iss":           "go-falcon",
		"exp":           time.Now().Add(24 * time.Hour).Unix(),
		"iat":           time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// ValidateJWT validates a JWT token and returns the claims
func (h *EVESSOHandler) ValidateJWT(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return h.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken refreshes an access token using refresh token
func (h *EVESSOHandler) RefreshToken(ctx context.Context, refreshToken string) (*EVETokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", EVETokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "go-falcon/1.0.0 (EVE Online SSO integration)")
	
	auth := base64.StdEncoding.EncodeToString([]byte(h.clientID + ":" + h.clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp EVETokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	return &tokenResp, nil
}

// generateState creates a cryptographically secure random state parameter
func (h *EVESSOHandler) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// validateState validates the state parameter and cleans up expired states
func (h *EVESSOHandler) validateState(state string) bool {
	authState, exists := h.states[state]
	if !exists {
		return false
	}

	// State expires after 15 minutes
	if time.Since(authState.CreatedAt) > 15*time.Minute {
		delete(h.states, state)
		return false
	}

	return true
}

// CleanupExpiredStates removes expired state entries (should be called periodically)
func (h *EVESSOHandler) CleanupExpiredStates() {
	now := time.Now()
	for state, authState := range h.states {
		if now.Sub(authState.CreatedAt) > 15*time.Minute {
			delete(h.states, state)
			slog.Debug("Cleaned up expired auth state", slog.String("state", state))
		}
	}
}