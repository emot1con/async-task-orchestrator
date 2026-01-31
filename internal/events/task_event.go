package events

import "time"

// TaskCreatedEvent - emitted when task is created
type TaskCreatedEvent struct {
	BaseEvent
	Data     TaskCreatedData `json:"data"`
	Metadata Metadata        `json:"metadata"`
}

type TaskCreatedData struct {
	TaskID    int       `json:"task_id"`
	UserID    int       `json:"user_id"`
	TaskType  string    `json:"task_type"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskCompletedEvent - emitted when task succeeds
type TaskCompletedEvent struct {
	BaseEvent
	Data     TaskCompletedData `json:"data"`
	Metadata Metadata          `json:"metadata"`
}

type TaskCompletedData struct {
	TaskID           int       `json:"task_id"`
	UserID           int       `json:"user_id"`
	TaskType         string    `json:"task_type"`
	Status           string    `json:"status"`
	ResultFile       string    `json:"result_file,omitempty"`
	ProcessingTimeMs int64     `json:"processing_time_ms"`
	CompletedAt      time.Time `json:"completed_at"`
}

// TaskFailedEvent - emitted when task fails
type TaskFailedEvent struct {
	BaseEvent
	Data     TaskFailedData `json:"data"`
	Metadata Metadata       `json:"metadata"`
}

type TaskFailedData struct {
	TaskID       int       `json:"task_id"`
	UserID       int       `json:"user_id"`
	TaskType     string    `json:"task_type"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message"`
	ErrorCode    string    `json:"error_code,omitempty"`
	RetryCount   int       `json:"retry_count"`
	FailedAt     time.Time `json:"failed_at"`
}
