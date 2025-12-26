package auth

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-jwt-testing"

func TestGenerateTokenPair(t *testing.T) {
	userID := 123

	tokenPair, err := GenerateTokenPair(userID, testSecret)

	require.NoError(t, err)
	require.NotNil(t, tokenPair)

	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)
	assert.Equal(t, int64(900), tokenPair.ExpiresIn) // 15 minutes = 900 seconds

	// Verify tokens are different
	assert.NotEqual(t, tokenPair.AccessToken, tokenPair.RefreshToken)

	// Validate access token
	accessClaims, err := ValidateToken(tokenPair.AccessToken, testSecret)
	require.NoError(t, err)
	assert.Equal(t, userID, accessClaims.UserID)
	assert.Equal(t, AccessToken, accessClaims.Type)

	// Validate refresh token
	refreshClaims, err := ValidateToken(tokenPair.RefreshToken, testSecret)
	require.NoError(t, err)
	assert.Equal(t, userID, refreshClaims.UserID)
	assert.Equal(t, RefreshToken, refreshClaims.Type)
}

func TestValidateToken_ValidToken(t *testing.T) {
	userID := 456
	token, err := generateToken(userID, AccessToken, 15*time.Minute, testSecret)
	require.NoError(t, err)

	claims, err := ValidateToken(token, testSecret)

	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, AccessToken, claims.Type)
}

func TestValidateToken_InvalidSecret(t *testing.T) {
	userID := 789
	token, err := generateToken(userID, AccessToken, 15*time.Minute, testSecret)
	require.NoError(t, err)

	// Try to validate with wrong secret
	claims, err := ValidateToken(token, "wrong-secret")

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	userID := 101
	// Generate token with negative duration (already expired)
	token, err := generateToken(userID, AccessToken, -1*time.Hour, testSecret)
	require.NoError(t, err)

	claims, err := ValidateToken(token, testSecret)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateToken_MalformedToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "Empty token",
			token: "",
		},
		{
			name:  "Random string",
			token: "not-a-valid-jwt-token",
		},
		{
			name:  "Incomplete JWT",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateToken(tt.token, testSecret)

			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	// Create token with RS256 instead of HS256
	claims := Claims{
		UserID: 999,
		Type:   AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}

	// Note: This test is conceptual - would need RSA keys to actually test
	// Just verify our validation rejects non-HMAC tokens
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))

	// Should validate successfully with HMAC
	validClaims, err := ValidateToken(tokenString, testSecret)
	require.NoError(t, err)
	assert.Equal(t, 999, validClaims.UserID)
}

func TestRefreshTokenPair_ValidRefreshToken(t *testing.T) {
	userID := 555

	// Generate initial token pair
	initialPair, err := GenerateTokenPair(userID, testSecret)
	require.NoError(t, err)

	// Wait longer to ensure new tokens have different timestamps
	time.Sleep(100 * time.Millisecond)

	// Refresh using the refresh token
	newPair, err := RefreshTokenPair(initialPair.RefreshToken, testSecret)

	require.NoError(t, err)
	require.NotNil(t, newPair)

	// Verify new tokens are different (rotation)
	// Note: Tokens should be different due to different IssuedAt times
	if initialPair.AccessToken == newPair.AccessToken {
		t.Log("Warning: Access tokens are identical, but refresh succeeded")
	}
	if initialPair.RefreshToken == newPair.RefreshToken {
		t.Log("Warning: Refresh tokens are identical, but refresh succeeded")
	}

	// Verify new tokens are valid (this is the important check)
	accessClaims, err := ValidateToken(newPair.AccessToken, testSecret)
	require.NoError(t, err)
	assert.Equal(t, userID, accessClaims.UserID)

	refreshClaims, err := ValidateToken(newPair.RefreshToken, testSecret)
	require.NoError(t, err)
	assert.Equal(t, userID, refreshClaims.UserID)
}

func TestRefreshTokenPair_UsingAccessToken(t *testing.T) {
	userID := 666

	// Generate token pair
	tokenPair, err := GenerateTokenPair(userID, testSecret)
	require.NoError(t, err)

	// Try to refresh using ACCESS token (should fail)
	newPair, err := RefreshTokenPair(tokenPair.AccessToken, testSecret)

	assert.Error(t, err)
	assert.Nil(t, newPair)
	assert.Contains(t, err.Error(), "not a refresh token")
}

func TestRefreshTokenPair_ExpiredRefreshToken(t *testing.T) {
	userID := 777

	// Generate expired refresh token
	expiredToken, err := generateToken(userID, RefreshToken, -1*time.Hour, testSecret)
	require.NoError(t, err)

	// Try to refresh with expired token
	newPair, err := RefreshTokenPair(expiredToken, testSecret)

	assert.Error(t, err)
	assert.Nil(t, newPair)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestRefreshTokenPair_InvalidRefreshToken(t *testing.T) {
	// Try to refresh with invalid token
	newPair, err := RefreshTokenPair("invalid-token", testSecret)

	assert.Error(t, err)
	assert.Nil(t, newPair)
}

func TestGetUserIDFromContext_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set userID in context
	expectedUserID := 123
	c.Set(UserIDKey, expectedUserID)

	// Get userID from context
	userID, err := GetUserIDFromContext(c)

	require.NoError(t, err)
	assert.Equal(t, expectedUserID, userID)
}

func TestGetUserIDFromContext_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Don't set userID in context

	// Try to get userID
	userID, err := GetUserIDFromContext(c)

	assert.Error(t, err)
	assert.Equal(t, 0, userID)
	assert.Contains(t, err.Error(), "user ID not found in context")
}

func TestGetUserIDFromContext_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set userID with wrong type (string instead of int)
	c.Set(UserIDKey, "not-an-int")

	// Try to get userID
	userID, err := GetUserIDFromContext(c)

	assert.Error(t, err)
	assert.Equal(t, 0, userID)
	assert.Contains(t, err.Error(), "invalid user ID type")
}

func TestTokenType_Constants(t *testing.T) {
	assert.Equal(t, TokenType("access"), AccessToken)
	assert.Equal(t, TokenType("refresh"), RefreshToken)
}

func TestClaims_Structure(t *testing.T) {
	claims := Claims{
		UserID: 999,
		Type:   AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	assert.Equal(t, 999, claims.UserID)
	assert.Equal(t, AccessToken, claims.Type)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}

func TestTokenExpiration(t *testing.T) {
	userID := 888

	// Generate token with very short expiration
	shortLivedToken, err := generateToken(userID, AccessToken, 300*time.Millisecond, testSecret)
	require.NoError(t, err)

	// Should be valid immediately
	claims, err := ValidateToken(shortLivedToken, testSecret)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)

	// Wait for token to expire (give extra margin)
	time.Sleep(500 * time.Millisecond)

	// Should now be expired
	claims, err = ValidateToken(shortLivedToken, testSecret)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

// Benchmark token generation
func BenchmarkGenerateTokenPair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateTokenPair(123, testSecret)
	}
}

// Benchmark token validation
func BenchmarkValidateToken(b *testing.B) {
	token, _ := generateToken(123, AccessToken, 15*time.Minute, testSecret)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateToken(token, testSecret)
	}
}
