package task_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"task_handler/internal/auth"
	"task_handler/internal/domain/task"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- Mock TaskService ----

type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) CreateTask(t *task.Task) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockTaskService) GetTask(taskID int) (*task.Task, error) {
	args := m.Called(taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*task.Task), args.Error(1)
}

func (m *MockTaskService) GetTasks(userID int, page int, limit int) ([]*task.Task, error) {
	args := m.Called(userID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

// ---- Helpers ----

func setupRouter(svc task.TaskServiceInterface) *gin.Engine {
	r := gin.New()
	ctrl := task.NewTaskController(svc)

	// Inject userID directly into context for test (bypass JWT middleware)
	authMiddleware := func(c *gin.Context) {
		c.Set(auth.UserIDKey, 1)
		c.Next()
	}

	r.POST("/tasks", authMiddleware, ctrl.CreateTask)
	r.GET("/tasks/:id", authMiddleware, ctrl.GetTask)
	r.GET("/tasks", authMiddleware, ctrl.GetTasksByUser)
	return r
}

// ---- CreateTask Tests ----

func TestCreateTask_Success(t *testing.T) {
	svc := new(MockTaskService)
	svc.On("CreateTask", mock.MatchedBy(func(tk *task.Task) bool {
		return tk.UserID == 1 && tk.TaskType == "generate_report" && tk.Status == "PENDING"
	})).Return(nil)

	r := setupRouter(svc)
	body := `{"task_type":"generate_report"}`
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Task created successfully", resp["message"])
	svc.AssertExpectations(t)
}

func TestCreateTask_InvalidTaskType(t *testing.T) {
	svc := new(MockTaskService)
	r := setupRouter(svc)

	body := `{"task_type":"unknown_type"}`
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "CreateTask")
}

func TestCreateTask_MissingTaskType(t *testing.T) {
	svc := new(MockTaskService)
	r := setupRouter(svc)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "CreateTask")
}

func TestCreateTask_ServiceError(t *testing.T) {
	svc := new(MockTaskService)
	svc.On("CreateTask", mock.Anything).Return(errors.New("db error"))

	r := setupRouter(svc)
	body := `{"task_type":"send_email"}`
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateTask_AllValidTypes(t *testing.T) {
	validTypes := []string{"send_email", "generate_report", "resize_image", "cleanup_temp"}

	for _, taskType := range validTypes {
		t.Run(taskType, func(t *testing.T) {
			svc := new(MockTaskService)
			svc.On("CreateTask", mock.Anything).Return(nil)

			r := setupRouter(svc)
			body, _ := json.Marshal(map[string]string{"task_type": taskType})
			req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)
		})
	}
}

// ---- GetTask Tests ----

func TestGetTask_Success(t *testing.T) {
	svc := new(MockTaskService)
	now := time.Now()
	expectedTask := &task.Task{
		TaskID:    1,
		UserID:    1,
		TaskType:  "generate_report",
		Status:    "SUCCESS",
		CreatedAt: now,
		UpdatedAt: now,
	}
	svc.On("GetTask", 1).Return(expectedTask, nil)

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/tasks/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["id"])
	assert.Equal(t, "generate_report", resp["task_type"])
}

func TestGetTask_NotFound(t *testing.T) {
	svc := new(MockTaskService)
	svc.On("GetTask", 999).Return(nil, errors.New("task not found"))

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/tasks/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetTask_InvalidID(t *testing.T) {
	svc := new(MockTaskService)
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/tasks/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "GetTask")
}

func TestGetTask_Forbidden_OtherUserTask(t *testing.T) {
	svc := new(MockTaskService)
	// Task belongs to userID=99, but authenticated user is userID=1
	taskOfAnotherUser := &task.Task{TaskID: 5, UserID: 99, TaskType: "send_email", Status: "PENDING"}
	svc.On("GetTask", 5).Return(taskOfAnotherUser, nil)

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/tasks/5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---- GetTasksByUser Tests ----

func TestGetTasksByUser_Success(t *testing.T) {
	svc := new(MockTaskService)
	tasks := []*task.Task{
		{TaskID: 1, UserID: 1, TaskType: "send_email", Status: "PENDING"},
		{TaskID: 2, UserID: 1, TaskType: "generate_report", Status: "SUCCESS"},
	}
	svc.On("GetTasks", 1, 1, 15).Return(tasks, nil)

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/tasks?page=1&limit=15", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["count"])
}

func TestGetTasksByUser_DefaultPagination(t *testing.T) {
	svc := new(MockTaskService)
	svc.On("GetTasks", 1, 1, 15).Return([]*task.Task{}, nil)

	r := setupRouter(svc)
	// No page/limit query params → defaults to page=1, limit=15
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertCalled(t, "GetTasks", 1, 1, 15)
}

func TestGetTasksByUser_ServiceError(t *testing.T) {
	svc := new(MockTaskService)
	svc.On("GetTasks", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/tasks?page=1&limit=15", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
