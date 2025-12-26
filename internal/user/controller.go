package user

import (
	"fmt"
	"net/http"
	"strings"
	"task_handler/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type UserController struct {
	userService UserServiceInterface
	jwtSecret   string
}

func NewUserController(userService UserServiceInterface, jwtSecret string) *UserController {
	return &UserController{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

// Register handles user registration
func (a *UserController) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=50"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create user
	userID, err := a.userService.CreateUser(req.Username, req.Password)
	if err != nil {
		// Check for service layer duplicate user error
		if err.Error() == "username already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		// Check if username already exists using PostgreSQL error code
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		// Fallback: check for common duplicate key error messages
		if strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "UNIQUE constraint") ||
			strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create user: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user_id": userID,
	})
}

// Login handles user login and returns JWT tokens
func (a *UserController) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate credentials
	tokens, err := a.userService.LoginUser(req.Username, req.Password, a.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// RefreshToken handles refresh token requests with token rotation
func (a *UserController) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	tokenPair, err := auth.RefreshTokenPair(req.RefreshToken, a.jwtSecret)
	if err != nil {
		if err == auth.ErrExpiredToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expired, please login again"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken, // NEW refresh token
		"token_type":    "Bearer",
		"expires_in":    tokenPair.ExpiresIn,
	})
}
