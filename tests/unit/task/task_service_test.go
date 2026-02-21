package task_test

import (
	"errors"
	"task_handler/internal/domain/task"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockTaskService_CreateTask_Success(t *testing.T) {
	svc := new(MockTaskService)
	taskInput := &task.Task{UserID: 1, TaskType: "generate_report", Status: "PENDING"}
	svc.On("CreateTask", taskInput).Return(nil)

	err := svc.CreateTask(taskInput)

	assert.NoError(t, err)
	svc.AssertExpectations(t)
}

func TestMockTaskService_CreateTask_Error(t *testing.T) {
	svc := new(MockTaskService)
	taskInput := &task.Task{UserID: 0, TaskType: "", Status: ""}
	svc.On("CreateTask", taskInput).Return(errors.New("invalid task payload"))

	err := svc.CreateTask(taskInput)

	assert.EqualError(t, err, "invalid task payload")
	svc.AssertExpectations(t)
}

func TestMockTaskService_GetTask_Found(t *testing.T) {
	svc := new(MockTaskService)
	now := time.Now()
	expected := &task.Task{TaskID: 5, UserID: 2, TaskType: "send_email", Status: "SUCCESS", CreatedAt: now}
	svc.On("GetTask", 5).Return(expected, nil)

	result, err := svc.GetTask(5)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
	svc.AssertExpectations(t)
}

func TestMockTaskService_GetTask_NotFound(t *testing.T) {
	svc := new(MockTaskService)
	svc.On("GetTask", 999).Return(nil, errors.New("task not found"))

	result, err := svc.GetTask(999)

	assert.Nil(t, result)
	assert.EqualError(t, err, "task not found")
	svc.AssertExpectations(t)
}

func TestMockTaskService_GetTasks_Success(t *testing.T) {
	svc := new(MockTaskService)
	tasks := []*task.Task{
		{TaskID: 1, UserID: 3, TaskType: "resize_image", Status: "PENDING"},
		{TaskID: 2, UserID: 3, TaskType: "cleanup_temp", Status: "SUCCESS"},
	}
	svc.On("GetTasks", 3, 1, 15).Return(tasks, nil)

	result, err := svc.GetTasks(3, 1, 15)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, tasks, result)
	svc.AssertExpectations(t)
}

func TestMockTaskService_GetTasks_Empty(t *testing.T) {
	svc := new(MockTaskService)
	svc.On("GetTasks", 99, 1, 15).Return([]*task.Task{}, nil)

	result, err := svc.GetTasks(99, 1, 15)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestTaskModel_Fields(t *testing.T) {
	resultFile := "report.pdf"
	errMsg := "timeout"
	now := time.Now()

	tk := &task.Task{
		TaskID:       10,
		UserID:       5,
		TaskType:     "generate_report",
		Status:       "SUCCESS",
		ResultFile:   &resultFile,
		ErrorMessage: &errMsg,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, 10, tk.TaskID)
	assert.Equal(t, 5, tk.UserID)
	assert.Equal(t, "generate_report", tk.TaskType)
	assert.Equal(t, "SUCCESS", tk.Status)
	assert.Equal(t, "report.pdf", *tk.ResultFile)
	assert.Equal(t, "timeout", *tk.ErrorMessage)
}
