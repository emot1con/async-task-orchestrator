package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTaskService is a mock implementation of TaskServiceInterface
type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) CreateTask(task *Task) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *MockTaskService) GetTask(id int) (*Task, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Task), args.Error(1)
}

func (m *MockTaskService) GetTasks(userID int) ([]*Task, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Task), args.Error(1)
}

// setupTestRouter creates a test router with mocked service
func setupTestRouter(service TaskServiceInterface) (*gin.Engine, *TaskController) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	controller := NewTaskController(service)

	return router, controller
}

// Helper to add authenticated user to context
func addAuthenticatedUser(c *gin.Context, userID int) {
	c.Set("userID", userID)
}

func TestGetTask_Success_OwnTask(t *testing.T) {
	mockService := new(MockTaskService)
	router, controller := setupTestRouter(mockService)

	authenticatedUserID := 1
	taskID := 123

	expectedTask := &Task{
		ID:        taskID,
		UserID:    authenticatedUserID, // Same as authenticated user
		TaskType:  "IMAGE_RESIZE",
		Status:    "COMPLETED",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockService.On("GetTask", taskID).Return(expectedTask, nil)

	// Setup route with auth middleware mock
	router.GET("/tasks/:id", func(c *gin.Context) {
		addAuthenticatedUser(c, authenticatedUserID)
		controller.GetTask(c)
	})

	req := httptest.NewRequest("GET", fmt.Sprintf("/tasks/%d", taskID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(taskID), response["id"])
	assert.Equal(t, float64(authenticatedUserID), response["user_id"])
	assert.Equal(t, "IMAGE_RESIZE", response["task_type"])
	assert.Equal(t, "COMPLETED", response["status"])

	mockService.AssertExpectations(t)
}

func TestGetTask_Forbidden_OtherUserTask(t *testing.T) {
	mockService := new(MockTaskService)
	router, controller := setupTestRouter(mockService)

	authenticatedUserID := 1
	taskOwnerID := 2 // Different user
	taskID := 123

	expectedTask := &Task{
		ID:        taskID,
		UserID:    taskOwnerID, // Different from authenticated user
		TaskType:  "IMAGE_RESIZE",
		Status:    "COMPLETED",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockService.On("GetTask", taskID).Return(expectedTask, nil)

	// Setup route
	router.GET("/tasks/:id", func(c *gin.Context) {
		addAuthenticatedUser(c, authenticatedUserID)
		controller.GetTask(c)
	})

	req := httptest.NewRequest("GET", fmt.Sprintf("/tasks/%d", taskID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "You can only view your own tasks")

	mockService.AssertExpectations(t)
}

func TestGetTask_NotFound(t *testing.T) {
	mockService := new(MockTaskService)
	router, controller := setupTestRouter(mockService)

	authenticatedUserID := 1
	taskID := 999

	mockService.On("GetTask", taskID).Return(nil, errors.New("task not found"))

	router.GET("/tasks/:id", func(c *gin.Context) {
		addAuthenticatedUser(c, authenticatedUserID)
		controller.GetTask(c)
	})

	req := httptest.NewRequest("GET", fmt.Sprintf("/tasks/%d", taskID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Task not found")

	mockService.AssertExpectations(t)
}

func TestGetTask_InvalidTaskID(t *testing.T) {
	mockService := new(MockTaskService)
	router, controller := setupTestRouter(mockService)

	router.GET("/tasks/:id", func(c *gin.Context) {
		addAuthenticatedUser(c, 1)
		controller.GetTask(c)
	})

	req := httptest.NewRequest("GET", "/tasks/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Invalid task ID")

	// Should not call service
	mockService.AssertNotCalled(t, "GetTask")
}

func TestGetTask_Unauthorized_NoUserInContext(t *testing.T) {
	mockService := new(MockTaskService)
	router, controller := setupTestRouter(mockService)

	taskID := 123

	expectedTask := &Task{
		ID:        taskID,
		UserID:    1,
		TaskType:  "IMAGE_RESIZE",
		Status:    "COMPLETED",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockService.On("GetTask", taskID).Return(expectedTask, nil)

	// Don't add user to context (simulating missing JWT middleware)
	router.GET("/tasks/:id", controller.GetTask)

	req := httptest.NewRequest("GET", fmt.Sprintf("/tasks/%d", taskID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "User ID not found in context")
}

// TestGetTasksByUser_Success tests that user can retrieve their own tasks via JWT context
// Design: No user_id parameter in URL - directly uses authenticated user from JWT
func TestGetTasksByUser_Success_OwnTasks(t *testing.T) {
	mockService := new(MockTaskService)
	router, controller := setupTestRouter(mockService)

	authenticatedUserID := 1

	expectedTasks := []*Task{
		{
			ID:        1,
			UserID:    authenticatedUserID,
			TaskType:  "IMAGE_RESIZE",
			Status:    "COMPLETED",
			CreatedAt: time.Now(),
		},
		{
			ID:        2,
			UserID:    authenticatedUserID,
			TaskType:  "DATA_EXPORT",
			Status:    "PROCESSING",
			CreatedAt: time.Now(),
		},
	}

	mockService.On("GetTasks", authenticatedUserID).Return(expectedTasks, nil)

	// Route without :user_id parameter - uses JWT context only
	router.GET("/users/tasks", func(c *gin.Context) {
		addAuthenticatedUser(c, authenticatedUserID)
		controller.GetTasksByUser(c)
	})

	req := httptest.NewRequest("GET", "/users/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	tasks, ok := response["tasks"].([]interface{})
	require.True(t, ok)
	assert.Len(t, tasks, 2)

	mockService.AssertExpectations(t)
}

// TestGetTasksByUser_EmptyList tests that endpoint correctly handles empty task list
func TestGetTasksByUser_EmptyList(t *testing.T) {
	mockService := new(MockTaskService)
	router, controller := setupTestRouter(mockService)

	authenticatedUserID := 1

	// Return empty list
	mockService.On("GetTasks", authenticatedUserID).Return([]*Task{}, nil)

	router.GET("/users/tasks", func(c *gin.Context) {
		addAuthenticatedUser(c, authenticatedUserID)
		controller.GetTasksByUser(c)
	})

	req := httptest.NewRequest("GET", "/users/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	tasks, ok := response["tasks"].([]interface{})
	require.True(t, ok)
	assert.Len(t, tasks, 0)

	mockService.AssertExpectations(t)
}

func TestCreateTask_Success(t *testing.T) {
	mockService := new(MockTaskService)
	router, controller := setupTestRouter(mockService)

	authenticatedUserID := 1

	mockService.On("CreateTask", mock.AnythingOfType("*task.Task")).Return(nil).Run(func(args mock.Arguments) {
		task := args.Get(0).(*Task)
		task.ID = 123 // Simulate DB assigning ID
	})

	router.POST("/tasks", func(c *gin.Context) {
		addAuthenticatedUser(c, authenticatedUserID)
		controller.CreateTask(c)
	})

	reqBody := `{"task_type": "IMAGE_RESIZE"}`
	req := httptest.NewRequest("POST", "/tasks", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(123), response["task_id"])
	assert.Equal(t, "PENDING", response["status"])
	assert.Contains(t, response["message"], "Task created successfully")

	mockService.AssertExpectations(t)
}

func TestCreateTask_InvalidRequest(t *testing.T) {
	mockService := new(MockTaskService)
	router, controller := setupTestRouter(mockService)

	router.POST("/tasks", func(c *gin.Context) {
		addAuthenticatedUser(c, 1)
		controller.CreateTask(c)
	})

	// Invalid JSON
	reqBody := `{"task_type": }`
	req := httptest.NewRequest("POST", "/tasks", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockService.AssertNotCalled(t, "CreateTask")
}

func TestCreateTask_ServiceError(t *testing.T) {
	mockService := new(MockTaskService)
	router, controller := setupTestRouter(mockService)

	authenticatedUserID := 1

	mockService.On("CreateTask", mock.AnythingOfType("*task.Task")).Return(errors.New("database error"))

	router.POST("/tasks", func(c *gin.Context) {
		addAuthenticatedUser(c, authenticatedUserID)
		controller.CreateTask(c)
	})

	reqBody := `{"task_type": "IMAGE_RESIZE"}`
	req := httptest.NewRequest("POST", "/tasks", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockService.AssertExpectations(t)
}
