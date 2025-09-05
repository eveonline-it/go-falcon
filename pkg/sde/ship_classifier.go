package sde

import (
	"fmt"
	"strconv"
)

// ShipCategory represents a tracked ship category
type ShipCategory struct {
	Name    string
	GroupID int
}

// Ship categories we want to track based on Group IDs from EVE SDE
var TrackedShipCategories = map[int]string{
	541:  "interdictor", // Interdictor (Sabre, Eris, Heretic, Flycatcher)
	833:  "forcerecon",  // Force Recon Ship (Falcon, Rapier, Pilgrim, Arazu)
	963:  "strategic",   // Strategic Cruiser (Tengu, Legion, Proteus, Loki)
	894:  "hic",         // Heavy Interdiction Cruiser
	1972: "monitor",     // Flag Cruiser (Monitor)
	898:  "blackops",    // Black Ops
	900:  "marauders",   // Marauder
	1538: "fax",         // Force Auxiliary
	485:  "dread",       // Dreadnought
	547:  "carrier",     // Carrier
	659:  "super",       // Supercarrier
	30:   "titan",       // Titan
	4594: "lancer",      // Lancer Dreadnought
}

// Reverse mapping for quick category to group ID lookups
var CategoryToGroupID = map[string]int{
	"interdictor": 541,
	"forcerecon":  833,
	"strategic":   963,
	"hic":         894,
	"monitor":     1972,
	"blackops":    898,
	"marauders":   900,
	"fax":         1538,
	"dread":       485,
	"carrier":     547,
	"super":       659,
	"titan":       30,
	"lancer":      4594,
}

// ShipClassifier provides ship categorization functionality
type ShipClassifier struct {
	sdeService SDEService
}

// NewShipClassifier creates a new ship classifier
func NewShipClassifier(sdeService SDEService) *ShipClassifier {
	return &ShipClassifier{
		sdeService: sdeService,
	}
}

// GetShipCategory determines the category of a ship from its type ID
// Returns the category name or empty string if not a tracked category
func (sc *ShipClassifier) GetShipCategory(shipTypeID int64) (string, error) {
	if shipTypeID == 0 {
		return "", nil
	}

	shipType, err := sc.sdeService.GetType(strconv.FormatInt(shipTypeID, 10))
	if err != nil {
		return "", fmt.Errorf("failed to get ship type %d: %w", shipTypeID, err)
	}

	// Check if this ship's group ID matches any tracked category
	if category, exists := TrackedShipCategories[shipType.GroupID]; exists {
		return category, nil
	}

	return "", nil
}

// IsTrackedShipCategory checks if a ship belongs to any tracked category
func (sc *ShipClassifier) IsTrackedShipCategory(shipTypeID int64) (bool, error) {
	category, err := sc.GetShipCategory(shipTypeID)
	if err != nil {
		return false, err
	}
	return category != "", nil
}

// GetShipsByCategory returns all ships in a specific category
func (sc *ShipClassifier) GetShipsByCategory(categoryKey string) ([]*Type, error) {
	groupID, exists := CategoryToGroupID[categoryKey]
	if !exists {
		return nil, fmt.Errorf("category %s not found", categoryKey)
	}

	ships, err := sc.sdeService.GetTypesByGroupID(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ships for category %s: %w", categoryKey, err)
	}

	return ships, nil
}

// GetShipName gets the English name of a ship type
func (sc *ShipClassifier) GetShipName(shipTypeID int64) (string, error) {
	if shipTypeID == 0 {
		return "", nil
	}

	shipType, err := sc.sdeService.GetType(strconv.FormatInt(shipTypeID, 10))
	if err != nil {
		return "", fmt.Errorf("failed to get ship type %d: %w", shipTypeID, err)
	}

	if enName, ok := shipType.Name["en"]; ok {
		return enName, nil
	}

	return "", fmt.Errorf("no English name found for ship type %d", shipTypeID)
}

// GetTrackedCategories returns all tracked ship categories
func (sc *ShipClassifier) GetTrackedCategories() []string {
	categories := make([]string, 0, len(CategoryToGroupID))
	for category := range CategoryToGroupID {
		categories = append(categories, category)
	}
	return categories
}

// IsShipInCategory checks if a specific ship type belongs to a category
func (sc *ShipClassifier) IsShipInCategory(shipTypeID int64, categoryKey string) (bool, error) {
	category, err := sc.GetShipCategory(shipTypeID)
	if err != nil {
		return false, err
	}
	return category == categoryKey, nil
}

// GetShipCategoryInfo returns category information for a ship
func (sc *ShipClassifier) GetShipCategoryInfo(shipTypeID int64) (*ShipCategoryInfo, error) {
	if shipTypeID == 0 {
		return nil, nil
	}

	category, err := sc.GetShipCategory(shipTypeID)
	if err != nil {
		return nil, err
	}

	if category == "" {
		return nil, nil
	}

	shipName, err := sc.GetShipName(shipTypeID)
	if err != nil {
		return nil, err
	}

	return &ShipCategoryInfo{
		ShipTypeID: shipTypeID,
		ShipName:   shipName,
		Category:   category,
		GroupID:    CategoryToGroupID[category],
	}, nil
}

// ShipCategoryInfo represents ship classification information
type ShipCategoryInfo struct {
	ShipTypeID int64  `json:"ship_type_id"`
	ShipName   string `json:"ship_name"`
	Category   string `json:"category"`
	GroupID    int    `json:"group_id"`
}

// ValidateShipCategories validates that all tracked categories have valid group IDs in SDE
func (sc *ShipClassifier) ValidateShipCategories() error {
	for category, groupID := range CategoryToGroupID {
		ships, err := sc.sdeService.GetTypesByGroupID(groupID)
		if err != nil {
			return fmt.Errorf("validation failed for category %s (groupID %d): %w", category, groupID, err)
		}

		if len(ships) == 0 {
			return fmt.Errorf("no ships found for category %s (groupID %d)", category, groupID)
		}
	}
	return nil
}
