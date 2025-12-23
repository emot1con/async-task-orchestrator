package middleware

import (
	"errors"
	"strings"
	"task_handler/internal/auth"

	"github.com/gin-gonic/gin"
)

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
		claims, err := auth.ValidateToken(tokenString, secret)
		if err != nil {
			if errors.Is(err, auth.ErrExpiredToken) {
				c.JSON(401, gin.H{"error": "Token expired"})
			} else {
				c.JSON(401, gin.H{"error": "Invalid token"})
			}
			c.Abort()
			return
		}

		// Check if it's an access token
		if claims.Type != auth.AccessToken {
			c.JSON(401, gin.H{"error": "Invalid token type"})
			c.Abort()
			return
		}

		// Set userID in context
		c.Set(auth.UserIDKey, claims.UserID)
		c.Next()
	}
}
