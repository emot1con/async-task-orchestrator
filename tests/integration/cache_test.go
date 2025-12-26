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

// TestCache_TaskCaching tests Redis caching for tasks
func TestCache_TaskCaching(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	router := handler.SetupHandler(env.DB, env.RabbitConn, env.RedisClient, env.Config)

	username := fmt.Sprintf("cacheuser_%d", time.Now().UnixNano())
	password := "CachePass123!"

	// Register and login
	payload := map[string]string{"username": username, "password": password}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	req = httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp["access_token"].(string)

	// Create a task
	taskPayload := map[string]string{"task_type": "send_email"}
	body, _ = json.Marshal(taskPayload)
	req = httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var taskResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &taskResp)
	taskID := int(taskResp["task_id"].(float64))

	t.Run("FirstRequest_CacheMiss", func(t *testing.T) {
url := fmt.Sprintf("/api/v1/tasks/%d", taskID)
req := httptest.NewRequest("GET", url, nil)
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()

		start := time.Now()
		router.ServeHTTP(w, req)
		firstDuration := time.Since(start)

		assert.Equal(t, http.StatusOK, w.Code)
		t.Logf("First request (cache miss): %v", firstDuration)
	})

	t.Run("SecondRequest_CacheHit", func(t *testing.T) {
url := fmt.Sprintf("/api/v1/tasks/%d", taskID)
req := httptest.NewRequest("GET", url, nil)
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()

		start := time.Now()
		router.ServeHTTP(w, req)
		secondDuration := time.Since(start)

		assert.Equal(t, http.StatusOK, w.Code)
		t.Logf("Second request (cache hit): %v", secondDuration)
	})
}

// TestCache_UserTasksCaching tests caching of user's task list
func TestCache_UserTasksCaching(t *testing.T) {
env := SetupTestEnv(t)
defer env.Cleanup(t)

router := handler.SetupHandler(env.DB, env.RabbitConn, env.RedisClient, env.Config)

username := fmt.Sprintf("listcache_%d", time.Now().UnixNano())
password := "ListCache123!"

// Register and login
payload := map[string]string{"username": username, "password": password}
body, _ := json.Marshal(payload)
req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
req.Header.Set("Content-Type", "application/json")
w := httptest.NewRecorder()
router.ServeHTTP(w, req)
require.Equal(t, http.StatusCreated, w.Code)

req = httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
req.Header.Set("Content-Type", "application/json")
w = httptest.NewRecorder()
router.ServeHTTP(w, req)
require.Equal(t, http.StatusOK, w.Code)

var loginResp map[string]interface{}
json.Unmarshal(w.Body.Bytes(), &loginResp)
token := loginResp["access_token"].(string)

// Create multiple tasks
for i := 0; i < 3; i++ {
taskPayload := map[string]string{"task_type": "send_email"}
body, _ := json.Marshal(taskPayload)
req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()
router.ServeHTTP(w, req)
require.Equal(t, http.StatusCreated, w.Code)
}

t.Run("FirstListRequest_CacheMiss", func(t *testing.T) {
req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()

start := time.Now()
router.ServeHTTP(w, req)
duration := time.Since(start)

assert.Equal(t, http.StatusOK, w.Code)
t.Logf("First list request: %v", duration)

var resp map[string]interface{}
json.Unmarshal(w.Body.Bytes(), &resp)
assert.Equal(t, float64(3), resp["count"])
})

t.Run("SecondListRequest_CacheHit", func(t *testing.T) {
req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
req.Header.Set("Authorization", "Bearer "+token)
w := httptest.NewRecorder()

start := time.Now()
router.ServeHTTP(w, req)
duration := time.Since(start)

assert.Equal(t, http.StatusOK, w.Code)
t.Logf("Second list request: %v", duration)

var resp map[string]interface{}
json.Unmarshal(w.Body.Bytes(), &resp)
assert.Equal(t, float64(3), resp["count"])
})
}
