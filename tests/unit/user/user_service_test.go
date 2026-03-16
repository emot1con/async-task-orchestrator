package user_controller_test

import (
	"database/sql"
	"errors"
	"task_handler/internal/auth"
	"task_handler/internal/domain/user"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testJWTSecretSvc = "svc-test-secret"

// ---- Mock UserRepository ----

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(tx *sql.Tx, u *user.User) (int, error) {
	args := m.Called(tx, u)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepository) GetByID(db *sql.DB, id int) (*user.User, error) {
	args := m.Called(db, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(db *sql.DB, username string) (*user.User, error) {
	args := m.Called(db, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) UpdatePassword(tx *sql.Tx, id int, hashedPassword string) error {
	args := m.Called(tx, id, hashedPassword)
	return args.Error(0)
}

// ---- Helper: DB with sqlmock ----

func setupSvcDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, mock
}

// ---- CreateUser ----

func TestUserService_CreateUser_Success(t *testing.T) {
	db, dbmock := setupSvcDB(t)
	repo := new(MockUserRepository)

	// User does not exist
	repo.On("GetByUsername", db, "alice").Return(nil, errors.New("user not found"))
	// Expect Begin + Insert + Commit
	dbmock.ExpectBegin()
	repo.On("Create", mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(u *user.User) bool {
		return u.Username == "alice" && u.Password != "password123" // hashed
	})).Return(42, nil)
	dbmock.ExpectCommit()

	svc := user.NewUserService(repo, db)
	id, err := svc.CreateUser("alice", "password123")

	require.NoError(t, err)
	assert.Equal(t, 42, id)
	repo.AssertExpectations(t)
}

func TestUserService_CreateUser_AlreadyExists(t *testing.T) {
	db, _ := setupSvcDB(t)
	repo := new(MockUserRepository)

	existing := &user.User{ID: 1, Username: "alice"}
	repo.On("GetByUsername", db, "alice").Return(existing, nil)

	svc := user.NewUserService(repo, db)
	id, err := svc.CreateUser("alice", "anypassword")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username already exists")
	assert.Equal(t, 0, id)
}

func TestUserService_CreateUser_RepoError(t *testing.T) {
	db, dbmock := setupSvcDB(t)
	repo := new(MockUserRepository)

	repo.On("GetByUsername", db, "bob").Return(nil, errors.New("not found"))
	dbmock.ExpectBegin()
	repo.On("Create", mock.AnythingOfType("*sql.Tx"), mock.Anything).Return(0, errors.New("db error"))
	dbmock.ExpectRollback()

	svc := user.NewUserService(repo, db)
	id, err := svc.CreateUser("bob", "pass123")

	assert.Error(t, err)
	assert.Equal(t, 0, id)
}

// ---- LoginUser ----

func TestUserService_LoginUser_Success(t *testing.T) {
	db, _ := setupSvcDB(t)
	repo := new(MockUserRepository)

	hash, err := auth.GeneratePasswordHash("secret123")
	require.NoError(t, err)

	existing := &user.User{ID: 7, Username: "carol", Password: hash, CreatedAt: time.Now()}
	repo.On("GetByUsername", db, "carol").Return(existing, nil)

	svc := user.NewUserService(repo, db)
	tokens, err := svc.LoginUser("carol", "secret123", testJWTSecretSvc)

	require.NoError(t, err)
	require.NotNil(t, tokens)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestUserService_LoginUser_UserNotFound(t *testing.T) {
	db, _ := setupSvcDB(t)
	repo := new(MockUserRepository)

	repo.On("GetByUsername", db, "ghost").Return(nil, errors.New("user not found"))

	svc := user.NewUserService(repo, db)
	tokens, err := svc.LoginUser("ghost", "pass", testJWTSecretSvc)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
	assert.Nil(t, tokens)
}

func TestUserService_LoginUser_WrongPassword(t *testing.T) {
	db, _ := setupSvcDB(t)
	repo := new(MockUserRepository)

	hash, _ := auth.GeneratePasswordHash("correctpassword")
	existing := &user.User{ID: 1, Username: "dave", Password: hash, CreatedAt: time.Now()}
	repo.On("GetByUsername", db, "dave").Return(existing, nil)

	svc := user.NewUserService(repo, db)
	tokens, err := svc.LoginUser("dave", "wrongpassword", testJWTSecretSvc)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
	assert.Nil(t, tokens)
}

// ---- GetUserByID ----

func TestUserService_GetUserByID_Found(t *testing.T) {
	db, _ := setupSvcDB(t)
	repo := new(MockUserRepository)

	u := &user.User{ID: 5, Username: "eve", CreatedAt: time.Now()}
	repo.On("GetByID", db, 5).Return(u, nil)

	svc := user.NewUserService(repo, db)
	result, err := svc.GetUserByID(5)

	require.NoError(t, err)
	assert.Equal(t, 5, result.ID)
	assert.Equal(t, "eve", result.Username)
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	db, _ := setupSvcDB(t)
	repo := new(MockUserRepository)

	repo.On("GetByID", db, 999).Return(nil, errors.New("user not found"))

	svc := user.NewUserService(repo, db)
	result, err := svc.GetUserByID(999)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ---- GetUserByUsername ----

func TestUserService_GetUserByUsername_Found(t *testing.T) {
	db, _ := setupSvcDB(t)
	repo := new(MockUserRepository)

	u := &user.User{ID: 3, Username: "frank", CreatedAt: time.Now()}
	repo.On("GetByUsername", db, "frank").Return(u, nil)

	svc := user.NewUserService(repo, db)
	result, err := svc.GetUserByUsername("frank")

	require.NoError(t, err)
	assert.Equal(t, "frank", result.Username)
}

func TestUserService_GetUserByUsername_NotFound(t *testing.T) {
	db, _ := setupSvcDB(t)
	repo := new(MockUserRepository)

	repo.On("GetByUsername", db, "nobody").Return(nil, errors.New("user not found"))

	svc := user.NewUserService(repo, db)
	result, err := svc.GetUserByUsername("nobody")

	assert.Error(t, err)
	assert.Nil(t, result)
}
