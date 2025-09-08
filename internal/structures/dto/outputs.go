package dto

import (
	"time"

	"go-falcon/internal/structures/models"
)

// StructureResponse represents a structure in the response
type StructureResponse struct {
	StructureID       int64      `json:"structure_id" doc:"EVE structure ID"`
	Name              string     `json:"name" doc:"Structure name"`
	OwnerID           int32      `json:"owner_id" doc:"Owner corporation ID"`
	SolarSystemID     int32      `json:"solar_system_id" doc:"Solar system ID"`
	SolarSystemName   string     `json:"solar_system_name,omitempty" doc:"Solar system name"`
	RegionID          int32      `json:"region_id,omitempty" doc:"Region ID"`
	RegionName        string     `json:"region_name,omitempty" doc:"Region name"`
	ConstellationID   int32      `json:"constellation_id,omitempty" doc:"Constellation ID"`
	ConstellationName string     `json:"constellation_name,omitempty" doc:"Constellation name"`
	TypeID            int32      `json:"type_id" doc:"Structure type ID"`
	TypeName          string     `json:"type_name,omitempty" doc:"Structure type name"`
	IsNPCStation      bool       `json:"is_npc_station" doc:"Whether this is an NPC station"`
	Services          []string   `json:"services,omitempty" doc:"Available services"`
	State             string     `json:"state,omitempty" doc:"Structure state"`
	FuelExpires       *time.Time `json:"fuel_expires,omitempty" doc:"When fuel expires"`
	UpdatedAt         time.Time  `json:"updated_at" doc:"Last update time"`
}

// StructureListResponse represents a list of structures
type StructureListResponse struct {
	Structures []StructureResponse `json:"structures" doc:"List of structures"`
	Total      int                 `json:"total" doc:"Total number of structures"`
	Page       int                 `json:"page,omitempty" doc:"Current page"`
	PageSize   int                 `json:"page_size,omitempty" doc:"Page size"`
}

// StructureAccessResponse represents structure access information
type StructureAccessResponse struct {
	StructureID int64     `json:"structure_id" doc:"EVE structure ID"`
	CharacterID int32     `json:"character_id" doc:"Character ID"`
	HasAccess   bool      `json:"has_access" doc:"Whether character has access"`
	LastChecked time.Time `json:"last_checked" doc:"Last access check time"`
}

// BulkRefreshResponse represents the result of a bulk refresh operation
type BulkRefreshResponse struct {
	Refreshed []int64  `json:"refreshed" doc:"Successfully refreshed structure IDs"`
	Failed    []int64  `json:"failed" doc:"Failed structure IDs"`
	Errors    []string `json:"errors,omitempty" doc:"Error messages"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body StructureModuleStatusResponse `json:"body"`
}

// StructureOutput represents a single structure response
type StructureOutput struct {
	Body StructureResponse `json:"body"`
}

// StructureListOutput represents a list of structures response
type StructureListOutput struct {
	Body StructureListResponse `json:"body"`
}

// StructureModuleStatusResponse represents the actual status response data
type StructureModuleStatusResponse struct {
	Module  string `json:"module" description:"Module name"`
	Status  string `json:"status" enum:"healthy,degraded,unhealthy" description:"Module health status"`
	Message string `json:"message,omitempty" description:"Optional status message or error details"`
}

// ToStructureResponse converts a model to a response DTO
func ToStructureResponse(structure *models.Structure) StructureResponse {
	return StructureResponse{
		StructureID:       structure.StructureID,
		Name:              structure.Name,
		OwnerID:           structure.OwnerID,
		SolarSystemID:     structure.SolarSystemID,
		SolarSystemName:   structure.SolarSystemName,
		RegionID:          structure.RegionID,
		RegionName:        structure.RegionName,
		ConstellationID:   structure.ConstellationID,
		ConstellationName: structure.ConstellationName,
		TypeID:            structure.TypeID,
		TypeName:          structure.TypeName,
		IsNPCStation:      structure.IsNPCStation,
		Services:          structure.Services,
		State:             structure.State,
		FuelExpires:       structure.FuelExpires,
		UpdatedAt:         structure.UpdatedAt,
	}
}
