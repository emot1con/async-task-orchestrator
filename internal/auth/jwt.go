package auth

import (
	"errors"
	"fmt"
	"strings"
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
func RefreshAccessToken(refreshTokenString string, secret string) (string, error) {
	// Validate refresh token
	claims, err := ValidateToken(refreshTokenString, secret)
	if err != nil {
		return "", err
	}

	// Check if it's a refresh token
	if claims.Type != RefreshToken {
		return "", errors.New("not a refresh token")
	}

	// Generate new access token
	return generateToken(claims.UserID, AccessToken, 15*time.Minute, secret)
}

// AuthMiddleware validates JWT and extracts userID
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(401, gin.H{"error": "Invalid authorization format. Use: Bearer <token>"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := ValidateToken(tokenString, secret)
		if err != nil {
			if errors.Is(err, ErrExpiredToken) {
				c.JSON(401, gin.H{"error": "Token expired"})
			} else {
				c.JSON(401, gin.H{"error": "Invalid token"})
			}
			c.Abort()
			return
		}

		// Check if it's an access token
		if claims.Type != AccessToken {
			c.JSON(401, gin.H{"error": "Invalid token type"})
			c.Abort()
			return
		}

		// Set userID in context
		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
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
