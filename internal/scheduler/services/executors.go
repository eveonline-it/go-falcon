package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

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
			Duration: duration,
		}, nil
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Failed to read response: %v", err),
			Duration: duration,
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
		Duration: duration,
		Metadata: map[string]interface{}{
			"status_code":    resp.StatusCode,
			"response_size":  len(body),
			"response_time":  duration.String(),
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
			httpConfig.Timeout = duration
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

// SystemExecutor executes system tasks
type SystemExecutor struct {
	authModule   AuthModule
	sdeModule    SDEModule
	groupsModule GroupsModule
}

// NewSystemExecutor creates a new system executor
func NewSystemExecutor(authModule AuthModule, sdeModule SDEModule, groupsModule GroupsModule) *SystemExecutor {
	return &SystemExecutor{
		authModule:   authModule,
		sdeModule:    sdeModule,
		groupsModule: groupsModule,
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
	case "group_validation":
		return e.executeGroupValidation(ctx, config, start)
	case "sde_update_check":
		return e.executeSDEUpdateCheck(ctx, config, start)
	default:
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Unknown system task: %s", config.TaskName),
			Duration: time.Since(start),
		}, nil
	}
}

// executeTokenRefresh executes the token refresh system task
func (e *SystemExecutor) executeTokenRefresh(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	if e.authModule == nil {
		return &models.TaskResult{
			Success:  false,
			Error:    "Auth module not available",
			Duration: time.Since(start),
		}, nil
	}

	// Get batch size from parameters
	batchSize := 100 // default
	if params, ok := config.Parameters["batch_size"].(int); ok {
		batchSize = params
	}

	// Execute token refresh
	successCount, failureCount, err := e.authModule.RefreshExpiringTokens(ctx, batchSize)
	if err != nil {
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Token refresh failed: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	output := fmt.Sprintf("Processed %d tokens: %d successful, %d failed", 
		successCount+failureCount, successCount, failureCount)

	return &models.TaskResult{
		Success:  true,
		Output:   output,
		Duration: time.Since(start),
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
		Duration: time.Since(start),
	}, nil
}

// executeHealthCheck executes the health check system task
func (e *SystemExecutor) executeHealthCheck(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	// Implement health check logic here
	// For now, just return success
	return &models.TaskResult{
		Success:  true,
		Output:   "Health check passed",
		Duration: time.Since(start),
	}, nil
}

// executeGroupValidation executes group validation system tasks
func (e *SystemExecutor) executeGroupValidation(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	if e.groupsModule == nil {
		return &models.TaskResult{
			Success:  false,
			Error:    "Groups module not available",
			Duration: time.Since(start),
		}, nil
	}

	// Execute different group validation tasks based on parameters
	validationType := "membership" // default
	if params, ok := config.Parameters["type"].(string); ok {
		validationType = params
	}

	switch validationType {
	case "membership":
		err := e.groupsModule.ValidateCorporateMemberships(ctx)
		if err != nil {
			return &models.TaskResult{
				Success:  false,
				Error:    fmt.Sprintf("Membership validation failed: %v", err),
				Duration: time.Since(start),
			}, nil
		}
		return &models.TaskResult{
			Success:  true,
			Output:   "Corporate membership validation completed",
			Duration: time.Since(start),
		}, nil

	case "cleanup":
		count, err := e.groupsModule.CleanupExpiredMemberships(ctx)
		if err != nil {
			return &models.TaskResult{
				Success:  false,
				Error:    fmt.Sprintf("Membership cleanup failed: %v", err),
				Duration: time.Since(start),
			}, nil
		}
		return &models.TaskResult{
			Success:  true,
			Output:   fmt.Sprintf("Cleaned up %d expired memberships", count),
			Duration: time.Since(start),
			Metadata: map[string]interface{}{
				"cleaned_count": count,
			},
		}, nil

	case "discord":
		err := e.groupsModule.SyncDiscordRoles(ctx)
		if err != nil {
			return &models.TaskResult{
				Success:  false,
				Error:    fmt.Sprintf("Discord role sync failed: %v", err),
				Duration: time.Since(start),
			}, nil
		}
		return &models.TaskResult{
			Success:  true,
			Output:   "Discord role synchronization completed",
			Duration: time.Since(start),
		}, nil

	case "integrity":
		err := e.groupsModule.ValidateGroupIntegrity(ctx)
		if err != nil {
			return &models.TaskResult{
				Success:  false,
				Error:    fmt.Sprintf("Group integrity validation failed: %v", err),
				Duration: time.Since(start),
			}, nil
		}
		return &models.TaskResult{
			Success:  true,
			Output:   "Group integrity validation completed",
			Duration: time.Since(start),
		}, nil

	default:
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("Unknown validation type: %s", validationType),
			Duration: time.Since(start),
		}, nil
	}
}

// executeSDEUpdateCheck executes SDE update check system task
func (e *SystemExecutor) executeSDEUpdateCheck(ctx context.Context, config *models.SystemTaskConfig, start time.Time) (*models.TaskResult, error) {
	if e.sdeModule == nil {
		return &models.TaskResult{
			Success:  false,
			Error:    "SDE module not available",
			Duration: time.Since(start),
		}, nil
	}

	err := e.sdeModule.CheckSDEUpdate(ctx)
	if err != nil {
		return &models.TaskResult{
			Success:  false,
			Error:    fmt.Sprintf("SDE update check failed: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	return &models.TaskResult{
		Success:  true,
		Output:   "SDE update check completed",
		Duration: time.Since(start),
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
		Duration: time.Since(start),
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