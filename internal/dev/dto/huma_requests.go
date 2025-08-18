package dto

// HealthCheckInput represents the input for health check endpoint
type HealthCheckInput struct {
	// No body needed for health check
}

// HealthCheckOutput represents the output for health check endpoint
type HealthCheckOutput struct {
	Body DevHealthResponse `json:"body"`
}

// ESIStatusInput represents the input for ESI status endpoint
type ESIStatusInput struct {
	// No body needed for ESI status
}

// ESIStatusOutput represents the output for ESI status endpoint
type ESIStatusOutput struct {
	Body ESIStatusResponse `json:"body"`
}

// CharacterInfoInput represents the input for character information
type CharacterInfoInput struct {
	CharacterID     int    `path:"character_id" validate:"required" minimum:"1" maximum:"2147483647" doc:"EVE Online character ID"`
	Authorization   string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie          string `header:"Cookie" doc:"Session cookie for authentication"`
}

// CharacterInfoOutput represents the output for character information
type CharacterInfoOutput struct {
	Body CharacterResponse `json:"body"`
}

// SDEStatusInput represents the input for SDE status endpoint
type SDEStatusInput struct {
	// No body needed for SDE status
}

// SDEStatusOutput represents the output for SDE status endpoint
type SDEStatusOutput struct {
	Body DevSDEStatusResponse `json:"body"`
}

// ServiceDiscoveryInput represents the input for service discovery
type ServiceDiscoveryInput struct {
	ServiceName string `query:"service_name" validate:"omitempty" maxLength:"100" doc:"Optional service name to filter"`
	Detailed    bool   `query:"detailed" doc:"Include detailed endpoint information"`
}

// ServiceDiscoveryOutput represents the output for service discovery
type ServiceDiscoveryOutput struct {
	Body ServiceDiscoveryResponse `json:"body"`
}

// AllianceInfoInput represents the input for alliance information
type AllianceInfoInput struct {
	AllianceID int `path:"alliance_id" validate:"required" minimum:"1" maximum:"2147483647" doc:"EVE Online alliance ID"`
}

// AllianceInfoOutput represents the output for alliance information
type AllianceInfoOutput struct {
	Body AllianceResponse `json:"body"`
}

// CorporationInfoInput represents the input for corporation information
type CorporationInfoInput struct {
	CorporationID int `path:"corporation_id" validate:"required" minimum:"1" maximum:"2147483647" doc:"EVE Online corporation ID"`
}

// CorporationInfoOutput represents the output for corporation information
type CorporationInfoOutput struct {
	Body CorporationResponse `json:"body"`
}

// SystemInfoInput represents the input for solar system information
type SystemInfoInput struct {
	SystemID int `path:"system_id" validate:"required" minimum:"1" maximum:"2147483647" doc:"EVE Online solar system ID"`
}

// SystemInfoOutput represents the output for solar system information
type SystemInfoOutput struct {
	Body SystemResponse `json:"body"`
}

// SDEEntityInput represents the input for SDE entity data
type SDEEntityInput struct {
	Type string `path:"type" validate:"required" minLength:"1" maxLength:"50" doc:"SDE entity type (e.g., agents, types, categories)"`
	ID   string `path:"id" validate:"required" minLength:"1" maxLength:"100" doc:"Entity ID"`
}

// SDEEntityOutput represents the output for SDE entity data
type SDEEntityOutput struct {
	Body SDEEntityResponse `json:"body"`
}

// SDETypesInput represents the input for SDE types collection
type SDETypesInput struct {
	Published bool `query:"published" doc:"Filter to only published types"`
}

// SDETypesOutput represents the output for SDE types collection
type SDETypesOutput struct {
	Body SDETypesResponse `json:"body"`
}

// UniverseDataInput represents the input for universe data
type UniverseDataInput struct {
	Type          string `path:"type" validate:"required,oneof=eve abyssal wormhole void hidden" doc:"Universe type"`
	Region        string `path:"region" validate:"required" minLength:"1" maxLength:"100" doc:"Region name"`
	Constellation string `path:"constellation" validate:"omitempty" minLength:"1" maxLength:"100" doc:"Constellation name (optional)"`
	System        string `path:"system" validate:"omitempty" minLength:"1" maxLength:"100" doc:"System name (optional)"`
}

// UniverseDataOutput represents the output for universe data
type UniverseDataOutput struct {
	Body UniverseResponse `json:"body"`
}

// UniverseSystemsInput represents the input for universe systems collection
type UniverseSystemsInput struct {
	Type          string `path:"type" validate:"required,oneof=eve abyssal wormhole void hidden" doc:"Universe type"`
	Region        string `path:"region" validate:"required" minLength:"1" maxLength:"100" doc:"Region name"`
	Constellation string `path:"constellation" validate:"omitempty" minLength:"1" maxLength:"100" doc:"Constellation name (optional)"`
}

// UniverseSystemsOutput represents the output for universe systems collection
type UniverseSystemsOutput struct {
	Body UniverseSystemsResponse `json:"body"`
}

// RedisSDEEntityInput represents the input for Redis SDE entity data
type RedisSDEEntityInput struct {
	Type string `path:"type" validate:"required" minLength:"1" maxLength:"50" doc:"SDE entity type in Redis"`
	ID   string `path:"id" validate:"omitempty" minLength:"1" maxLength:"100" doc:"Entity ID (optional for collection queries)"`
}

// RedisSDEEntityOutput represents the output for Redis SDE entity data
type RedisSDEEntityOutput struct {
	Body SDEEntityResponse `json:"body"`
}