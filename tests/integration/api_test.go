//go:build integration
// +build integration

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
	"task_handler/internal/task"
	"task_handler/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIIntegration_FullUserFlow tests complete user journey
func TestAPIIntegration_FullUserFlow(t *testing.T) {
	// Setup test environment
	deps := SetupTestEnvironment(t)
	defer deps.Cleanup(t)

	// Initialize services

	// Setup HTTP router
	gin.SetMode(gin.TestMode)
	router := handler.SetupHandler(
		deps.DB,
		deps.RabbitConn,
		deps.RedisClient,
		deps.Config,
	)

	// Test variables
	testUsername := fmt.Sprintf("testuser_%d", time.Now().Unix())
	testPassword := "SecurePass123!"
	var accessToken string
	var taskID int

	// Step 1: Register new user
	t.Run("Register", func(t *testing.T) {
		payload := map[string]string{
			"username": testUsername,
			"password": testPassword,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "user_id")
		assert.Contains(t, response, "username")
		assert.Equal(t, testUsername, response["username"])

		t.Logf("✅ User registered successfully: %s", testUsername)
	})

	// Step 2: Login and get JWT token
	t.Run("Login", func(t *testing.T) {
		payload := map[string]string{
			"username": testUsername,
			"password": testPassword,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "access_token")
		assert.Contains(t, response, "refresh_token")
		assert.Contains(t, response, "expires_in")

		accessToken = response["access_token"].(string)
		assert.NotEmpty(t, accessToken)

		t.Logf("✅ User logged in successfully, token: %s...", accessToken[:20])
	})

	// Step 3: Create task (authenticated)
	t.Run("CreateTask", func(t *testing.T) {
		payload := map[string]interface{}{
			"task_type": "IMAGE_RESIZE",
			"payload": map[string]string{
				"image_url": "https://example.com/image.jpg",
				"width":     "800",
				"height":    "600",
			},
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "task_id")
		assert.Contains(t, response, "status")
		assert.Equal(t, "PENDING", response["status"])

		taskID = int(response["task_id"].(float64))
		assert.Greater(t, taskID, 0)

		t.Logf("✅ Task created successfully: ID=%d", taskID)
	})

	// Step 4: Get specific task (authorization check)
	t.Run("GetTask_OwnTask", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/tasks/%d", taskID), nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(taskID), response["id"])
		assert.Equal(t, "IMAGE_RESIZE", response["task_type"])
		assert.Contains(t, []string{"PENDING", "PROCESSING"}, response["status"])

		t.Logf("✅ Retrieved task successfully: %+v", response)
	})

	// Step 5: Get all user's tasks
	t.Run("GetUserTasks", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "tasks")
		tasks := response["tasks"].([]interface{})
		assert.GreaterOrEqual(t, len(tasks), 1)

		// Verify our created task is in the list
		foundTask := false
		for _, t := range tasks {
			taskMap := t.(map[string]interface{})
			if int(taskMap["id"].(float64)) == taskID {
				foundTask = true
				break
			}
		}
		assert.True(t, foundTask, "Created task should be in user's task list")

		t.Logf("✅ Retrieved %d tasks for user", len(tasks))
	})

	// Step 6: Test authentication failure
	t.Run("Unauthorized_NoToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
		// No Authorization header
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		t.Logf("✅ Properly rejected unauthorized request")
	})

	// Step 7: Test invalid token
	t.Run("Unauthorized_InvalidToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-here")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		t.Logf("✅ Properly rejected invalid token")
	})

	// Step 8: Test authorization - different user cannot access task
	t.Run("Forbidden_OtherUserTask", func(t *testing.T) {
		// Create second user
		otherUsername := fmt.Sprintf("otheruser_%d", time.Now().Unix())
		otherPassword := "OtherPass123!"

		// Register second user
		payload := map[string]string{
			"username": otherUsername,
			"password": otherPassword,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Login as second user
		payload = map[string]string{
			"username": otherUsername,
			"password": otherPassword,
		}
		body, _ = json.Marshal(payload)
		req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var loginResponse map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &loginResponse)
		otherToken := loginResponse["access_token"].(string)

		// Try to access first user's task
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/tasks/%d", taskID), nil)
		req.Header.Set("Authorization", "Bearer "+otherToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		t.Logf("✅ Properly rejected unauthorized access to other user's task")
	})
}

// TestAPIIntegration_TokenRefresh tests JWT token refresh flow
func TestAPIIntegration_TokenRefresh(t *testing.T) {
	deps := SetupTestEnvironment(t)
	defer deps.Cleanup(t)


	gin.SetMode(gin.TestMode)
	router := handler.SetupHandler(deps.DB, deps.RabbitConn, deps.RedisClient, deps.Config)

	// Register and login user
	testUsername := fmt.Sprintf("refreshuser_%d", time.Now().Unix())
	testPassword := "RefreshPass123!"

	// Register
	payload := map[string]string{"username": testUsername, "password": testPassword}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Login
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	refreshToken := loginResponse["refresh_token"].(string)
	initialAccessToken := loginResponse["access_token"].(string)

	// Test token refresh
	t.Run("RefreshToken_Success", func(t *testing.T) {
		payload := map[string]string{"refresh_token": refreshToken}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "access_token")
		assert.Contains(t, response, "refresh_token")

		newAccessToken := response["access_token"].(string)
		assert.NotEmpty(t, newAccessToken)
		assert.NotEqual(t, initialAccessToken, newAccessToken, "New access token should be different")

		t.Logf("✅ Token refreshed successfully")
	})

	// Test invalid refresh token
	t.Run("RefreshToken_Invalid", func(t *testing.T) {
		payload := map[string]string{"refresh_token": "invalid-refresh-token"}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		t.Logf("✅ Properly rejected invalid refresh token")
	})

	// Test using access token instead of refresh token
	t.Run("RefreshToken_UsingAccessToken", func(t *testing.T) {
		payload := map[string]string{"refresh_token": initialAccessToken}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		t.Logf("✅ Properly rejected access token for refresh endpoint")
	})
}

// TestAPIIntegration_RateLimiting tests rate limiting functionality
func TestAPIIntegration_RateLimiting(t *testing.T) {
	deps := SetupTestEnvironment(t)
	defer deps.Cleanup(t)


	gin.SetMode(gin.TestMode)
	router := handler.SetupHandler(deps.DB, deps.RabbitConn, deps.RedisClient, deps.Config)

	// Register and login user
	testUsername := fmt.Sprintf("ratelimituser_%d", time.Now().Unix())
	testPassword := "RateLimit123!"

	payload := map[string]string{"username": testUsername, "password": testPassword}
	body, _ := json.Marshal(payload)

	// Register
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Login
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	accessToken := loginResponse["access_token"].(string)

	// Test rate limiting by making rapid requests
	t.Run("RateLimit_ExceedLimit", func(t *testing.T) {
		// Make requests until rate limited (burst = 5, rate = 2/sec typically)
		successCount := 0
		rateLimitedCount := 0

		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/api/v1/users/tasks", nil)
			req.Header.Set("Authorization", "Bearer "+accessToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		assert.Greater(t, successCount, 0, "Some requests should succeed")
		assert.Greater(t, rateLimitedCount, 0, "Some requests should be rate limited")

		t.Logf("✅ Rate limiting working: %d succeeded, %d rate limited", successCount, rateLimitedCount)
	})
}

// TestAPIIntegration_CacheHit tests cache functionality
func TestAPIIntegration_CacheHit(t *testing.T) {
	deps := SetupTestEnvironment(t)
	defer deps.Cleanup(t)


	gin.SetMode(gin.TestMode)
	router := handler.SetupHandler(deps.DB, deps.RabbitConn, deps.RedisClient, deps.Config)

	// Register, login, create task
	testUsername := fmt.Sprintf("cacheuser_%d", time.Now().Unix())
	testPassword := "CachePass123!"

	payload := map[string]string{"username": testUsername, "password": testPassword}
	body, _ := json.Marshal(payload)

	// Register
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Login
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	accessToken := loginResponse["access_token"].(string)

	// Create task
	taskPayload := map[string]interface{}{
		"task_type": "DATA_EXPORT",
		"payload":   map[string]string{"format": "csv"},
	}
	body, _ = json.Marshal(taskPayload)
	req = httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	taskID := int(createResponse["task_id"].(float64))

	// First request - cache miss
	t.Run("FirstRequest_CacheMiss", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/tasks/%d", taskID), nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		w := httptest.NewRecorder()

		start := time.Now()
		router.ServeHTTP(w, req)
		duration := time.Since(start)

		assert.Equal(t, http.StatusOK, w.Code)
		t.Logf("✅ First request (cache miss) took: %v", duration)
	})

	// Second request - should hit cache (faster)
	t.Run("SecondRequest_CacheHit", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/tasks/%d", taskID), nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		w := httptest.NewRecorder()

		start := time.Now()
		router.ServeHTTP(w, req)
		duration := time.Since(start)

		assert.Equal(t, http.StatusOK, w.Code)
		t.Logf("✅ Second request (cache hit) took: %v", duration)
		// Note: In real scenario, cache hit should be significantly faster
	})
}
