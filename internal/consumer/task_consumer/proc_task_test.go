package task_consumer

// White-box test: same package as proc_task.go to access unexported handleTask,
// processSendEmail, processGenerateReport, processResizeImage, processCleanupTemp.

import (
	"task_handler/internal/domain/task"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makePayload(taskID int, taskType string) *task.TaskPayload {
	return &task.TaskPayload{
		TaskID:   taskID,
		UserID:   1,
		TaskType: taskType,
	}
}

// ---- handleTask dispatch router ----

func TestHandleTask_SendEmail(t *testing.T) {
	err := handleTask(makePayload(1, "send_email"), 1)
	assert.NoError(t, err)
}

func TestHandleTask_GenerateReport(t *testing.T) {
	err := handleTask(makePayload(2, "generate_report"), 1)
	assert.NoError(t, err)
}

func TestHandleTask_ResizeImage(t *testing.T) {
	err := handleTask(makePayload(3, "resize_image"), 1)
	assert.NoError(t, err)
}

func TestHandleTask_CleanupTemp(t *testing.T) {
	err := handleTask(makePayload(4, "cleanup_temp"), 1)
	assert.NoError(t, err)
}

func TestHandleTask_UnknownType(t *testing.T) {
	err := handleTask(makePayload(5, "unknown_task"), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown task type")
}

func TestHandleTask_EmptyType(t *testing.T) {
	err := handleTask(makePayload(6, ""), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown task type")
}

// ---- Individual processor functions ----

func TestProcessSendEmail(t *testing.T) {
	err := processSendEmail(makePayload(10, "send_email"), 2)
	assert.NoError(t, err)
}

func TestProcessGenerateReport(t *testing.T) {
	err := processGenerateReport(makePayload(11, "generate_report"), 2)
	assert.NoError(t, err)
}

func TestProcessResizeImage(t *testing.T) {
	err := processResizeImage(makePayload(12, "resize_image"), 2)
	assert.NoError(t, err)
}

func TestProcessCleanupTemp(t *testing.T) {
	err := processCleanupTemp(makePayload(13, "cleanup_temp"), 2)
	assert.NoError(t, err)
}

// ---- Multiple workers test ----

func TestHandleTask_MultipleWorkerIDs(t *testing.T) {
	for _, workerID := range []int{1, 2, 3} {
		err := handleTask(makePayload(workerID, "send_email"), workerID)
		assert.NoError(t, err, "worker %d should handle send_email", workerID)
	}
}

// ---- All task types in a table test ----

func TestHandleTask_AllValidTypes(t *testing.T) {
	validTypes := []string{
		"send_email",
		"generate_report",
		"resize_image",
		"cleanup_temp",
	}

	for i, taskType := range validTypes {
		t.Run(taskType, func(t *testing.T) {
			err := handleTask(makePayload(100+i, taskType), 1)
			assert.NoError(t, err)
		})
	}
}
