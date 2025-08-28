package sde

// SDEService defines the interface for accessing EVE Online SDE data
type SDEService interface {
	// Agent operations
	GetAgent(id string) (*Agent, error)
	GetAgentsByLocation(locationID int) ([]*Agent, error)
	GetAllAgents() (map[string]*Agent, error)

	// Category operations
	GetCategory(id string) (*Category, error)
	GetPublishedCategories() (map[string]*Category, error)
	GetAllCategories() (map[string]*Category, error)

	// Blueprint operations
	GetBlueprint(id string) (*Blueprint, error)
	GetAllBlueprints() (map[string]*Blueprint, error)

	// Market group operations
	GetMarketGroup(id string) (*MarketGroup, error)
	GetAllMarketGroups() (map[string]*MarketGroup, error)

	// Meta group operations
	GetMetaGroup(id string) (*MetaGroup, error)
	GetAllMetaGroups() (map[string]*MetaGroup, error)

	// NPC corporation operations
	GetNPCCorporation(id string) (*NPCCorporation, error)
	GetAllNPCCorporations() (map[string]*NPCCorporation, error)
	GetNPCCorporationsByFaction(factionID int) ([]*NPCCorporation, error)

	// Type ID operations
	GetTypeID(id string) (*TypeID, error)
	GetAllTypeIDs() (map[string]*TypeID, error)

	// Type operations
	GetType(id string) (*Type, error)
	GetAllTypes() (map[string]*Type, error)
	GetPublishedTypes() (map[string]*Type, error)
	GetTypesByGroupID(groupID int) ([]*Type, error)

	// Type material operations
	GetTypeMaterials(typeID string) ([]*TypeMaterial, error)

	// Race operations
	GetRace(id string) (*Race, error)
	GetAllRaces() (map[string]*Race, error)

	// Faction operations
	GetFaction(id string) (*Faction, error)
	GetAllFactions() (map[string]*Faction, error)

	// Bloodline operations
	GetBloodline(id string) (*Bloodline, error)
	GetAllBloodlines() (map[string]*Bloodline, error)

	// Group operations
	GetGroup(id string) (*Group, error)
	GetAllGroups() (map[string]*Group, error)

	// Dogma attribute operations
	GetDogmaAttribute(id string) (*DogmaAttribute, error)
	GetAllDogmaAttributes() (map[string]*DogmaAttribute, error)

	// Ancestry operations
	GetAncestry(id string) (*Ancestry, error)
	GetAllAncestries() (map[string]*Ancestry, error)

	// Certificate operations
	GetCertificate(id string) (*Certificate, error)
	GetAllCertificates() (map[string]*Certificate, error)

	// Character attribute operations
	GetCharacterAttribute(id string) (*CharacterAttribute, error)
	GetAllCharacterAttributes() (map[string]*CharacterAttribute, error)

	// Skin operations
	GetSkin(id string) (*Skin, error)
	GetAllSkins() (map[string]*Skin, error)

	// Station operations
	GetStaStation(stationID int) (*StaStation, error)
	GetAllStaStations() ([]*StaStation, error)
	GetStaStationsBySolarSystem(solarSystemID int) ([]*StaStation, error)

	// Dogma effect operations
	GetDogmaEffect(id string) (*DogmaEffect, error)
	GetAllDogmaEffects() (map[string]*DogmaEffect, error)

	// Icon ID operations
	GetIconID(id string) (*IconID, error)
	GetAllIconIDs() (map[string]*IconID, error)

	// Graphic ID operations
	GetGraphicID(id string) (*GraphicID, error)
	GetAllGraphicIDs() (map[string]*GraphicID, error)

	// Type dogma operations
	GetTypeDogma(typeID string) (*TypeDogma, error)
	GetAllTypeDogma() (map[string]*TypeDogma, error)

	// Inventory flag operations
	GetInvFlag(flagID int) (*InvFlag, error)
	GetAllInvFlags() ([]*InvFlag, error)

	// Station service operations
	GetStationService(id string) (*StationService, error)
	GetAllStationServices() (map[string]*StationService, error)

	// Station operation operations
	GetStationOperation(id string) (*StationOperation, error)
	GetAllStationOperations() (map[string]*StationOperation, error)

	// Research agent operations
	GetResearchAgent(id string) (*ResearchAgent, error)
	GetAllResearchAgents() (map[string]*ResearchAgent, error)

	// Agent in space operations
	GetAgentInSpace(id string) (*AgentInSpace, error)
	GetAllAgentsInSpace() (map[string]*AgentInSpace, error)

	// Contraband type operations
	GetContrabandType(id string) (*ContrabandType, error)
	GetAllContrabandTypes() (map[string]*ContrabandType, error)

	// Corporation activity operations
	GetCorporationActivity(id string) (*CorporationActivity, error)
	GetAllCorporationActivities() (map[string]*CorporationActivity, error)

	// Inventory item operations
	GetInvItem(itemID int) (*InvItem, error)
	GetAllInvItems() ([]*InvItem, error)

	// NPC corporation division operations
	GetNPCCorporationDivision(id string) (*NPCCorporationDivision, error)
	GetAllNPCCorporationDivisions() (map[string]*NPCCorporationDivision, error)

	// Control tower resource operations
	GetControlTowerResources(id string) (*ControlTowerResources, error)
	GetAllControlTowerResources() (map[string]*ControlTowerResources, error)

	// Dogma attribute category operations
	GetDogmaAttributeCategory(id string) (*DogmaAttributeCategory, error)
	GetAllDogmaAttributeCategories() (map[string]*DogmaAttributeCategory, error)

	// Inventory name operations
	GetInvName(itemID int) (*InvName, error)
	GetAllInvNames() ([]*InvName, error)

	// Inventory position operations
	GetInvPosition(itemID int) (*InvPosition, error)
	GetAllInvPositions() ([]*InvPosition, error)

	// Inventory unique name operations
	GetInvUniqueName(itemID int) (*InvUniqueName, error)
	GetAllInvUniqueNames() ([]*InvUniqueName, error)

	// Planet resource operations
	GetPlanetResource(id string) (*PlanetResource, error)
	GetAllPlanetResources() (map[string]*PlanetResource, error)

	// Planet schematic operations
	GetPlanetSchematic(id string) (*PlanetSchematic, error)
	GetAllPlanetSchematics() (map[string]*PlanetSchematic, error)

	// Skin license operations
	GetSkinLicense(id string) (*SkinLicense, error)
	GetAllSkinLicenses() (map[string]*SkinLicense, error)

	// Skin material operations
	GetSkinMaterial(id string) (*SkinMaterial, error)
	GetAllSkinMaterials() (map[string]*SkinMaterial, error)

	// Sovereignty upgrade operations
	GetSovereigntyUpgrade(id string) (*SovereigntyUpgrade, error)
	GetAllSovereigntyUpgrades() (map[string]*SovereigntyUpgrade, error)

	// Translation language operations
	GetTranslationLanguage(code string) (*TranslationLanguage, error)
	GetAllTranslationLanguages() (map[string]*TranslationLanguage, error)

	// Service status
	IsLoaded() bool

	// Memory inspection and management operations
	GetLoadedDataTypes() []string
	GetDataTypeStats(dataType string) DataTypeStats
	GetTotalMemoryUsage() int64
	GetLoadStatus() map[string]DataTypeStatus
	ReloadDataType(dataType string) error
	ReloadAll() error
}
