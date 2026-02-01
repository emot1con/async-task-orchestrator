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
func NewTaskCompletedEvent(taskID, userID int, taskType, resultFile string, processingTimeMs int64, workerID string, correlationID string) *TaskSucceddEvent {
	return &TaskSucceddEvent{
		BaseEvent: NewBaseEvent(EventTypeTaskSuccedd, SourceWorker),
		Data: TaskSucceddData{
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

// NewUserRegisteredEvent creates a user.registered event
func NewUserRegisteredEvent(userID int, username string, correlationID string) *UserRegisteredEvent {
	return &UserRegisteredEvent{
		BaseEvent: NewBaseEvent(EventTypeUserRegistered, SourceAPI),
		Data: UserRegisteredData{
			UserID:       userID,
			Username:     username,
			RegisteredAt: time.Now().UTC(),
		},
		Metadata: Metadata{
			CorrelationID: correlationID,
		},
	}
}

// NewUserLoggedInEvent creates a user.logged_in event
func NewUserLoggedInEvent(userID int, username string, ipAddress string, userAgent string, correlationID string) *UserLoggedInEvent {
	return &UserLoggedInEvent{
		BaseEvent: NewBaseEvent(EventTypeUserLoggedIn, SourceAPI),
		Data: UserLoggedInData{
			UserID:     userID,
			Username:   username,
			IPAddress:  ipAddress,
			UserAgent:  userAgent,
			LoggedInAt: time.Now().UTC(),
		},
		Metadata: Metadata{
			CorrelationID: correlationID,
		},
	}
}
