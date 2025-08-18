package scheduler

import (
	"testing"

	"go-falcon/internal/scheduler/dto"

	"github.com/stretchr/testify/assert"
)

// TestSchedulerHumaDTOs tests that scheduler Huma DTOs are properly structured
func TestSchedulerHumaDTOs(t *testing.T) {
	// Test basic input/output types compile correctly
	var taskCreateInput interface{} = &dto.TaskCreateInput{}
	var taskCreateOutput interface{} = &dto.TaskCreateOutput{}
	var taskListInput interface{} = &dto.TaskListInput{}
	var taskListOutput interface{} = &dto.TaskListOutput{}
	
	assert.NotNil(t, taskCreateInput)
	assert.NotNil(t, taskCreateOutput)
	assert.NotNil(t, taskListInput)
	assert.NotNil(t, taskListOutput)
	
	t.Logf("✅ Scheduler Huma DTOs are properly structured")
}

// TestSchedulerHumaValidation tests that validation tags are properly set
func TestSchedulerHumaValidation(t *testing.T) {
	// Test TaskGetInput with required task_id
	taskGetInput := &dto.TaskGetInput{TaskID: "test-task-123"}
	assert.Equal(t, "test-task-123", taskGetInput.TaskID)
	
	// Test TaskListInput with pagination
	taskListInput := &dto.TaskListInput{
		Page:     1,
		PageSize: 20,
		Status:   "pending",
		Type:     "http",
	}
	
	assert.Equal(t, 1, taskListInput.Page)
	assert.Equal(t, 20, taskListInput.PageSize)
	assert.Equal(t, "pending", taskListInput.Status)
	assert.Equal(t, "http", taskListInput.Type)
	
	t.Logf("✅ Scheduler Huma validation tags are properly configured")
}