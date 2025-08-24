package sde

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Service provides in-memory access to EVE Online SDE data
type Service struct {
	agents          map[string]*Agent
	categories      map[string]*Category
	blueprints      map[string]*Blueprint
	marketGroups    map[string]*MarketGroup
	metaGroups      map[string]*MetaGroup
	npcCorporations map[string]*NPCCorporation
	typeIDs         map[string]*TypeID
	types           map[string]*Type
	typeMaterials   map[string][]*TypeMaterial
	loaded          bool
	loadMu          sync.Mutex // Only used during initial loading
	dataDir         string
}

// NewService creates a new SDE service instance
func NewService(dataDir string) *Service {
	return &Service{
		agents:          make(map[string]*Agent),
		categories:      make(map[string]*Category),
		blueprints:      make(map[string]*Blueprint),
		marketGroups:    make(map[string]*MarketGroup),
		metaGroups:      make(map[string]*MetaGroup),
		npcCorporations: make(map[string]*NPCCorporation),
		typeIDs:         make(map[string]*TypeID),
		types:           make(map[string]*Type),
		typeMaterials:   make(map[string][]*TypeMaterial),
		dataDir:         dataDir,
	}
}

// ensureLoaded loads SDE data if not already loaded
func (s *Service) ensureLoaded() error {
	// Fast path: data already loaded, no locking needed
	if s.loaded {
		return nil
	}

	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	// Double-check after acquiring lock (another goroutine might have loaded it)
	if s.loaded {
		return nil
	}

	if err := s.loadAgents(); err != nil {
		return fmt.Errorf("failed to load agents: %w", err)
	}

	if err := s.loadCategories(); err != nil {
		return fmt.Errorf("failed to load categories: %w", err)
	}

	if err := s.loadBlueprints(); err != nil {
		return fmt.Errorf("failed to load blueprints: %w", err)
	}

	if err := s.loadMarketGroups(); err != nil {
		return fmt.Errorf("failed to load market groups: %w", err)
	}

	if err := s.loadMetaGroups(); err != nil {
		return fmt.Errorf("failed to load meta groups: %w", err)
	}

	if err := s.loadNPCCorporations(); err != nil {
		return fmt.Errorf("failed to load NPC corporations: %w", err)
	}

	if err := s.loadTypeIDs(); err != nil {
		return fmt.Errorf("failed to load type IDs: %w", err)
	}

	if err := s.loadTypes(); err != nil {
		return fmt.Errorf("failed to load types: %w", err)
	}

	if err := s.loadTypeMaterials(); err != nil {
		return fmt.Errorf("failed to load type materials: %w", err)
	}

	s.loaded = true

	// Log memory usage after loading SDE data
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("SDE data loaded successfully",
		"agents_count", len(s.agents),
		"categories_count", len(s.categories),
		"blueprints_count", len(s.blueprints),
		"market_groups_count", len(s.marketGroups),
		"meta_groups_count", len(s.metaGroups),
		"npc_corporations_count", len(s.npcCorporations),
		"type_ids_count", len(s.typeIDs),
		"types_count", len(s.types),
		"type_materials_count", len(s.typeMaterials),
		"heap_size", formatBytes(m.HeapAlloc),
		"total_alloc", formatBytes(m.TotalAlloc),
	)

	return nil
}

// loadAgents loads agent data from JSON file
func (s *Service) loadAgents() error {
	filePath := filepath.Join(s.dataDir, "agents.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read agents file: %w", err)
	}

	var agents map[string]*Agent
	if err := json.Unmarshal(data, &agents); err != nil {
		return fmt.Errorf("failed to unmarshal agents: %w", err)
	}

	s.agents = agents
	return nil
}

// loadCategories loads category data from JSON file
func (s *Service) loadCategories() error {
	filePath := filepath.Join(s.dataDir, "categories.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read categories file: %w", err)
	}

	var categories map[string]*Category
	if err := json.Unmarshal(data, &categories); err != nil {
		return fmt.Errorf("failed to unmarshal categories: %w", err)
	}

	s.categories = categories
	return nil
}

// loadBlueprints loads blueprint data from JSON file
func (s *Service) loadBlueprints() error {
	filePath := filepath.Join(s.dataDir, "blueprints.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read blueprints file: %w", err)
	}

	var blueprints map[string]*Blueprint
	if err := json.Unmarshal(data, &blueprints); err != nil {
		return fmt.Errorf("failed to unmarshal blueprints: %w", err)
	}

	s.blueprints = blueprints
	return nil
}

// GetAgent retrieves an agent by ID
func (s *Service) GetAgent(id string) (*Agent, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	agent, exists := s.agents[id]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", id)
	}

	return agent, nil
}

// GetCategory retrieves a category by ID
func (s *Service) GetCategory(id string) (*Category, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	category, exists := s.categories[id]
	if !exists {
		return nil, fmt.Errorf("category %s not found", id)
	}

	return category, nil
}

// GetBlueprint retrieves a blueprint by ID
func (s *Service) GetBlueprint(id string) (*Blueprint, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	blueprint, exists := s.blueprints[id]
	if !exists {
		return nil, fmt.Errorf("blueprint %s not found", id)
	}

	return blueprint, nil
}

// GetAgentsByLocation returns all agents at a specific location
func (s *Service) GetAgentsByLocation(locationID int) ([]*Agent, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	var agents []*Agent
	for _, agent := range s.agents {
		if agent.LocationID == locationID {
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

// GetPublishedCategories returns all published categories
func (s *Service) GetPublishedCategories() (map[string]*Category, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	published := make(map[string]*Category)
	for id, category := range s.categories {
		if category.Published {
			published[id] = category
		}
	}

	return published, nil
}

// GetAllAgents returns all agents
func (s *Service) GetAllAgents() (map[string]*Agent, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	// Return a copy to prevent external modification
	agents := make(map[string]*Agent, len(s.agents))
	for id, agent := range s.agents {
		agents[id] = agent
	}

	return agents, nil
}

// GetAllCategories returns all categories
func (s *Service) GetAllCategories() (map[string]*Category, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	// Return a copy to prevent external modification
	categories := make(map[string]*Category, len(s.categories))
	for id, category := range s.categories {
		categories[id] = category
	}

	return categories, nil
}

// GetAllBlueprints returns all blueprints
func (s *Service) GetAllBlueprints() (map[string]*Blueprint, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	// Return a copy to prevent external modification
	blueprints := make(map[string]*Blueprint, len(s.blueprints))
	for id, blueprint := range s.blueprints {
		blueprints[id] = blueprint
	}

	return blueprints, nil
}

// IsLoaded returns whether SDE data has been loaded
func (s *Service) IsLoaded() bool {
	return s.loaded
}

// loadMarketGroups loads market group data from JSON file
func (s *Service) loadMarketGroups() error {
	filePath := filepath.Join(s.dataDir, "marketGroups.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read market groups file: %w", err)
	}

	var marketGroups map[string]*MarketGroup
	if err := json.Unmarshal(data, &marketGroups); err != nil {
		return fmt.Errorf("failed to unmarshal market groups: %w", err)
	}

	s.marketGroups = marketGroups
	return nil
}

// loadMetaGroups loads meta group data from JSON file
func (s *Service) loadMetaGroups() error {
	filePath := filepath.Join(s.dataDir, "metaGroups.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read meta groups file: %w", err)
	}

	var metaGroups map[string]*MetaGroup
	if err := json.Unmarshal(data, &metaGroups); err != nil {
		return fmt.Errorf("failed to unmarshal meta groups: %w", err)
	}

	s.metaGroups = metaGroups
	return nil
}

// loadNPCCorporations loads NPC corporation data from JSON file
func (s *Service) loadNPCCorporations() error {
	filePath := filepath.Join(s.dataDir, "npcCorporations.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read NPC corporations file: %w", err)
	}

	var npcCorporations map[string]*NPCCorporation
	if err := json.Unmarshal(data, &npcCorporations); err != nil {
		return fmt.Errorf("failed to unmarshal NPC corporations: %w", err)
	}

	s.npcCorporations = npcCorporations
	return nil
}

// loadTypeIDs creates lightweight TypeID data from already loaded types
func (s *Service) loadTypeIDs() error {
	// TypeIDs will be populated from types data after types are loaded
	// This method is kept for consistency but actual loading happens in loadTypes
	return nil
}

// loadTypes loads type data from JSON file and creates TypeID data
func (s *Service) loadTypes() error {
	filePath := filepath.Join(s.dataDir, "types.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read types file: %w", err)
	}

	var types map[string]*Type
	if err := json.Unmarshal(data, &types); err != nil {
		return fmt.Errorf("failed to unmarshal types: %w", err)
	}

	s.types = types

	// Create lightweight TypeID data from loaded types
	typeIDs := make(map[string]*TypeID, len(types))
	for id, fullType := range types {
		typeIDs[id] = &TypeID{
			Name:        fullType.Name,
			Description: fullType.Description,
			GroupID:     fullType.GroupID,
			Published:   fullType.Published,
		}
	}
	s.typeIDs = typeIDs

	return nil
}

// loadTypeMaterials loads type material data from JSON file
func (s *Service) loadTypeMaterials() error {
	filePath := filepath.Join(s.dataDir, "typeMaterials.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read type materials file: %w", err)
	}

	var rawTypeMaterials map[string]*TypeMaterialData
	if err := json.Unmarshal(data, &rawTypeMaterials); err != nil {
		return fmt.Errorf("failed to unmarshal type materials: %w", err)
	}

	// Convert from TypeMaterialData to []*TypeMaterial
	typeMaterials := make(map[string][]*TypeMaterial, len(rawTypeMaterials))
	for typeID, materialData := range rawTypeMaterials {
		typeMaterials[typeID] = materialData.Materials
	}

	s.typeMaterials = typeMaterials
	return nil
}

// GetMarketGroup retrieves a market group by ID
func (s *Service) GetMarketGroup(id string) (*MarketGroup, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	marketGroup, exists := s.marketGroups[id]
	if !exists {
		return nil, fmt.Errorf("market group %s not found", id)
	}

	return marketGroup, nil
}

// GetMetaGroup retrieves a meta group by ID
func (s *Service) GetMetaGroup(id string) (*MetaGroup, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	metaGroup, exists := s.metaGroups[id]
	if !exists {
		return nil, fmt.Errorf("meta group %s not found", id)
	}

	return metaGroup, nil
}

// GetNPCCorporation retrieves an NPC corporation by ID
func (s *Service) GetNPCCorporation(id string) (*NPCCorporation, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	npcCorp, exists := s.npcCorporations[id]
	if !exists {
		return nil, fmt.Errorf("NPC corporation %s not found", id)
	}

	return npcCorp, nil
}

// GetTypeID retrieves a type ID by ID
func (s *Service) GetTypeID(id string) (*TypeID, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	typeID, exists := s.typeIDs[id]
	if !exists {
		return nil, fmt.Errorf("type ID %s not found", id)
	}

	return typeID, nil
}

// GetType retrieves a type by ID
func (s *Service) GetType(id string) (*Type, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	typeData, exists := s.types[id]
	if !exists {
		return nil, fmt.Errorf("type %s not found", id)
	}

	return typeData, nil
}

// GetTypeMaterials retrieves type materials by type ID
func (s *Service) GetTypeMaterials(typeID string) ([]*TypeMaterial, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	materials, exists := s.typeMaterials[typeID]
	if !exists {
		return nil, fmt.Errorf("type materials for %s not found", typeID)
	}

	return materials, nil
}

// GetAllMarketGroups returns all market groups
func (s *Service) GetAllMarketGroups() (map[string]*MarketGroup, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	marketGroups := make(map[string]*MarketGroup, len(s.marketGroups))
	for id, marketGroup := range s.marketGroups {
		marketGroups[id] = marketGroup
	}

	return marketGroups, nil
}

// GetAllMetaGroups returns all meta groups
func (s *Service) GetAllMetaGroups() (map[string]*MetaGroup, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	metaGroups := make(map[string]*MetaGroup, len(s.metaGroups))
	for id, metaGroup := range s.metaGroups {
		metaGroups[id] = metaGroup
	}

	return metaGroups, nil
}

// GetAllNPCCorporations returns all NPC corporations
func (s *Service) GetAllNPCCorporations() (map[string]*NPCCorporation, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	npcCorporations := make(map[string]*NPCCorporation, len(s.npcCorporations))
	for id, npcCorp := range s.npcCorporations {
		npcCorporations[id] = npcCorp
	}

	return npcCorporations, nil
}

// GetAllTypeIDs returns all type IDs
func (s *Service) GetAllTypeIDs() (map[string]*TypeID, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	typeIDs := make(map[string]*TypeID, len(s.typeIDs))
	for id, typeID := range s.typeIDs {
		typeIDs[id] = typeID
	}

	return typeIDs, nil
}

// GetAllTypes returns all types
func (s *Service) GetAllTypes() (map[string]*Type, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	types := make(map[string]*Type, len(s.types))
	for id, typeData := range s.types {
		types[id] = typeData
	}

	return types, nil
}

// GetPublishedTypes returns all published types
func (s *Service) GetPublishedTypes() (map[string]*Type, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	published := make(map[string]*Type)
	for id, typeData := range s.types {
		if typeData.Published {
			published[id] = typeData
		}
	}

	return published, nil
}

// GetTypesByGroupID returns all types that belong to a specific group
func (s *Service) GetTypesByGroupID(groupID int) ([]*Type, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	var types []*Type
	for _, typeData := range s.types {
		if typeData.GroupID == groupID {
			types = append(types, typeData)
		}
	}

	return types, nil
}

// GetNPCCorporationsByFaction returns all NPC corporations that belong to a specific faction
func (s *Service) GetNPCCorporationsByFaction(factionID int) ([]*NPCCorporation, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	var corporations []*NPCCorporation
	for _, corp := range s.npcCorporations {
		if corp.FactionID == factionID {
			corporations = append(corporations, corp)
		}
	}

	return corporations, nil
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
