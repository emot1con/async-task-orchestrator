package user

import (
	"net/http"
	"task_handler/internal/auth"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService UserServiceInterface
	jwtSecret   string
}

func NewUserController(userService UserServiceInterface) *UserController {
	return &UserController{
		userService: userService,
	}
}

// SetupRoutes setup auth routes (register, login, refresh)
func (a *UserController) SetupRoutes(r *gin.Engine, userService UserServiceInterface) {
	controller := NewUserController(userService)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", controller.Register)
		authGroup.POST("/login", controller.Login)
		authGroup.POST("/refresh", controller.RefreshToken)
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
		// Check if username already exists
		if err.Error() == "UNIQUE constraint failed" ||
			err.Error() == "duplicate key value violates unique constraint" ||
			err.Error() == "pq: duplicate key value violates unique constraint \"users_username_key\"" {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
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

// RefreshToken handles refresh token requests
func (a *UserController) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Generate new access token from refresh token
	accessToken, err := auth.RefreshAccessToken(req.RefreshToken, a.jwtSecret)
	if err != nil {
		if err == auth.ErrExpiredToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expired"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"expires_in":   900, // 15 minutes in seconds
	})
}
