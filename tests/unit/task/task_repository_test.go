package task_test

import (
	"database/sql"
	"testing"
	"time"

	"task_handler/internal/domain/task"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func newTx(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) *sql.Tx {
	t.Helper()
	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	return tx
}

// ---- Create ----

func TestTaskRepository_Create_Success(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO tasks`).
		WithArgs(1, "generate_report", "PENDING").
		WillReturnRows(sqlmock.NewRows([]string{"task_id"}).AddRow(99))

	tx := newTx(t, db, mock)
	id, err := repo.Create(tx, &task.Task{UserID: 1, TaskType: "generate_report", Status: "PENDING"})

	require.NoError(t, err)
	assert.Equal(t, 99, id)
}

func TestTaskRepository_Create_DBError(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO tasks`).
		WithArgs(1, "resize_image", "PENDING").
		WillReturnError(sql.ErrConnDone)

	tx := newTx(t, db, mock)
	id, err := repo.Create(tx, &task.Task{UserID: 1, TaskType: "resize_image", Status: "PENDING"})

	assert.Error(t, err)
	assert.Equal(t, 0, id)
}

// ---- GetByID ----

func TestTaskRepository_GetByID_Found(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"task_id", "user_id", "task_type", "status",
		"result_file", "error_message", "created_at", "updated_at",
	}).AddRow(5, 2, "send_email", "SUCCESS", nil, nil, now, now)

	mock.ExpectQuery(`SELECT`).WithArgs(5).WillReturnRows(rows)

	result, err := repo.GetByID(db, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, result.TaskID)
	assert.Equal(t, 2, result.UserID)
	assert.Equal(t, "send_email", result.TaskType)
	assert.Equal(t, "SUCCESS", result.Status)
}

func TestTaskRepository_GetByID_NotFound(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectQuery(`SELECT`).WithArgs(999).WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(db, 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
	assert.Nil(t, result)
}

func TestTaskRepository_GetByID_DBError(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectQuery(`SELECT`).WithArgs(1).WillReturnError(sql.ErrConnDone)

	result, err := repo.GetByID(db, 1)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// ---- GetByUserID ----

func TestTaskRepository_GetByUserID_Success(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"task_id", "user_id", "task_type", "status",
		"result_file", "error_message", "created_at", "updated_at",
	}).
		AddRow(1, 3, "send_email", "PENDING", nil, nil, now, now).
		AddRow(2, 3, "cleanup_temp", "SUCCESS", nil, nil, now, now)

	mock.ExpectQuery(`SELECT`).WithArgs(3, 15, 0).WillReturnRows(rows)

	tasks, err := repo.GetByUserID(db, 3, 0, 15)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, 1, tasks[0].TaskID)
	assert.Equal(t, 2, tasks[1].TaskID)
}

func TestTaskRepository_GetByUserID_Empty(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	rows := sqlmock.NewRows([]string{
		"task_id", "user_id", "task_type", "status",
		"result_file", "error_message", "created_at", "updated_at",
	})

	mock.ExpectQuery(`SELECT`).WithArgs(99, 15, 0).WillReturnRows(rows)

	tasks, err := repo.GetByUserID(db, 99, 0, 15)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestTaskRepository_GetByUserID_DBError(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectQuery(`SELECT`).WithArgs(1, 15, 0).WillReturnError(sql.ErrConnDone)

	tasks, err := repo.GetByUserID(db, 1, 0, 15)
	assert.Error(t, err)
	assert.Nil(t, tasks)
}

// ---- MarkProcessing ----

func TestTaskRepository_MarkProcessing_Success(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE tasks`).
		WithArgs(7).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tx := newTx(t, db, mock)
	err := repo.MarkProcessing(tx, 7)
	assert.NoError(t, err)
}

func TestTaskRepository_MarkProcessing_DBError(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE tasks`).
		WithArgs(7).
		WillReturnError(sql.ErrConnDone)

	tx := newTx(t, db, mock)
	err := repo.MarkProcessing(tx, 7)
	assert.Error(t, err)
}

// ---- MarkSuccess ----

func TestTaskRepository_MarkSuccess_Success(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE tasks`).
		WithArgs("result.txt", 10).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tx := newTx(t, db, mock)
	err := repo.MarkSuccess(tx, 10, "result.txt")
	assert.NoError(t, err)
}

func TestTaskRepository_MarkSuccess_DBError(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE tasks`).
		WithArgs("result.txt", 10).
		WillReturnError(sql.ErrConnDone)

	tx := newTx(t, db, mock)
	err := repo.MarkSuccess(tx, 10, "result.txt")
	assert.Error(t, err)
}

// ---- MarkFailed ----

func TestTaskRepository_MarkFailed_Success(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE tasks`).
		WithArgs("disk full", 3).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tx := newTx(t, db, mock)
	err := repo.MarkFailed(tx, 3, "disk full")
	assert.NoError(t, err)
}

func TestTaskRepository_MarkFailed_DBError(t *testing.T) {
	db, mock := newDB(t)
	repo := task.NewTaskRepository()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE tasks`).
		WithArgs("disk full", 3).
		WillReturnError(sql.ErrConnDone)

	tx := newTx(t, db, mock)
	err := repo.MarkFailed(tx, 3, "disk full")
	assert.Error(t, err)
}
