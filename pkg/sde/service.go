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
	agents                   map[string]*Agent
	categories               map[string]*Category
	blueprints               map[string]*Blueprint
	marketGroups             map[string]*MarketGroup
	metaGroups               map[string]*MetaGroup
	npcCorporations          map[string]*NPCCorporation
	typeIDs                  map[string]*TypeID
	types                    map[string]*Type
	typeMaterials            map[string][]*TypeMaterial
	races                    map[string]*Race
	factions                 map[string]*Faction
	bloodlines               map[string]*Bloodline
	groups                   map[string]*Group
	dogmaAttributes          map[string]*DogmaAttribute
	ancestries               map[string]*Ancestry
	certificates             map[string]*Certificate
	characterAttributes      map[string]*CharacterAttribute
	skins                    map[string]*Skin
	staStations              []*StaStation // Array since stations use array format
	dogmaEffects             map[string]*DogmaEffect
	iconIDs                  map[string]*IconID
	graphicIDs               map[string]*GraphicID
	typeDogma                map[string]*TypeDogma
	invFlags                 []*InvFlag // Array format
	stationServices          map[string]*StationService
	stationOperations        map[string]*StationOperation
	researchAgents           map[string]*ResearchAgent
	agentsInSpace            map[string]*AgentInSpace
	contrabandTypes          map[string]*ContrabandType
	corporationActivities    map[string]*CorporationActivity
	invItems                 []*InvItem // Array format
	npcCorporationDivisions  map[string]*NPCCorporationDivision
	controlTowerResources    map[string]*ControlTowerResources
	dogmaAttributeCategories map[string]*DogmaAttributeCategory
	invNames                 []*InvName       // Array format
	invPositions             []*InvPosition   // Array format
	invUniqueNames           []*InvUniqueName // Array format
	planetResources          map[string]*PlanetResource
	planetSchematics         map[string]*PlanetSchematic
	skinLicenses             map[string]*SkinLicense
	skinMaterials            map[string]*SkinMaterial
	sovereigntyUpgrades      map[string]*SovereigntyUpgrade
	translationLanguages     map[string]*TranslationLanguage
	loaded                   bool
	loadMu                   sync.Mutex // Only used during initial loading
	dataDir                  string
}

// NewService creates a new SDE service instance
func NewService(dataDir string) *Service {
	return &Service{
		agents:                   make(map[string]*Agent),
		categories:               make(map[string]*Category),
		blueprints:               make(map[string]*Blueprint),
		marketGroups:             make(map[string]*MarketGroup),
		metaGroups:               make(map[string]*MetaGroup),
		npcCorporations:          make(map[string]*NPCCorporation),
		typeIDs:                  make(map[string]*TypeID),
		types:                    make(map[string]*Type),
		typeMaterials:            make(map[string][]*TypeMaterial),
		races:                    make(map[string]*Race),
		factions:                 make(map[string]*Faction),
		bloodlines:               make(map[string]*Bloodline),
		groups:                   make(map[string]*Group),
		dogmaAttributes:          make(map[string]*DogmaAttribute),
		ancestries:               make(map[string]*Ancestry),
		certificates:             make(map[string]*Certificate),
		characterAttributes:      make(map[string]*CharacterAttribute),
		skins:                    make(map[string]*Skin),
		staStations:              []*StaStation{},
		dogmaEffects:             make(map[string]*DogmaEffect),
		iconIDs:                  make(map[string]*IconID),
		graphicIDs:               make(map[string]*GraphicID),
		typeDogma:                make(map[string]*TypeDogma),
		invFlags:                 []*InvFlag{},
		stationServices:          make(map[string]*StationService),
		stationOperations:        make(map[string]*StationOperation),
		researchAgents:           make(map[string]*ResearchAgent),
		agentsInSpace:            make(map[string]*AgentInSpace),
		contrabandTypes:          make(map[string]*ContrabandType),
		corporationActivities:    make(map[string]*CorporationActivity),
		invItems:                 []*InvItem{},
		npcCorporationDivisions:  make(map[string]*NPCCorporationDivision),
		controlTowerResources:    make(map[string]*ControlTowerResources),
		dogmaAttributeCategories: make(map[string]*DogmaAttributeCategory),
		invNames:                 []*InvName{},
		invPositions:             []*InvPosition{},
		invUniqueNames:           []*InvUniqueName{},
		planetResources:          make(map[string]*PlanetResource),
		planetSchematics:         make(map[string]*PlanetSchematic),
		skinLicenses:             make(map[string]*SkinLicense),
		skinMaterials:            make(map[string]*SkinMaterial),
		sovereigntyUpgrades:      make(map[string]*SovereigntyUpgrade),
		translationLanguages:     make(map[string]*TranslationLanguage),
		dataDir:                  dataDir,
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

	if err := s.loadRaces(); err != nil {
		return fmt.Errorf("failed to load races: %w", err)
	}

	if err := s.loadFactions(); err != nil {
		return fmt.Errorf("failed to load factions: %w", err)
	}

	if err := s.loadBloodlines(); err != nil {
		return fmt.Errorf("failed to load bloodlines: %w", err)
	}

	if err := s.loadGroups(); err != nil {
		return fmt.Errorf("failed to load groups: %w", err)
	}

	if err := s.loadDogmaAttributes(); err != nil {
		return fmt.Errorf("failed to load dogma attributes: %w", err)
	}

	if err := s.loadAncestries(); err != nil {
		return fmt.Errorf("failed to load ancestries: %w", err)
	}

	if err := s.loadCertificates(); err != nil {
		return fmt.Errorf("failed to load certificates: %w", err)
	}

	if err := s.loadCharacterAttributes(); err != nil {
		return fmt.Errorf("failed to load character attributes: %w", err)
	}

	if err := s.loadSkins(); err != nil {
		return fmt.Errorf("failed to load skins: %w", err)
	}

	if err := s.loadStaStations(); err != nil {
		return fmt.Errorf("failed to load stations: %w", err)
	}

	if err := s.loadDogmaEffects(); err != nil {
		return fmt.Errorf("failed to load dogma effects: %w", err)
	}

	if err := s.loadIconIDs(); err != nil {
		return fmt.Errorf("failed to load icon IDs: %w", err)
	}

	if err := s.loadGraphicIDs(); err != nil {
		return fmt.Errorf("failed to load graphic IDs: %w", err)
	}

	if err := s.loadTypeDogma(); err != nil {
		return fmt.Errorf("failed to load type dogma: %w", err)
	}

	if err := s.loadInvFlags(); err != nil {
		return fmt.Errorf("failed to load inventory flags: %w", err)
	}

	if err := s.loadStationServices(); err != nil {
		return fmt.Errorf("failed to load station services: %w", err)
	}

	if err := s.loadStationOperations(); err != nil {
		return fmt.Errorf("failed to load station operations: %w", err)
	}

	if err := s.loadResearchAgents(); err != nil {
		return fmt.Errorf("failed to load research agents: %w", err)
	}

	if err := s.loadAgentsInSpace(); err != nil {
		return fmt.Errorf("failed to load agents in space: %w", err)
	}

	if err := s.loadContrabandTypes(); err != nil {
		return fmt.Errorf("failed to load contraband types: %w", err)
	}

	if err := s.loadCorporationActivities(); err != nil {
		return fmt.Errorf("failed to load corporation activities: %w", err)
	}

	if err := s.loadInvItems(); err != nil {
		return fmt.Errorf("failed to load inventory items: %w", err)
	}

	if err := s.loadNPCCorporationDivisions(); err != nil {
		return fmt.Errorf("failed to load NPC corporation divisions: %w", err)
	}

	if err := s.loadControlTowerResources(); err != nil {
		return fmt.Errorf("failed to load control tower resources: %w", err)
	}

	if err := s.loadDogmaAttributeCategories(); err != nil {
		return fmt.Errorf("failed to load dogma attribute categories: %w", err)
	}

	if err := s.loadInvNames(); err != nil {
		return fmt.Errorf("failed to load inventory names: %w", err)
	}

	if err := s.loadInvPositions(); err != nil {
		return fmt.Errorf("failed to load inventory positions: %w", err)
	}

	if err := s.loadInvUniqueNames(); err != nil {
		return fmt.Errorf("failed to load inventory unique names: %w", err)
	}

	if err := s.loadPlanetResources(); err != nil {
		return fmt.Errorf("failed to load planet resources: %w", err)
	}

	if err := s.loadPlanetSchematics(); err != nil {
		return fmt.Errorf("failed to load planet schematics: %w", err)
	}

	if err := s.loadSkinLicenses(); err != nil {
		return fmt.Errorf("failed to load skin licenses: %w", err)
	}

	if err := s.loadSkinMaterials(); err != nil {
		return fmt.Errorf("failed to load skin materials: %w", err)
	}

	if err := s.loadSovereigntyUpgrades(); err != nil {
		return fmt.Errorf("failed to load sovereignty upgrades: %w", err)
	}

	if err := s.loadTranslationLanguages(); err != nil {
		return fmt.Errorf("failed to load translation languages: %w", err)
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
		"races_count", len(s.races),
		"factions_count", len(s.factions),
		"bloodlines_count", len(s.bloodlines),
		"groups_count", len(s.groups),
		"dogma_attributes_count", len(s.dogmaAttributes),
		"ancestries_count", len(s.ancestries),
		"certificates_count", len(s.certificates),
		"character_attributes_count", len(s.characterAttributes),
		"skins_count", len(s.skins),
		"stations_count", len(s.staStations),
		"dogma_effects_count", len(s.dogmaEffects),
		"icon_ids_count", len(s.iconIDs),
		"graphic_ids_count", len(s.graphicIDs),
		"type_dogma_count", len(s.typeDogma),
		"inv_flags_count", len(s.invFlags),
		"station_services_count", len(s.stationServices),
		"station_operations_count", len(s.stationOperations),
		"research_agents_count", len(s.researchAgents),
		"agents_in_space_count", len(s.agentsInSpace),
		"contraband_types_count", len(s.contrabandTypes),
		"corporation_activities_count", len(s.corporationActivities),
		"inv_items_count", len(s.invItems),
		"npc_corporation_divisions_count", len(s.npcCorporationDivisions),
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

// loadRaces loads race data from JSON file
func (s *Service) loadRaces() error {
	filePath := filepath.Join(s.dataDir, "races.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read races file: %w", err)
	}

	var races map[string]*Race
	if err := json.Unmarshal(data, &races); err != nil {
		return fmt.Errorf("failed to unmarshal races: %w", err)
	}

	s.races = races
	return nil
}

// GetRace retrieves a race by ID
func (s *Service) GetRace(id string) (*Race, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	race, exists := s.races[id]
	if !exists {
		return nil, fmt.Errorf("race %s not found", id)
	}

	return race, nil
}

// loadFactions loads faction data from JSON file
func (s *Service) loadFactions() error {
	filePath := filepath.Join(s.dataDir, "factions.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read factions file: %w", err)
	}

	var factions map[string]*Faction
	if err := json.Unmarshal(data, &factions); err != nil {
		return fmt.Errorf("failed to unmarshal factions: %w", err)
	}

	s.factions = factions
	return nil
}

// GetFaction retrieves a faction by ID
func (s *Service) GetFaction(id string) (*Faction, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	faction, exists := s.factions[id]
	if !exists {
		return nil, fmt.Errorf("faction %s not found", id)
	}

	return faction, nil
}

// loadBloodlines loads bloodline data from JSON file
func (s *Service) loadBloodlines() error {
	filePath := filepath.Join(s.dataDir, "bloodlines.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read bloodlines file: %w", err)
	}

	var bloodlines map[string]*Bloodline
	if err := json.Unmarshal(data, &bloodlines); err != nil {
		return fmt.Errorf("failed to unmarshal bloodlines: %w", err)
	}

	s.bloodlines = bloodlines
	return nil
}

// GetBloodline retrieves a bloodline by ID
func (s *Service) GetBloodline(id string) (*Bloodline, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	bloodline, exists := s.bloodlines[id]
	if !exists {
		return nil, fmt.Errorf("bloodline %s not found", id)
	}

	return bloodline, nil
}

// loadGroups loads group data from JSON file
func (s *Service) loadGroups() error {
	filePath := filepath.Join(s.dataDir, "groups.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read groups file: %w", err)
	}

	var groups map[string]*Group
	if err := json.Unmarshal(data, &groups); err != nil {
		return fmt.Errorf("failed to unmarshal groups: %w", err)
	}

	s.groups = groups
	return nil
}

// GetGroup retrieves a group by ID
func (s *Service) GetGroup(id string) (*Group, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	group, exists := s.groups[id]
	if !exists {
		return nil, fmt.Errorf("group %s not found", id)
	}

	return group, nil
}

// loadDogmaAttributes loads dogma attribute data from JSON file
func (s *Service) loadDogmaAttributes() error {
	filePath := filepath.Join(s.dataDir, "dogmaAttributes.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read dogma attributes file: %w", err)
	}

	var dogmaAttributes map[string]*DogmaAttribute
	if err := json.Unmarshal(data, &dogmaAttributes); err != nil {
		return fmt.Errorf("failed to unmarshal dogma attributes: %w", err)
	}

	s.dogmaAttributes = dogmaAttributes
	return nil
}

// GetDogmaAttribute retrieves a dogma attribute by ID
func (s *Service) GetDogmaAttribute(id string) (*DogmaAttribute, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	dogmaAttribute, exists := s.dogmaAttributes[id]
	if !exists {
		return nil, fmt.Errorf("dogma attribute %s not found", id)
	}

	return dogmaAttribute, nil
}

// loadAncestries loads ancestry data from JSON file
func (s *Service) loadAncestries() error {
	filePath := filepath.Join(s.dataDir, "ancestries.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read ancestries file: %w", err)
	}

	var ancestries map[string]*Ancestry
	if err := json.Unmarshal(data, &ancestries); err != nil {
		return fmt.Errorf("failed to unmarshal ancestries: %w", err)
	}

	s.ancestries = ancestries
	return nil
}

// GetAncestry retrieves an ancestry by ID
func (s *Service) GetAncestry(id string) (*Ancestry, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	ancestry, exists := s.ancestries[id]
	if !exists {
		return nil, fmt.Errorf("ancestry %s not found", id)
	}

	return ancestry, nil
}

// loadCertificates loads certificate data from JSON file
func (s *Service) loadCertificates() error {
	filePath := filepath.Join(s.dataDir, "certificates.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read certificates file: %w", err)
	}

	var certificates map[string]*Certificate
	if err := json.Unmarshal(data, &certificates); err != nil {
		return fmt.Errorf("failed to unmarshal certificates: %w", err)
	}

	s.certificates = certificates
	return nil
}

// GetCertificate retrieves a certificate by ID
func (s *Service) GetCertificate(id string) (*Certificate, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	certificate, exists := s.certificates[id]
	if !exists {
		return nil, fmt.Errorf("certificate %s not found", id)
	}

	return certificate, nil
}

// loadCharacterAttributes loads character attribute data from JSON file
func (s *Service) loadCharacterAttributes() error {
	filePath := filepath.Join(s.dataDir, "characterAttributes.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read character attributes file: %w", err)
	}

	var characterAttributes map[string]*CharacterAttribute
	if err := json.Unmarshal(data, &characterAttributes); err != nil {
		return fmt.Errorf("failed to unmarshal character attributes: %w", err)
	}

	s.characterAttributes = characterAttributes
	return nil
}

// loadSkins loads skin data from JSON file
func (s *Service) loadSkins() error {
	filePath := filepath.Join(s.dataDir, "skins.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read skins file: %w", err)
	}

	var skins map[string]*Skin
	if err := json.Unmarshal(data, &skins); err != nil {
		return fmt.Errorf("failed to unmarshal skins: %w", err)
	}

	s.skins = skins
	return nil
}

// loadStaStations loads station data from JSON file
func (s *Service) loadStaStations() error {
	filePath := filepath.Join(s.dataDir, "staStations.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read stations file: %w", err)
	}

	var stations []*StaStation
	if err := json.Unmarshal(data, &stations); err != nil {
		return fmt.Errorf("failed to unmarshal stations: %w", err)
	}

	s.staStations = stations
	return nil
}

// loadDogmaEffects loads dogma effects data from JSON file
func (s *Service) loadDogmaEffects() error {
	filePath := filepath.Join(s.dataDir, "dogmaEffects.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read dogma effects file: %w", err)
	}

	var dogmaEffects map[string]*DogmaEffect
	if err := json.Unmarshal(data, &dogmaEffects); err != nil {
		return fmt.Errorf("failed to unmarshal dogma effects: %w", err)
	}

	s.dogmaEffects = dogmaEffects
	return nil
}

// loadIconIDs loads icon ID data from JSON file
func (s *Service) loadIconIDs() error {
	filePath := filepath.Join(s.dataDir, "iconIDs.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read icon IDs file: %w", err)
	}

	var iconIDs map[string]*IconID
	if err := json.Unmarshal(data, &iconIDs); err != nil {
		return fmt.Errorf("failed to unmarshal icon IDs: %w", err)
	}

	s.iconIDs = iconIDs
	return nil
}

// loadGraphicIDs loads graphic ID data from JSON file
func (s *Service) loadGraphicIDs() error {
	filePath := filepath.Join(s.dataDir, "graphicIDs.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read graphic IDs file: %w", err)
	}

	var graphicIDs map[string]*GraphicID
	if err := json.Unmarshal(data, &graphicIDs); err != nil {
		return fmt.Errorf("failed to unmarshal graphic IDs: %w", err)
	}

	s.graphicIDs = graphicIDs
	return nil
}

// loadTypeDogma loads type dogma data from JSON file
func (s *Service) loadTypeDogma() error {
	filePath := filepath.Join(s.dataDir, "typeDogma.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read type dogma file: %w", err)
	}

	var typeDogma map[string]*TypeDogma
	if err := json.Unmarshal(data, &typeDogma); err != nil {
		return fmt.Errorf("failed to unmarshal type dogma: %w", err)
	}

	s.typeDogma = typeDogma
	return nil
}

// loadInvFlags loads inventory flags data from JSON file
func (s *Service) loadInvFlags() error {
	filePath := filepath.Join(s.dataDir, "invFlags.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read inventory flags file: %w", err)
	}

	var invFlags []*InvFlag
	if err := json.Unmarshal(data, &invFlags); err != nil {
		return fmt.Errorf("failed to unmarshal inventory flags: %w", err)
	}

	s.invFlags = invFlags
	return nil
}

// loadStationServices loads station services data from JSON file
func (s *Service) loadStationServices() error {
	filePath := filepath.Join(s.dataDir, "stationServices.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read station services file: %w", err)
	}

	var stationServices map[string]*StationService
	if err := json.Unmarshal(data, &stationServices); err != nil {
		return fmt.Errorf("failed to unmarshal station services: %w", err)
	}

	s.stationServices = stationServices
	return nil
}

// loadStationOperations loads station operations data from JSON file
func (s *Service) loadStationOperations() error {
	filePath := filepath.Join(s.dataDir, "stationOperations.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read station operations file: %w", err)
	}

	var stationOperations map[string]*StationOperation
	if err := json.Unmarshal(data, &stationOperations); err != nil {
		return fmt.Errorf("failed to unmarshal station operations: %w", err)
	}

	s.stationOperations = stationOperations
	return nil
}

// loadResearchAgents loads research agents data from JSON file
func (s *Service) loadResearchAgents() error {
	filePath := filepath.Join(s.dataDir, "researchAgents.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read research agents file: %w", err)
	}

	var researchAgents map[string]*ResearchAgent
	if err := json.Unmarshal(data, &researchAgents); err != nil {
		return fmt.Errorf("failed to unmarshal research agents: %w", err)
	}

	s.researchAgents = researchAgents
	return nil
}

// loadAgentsInSpace loads agents in space data from JSON file
func (s *Service) loadAgentsInSpace() error {
	filePath := filepath.Join(s.dataDir, "agentsInSpace.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read agents in space file: %w", err)
	}

	var agentsInSpace map[string]*AgentInSpace
	if err := json.Unmarshal(data, &agentsInSpace); err != nil {
		return fmt.Errorf("failed to unmarshal agents in space: %w", err)
	}

	s.agentsInSpace = agentsInSpace
	return nil
}

// loadContrabandTypes loads contraband types data from JSON file
func (s *Service) loadContrabandTypes() error {
	filePath := filepath.Join(s.dataDir, "contrabandTypes.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read contraband types file: %w", err)
	}

	var contrabandTypes map[string]*ContrabandType
	if err := json.Unmarshal(data, &contrabandTypes); err != nil {
		return fmt.Errorf("failed to unmarshal contraband types: %w", err)
	}

	s.contrabandTypes = contrabandTypes
	return nil
}

// loadCorporationActivities loads corporation activities data from JSON file
func (s *Service) loadCorporationActivities() error {
	filePath := filepath.Join(s.dataDir, "corporationActivities.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read corporation activities file: %w", err)
	}

	var corporationActivities map[string]*CorporationActivity
	if err := json.Unmarshal(data, &corporationActivities); err != nil {
		return fmt.Errorf("failed to unmarshal corporation activities: %w", err)
	}

	s.corporationActivities = corporationActivities
	return nil
}

// loadInvItems loads inventory items data from JSON file
func (s *Service) loadInvItems() error {
	filePath := filepath.Join(s.dataDir, "invItems.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read inventory items file: %w", err)
	}

	var invItems []*InvItem
	if err := json.Unmarshal(data, &invItems); err != nil {
		return fmt.Errorf("failed to unmarshal inventory items: %w", err)
	}

	s.invItems = invItems
	return nil
}

// loadNPCCorporationDivisions loads NPC corporation divisions data from JSON file
func (s *Service) loadNPCCorporationDivisions() error {
	filePath := filepath.Join(s.dataDir, "npcCorporationDivisions.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read NPC corporation divisions file: %w", err)
	}

	var npcCorporationDivisions map[string]*NPCCorporationDivision
	if err := json.Unmarshal(data, &npcCorporationDivisions); err != nil {
		return fmt.Errorf("failed to unmarshal NPC corporation divisions: %w", err)
	}

	s.npcCorporationDivisions = npcCorporationDivisions
	return nil
}

// loadControlTowerResources loads control tower resources data from JSON file
func (s *Service) loadControlTowerResources() error {
	filePath := filepath.Join(s.dataDir, "controlTowerResources.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read control tower resources file: %w", err)
	}

	var controlTowerResources map[string]*ControlTowerResources
	if err := json.Unmarshal(data, &controlTowerResources); err != nil {
		return fmt.Errorf("failed to unmarshal control tower resources: %w", err)
	}

	s.controlTowerResources = controlTowerResources
	return nil
}

// loadDogmaAttributeCategories loads dogma attribute categories data from JSON file
func (s *Service) loadDogmaAttributeCategories() error {
	filePath := filepath.Join(s.dataDir, "dogmaAttributeCategories.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read dogma attribute categories file: %w", err)
	}

	var dogmaAttributeCategories map[string]*DogmaAttributeCategory
	if err := json.Unmarshal(data, &dogmaAttributeCategories); err != nil {
		return fmt.Errorf("failed to unmarshal dogma attribute categories: %w", err)
	}

	s.dogmaAttributeCategories = dogmaAttributeCategories
	return nil
}

// loadInvNames loads inventory names data from JSON file
func (s *Service) loadInvNames() error {
	filePath := filepath.Join(s.dataDir, "invNames.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read inventory names file: %w", err)
	}

	var invNames []*InvName
	if err := json.Unmarshal(data, &invNames); err != nil {
		return fmt.Errorf("failed to unmarshal inventory names: %w", err)
	}

	s.invNames = invNames
	return nil
}

// loadInvPositions loads inventory positions data from JSON file
func (s *Service) loadInvPositions() error {
	filePath := filepath.Join(s.dataDir, "invPositions.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read inventory positions file: %w", err)
	}

	var invPositions []*InvPosition
	if err := json.Unmarshal(data, &invPositions); err != nil {
		return fmt.Errorf("failed to unmarshal inventory positions: %w", err)
	}

	s.invPositions = invPositions
	return nil
}

// loadInvUniqueNames loads inventory unique names data from JSON file
func (s *Service) loadInvUniqueNames() error {
	filePath := filepath.Join(s.dataDir, "invUniqueNames.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read inventory unique names file: %w", err)
	}

	var invUniqueNames []*InvUniqueName
	if err := json.Unmarshal(data, &invUniqueNames); err != nil {
		return fmt.Errorf("failed to unmarshal inventory unique names: %w", err)
	}

	s.invUniqueNames = invUniqueNames
	return nil
}

// loadPlanetResources loads planet resources data from JSON file
func (s *Service) loadPlanetResources() error {
	filePath := filepath.Join(s.dataDir, "planetResources.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read planet resources file: %w", err)
	}

	var planetResources map[string]*PlanetResource
	if err := json.Unmarshal(data, &planetResources); err != nil {
		return fmt.Errorf("failed to unmarshal planet resources: %w", err)
	}

	s.planetResources = planetResources
	return nil
}

// loadPlanetSchematics loads planet schematics data from JSON file
func (s *Service) loadPlanetSchematics() error {
	filePath := filepath.Join(s.dataDir, "planetSchematics.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read planet schematics file: %w", err)
	}

	var planetSchematics map[string]*PlanetSchematic
	if err := json.Unmarshal(data, &planetSchematics); err != nil {
		return fmt.Errorf("failed to unmarshal planet schematics: %w", err)
	}

	s.planetSchematics = planetSchematics
	return nil
}

// loadSkinLicenses loads skin licenses data from JSON file
func (s *Service) loadSkinLicenses() error {
	filePath := filepath.Join(s.dataDir, "skinLicenses.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read skin licenses file: %w", err)
	}

	var skinLicenses map[string]*SkinLicense
	if err := json.Unmarshal(data, &skinLicenses); err != nil {
		return fmt.Errorf("failed to unmarshal skin licenses: %w", err)
	}

	s.skinLicenses = skinLicenses
	return nil
}

// loadSkinMaterials loads skin materials data from JSON file
func (s *Service) loadSkinMaterials() error {
	filePath := filepath.Join(s.dataDir, "skinMaterials.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read skin materials file: %w", err)
	}

	var skinMaterials map[string]*SkinMaterial
	if err := json.Unmarshal(data, &skinMaterials); err != nil {
		return fmt.Errorf("failed to unmarshal skin materials: %w", err)
	}

	s.skinMaterials = skinMaterials
	return nil
}

// loadSovereigntyUpgrades loads sovereignty upgrades data from JSON file
func (s *Service) loadSovereigntyUpgrades() error {
	filePath := filepath.Join(s.dataDir, "sovereigntyUpgrades.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read sovereignty upgrades file: %w", err)
	}

	var sovereigntyUpgrades map[string]*SovereigntyUpgrade
	if err := json.Unmarshal(data, &sovereigntyUpgrades); err != nil {
		return fmt.Errorf("failed to unmarshal sovereignty upgrades: %w", err)
	}

	s.sovereigntyUpgrades = sovereigntyUpgrades
	return nil
}

// loadTranslationLanguages loads translation languages data from JSON file
func (s *Service) loadTranslationLanguages() error {
	filePath := filepath.Join(s.dataDir, "translationLanguages.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read translation languages file: %w", err)
	}

	var rawLanguages map[string]string
	if err := json.Unmarshal(data, &rawLanguages); err != nil {
		return fmt.Errorf("failed to unmarshal translation languages: %w", err)
	}

	// Convert to structured format
	translationLanguages := make(map[string]*TranslationLanguage)
	for code, name := range rawLanguages {
		translationLanguages[code] = &TranslationLanguage{
			Code: code,
			Name: name,
		}
	}

	s.translationLanguages = translationLanguages
	return nil
}

// GetCharacterAttribute retrieves a character attribute by ID
func (s *Service) GetCharacterAttribute(id string) (*CharacterAttribute, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	characterAttribute, exists := s.characterAttributes[id]
	if !exists {
		return nil, fmt.Errorf("character attribute %s not found", id)
	}

	return characterAttribute, nil
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

// GetAllRaces returns all races
func (s *Service) GetAllRaces() (map[string]*Race, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	races := make(map[string]*Race, len(s.races))
	for id, race := range s.races {
		races[id] = race
	}

	return races, nil
}

// GetAllFactions returns all factions
func (s *Service) GetAllFactions() (map[string]*Faction, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	factions := make(map[string]*Faction, len(s.factions))
	for id, faction := range s.factions {
		factions[id] = faction
	}

	return factions, nil
}

// GetAllBloodlines returns all bloodlines
func (s *Service) GetAllBloodlines() (map[string]*Bloodline, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	bloodlines := make(map[string]*Bloodline, len(s.bloodlines))
	for id, bloodline := range s.bloodlines {
		bloodlines[id] = bloodline
	}

	return bloodlines, nil
}

// GetAllGroups returns all groups
func (s *Service) GetAllGroups() (map[string]*Group, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	groups := make(map[string]*Group, len(s.groups))
	for id, group := range s.groups {
		groups[id] = group
	}

	return groups, nil
}

// GetAllDogmaAttributes returns all dogma attributes
func (s *Service) GetAllDogmaAttributes() (map[string]*DogmaAttribute, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	dogmaAttributes := make(map[string]*DogmaAttribute, len(s.dogmaAttributes))
	for id, dogmaAttribute := range s.dogmaAttributes {
		dogmaAttributes[id] = dogmaAttribute
	}

	return dogmaAttributes, nil
}

// GetAllAncestries returns all ancestries
func (s *Service) GetAllAncestries() (map[string]*Ancestry, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	ancestries := make(map[string]*Ancestry, len(s.ancestries))
	for id, ancestry := range s.ancestries {
		ancestries[id] = ancestry
	}

	return ancestries, nil
}

// GetAllCertificates returns all certificates
func (s *Service) GetAllCertificates() (map[string]*Certificate, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	certificates := make(map[string]*Certificate, len(s.certificates))
	for id, certificate := range s.certificates {
		certificates[id] = certificate
	}

	return certificates, nil
}

// GetAllCharacterAttributes returns all character attributes
func (s *Service) GetAllCharacterAttributes() (map[string]*CharacterAttribute, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	characterAttributes := make(map[string]*CharacterAttribute, len(s.characterAttributes))
	for id, characterAttribute := range s.characterAttributes {
		characterAttributes[id] = characterAttribute
	}

	return characterAttributes, nil
}

// GetSkin retrieves a skin by ID
func (s *Service) GetSkin(id string) (*Skin, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	skin, exists := s.skins[id]
	if !exists {
		return nil, fmt.Errorf("skin %s not found", id)
	}

	return skin, nil
}

// GetAllSkins returns all skins
func (s *Service) GetAllSkins() (map[string]*Skin, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	skins := make(map[string]*Skin, len(s.skins))
	for id, skin := range s.skins {
		skins[id] = skin
	}

	return skins, nil
}

// GetStaStation retrieves a station by station ID
func (s *Service) GetStaStation(stationID int) (*StaStation, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	for _, station := range s.staStations {
		if station.StationID == stationID {
			return station, nil
		}
	}

	return nil, fmt.Errorf("station %d not found", stationID)
}

// GetAllStaStations returns all stations
func (s *Service) GetAllStaStations() ([]*StaStation, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	stations := make([]*StaStation, len(s.staStations))
	copy(stations, s.staStations)

	return stations, nil
}

// GetStaStationsBySolarSystem returns all stations in a solar system
func (s *Service) GetStaStationsBySolarSystem(solarSystemID int) ([]*StaStation, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	var stations []*StaStation
	for _, station := range s.staStations {
		if station.SolarSystemID == solarSystemID {
			stations = append(stations, station)
		}
	}

	return stations, nil
}

// GetDogmaEffect retrieves a dogma effect by ID
func (s *Service) GetDogmaEffect(id string) (*DogmaEffect, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	effect, exists := s.dogmaEffects[id]
	if !exists {
		return nil, fmt.Errorf("dogma effect %s not found", id)
	}

	return effect, nil
}

// GetAllDogmaEffects returns all dogma effects
func (s *Service) GetAllDogmaEffects() (map[string]*DogmaEffect, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	effects := make(map[string]*DogmaEffect, len(s.dogmaEffects))
	for id, effect := range s.dogmaEffects {
		effects[id] = effect
	}

	return effects, nil
}

// GetIconID retrieves an icon ID by ID
func (s *Service) GetIconID(id string) (*IconID, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	iconID, exists := s.iconIDs[id]
	if !exists {
		return nil, fmt.Errorf("icon ID %s not found", id)
	}

	return iconID, nil
}

// GetAllIconIDs returns all icon IDs
func (s *Service) GetAllIconIDs() (map[string]*IconID, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	iconIDs := make(map[string]*IconID, len(s.iconIDs))
	for id, iconID := range s.iconIDs {
		iconIDs[id] = iconID
	}

	return iconIDs, nil
}

// GetGraphicID retrieves a graphic ID by ID
func (s *Service) GetGraphicID(id string) (*GraphicID, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	graphicID, exists := s.graphicIDs[id]
	if !exists {
		return nil, fmt.Errorf("graphic ID %s not found", id)
	}

	return graphicID, nil
}

// GetAllGraphicIDs returns all graphic IDs
func (s *Service) GetAllGraphicIDs() (map[string]*GraphicID, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	graphicIDs := make(map[string]*GraphicID, len(s.graphicIDs))
	for id, graphicID := range s.graphicIDs {
		graphicIDs[id] = graphicID
	}

	return graphicIDs, nil
}

// GetTypeDogma retrieves type dogma by type ID
func (s *Service) GetTypeDogma(typeID string) (*TypeDogma, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	typeDogma, exists := s.typeDogma[typeID]
	if !exists {
		return nil, fmt.Errorf("type dogma for type %s not found", typeID)
	}

	return typeDogma, nil
}

// GetAllTypeDogma returns all type dogma data
func (s *Service) GetAllTypeDogma() (map[string]*TypeDogma, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	typeDogma := make(map[string]*TypeDogma, len(s.typeDogma))
	for id, dogma := range s.typeDogma {
		typeDogma[id] = dogma
	}

	return typeDogma, nil
}

// GetInvFlag retrieves an inventory flag by flag ID
func (s *Service) GetInvFlag(flagID int) (*InvFlag, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	for _, flag := range s.invFlags {
		if flag.FlagID == flagID {
			return flag, nil
		}
	}

	return nil, fmt.Errorf("inventory flag %d not found", flagID)
}

// GetAllInvFlags returns all inventory flags
func (s *Service) GetAllInvFlags() ([]*InvFlag, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	flags := make([]*InvFlag, len(s.invFlags))
	copy(flags, s.invFlags)

	return flags, nil
}

// GetStationService retrieves a station service by ID
func (s *Service) GetStationService(id string) (*StationService, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	service, exists := s.stationServices[id]
	if !exists {
		return nil, fmt.Errorf("station service %s not found", id)
	}

	return service, nil
}

// GetAllStationServices returns all station services
func (s *Service) GetAllStationServices() (map[string]*StationService, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	services := make(map[string]*StationService, len(s.stationServices))
	for id, service := range s.stationServices {
		services[id] = service
	}

	return services, nil
}

// GetStationOperation retrieves a station operation by ID
func (s *Service) GetStationOperation(id string) (*StationOperation, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	operation, exists := s.stationOperations[id]
	if !exists {
		return nil, fmt.Errorf("station operation %s not found", id)
	}

	return operation, nil
}

// GetAllStationOperations returns all station operations
func (s *Service) GetAllStationOperations() (map[string]*StationOperation, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	operations := make(map[string]*StationOperation, len(s.stationOperations))
	for id, operation := range s.stationOperations {
		operations[id] = operation
	}

	return operations, nil
}

// GetResearchAgent retrieves a research agent by ID
func (s *Service) GetResearchAgent(id string) (*ResearchAgent, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	agent, exists := s.researchAgents[id]
	if !exists {
		return nil, fmt.Errorf("research agent %s not found", id)
	}

	return agent, nil
}

// GetAllResearchAgents returns all research agents
func (s *Service) GetAllResearchAgents() (map[string]*ResearchAgent, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	agents := make(map[string]*ResearchAgent, len(s.researchAgents))
	for id, agent := range s.researchAgents {
		agents[id] = agent
	}

	return agents, nil
}

// GetAgentInSpace retrieves an agent in space by ID
func (s *Service) GetAgentInSpace(id string) (*AgentInSpace, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	agent, exists := s.agentsInSpace[id]
	if !exists {
		return nil, fmt.Errorf("agent in space %s not found", id)
	}

	return agent, nil
}

// GetAllAgentsInSpace returns all agents in space
func (s *Service) GetAllAgentsInSpace() (map[string]*AgentInSpace, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	agents := make(map[string]*AgentInSpace, len(s.agentsInSpace))
	for id, agent := range s.agentsInSpace {
		agents[id] = agent
	}

	return agents, nil
}

// GetContrabandType retrieves a contraband type by ID
func (s *Service) GetContrabandType(id string) (*ContrabandType, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	contraband, exists := s.contrabandTypes[id]
	if !exists {
		return nil, fmt.Errorf("contraband type %s not found", id)
	}

	return contraband, nil
}

// GetAllContrabandTypes returns all contraband types
func (s *Service) GetAllContrabandTypes() (map[string]*ContrabandType, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	contraband := make(map[string]*ContrabandType, len(s.contrabandTypes))
	for id, cb := range s.contrabandTypes {
		contraband[id] = cb
	}

	return contraband, nil
}

// GetCorporationActivity retrieves a corporation activity by ID
func (s *Service) GetCorporationActivity(id string) (*CorporationActivity, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	activity, exists := s.corporationActivities[id]
	if !exists {
		return nil, fmt.Errorf("corporation activity %s not found", id)
	}

	return activity, nil
}

// GetAllCorporationActivities returns all corporation activities
func (s *Service) GetAllCorporationActivities() (map[string]*CorporationActivity, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	activities := make(map[string]*CorporationActivity, len(s.corporationActivities))
	for id, activity := range s.corporationActivities {
		activities[id] = activity
	}

	return activities, nil
}

// GetInvItem retrieves an inventory item by item ID
func (s *Service) GetInvItem(itemID int) (*InvItem, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	for _, item := range s.invItems {
		if item.ItemID == itemID {
			return item, nil
		}
	}

	return nil, fmt.Errorf("inventory item %d not found", itemID)
}

// GetAllInvItems returns all inventory items
func (s *Service) GetAllInvItems() ([]*InvItem, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	items := make([]*InvItem, len(s.invItems))
	copy(items, s.invItems)

	return items, nil
}

// GetNPCCorporationDivision retrieves an NPC corporation division by ID
func (s *Service) GetNPCCorporationDivision(id string) (*NPCCorporationDivision, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	division, exists := s.npcCorporationDivisions[id]
	if !exists {
		return nil, fmt.Errorf("NPC corporation division %s not found", id)
	}

	return division, nil
}

// GetAllNPCCorporationDivisions returns all NPC corporation divisions
func (s *Service) GetAllNPCCorporationDivisions() (map[string]*NPCCorporationDivision, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	divisions := make(map[string]*NPCCorporationDivision, len(s.npcCorporationDivisions))
	for id, division := range s.npcCorporationDivisions {
		divisions[id] = division
	}

	return divisions, nil
}

// GetControlTowerResources retrieves control tower resources by ID
func (s *Service) GetControlTowerResources(id string) (*ControlTowerResources, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	resources, exists := s.controlTowerResources[id]
	if !exists {
		return nil, fmt.Errorf("control tower resources %s not found", id)
	}

	return resources, nil
}

// GetAllControlTowerResources returns all control tower resources
func (s *Service) GetAllControlTowerResources() (map[string]*ControlTowerResources, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	resources := make(map[string]*ControlTowerResources, len(s.controlTowerResources))
	for id, resource := range s.controlTowerResources {
		resources[id] = resource
	}

	return resources, nil
}

// GetDogmaAttributeCategory retrieves a dogma attribute category by ID
func (s *Service) GetDogmaAttributeCategory(id string) (*DogmaAttributeCategory, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	category, exists := s.dogmaAttributeCategories[id]
	if !exists {
		return nil, fmt.Errorf("dogma attribute category %s not found", id)
	}

	return category, nil
}

// GetAllDogmaAttributeCategories returns all dogma attribute categories
func (s *Service) GetAllDogmaAttributeCategories() (map[string]*DogmaAttributeCategory, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	categories := make(map[string]*DogmaAttributeCategory, len(s.dogmaAttributeCategories))
	for id, category := range s.dogmaAttributeCategories {
		categories[id] = category
	}

	return categories, nil
}

// GetInvName retrieves an inventory name by item ID
func (s *Service) GetInvName(itemID int) (*InvName, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	for _, name := range s.invNames {
		if name.ItemID == itemID {
			return name, nil
		}
	}

	return nil, fmt.Errorf("inventory name for item %d not found", itemID)
}

// GetAllInvNames returns all inventory names
func (s *Service) GetAllInvNames() ([]*InvName, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	names := make([]*InvName, len(s.invNames))
	copy(names, s.invNames)

	return names, nil
}

// GetInvPosition retrieves an inventory position by item ID
func (s *Service) GetInvPosition(itemID int) (*InvPosition, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	for _, position := range s.invPositions {
		if position.ItemID == itemID {
			return position, nil
		}
	}

	return nil, fmt.Errorf("inventory position for item %d not found", itemID)
}

// GetAllInvPositions returns all inventory positions
func (s *Service) GetAllInvPositions() ([]*InvPosition, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	positions := make([]*InvPosition, len(s.invPositions))
	copy(positions, s.invPositions)

	return positions, nil
}

// GetInvUniqueName retrieves an inventory unique name by item ID
func (s *Service) GetInvUniqueName(itemID int) (*InvUniqueName, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	for _, uniqueName := range s.invUniqueNames {
		if uniqueName.ItemID == itemID {
			return uniqueName, nil
		}
	}

	return nil, fmt.Errorf("inventory unique name for item %d not found", itemID)
}

// GetAllInvUniqueNames returns all inventory unique names
func (s *Service) GetAllInvUniqueNames() ([]*InvUniqueName, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	uniqueNames := make([]*InvUniqueName, len(s.invUniqueNames))
	copy(uniqueNames, s.invUniqueNames)

	return uniqueNames, nil
}

// GetPlanetResource retrieves a planet resource by ID
func (s *Service) GetPlanetResource(id string) (*PlanetResource, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	resource, exists := s.planetResources[id]
	if !exists {
		return nil, fmt.Errorf("planet resource %s not found", id)
	}

	return resource, nil
}

// GetAllPlanetResources returns all planet resources
func (s *Service) GetAllPlanetResources() (map[string]*PlanetResource, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	resources := make(map[string]*PlanetResource, len(s.planetResources))
	for id, resource := range s.planetResources {
		resources[id] = resource
	}

	return resources, nil
}

// GetPlanetSchematic retrieves a planet schematic by ID
func (s *Service) GetPlanetSchematic(id string) (*PlanetSchematic, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	schematic, exists := s.planetSchematics[id]
	if !exists {
		return nil, fmt.Errorf("planet schematic %s not found", id)
	}

	return schematic, nil
}

// GetAllPlanetSchematics returns all planet schematics
func (s *Service) GetAllPlanetSchematics() (map[string]*PlanetSchematic, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	schematics := make(map[string]*PlanetSchematic, len(s.planetSchematics))
	for id, schematic := range s.planetSchematics {
		schematics[id] = schematic
	}

	return schematics, nil
}

// GetSkinLicense retrieves a skin license by ID
func (s *Service) GetSkinLicense(id string) (*SkinLicense, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	license, exists := s.skinLicenses[id]
	if !exists {
		return nil, fmt.Errorf("skin license %s not found", id)
	}

	return license, nil
}

// GetAllSkinLicenses returns all skin licenses
func (s *Service) GetAllSkinLicenses() (map[string]*SkinLicense, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	licenses := make(map[string]*SkinLicense, len(s.skinLicenses))
	for id, license := range s.skinLicenses {
		licenses[id] = license
	}

	return licenses, nil
}

// GetSkinMaterial retrieves a skin material by ID
func (s *Service) GetSkinMaterial(id string) (*SkinMaterial, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	material, exists := s.skinMaterials[id]
	if !exists {
		return nil, fmt.Errorf("skin material %s not found", id)
	}

	return material, nil
}

// GetAllSkinMaterials returns all skin materials
func (s *Service) GetAllSkinMaterials() (map[string]*SkinMaterial, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	materials := make(map[string]*SkinMaterial, len(s.skinMaterials))
	for id, material := range s.skinMaterials {
		materials[id] = material
	}

	return materials, nil
}

// GetSovereigntyUpgrade retrieves a sovereignty upgrade by ID
func (s *Service) GetSovereigntyUpgrade(id string) (*SovereigntyUpgrade, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	upgrade, exists := s.sovereigntyUpgrades[id]
	if !exists {
		return nil, fmt.Errorf("sovereignty upgrade %s not found", id)
	}

	return upgrade, nil
}

// GetAllSovereigntyUpgrades returns all sovereignty upgrades
func (s *Service) GetAllSovereigntyUpgrades() (map[string]*SovereigntyUpgrade, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	upgrades := make(map[string]*SovereigntyUpgrade, len(s.sovereigntyUpgrades))
	for id, upgrade := range s.sovereigntyUpgrades {
		upgrades[id] = upgrade
	}

	return upgrades, nil
}

// GetTranslationLanguage retrieves a translation language by code
func (s *Service) GetTranslationLanguage(code string) (*TranslationLanguage, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	language, exists := s.translationLanguages[code]
	if !exists {
		return nil, fmt.Errorf("translation language %s not found", code)
	}

	return language, nil
}

// GetAllTranslationLanguages returns all translation languages
func (s *Service) GetAllTranslationLanguages() (map[string]*TranslationLanguage, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	languages := make(map[string]*TranslationLanguage, len(s.translationLanguages))
	for code, language := range s.translationLanguages {
		languages[code] = language
	}

	return languages, nil
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
