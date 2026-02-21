package integration

import (
	"task_handler/internal/auth"
	"task_handler/internal/domain/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_UserRepository_CreateAndGetByUsername(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	repo := user.NewUserRepository()

	tx, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	hashedPwd, err := auth.GeneratePasswordHash("testpassword")
	require.NoError(t, err)

	u := &user.User{
		Username: "integration_user",
		Password: hashedPwd,
	}

	id, err := repo.Create(tx, u)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	require.NoError(t, tx.Commit())

	// Get by username
	found, err := repo.GetByUsername(testEnv.DB, "integration_user")
	require.NoError(t, err)
	assert.Equal(t, "integration_user", found.Username)
	assert.Equal(t, id, found.ID)
}

func TestIntegration_UserRepository_GetByID(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	repo := user.NewUserRepository()

	tx, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	hashedPwd, _ := auth.GeneratePasswordHash("pass")
	id, err := repo.Create(tx, &user.User{Username: "user_get_by_id", Password: hashedPwd})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	found, err := repo.GetByID(testEnv.DB, id)
	require.NoError(t, err)
	assert.Equal(t, id, found.ID)
	assert.Equal(t, "user_get_by_id", found.Username)
}

func TestIntegration_UserRepository_GetByID_NotFound(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	repo := user.NewUserRepository()

	_, err := repo.GetByID(testEnv.DB, 99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIntegration_UserRepository_GetByUsername_NotFound(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	repo := user.NewUserRepository()

	_, err := repo.GetByUsername(testEnv.DB, "nonexistent_user")
	assert.Error(t, err)
}

func TestIntegration_UserRepository_DuplicateUsername(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	repo := user.NewUserRepository()

	tx, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	hashedPwd, _ := auth.GeneratePasswordHash("pass")
	_, err = repo.Create(tx, &user.User{Username: "duplicate_user", Password: hashedPwd})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	// Second insert with same username should fail
	tx2, err := testEnv.DB.Begin()
	require.NoError(t, err)
	defer tx2.Rollback()

	_, err = repo.Create(tx2, &user.User{Username: "duplicate_user", Password: hashedPwd})
	assert.Error(t, err)
}

func TestIntegration_UserService_CreateAndLogin(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	repo := user.NewUserRepository()
	svc := user.NewUserService(repo, testEnv.DB)

	id, err := svc.CreateUser("svc_user", "mypassword")
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	tokens, err := svc.LoginUser("svc_user", "mypassword", "testsecret123")
	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestIntegration_UserService_CreateDuplicate(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	repo := user.NewUserRepository()
	svc := user.NewUserService(repo, testEnv.DB)

	_, err := svc.CreateUser("dup_svc_user", "pass")
	require.NoError(t, err)

	_, err = svc.CreateUser("dup_svc_user", "pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestIntegration_UserService_LoginInvalidCredentials(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	repo := user.NewUserRepository()
	svc := user.NewUserService(repo, testEnv.DB)

	_, _ = svc.CreateUser("login_user", "correctpass")

	_, err := svc.LoginUser("login_user", "wrongpass", "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestIntegration_UserService_LoginNonExistentUser(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}
	cleanupTables(t)

	repo := user.NewUserRepository()
	svc := user.NewUserService(repo, testEnv.DB)

	_, err := svc.LoginUser("ghost_user", "pass", "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}
