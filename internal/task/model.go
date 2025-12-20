package task

import "time"

type Task struct {
	ID           int
	UserID       int
	TaskType     string
	Status       string
	ResultFile   *string
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TaskPayload struct {
	ID       int    `json:"id"`
	UserID   int    `json:"user_id"`
	TaskType string `json:"task_type"`
}

type TaskResponse struct {
	ID         int
	Status     string
	ResultFile *string
	Error      *string
}
