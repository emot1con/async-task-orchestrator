package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"task_handler/internal/auth"
	"task_handler/internal/middleware"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const jwtSecret = "middleware-test-secret"

func setupAuthRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.AuthMiddleware(jwtSecret))
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get(auth.UserIDKey)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	tokens, err := auth.GenerateTokenPair(7, jwtSecret)
	if err != nil {
		t.Fatal(err)
	}

	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidFormat_NoBearerPrefix(t *testing.T) {
	tokens, _ := auth.GenerateTokenPair(1, jwtSecret)

	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", tokens.AccessToken) // missing "Bearer "
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.value")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	tokens, _ := auth.GenerateTokenPair(1, "other-secret")

	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_RefreshTokenRejected(t *testing.T) {
	// Refresh token should NOT pass access-only middleware
	tokens, _ := auth.GenerateTokenPair(1, jwtSecret)

	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.RefreshToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_SetsUserIDInContext(t *testing.T) {
	tokens, _ := auth.GenerateTokenPair(99, jwtSecret)

	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "99")
}
