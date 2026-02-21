package integration

import (
	"task_handler/internal/auth"
	"task_handler/internal/domain/task"
	"task_handler/internal/domain/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestUser is a helper that inserts a user and returns its ID.
func createTestUser(t *testing.T, username string) int {
	t.Helper()
	repo := user.NewUserRepository()
	tx, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	hashedPwd, _ := auth.GeneratePasswordHash("testpass")
	id, err := repo.Create(tx, &user.User{Username: username, Password: hashedPwd})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	return id
}

func TestIntegration_TaskRepository_Create(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "task_creator")
	repo := task.NewTaskRepository()

	tx, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	tk := &task.Task{
		UserID:   userID,
		TaskType: "generate_report",
		Status:   "PENDING",
	}

	taskID, err := repo.Create(tx, tk)
	require.NoError(t, err)
	assert.Greater(t, taskID, 0)
	require.NoError(t, tx.Commit())
}

func TestIntegration_TaskRepository_GetByID(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "task_getter")
	repo := task.NewTaskRepository()

	tx, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	taskID, err := repo.Create(tx, &task.Task{UserID: userID, TaskType: "send_email", Status: "PENDING"})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	found, err := repo.GetByID(testEnv.DB, taskID)
	require.NoError(t, err)
	assert.Equal(t, taskID, found.TaskID)
	assert.Equal(t, userID, found.UserID)
	assert.Equal(t, "send_email", found.TaskType)
	assert.Equal(t, "PENDING", found.Status)
}

func TestIntegration_TaskRepository_GetByID_NotFound(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	repo := task.NewTaskRepository()
	_, err := repo.GetByID(testEnv.DB, 99999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIntegration_TaskRepository_GetByUserID(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "tasks_by_user")
	repo := task.NewTaskRepository()

	tx, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	taskTypes := []string{"send_email", "generate_report", "resize_image"}
	for _, tt := range taskTypes {
		_, err := repo.Create(tx, &task.Task{UserID: userID, TaskType: tt, Status: "PENDING"})
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit())

	tasks, err := repo.GetByUserID(testEnv.DB, userID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func TestIntegration_TaskRepository_GetByUserID_Pagination(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "pagination_user")
	repo := task.NewTaskRepository()

	tx, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	for i := 0; i < 5; i++ {
		_, err := repo.Create(tx, &task.Task{UserID: userID, TaskType: "send_email", Status: "PENDING"})
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit())

	// Page 1: limit 2
	page1, err := repo.GetByUserID(testEnv.DB, userID, 0, 2)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// Page 2: limit 2 offset 2
	page2, err := repo.GetByUserID(testEnv.DB, userID, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// IDs should not overlap
	assert.NotEqual(t, page1[0].TaskID, page2[0].TaskID)
}

func TestIntegration_TaskRepository_MarkProcessing(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "mark_processing_user")
	repo := task.NewTaskRepository()

	tx, _ := testEnv.DB.Begin()
	defer func() { _ = tx.Rollback() }()
	taskID, _ := repo.Create(tx, &task.Task{UserID: userID, TaskType: "cleanup_temp", Status: "PENDING"})
	require.NoError(t, tx.Commit())

	tx2, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()

	err = repo.MarkProcessing(tx2, taskID)
	require.NoError(t, err)
	require.NoError(t, tx2.Commit())

	found, _ := repo.GetByID(testEnv.DB, taskID)
	assert.Equal(t, "PROCESSING", found.Status)
}

func TestIntegration_TaskRepository_MarkSuccess(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "mark_success_user")
	repo := task.NewTaskRepository()

	tx, _ := testEnv.DB.Begin()
	defer func() { _ = tx.Rollback() }()
	taskID, _ := repo.Create(tx, &task.Task{UserID: userID, TaskType: "generate_report", Status: "PROCESSING"})
	require.NoError(t, tx.Commit())

	tx2, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()

	err = repo.MarkSuccess(tx2, taskID, "report_2026.pdf")
	require.NoError(t, err)
	require.NoError(t, tx2.Commit())

	found, _ := repo.GetByID(testEnv.DB, taskID)
	assert.Equal(t, "SUCCESS", found.Status)
	require.NotNil(t, found.ResultFile)
	assert.Equal(t, "report_2026.pdf", *found.ResultFile)
}

func TestIntegration_TaskRepository_MarkFailed(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "mark_failed_user")
	repo := task.NewTaskRepository()

	tx, _ := testEnv.DB.Begin()
	defer func() { _ = tx.Rollback() }()
	taskID, _ := repo.Create(tx, &task.Task{UserID: userID, TaskType: "resize_image", Status: "PROCESSING"})
	require.NoError(t, tx.Commit())

	tx2, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()

	err = repo.MarkFailed(tx2, taskID, "out of memory")
	require.NoError(t, err)
	require.NoError(t, tx2.Commit())

	found, _ := repo.GetByID(testEnv.DB, taskID)
	assert.Equal(t, "FAILED", found.Status)
	require.NotNil(t, found.ErrorMessage)
	assert.Equal(t, "out of memory", *found.ErrorMessage)
}

func TestIntegration_TaskRepository_GetByUserID_Empty(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "empty_tasks_user")
	repo := task.NewTaskRepository()

	tasks, err := repo.GetByUserID(testEnv.DB, userID, 0, 10)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}
