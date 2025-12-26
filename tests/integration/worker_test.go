//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"task_handler/internal/task"
	"task_handler/internal/user"
	"task_handler/internal/worker"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkerIntegration_TaskProcessing tests end-to-end task processing
func TestWorkerIntegration_TaskProcessing(t *testing.T) {
	deps := SetupTestEnvironment(t)
	defer deps.Cleanup(t)

	// Create test user
	userRepo := user.NewUserRepository()
	userService := user.NewUserService(userRepo, deps.DB)

	testUsername := fmt.Sprintf("workeruser_%d", time.Now().Unix())
	userID, err := userService.CreateUser(testUsername, "WorkerPass123!")
	require.NoError(t, err)

	// Initialize task repository and service
	taskRepo := task.NewTaskRepository()
	taskService := task.NewTaskService(taskRepo, deps.DB, deps.RabbitConn, deps.RedisClient)

	// Create a task
	testTask := &task.Task{
		UserID:   userID,
		TaskType: "send_email",
		Status:   "PENDING",
	}

	err = taskService.CreateTask(testTask)
	require.NoError(t, err)
	assert.Greater(t, testTask.ID, 0)

	t.Logf("✅ Task created: ID=%d, Type=%s", testTask.ID, testTask.TaskType)

	// Start worker in goroutine
	workerDone := make(chan bool)
	go func() {
		// Worker will process one message and exit when channel closes
		worker.StartWorker(deps.RabbitConn, deps.DB, taskRepo, 1)
		workerDone <- true
	}()

	// Wait for task to be processed
	WaitForCondition(t, func() bool {
		retrievedTask, err := taskRepo.GetByID(deps.DB, testTask.ID)
		if err != nil {
			return false
		}
		return retrievedTask.Status == "COMPLETED" || retrievedTask.Status == "FAILED"
	}, 10*time.Second, "task to be processed")

	// Verify task was processed
	processedTask, err := taskRepo.GetByID(deps.DB, testTask.ID)
	require.NoError(t, err)

	assert.Contains(t, []string{"COMPLETED", "FAILED"}, processedTask.Status)
	assert.NotNil(t, processedTask.UpdatedAt)

	t.Logf("✅ Task processed: Status=%s", processedTask.Status)

	// Cleanup: close connection to stop worker
	deps.RabbitConn.Close()

	select {
	case <-workerDone:
		t.Log("✅ Worker stopped gracefully")
	case <-time.After(5 * time.Second):
		t.Log("⚠️  Worker didn't stop within timeout")
	}
}

// TestWorkerIntegration_TaskRetry tests retry mechanism
func TestWorkerIntegration_TaskRetry(t *testing.T) {
	deps := SetupTestEnvironment(t)
	defer deps.Cleanup(t)

	// Create test user
	userRepo := user.NewUserRepository()
	userService := user.NewUserService(userRepo, deps.DB)

	testUsername := fmt.Sprintf("retryuser_%d", time.Now().Unix())
	userID, err := userService.CreateUser(testUsername, "RetryPass123!")
	require.NoError(t, err)

	taskRepo := task.NewTaskRepository()

	// Manually insert a task that will fail using transaction
	testTask := &task.Task{
		UserID:   userID,
		TaskType: "INVALID_TASK_TYPE", // This will cause handleTask to fail
		Status:   "PENDING",
	}

	tx, err := deps.DB.Begin()
	require.NoError(t, err)
	taskID, err := taskRepo.Create(tx, testTask)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)
	testTask.ID = taskID

	// Publish to queue with retry count
	ch, err := deps.RabbitConn.Channel()
	require.NoError(t, err)
	defer ch.Close()

	// Declare queue
	_, err = ch.QueueDeclare("task_queue", true, false, false, false, nil)
	require.NoError(t, err)

	// Publish message with retry header
	payload := task.TaskPayload{
		ID:       testTask.ID,
		UserID:   userID,
		TaskType: "INVALID_TASK_TYPE",
	}
	body, _ := json.Marshal(payload)

	err = ch.Publish(
		"",
		"task_queue",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Headers: amqp.Table{
				"x-retry-count": int32(3), // Already at max retries
			},
		},
	)
	require.NoError(t, err)

	t.Logf("✅ Published task with max retries: ID=%d", testTask.ID)

	// Start worker
	workerDone := make(chan bool)
	go func() {
		worker.StartWorker(deps.RabbitConn, deps.DB, taskRepo, 1)
		workerDone <- true
	}()

	// Wait for task to be marked as FAILED
	WaitForCondition(t, func() bool {
		retrievedTask, err := taskRepo.GetByID(deps.DB, testTask.ID)
		if err != nil {
			return false
		}
		return retrievedTask.Status == "FAILED"
	}, 10*time.Second, "task to be marked as failed")

	// Verify task was marked as FAILED
	failedTask, err := taskRepo.GetByID(deps.DB, testTask.ID)
	require.NoError(t, err)

	assert.Equal(t, "FAILED", failedTask.Status)
	assert.NotNil(t, failedTask.ErrorMessage)
	if failedTask.ErrorMessage != nil {
		assert.Contains(t, *failedTask.ErrorMessage, "max retries")
		t.Logf("✅ Task properly failed after max retries: Error=%s", *failedTask.ErrorMessage)
	}

	// Cleanup
	deps.RabbitConn.Close()

	select {
	case <-workerDone:
		t.Log("✅ Worker stopped")
	case <-time.After(5 * time.Second):
		t.Log("⚠️  Worker timeout")
	}
}

// TestWorkerIntegration_ConcurrentProcessing tests multiple workers
func TestWorkerIntegration_ConcurrentProcessing(t *testing.T) {
	deps := SetupTestEnvironment(t)
	defer deps.Cleanup(t)

	// Create test user
	userRepo := user.NewUserRepository()
	userService := user.NewUserService(userRepo, deps.DB)

	testUsername := fmt.Sprintf("concurrentuser_%d", time.Now().Unix())
	userID, err := userService.CreateUser(testUsername, "ConcurrentPass123!")
	require.NoError(t, err)

	// Initialize services
	taskRepo := task.NewTaskRepository()
	taskService := task.NewTaskService(taskRepo, deps.DB, deps.RabbitConn, deps.RedisClient)

	// Create multiple tasks
	numTasks := 5
	taskIDs := make([]int, numTasks)

	for i := 0; i < numTasks; i++ {
		testTask := &task.Task{
			UserID:   userID,
			TaskType: "send_email",
			Status:   "PENDING",
		}

		err := taskService.CreateTask(testTask)
		require.NoError(t, err)
		taskIDs[i] = testTask.ID
	}

	t.Logf("✅ Created %d tasks", numTasks)

	// Start multiple workers
	numWorkers := 3
	workersDone := make(chan bool, numWorkers)

	for i := 1; i <= numWorkers; i++ {
		workerID := i
		go func() {
			worker.StartWorker(deps.RabbitConn, deps.DB, taskRepo, workerID)
			workersDone <- true
		}()
	}

	t.Logf("✅ Started %d workers", numWorkers)

	// Wait for all tasks to be processed
	WaitForCondition(t, func() bool {
		completedCount := 0
		for _, taskID := range taskIDs {
			retrievedTask, err := taskRepo.GetByID(deps.DB, taskID)
			if err != nil {
				continue
			}
			if retrievedTask.Status == "COMPLETED" || retrievedTask.Status == "FAILED" {
				completedCount++
			}
		}
		return completedCount == numTasks
	}, 15*time.Second, "all tasks to be processed")

	// Verify all tasks were processed
	completedCount := 0
	failedCount := 0

	for _, taskID := range taskIDs {
		processedTask, err := taskRepo.GetByID(deps.DB, taskID)
		require.NoError(t, err)

		if processedTask.Status == "COMPLETED" {
			completedCount++
		} else if processedTask.Status == "FAILED" {
			failedCount++
		}
	}

	t.Logf("✅ Task processing complete: %d completed, %d failed out of %d total",
		completedCount, failedCount, numTasks)

	assert.Equal(t, numTasks, completedCount+failedCount, "All tasks should be processed")

	// Cleanup
	deps.RabbitConn.Close()

	// Wait for workers to stop
	stoppedCount := 0
	for i := 0; i < numWorkers; i++ {
		select {
		case <-workersDone:
			stoppedCount++
		case <-time.After(5 * time.Second):
			break
		}
	}

	t.Logf("✅ %d/%d workers stopped", stoppedCount, numWorkers)
}

// TestWorkerIntegration_TaskStateTransitions tests state machine
func TestWorkerIntegration_TaskStateTransitions(t *testing.T) {
	deps := SetupTestEnvironment(t)
	defer deps.Cleanup(t)

	// Create test user
	userRepo := user.NewUserRepository()
	userService := user.NewUserService(userRepo, deps.DB)

	testUsername := fmt.Sprintf("stateuser_%d", time.Now().Unix())
	userID, err := userService.CreateUser(testUsername, "StatePass123!")
	require.NoError(t, err)

	// Initialize services
	taskRepo := task.NewTaskRepository()
	taskService := task.NewTaskService(taskRepo, deps.DB, deps.RabbitConn, deps.RedisClient)

	// Create task
	testTask := &task.Task{
		UserID:   userID,
		TaskType: "send_email",
		Status:   "PENDING",
	}

	err = taskService.CreateTask(testTask)
	require.NoError(t, err)

	t.Logf("✅ Task created: ID=%d, Status=%s", testTask.ID, testTask.Status)

	// Verify initial state
	retrievedTask, err := taskRepo.GetByID(deps.DB, testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", retrievedTask.Status)

	// Start worker
	go worker.StartWorker(deps.RabbitConn, deps.DB, taskRepo, 1)

	// Wait for PROCESSING state
	time.Sleep(500 * time.Millisecond)
	processingTask, err := taskRepo.GetByID(deps.DB, testTask.ID)
	if err == nil && processingTask.Status == "PROCESSING" {
		t.Logf("✅ Task transitioned to PROCESSING")
	}

	// Wait for final state
	WaitForCondition(t, func() bool {
		finalTask, err := taskRepo.GetByID(deps.DB, testTask.ID)
		if err != nil {
			return false
		}
		return finalTask.Status == "COMPLETED" || finalTask.Status == "FAILED"
	}, 10*time.Second, "task to reach final state")

	// Verify final state
	finalTask, err := taskRepo.GetByID(deps.DB, testTask.ID)
	require.NoError(t, err)

	assert.Contains(t, []string{"COMPLETED", "FAILED"}, finalTask.Status)
	t.Logf("✅ Task reached final state: %s", finalTask.Status)

	// Verify timestamps
	assert.NotNil(t, finalTask.CreatedAt)
	assert.NotNil(t, finalTask.UpdatedAt)
	assert.True(t, finalTask.UpdatedAt.After(finalTask.CreatedAt) || finalTask.UpdatedAt.Equal(finalTask.CreatedAt))

	t.Logf("✅ Timestamps valid: Created=%v, Updated=%v", finalTask.CreatedAt, finalTask.UpdatedAt)

	// Cleanup
	deps.RabbitConn.Close()
}
