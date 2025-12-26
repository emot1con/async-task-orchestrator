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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestAuth_RegisterLoginFlow tests complete authentication flow
func TestAuth_RegisterLoginFlow(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	router := handler.SetupHandler(env.DB, env.RabbitConn, env.RedisClient, env.Config)

	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	password := "SecurePass123!"

	var accessToken, refreshToken string

	t.Run("Register_Success", func(t *testing.T) {
		payload := map[string]string{"username": username, "password": password}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "User created successfully", resp["message"])
		assert.NotNil(t, resp["user_id"])
	})

	t.Run("Register_Duplicate", func(t *testing.T) {
		payload := map[string]string{"username": username, "password": password}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("Login_Success", func(t *testing.T) {
		payload := map[string]string{"username": username, "password": password}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp, "access_token")
		assert.Contains(t, resp, "refresh_token")
		assert.Contains(t, resp, "expires_in")

		accessToken = resp["access_token"].(string)
		refreshToken = resp["refresh_token"].(string)
	})

	t.Run("Login_WrongPassword", func(t *testing.T) {
		payload := map[string]string{"username": username, "password": "wrongpass"}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RefreshToken_Success", func(t *testing.T) {
		payload := map[string]string{"refresh_token": refreshToken}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp, "access_token")
		assert.Contains(t, resp, "refresh_token")
	})

	t.Run("RefreshToken_Invalid", func(t *testing.T) {
		payload := map[string]string{"refresh_token": "invalid-token"}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RefreshToken_UsingAccessToken", func(t *testing.T) {
		payload := map[string]string{"refresh_token": accessToken}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestAuth_ValidationErrors tests input validation
func TestAuth_ValidationErrors(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup(t)

	router := handler.SetupHandler(env.DB, env.RabbitConn, env.RedisClient, env.Config)

	t.Run("Register_ShortUsername", func(t *testing.T) {
		payload := map[string]string{"username": "ab", "password": "password123"}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Register_ShortPassword", func(t *testing.T) {
		payload := map[string]string{"username": "validuser", "password": "12345"}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Register_MissingFields", func(t *testing.T) {
		payload := map[string]string{}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
