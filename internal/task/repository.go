package task

import (
	"database/sql"
	"errors"
)

type TaskRepository struct{}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{}
}

func (r *TaskRepository) Create(
	tx *sql.Tx,
	task *Task,
) error {
	query := `
		INSERT INTO tasks (
			id, user_id, task_type, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`

	_, err := tx.Exec(
		query,
		task.ID,
		task.UserID,
		task.TaskType,
		task.Status,
	)
	return err
}

func (r *TaskRepository) GetByID(
	db *sql.DB,
	id string,
) (*Task, error) {
	query := `
		SELECT
			id, user_id, task_type, status,
			result_file, error_message,
			created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	row := db.QueryRow(query, id)

	var t Task
	err := row.Scan(
		&t.ID,
		&t.UserID,
		&t.TaskType,
		&t.Status,
		&t.ResultFile,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("task not found")
		}
		return nil, err
	}

	return &t, nil
}

func (r *TaskRepository) MarkProcessing(
	tx *sql.Tx,
	id string,
) error {
	query := `
		UPDATE tasks
		SET status = 'PROCESSING', updated_at = NOW()
		WHERE id = $1
	`
	_, err := tx.Exec(query, id)
	return err
}

func (r *TaskRepository) MarkSuccess(
	tx *sql.Tx,
	id string,
	resultFile string,
) error {
	query := `
		UPDATE tasks
		SET status = 'SUCCESS',
		    result_file = $1,
		    updated_at = NOW()
		WHERE id = $2
	`
	_, err := tx.Exec(query, resultFile, id)
	return err
}

func (r *TaskRepository) MarkFailed(
	tx *sql.Tx,
	id string,
	errorMessage string,
) error {
	query := `
		UPDATE tasks
		SET status = 'FAILED',
		    error_message = $1,
		    updated_at = NOW()
		WHERE id = $2
	`
	_, err := tx.Exec(query, errorMessage, id)
	return err
}
