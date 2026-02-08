package router

import (
	"task_handler/internal/domain/task"
	"task_handler/internal/domain/user"
	"task_handler/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// setupRoutes configures all application routes
func SetupRoutes(r *gin.Engine, userCtrl *user.UserController, taskCtrl *task.TaskController, redisClient *redis.Client, jwtSecret string) {
	r.Use(gin.Recovery())

	// Public routes - Authentication
	authGroup := r.Group("/task-handler/auth")
	authGroup.Use(middleware.LoggerMiddleware())
	{
		authGroup.POST("/register", userCtrl.Register)
		authGroup.POST("/login", userCtrl.Login)
		authGroup.POST("/refresh", userCtrl.RefreshToken)
	}

	// Protected routes - API v1
	api := r.Group("/task-handler/api/v1")
	api.Use(middleware.LoggerMiddleware())
	api.Use(middleware.AuthMiddleware(jwtSecret))
	api.Use(middleware.RateLimiterMiddleware(redisClient, middleware.DefaultRateLimiterConfig()))
	{
		// Task endpoints
		api.POST("/tasks", taskCtrl.CreateTask)
		api.GET("/tasks/:id", taskCtrl.GetTask)
		api.GET("/users/tasks", taskCtrl.GetTasksByUser)
	}
}
