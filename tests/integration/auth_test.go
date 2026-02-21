package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"task_handler/internal/auth"
	"task_handler/internal/domain/task"
	"task_handler/internal/domain/user"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const authSecret = "integration-test-secret"

// ---- Router setup ----

func buildRouter() *gin.Engine {
	userRepo := user.NewUserRepository()
	userSvc := user.NewUserService(userRepo, testEnv.DB)
	userCtrl := user.NewUserController(userSvc, authSecret)

	// Use mock email sender + no RabbitMQ for auth test
	// TaskController needs a real service but we stub the queue via a noop
	r := gin.New()
	r.POST("/api/v1/register", userCtrl.Register)
	r.POST("/api/v1/login", userCtrl.Login)
	r.POST("/api/v1/refresh", userCtrl.RefreshToken)
	return r
}

// integrationTaskService uses the real repo + DB but skips RabbitMQ publishing.
type integrationTaskService struct {
	repo task.TaskRepositoryInterface
}

func (s *integrationTaskService) CreateTask(t *task.Task) error {
	if t.UserID == 0 || t.TaskType == "" {
		return fmt.Errorf("invalid task payload")
	}
	tx, err := testEnv.DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	taskID, err := s.repo.Create(tx, t)
	if err != nil {
		return err
	}
	t.TaskID = taskID
	return tx.Commit()
}

func (s *integrationTaskService) GetTask(taskID int) (*task.Task, error) {
	return s.repo.GetByID(testEnv.DB, taskID)
}

func (s *integrationTaskService) GetTasks(userID int, page int, limit int) ([]*task.Task, error) {
	offset := (page - 1) * limit
	return s.repo.GetByUserID(testEnv.DB, userID, offset, limit)
}

func buildTaskRouterWithSvc(userID int) *gin.Engine {
	taskRepo := task.NewTaskRepository()
	svc := &integrationTaskService{repo: taskRepo}
	ctrl := task.NewTaskController(svc)

	r := gin.New()
	injectUser := func(c *gin.Context) {
		c.Set(auth.UserIDKey, userID)
		c.Next()
	}
	r.POST("/api/v1/tasks", injectUser, ctrl.CreateTask)
	r.GET("/api/v1/tasks/:id", injectUser, ctrl.GetTask)
	r.GET("/api/v1/tasks", injectUser, ctrl.GetTasksByUser)
	return r
}

// ---- Auth Integration Tests ----

func TestIntegration_Auth_RegisterAndLogin(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	r := buildRouter()

	// Register
	regBody := `{"username":"auth_user","password":"securepassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Login
	loginBody := `{"username":"auth_user","password":"securepassword"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBufferString(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	var tokens map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &tokens))
	assert.NotEmpty(t, tokens["access_token"])
	assert.NotEmpty(t, tokens["refresh_token"])
}

func TestIntegration_Auth_RegisterDuplicate(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	r := buildRouter()
	body := `{"username":"dup_user","password":"pass"}`

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if i == 0 {
			assert.Equal(t, http.StatusCreated, w.Code)
		} else {
			assert.Equal(t, http.StatusConflict, w.Code)
		}
	}
}

func TestIntegration_Auth_LoginWrongPassword(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	r := buildRouter()

	// Register
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register",
		bytes.NewBufferString(`{"username":"wrongpass_user","password":"correctpass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Login with wrong password
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/login",
		bytes.NewBufferString(`{"username":"wrongpass_user","password":"wrongpass"}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestIntegration_Auth_RefreshToken(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	r := buildRouter()

	// Register & login
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register",
		bytes.NewBufferString(`{"username":"refresh_user","password":"pass123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/login",
		bytes.NewBufferString(`{"username":"refresh_user","password":"pass123"}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &loginResp))
	refreshToken := loginResp["refresh_token"].(string)

	// Refresh
	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", bytes.NewBuffer(refreshBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)
	var refreshResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &refreshResp))
	assert.NotEmpty(t, refreshResp["access_token"])
}

// ---- Task API Integration Tests ----

func TestIntegration_Task_CreateAndGet(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "task_api_user")
	r := buildTaskRouterWithSvc(userID)

	// Create task
	body := `{"task_type":"generate_report"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	taskID := int(createResp["task_id"].(float64))
	assert.Greater(t, taskID, 0)

	// Get task
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", taskID), nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code)
	var getResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &getResp))
	assert.Equal(t, float64(taskID), getResp["id"])
	assert.Equal(t, "generate_report", getResp["task_type"])
	assert.Equal(t, "PENDING", getResp["status"])
}

func TestIntegration_Task_GetList(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "task_list_user")
	r := buildTaskRouterWithSvc(userID)

	// Create 3 tasks
	for _, tt := range []string{"send_email", "resize_image", "cleanup_temp"} {
		body, _ := json.Marshal(map[string]string{"task_type": tt})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// Get list
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(3), resp["count"])
}

func TestIntegration_Task_Forbidden_OtherUser(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	ownerID := createTestUser(t, "task_owner")
	otherID := createTestUser(t, "task_other")

	// Create task as owner
	ownerRouter := buildTaskRouterWithSvc(ownerID)
	body := `{"task_type":"send_email"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ownerRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	taskID := int(createResp["task_id"].(float64))

	// Try to get it as another user
	otherRouter := buildTaskRouterWithSvc(otherID)
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", taskID), nil)
	w2 := httptest.NewRecorder()
	otherRouter.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusForbidden, w2.Code)
}

func TestIntegration_Task_InvalidType(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	userID := createTestUser(t, "invalid_task_user")
	r := buildTaskRouterWithSvc(userID)

	body := `{"task_type":"hack_the_planet"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
