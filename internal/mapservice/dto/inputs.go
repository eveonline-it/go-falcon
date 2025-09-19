package dto

// CreateSignatureInput represents the input for creating a signature
type CreateSignatureInput struct {
	SystemID     int32   `json:"system_id" validate:"required" doc:"EVE System ID"`
	SignatureID  string  `json:"signature_id" validate:"required,min=3,max=7" doc:"In-game signature ID (e.g., ABC-123)"`
	Type         string  `json:"type" validate:"required,oneof=Combat Data Relic Gas Wormhole Unknown" doc:"Signature type"`
	Name         string  `json:"name,omitempty" validate:"max=100" doc:"Optional signature name"`
	Description  string  `json:"description,omitempty" validate:"max=500" doc:"Optional description"`
	Strength     float32 `json:"strength,omitempty" validate:"min=0,max=100" doc:"Signal strength percentage"`
	SharingLevel string  `json:"sharing_level" validate:"required,oneof=private corporation alliance" doc:"Visibility level"`
	ExpiresIn    int     `json:"expires_in,omitempty" validate:"min=0,max=72" doc:"Hours until expiration (0 = no expiration)"`
}

// UpdateSignatureInput represents the input for updating a signature
type UpdateSignatureInput struct {
	Type        *string  `json:"type,omitempty" validate:"omitempty,oneof=Combat Data Relic Gas Wormhole Unknown" doc:"Signature type"`
	Name        *string  `json:"name,omitempty" validate:"omitempty,max=100" doc:"Signature name"`
	Description *string  `json:"description,omitempty" validate:"omitempty,max=500" doc:"Description"`
	Strength    *float32 `json:"strength,omitempty" validate:"omitempty,min=0,max=100" doc:"Signal strength"`
}

// CreateWormholeInput represents the input for creating a wormhole connection
type CreateWormholeInput struct {
	FromSystemID    int32  `json:"from_system_id" validate:"required" doc:"Origin system ID"`
	ToSystemID      int32  `json:"to_system_id" validate:"required" doc:"Destination system ID"`
	FromSignatureID string `json:"from_signature_id" validate:"required,min=3,max=7" doc:"Origin signature ID"`
	ToSignatureID   string `json:"to_signature_id,omitempty" validate:"min=3,max=7" doc:"Destination signature ID (K162)"`
	WormholeType    string `json:"wormhole_type,omitempty" validate:"max=10" doc:"Wormhole type code (e.g., B274)"`
	MassStatus      string `json:"mass_status" validate:"required,oneof=stable destabilized critical" doc:"Mass status"`
	TimeStatus      string `json:"time_status" validate:"required,oneof=stable eol" doc:"Time status (eol = end of life)"`
	SharingLevel    string `json:"sharing_level" validate:"required,oneof=private corporation alliance" doc:"Visibility level"`
}

// UpdateWormholeInput represents the input for updating a wormhole
type UpdateWormholeInput struct {
	ToSignatureID *string `json:"to_signature_id,omitempty" validate:"omitempty,min=3,max=7" doc:"K162 signature on other side"`
	WormholeType  *string `json:"wormhole_type,omitempty" validate:"omitempty,max=10" doc:"Wormhole type code"`
	MassStatus    *string `json:"mass_status,omitempty" validate:"omitempty,oneof=stable destabilized critical" doc:"Mass status"`
	TimeStatus    *string `json:"time_status,omitempty" validate:"omitempty,oneof=stable eol" doc:"Time status"`
}

// CreateNoteInput represents the input for creating a map note
type CreateNoteInput struct {
	SystemID     int32   `json:"system_id" validate:"required" doc:"EVE System ID"`
	Text         string  `json:"text" validate:"required,min=1,max=500" doc:"Note text"`
	Size         string  `json:"size" validate:"required,oneof=small medium large" doc:"Note size"`
	Color        string  `json:"color,omitempty" validate:"omitempty,hexcolor" doc:"Note color (hex)"`
	PosX         float32 `json:"pos_x,omitempty" doc:"X position on map"`
	PosY         float32 `json:"pos_y,omitempty" doc:"Y position on map"`
	SharingLevel string  `json:"sharing_level" validate:"required,oneof=private corporation alliance" doc:"Visibility level"`
	ExpiresIn    int     `json:"expires_in,omitempty" validate:"min=0,max=168" doc:"Hours until expiration (0 = no expiration)"`
}

// RouteInput represents the input for calculating a route
type RouteInput struct {
	FromSystemID   int32   `json:"from_system_id" query:"from" validate:"required" doc:"Origin system ID"`
	ToSystemID     int32   `json:"to_system_id" query:"to" validate:"required" doc:"Destination system ID"`
	RouteType      string  `json:"route_type" query:"type" validate:"required,oneof=shortest safest avoid_null" doc:"Route calculation type"`
	IncludeWH      bool    `json:"include_wh" query:"include_wh" doc:"Include wormhole connections"`
	IncludeThera   bool    `json:"include_thera" query:"include_thera" doc:"Include Thera connections"`
	AvoidSystemIDs []int32 `json:"avoid_systems" query:"avoid" doc:"System IDs to avoid"`
}

// GetSignaturesInput represents the input for retrieving signatures
type GetSignaturesInput struct {
	SystemID       int32  `query:"system_id" doc:"Filter by system ID"`
	Type           string `query:"type" validate:"omitempty,oneof=Combat Data Relic Gas Wormhole Unknown" doc:"Filter by type"`
	SharingLevel   string `query:"sharing" validate:"omitempty,oneof=private corporation alliance all" doc:"Filter by sharing level"`
	IncludeExpired bool   `query:"include_expired" doc:"Include expired signatures"`
}

// GetWormholesInput represents the input for retrieving wormholes
type GetWormholesInput struct {
	SystemID       int32  `query:"system_id" doc:"Filter by system ID (either side)"`
	SharingLevel   string `query:"sharing" validate:"omitempty,oneof=private corporation alliance all" doc:"Filter by sharing level"`
	IncludeExpired bool   `query:"include_expired" doc:"Include expired connections"`
}

// BatchSignatureInput represents bulk signature operations
type BatchSignatureInput struct {
	SystemID   int32                  `json:"system_id" validate:"required" doc:"System ID for all signatures"`
	Signatures []CreateSignatureInput `json:"signatures" validate:"required,min=1,max=100,dive" doc:"List of signatures to create"`
	DeleteOld  bool                   `json:"delete_old" doc:"Delete existing signatures not in this list"`
}

// SystemActivityInput represents the input for getting system activity
type SystemActivityInput struct {
	SystemIDs []int32 `json:"system_ids" query:"systems" validate:"required,min=1,max=100" doc:"System IDs to fetch activity for"`
}

// UpdateNoteInput represents the input for updating a map note
type UpdateNoteInput struct {
	Text  *string  `json:"text,omitempty" validate:"omitempty,min=1,max=500" doc:"Note text"`
	Size  *string  `json:"size,omitempty" validate:"omitempty,oneof=small medium large" doc:"Note size"`
	Color *string  `json:"color,omitempty" validate:"omitempty,hexcolor" doc:"Note color"`
	PosX  *float32 `json:"pos_x,omitempty" doc:"X position"`
	PosY  *float32 `json:"pos_y,omitempty" doc:"Y position"`
}

// SearchSystemInput represents the input for searching systems
type SearchSystemInput struct {
	Query string `query:"q" validate:"required,min=2" doc:"Search query"`
	Limit int    `query:"limit" validate:"min=1,max=100" doc:"Maximum results" default:"20"`
}

// Authentication-enabled input DTOs for protected endpoints

// CreateSignatureInputWithAuth adds authentication to CreateSignatureInput
type CreateSignatureInputWithAuth struct {
	CreateSignatureInput
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// GetSignaturesInputWithAuth adds authentication to GetSignaturesInput
type GetSignaturesInputWithAuth struct {
	GetSignaturesInput
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// GetSignatureInputWithAuth for getting a specific signature by ID
type GetSignatureInputWithAuth struct {
	SignatureID   string `path:"signature_id" doc:"Signature ID"`
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// UpdateSignatureInputWithAuth adds authentication to UpdateSignatureInput
type UpdateSignatureInputWithAuth struct {
	SignatureID          string `path:"signature_id" doc:"Signature ID"`
	UpdateSignatureInput `json:",inline"`
	Authorization        string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie               string `header:"Cookie" doc:"Authentication cookie"`
}

// DeleteSignatureInputWithAuth for deleting a signature
type DeleteSignatureInputWithAuth struct {
	SignatureID   string `path:"signature_id" doc:"Signature ID"`
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// BatchSignatureInputWithAuth adds authentication to BatchSignatureInput
type BatchSignatureInputWithAuth struct {
	BatchSignatureInput
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// Wormhole authentication-enabled input DTOs

// CreateWormholeInputWithAuth adds authentication to CreateWormholeInput
type CreateWormholeInputWithAuth struct {
	CreateWormholeInput
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// GetWormholesInputWithAuth adds authentication to GetWormholesInput
type GetWormholesInputWithAuth struct {
	GetWormholesInput
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// GetWormholeInputWithAuth for getting a specific wormhole by ID
type GetWormholeInputWithAuth struct {
	WormholeID    string `path:"wormhole_id" doc:"Wormhole ID"`
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// UpdateWormholeInputWithAuth adds authentication to UpdateWormholeInput
type UpdateWormholeInputWithAuth struct {
	WormholeID          string `path:"wormhole_id" doc:"Wormhole ID"`
	UpdateWormholeInput `json:",inline"`
	Authorization       string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie              string `header:"Cookie" doc:"Authentication cookie"`
}

// DeleteWormholeInputWithAuth for deleting a wormhole
type DeleteWormholeInputWithAuth struct {
	WormholeID    string `path:"wormhole_id" doc:"Wormhole ID"`
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// BatchWormholeInput represents bulk wormhole operations
type BatchWormholeInput struct {
	SystemID  int32                 `json:"system_id" validate:"required" doc:"System ID for all wormholes"`
	Wormholes []CreateWormholeInput `json:"wormholes" validate:"required,min=1,max=50,dive" doc:"List of wormholes to create"`
	DeleteOld bool                  `json:"delete_old" doc:"Delete existing wormholes not in this list"`
}

// BatchWormholeInputWithAuth adds authentication to BatchWormholeInput
type BatchWormholeInputWithAuth struct {
	BatchWormholeInput
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}
