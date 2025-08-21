package dto

// GetCharacterProfileInput represents the input for getting a character profile
type GetCharacterProfileInput struct {
	CharacterID int `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
}