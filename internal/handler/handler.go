package handler

import (
	"database/sql"
	"task_handler/internal/config"
	"task_handler/internal/middleware"
	"task_handler/internal/task"
	"task_handler/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rabbitmq/amqp091-go"
)

// SetupHandler initializes all dependencies and routes
func SetupHandler(db *sql.DB, conn *amqp091.Connection, redisClient *redis.Client, cfg *config.Config) *gin.Engine {

	r := gin.Default()

	// Initialize repositories
	userRepo := user.NewUserRepository()
	taskRepo := task.NewTaskRepository()

	// Initialize services
	userService := user.NewUserService(userRepo, db)
	taskService := task.NewTaskService(taskRepo, db, conn, redisClient)

	// Initialize controllers
	userController := user.NewUserController(userService, cfg.JWT.Secret)
	taskController := task.NewTaskController(taskService)

	// Setup routes
	setupRoutes(r, userController, taskController, redisClient, cfg.JWT.Secret)

	return r
}

// setupRoutes configures all application routes
func setupRoutes(r *gin.Engine, userCtrl *user.UserController, taskCtrl *task.TaskController, redisClient *redis.Client, jwtSecret string) {

	// Public routes - Authentication
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", userCtrl.Register)
		authGroup.POST("/login", userCtrl.Login)
		authGroup.POST("/refresh", userCtrl.RefreshToken)
	}

	// Protected routes - API v1
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(jwtSecret))                                                 // Apply JWT middleware to all routes in this group
	api.Use(middleware.RateLimiterMiddleware(redisClient, middleware.DefaultRateLimiterConfig())) // Apply rate limiter
	{
		// Task endpoints
		api.POST("/tasks", taskCtrl.CreateTask)
		api.GET("/tasks/:id", taskCtrl.GetTask)
		api.GET("/users/:user_id/tasks", taskCtrl.GetTasksByUser)
	}
}
