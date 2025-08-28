# SDE Package (pkg/sde)

## Overview
EVE Online Static Data Export (SDE) in-memory service providing ultra-fast access to game static data including agents, categories, blueprints, types, market information, and complete EVE universe data. Single instance shared across all modules.

## Core Features
- **In-Memory Storage**: All data loaded at startup for nanosecond access
- **Type-Safe Access**: Structured Go types with proper JSON unmarshaling  
- **Lazy Loading**: Data loaded on first access to optimize startup
- **Thread-Safe**: Concurrent access via read-write mutexes
- **Extensible**: Easy to add new SDE data types

## Data Sources
- **Source Files**: `data/sde/*.json` converted from CCP's YAML format
- **Processing System**: Web-based SDE management via `internal/sde` module
- **Update Process**: Automated detection and web-based management of new static data

## Available Data Types

### Fully Implemented (46 types)

The SDE service currently loads and supports these data types for import:

- **`agents`**: Mission agents with location and corporation info
- **`categories`**: Item categories with internationalized names  
- **`blueprints`**: Manufacturing blueprints with material requirements
- **`marketGroups`**: Market categorization and hierarchy
- **`metaGroups`**: Item meta group classifications
- **`npcCorporations`**: NPC corporation data with faction info
- **`typeIDs`**: Basic type information (lightweight)
- **`types`**: Complete item type database with attributes
- **`typeMaterials`**: Manufacturing material requirements per type
- **`races`**: Character races with skills and ship information
- **`factions`**: EVE factions with member races and corporation relationships  
- **`bloodlines`**: Character bloodlines with attributes and racial information
- **`groups`**: Item groups with categorization and property flags
- **`dogmaAttributes`**: Item attributes and properties for EVE mechanics
- **`ancestries`**: Character ancestry information with bloodline relationships
- **`certificates`**: Skill certificates with tiered requirements (basic → elite)
- **`characterAttributes`**: Core character attributes (Intelligence, Charisma, etc.)
- **`skins`**: Ship skins with visibility and material information
- **`staStations`**: Station data with location and service information
- **`dogmaEffects`**: Dogma effects with modifiers and attributes
- **`iconIDs`**: Icon ID to file path mappings
- **`graphicIDs`**: Graphic ID definitions with SOF data
- **`typeDogma`**: Type dogma attributes and effects for items
- **`invFlags`**: Inventory flag definitions with names and order
- **`stationServices`**: Station service definitions with internationalized names
- **`stationOperations`**: Station operations with manufacturing factors
- **`researchAgents`**: Research agents with skill requirements
- **`agentsInSpace`**: Agents located in space with dungeon and location information
- **`contrabandTypes`**: Contraband types with faction-specific penalties and restrictions
- **`corporationActivities`**: Corporation activities with internationalized names
- **`invItems`**: Inventory items with location, ownership, and quantity information
- **`npcCorporationDivisions`**: NPC corporation divisions with internationalized names
- **`controlTowerResources`**: Control tower resource requirements with faction and security restrictions
- **`dogmaAttributeCategories`**: Dogma attribute categories with names and descriptions
- **`invNames`**: Inventory names mapping item IDs to readable names
- **`invPositions`**: Inventory position and orientation data for items in space
- **`invUniqueNames`**: Unique inventory names with group information
- **`planetResources`**: Planet resource power and workforce requirements
- **`planetSchematics`**: Planetary interaction schematics with cycle times and material flows
- **`skinLicenses`**: Ship skin licenses with duration and type information
- **`skinMaterials`**: Skin material definitions with display names and material sets
- **`sovereigntyUpgrades`**: Sovereignty upgrade specifications with fuel and resource costs
- **`translationLanguages`**: Language code to name mappings for internationalization

**Universe Data** (3 types - 9,725 files):
- **`regions`**: EVE Online regions with boundaries, factions, and nebula information (113 files)
- **`constellations`**: Star constellation data with positioning and radius information (1,175 files)
- **`solarSystems`**: Complete solar system data including planets, moons, stations, stargates, and asteroid belts (8,437 files)

**Complete Coverage**: All 46 SDE data types from the EVE Online Static Data Export are now fully implemented and supported, including complete universe structure (9,725+ files).

### Import Options

**Import All Supported Data Types** (default):
```bash
POST /sde/import
# Empty request body imports all 46 supported types
```

**Import Specific Data Types**:
```bash
POST /sde/import
{
  "data_types": ["agents", "types", "categories", "races", "factions", "dogmaAttributes", "regions", "constellations", "solarSystems"]
}
```

**Currently Supported Data Type Names**:
- `agents`
- `categories` 
- `blueprints`
- `marketGroups`
- `metaGroups`
- `npcCorporations`
- `typeIDs`
- `types`
- `typeMaterials`
- `races`
- `factions`
- `bloodlines`
- `groups`
- `dogmaAttributes`
- `ancestries`
- `certificates`
- `characterAttributes`
- `skins`
- `staStations`
- `dogmaEffects`
- `iconIDs`
- `graphicIDs`
- `typeDogma`
- `invFlags`
- `stationServices`
- `stationOperations`
- `researchAgents`
- `agentsInSpace`
- `contrabandTypes`
- `corporationActivities`
- `invItems`
- `npcCorporationDivisions`
- `controlTowerResources`
- `dogmaAttributeCategories`
- `invNames`
- `invPositions`
- `invUniqueNames`
- `planetResources`
- `planetSchematics`
- `skinLicenses`
- `skinMaterials`
- `sovereigntyUpgrades`
- `translationLanguages`
- `regions`
- `constellations`
- `solarSystems`

## Performance Characteristics
- **Memory Usage**: ~400MB with complete dataset including universe data (9,725+ files)
- **Access Speed**: O(1) map lookups, O(log n) sorted searches
- **Startup Impact**: 3-7 seconds initial load time (including universe data)
- **No Network Calls**: All data served from memory
- **Universe Coverage**: Complete EVE universe in memory (113 regions, 1,175 constellations, 8,437 solar systems)

## Usage Examples
```go
// Direct data access
agent, err := sdeService.GetAgent("3008416")
category, err := sdeService.GetCategory("1")
blueprint, err := sdeService.GetBlueprint("1000001")
race, err := sdeService.GetRace("1")
faction, err := sdeService.GetFaction("500001")
bloodline, err := sdeService.GetBloodline("1")
group, err := sdeService.GetGroup("25")
dogmaAttribute, err := sdeService.GetDogmaAttribute("1000")
ancestry, err := sdeService.GetAncestry("1")
certificate, err := sdeService.GetCertificate("100")
charAttribute, err := sdeService.GetCharacterAttribute("1")
skin, err := sdeService.GetSkin("10")
station, err := sdeService.GetStaStation(60000004)
dogmaEffect, err := sdeService.GetDogmaEffect("10")
iconID, err := sdeService.GetIconID("1001")
graphicID, err := sdeService.GetGraphicID("100")

// Query operations
agents := sdeService.GetAgentsByLocation(60000004)
publishedTypes := sdeService.GetPublishedTypes()
marketGroups := sdeService.GetAllMarketGroups()
allRaces := sdeService.GetAllRaces()
allFactions := sdeService.GetAllFactions()
allDogmaAttributes := sdeService.GetAllDogmaAttributes()
allCertificates := sdeService.GetAllCertificates()
allSkins := sdeService.GetAllSkins()
stationsBySystem := sdeService.GetStaStationsBySolarSystem(30002780)
allDogmaEffects := sdeService.GetAllDogmaEffects()
allIconIDs := sdeService.GetAllIconIDs()
allGraphicIDs := sdeService.GetAllGraphicIDs()
typeDogma, err := sdeService.GetTypeDogma("588")
invFlag, err := sdeService.GetInvFlag(4)
stationService, err := sdeService.GetStationService("1")
stationOperation, err := sdeService.GetStationOperation("1")
researchAgent, err := sdeService.GetResearchAgent("3003869")
allTypeDogma := sdeService.GetAllTypeDogma()
allInvFlags := sdeService.GetAllInvFlags()

// New data types
agentInSpace, err := sdeService.GetAgentInSpace("1000001")
allAgentsInSpace := sdeService.GetAllAgentsInSpace()
contrabandType, err := sdeService.GetContrabandType("11855")
allContrabandTypes := sdeService.GetAllContrabandTypes()
corporationActivity, err := sdeService.GetCorporationActivity("1")
allCorporationActivities := sdeService.GetAllCorporationActivities()
invItem, err := sdeService.GetInvItem(1000001)
allInvItems := sdeService.GetAllInvItems()
npcCorpDivision, err := sdeService.GetNPCCorporationDivision("1")
allNPCCorpDivisions := sdeService.GetAllNPCCorporationDivisions()
controlTowerResource, err := sdeService.GetControlTowerResources("12235")
allControlTowerResources := sdeService.GetAllControlTowerResources()
dogmaAttrCategory, err := sdeService.GetDogmaAttributeCategory("1")
allDogmaAttrCategories := sdeService.GetAllDogmaAttributeCategories()
invName, err := sdeService.GetInvName(1000001)
allInvNames := sdeService.GetAllInvNames()
invPosition, err := sdeService.GetInvPosition(1000001)
allInvPositions := sdeService.GetAllInvPositions()
invUniqueName, err := sdeService.GetInvUniqueName(500001)
allInvUniqueNames := sdeService.GetAllInvUniqueNames()
planetResource, err := sdeService.GetPlanetResource("40013180")
allPlanetResources := sdeService.GetAllPlanetResources()
planetSchematic, err := sdeService.GetPlanetSchematic("100")
allPlanetSchematics := sdeService.GetAllPlanetSchematics()
skinLicense, err := sdeService.GetSkinLicense("34599")
allSkinLicenses := sdeService.GetAllSkinLicenses()
skinMaterial, err := sdeService.GetSkinMaterial("1")
allSkinMaterials := sdeService.GetAllSkinMaterials()
sovereigntyUpgrade, err := sdeService.GetSovereigntyUpgrade("81615")
allSovereigntyUpgrades := sdeService.GetAllSovereigntyUpgrades()
translationLanguage, err := sdeService.GetTranslationLanguage("en-us")
allTranslationLanguages := sdeService.GetAllTranslationLanguages()

// Universe data access
region, err := sdeService.GetRegion(10000002)  // The Forge region
allRegions := sdeService.GetAllRegions()
constellation, err := sdeService.GetConstellation(20000020)  // Kimotoro constellation  
allConstellations := sdeService.GetAllConstellations()
solarSystem, err := sdeService.GetSolarSystem(30000142)  // Jita solar system
allSolarSystems := sdeService.GetAllSolarSystems()
constellationsByRegion := sdeService.GetConstellationsByRegion(10000002)  // All constellations in The Forge
systemsByConstellation := sdeService.GetSolarSystemsByConstellation(20000020)  // All systems in Kimotoro
```

## Service Interface
```go
type SDEService interface {
    IsLoaded() bool
    GetAgent(agentID string) (*Agent, error)
    GetCategory(categoryID string) (*Category, error)
    GetBlueprint(blueprintID string) (*Blueprint, error)
    // ... many more data access methods
    
    // Universe data methods
    GetRegion(regionID int) (*Region, error)
    GetAllRegions() (map[int]*Region, error)
    GetConstellation(constellationID int) (*Constellation, error)
    GetAllConstellations() (map[int]*Constellation, error)
    GetSolarSystem(solarSystemID int) (*SolarSystem, error)
    GetAllSolarSystems() (map[int]*SolarSystem, error)
    GetConstellationsByRegion(regionID int) ([]*Constellation, error)
    GetSolarSystemsByConstellation(constellationID int) ([]*SolarSystem, error)
}
```

## Integration Points
- **Module Access**: Available through base module interface
- **ESI Enrichment**: Combines with live ESI data for comprehensive game context
- **Universe Context**: Provides location-based enrichment for character, corporation, and alliance data
- **Development Testing**: Used by dev module for validation
- **Query Operations**: Supports complex data queries and filtering across universe hierarchy

## Thread Safety
- **Read-Write Mutexes**: Safe concurrent access
- **Immutable Data**: No modifications after loading
- **Lazy Initialization**: Thread-safe first-time loading
- **Shared Instance**: Single service across all modules