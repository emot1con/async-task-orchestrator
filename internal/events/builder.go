package events

import "time"

func NewTaskCreatedEvent(taskID, userID int, taskType string, correlactionID string) *TaskCreatedEvent {
	return &TaskCreatedEvent{
		BaseEvent: NewBaseEvent(EventTypeTaskCreated, SourceAPI),
		Data: TaskCreatedData{
			TaskID:    taskID,
			UserID:    userID,
			TaskType:  taskType,
			Status:    StatusPending,
			CreatedAt: time.Now().UTC(),
		},
		Metadata: Metadata{
			CorrelationID: correlactionID,
		},
	}
}

// NewTaskCompletedEvent creates a task.completed event
func NewTaskCompletedEvent(taskID, userID int, taskType, resultFile string, processingTimeMs int64, workerID string, correlationID string) *TaskCompletedEvent {
	return &TaskCompletedEvent{
		BaseEvent: NewBaseEvent(EventTypeTaskCompleted, SourceWorker),
		Data: TaskCompletedData{
			TaskID:           taskID,
			UserID:           userID,
			TaskType:         taskType,
			Status:           "SUCCESS",
			ResultFile:       resultFile,
			ProcessingTimeMs: processingTimeMs,
			CompletedAt:      time.Now().UTC(),
		},
		Metadata: Metadata{
			CorrelationID: correlationID,
			WorkerID:      workerID,
		},
	}
}

// NewTaskFailedEvent creates a task.failed event
func NewTaskFailedEvent(taskID, userID int, taskType, errorMsg, errorCode string, retryCount int, workerID, correlationID string) *TaskFailedEvent {
	return &TaskFailedEvent{
		BaseEvent: NewBaseEvent(EventTypeTaskFailed, SourceWorker),
		Data: TaskFailedData{
			TaskID:       taskID,
			UserID:       userID,
			TaskType:     taskType,
			Status:       "FAILED",
			ErrorMessage: errorMsg,
			ErrorCode:    errorCode,
			RetryCount:   retryCount,
			FailedAt:     time.Now().UTC(),
		},
		Metadata: Metadata{
			CorrelationID: correlationID,
			WorkerID:      workerID,
		},
	}
}
