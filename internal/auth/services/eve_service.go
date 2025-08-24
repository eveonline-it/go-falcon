package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go-falcon/internal/auth/models"
	"go-falcon/pkg/config"

	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const (
	EVEAuthURL      = "https://login.eveonline.com/v2/oauth/authorize"
	EVETokenURL     = "https://login.eveonline.com/v2/oauth/token"
	EVEVerifyURL    = "https://login.eveonline.com/oauth/verify"
	EVEJWKSEndpoint = "https://login.eveonline.com/oauth/jwks"
	EVEIssuer       = "https://login.eveonline.com"
	EVEAudience     = "EVE Online"
)

// JWKSCache caches JWT public keys for validation
type JWKSCache struct {
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
	mu        sync.RWMutex
}

// EVEService handles EVE Online SSO integration
type EVEService struct {
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       string
	jwtSecret    []byte
	jwksCache    *JWKSCache
	repository   *Repository
}

// NewEVEService creates a new EVE SSO service
func NewEVEService(repository *Repository) *EVEService {
	return &EVEService{
		clientID:     config.GetEVEClientID(),
		clientSecret: config.GetEVEClientSecret(),
		redirectURI:  config.GetEVERedirectURI(),
		scopes:       config.GetEVEScopes(),
		jwtSecret:    []byte(config.GetJWTSecret()),
		repository:   repository,
		jwksCache: &JWKSCache{
			keys: make(map[string]*rsa.PublicKey),
		},
	}
}

// GenerateAuthURL generates an EVE SSO authorization URL
func (s *EVEService) GenerateAuthURL(ctx context.Context, withScopes bool, userID string) (string, string, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.eve_service.generate_auth_url")
	defer span.End()

	// Generate secure random state
	state, err := s.generateSecureState()
	if err != nil {
		span.RecordError(err)
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state in database
	loginState := &models.EVELoginState{
		State:  state,
		UserID: userID,
	}
	if err := s.repository.StoreLoginState(ctx, loginState); err != nil {
		span.RecordError(err)
		return "", "", fmt.Errorf("failed to store login state: %w", err)
	}

	// Build authorization URL
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("redirect_uri", s.redirectURI)
	params.Set("client_id", s.clientID)
	params.Set("state", state)

	if withScopes && s.scopes != "" {
		params.Set("scope", s.scopes)
	}

	authURL := EVEAuthURL + "?" + params.Encode()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "generate_auth_url"),
		attribute.Bool("with_scopes", withScopes),
		attribute.String("state", state),
	)

	return authURL, state, nil
}

// HandleCallback processes the OAuth callback from EVE
func (s *EVEService) HandleCallback(ctx context.Context, code, state string) (*models.EVECharacterInfo, *models.EVETokenResponse, string, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.eve_service.handle_callback")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "handle_callback"),
		attribute.String("state", state),
	)

	// Validate state
	loginState, err := s.repository.GetLoginState(ctx, state)
	if err != nil {
		span.RecordError(err)
		return nil, nil, "", fmt.Errorf("failed to validate state: %w", err)
	}
	if loginState == nil {
		err := errors.New("invalid or expired state")
		span.RecordError(err)
		return nil, nil, "", err
	}

	// Exchange code for tokens
	tokenResponse, err := s.exchangeCodeForToken(ctx, code)
	if err != nil {
		span.RecordError(err)
		return nil, nil, "", fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Verify token and get character info
	charInfo, err := s.verifyAccessToken(ctx, tokenResponse.AccessToken)
	if err != nil {
		span.RecordError(err)
		return nil, nil, "", fmt.Errorf("failed to verify access token: %w", err)
	}

	return charInfo, tokenResponse, loginState.UserID, nil
}

// ValidateJWT validates a JWT token and returns user information
func (s *EVEService) ValidateJWT(tokenString string) (*models.AuthenticatedUser, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid JWT token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid JWT claims")
	}

	// Extract user information from claims
	userID, _ := claims["user_id"].(string)
	characterIDFloat, _ := claims["character_id"].(float64)
	characterID := int(characterIDFloat)
	characterName, _ := claims["character_name"].(string)
	scopes, _ := claims["scopes"].(string)

	return &models.AuthenticatedUser{
		UserID:        userID,
		CharacterID:   characterID,
		CharacterName: characterName,
		Scopes:        scopes,
	}, nil
}

// GenerateJWT creates a JWT token for the authenticated user
func (s *EVEService) GenerateJWT(userID string, characterID int, characterName, scopes string) (string, time.Time, error) {
	expiresAt := time.Now().Add(config.GetCookieDuration())

	claims := jwt.MapClaims{
		"user_id":        userID,
		"character_id":   characterID,
		"character_name": characterName,
		"scopes":         scopes,
		"exp":            expiresAt.Unix(),
		"iat":            time.Now().Unix(),
		"iss":            "go-falcon",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign JWT: %w", err)
	}

	return tokenString, expiresAt, nil
}

// RefreshAccessToken refreshes an EVE access token using refresh token
func (s *EVEService) RefreshAccessToken(ctx context.Context, refreshToken string) (*models.EVETokenResponse, error) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(ctx, "auth.eve_service.refresh_access_token")
	defer span.End()

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", EVETokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(s.clientID+":"+s.clientSecret)))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("EVE token refresh failed: %s - %s", resp.Status, string(body))
		span.RecordError(err)
		return nil, err
	}

	var tokenResp models.EVETokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		span.RecordError(err)
		return nil, err
	}

	return &tokenResp, nil
}

// exchangeCodeForToken exchanges authorization code for access token
func (s *EVEService) exchangeCodeForToken(ctx context.Context, code string) (*models.EVETokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, "POST", EVETokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(s.clientID+":"+s.clientSecret)))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("EVE token exchange failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp models.EVETokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// verifyAccessToken verifies access token and gets character info
func (s *EVEService) verifyAccessToken(ctx context.Context, accessToken string) (*models.EVECharacterInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", EVEVerifyURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("EVE token verification failed: %s - %s", resp.Status, string(body))
	}

	var charInfo models.EVECharacterInfo
	if err := json.NewDecoder(resp.Body).Decode(&charInfo); err != nil {
		return nil, err
	}

	return &charInfo, nil
}

// generateSecureState generates a cryptographically secure random state
func (s *EVEService) generateSecureState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CleanupExpiredStates removes expired OAuth states
func (s *EVEService) CleanupExpiredStates(ctx context.Context) error {
	return s.repository.CleanupExpiredStates(ctx)
}
