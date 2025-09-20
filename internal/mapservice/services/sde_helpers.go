package services

import (
	"fmt"
	"strconv"

	"go-falcon/pkg/sde"
)

// Helper methods for SDE integration

// GetSystemName retrieves the name for a solar system using SDE name lookup
func GetSystemName(sdeService *sde.Service, system *sde.SolarSystem) string {
	if system == nil || sdeService == nil {
		return ""
	}

	// Look up the name using the SolarSystemID (not SolarSystemNameID)
	// The invNames.json uses solarSystemID as the itemID for solar systems
	if invName, err := sdeService.GetInvName(system.SolarSystemID); err == nil && invName != nil {
		if name, ok := invName.ItemName.(string); ok {
			return name
		}
	}

	// Fallback to system ID if name lookup fails
	return fmt.Sprintf("System-%d", system.SolarSystemID)
}

// GetRegionName retrieves the name for a region using SDE name lookup
func GetRegionName(sdeService *sde.Service, region *sde.Region) string {
	if region == nil || sdeService == nil {
		return ""
	}

	// Look up the name using the RegionID (not NameID)
	// The invNames.json uses regionID as the itemID for regions
	if invName, err := sdeService.GetInvName(region.RegionID); err == nil && invName != nil {
		if name, ok := invName.ItemName.(string); ok {
			return name
		}
	}

	// Fallback to region ID if name lookup fails
	return fmt.Sprintf("Region-%d", region.RegionID)
}

// GetConstellationName retrieves the name for a constellation using SDE name lookup
func GetConstellationName(sdeService *sde.Service, constellation *sde.Constellation) string {
	if constellation == nil || sdeService == nil {
		return ""
	}

	// Look up the name using the ConstellationID (not NameID)
	// The invNames.json uses constellationID as the itemID for constellations
	if invName, err := sdeService.GetInvName(constellation.ConstellationID); err == nil && invName != nil {
		if name, ok := invName.ItemName.(string); ok {
			return name
		}
	}

	// Fallback to constellation ID if name lookup fails
	return fmt.Sprintf("Constellation-%d", constellation.ConstellationID)
}

// getConstellationIDForSystem finds which constellation a system belongs to
func getConstellationIDForSystem(sdeService *sde.Service, system *sde.SolarSystem) int32 {
	if system == nil || sdeService == nil {
		return 0
	}

	// Iterate through all constellations to find which one contains this system
	allConstellations, err := sdeService.GetAllConstellations()
	if err != nil {
		return 0
	}

	for constellationID, constellation := range allConstellations {
		systems, err := sdeService.GetSolarSystemsByConstellation(constellationID)
		if err != nil {
			continue
		}

		for _, sys := range systems {
			if sys.SolarSystemID == system.SolarSystemID {
				return int32(constellation.ConstellationID)
			}
		}
	}

	return 0
}

// getRegionIDForConstellation finds which region a constellation belongs to
func getRegionIDForConstellation(sdeService *sde.Service, constellation *sde.Constellation) int32 {
	if constellation == nil || sdeService == nil {
		return 0
	}

	// Iterate through all regions to find which one contains this constellation
	allRegions, err := sdeService.GetAllRegions()
	if err != nil {
		return 0
	}

	for regionID, region := range allRegions {
		constellations, err := sdeService.GetConstellationsByRegion(regionID)
		if err != nil {
			continue
		}

		for _, cons := range constellations {
			if cons.ConstellationID == constellation.ConstellationID {
				return int32(region.RegionID)
			}
		}
	}

	return 0
}

// getSystemsByRegion retrieves all systems in a region
func getSystemsByRegion(sdeService *sde.Service, regionID int) ([]*sde.SolarSystem, error) {
	if sdeService == nil {
		return nil, fmt.Errorf("SDE service is nil")
	}

	var systems []*sde.SolarSystem

	// Get all constellations in the region
	constellations, err := sdeService.GetConstellationsByRegion(regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get constellations for region %d: %w", regionID, err)
	}

	// Get all systems in each constellation
	for _, constellation := range constellations {
		constellationSystems, err := sdeService.GetSolarSystemsByConstellation(constellation.ConstellationID)
		if err != nil {
			continue // Skip constellations that can't be loaded
		}
		systems = append(systems, constellationSystems...)
	}

	return systems, nil
}

// StargateDestination represents information about a stargate's destination
type StargateDestination struct {
	SystemID int32
	Stargate *sde.Stargate
}

// getStargateDestinationSystem finds the system that a stargate connects to
func getStargateDestinationSystem(sdeService *sde.Service, destinationGateID int) *StargateDestination {
	// We need to find which system contains the destination stargate
	allSystems, err := sdeService.GetAllSolarSystems()
	if err != nil {
		return nil
	}

	destinationIDStr := strconv.Itoa(destinationGateID)
	for systemID, system := range allSystems {
		if system.Stargates != nil {
			if stargate, exists := system.Stargates[destinationIDStr]; exists {
				return &StargateDestination{
					SystemID: int32(systemID),
					Stargate: stargate,
				}
			}
		}
	}

	return nil
}
