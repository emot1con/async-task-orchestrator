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
	ID       int
	UserID   int
	TaskType string
}

type TaskResponse struct {
	ID         int
	Status     string
	ResultFile *string
	Error      *string
}
