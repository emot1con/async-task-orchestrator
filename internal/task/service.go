package task

import (
	"database/sql"
	"fmt"
	"task_handler/internal/queue"
	"task_handler/internal/utils"

	amqp "github.com/rabbitmq/amqp091-go"
)

type TaskService struct {
	repo *TaskRepository
	conn *amqp.Connection
	DB   *sql.DB
}

func NewTaskService(repo *TaskRepository, db *sql.DB, conn *amqp.Connection) *TaskService {
	return &TaskService{
		repo: repo,
		DB:   db,
		conn: conn,
	}
}

func (s *TaskService) CreateTask(task *Task) error {
	if task.UserID == 0 || task.TaskType == "" {
		return fmt.Errorf("invalid task payload")
	}

	if err := utils.WithTransaction(s.DB, func(tx *sql.Tx) error {
		return s.repo.Create(tx, task)
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
			ContentType: "text/plain",
			Body:        []byte(task.TaskType),
		},
	)
}
