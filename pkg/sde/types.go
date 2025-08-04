package sde

// Agent represents an EVE Online agent from the SDE
type Agent struct {
	AgentTypeID    int  `json:"agentTypeID"`
	CorporationID  int  `json:"corporationID"`
	DivisionID     int  `json:"divisionID"`
	IsLocator      bool `json:"isLocator"`
	Level          int  `json:"level"`
	LocationID     int  `json:"locationID"`
}

// Category represents an EVE Online item category from the SDE
type Category struct {
	Name      map[string]string `json:"name"`      // Internationalized names
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