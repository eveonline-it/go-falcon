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

	// Service status
	IsLoaded() bool
}
