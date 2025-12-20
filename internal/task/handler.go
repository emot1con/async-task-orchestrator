package task

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
)

type CreateTaskRequest struct {
	UserID   int    `json:"user_id" binding:"required"`
	TaskType string `json:"task_type" binding:"required"`
}

// SetupRoutes setup task routes dengan handler langsung
func SetupRoutes(r *gin.Engine, db *sql.DB, conn *amqp.Connection, rdb *redis.Client) {
	repo := NewTaskRepository()
	service := NewTaskService(repo, db, conn, rdb)

	api := r.Group("/api/v1")
	{
		// Create task
		api.POST("/tasks", func(c *gin.Context) {
			var req CreateTaskRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			task := &Task{
				UserID:   req.UserID,
				TaskType: req.TaskType,
				Status:   "PENDING",
			}

			if err := service.CreateTask(task); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"task_id": task.ID,
				"status":  task.Status,
				"message": "Task created successfully",
			})
		})

		// Get task status
		api.GET("/tasks/:id", func(c *gin.Context) {
			id, err := strconv.Atoi(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
				return
			}

			task, err := service.GetTask(id)
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
		})

		// Get tasks by user
		api.GET("/users/:user_id/tasks", func(c *gin.Context) {
			userID, err := strconv.Atoi(c.Param("user_id"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
				return
			}

			tasks, err := service.GetTasks(userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tasks"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"tasks": tasks, "count": len(tasks)})
		})
	}
}
