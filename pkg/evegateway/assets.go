package evegateway

import (
	"context"
)

// Asset represents an EVE Online asset
type Asset struct {
	ItemID          int64  `json:"item_id"`
	TypeID          int32  `json:"type_id"`
	LocationID      int64  `json:"location_id"`
	LocationFlag    string `json:"location_flag"`
	LocationType    string `json:"location_type"`
	Quantity        int32  `json:"quantity"`
	IsSingleton     bool   `json:"is_singleton"`
	IsBlueprintCopy bool   `json:"is_blueprint_copy,omitempty"`
}

// GetCharacterAssets retrieves character assets from ESI
func (c *Client) GetCharacterAssets(ctx context.Context, characterID int32) ([]Asset, error) {
	// TODO: Implement actual ESI call with pagination
	// For now, return dummy assets for testing
	return []Asset{
		{
			ItemID:       1000000001,
			TypeID:       34,
			LocationID:   60003760,
			LocationFlag: "Hangar",
			LocationType: "station",
			Quantity:     100,
			IsSingleton:  false,
		},
		{
			ItemID:       1000000002,
			TypeID:       35,
			LocationID:   60003760,
			LocationFlag: "Hangar",
			LocationType: "station",
			Quantity:     200,
			IsSingleton:  false,
		},
	}, nil
}

// GetCorporationAssets retrieves corporation assets from ESI
func (c *Client) GetCorporationAssets(ctx context.Context, corporationID, characterID int32) ([]Asset, error) {
	// TODO: Implement actual ESI call with pagination
	// Character must have appropriate roles (Director/Accountant)
	// For now, return dummy assets for testing
	return []Asset{
		{
			ItemID:       2000000001,
			TypeID:       36,
			LocationID:   60003760,
			LocationFlag: "CorpSAG1",
			LocationType: "station",
			Quantity:     1000,
			IsSingleton:  false,
		},
	}, nil
}
