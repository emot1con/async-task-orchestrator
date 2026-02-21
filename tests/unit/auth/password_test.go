package auth_test

import (
	"task_handler/internal/auth"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePasswordHash_Success(t *testing.T) {
	hash, err := auth.GeneratePasswordHash("mypassword123")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "mypassword123", hash)
}

func TestGeneratePasswordHash_DifferentHashesForSamePassword(t *testing.T) {
	// bcrypt uses salt so same password should produce different hashes
	hash1, err1 := auth.GeneratePasswordHash("samepassword")
	hash2, err2 := auth.GeneratePasswordHash("samepassword")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, hash1, hash2)
}

func TestComparePasswordHash_CorrectPassword(t *testing.T) {
	password := "secret123"
	hash, err := auth.GeneratePasswordHash(password)
	require.NoError(t, err)

	err = auth.ComparePasswordHash([]byte(hash), password)

	assert.NoError(t, err)
}

func TestComparePasswordHash_WrongPassword(t *testing.T) {
	hash, err := auth.GeneratePasswordHash("correctpassword")
	require.NoError(t, err)

	err = auth.ComparePasswordHash([]byte(hash), "wrongpassword")

	assert.Error(t, err)
}

func TestComparePasswordHash_EmptyPassword(t *testing.T) {
	hash, err := auth.GeneratePasswordHash("somepassword")
	require.NoError(t, err)

	err = auth.ComparePasswordHash([]byte(hash), "")

	assert.Error(t, err)
}

func TestGeneratePasswordHash_LongPassword(t *testing.T) {
	longPassword := "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6A7B8C9D0"
	hash, err := auth.GeneratePasswordHash(longPassword)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestGeneratePasswordHash_EmptyPassword(t *testing.T) {
	hash, err := auth.GeneratePasswordHash("")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}
