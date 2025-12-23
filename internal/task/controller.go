package task

import (
	"net/http"
	"strconv"
	"task_handler/internal/auth"
	"task_handler/internal/observability"

	"github.com/gin-gonic/gin"
)

type TaskController struct {
	service TaskServiceInterface
}

func NewTaskController(service TaskServiceInterface) *TaskController {
	return &TaskController{
		service: service,
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

	// Track task creation metric
	observability.GlobalMetrics.TasksCreatedTotal.WithLabelValues(task.TaskType).Inc()

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
