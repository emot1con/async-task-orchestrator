package task

import "time"

type Task struct {
	ID           string
	UserID       int
	TaskType     string
	Status       string
	ResultFile   *string
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// type TaskPayload struct {
// 	UserID   int
// 	TaskType string
// }

// type TaskResponse struct {
// 	ID         string
// 	Status     string
// 	ResultFile *string
// 	Error      *string
// }
