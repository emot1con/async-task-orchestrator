package user_controller_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"task_handler/internal/auth"
	"task_handler/internal/domain/user"
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

const testJWTSecret = "test-secret"

// ---- Mock UserService ----

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(username, password string) (int, error) {
	args := m.Called(username, password)
	return args.Int(0), args.Error(1)
}

func (m *MockUserService) LoginUser(username, password, jwtSecret string) (*auth.TokenPair, error) {
	args := m.Called(username, password, jwtSecret)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}

func (m *MockUserService) GetUserByID(id int) (*user.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUserByUsername(username string) (*user.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

// ---- Helper ----

func setupUserRouter(svc user.UserServiceInterface) *gin.Engine {
	r := gin.New()
	ctrl := user.NewUserController(svc, testJWTSecret)
	r.POST("/register", ctrl.Register)
	r.POST("/login", ctrl.Login)
	r.POST("/refresh", ctrl.RefreshToken)
	return r
}

// ---- Register Tests ----

func TestRegister_Success(t *testing.T) {
	svc := new(MockUserService)
	svc.On("CreateUser", "alice", "password123").Return(42, nil)

	r := setupUserRouter(svc)
	body := `{"username":"alice","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "User created successfully", resp["message"])
	assert.Equal(t, float64(42), resp["user_id"])
	svc.AssertExpectations(t)
}

func TestRegister_DuplicateUsername(t *testing.T) {
	svc := new(MockUserService)
	svc.On("CreateUser", "alice", "pass").Return(0, errors.New("username already exists"))

	r := setupUserRouter(svc)
	body := `{"username":"alice","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRegister_EmptyBody(t *testing.T) {
	// UserPayload has no binding:"required" tags, so empty JSON passes ShouldBindJSON.
	// The controller calls CreateUser with empty strings.
	svc := new(MockUserService)
	svc.On("CreateUser", "", "").Return(0, errors.New("username already exists"))

	r := setupUserRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 409 because service returns "username already exists"
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRegister_ServiceError(t *testing.T) {
	svc := new(MockUserService)
	svc.On("CreateUser", "bob", "pass").Return(0, errors.New("database error"))

	r := setupUserRouter(svc)
	body := `{"username":"bob","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- Login Tests ----

func TestLogin_Success(t *testing.T) {
	svc := new(MockUserService)
	tokens := &auth.TokenPair{
		AccessToken:  "access.token.here",
		RefreshToken: "refresh.token.here",
		ExpiresIn:    900,
	}
	svc.On("LoginUser", "alice", "password123", testJWTSecret).Return(tokens, nil)

	r := setupUserRouter(svc)
	body := `{"username":"alice","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "access.token.here", resp["access_token"])
	assert.Equal(t, "refresh.token.here", resp["refresh_token"])
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc := new(MockUserService)
	svc.On("LoginUser", "alice", "wrongpass", testJWTSecret).Return(nil, errors.New("invalid credentials"))

	r := setupUserRouter(svc)
	body := `{"username":"alice","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_EmptyBody(t *testing.T) {
	// Empty JSON passes ShouldBindJSON (no binding:"required" on UserPayload).
	// The controller calls LoginUser with empty strings.
	svc := new(MockUserService)
	svc.On("LoginUser", "", "", testJWTSecret).Return(nil, errors.New("invalid credentials"))

	r := setupUserRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---- RefreshToken Tests ----

func TestRefreshToken_Success(t *testing.T) {
	// Generate a real refresh token
	tokenPair, err := auth.GenerateTokenPair(1, testJWTSecret)
	require.NoError(t, err)

	svc := new(MockUserService)
	r := setupUserRouter(svc)

	body, _ := json.Marshal(user.RefreshTokenRequest{RefreshToken: tokenPair.RefreshToken})
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc := new(MockUserService)
	r := setupUserRouter(svc)

	body := `{"refresh_token":"invalid.token"}`
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefreshToken_MissingToken(t *testing.T) {
	svc := new(MockUserService)
	r := setupUserRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserModel(t *testing.T) {
	now := time.Now()
	u := &user.User{
		ID:        1,
		Username:  "alice",
		Password:  "hashedpassword",
		CreatedAt: now,
	}

	b, err := json.Marshal(u)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &parsed))

	// Password should NOT be in JSON output
	_, hasPassword := parsed["password"]
	assert.False(t, hasPassword, "password should not be serialized to JSON")
	assert.Equal(t, float64(1), parsed["id"])
	assert.Equal(t, "alice", parsed["username"])
}
