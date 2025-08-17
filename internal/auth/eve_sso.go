package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go-falcon/pkg/config"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	EVEAuthURL      = "https://login.eveonline.com/v2/oauth/authorize"
	EVETokenURL     = "https://login.eveonline.com/v2/oauth/token"
	EVEVerifyURL    = "https://login.eveonline.com/oauth/verify"
	EVEJWKSEndpoint = "https://login.eveonline.com/oauth/jwks"
	EVEIssuer       = "https://login.eveonline.com"  // Full URL as used by EVE
	EVEAudience     = "EVE Online"
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

// JWKS structures for JWT validation
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kid string `json:"kid"` // Key ID
	Kty string `json:"kty"` // Key Type
	Use string `json:"use"` // Usage
	N   string `json:"n"`   // Modulus
	E   string `json:"e"`   // Exponent
	Alg string `json:"alg"` // Algorithm
}

// JWKS cache
type JWKSCache struct {
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
	mu        sync.RWMutex
}

type EVESSOHandler struct {
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       string
	jwtSecret    []byte
	states       map[string]*EVEAuthState // In production, use Redis for this
	jwksCache    *JWKSCache
}

func NewEVESSOHandler() *EVESSOHandler {
	return &EVESSOHandler{
		clientID:     config.GetEVEClientID(),
		clientSecret: config.GetEVEClientSecret(),
		redirectURI:  config.GetEVERedirectURI(),
		scopes:       config.GetEVEScopes(),
		jwtSecret:    []byte(config.GetJWTSecret()),
		states:       make(map[string]*EVEAuthState),
		jwksCache: &JWKSCache{
			keys: make(map[string]*rsa.PublicKey),
		},
	}
}

// GenerateAuthURL creates the EVE Online SSO authorization URL
func (h *EVESSOHandler) GenerateAuthURL() (string, string, error) {
	return h.GenerateAuthURLWithScopes(h.scopes)
}

// GenerateAuthURLWithScopes creates the EVE Online SSO authorization URL with custom scopes
func (h *EVESSOHandler) GenerateAuthURLWithScopes(scopes string) (string, string, error) {
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
		"state":         {state},
	}

	// Only add scope parameter if scopes are provided
	if scopes != "" {
		params.Set("scope", scopes)
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

// VerifyToken verifies and extracts character information from JWT access token
func (h *EVESSOHandler) VerifyToken(ctx context.Context, accessToken string) (*EVECharacterInfo, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.eve.verify_token")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "verify_jwt_token"),
	)

	// Parse JWT token without verification first to get the header
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			span.SetAttributes(attribute.String("error.type", "missing_key_id"))
			return nil, errors.New("missing key ID in token header")
		}

		span.SetAttributes(attribute.String("jwt.key_id", kid))

		// Get public key from JWKS
		publicKey, err := h.getPublicKey(ctx, kid)
		if err != nil {
			span.SetAttributes(attribute.String("error.type", "public_key_fetch_failed"))
			return nil, fmt.Errorf("failed to get public key: %w", err)
		}

		span.SetAttributes(attribute.Bool("jwt.public_key_found", true))
		return publicKey, nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse/verify JWT token")
		return nil, fmt.Errorf("failed to parse/verify JWT token: %w", err)
	}

	if !token.Valid {
		span.SetStatus(codes.Error, "Invalid JWT token")
		span.SetAttributes(attribute.String("error.type", "invalid_token"))
		return nil, errors.New("invalid JWT token")
	}

	span.SetAttributes(attribute.Bool("jwt.signature_valid", true))

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("failed to extract claims from token")
	}

	// Validate issuer
	if iss, ok := claims["iss"].(string); !ok || iss != EVEIssuer {
		span.SetStatus(codes.Error, "Invalid issuer")
		span.SetAttributes(
			attribute.String("error.type", "invalid_issuer"),
			attribute.String("expected_issuer", EVEIssuer),
			attribute.String("actual_issuer", fmt.Sprintf("%v", claims["iss"])),
		)
		return nil, fmt.Errorf("invalid issuer: expected %s, got %v", EVEIssuer, claims["iss"])
	}

	span.SetAttributes(attribute.Bool("jwt.issuer_valid", true))

	// Validate audience - should contain "EVE Online" and optionally client ID
	if aud, ok := claims["aud"].([]interface{}); ok {
		validAudience := false
		for _, a := range aud {
			if audStr, ok := a.(string); ok && (audStr == EVEAudience || audStr == h.clientID) {
				validAudience = true
				break
			}
		}
		if !validAudience {
			return nil, fmt.Errorf("invalid audience: expected %s or %s in %v", EVEAudience, h.clientID, aud)
		}
	} else if audStr, ok := claims["aud"].(string); ok {
		if audStr != EVEAudience && audStr != h.clientID {
			return nil, fmt.Errorf("invalid audience: expected %s or %s, got %s", EVEAudience, h.clientID, audStr)
		}
	} else {
		return nil, errors.New("missing or invalid audience claim")
	}

	// Extract character information from claims
	charInfo := &EVECharacterInfo{}

	if sub, ok := claims["sub"].(string); ok {
		// Parse character ID from subject (format: "CHARACTER:EVE:123456")
		parts := strings.Split(sub, ":")
		if len(parts) >= 3 {
			if charID, err := parseInt(parts[2]); err == nil {
				charInfo.CharacterID = charID
			}
		}
	}

	if name, ok := claims["name"].(string); ok {
		charInfo.CharacterName = name
	}

	if scp, ok := claims["scp"].([]interface{}); ok {
		scopes := make([]string, 0, len(scp))
		for _, s := range scp {
			if scope, ok := s.(string); ok {
				scopes = append(scopes, scope)
			}
		}
		charInfo.Scopes = strings.Join(scopes, " ")
	} else if scopeStr, ok := claims["scp"].(string); ok {
		charInfo.Scopes = scopeStr
	}

	if exp, ok := claims["exp"].(float64); ok {
		charInfo.ExpiresOn = time.Unix(int64(exp), 0).Format(time.RFC3339)
	}

	if azp, ok := claims["azp"].(string); ok {
		charInfo.CharacterOwnerHash = azp
	}

	charInfo.TokenType = "Bearer"
	charInfo.IntellectualProperty = "EVE"

	// Validate that we have required fields
	if charInfo.CharacterID == 0 || charInfo.CharacterName == "" {
		span.SetStatus(codes.Error, "Missing required character information")
		span.SetAttributes(attribute.String("error.type", "missing_character_info"))
		return nil, errors.New("missing required character information in token")
	}

	span.SetAttributes(
		attribute.Bool("jwt.validation_success", true),
		attribute.Int("eve.character_id", charInfo.CharacterID),
		attribute.String("eve.character_name", charInfo.CharacterName),
		attribute.String("eve.scopes", charInfo.Scopes),
	)

	slog.Info("JWT token validated successfully", 
		slog.Int("character_id", charInfo.CharacterID),
		slog.String("character_name", charInfo.CharacterName))

	return charInfo, nil
}

// Helper function to parse integer safely
func parseInt(s string) (int, error) {
	var i int
	if n, err := fmt.Sscanf(s, "%d", &i); err != nil || n != 1 {
		return 0, fmt.Errorf("failed to parse integer: %s", s)
	}
	return i, nil
}

// GenerateJWT creates a JWT token for the authenticated user
func (h *EVESSOHandler) GenerateJWT(charInfo *EVECharacterInfo, userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":        userID,
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

// fetchJWKS fetches and caches JWKS from EVE Online
func (h *EVESSOHandler) fetchJWKS(ctx context.Context) error {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.eve.fetch_jwks")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "fetch_jwks"),
		attribute.String("jwks_endpoint", EVEJWKSEndpoint),
	)

	h.jwksCache.mu.Lock()
	defer h.jwksCache.mu.Unlock()

	// Check if cache is still valid (cache for 1 hour)
	if time.Now().Before(h.jwksCache.expiresAt) && len(h.jwksCache.keys) > 0 {
		span.SetAttributes(
			attribute.Bool("jwks.cache_hit", true),
			attribute.Int("jwks.cached_keys", len(h.jwksCache.keys)),
		)
		return nil
	}

	span.SetAttributes(attribute.Bool("jwks.cache_hit", false))
	slog.Info("Fetching JWKS from EVE Online", slog.String("endpoint", EVEJWKSEndpoint))

	req, err := http.NewRequestWithContext(ctx, "GET", EVEJWKSEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create JWKS request: %w", err)
	}

	req.Header.Set("User-Agent", "go-falcon/1.0.0 (EVE Online SSO integration)")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("JWKS fetch failed with status %d: %s", resp.StatusCode, string(body))
	}

	var jwksResp JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwksResp); err != nil {
		return fmt.Errorf("failed to decode JWKS response: %w", err)
	}

	// Convert JWKs to RSA public keys
	newKeys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwksResp.Keys {
		if jwk.Kty != "RSA" {
			continue // Skip non-RSA keys
		}

		pubKey, err := h.jwkToRSAPublicKey(jwk)
		if err != nil {
			slog.Warn("Failed to convert JWK to RSA public key", 
				slog.String("kid", jwk.Kid), 
				slog.String("error", err.Error()))
			continue
		}

		newKeys[jwk.Kid] = pubKey
	}

	if len(newKeys) == 0 {
		span.SetStatus(codes.Error, "No valid RSA keys found")
		span.SetAttributes(attribute.String("error.type", "no_valid_keys"))
		return errors.New("no valid RSA keys found in JWKS")
	}

	h.jwksCache.keys = newKeys
	h.jwksCache.expiresAt = time.Now().Add(1 * time.Hour) // Cache for 1 hour

	span.SetAttributes(
		attribute.Bool("jwks.fetch_success", true),
		attribute.Int("jwks.key_count", len(newKeys)),
		attribute.String("jwks.cache_expires", h.jwksCache.expiresAt.Format(time.RFC3339)),
	)

	slog.Info("JWKS cached successfully", slog.Int("key_count", len(newKeys)))
	return nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key
func (h *EVESSOHandler) jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode base64url encoded modulus and exponent
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	// Create RSA public key
	pubKey := &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}

	return pubKey, nil
}

// getPublicKey retrieves a public key by kid from the cache
func (h *EVESSOHandler) getPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Ensure JWKS is fetched and cached
	if err := h.fetchJWKS(ctx); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	h.jwksCache.mu.RLock()
	defer h.jwksCache.mu.RUnlock()

	key, exists := h.jwksCache.keys[kid]
	if !exists {
		return nil, fmt.Errorf("key with id %s not found in JWKS", kid)
	}

	return key, nil
}