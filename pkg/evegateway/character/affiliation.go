package character

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CharacterAffiliation represents a character's corporation and alliance affiliations from ESI
type CharacterAffiliation struct {
	CharacterID   int `json:"character_id"`
	CorporationID int `json:"corporation_id"`
	AllianceID    int `json:"alliance_id,omitempty"`
	FactionID     int `json:"faction_id,omitempty"`
}

// CharacterAffiliationResult contains the affiliation response with cache metadata
type CharacterAffiliationResult struct {
	Affiliations []CharacterAffiliation
	ExpiresAt    time.Time
	CacheHit     bool
}

// GetCharactersAffiliation performs bulk lookup of character affiliations
// Maximum of 1000 character IDs per request as per ESI specification
func (c *CharacterClient) GetCharactersAffiliation(ctx context.Context, characterIDs []int) ([]CharacterAffiliation, error) {
	// Validate input
	if len(characterIDs) == 0 {
		return []CharacterAffiliation{}, nil
	}
	if len(characterIDs) > 1000 {
		return nil, fmt.Errorf("maximum 1000 character IDs allowed per request, got %d", len(characterIDs))
	}

	// Prepare request body
	requestBody, err := json.Marshal(characterIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal character IDs: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/latest/characters/affiliation/", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute request with retry
	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to get character affiliations: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	// Parse response
	var affiliations []CharacterAffiliation
	if err := json.NewDecoder(resp.Body).Decode(&affiliations); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return affiliations, nil
}

// GetCharactersAffiliationWithCache performs bulk lookup with caching support
func (c *CharacterClient) GetCharactersAffiliationWithCache(ctx context.Context, characterIDs []int) (*CharacterAffiliationResult, error) {
	// Validate input
	if len(characterIDs) == 0 {
		return &CharacterAffiliationResult{
			Affiliations: []CharacterAffiliation{},
			ExpiresAt:    time.Now(),
			CacheHit:     false,
		}, nil
	}
	if len(characterIDs) > 1000 {
		return nil, fmt.Errorf("maximum 1000 character IDs allowed per request, got %d", len(characterIDs))
	}

	// Create cache key based on character IDs
	// Note: This is a simplistic approach; in production you might want a more sophisticated cache key
	cacheKeyData, _ := json.Marshal(characterIDs)
	cacheKey := fmt.Sprintf("affiliation:%x", cacheKeyData)

	// Check cache
	if cachedData, exists, err := c.cacheManager.Get(cacheKey); err == nil && exists {
		var affiliations []CharacterAffiliation
		if err := json.Unmarshal(cachedData, &affiliations); err == nil {
			// Get cache expiry
			expiresAt := time.Now().Add(1 * time.Hour) // Default 1 hour if not found
			if metadata, err := c.cacheManager.GetMetadata(cacheKey); err == nil && metadata != nil {
				if exp, ok := metadata["expires_at"].(time.Time); ok {
					expiresAt = exp
				}
			}
			
			return &CharacterAffiliationResult{
				Affiliations: affiliations,
				ExpiresAt:    expiresAt,
				CacheHit:     true,
			}, nil
		}
	}

	// Prepare request body
	requestBody, err := json.Marshal(characterIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal character IDs: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/latest/characters/affiliation/", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Add conditional headers if we have cached data
	c.cacheManager.SetConditionalHeaders(req, cacheKey)

	// Execute request with retry
	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to get character affiliations: %w", err)
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		// Refresh cache expiry
		c.cacheManager.RefreshExpiry(cacheKey, resp.Header)
		
		// Get cached data
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			var affiliations []CharacterAffiliation
			if err := json.Unmarshal(cachedData, &affiliations); err == nil {
				expiresAt := time.Now().Add(1 * time.Hour) // Default
				if exp := resp.Header.Get("Expires"); exp != "" {
					if t, err := time.Parse(time.RFC1123, exp); err == nil {
						expiresAt = t
					}
				}
				
				return &CharacterAffiliationResult{
					Affiliations: affiliations,
					ExpiresAt:    expiresAt,
					CacheHit:     true,
				}, nil
			}
		}
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	// Parse response
	var affiliations []CharacterAffiliation
	if err := json.NewDecoder(resp.Body).Decode(&affiliations); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Cache the response
	if data, err := json.Marshal(affiliations); err == nil {
		c.cacheManager.Set(cacheKey, data, resp.Header)
	}

	// Parse expiry time from headers
	expiresAt := time.Now().Add(1 * time.Hour) // Default to 1 hour
	if exp := resp.Header.Get("Expires"); exp != "" {
		if t, err := time.Parse(time.RFC1123, exp); err == nil {
			expiresAt = t
		}
	}

	return &CharacterAffiliationResult{
		Affiliations: affiliations,
		ExpiresAt:    expiresAt,
		CacheHit:     false,
	}, nil
}