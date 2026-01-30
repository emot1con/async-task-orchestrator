package handler

import (
	"database/sql"
	"errors"
	"task_handler/internal/config"
	"task_handler/internal/domain/task"
	"task_handler/internal/domain/user"
	"task_handler/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rabbitmq/amqp091-go"
)

// SetupHandler initializes all dependencies and routes
func SetupHandler(db *sql.DB, conn *amqp091.Connection, redisClient *redis.Client, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		ctx := c.Request.Context()
		if err := db.PingContext(ctx); err != nil {
			c.JSON(503, gin.H{
				"status": "unhealthy",
				"db":     "down",
			})
			return
		}

		redisStatus := "ok"
		if err := redisClient.Ping(ctx).Err(); err != nil {
			redisStatus = "down"
		}

		if err := checkRabbitMQ(conn); err != nil {
			c.JSON(503, gin.H{
				"status": "unhealthy",
				"db":     "down",
			})
			return
		}

		c.JSON(200, gin.H{
			"status":   "healthy",
			"db":       "ok",
			"redis":    redisStatus,
			"rabbitmq": "ok",
		})
	})

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
	authGroup := r.Group("/task-handler/auth")
	{
		authGroup.POST("/register", userCtrl.Register)
		authGroup.POST("/login", userCtrl.Login)
		authGroup.POST("/refresh", userCtrl.RefreshToken)
	}

	// Protected routes - API v1
	api := r.Group("/task-handler/api/v1")
	api.Use(middleware.AuthMiddleware(jwtSecret))
	api.Use(middleware.RateLimiterMiddleware(redisClient, middleware.DefaultRateLimiterConfig()))
	{
		// Task endpoints
		api.POST("/tasks", taskCtrl.CreateTask)
		api.GET("/tasks/:id", taskCtrl.GetTask)
		api.GET("/users/tasks", taskCtrl.GetTasksByUser)
	}
}

func checkRabbitMQ(conn *amqp091.Connection) error {
	if conn == nil || conn.IsClosed() {
		return errors.New("rabbitmq connection closed")
	}

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return nil
}
