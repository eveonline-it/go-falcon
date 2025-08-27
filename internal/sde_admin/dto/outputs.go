package dto

import (
	"go-falcon/internal/sde_admin/models"
	"time"
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

// ImportSDEOutput represents the output for starting an SDE import
type ImportSDEOutput struct {
	Body ImportSDEResponse `json:"body"`
}

// ImportSDEResponse represents the response after starting an SDE import
type ImportSDEResponse struct {
	ImportID  string `json:"import_id" doc:"Unique ID for tracking this import operation"`
	Status    string `json:"status" doc:"Current status of the import operation"`
	Message   string `json:"message" doc:"Human-readable status message"`
	StartTime string `json:"start_time" doc:"Timestamp when the import was started"`
}

// ImportStatusOutput represents the output for import status endpoint
type ImportStatusOutput struct {
	Body ImportStatusResponse `json:"body"`
}

// ImportStatusResponse represents the current status of an import operation
type ImportStatusResponse struct {
	ImportID  string                 `json:"import_id" doc:"Unique ID of the import operation"`
	Status    string                 `json:"status" doc:"Current status: pending, running, completed, failed"`
	StartTime *string                `json:"start_time,omitempty" doc:"When the import started"`
	EndTime   *string                `json:"end_time,omitempty" doc:"When the import completed/failed"`
	Duration  *string                `json:"duration,omitempty" doc:"Total duration of the import"`
	Progress  ImportProgressResponse `json:"progress" doc:"Detailed progress information"`
	Error     string                 `json:"error,omitempty" doc:"Error message if import failed"`
	CreatedAt string                 `json:"created_at" doc:"When the import was created"`
	UpdatedAt string                 `json:"updated_at" doc:"When the status was last updated"`
}

// ImportProgressResponse represents detailed progress information
type ImportProgressResponse struct {
	TotalSteps      int                               `json:"total_steps" doc:"Total number of steps in the import"`
	CompletedSteps  int                               `json:"completed_steps" doc:"Number of completed steps"`
	CurrentStep     string                            `json:"current_step" doc:"Description of the current step"`
	PercentComplete float64                           `json:"percent_complete" doc:"Completion percentage (0-100)"`
	DataTypes       map[string]DataTypeStatusResponse `json:"data_types" doc:"Status of each data type being imported"`
}

// DataTypeStatusResponse represents the status of importing a specific data type
type DataTypeStatusResponse struct {
	Name            string  `json:"name" doc:"Name of the data type"`
	Status          string  `json:"status" doc:"Status: pending, processing, completed, failed"`
	Count           int     `json:"count" doc:"Total number of items to import"`
	Processed       int     `json:"processed" doc:"Number of items processed"`
	PercentComplete float64 `json:"percent_complete" doc:"Completion percentage for this data type (0-100)"`
	Error           string  `json:"error,omitempty" doc:"Error message if this data type failed"`
}

// SDEStatsOutput represents the output for SDE statistics endpoint
type SDEStatsOutput struct {
	Body SDEStatsResponse `json:"body"`
}

// SDEStatsResponse represents statistics about SDE data in Redis
type SDEStatsResponse struct {
	TotalKeys       int                      `json:"total_keys" doc:"Total number of SDE keys in Redis"`
	DataTypes       map[string]DataTypeStats `json:"data_types" doc:"Statistics for each data type"`
	LastImport      *string                  `json:"last_import,omitempty" doc:"Timestamp of last successful import"`
	RedisMemoryUsed string                   `json:"redis_memory_used" doc:"Memory used by Redis"`
}

// DataTypeStats represents statistics for a specific data type
type DataTypeStats struct {
	Count      int    `json:"count" doc:"Number of items stored"`
	MemoryUsed string `json:"memory_used,omitempty" doc:"Estimated memory used"`
	KeyPattern string `json:"key_pattern" doc:"Redis key pattern used"`
}

// ClearSDEOutput represents the output for clearing SDE data
type ClearSDEOutput struct {
	Body ClearSDEResponse `json:"body"`
}

// ClearSDEResponse represents the response after clearing SDE data
type ClearSDEResponse struct {
	Success     bool   `json:"success" doc:"Whether the clear operation was successful"`
	Message     string `json:"message" doc:"Human-readable status message"`
	KeysDeleted int    `json:"keys_deleted" doc:"Number of keys deleted from Redis"`
}

// ConvertFromModel converts models.ImportStatus to ImportStatusResponse
func ConvertFromModel(status *models.ImportStatus) *ImportStatusResponse {
	response := &ImportStatusResponse{
		ImportID:  status.ID,
		Status:    status.Status,
		Error:     status.Error,
		CreatedAt: status.CreatedAt.Format(time.RFC3339),
		UpdatedAt: status.UpdatedAt.Format(time.RFC3339),
	}

	if status.StartTime != nil {
		startTime := status.StartTime.Format(time.RFC3339)
		response.StartTime = &startTime
	}

	if status.EndTime != nil {
		endTime := status.EndTime.Format(time.RFC3339)
		response.EndTime = &endTime

		if status.StartTime != nil {
			duration := status.EndTime.Sub(*status.StartTime)
			durationStr := duration.String()
			response.Duration = &durationStr
		}
	}

	// Convert progress
	response.Progress = ImportProgressResponse{
		TotalSteps:     status.Progress.TotalSteps,
		CompletedSteps: status.Progress.CompletedSteps,
		CurrentStep:    status.Progress.CurrentStep,
		DataTypes:      make(map[string]DataTypeStatusResponse),
	}

	// Calculate overall progress percentage
	if status.Progress.TotalSteps > 0 {
		response.Progress.PercentComplete = float64(status.Progress.CompletedSteps) / float64(status.Progress.TotalSteps) * 100
	}

	// Convert data type statuses
	for name, dataType := range status.Progress.DataTypes {
		dtResponse := DataTypeStatusResponse{
			Name:      dataType.Name,
			Status:    dataType.Status,
			Count:     dataType.Count,
			Processed: dataType.Processed,
			Error:     dataType.Error,
		}

		// Calculate percentage for this data type
		if dataType.Count > 0 {
			dtResponse.PercentComplete = float64(dataType.Processed) / float64(dataType.Count) * 100
		}

		response.Progress.DataTypes[name] = dtResponse
	}

	return response
}
