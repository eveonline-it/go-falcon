package sde

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"reflect"
	"runtime"
	"time"
	"unsafe"
)

// DataTypeStatus represents the load status of a single data type
type DataTypeStatus struct {
	Name         string `json:"name"`
	Loaded       bool   `json:"loaded"`
	Count        int    `json:"count"`
	MemoryBytes  int64  `json:"memory_bytes"`
	LoadError    string `json:"load_error,omitempty"`
	LastReloaded int64  `json:"last_reloaded,omitempty"` // Unix timestamp
}

// DataTypeStats provides detailed statistics for a data type
type DataTypeStats struct {
	Name        string `json:"name"`
	Count       int    `json:"count"`
	MemoryBytes int64  `json:"memory_bytes"`
	Loaded      bool   `json:"loaded"`
	FilePath    string `json:"file_path"`
}

// GetLoadedDataTypes returns a list of all loaded data types
func (s *Service) GetLoadedDataTypes() []string {
	if !s.loaded {
		return []string{}
	}

	types := []string{}
	if len(s.agents) > 0 {
		types = append(types, "agents")
	}
	if len(s.categories) > 0 {
		types = append(types, "categories")
	}
	if len(s.blueprints) > 0 {
		types = append(types, "blueprints")
	}
	if len(s.marketGroups) > 0 {
		types = append(types, "marketGroups")
	}
	if len(s.metaGroups) > 0 {
		types = append(types, "metaGroups")
	}
	if len(s.npcCorporations) > 0 {
		types = append(types, "npcCorporations")
	}
	if len(s.typeIDs) > 0 {
		types = append(types, "typeIDs")
	}
	if len(s.types) > 0 {
		types = append(types, "types")
	}
	if len(s.typeMaterials) > 0 {
		types = append(types, "typeMaterials")
	}
	if len(s.races) > 0 {
		types = append(types, "races")
	}
	if len(s.factions) > 0 {
		types = append(types, "factions")
	}
	if len(s.bloodlines) > 0 {
		types = append(types, "bloodlines")
	}
	if len(s.groups) > 0 {
		types = append(types, "groups")
	}
	if len(s.dogmaAttributes) > 0 {
		types = append(types, "dogmaAttributes")
	}
	if len(s.ancestries) > 0 {
		types = append(types, "ancestries")
	}
	if len(s.certificates) > 0 {
		types = append(types, "certificates")
	}
	if len(s.characterAttributes) > 0 {
		types = append(types, "characterAttributes")
	}
	if len(s.skins) > 0 {
		types = append(types, "skins")
	}
	if len(s.staStations) > 0 {
		types = append(types, "staStations")
	}
	if len(s.dogmaEffects) > 0 {
		types = append(types, "dogmaEffects")
	}
	if len(s.iconIDs) > 0 {
		types = append(types, "iconIDs")
	}
	if len(s.graphicIDs) > 0 {
		types = append(types, "graphicIDs")
	}
	if len(s.typeDogma) > 0 {
		types = append(types, "typeDogma")
	}
	if len(s.invFlags) > 0 {
		types = append(types, "invFlags")
	}
	if len(s.stationServices) > 0 {
		types = append(types, "stationServices")
	}
	if len(s.stationOperations) > 0 {
		types = append(types, "stationOperations")
	}
	if len(s.researchAgents) > 0 {
		types = append(types, "researchAgents")
	}
	if len(s.agentsInSpace) > 0 {
		types = append(types, "agentsInSpace")
	}
	if len(s.contrabandTypes) > 0 {
		types = append(types, "contrabandTypes")
	}
	if len(s.corporationActivities) > 0 {
		types = append(types, "corporationActivities")
	}
	if len(s.invItems) > 0 {
		types = append(types, "invItems")
	}
	if len(s.npcCorporationDivisions) > 0 {
		types = append(types, "npcCorporationDivisions")
	}
	if len(s.controlTowerResources) > 0 {
		types = append(types, "controlTowerResources")
	}
	if len(s.dogmaAttributeCategories) > 0 {
		types = append(types, "dogmaAttributeCategories")
	}
	if len(s.invNames) > 0 {
		types = append(types, "invNames")
	}
	if len(s.invPositions) > 0 {
		types = append(types, "invPositions")
	}
	if len(s.invUniqueNames) > 0 {
		types = append(types, "invUniqueNames")
	}
	if len(s.planetResources) > 0 {
		types = append(types, "planetResources")
	}
	if len(s.planetSchematics) > 0 {
		types = append(types, "planetSchematics")
	}
	if len(s.skinLicenses) > 0 {
		types = append(types, "skinLicenses")
	}
	if len(s.skinMaterials) > 0 {
		types = append(types, "skinMaterials")
	}
	if len(s.sovereigntyUpgrades) > 0 {
		types = append(types, "sovereigntyUpgrades")
	}
	if len(s.translationLanguages) > 0 {
		types = append(types, "translationLanguages")
	}
	return types
}

// GetDataTypeStats returns statistics for a specific data type
func (s *Service) GetDataTypeStats(dataType string) DataTypeStats {
	stats := DataTypeStats{
		Name:   dataType,
		Loaded: s.loaded,
	}

	switch dataType {
	case "agents":
		stats.Count = len(s.agents)
		stats.MemoryBytes = estimateMapMemory(s.agents)
		stats.FilePath = filepath.Join(s.dataDir, "agents.json")
	case "categories":
		stats.Count = len(s.categories)
		stats.MemoryBytes = estimateMapMemory(s.categories)
		stats.FilePath = filepath.Join(s.dataDir, "categories.json")
	case "blueprints":
		stats.Count = len(s.blueprints)
		stats.MemoryBytes = estimateMapMemory(s.blueprints)
		stats.FilePath = filepath.Join(s.dataDir, "blueprints.json")
	case "marketGroups":
		stats.Count = len(s.marketGroups)
		stats.MemoryBytes = estimateMapMemory(s.marketGroups)
		stats.FilePath = filepath.Join(s.dataDir, "marketGroups.json")
	case "metaGroups":
		stats.Count = len(s.metaGroups)
		stats.MemoryBytes = estimateMapMemory(s.metaGroups)
		stats.FilePath = filepath.Join(s.dataDir, "metaGroups.json")
	case "npcCorporations":
		stats.Count = len(s.npcCorporations)
		stats.MemoryBytes = estimateMapMemory(s.npcCorporations)
		stats.FilePath = filepath.Join(s.dataDir, "npcCorporations.json")
	case "typeIDs":
		stats.Count = len(s.typeIDs)
		stats.MemoryBytes = estimateMapMemory(s.typeIDs)
		stats.FilePath = filepath.Join(s.dataDir, "typeIDs.json")
	case "types":
		stats.Count = len(s.types)
		stats.MemoryBytes = estimateMapMemory(s.types)
		stats.FilePath = filepath.Join(s.dataDir, "types.json")
	case "typeMaterials":
		stats.Count = len(s.typeMaterials)
		stats.MemoryBytes = estimateMapMemory(s.typeMaterials)
		stats.FilePath = filepath.Join(s.dataDir, "typeMaterials.json")
	case "races":
		stats.Count = len(s.races)
		stats.MemoryBytes = estimateMapMemory(s.races)
		stats.FilePath = filepath.Join(s.dataDir, "races.json")
	case "factions":
		stats.Count = len(s.factions)
		stats.MemoryBytes = estimateMapMemory(s.factions)
		stats.FilePath = filepath.Join(s.dataDir, "factions.json")
	case "bloodlines":
		stats.Count = len(s.bloodlines)
		stats.MemoryBytes = estimateMapMemory(s.bloodlines)
		stats.FilePath = filepath.Join(s.dataDir, "bloodlines.json")
	case "groups":
		stats.Count = len(s.groups)
		stats.MemoryBytes = estimateMapMemory(s.groups)
		stats.FilePath = filepath.Join(s.dataDir, "groups.json")
	case "dogmaAttributes":
		stats.Count = len(s.dogmaAttributes)
		stats.MemoryBytes = estimateMapMemory(s.dogmaAttributes)
		stats.FilePath = filepath.Join(s.dataDir, "dogmaAttributes.json")
	case "ancestries":
		stats.Count = len(s.ancestries)
		stats.MemoryBytes = estimateMapMemory(s.ancestries)
		stats.FilePath = filepath.Join(s.dataDir, "ancestries.json")
	case "certificates":
		stats.Count = len(s.certificates)
		stats.MemoryBytes = estimateMapMemory(s.certificates)
		stats.FilePath = filepath.Join(s.dataDir, "certificates.json")
	case "characterAttributes":
		stats.Count = len(s.characterAttributes)
		stats.MemoryBytes = estimateMapMemory(s.characterAttributes)
		stats.FilePath = filepath.Join(s.dataDir, "characterAttributes.json")
	case "skins":
		stats.Count = len(s.skins)
		stats.MemoryBytes = estimateMapMemory(s.skins)
		stats.FilePath = filepath.Join(s.dataDir, "skins.json")
	case "staStations":
		stats.Count = len(s.staStations)
		stats.MemoryBytes = estimateSliceMemory(s.staStations)
		stats.FilePath = filepath.Join(s.dataDir, "staStations.json")
	case "dogmaEffects":
		stats.Count = len(s.dogmaEffects)
		stats.MemoryBytes = estimateMapMemory(s.dogmaEffects)
		stats.FilePath = filepath.Join(s.dataDir, "dogmaEffects.json")
	case "iconIDs":
		stats.Count = len(s.iconIDs)
		stats.MemoryBytes = estimateMapMemory(s.iconIDs)
		stats.FilePath = filepath.Join(s.dataDir, "iconIDs.json")
	case "graphicIDs":
		stats.Count = len(s.graphicIDs)
		stats.MemoryBytes = estimateMapMemory(s.graphicIDs)
		stats.FilePath = filepath.Join(s.dataDir, "graphicIDs.json")
	case "typeDogma":
		stats.Count = len(s.typeDogma)
		stats.MemoryBytes = estimateMapMemory(s.typeDogma)
		stats.FilePath = filepath.Join(s.dataDir, "typeDogma.json")
	case "invFlags":
		stats.Count = len(s.invFlags)
		stats.MemoryBytes = estimateSliceMemory(s.invFlags)
		stats.FilePath = filepath.Join(s.dataDir, "invFlags.json")
	case "stationServices":
		stats.Count = len(s.stationServices)
		stats.MemoryBytes = estimateMapMemory(s.stationServices)
		stats.FilePath = filepath.Join(s.dataDir, "stationServices.json")
	case "stationOperations":
		stats.Count = len(s.stationOperations)
		stats.MemoryBytes = estimateMapMemory(s.stationOperations)
		stats.FilePath = filepath.Join(s.dataDir, "stationOperations.json")
	case "researchAgents":
		stats.Count = len(s.researchAgents)
		stats.MemoryBytes = estimateMapMemory(s.researchAgents)
		stats.FilePath = filepath.Join(s.dataDir, "researchAgents.json")
	case "agentsInSpace":
		stats.Count = len(s.agentsInSpace)
		stats.MemoryBytes = estimateMapMemory(s.agentsInSpace)
		stats.FilePath = filepath.Join(s.dataDir, "agentsInSpace.json")
	case "contrabandTypes":
		stats.Count = len(s.contrabandTypes)
		stats.MemoryBytes = estimateMapMemory(s.contrabandTypes)
		stats.FilePath = filepath.Join(s.dataDir, "contrabandTypes.json")
	case "corporationActivities":
		stats.Count = len(s.corporationActivities)
		stats.MemoryBytes = estimateMapMemory(s.corporationActivities)
		stats.FilePath = filepath.Join(s.dataDir, "corporationActivities.json")
	case "invItems":
		stats.Count = len(s.invItems)
		stats.MemoryBytes = estimateSliceMemory(s.invItems)
		stats.FilePath = filepath.Join(s.dataDir, "invItems.json")
	case "npcCorporationDivisions":
		stats.Count = len(s.npcCorporationDivisions)
		stats.MemoryBytes = estimateMapMemory(s.npcCorporationDivisions)
		stats.FilePath = filepath.Join(s.dataDir, "npcCorporationDivisions.json")
	case "controlTowerResources":
		stats.Count = len(s.controlTowerResources)
		stats.MemoryBytes = estimateMapMemory(s.controlTowerResources)
		stats.FilePath = filepath.Join(s.dataDir, "controlTowerResources.json")
	case "dogmaAttributeCategories":
		stats.Count = len(s.dogmaAttributeCategories)
		stats.MemoryBytes = estimateMapMemory(s.dogmaAttributeCategories)
		stats.FilePath = filepath.Join(s.dataDir, "dogmaAttributeCategories.json")
	case "invNames":
		stats.Count = len(s.invNames)
		stats.MemoryBytes = estimateSliceMemory(s.invNames)
		stats.FilePath = filepath.Join(s.dataDir, "invNames.json")
	case "invPositions":
		stats.Count = len(s.invPositions)
		stats.MemoryBytes = estimateSliceMemory(s.invPositions)
		stats.FilePath = filepath.Join(s.dataDir, "invPositions.json")
	case "invUniqueNames":
		stats.Count = len(s.invUniqueNames)
		stats.MemoryBytes = estimateSliceMemory(s.invUniqueNames)
		stats.FilePath = filepath.Join(s.dataDir, "invUniqueNames.json")
	case "planetResources":
		stats.Count = len(s.planetResources)
		stats.MemoryBytes = estimateMapMemory(s.planetResources)
		stats.FilePath = filepath.Join(s.dataDir, "planetResources.json")
	case "planetSchematics":
		stats.Count = len(s.planetSchematics)
		stats.MemoryBytes = estimateMapMemory(s.planetSchematics)
		stats.FilePath = filepath.Join(s.dataDir, "planetSchematics.json")
	case "skinLicenses":
		stats.Count = len(s.skinLicenses)
		stats.MemoryBytes = estimateMapMemory(s.skinLicenses)
		stats.FilePath = filepath.Join(s.dataDir, "skinLicenses.json")
	case "skinMaterials":
		stats.Count = len(s.skinMaterials)
		stats.MemoryBytes = estimateMapMemory(s.skinMaterials)
		stats.FilePath = filepath.Join(s.dataDir, "skinMaterials.json")
	case "sovereigntyUpgrades":
		stats.Count = len(s.sovereigntyUpgrades)
		stats.MemoryBytes = estimateMapMemory(s.sovereigntyUpgrades)
		stats.FilePath = filepath.Join(s.dataDir, "sovereigntyUpgrades.json")
	case "translationLanguages":
		stats.Count = len(s.translationLanguages)
		stats.MemoryBytes = estimateMapMemory(s.translationLanguages)
		stats.FilePath = filepath.Join(s.dataDir, "translationLanguages.json")
	}

	return stats
}

// GetTotalMemoryUsage returns estimated total memory usage
func (s *Service) GetTotalMemoryUsage() int64 {
	if !s.loaded {
		return 0
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Return allocated memory for heap objects
	return int64(memStats.Alloc)
}

// GetLoadStatus returns the load status of all data types
func (s *Service) GetLoadStatus() map[string]DataTypeStatus {
	status := make(map[string]DataTypeStatus)

	dataTypes := []string{
		"agents", "categories", "blueprints", "marketGroups", "metaGroups",
		"npcCorporations", "typeIDs", "types", "typeMaterials", "races",
		"factions", "bloodlines", "groups", "dogmaAttributes", "ancestries",
		"certificates", "characterAttributes", "skins", "staStations", "dogmaEffects",
		"iconIDs", "graphicIDs", "typeDogma", "invFlags", "stationServices",
		"stationOperations", "researchAgents", "agentsInSpace", "contrabandTypes",
		"corporationActivities", "invItems", "npcCorporationDivisions",
		"controlTowerResources", "dogmaAttributeCategories", "invNames",
		"invPositions", "invUniqueNames", "planetResources", "planetSchematics",
		"skinLicenses", "skinMaterials", "sovereigntyUpgrades", "translationLanguages",
	}

	for _, dt := range dataTypes {
		stats := s.GetDataTypeStats(dt)
		status[dt] = DataTypeStatus{
			Name:        dt,
			Loaded:      stats.Count > 0,
			Count:       stats.Count,
			MemoryBytes: stats.MemoryBytes,
		}
	}

	return status
}

// ReloadDataType reloads a specific data type from file
func (s *Service) ReloadDataType(dataType string) error {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	switch dataType {
	case "agents":
		return s.loadAgents()
	case "categories":
		return s.loadCategories()
	case "blueprints":
		return s.loadBlueprints()
	case "marketGroups":
		return s.loadMarketGroups()
	case "metaGroups":
		return s.loadMetaGroups()
	case "npcCorporations":
		return s.loadNPCCorporations()
	case "typeIDs":
		return s.loadTypeIDs()
	case "types":
		return s.loadTypes()
	case "typeMaterials":
		return s.loadTypeMaterials()
	case "races":
		return s.loadRaces()
	case "factions":
		return s.loadFactions()
	case "bloodlines":
		return s.loadBloodlines()
	case "groups":
		return s.loadGroups()
	case "dogmaAttributes":
		return s.loadDogmaAttributes()
	case "ancestries":
		return s.loadAncestries()
	case "certificates":
		return s.loadCertificates()
	case "characterAttributes":
		return s.loadCharacterAttributes()
	case "skins":
		return s.loadSkins()
	case "staStations":
		return s.loadStaStations()
	case "dogmaEffects":
		return s.loadDogmaEffects()
	case "iconIDs":
		return s.loadIconIDs()
	case "graphicIDs":
		return s.loadGraphicIDs()
	case "typeDogma":
		return s.loadTypeDogma()
	case "invFlags":
		return s.loadInvFlags()
	case "stationServices":
		return s.loadStationServices()
	case "stationOperations":
		return s.loadStationOperations()
	case "researchAgents":
		return s.loadResearchAgents()
	case "agentsInSpace":
		return s.loadAgentsInSpace()
	case "contrabandTypes":
		return s.loadContrabandTypes()
	case "corporationActivities":
		return s.loadCorporationActivities()
	case "invItems":
		return s.loadInvItems()
	case "npcCorporationDivisions":
		return s.loadNPCCorporationDivisions()
	case "controlTowerResources":
		return s.loadControlTowerResources()
	case "dogmaAttributeCategories":
		return s.loadDogmaAttributeCategories()
	case "invNames":
		return s.loadInvNames()
	case "invPositions":
		return s.loadInvPositions()
	case "invUniqueNames":
		return s.loadInvUniqueNames()
	case "planetResources":
		return s.loadPlanetResources()
	case "planetSchematics":
		return s.loadPlanetSchematics()
	case "skinLicenses":
		return s.loadSkinLicenses()
	case "skinMaterials":
		return s.loadSkinMaterials()
	case "sovereigntyUpgrades":
		return s.loadSovereigntyUpgrades()
	case "translationLanguages":
		return s.loadTranslationLanguages()
	default:
		return fmt.Errorf("unknown data type: %s", dataType)
	}
}

// ReloadAll reloads all data types from files
func (s *Service) ReloadAll() error {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	startTime := time.Now()
	slog.Debug("SDE ReloadAll started", "data_dir", s.dataDir, "timestamp", startTime.Unix())

	// Reset loaded flag
	s.loaded = false

	// Clear existing data
	s.agents = make(map[string]*Agent)
	s.categories = make(map[string]*Category)
	s.blueprints = make(map[string]*Blueprint)
	s.marketGroups = make(map[string]*MarketGroup)
	s.metaGroups = make(map[string]*MetaGroup)
	s.npcCorporations = make(map[string]*NPCCorporation)
	s.typeIDs = make(map[string]*TypeID)
	s.types = make(map[string]*Type)
	s.typeMaterials = make(map[string][]*TypeMaterial)
	s.races = make(map[string]*Race)
	s.factions = make(map[string]*Faction)
	s.bloodlines = make(map[string]*Bloodline)
	s.groups = make(map[string]*Group)
	s.dogmaAttributes = make(map[string]*DogmaAttribute)
	s.ancestries = make(map[string]*Ancestry)
	s.certificates = make(map[string]*Certificate)
	s.characterAttributes = make(map[string]*CharacterAttribute)
	s.skins = make(map[string]*Skin)
	s.staStations = []*StaStation{}
	s.dogmaEffects = make(map[string]*DogmaEffect)
	s.iconIDs = make(map[string]*IconID)
	s.graphicIDs = make(map[string]*GraphicID)
	s.typeDogma = make(map[string]*TypeDogma)
	s.invFlags = []*InvFlag{}
	s.stationServices = make(map[string]*StationService)
	s.stationOperations = make(map[string]*StationOperation)
	s.researchAgents = make(map[string]*ResearchAgent)
	s.agentsInSpace = make(map[string]*AgentInSpace)
	s.contrabandTypes = make(map[string]*ContrabandType)
	s.corporationActivities = make(map[string]*CorporationActivity)
	s.invItems = []*InvItem{}
	s.npcCorporationDivisions = make(map[string]*NPCCorporationDivision)
	s.controlTowerResources = make(map[string]*ControlTowerResources)
	s.dogmaAttributeCategories = make(map[string]*DogmaAttributeCategory)
	s.invNames = []*InvName{}
	s.invPositions = []*InvPosition{}
	s.invUniqueNames = []*InvUniqueName{}
	s.planetResources = make(map[string]*PlanetResource)
	s.planetSchematics = make(map[string]*PlanetSchematic)
	s.skinLicenses = make(map[string]*SkinLicense)
	s.skinMaterials = make(map[string]*SkinMaterial)
	s.sovereigntyUpgrades = make(map[string]*SovereigntyUpgrade)
	s.translationLanguages = make(map[string]*TranslationLanguage)

	// Reload all data using ensureLoaded logic (without the lock since we already have it)
	s.loadMu.Unlock()
	err := s.ensureLoaded()
	s.loadMu.Lock()

	// Log completion with timing
	reloadDuration := time.Since(startTime)
	if err == nil {
		slog.Debug("SDE ReloadAll completed successfully",
			"total_data_types", 43,
			"reload_duration_ms", reloadDuration.Milliseconds(),
			"timestamp", time.Now().Unix())
	} else {
		slog.Error("SDE ReloadAll failed",
			"error", err,
			"reload_duration_ms", reloadDuration.Milliseconds(),
			"timestamp", time.Now().Unix())
	}

	return err
}

// estimateMapMemory estimates the memory usage of a map
func estimateMapMemory(m interface{}) int64 {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		return 0
	}

	// Basic map overhead (8 bytes per bucket on 64-bit systems)
	// Plus the size of the actual data
	mapSize := int64(v.Len() * 8)

	// Estimate size based on type - this is a rough approximation
	// In reality, you'd need deep reflection to get accurate sizes
	if v.Len() > 0 {
		// Assume average struct size of 200 bytes (rough estimate)
		mapSize += int64(v.Len() * 200)
	}

	return mapSize
}

// estimateSliceMemory estimates the memory usage of a slice
func estimateSliceMemory(s interface{}) int64 {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Slice {
		return 0
	}

	// Slice header is 24 bytes on 64-bit systems
	sliceSize := int64(24)

	// Add the size of the actual data
	if v.Len() > 0 {
		// Get the size of one element
		elemSize := int64(unsafe.Sizeof(v.Index(0).Interface()))
		// For pointers, estimate the pointed-to struct size
		if v.Type().Elem().Kind() == reflect.Ptr {
			elemSize = 200 // Rough estimate for struct size
		}
		sliceSize += int64(v.Len()) * elemSize
	}

	return sliceSize
}
