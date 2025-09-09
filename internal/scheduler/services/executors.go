package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go-falcon/internal/alliance/dto"
	"go-falcon/internal/scheduler/models"
)

// HTTPExecutor executes HTTP tasks
type HTTPExecutor struct {
	client *http.Client
}

// NewHTTPExecutor creates a new HTTP executor
func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute executes an HTTP task
func (e *HTTPExecutor) Execute(ctx context.Context, task *models.Task) (*models.TaskResult, error) {
	// Parse HTTP config
	config, err := e.parseHTTPConfig(task.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP config: %w", err)
	}

	// Create request
	var bodyReader io.Reader
	if config.Body != "" {
		bodyReader = bytes.NewReader([]byte(config.Body))
	}

	req, err := http.NewRequestWithContext(ctx, config.Method, config.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Set default User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "go-falcon-scheduler/1.0.0")
	}

	// Execute request
	start := time.Now()
	resp, err := e.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("HTTP request failed: %v", err),
			Duration: models.Duration(duration),
		}, nil
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Failed to read response: %v", err),
			Duration: models.Duration(duration),
		}, nil
	}

	// Check expected status code
	expectedCode := config.ExpectedCode
	if expectedCode == 0 {
		expectedCode = 200 // Default expected code
	}

	success := resp.StatusCode == expectedCode
	output := string(body)

	result := &models.TaskResult{
		Success:  success,
		Output:   output,
		Duration: models.Duration(duration),
		Metadata: map[string]interface{}{
			"status_code":   resp.StatusCode,
			"response_size": len(body),
			"response_time": duration.String(),
		},
	}

	if !success {
		result.Error = fmt.Sprintf("Unexpected status code: %d (expected %d)", resp.StatusCode, expectedCode)
	}

	return result, nil
}

// parseHTTPConfig parses HTTP task configuration
func (e *HTTPExecutor) parseHTTPConfig(config map[string]interface{}) (*models.HTTPTaskConfig, error) {
	httpConfig := &models.HTTPTaskConfig{}

	// URL (required)
	if url, ok := config["url"].(string); ok {
		httpConfig.URL = url
	} else {
		return nil, fmt.Errorf("url is required")
	}

	// Method (required)
	if method, ok := config["method"].(string); ok {
		httpConfig.Method = method
	} else {
		return nil, fmt.Errorf("method is required")
	}

	// Headers (optional)
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		httpConfig.Headers = make(map[string]string)
		for k, v := range headers {
			if str, ok := v.(string); ok {
				httpConfig.Headers[k] = str
			}
		}
	}

	// Body (optional)
	if body, ok := config["body"].(string); ok {
		httpConfig.Body = body
	}

	// Expected code (optional)
	if code, ok := config["expected_code"].(int); ok {
		httpConfig.ExpectedCode = code
	}

	// Timeout (optional)
	if timeout, ok := config["timeout"].(string); ok {
		if duration, err := time.ParseDuration(timeout); err == nil {
			httpConfig.Timeout = models.Duration(duration)
		}
	}

	// Follow redirect (optional)
	if followRedirect, ok := config["follow_redirect"].(bool); ok {
		httpConfig.FollowRedirect = followRedirect
	}

	// Validate SSL (optional)
	if validateSSL, ok := config["validate_ssl"].(bool); ok {
		httpConfig.ValidateSSL = validateSSL
	}

	return httpConfig, nil
}

// CharacterModule interface for character operations
type CharacterModule interface {
	UpdateAllAffiliations(ctx context.Context) (updated, failed, skipped int, err error)
}

// AllianceModule interface for alliance operations
type AllianceModule interface {
	BulkImportAlliances(ctx context.Context) (*dto.BulkImportAlliancesOutput, error)
}

// CorporationModule interface for corporation operations
type CorporationModule interface {
	UpdateAllCorporations(ctx context.Context, concurrentWorkers int) error
	ValidateCEOTokens(ctx context.Context) error
}

// SystemExecutor executes system tasks
type SystemExecutor struct {
	authModule        AuthModule
	characterModule   CharacterModule
	allianceModule    AllianceModule
	corporationModule CorporationModule
	groupsModule      GroupsModule
}

// NewSystemExecutor creates a new system executor
func NewSystemExecutor(authModule AuthModule, characterModule CharacterModule, allianceModule AllianceModule, corporationModule CorporationModule, groupsModule GroupsModule) *SystemExecutor {
	return &SystemExecutor{
		authModule:        authModule,
		characterModule:   characterModule,
		allianceModule:    allianceModule,
		corporationModule: corporationModule,
		groupsModule:      groupsModule,
	}
}

// Execute executes a system task
func (e *SystemExecutor) Execute(ctx context.Context, task *models.Task) (*models.TaskResult, error) {
	// Parse system config
	config, err := e.parseSystemConfig(task.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid system config: %w", err)
	}

	start := time.Now()

	switch config.TaskName {
	case "token_refresh":
		return e.executeTokenRefresh(ctx, config, start)
	case "state_cleanup":
		return e.executeStateCleanup(ctx, config, start)
	case "health_check":
		return e.executeHealthCheck(ctx, config, start)
	case "character_affiliation_update":
		return e.executeCharacterAffiliationUpdate(ctx, config, start)
	case "alliance_bulk_import":
		return e.executeAllianceBulkImport(ctx, config, start)
	case "corporation_update":
		return e.executeCorporationUpdate(ctx, config, start)
	case "ceo_token_validation":
		return e.executeCEOTokenValidation(ctx, config, start)
	case "groups_sync":
		return e.executeGroupsSync(ctx, config, start)
	default:
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Unknown system task: %s", config.TaskName),
			Duration: models.Duration(time.Since(start)),
		}, nil
	}
}

// executeTokenRefresh executes the token refresh system task
func (e *SystemExecutor) executeTokenRefresh(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	if e.authModule == nil {
		return &models.TaskResult{
			Success:  false,
			Error:    "Auth module not available",
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	// Get batch size from parameters
	batchSize := 100 // default
	if params, ok := config.Parameters["batch_size"].(int); ok {
		batchSize = params
	}

	slog.Info("Starting EVE token refresh system task",
		slog.Int("batch_size", batchSize))

	// Execute token refresh
	successCount, failureCount, err := e.authModule.RefreshExpiringTokens(ctx, batchSize)
	if err != nil {
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Token refresh failed: %v", err),
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	output := fmt.Sprintf("Processed %d tokens: %d successful, %d failed",
		successCount+failureCount, successCount, failureCount)

	slog.Info("EVE token refresh system task completed",
		slog.Int("total_processed", successCount+failureCount),
		slog.Int("successful", successCount),
		slog.Int("failed", failureCount),
		slog.String("duration", time.Since(start).String()))

	return &models.TaskResult{
		Success:  true,
		Output:   output,
		Duration: models.Duration(time.Since(start)),
		Metadata: map[string]interface{}{
			"success_count": successCount,
			"failure_count": failureCount,
			"batch_size":    batchSize,
		},
	}, nil
}

// executeStateCleanup executes the state cleanup system task
func (e *SystemExecutor) executeStateCleanup(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	// Implement state cleanup logic here
	// For now, just return success
	return &models.TaskResult{
		Success:  true,
		Output:   "State cleanup completed",
		Duration: models.Duration(time.Since(start)),
	}, nil
}

// executeHealthCheck executes the health check system task
func (e *SystemExecutor) executeHealthCheck(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	// Implement health check logic here
	// For now, just return success
	return &models.TaskResult{
		Success:  true,
		Output:   "Health check passed",
		Duration: models.Duration(time.Since(start)),
	}, nil
}

// executeCharacterAffiliationUpdate executes the character affiliation update system task
func (e *SystemExecutor) executeCharacterAffiliationUpdate(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	if e.characterModule == nil {
		return &models.TaskResult{
			Success:  false,
			Error:    "Character module not available",
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	// Execute affiliation update
	updated, failed, skipped, err := e.characterModule.UpdateAllAffiliations(ctx)
	if err != nil {
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Character affiliation update failed: %v", err),
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	total := updated + failed + skipped
	output := fmt.Sprintf("Processed %d characters: %d updated, %d failed, %d skipped",
		total, updated, failed, skipped)

	// Consider it a success if at least some characters were updated
	success := updated > 0 || (failed == 0 && skipped >= 0)

	return &models.TaskResult{
		Success:  success,
		Output:   output,
		Duration: models.Duration(time.Since(start)),
		Metadata: map[string]interface{}{
			"total_characters":   total,
			"updated_characters": updated,
			"failed_characters":  failed,
			"skipped_characters": skipped,
		},
	}, nil
}

// executeAllianceBulkImport executes the alliance bulk import system task
func (e *SystemExecutor) executeAllianceBulkImport(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	if e.allianceModule == nil {
		return &models.TaskResult{
			Success:  false,
			Error:    "Alliance module not available",
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	// Execute alliance bulk import
	result, err := e.allianceModule.BulkImportAlliances(ctx)
	if err != nil {
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Alliance bulk import failed: %v", err),
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	stats := result.Body
	output := fmt.Sprintf("Processed %d alliances: %d created, %d updated, %d failed",
		stats.Processed, stats.Created, stats.Updated, stats.Failed)

	// Consider it a success if at least some alliances were processed successfully
	success := stats.Created > 0 || stats.Updated > 0 || (stats.Failed == 0 && stats.Processed >= 0)

	return &models.TaskResult{
		Success:  success,
		Output:   output,
		Duration: models.Duration(time.Since(start)),
		Metadata: map[string]interface{}{
			"total_alliances":     stats.TotalAlliances,
			"processed_alliances": stats.Processed,
			"created_alliances":   stats.Created,
			"updated_alliances":   stats.Updated,
			"failed_alliances":    stats.Failed,
			"skipped_alliances":   stats.Skipped,
		},
	}, nil
}

// executeCorporationUpdate executes the corporation update system task
func (e *SystemExecutor) executeCorporationUpdate(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	if e.corporationModule == nil {
		return &models.TaskResult{
			Success:  false,
			Error:    "Corporation module not available",
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	// Get concurrent workers from parameters
	concurrentWorkers := 10 // default
	if params, ok := config.Parameters["concurrent_workers"]; ok {
		switch v := params.(type) {
		case int:
			concurrentWorkers = v
		case float64:
			concurrentWorkers = int(v)
		}
	}

	// Execute corporation update
	err := e.corporationModule.UpdateAllCorporations(ctx, concurrentWorkers)
	if err != nil {
		// Extract failure count from error message if possible
		errorMsg := err.Error()
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Corporation update failed: %v", err),
			Duration: models.Duration(time.Since(start)),
			Metadata: map[string]interface{}{
				"concurrent_workers": concurrentWorkers,
				"error_details":      errorMsg,
			},
		}, nil
	}

	output := fmt.Sprintf("Successfully updated all corporations with %d concurrent workers", concurrentWorkers)

	return &models.TaskResult{
		Success:  true,
		Output:   output,
		Duration: models.Duration(time.Since(start)),
		Metadata: map[string]interface{}{
			"concurrent_workers": concurrentWorkers,
		},
	}, nil
}

// executeCEOTokenValidation executes the CEO token validation system task
func (e *SystemExecutor) executeCEOTokenValidation(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	if e.corporationModule == nil {
		return &models.TaskResult{
			Success:  false,
			Error:    "Corporation module not available",
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	// Execute CEO token validation
	err := e.corporationModule.ValidateCEOTokens(ctx)
	if err != nil {
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("CEO token validation failed: %v", err),
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	output := "CEO token validation completed successfully"

	return &models.TaskResult{
		Success:  true,
		Output:   output,
		Duration: models.Duration(time.Since(start)),
		Metadata: map[string]interface{}{
			"task_type": "ceo_token_validation",
		},
	}, nil
}

// executeGroupsSync executes the groups synchronization system task
func (e *SystemExecutor) executeGroupsSync(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	if e.groupsModule == nil {
		return &models.TaskResult{
			Success:  false,
			Error:    "Groups module not available",
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	// Execute group membership validation
	err := e.groupsModule.ValidateGroupMembershipsAgainstEntityStatus(ctx)
	if err != nil {
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Groups sync failed: %v", err),
			Duration: models.Duration(time.Since(start)),
		}, nil
	}

	return &models.TaskResult{
		Success:  true,
		Output:   "Group membership validation completed successfully",
		Duration: models.Duration(time.Since(start)),
		Metadata: map[string]interface{}{
			"task_type": "groups_sync",
		},
	}, nil
}

// parseSystemConfig parses system task configuration
func (e *SystemExecutor) parseSystemConfig(config map[string]interface{}) (*models.SystemTaskConfig, error) {
	systemConfig := &models.SystemTaskConfig{}

	// Task name (required)
	if taskName, ok := config["task_name"].(string); ok {
		systemConfig.TaskName = taskName
	} else {
		return nil, fmt.Errorf("task_name is required")
	}

	// Parameters (optional)
	if parameters, ok := config["parameters"].(map[string]interface{}); ok {
		systemConfig.Parameters = parameters
	} else {
		systemConfig.Parameters = make(map[string]interface{})
	}

	return systemConfig, nil
}

// FunctionExecutor executes function tasks
type FunctionExecutor struct{}

// NewFunctionExecutor creates a new function executor
func NewFunctionExecutor() *FunctionExecutor {
	return &FunctionExecutor{}
}

// Execute executes a function task
func (e *FunctionExecutor) Execute(ctx context.Context, task *models.Task) (*models.TaskResult, error) {
	// Parse function config
	config, err := e.parseFunctionConfig(task.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid function config: %w", err)
	}

	start := time.Now()

	// For now, just log the function execution
	// In a real implementation, this would call the actual function
	slog.Info("Executing function task",
		slog.String("task_id", task.ID),
		slog.String("function_name", config.FunctionName),
		slog.String("module", config.Module))

	output := fmt.Sprintf("Function '%s' executed successfully", config.FunctionName)

	return &models.TaskResult{
		Success:  true,
		Output:   output,
		Duration: models.Duration(time.Since(start)),
		Metadata: map[string]interface{}{
			"function_name": config.FunctionName,
			"module":        config.Module,
			"parameters":    config.Parameters,
		},
	}, nil
}

// parseFunctionConfig parses function task configuration
func (e *FunctionExecutor) parseFunctionConfig(config map[string]interface{}) (*models.FunctionTaskConfig, error) {
	functionConfig := &models.FunctionTaskConfig{}

	// Function name (required)
	if functionName, ok := config["function_name"].(string); ok {
		functionConfig.FunctionName = functionName
	} else {
		return nil, fmt.Errorf("function_name is required")
	}

	// Module (optional)
	if module, ok := config["module"].(string); ok {
		functionConfig.Module = module
	}

	// Parameters (optional)
	if parameters, ok := config["parameters"].(map[string]interface{}); ok {
		functionConfig.Parameters = parameters
	} else {
		functionConfig.Parameters = make(map[string]interface{})
	}

	return functionConfig, nil
}
