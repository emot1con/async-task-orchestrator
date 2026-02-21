package auth_test

import (
	"task_handler/internal/auth"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-unit-tests"

func TestGenerateTokenPair_Success(t *testing.T) {
	tokens, err := auth.GenerateTokenPair(1, testSecret)

	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, int64(900), tokens.ExpiresIn) // 15 minutes = 900 seconds
	assert.NotEqual(t, tokens.AccessToken, tokens.RefreshToken)
}

func TestGenerateTokenPair_DifferentUsers(t *testing.T) {
	tokens1, err1 := auth.GenerateTokenPair(1, testSecret)
	tokens2, err2 := auth.GenerateTokenPair(2, testSecret)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, tokens1.AccessToken, tokens2.AccessToken)
	assert.NotEqual(t, tokens1.RefreshToken, tokens2.RefreshToken)
}

func TestValidateToken_ValidAccessToken(t *testing.T) {
	tokens, err := auth.GenerateTokenPair(42, testSecret)
	require.NoError(t, err)

	claims, err := auth.ValidateToken(tokens.AccessToken, testSecret)

	require.NoError(t, err)
	assert.Equal(t, 42, claims.UserID)
	assert.Equal(t, auth.AccessToken, claims.Type)
}

func TestValidateToken_ValidRefreshToken(t *testing.T) {
	tokens, err := auth.GenerateTokenPair(42, testSecret)
	require.NoError(t, err)

	claims, err := auth.ValidateToken(tokens.RefreshToken, testSecret)

	require.NoError(t, err)
	assert.Equal(t, 42, claims.UserID)
	assert.Equal(t, auth.RefreshToken, claims.Type)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	_, err := auth.ValidateToken("invalid.token.string", testSecret)

	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	tokens, err := auth.GenerateTokenPair(1, testSecret)
	require.NoError(t, err)

	_, err = auth.ValidateToken(tokens.AccessToken, "wrong-secret")

	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Build an already-expired token
	claims := auth.Claims{
		UserID: 1,
		Type:   auth.AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = auth.ValidateToken(tokenString, testSecret)

	assert.ErrorIs(t, err, auth.ErrExpiredToken)
}

func TestValidateToken_EmptyToken(t *testing.T) {
	_, err := auth.ValidateToken("", testSecret)

	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestRefreshTokenPair_Success(t *testing.T) {
	original, err := auth.GenerateTokenPair(10, testSecret)
	require.NoError(t, err)

	// Small sleep so new tokens have a different IssuedAt timestamp
	time.Sleep(1 * time.Second)

	newPair, err := auth.RefreshTokenPair(original.RefreshToken, testSecret)

	require.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)
	assert.NotEmpty(t, newPair.RefreshToken)
	assert.Equal(t, int64(900), newPair.ExpiresIn)
	// New tokens should differ from the originals (token rotation)
	assert.NotEqual(t, original.AccessToken, newPair.AccessToken)
	assert.NotEqual(t, original.RefreshToken, newPair.RefreshToken)
}

func TestRefreshTokenPair_WithAccessToken_Fails(t *testing.T) {
	tokens, err := auth.GenerateTokenPair(1, testSecret)
	require.NoError(t, err)

	// Passing access token to RefreshTokenPair should fail
	_, err = auth.RefreshTokenPair(tokens.AccessToken, testSecret)

	assert.Error(t, err)
}

func TestRefreshTokenPair_InvalidToken(t *testing.T) {
	_, err := auth.RefreshTokenPair("totally-invalid", testSecret)

	assert.Error(t, err)
}

func TestRefreshTokenPair_ExpiredRefreshToken(t *testing.T) {
	claims := auth.Claims{
		UserID: 1,
		Type:   auth.RefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = auth.RefreshTokenPair(tokenString, testSecret)

	assert.Error(t, err)
}
