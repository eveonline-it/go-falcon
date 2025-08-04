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

	// Service status
	IsLoaded() bool
}