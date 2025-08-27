package models

import "time"

// ImportStatus represents the status of an SDE import operation
type ImportStatus struct {
	ID        string         `json:"id" bson:"_id"`
	Status    string         `json:"status" bson:"status"` // pending, running, completed, failed
	StartTime *time.Time     `json:"start_time" bson:"start_time,omitempty"`
	EndTime   *time.Time     `json:"end_time" bson:"end_time,omitempty"`
	Progress  ImportProgress `json:"progress" bson:"progress"`
	Error     string         `json:"error,omitempty" bson:"error,omitempty"`
	CreatedAt time.Time      `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" bson:"updated_at"`
}

// ImportProgress tracks the progress of importing different SDE data types
type ImportProgress struct {
	TotalSteps     int                       `json:"total_steps" bson:"total_steps"`
	CompletedSteps int                       `json:"completed_steps" bson:"completed_steps"`
	CurrentStep    string                    `json:"current_step" bson:"current_step"`
	DataTypes      map[string]DataTypeStatus `json:"data_types" bson:"data_types"`
}

// DataTypeStatus represents the import status of a specific data type
type DataTypeStatus struct {
	Name      string `json:"name" bson:"name"`
	Status    string `json:"status" bson:"status"` // pending, processing, completed, failed
	Count     int    `json:"count" bson:"count"`
	Processed int    `json:"processed" bson:"processed"`
	Error     string `json:"error,omitempty" bson:"error,omitempty"`
}

// SDEDataType represents the different types of SDE data that can be imported
type SDEDataType string

const (
	DataTypeAgents          SDEDataType = "agents"
	DataTypeCategories      SDEDataType = "categories"
	DataTypeBlueprints      SDEDataType = "blueprints"
	DataTypeMarketGroups    SDEDataType = "marketGroups"
	DataTypeMetaGroups      SDEDataType = "metaGroups"
	DataTypeNPCCorporations SDEDataType = "npcCorporations"
	DataTypeTypeIDs         SDEDataType = "typeIDs"
	DataTypeTypes           SDEDataType = "types"
	DataTypeTypeMaterials   SDEDataType = "typeMaterials"
)

// GetAllDataTypes returns all available SDE data types
func GetAllDataTypes() []SDEDataType {
	return []SDEDataType{
		DataTypeAgents,
		DataTypeCategories,
		DataTypeBlueprints,
		DataTypeMarketGroups,
		DataTypeMetaGroups,
		DataTypeNPCCorporations,
		DataTypeTypeIDs,
		DataTypeTypes,
		DataTypeTypeMaterials,
	}
}
