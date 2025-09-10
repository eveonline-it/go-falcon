package dto

// GetCharacterProfileInput represents the input for getting a character profile
type GetCharacterProfileInput struct {
	CharacterID int `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
}

// GetCharacterProfileAuthInput represents the authenticated input for getting a character profile
type GetCharacterProfileAuthInput struct {
	CharacterID   int    `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Authorization string `header:"Authorization" doc:"JWT Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// SearchCharactersByNameInput represents the input for searching characters by name
type SearchCharactersByNameInput struct {
	Name string `query:"name" validate:"required" minLength:"3" maxLength:"100" doc:"Character name to search for (minimum 3 characters)"`
}

// SearchCharactersByNameAuthInput represents the authenticated input for searching characters by name
type SearchCharactersByNameAuthInput struct {
	Name          string `query:"name" validate:"required" minLength:"3" maxLength:"100" doc:"Character name to search for (minimum 3 characters)"`
	Authorization string `header:"Authorization" doc:"JWT Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// GetCharacterAttributesInput represents the authenticated input for getting character attributes
type GetCharacterAttributesInput struct {
	CharacterID   int    `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Authorization string `header:"Authorization" doc:"JWT Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// GetCharacterSkillQueueInput represents the authenticated input for getting character skill queue
type GetCharacterSkillQueueInput struct {
	CharacterID   int    `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Authorization string `header:"Authorization" doc:"JWT Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// GetCharacterSkillsInput represents the authenticated input for getting character skills
type GetCharacterSkillsInput struct {
	CharacterID   int    `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Authorization string `header:"Authorization" doc:"JWT Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// GetCharacterCorporationHistoryInput represents the authenticated input for getting character corporation history
type GetCharacterCorporationHistoryInput struct {
	CharacterID   int    `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Authorization string `header:"Authorization" doc:"JWT Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// GetCharacterClonesInput represents the authenticated input for getting character clones
type GetCharacterClonesInput struct {
	CharacterID   int    `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Authorization string `header:"Authorization" doc:"JWT Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// GetCharacterImplantsInput represents the authenticated input for getting character implants
type GetCharacterImplantsInput struct {
	CharacterID   int    `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Authorization string `header:"Authorization" doc:"JWT Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}
