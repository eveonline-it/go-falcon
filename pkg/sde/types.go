package sde

import "encoding/json"

// Agent represents an EVE Online agent from the SDE
type Agent struct {
	AgentTypeID   int  `json:"agentTypeID"`
	CorporationID int  `json:"corporationID"`
	DivisionID    int  `json:"divisionID"`
	IsLocator     bool `json:"isLocator"`
	Level         int  `json:"level"`
	LocationID    int  `json:"locationID"`
}

// Category represents an EVE Online item category from the SDE
type Category struct {
	Name      map[string]string `json:"name"` // Internationalized names
	Published bool              `json:"published"`
}

// Blueprint represents an EVE Online blueprint from the SDE
type Blueprint struct {
	Activities map[string]Activity `json:"activities"`
}

// Activity represents blueprint activities (manufacturing, research, etc.)
type Activity struct {
	Materials []Material `json:"materials,omitempty"`
	Products  []Product  `json:"products,omitempty"`
	Skills    []Skill    `json:"skills,omitempty"`
	Time      int        `json:"time,omitempty"`
}

// Material represents required materials for blueprint activities
type Material struct {
	Quantity int `json:"quantity"`
	TypeID   int `json:"typeID"`
}

// Product represents products from blueprint activities
type Product struct {
	Quantity    int     `json:"quantity"`
	TypeID      int     `json:"typeID"`
	Probability float64 `json:"probability,omitempty"`
}

// Skill represents required skills for blueprint activities
type Skill struct {
	Level  int `json:"level"`
	TypeID int `json:"typeID"`
}

// MarketGroup represents an EVE Online market group from the SDE
type MarketGroup struct {
	DescriptionID map[string]string `json:"descriptionID,omitempty"`
	HasTypes      bool              `json:"hasTypes,omitempty"`
	IconID        int               `json:"iconID,omitempty"`
	NameID        map[string]string `json:"nameID"`
	ParentGroupID int               `json:"parentGroupID,omitempty"`
}

// MetaGroup represents an EVE Online meta group from the SDE
type MetaGroup struct {
	Color      []float64         `json:"color,omitempty"`
	IconID     int               `json:"iconID,omitempty"`
	IconSuffix string            `json:"iconSuffix,omitempty"`
	NameID     map[string]string `json:"nameID"`
}

// FlexibleString is a type that can unmarshal both string and boolean values
type FlexibleString struct {
	Value string
}

// UnmarshalJSON implements custom unmarshaling for FlexibleString
func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		fs.Value = str
		return nil
	}

	// If that fails, try as boolean
	var boolean bool
	if err := json.Unmarshal(data, &boolean); err == nil {
		if boolean {
			fs.Value = "true"
		} else {
			fs.Value = "false"
		}
		return nil
	}

	// If both fail, return empty string
	fs.Value = ""
	return nil
}

// MarshalJSON implements custom marshaling for FlexibleString
func (fs FlexibleString) MarshalJSON() ([]byte, error) {
	return json.Marshal(fs.Value)
}

// String returns the string value
func (fs FlexibleString) String() string {
	return fs.Value
}

// NPCCorporation represents an EVE Online NPC corporation from the SDE
type NPCCorporation struct {
	AllowedMemberRaces         []int             `json:"allowedMemberRaces,omitempty"`
	CeoID                      int               `json:"ceoID,omitempty"`
	Deleted                    bool              `json:"deleted,omitempty"`
	DescriptionID              map[string]string `json:"descriptionID,omitempty"`
	Extent                     FlexibleString    `json:"extent,omitempty"`
	FactionID                  int               `json:"factionID,omitempty"`
	HasPlayerPersonnelManager  bool              `json:"hasPlayerPersonnelManager,omitempty"`
	IconID                     int               `json:"iconID,omitempty"`
	InitialPrice               float64           `json:"initialPrice,omitempty"`
	MemberLimit                int               `json:"memberLimit,omitempty"`
	MinSecurity                float64           `json:"minSecurity,omitempty"`
	MinimumJoinStanding        float64           `json:"minimumJoinStanding,omitempty"`
	NameID                     map[string]string `json:"nameID"`
	PublicShares               int64             `json:"publicShares,omitempty"`
	SendCharTerminationMessage bool              `json:"sendCharTerminationMessage,omitempty"`
	Shares                     int64             `json:"shares,omitempty"`
	Size                       string            `json:"size,omitempty"`
	SizeFactor                 float64           `json:"sizeFactor,omitempty"`
	SolarSystemID              int               `json:"solarSystemID,omitempty"`
	StationID                  int               `json:"stationID,omitempty"`
	TaxRate                    float64           `json:"taxRate,omitempty"`
	TickerName                 FlexibleString    `json:"tickerName,omitempty"`
	UniqueName                 bool              `json:"uniqueName,omitempty"`
}

// TypeID represents basic type information from typeIDs.yaml
type TypeID struct {
	Name        map[string]string `json:"name"`
	Description map[string]string `json:"description,omitempty"`
	GroupID     int               `json:"groupID,omitempty"`
	Published   bool              `json:"published,omitempty"`
}

// Type represents detailed type information from types.yaml
type Type struct {
	BasePrice      float64           `json:"basePrice,omitempty"`
	Capacity       float64           `json:"capacity,omitempty"`
	Description    map[string]string `json:"description,omitempty"`
	FactionID      int               `json:"factionID,omitempty"`
	GraphicID      int               `json:"graphicID,omitempty"`
	GroupID        int               `json:"groupID,omitempty"`
	IconID         int               `json:"iconID,omitempty"`
	Mass           float64           `json:"mass,omitempty"`
	Name           map[string]string `json:"name"`
	PackagedVolume float64           `json:"packagedVolume,omitempty"`
	PortionSize    int               `json:"portionSize,omitempty"`
	Published      bool              `json:"published,omitempty"`
	RaceID         int               `json:"raceID,omitempty"`
	Radius         float64           `json:"radius,omitempty"`
	SofFactionName string            `json:"sofFactionName,omitempty"`
	SoundID        int               `json:"soundID,omitempty"`
	Volume         float64           `json:"volume,omitempty"`
}

// TypeMaterial represents material requirements for a type
type TypeMaterial struct {
	MaterialTypeID int `json:"materialTypeID"`
	Quantity       int `json:"quantity"`
}

// TypeMaterialData represents the wrapper structure in typeMaterials.json
type TypeMaterialData struct {
	Materials []*TypeMaterial `json:"materials"`
}

// Race represents an EVE Online race from the SDE
type Race struct {
	DescriptionID map[string]string `json:"descriptionID"`
	IconID        int               `json:"iconID"`
	NameID        map[string]string `json:"nameID"`
	ShipTypeID    int               `json:"shipTypeID"`
	Skills        map[string]int    `json:"skills"` // skill_id -> level
}

// Faction represents an EVE Online faction from the SDE
type Faction struct {
	CorporationID        int               `json:"corporationID"`
	DescriptionID        map[string]string `json:"descriptionID"`
	FlatLogo             string            `json:"flatLogo,omitempty"`
	FlatLogoWithName     string            `json:"flatLogoWithName,omitempty"`
	IconID               int               `json:"iconID,omitempty"`
	MemberRaces          []int             `json:"memberRaces,omitempty"`
	MilitiaCorporationID int               `json:"militiaCorporationID,omitempty"`
	NameID               map[string]string `json:"nameID"`
	ShortDescriptionID   map[string]string `json:"shortDescriptionID,omitempty"`
	SizeFactor           int               `json:"sizeFactor,omitempty"`
	SolarSystemID        int               `json:"solarSystemID,omitempty"`
	UniqueName           bool              `json:"uniqueName,omitempty"`
}

// Bloodline represents an EVE Online bloodline from the SDE
type Bloodline struct {
	Charisma      int               `json:"charisma"`
	CorporationID int               `json:"corporationID"`
	DescriptionID map[string]string `json:"descriptionID"`
	IconID        int               `json:"iconID,omitempty"`
	Intelligence  int               `json:"intelligence"`
	Memory        int               `json:"memory"`
	NameID        map[string]string `json:"nameID"`
	Perception    int               `json:"perception"`
	RaceID        int               `json:"raceID"`
	Willpower     int               `json:"willpower"`
}

// Group represents an EVE Online group from the SDE
type Group struct {
	Anchorable           bool              `json:"anchorable"`
	Anchored             bool              `json:"anchored"`
	CategoryID           int               `json:"categoryID"`
	FittableNonSingleton bool              `json:"fittableNonSingleton"`
	Name                 map[string]string `json:"name"`
	Published            bool              `json:"published"`
	UseBasePrice         bool              `json:"useBasePrice"`
}

// DogmaAttribute represents an EVE Online dogma attribute from the SDE
type DogmaAttribute struct {
	AttributeID  int     `json:"attributeID"`
	CategoryID   int     `json:"categoryID,omitempty"`
	DataType     int     `json:"dataType"`
	DefaultValue float64 `json:"defaultValue"`
	Description  string  `json:"description,omitempty"`
	HighIsGood   bool    `json:"highIsGood"`
	IconID       int     `json:"iconID,omitempty"`
	Name         string  `json:"name"`
	Published    bool    `json:"published"`
	Stackable    bool    `json:"stackable"`
	UnitID       int     `json:"unitID,omitempty"`
}

// Ancestry represents an EVE Online ancestry from the SDE
type Ancestry struct {
	BloodlineID      int               `json:"bloodlineID"`
	Charisma         int               `json:"charisma"`
	DescriptionID    map[string]string `json:"descriptionID"`
	IconID           int               `json:"iconID,omitempty"`
	Intelligence     int               `json:"intelligence"`
	Memory           int               `json:"memory"`
	NameID           map[string]string `json:"nameID"`
	Perception       int               `json:"perception"`
	ShortDescription string            `json:"shortDescription,omitempty"`
	Willpower        int               `json:"willpower"`
}

// Certificate represents an EVE Online certificate from the SDE
type Certificate struct {
	Description    string                      `json:"description"`
	GroupID        int                         `json:"groupID"`
	Name           string                      `json:"name"`
	RecommendedFor []int                       `json:"recommendedFor,omitempty"`
	SkillTypes     map[string]CertificateSkill `json:"skillTypes"`
}

// CertificateSkill represents skill level requirements for a certificate
type CertificateSkill struct {
	Basic    int `json:"basic"`
	Standard int `json:"standard"`
	Improved int `json:"improved"`
	Advanced int `json:"advanced"`
	Elite    int `json:"elite"`
}

// CharacterAttribute represents an EVE Online character attribute from the SDE
type CharacterAttribute struct {
	Description      string            `json:"description"`
	IconID           int               `json:"iconID,omitempty"`
	NameID           map[string]string `json:"nameID"`
	Notes            string            `json:"notes,omitempty"`
	ShortDescription string            `json:"shortDescription,omitempty"`
}

// Skin represents an EVE Online ship skin from the SDE
type Skin struct {
	AllowCCPDevs       bool        `json:"allowCCPDevs"`
	InternalName       string      `json:"internalName"`
	SkinID             int         `json:"skinID"`
	SkinMaterialID     int         `json:"skinMaterialID"`
	SkinDescription    interface{} `json:"skinDescription,omitempty"`
	Types              []int       `json:"types"`
	VisibleSerenity    bool        `json:"visibleSerenity"`
	VisibleTranquility bool        `json:"visibleTranquility"`
}

// StaStation represents an EVE Online station from the SDE
type StaStation struct {
	ConstellationID          int     `json:"constellationID"`
	CorporationID            int     `json:"corporationID"`
	DockingCostPerVolume     float64 `json:"dockingCostPerVolume"`
	MaxShipVolumeDockable    float64 `json:"maxShipVolumeDockable"`
	OfficeRentalCost         int     `json:"officeRentalCost"`
	OperationID              int     `json:"operationID"`
	RegionID                 int     `json:"regionID"`
	ReprocessingEfficiency   float64 `json:"reprocessingEfficiency"`
	ReprocessingHangarFlag   int     `json:"reprocessingHangarFlag"`
	ReprocessingStationsTake float64 `json:"reprocessingStationsTake"`
	Security                 float64 `json:"security"`
	SolarSystemID            int     `json:"solarSystemID"`
	StationID                int     `json:"stationID"`
	StationName              string  `json:"stationName"`
	StationTypeID            int     `json:"stationTypeID"`
	X                        float64 `json:"x,omitempty"`
	Y                        float64 `json:"true,omitempty"` // Note: API uses "true" as field name
	Z                        float64 `json:"z,omitempty"`
}

// DogmaEffect represents an EVE Online dogma effect from the SDE
type DogmaEffect struct {
	DisallowAutoRepeat            bool           `json:"disallowAutoRepeat"`
	DischargeAttributeID          int            `json:"dischargeAttributeID,omitempty"`
	Distribution                  int            `json:"distribution,omitempty"`
	DurationAttributeID           int            `json:"durationAttributeID,omitempty"`
	EffectCategory                int            `json:"effectCategory"`
	EffectID                      int            `json:"effectID"`
	EffectName                    string         `json:"effectName"`
	ElectronicChance              bool           `json:"electronicChance"`
	FalloffAttributeID            int            `json:"falloffAttributeID,omitempty"`
	FittingUsageChanceAttributeID int            `json:"fittingUsageChanceAttributeID,omitempty"`
	Guid                          string         `json:"guid,omitempty"`
	IsAssistance                  bool           `json:"isAssistance"`
	IsOffensive                   bool           `json:"isOffensive"`
	IsWarpSafe                    bool           `json:"isWarpSafe"`
	ModifierInfo                  []ModifierInfo `json:"modifierInfo,omitempty"`
	PropulsionChance              bool           `json:"propulsionChance"`
	Published                     bool           `json:"published"`
	RangeAttributeID              int            `json:"rangeAttributeID,omitempty"`
	RangeChance                   bool           `json:"rangeChance"`
	TrackingSpeedAttributeID      int            `json:"trackingSpeedAttributeID,omitempty"`
}

// ModifierInfo represents modifier information for dogma effects
type ModifierInfo struct {
	Domain               string `json:"domain"`
	Function             string `json:"func"`
	ModifiedAttributeID  int    `json:"modifiedAttributeID"`
	ModifyingAttributeID int    `json:"modifyingAttributeID"`
	Operation            int    `json:"operation"`
	SkillTypeID          int    `json:"skillTypeID,omitempty"`
}

// IconID represents an EVE Online icon ID mapping from the SDE
type IconID struct {
	Description string `json:"description"`
	IconFile    string `json:"iconFile"`
}

// GraphicID represents an EVE Online graphic ID from the SDE
type GraphicID struct {
	Description    string `json:"description,omitempty"`
	GraphicFile    string `json:"graphicFile,omitempty"`
	IconFolder     string `json:"iconFolder,omitempty"`
	SofDNA         string `json:"sofDNA,omitempty"`
	SofFactionName string `json:"sofFactionName,omitempty"`
	SofHullName    string `json:"sofHullName,omitempty"`
	SofRaceName    string `json:"sofRaceName,omitempty"`
}

// TypeDogma represents dogma attributes for a specific type
type TypeDogma struct {
	DogmaAttributes []TypeDogmaAttribute `json:"dogmaAttributes"`
	DogmaEffects    []TypeDogmaEffect    `json:"dogmaEffects,omitempty"`
}

// TypeDogmaAttribute represents a single dogma attribute value
type TypeDogmaAttribute struct {
	AttributeID int     `json:"attributeID"`
	Value       float64 `json:"value"`
}

// TypeDogmaEffect represents a single dogma effect
type TypeDogmaEffect struct {
	EffectID  int  `json:"effectID"`
	IsDefault bool `json:"isDefault,omitempty"`
}

// InvFlag represents an inventory flag from the SDE
type InvFlag struct {
	FlagID   int    `json:"flagID"`
	FlagName string `json:"flagName"`
	FlagText string `json:"flagText"`
	OrderID  int    `json:"orderID"`
}

// StationService represents a station service from the SDE
type StationService struct {
	ServiceNameID map[string]string `json:"serviceNameID"`
}

// StationOperation represents a station operation from the SDE
type StationOperation struct {
	ActivityID          int               `json:"activityID"`
	AmarrStationType    int               `json:"amarrStationType,omitempty"`
	Border              float64           `json:"border"`
	CaldariStationType  int               `json:"caldariStationType,omitempty"`
	Corridor            float64           `json:"corridor"`
	DescriptionID       map[string]string `json:"descriptionID"`
	Fringe              float64           `json:"fringe"`
	GallenteStationType int               `json:"gallenteStationType,omitempty"`
	Hub                 float64           `json:"hub"`
	ManufacturingFactor float64           `json:"manufacturingFactor"`
	MinmatarStationType int               `json:"minmatarStationType,omitempty"`
	OperationNameID     map[string]string `json:"operationNameID"`
	ResearchFactor      float64           `json:"researchFactor,omitempty"`
}

// ResearchAgent represents a research agent from the SDE
type ResearchAgent struct {
	Skills []ResearchAgentSkill `json:"skills"`
}

// ResearchAgentSkill represents a skill for a research agent
type ResearchAgentSkill struct {
	TypeID int `json:"typeID"`
}

// AgentInSpace represents an agent in space from the SDE
type AgentInSpace struct {
	DungeonID     int `json:"dungeonID"`
	SolarSystemID int `json:"solarSystemID"`
	SpawnPointID  int `json:"spawnPointID"`
	TypeID        int `json:"typeID"`
}

// ContrabandType represents contraband information for a type
type ContrabandType struct {
	Factions map[string]ContrabandFaction `json:"factions"`
}

// ContrabandFaction represents faction-specific contraband rules
type ContrabandFaction struct {
	AttackMinSec     float64 `json:"attackMinSec"`
	ConfiscateMinSec float64 `json:"confiscateMinSec"`
	FineByValue      float64 `json:"fineByValue"`
	StandingLoss     float64 `json:"standingLoss"`
}

// CorporationActivity represents a corporation activity from the SDE
type CorporationActivity struct {
	NameID map[string]string `json:"nameID"`
}

// InvItem represents an inventory item from the SDE
type InvItem struct {
	FlagID     int `json:"flagID"`
	ItemID     int `json:"itemID"`
	LocationID int `json:"locationID"`
	OwnerID    int `json:"ownerID"`
	Quantity   int `json:"quantity"`
	TypeID     int `json:"typeID"`
}

// NPCCorporationDivision represents an NPC corporation division
type NPCCorporationDivision struct {
	Description      string            `json:"description"`
	InternalName     string            `json:"internalName"`
	LeaderTypeNameID map[string]string `json:"leaderTypeNameID"`
	NameID           map[string]string `json:"nameID"`
}

// ControlTowerResource represents a resource requirement for a control tower
type ControlTowerResource struct {
	Purpose          int      `json:"purpose"`
	Quantity         int      `json:"quantity"`
	ResourceTypeID   int      `json:"resourceTypeID"`
	FactionID        *int     `json:"factionID,omitempty"`
	MinSecurityLevel *float64 `json:"minSecurityLevel,omitempty"`
}

// ControlTowerResources represents a control tower with its resource requirements
type ControlTowerResources struct {
	Resources []ControlTowerResource `json:"resources"`
}

// DogmaAttributeCategory represents a category of dogma attributes
type DogmaAttributeCategory struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// InvName represents an inventory name mapping
type InvName struct {
	ItemID   int         `json:"itemID"`
	ItemName interface{} `json:"itemName"`
}

// InvPosition represents position and orientation data for an item
type InvPosition struct {
	ItemID int      `json:"itemID"`
	X      *float64 `json:"x,omitempty"`
	Y      *float64 `json:"y,omitempty"`
	Z      *float64 `json:"z,omitempty"`
	Yaw    *float64 `json:"yaw,omitempty"`
	Pitch  *float64 `json:"pitch,omitempty"`
	Roll   *float64 `json:"roll,omitempty"`
	True   *float64 `json:"true,omitempty"`
}

// InvUniqueName represents a unique name mapping with group information
type InvUniqueName struct {
	ItemID   int         `json:"itemID"`
	ItemName interface{} `json:"itemName"`
	GroupID  int         `json:"groupID"`
}

// PlanetResource represents power and workforce requirements for planet resources
type PlanetResource struct {
	Power     *int `json:"power,omitempty"`
	Workforce *int `json:"workforce,omitempty"`
}

// PlanetSchematicType represents input/output types for planet schematics
type PlanetSchematicType struct {
	IsInput  bool `json:"isInput"`
	Quantity int  `json:"quantity"`
}

// PlanetSchematic represents a planetary interaction schematic
type PlanetSchematic struct {
	CycleTime int                            `json:"cycleTime"`
	NameID    map[string]string              `json:"nameID"`
	Pins      []int                          `json:"pins"`
	Types     map[string]PlanetSchematicType `json:"types"`
}

// SkinLicense represents a skin license with duration and type information
type SkinLicense struct {
	Duration      int `json:"duration"`
	LicenseTypeID int `json:"licenseTypeID"`
	SkinID        int `json:"skinID"`
}

// SkinMaterial represents skin material information
type SkinMaterial struct {
	DisplayNameID  int `json:"displayNameID"`
	MaterialSetID  int `json:"materialSetID"`
	SkinMaterialID int `json:"skinMaterialID"`
}

// SovereigntyUpgrade represents sovereignty upgrade specifications
type SovereigntyUpgrade struct {
	FuelHourlyUpkeep       int    `json:"fuel_hourly_upkeep"`
	FuelStartupCost        int    `json:"fuel_startup_cost"`
	FuelTypeID             int    `json:"fuel_type_id"`
	MutuallyExclusiveGroup string `json:"mutually_exclusive_group"`
	PowerAllocation        int    `json:"power_allocation"`
	WorkforceAllocation    int    `json:"workforce_allocation"`
}

// TranslationLanguage represents a simple language code to name mapping
type TranslationLanguage struct {
	Code string `json:"-"` // This will be set from the map key
	Name string `json:"-"` // This will be set from the map value
}

// ===============================
// Universe Data Types
// ===============================

// Region represents an EVE Online region from the SDE universe data
type Region struct {
	Center          [3]float64 `json:"center"`
	DescriptionID   int        `json:"descriptionID,omitempty"`
	FactionID       int        `json:"factionID,omitempty"`
	Max             [3]float64 `json:"max"`
	Min             [3]float64 `json:"min"`
	NameID          int        `json:"nameID"`
	Nebula          int        `json:"nebula,omitempty"`
	RegionID        int        `json:"regionID"`
	WormholeClassID int        `json:"wormholeClassID,omitempty"`
}

// Constellation represents an EVE Online constellation from the SDE universe data
type Constellation struct {
	Center          [3]float64 `json:"center"`
	ConstellationID int        `json:"constellationID"`
	Max             [3]float64 `json:"max"`
	Min             [3]float64 `json:"min"`
	NameID          int        `json:"nameID"`
	Radius          float64    `json:"radius,omitempty"`
}

// SolarSystem represents an EVE Online solar system from the SDE universe data
type SolarSystem struct {
	Border            bool                 `json:"border"`
	Center            [3]float64           `json:"center"`
	Corridor          bool                 `json:"corridor"`
	Fringe            bool                 `json:"fringe"`
	Hub               bool                 `json:"hub"`
	International     bool                 `json:"international"`
	Luminosity        float64              `json:"luminosity"`
	Max               [3]float64           `json:"max"`
	Min               [3]float64           `json:"min"`
	Planets           map[string]*Planet   `json:"planets,omitempty"`
	Radius            float64              `json:"radius"`
	Regional          bool                 `json:"regional"`
	Security          float64              `json:"security"`
	SecurityClass     string               `json:"securityClass,omitempty"`
	SolarSystemID     int                  `json:"solarSystemID"`
	SolarSystemNameID int                  `json:"solarSystemNameID"`
	Star              *Star                `json:"star,omitempty"`
	Stargates         map[string]*Stargate `json:"stargates,omitempty"`
	SunTypeID         int                  `json:"sunTypeID,omitempty"`
	WormholeClassID   int                  `json:"wormholeClassID,omitempty"`
}

// Planet represents a planet in a solar system
type Planet struct {
	AsteroidBelts    map[string]*AsteroidBelt `json:"asteroidBelts,omitempty"`
	CelestialIndex   int                      `json:"celestialIndex"`
	Moons            map[string]*Moon         `json:"moons,omitempty"`
	NPCStations      map[string]*NPCStation   `json:"npcStations,omitempty"`
	PlanetAttributes *PlanetAttributes        `json:"planetAttributes,omitempty"`
	Position         [3]float64               `json:"position"`
	Radius           float64                  `json:"radius"`
	Statistics       *CelestialStatistics     `json:"statistics,omitempty"`
	TypeID           int                      `json:"typeID"`
}

// Moon represents a moon orbiting a planet
type Moon struct {
	NPCStations      map[string]*NPCStation `json:"npcStations,omitempty"`
	PlanetAttributes *PlanetAttributes      `json:"planetAttributes,omitempty"`
	Position         [3]float64             `json:"position"`
	Radius           float64                `json:"radius"`
	Statistics       *CelestialStatistics   `json:"statistics,omitempty"`
	TypeID           int                    `json:"typeID"`
}

// AsteroidBelt represents an asteroid belt in a solar system
type AsteroidBelt struct {
	Position   [3]float64           `json:"position"`
	Statistics *CelestialStatistics `json:"statistics,omitempty"`
	TypeID     int                  `json:"typeID"`
}

// Star represents the central star of a solar system
type Star struct {
	ID         int             `json:"id"`
	Radius     float64         `json:"radius"`
	Statistics *StarStatistics `json:"statistics,omitempty"`
	TypeID     int             `json:"typeID"`
}

// Stargate represents a stargate in a solar system
type Stargate struct {
	Destination int        `json:"destination"`
	Position    [3]float64 `json:"position"`
	TypeID      int        `json:"typeID"`
}

// NPCStation represents an NPC station on a moon or planet
type NPCStation struct {
	GraphicID                int        `json:"graphicID"`
	IsConquerable            bool       `json:"isConquerable"`
	OperationID              int        `json:"operationID"`
	OwnerID                  int        `json:"ownerID"`
	Position                 [3]float64 `json:"position"`
	ReprocessingEfficiency   float64    `json:"reprocessingEfficiency"`
	ReprocessingHangarFlag   int        `json:"reprocessingHangarFlag"`
	ReprocessingStationsTake float64    `json:"reprocessingStationsTake"`
	TypeID                   int        `json:"typeID"`
	UseOperationName         bool       `json:"useOperationName"`
}

// PlanetAttributes represents visual and shader attributes for planets and moons
type PlanetAttributes struct {
	HeightMap1   int  `json:"heightMap1"`
	HeightMap2   int  `json:"heightMap2"`
	Population   bool `json:"population"`
	ShaderPreset int  `json:"shaderPreset"`
}

// CelestialStatistics represents physical statistics for planets, moons, and asteroid belts
type CelestialStatistics struct {
	Age            *int64  `json:"age,omitempty"`
	Density        float64 `json:"density"`
	Eccentricity   float64 `json:"eccentricity"`
	EscapeVelocity float64 `json:"escapeVelocity"`
	Fragmented     bool    `json:"fragmented"`
	Life           float64 `json:"life"`
	Locked         bool    `json:"locked"`
	MassDust       float64 `json:"massDust"`
	MassGas        float64 `json:"massGas"`
	OrbitPeriod    float64 `json:"orbitPeriod"`
	OrbitRadius    float64 `json:"orbitRadius"`
	Pressure       float64 `json:"pressure"`
	Radius         float64 `json:"radius"`
	RotationRate   float64 `json:"rotationRate"`
	SpectralClass  string  `json:"spectralClass"`
	SurfaceGravity float64 `json:"surfaceGravity"`
	Temperature    float64 `json:"temperature"`
}

// StarStatistics represents statistics for stars
type StarStatistics struct {
	Age           int64   `json:"age"`
	Life          int64   `json:"life"`
	Locked        bool    `json:"locked"`
	Luminosity    float64 `json:"luminosity"`
	Radius        float64 `json:"radius"`
	SpectralClass string  `json:"spectralClass"`
	Temperature   int     `json:"temperature"`
}
