package task

import (
	"database/sql"
	"fmt"
	"task_handler/internal/queue"
	"task_handler/internal/utils"

	amqp "github.com/rabbitmq/amqp091-go"
)

type TaskService struct {
	repo TaskRepositoryInterface
	conn *amqp.Connection
	DB   *sql.DB
}

func NewTaskService(repo TaskRepositoryInterface, db *sql.DB, conn *amqp.Connection) *TaskService {
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
