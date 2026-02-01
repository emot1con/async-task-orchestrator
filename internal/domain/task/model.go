package task

import "time"

type Task struct {
	TaskID       int
	UserID       int
	TaskType     string
	Status       string
	ResultFile   *string
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TaskPayload struct {
	TaskID   int    `json:"task_id"`
	UserID   int    `json:"user_id"`
	TaskType string `json:"task_type"`
}

type TaskResponse struct {
	TaskID     int
	Status     string
	ResultFile *string
	Error      *string
}
