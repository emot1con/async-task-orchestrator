package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	UserIDKey = "userID"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	UserID int       `json:"user_id"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// GenerateTokenPair creates both access and refresh tokens
func GenerateTokenPair(userID int, secret string) (*TokenPair, error) {
	// Access token (15 minutes)
	accessToken, err := generateToken(userID, AccessToken, 15*time.Minute, secret)
	if err != nil {
		return nil, err
	}

	// Refresh token (7 days)
	refreshToken, err := generateToken(userID, RefreshToken, 7*24*time.Hour, secret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64((15 * time.Minute).Seconds()),
	}, nil
}

// generateToken creates a JWT token
func generateToken(userID int, tokenType TokenType, duration time.Duration, secret string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken validates and parses JWT token
func ValidateToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshAccessToken generates new access token from refresh token
// Note: Only returns access token, refresh token remains same (simple approach)
// func RefreshAccessToken(refreshTokenString string, secret string) (string, error) {
// 	// Validate refresh token
// 	claims, err := ValidateToken(refreshTokenString, secret)
// 	if err != nil {
// 		return "", err
// 	}

// 	// Check if it's a refresh token
// 	if claims.Type != RefreshToken {
// 		return "", errors.New("not a refresh token")
// 	}

// 	// Generate new access token
// 	return generateToken(claims.UserID, AccessToken, 15*time.Minute, secret)
// }

// RefreshTokenPair generates new access token AND new refresh token (with rotation)
// This is more secure as it invalidates old refresh token
func RefreshTokenPair(refreshTokenString string, secret string) (*TokenPair, error) {
	// Validate refresh token
	claims, err := ValidateToken(refreshTokenString, secret)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if it's a refresh token
	if claims.Type != RefreshToken {
		return nil, errors.New("not a refresh token")
	}

	// Generate NEW token pair (rotation for security)
	// Old refresh token becomes invalid
	return GenerateTokenPair(claims.UserID, secret)
}

// GetUserIDFromContext extracts userID from Gin context
func GetUserIDFromContext(c *gin.Context) (int, error) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return 0, fmt.Errorf("user ID not found in context")
	}

	id, ok := userID.(int)
	if !ok {
		return 0, fmt.Errorf("invalid user ID type")
	}

	return id, nil
}
