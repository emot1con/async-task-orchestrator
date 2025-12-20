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
	GetByUserID(db *sql.DB, userID int) ([]*Task, error)
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
		RETURNING id
	`

	var id int
	err := tx.QueryRow(
		query,
		task.UserID,
		task.TaskType,
		task.Status,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *TaskRepository) GetByID(
	db *sql.DB,
	id int,
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
) ([]*Task, error) {
	query := `
		SELECT
			id, user_id, task_type, status,
			result_file, error_message,
			created_at, updated_at
		FROM tasks
		WHERE user_id = $1
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*Task

	for rows.Next() {
		var t Task
		err := rows.Scan(
			&t.ID,
			&t.UserID,
			&t.TaskType,
			&t.Status,
			&t.ResultFile,
			&t.ErrorMessage,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			logrus.Error("Error scanning task row: ", err)
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
	id int,
) error {
	logrus.Info("Marking task as PROCESSING: ", id)
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
	id int,
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
	id int,
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
