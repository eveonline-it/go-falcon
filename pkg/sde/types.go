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
