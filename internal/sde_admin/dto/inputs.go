package dto

// AuthInput provides common authentication headers for secured endpoints
type AuthInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication" example:"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Cookie        string `header:"Cookie" doc:"Authentication cookie" example:"falcon_auth_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// ReloadSDERequest represents a request to reload SDE data from files
type ReloadSDERequest struct {
	// DataTypes specifies which SDE data types to reload
	// If empty, all data types will be reloaded
	DataTypes []string `json:"data_types,omitempty" example:"[\"agents\",\"types\"]" doc:"List of SDE data types to reload. Leave empty to reload all types."`
}

// CheckUpdatesRequest represents a request to check for SDE updates
type CheckUpdatesRequest struct {
	// Sources specifies which SDE sources to check
	// If empty, all configured sources will be checked
	Sources []string `json:"sources,omitempty" example:"[\"ccp-github\",\"hoboleaks\"]" doc:"List of SDE sources to check. Leave empty to check all configured sources."`
	// Force bypasses cache and forces a fresh check
	Force bool `json:"force,omitempty" doc:"Force check even if recently checked"`
}

// UpdateSDERequest represents a request to download and update SDE data
type UpdateSDERequest struct {
	// Source specifies which SDE source to use for update
	Source string `json:"source" example:"ccp-github" doc:"SDE source to download from (ccp-github, hoboleaks, custom)"`
	// Format specifies the expected format (yaml, json)
	Format string `json:"format,omitempty" example:"yaml" doc:"Expected data format (yaml, json). Auto-detected if not specified."`
	// URL for custom source downloads
	URL string `json:"url,omitempty" example:"https://github.com/ccpgames/eve-sde/archive/refs/heads/master.zip" doc:"Custom URL for SDE download (required for custom source)"`
	// BackupCurrent creates backup of current data before update
	BackupCurrent bool `json:"backup_current,omitempty" doc:"Create backup of current SDE data before updating"`
	// ConvertToJSON converts YAML files to JSON during processing
	ConvertToJSON bool `json:"convert_to_json" example:"true" doc:"Convert YAML files to JSON format"`
}

// RestoreBackupRequest represents a request to restore from backup
type RestoreBackupRequest struct {
	// BackupID specifies which backup to restore
	BackupID string `json:"backup_id" example:"backup_20241128_143022" doc:"Backup ID to restore from"`
	// DeleteBackup removes the backup after successful restore
	DeleteBackup bool `json:"delete_backup,omitempty" doc:"Delete backup after successful restore"`
}
