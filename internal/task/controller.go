package task

import (
	"database/sql"
	"net/http"
	"strconv"
	"task_handler/internal/auth"
	"task_handler/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
)

type TaskController struct {
	service TaskServiceInterface
}

func NewTaskController(service TaskServiceInterface) *TaskController {
	return &TaskController{
		service: service,
	}
}

// SetupRoutes setup task routes with JWT protection
func (a *TaskController) SetupRoutes(r *gin.Engine, db *sql.DB, conn *amqp.Connection, rdb *redis.Client, cfg *config.Config) {
	repo := NewTaskRepository()
	service := NewTaskService(repo, db, conn, rdb)
	controller := NewTaskController(service)

	// Protected routes - require JWT authentication
	api := r.Group("/api/v1")
	api.Use(auth.AuthMiddleware(cfg.JWT.Secret)) // Apply middleware to all routes in this group
	{
		// Create task
		api.POST("/tasks", controller.CreateTask)

		// Get task status
		api.GET("/tasks/:id", controller.GetTask)

		// Get tasks by user
		api.GET("/users/:user_id/tasks", controller.GetTasksByUser)
	}
}

// CreateTask handles task creation
func (tc *TaskController) CreateTask(c *gin.Context) {
	var req struct {
		TaskType string `json:"task_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract userID from JWT context
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	task := &Task{
		UserID:   userID,
		TaskType: req.TaskType,
		Status:   "PENDING",
	}

	if err := tc.service.CreateTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"task_id": task.ID,
		"status":  task.Status,
		"message": "Task created successfully",
	})
}

// GetTask handles getting task by ID
func (tc *TaskController) GetTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := tc.service.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            task.ID,
		"user_id":       task.UserID,
		"task_type":     task.TaskType,
		"status":        task.Status,
		"result_file":   task.ResultFile,
		"error_message": task.ErrorMessage,
		"created_at":    task.CreatedAt,
		"updated_at":    task.UpdatedAt,
	})
}

// GetTasksByUser handles getting all tasks for a user
func (tc *TaskController) GetTasksByUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	tasks, err := tc.service.GetTasks(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"count": len(tasks),
	})
}
