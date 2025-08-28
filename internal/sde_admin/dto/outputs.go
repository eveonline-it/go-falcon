package dto

import (
	"go-falcon/pkg/sde"
)

// StatusOutput represents the output for module status endpoint
type StatusOutput struct {
	Body SDEStatusResponse `json:"body"`
}

// SDEStatusResponse represents the module status response
type SDEStatusResponse struct {
	Module  string `json:"module" doc:"Module name"`
	Status  string `json:"status" doc:"Module health status"`
	Message string `json:"message,omitempty" doc:"Additional status message"`
}

// ReloadSDEOutput represents the output for reloading SDE data
type ReloadSDEOutput struct {
	Body ReloadSDEResponse `json:"body"`
}

// ReloadSDEResponse represents the response after reloading SDE data
type ReloadSDEResponse struct {
	Success    bool     `json:"success" doc:"Whether the reload operation was successful"`
	Message    string   `json:"message" doc:"Human-readable status message"`
	DataTypes  []string `json:"data_types" doc:"List of data types that were reloaded"`
	Duration   string   `json:"duration,omitempty" doc:"Duration of the reload operation"`
	ReloadedAt string   `json:"reloaded_at" doc:"Timestamp when the reload was completed"`
	Error      string   `json:"error,omitempty" doc:"Error message if reload failed"`
}

// MemoryStatusOutput represents the output for memory status endpoint
type MemoryStatusOutput struct {
	Body MemoryStatusResponse `json:"body"`
}

// MemoryStatusResponse represents the current in-memory data status
type MemoryStatusResponse struct {
	LoadedDataTypes  []string                          `json:"loaded_data_types" doc:"List of currently loaded data types"`
	TotalMemoryUsage int64                             `json:"total_memory_usage" doc:"Total estimated memory usage in bytes"`
	TotalDataTypes   int                               `json:"total_data_types" doc:"Total number of data types loaded"`
	TotalItems       int                               `json:"total_items" doc:"Total number of items across all data types"`
	DataTypeStatuses map[string]DataTypeStatusInMemory `json:"data_type_statuses" doc:"Detailed status of each data type"`
	IsLoaded         bool                              `json:"is_loaded" doc:"Whether SDE data is loaded in memory"`
	LastReloaded     *string                           `json:"last_reloaded,omitempty" doc:"Timestamp when data was last reloaded"`
}

// DataTypeStatusInMemory represents the status of a data type in memory
type DataTypeStatusInMemory struct {
	Name        string `json:"name" doc:"Name of the data type"`
	Loaded      bool   `json:"loaded" doc:"Whether this data type is loaded"`
	Count       int    `json:"count" doc:"Number of items loaded"`
	MemoryBytes int64  `json:"memory_bytes" doc:"Estimated memory usage in bytes"`
	FilePath    string `json:"file_path" doc:"Source file path"`
}

// SDEStatsOutput represents the output for SDE statistics endpoint
type SDEStatsOutput struct {
	Body SDEStatsResponse `json:"body"`
}

// SDEStatsResponse represents statistics about SDE data in memory
type SDEStatsResponse struct {
	TotalItems      int                              `json:"total_items" doc:"Total number of items loaded in memory"`
	TotalMemoryUsed int64                            `json:"total_memory_used" doc:"Total estimated memory usage in bytes"`
	DataTypes       map[string]DataTypeStatsResponse `json:"data_types" doc:"Statistics for each data type"`
	IsLoaded        bool                             `json:"is_loaded" doc:"Whether SDE data is loaded in memory"`
	LoadedCount     int                              `json:"loaded_count" doc:"Number of data types loaded"`
}

// DataTypeStatsResponse represents statistics for a specific data type
type DataTypeStatsResponse struct {
	Count       int    `json:"count" doc:"Number of items loaded"`
	MemoryBytes int64  `json:"memory_bytes" doc:"Estimated memory used in bytes"`
	Loaded      bool   `json:"loaded" doc:"Whether this data type is loaded"`
	FilePath    string `json:"file_path" doc:"Source file path"`
}

// ConvertToMemoryStatus converts SDE service data to MemoryStatusResponse
func ConvertToMemoryStatus(loadStatus map[string]sde.DataTypeStatus, loadedTypes []string, totalMemory int64, isLoaded bool) *MemoryStatusResponse {
	totalItems := 0
	dataTypeStatuses := make(map[string]DataTypeStatusInMemory)

	for name, status := range loadStatus {
		totalItems += status.Count
		dataTypeStatuses[name] = DataTypeStatusInMemory{
			Name:        status.Name,
			Loaded:      status.Loaded,
			Count:       status.Count,
			MemoryBytes: status.MemoryBytes,
			FilePath:    "", // Will be populated by the service
		}
	}

	return &MemoryStatusResponse{
		LoadedDataTypes:  loadedTypes,
		TotalMemoryUsage: totalMemory,
		TotalDataTypes:   len(loadedTypes),
		TotalItems:       totalItems,
		DataTypeStatuses: dataTypeStatuses,
		IsLoaded:         isLoaded,
	}
}

// ConvertToStatsResponse converts SDE service data to SDEStatsResponse
func ConvertToStatsResponse(loadStatus map[string]sde.DataTypeStatus, totalMemory int64, isLoaded bool) *SDEStatsResponse {
	totalItems := 0
	loadedCount := 0
	dataTypes := make(map[string]DataTypeStatsResponse)

	for name, status := range loadStatus {
		totalItems += status.Count
		if status.Loaded {
			loadedCount++
		}

		dataTypes[name] = DataTypeStatsResponse{
			Count:       status.Count,
			MemoryBytes: status.MemoryBytes,
			Loaded:      status.Loaded,
			FilePath:    "", // Will be populated by the service
		}
	}

	return &SDEStatsResponse{
		TotalItems:      totalItems,
		TotalMemoryUsed: totalMemory,
		DataTypes:       dataTypes,
		IsLoaded:        isLoaded,
		LoadedCount:     loadedCount,
	}
}

// VerificationOutput represents the output for data verification endpoint
type VerificationOutput struct {
	Body VerificationResponse `json:"body"`
}

// VerificationResponse represents the result of data integrity verification
type VerificationResponse struct {
	Status         string   `json:"status" doc:"Overall health status: healthy, warning, critical"`
	HealthScore    float64  `json:"health_score" doc:"Health score from 0-100"`
	TotalDataTypes int      `json:"total_data_types" doc:"Total number of data types"`
	LoadedTypes    int      `json:"loaded_types" doc:"Number of successfully loaded data types"`
	Issues         []string `json:"issues" doc:"List of detected issues"`
	VerifiedAt     string   `json:"verified_at" doc:"Timestamp when verification was performed"`
}

// SystemInfoOutput represents the output for system information endpoint
type SystemInfoOutput struct {
	Body SystemInfoResponse `json:"body"`
}

// SystemInfoResponse represents system information
type SystemInfoResponse struct {
	IsLoaded          bool    `json:"is_loaded" doc:"Whether SDE data is loaded"`
	LoadedDataTypes   int     `json:"loaded_data_types" doc:"Number of loaded data types"`
	EstimatedMemoryMB float64 `json:"estimated_memory_mb" doc:"Estimated memory usage in MB"`
	SystemMemoryMB    float64 `json:"system_memory_mb" doc:"Current system memory usage in MB"`
	GoRoutines        int     `json:"go_routines" doc:"Number of active goroutines"`
	Timestamp         string  `json:"timestamp" doc:"Current timestamp"`
}

// CheckUpdatesOutput represents the output for checking SDE updates
type CheckUpdatesOutput struct {
	Body CheckUpdatesResponse `json:"body"`
}

// CheckUpdatesResponse represents the result of checking for SDE updates from CCP official source
type CheckUpdatesResponse struct {
	UpdatesAvailable bool            `json:"updates_available" doc:"Whether updates are available"`
	CurrentVersion   string          `json:"current_version,omitempty" doc:"Current SDE version/hash"`
	LatestVersion    string          `json:"latest_version,omitempty" doc:"Latest available SDE version/hash"`
	CCPOfficial      SDESourceStatus `json:"ccp_official" doc:"Status of CCP official SDE source"`
	CheckedAt        string          `json:"checked_at" doc:"Timestamp when check was performed"`
}

// SDESourceStatus represents the status of an SDE source
type SDESourceStatus struct {
	Name          string  `json:"name" doc:"Source name (ccp-github, hoboleaks, etc.)"`
	Available     bool    `json:"available" doc:"Whether source is reachable"`
	LatestVersion string  `json:"latest_version,omitempty" doc:"Latest version/hash from this source"`
	LatestSize    int64   `json:"latest_size,omitempty" doc:"Size of latest version in bytes"`
	URL           string  `json:"url,omitempty" doc:"Source URL"`
	LastChecked   string  `json:"last_checked" doc:"Last time this source was checked"`
	Error         *string `json:"error,omitempty" doc:"Error message if source check failed"`
}

// UpdateSDEOutput represents the output for updating SDE data
type UpdateSDEOutput struct {
	Body UpdateSDEResponse `json:"body"`
}

// UpdateSDEResponse represents the result of an SDE update operation
type UpdateSDEResponse struct {
	Success        bool             `json:"success" doc:"Whether the update was successful"`
	Message        string           `json:"message" doc:"Human-readable status message"`
	Source         string           `json:"source" doc:"Source used for update"`
	OldVersion     string           `json:"old_version,omitempty" doc:"Previous SDE version"`
	NewVersion     string           `json:"new_version,omitempty" doc:"New SDE version"`
	UpdatedAt      string           `json:"updated_at" doc:"Timestamp when update was completed"`
	Duration       string           `json:"duration,omitempty" doc:"Duration of update operation"`
	DownloadedSize int64            `json:"downloaded_size,omitempty" doc:"Size of downloaded data in bytes"`
	ExtractedFiles int              `json:"extracted_files,omitempty" doc:"Number of files extracted"`
	ConvertedFiles int              `json:"converted_files,omitempty" doc:"Number of files converted from YAML to JSON"`
	ProcessingLog  []UpdateLogEntry `json:"processing_log,omitempty" doc:"Detailed processing log"`
	Error          *string          `json:"error,omitempty" doc:"Error message if update failed"`
}

// UpdateLogEntry represents a single entry in the update processing log
type UpdateLogEntry struct {
	Timestamp string `json:"timestamp" doc:"When this step occurred"`
	Step      string `json:"step" doc:"Update step name"`
	Message   string `json:"message" doc:"Step message"`
	Duration  string `json:"duration,omitempty" doc:"Duration of this step"`
	Success   bool   `json:"success" doc:"Whether this step succeeded"`
}
