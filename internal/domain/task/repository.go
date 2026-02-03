package task

import (
	"database/sql"
	"errors"

	"github.com/sirupsen/logrus"
)

type TaskRepository struct{}

type TaskRepositoryInterface interface {
	Create(tx *sql.Tx, task *Task) (int, error)
	GetByID(db *sql.DB, id int) (*Task, error)
	GetByUserID(db *sql.DB, userID int, offset int, limit int) ([]*Task, error)
	MarkProcessing(tx *sql.Tx, id int) error
	MarkSuccess(tx *sql.Tx, id int, resultFile string) error
	MarkFailed(tx *sql.Tx, id int, errorMessage string) error
}

func NewTaskRepository() TaskRepositoryInterface {
	return &TaskRepository{}
}

func (r *TaskRepository) Create(
	tx *sql.Tx,
	task *Task,
) (int, error) {
	query := `
		INSERT INTO tasks (
			user_id, task_type, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING task_id
	`

	var taskID int
	err := tx.QueryRow(
		query,
		task.UserID,
		task.TaskType,
		task.Status,
	).Scan(&taskID)

	if err != nil {
		return 0, err
	}

	return taskID, nil
}

func (r *TaskRepository) GetByID(
	db *sql.DB,
	taskID int,
) (*Task, error) {
	query := `
		SELECT
			task_id, user_id, task_type, status,
			result_file, error_message,
			created_at, updated_at
		FROM tasks
		WHERE task_id = $1
	`

	row := db.QueryRow(query, taskID)

	var t Task
	err := row.Scan(
		&t.TaskID,
		&t.UserID,
		&t.TaskType,
		&t.Status,
		&t.ResultFile,
		&t.ErrorMessage,
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

func (r *TaskRepository) GetByUserID(
	db *sql.DB,
	userID int,
	offset int,
	limit int,
) ([]*Task, error) {
	query := `
		SELECT
			task_id, user_id, task_type, status,
			result_file, error_message,
			created_at, updated_at
		FROM tasks
		WHERE user_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logrus.WithError(err).Warn("Failed to close rows")
		}
	}()
	var tasks []*Task

	for rows.Next() {
		var t Task
		err := rows.Scan(
			&t.TaskID,
			&t.UserID,
			&t.TaskType,
			&t.Status,
			&t.ResultFile,
			&t.ErrorMessage,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"service": "task_repository",
				"user_id": userID,
				"error":   err.Error(),
			}).Error("Error scanning task row")
			continue
		}
		tasks = append(tasks, &t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *TaskRepository) MarkProcessing(
	tx *sql.Tx,
	taskID int,
) error {
	query := `
		UPDATE tasks
		SET status = 'PROCESSING', updated_at = NOW()
		WHERE task_id = $1
	`
	_, err := tx.Exec(query, taskID)
	return err
}

func (r *TaskRepository) MarkSuccess(
	tx *sql.Tx,
	taskID int,
	resultFile string,
) error {
	query := `
		UPDATE tasks
		SET status = 'SUCCESS',
		    result_file = $1,
		    updated_at = NOW()
		WHERE task_id = $2
	`
	_, err := tx.Exec(query, resultFile, taskID)
	return err
}

func (r *TaskRepository) MarkFailed(
	tx *sql.Tx,
	id int,
	errorMessage string,
) error {
	query := `
		UPDATE tasks
		SET status = 'FAILED',
		    error_message = $1,
		    updated_at = NOW()
		WHERE task_id = $2
	`
	_, err := tx.Exec(query, errorMessage, id)
	return err
}
