package events_test

import (
	"task_handler/internal/events"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskCreatedEvent(t *testing.T) {
	event := events.NewTaskCreatedEvent(1, 10, "generate_report", "corr-123")

	require.NotNil(t, event)
	assert.Equal(t, 1, event.Data.TaskID)
	assert.Equal(t, 10, event.Data.UserID)
	assert.Equal(t, "generate_report", event.Data.TaskType)
	assert.Equal(t, events.StatusPending, event.Data.Status)
	assert.NotEmpty(t, event.EventID)
	assert.Equal(t, events.EventTypeTaskCreated, event.EventType)
	assert.Equal(t, events.EventVersionV1, event.EventVersion)
	assert.Equal(t, events.SourceAPI, event.Source)
	assert.Equal(t, "corr-123", event.Metadata.CorrelationID)
	assert.False(t, event.Timestamp.IsZero())
}

func TestNewTaskCreatedEvent_EmptyCorrelationID(t *testing.T) {
	event := events.NewTaskCreatedEvent(5, 20, "send_email", "")

	require.NotNil(t, event)
	assert.Empty(t, event.Metadata.CorrelationID)
	assert.Equal(t, 5, event.Data.TaskID)
}

func TestNewTaskCompletedEvent(t *testing.T) {
	event := events.NewTaskCompletedEvent(2, 11, "resize_image", "result.png", 1500, "worker-1", "corr-456")

	require.NotNil(t, event)
	assert.Equal(t, 2, event.Data.TaskID)
	assert.Equal(t, 11, event.Data.UserID)
	assert.Equal(t, "resize_image", event.Data.TaskType)
	assert.Equal(t, "SUCCESS", event.Data.Status)
	assert.Equal(t, "result.png", event.Data.ResultFile)
	assert.Equal(t, int64(1500), event.Data.ProcessingTimeMs)
	assert.Equal(t, events.EventTypeTaskSuccedd, event.EventType)
	assert.Equal(t, events.SourceWorker, event.Source)
	assert.Equal(t, "worker-1", event.Metadata.WorkerID)
	assert.Equal(t, "corr-456", event.Metadata.CorrelationID)
	assert.False(t, event.Data.CompletedAt.IsZero())
}

func TestNewTaskFailedEvent(t *testing.T) {
	event := events.NewTaskFailedEvent(3, 12, "cleanup_temp", "disk full", "DISK_ERROR", 2, "worker-2", "corr-789")

	require.NotNil(t, event)
	assert.Equal(t, 3, event.Data.TaskID)
	assert.Equal(t, 12, event.Data.UserID)
	assert.Equal(t, "cleanup_temp", event.Data.TaskType)
	assert.Equal(t, "FAILED", event.Data.Status)
	assert.Equal(t, "disk full", event.Data.ErrorMessage)
	assert.Equal(t, "DISK_ERROR", event.Data.ErrorCode)
	assert.Equal(t, 2, event.Data.RetryCount)
	assert.Equal(t, events.EventTypeTaskFailed, event.EventType)
	assert.Equal(t, events.SourceWorker, event.Source)
	assert.Equal(t, "worker-2", event.Metadata.WorkerID)
	assert.False(t, event.Data.FailedAt.IsZero())
}

func TestNewBaseEvent_UniqueEventIDs(t *testing.T) {
	event1 := events.NewBaseEvent("task.created", "api")
	event2 := events.NewBaseEvent("task.created", "api")

	// Each event should have unique ID
	assert.NotEqual(t, event1.EventID, event2.EventID)
}

func TestNewBaseEvent_Fields(t *testing.T) {
	event := events.NewBaseEvent(events.EventTypeTaskCreated, events.SourceAPI)

	assert.NotEmpty(t, event.EventID)
	assert.Equal(t, events.EventTypeTaskCreated, event.EventType)
	assert.Equal(t, events.EventVersionV1, event.EventVersion)
	assert.Equal(t, events.SourceAPI, event.Source)
	assert.False(t, event.Timestamp.IsZero())
}

func TestEventConstants(t *testing.T) {
	assert.Equal(t, "task.created", events.EventTypeTaskCreated)
	assert.Equal(t, "task.succedd", events.EventTypeTaskSuccedd)
	assert.Equal(t, "task.failed", events.EventTypeTaskFailed)
	assert.Equal(t, "PENDING", events.StatusPending)
	assert.Equal(t, "PROCESSING", events.StatusProcessing)
	assert.Equal(t, "SUCCESS", events.StatusSuccess)
	assert.Equal(t, "FAILED", events.StatusFailed)
	assert.Equal(t, "task_queue", events.TaskQueueName)
	assert.Equal(t, "notification_queue", events.NotificationQueueName)
	assert.Equal(t, 3, events.MaxRetries)
}
