package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"task_handler/internal/cache"
	"task_handler/internal/queue"
	"task_handler/internal/utils"
	"time"

	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type TaskServiceInterface interface {
	CreateTask(task *Task) error
	GetTask(taskID int) (*Task, error)
	GetTasks(userID int) ([]*Task, error)
}

type TaskService struct {
	repo  TaskRepositoryInterface
	conn  *amqp.Connection
	DB    *sql.DB
	cache *cache.TaskCache
}

func NewTaskService(repo TaskRepositoryInterface, db *sql.DB, conn *amqp.Connection, redisClient *redis.Client) TaskServiceInterface {
	return &TaskService{
		repo:  repo,
		DB:    db,
		conn:  conn,
		cache: cache.NewTaskCache(redisClient),
	}
}

func (s *TaskService) CreateTask(task *Task) error {
	if task.UserID == 0 || task.TaskType == "" {
		return fmt.Errorf("invalid task payload")
	}

	if err := utils.WithTransaction(s.DB, func(tx *sql.Tx) error {
		taskID, err := s.repo.Create(tx, task)
		if err != nil {
			return err
		}
		task.ID = taskID
		return nil
	}); err != nil {
		return err
	}

	ch, err := queue.CreateChannel(s.conn)
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Publish(
		"",
		"task_queue",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body: []byte(`{
			"id": ` + fmt.Sprintf("%d", task.ID) + `,
			"user_id": ` + fmt.Sprintf("%d", task.UserID) + `,
			"task_type": ` + fmt.Sprintf("%q", task.TaskType) + `
		}`),
		},
	)
}

func (s *TaskService) GetTask(taskID int) (*Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try cache first
	cacheKey := cache.TaskKey(taskID)
	cachedData, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cachedData != nil {
		var task Task
		if json.Unmarshal(cachedData, &task) == nil {
			logrus.Info("cache hit for task ", taskID)
			return &task, nil
		}
	}

	// Cache miss, get from DB
	task, err := s.repo.GetByID(s.DB, taskID)
	if err != nil {
		return nil, err
	}

	logrus.Info("cache miss for task ", taskID)
	// Set cache (ignore error, cache miss is not critical)
	if err := s.cache.Set(ctx, cacheKey, task); err != nil {
		logrus.WithError(err).Warn("Failed to set cache for task")
	}

	return task, nil
}

func (s *TaskService) GetTasks(userID int) ([]*Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try cache first
	cacheKey := cache.UserTasksKey(userID)
	cachedData, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cachedData != nil {
		var tasks []*Task
		if json.Unmarshal(cachedData, &tasks) == nil {
			logrus.Infof("cache hit for user %d tasks", userID)
			return tasks, nil
		}
	}
	logrus.Infof("cache miss for user %d tasks", userID)

	// Cache miss, get from DB
	tasks, err := s.repo.GetByUserID(s.DB, userID)
	if err != nil {
		return nil, err
	}

	// Set cache (ignore error, cache miss is not critical)
	if err := s.cache.Set(ctx, cacheKey, tasks); err != nil {
		logrus.WithError(err).Warn("Failed to set cache for user tasks")
	}

	return tasks, nil
}
