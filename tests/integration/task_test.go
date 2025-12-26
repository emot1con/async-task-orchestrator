//go:build integration

package integration

import (
"bytes"
"encoding/json"
"fmt"
"net/http"
"net/http/httptest"
"testing"
"time"

"task_handler/internal/handler"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

// Helper to create user and get token
func createUserAndLogin(t *testing.T, router http.Handler) (string, int) {
	t.Helper()

	username := fmt.Sprintf("user_%d", time.Now().UnixNano())
	password := "TestPass123!"

	// Register
	payload := map[string]string{"username": username, "password": password}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var regResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &regResp)
	userID := int(regResp["user_id"].(float64))

	// Login
	req = httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp["access_token"].(string)

	return token, userID
}

// TestTask_CRUD tests task create, read operations
func TestTask_CRUD(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	router := handler.SetupHandler(env.DB, env.RabbitConn, env.RedisClient, env.Config)
	token, _ := createUserAndLogin(t, router)

	var taskID int

	t.Run("CreateTask_Success", func(t *testing.T) {
payload := map[string]string{"task_type": "send_email"}
body, _ := json.Marshal(payload)

req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp, "task_id")
		assert.Equal(t, "PENDING", resp["status"])
		assert.Equal(t, "Task created successfully", resp["message"])

		taskID = int(resp["task_id"].(float64))
	})

	t.Run("GetTask_Success", func(t *testing.T) {
url := fmt.Sprintf("/api/v1/tasks/%d", taskID)
req := httptest.NewRequest("GET", url, nil)
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(taskID), resp["id"])
		assert.Equal(t, "send_email", resp["task_type"])
		assert.Equal(t, "PENDING", resp["status"])
	})

	t.Run("GetTask_NotFound", func(t *testing.T) {
req := httptest.NewRequest("GET", "/api/v1/tasks/99999", nil)
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GetUserTasks_Success", func(t *testing.T) {
req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp, "tasks")
		assert.Contains(t, resp, "count")
		assert.GreaterOrEqual(t, int(resp["count"].(float64)), 1)
	})

	t.Run("CreateTask_InvalidTaskType", func(t *testing.T) {
payload := map[string]string{}
body, _ := json.Marshal(payload)

req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTask_Ownership tests that users can only access own tasks
func TestTask_Ownership(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	router := handler.SetupHandler(env.DB, env.RabbitConn, env.RedisClient, env.Config)

	// Create two users
	token1, _ := createUserAndLogin(t, router)
	token2, _ := createUserAndLogin(t, router)

	// User1 creates a task
	var user1TaskID int
	t.Run("User1_CreateTask", func(t *testing.T) {
payload := map[string]string{"task_type": "generate_report"}
body, _ := json.Marshal(payload)

req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+token1)
w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		user1TaskID = int(resp["task_id"].(float64))
	})

	// User2 tries to access User1's task
t.Run("User2_CannotAccessUser1Task", func(t *testing.T) {
url := fmt.Sprintf("/api/v1/tasks/%d", user1TaskID)
req := httptest.NewRequest("GET", url, nil)
req.Header.Set("Authorization", "Bearer "+token2)
w := httptest.NewRecorder()
router.ServeHTTP(w, req)

assert.Equal(t, http.StatusForbidden, w.Code)

var resp map[string]interface{}
json.Unmarshal(w.Body.Bytes(), &resp)
assert.Contains(t, resp["error"], "only view your own tasks")
})

// User2's task list should be empty
	t.Run("User2_EmptyTaskList", func(t *testing.T) {
req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
req.Header.Set("Authorization", "Bearer "+token2)
w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(0), resp["count"])
	})
}

// TestTask_Unauthorized tests unauthorized access
func TestTask_Unauthorized(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	router := handler.SetupHandler(env.DB, env.RabbitConn, env.RedisClient, env.Config)

	t.Run("NoToken", func(t *testing.T) {
req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("InvalidToken", func(t *testing.T) {
req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
req.Header.Set("Authorization", "Bearer invalid-token")
w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("MalformedHeader", func(t *testing.T) {
req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
req.Header.Set("Authorization", "NotBearer token")
w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestTask_AllTaskTypes tests all supported task types
func TestTask_AllTaskTypes(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	router := handler.SetupHandler(env.DB, env.RabbitConn, env.RedisClient, env.Config)
	token, _ := createUserAndLogin(t, router)

	taskTypes := []string{"send_email", "generate_report", "resize_image", "cleanup_temp"}

	for _, taskType := range taskTypes {
		t.Run("Create_"+taskType, func(t *testing.T) {
payload := map[string]string{"task_type": taskType}
body, _ := json.Marshal(payload)

req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Equal(t, "PENDING", resp["status"])
		})
	}
}
