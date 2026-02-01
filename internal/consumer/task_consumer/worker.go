package task_consumer

import (
	"database/sql"
	"encoding/json"
	"task_handler/internal/domain/task"
	"task_handler/internal/events"
	"task_handler/internal/utils"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func StartWorker(conn *amqp.Connection, db *sql.DB, repo task.TaskRepositoryInterface, id int) {
	ch, err := conn.Channel()
	if err != nil {
		logrus.Fatalf("Worker %d failed to open channel: %v", id, err)
	}
	defer ch.Close()

	if err := ch.Qos(1, 0, false); err != nil {
		logrus.Fatalf("Worker %d failed to set QoS: %v", id, err)
	}

	msgs, err := ch.Consume(
		"task_queue",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logrus.Fatalf("Worker %d failed to start consuming messages: %v", id, err)
		return
	}

	logrus.Infof("Worker %d started", id)

	for msg := range msgs {
		var event events.TaskCreatedEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			logrus.Error("invalid payload")
			if err := msg.Nack(false, false); err != nil {
				logrus.WithError(err).Warn("Failed to nack message")
			}
			continue
		}

		task := new(task.TaskPayload)
		task.TaskID = event.Data.TaskID
		task.UserID = event.Data.UserID
		task.TaskType = event.Data.TaskType

		eventID := event.EventID
		// correlationID := event.Metadata.CorrelationID

		currentTask, err := repo.GetByID(db, task.TaskID)
		if err != nil {
			logrus.WithError(err).Error("Failed to get task status")
			msg.Nack(false, true) // requeue
			continue
		}
		if err := utils.CheckTaskStatus(currentTask.Status, eventID, msg, currentTask.UpdatedAt); err != nil {
			logrus.Error(err)
			continue
		}

		retryCount := int32(0)
		if msg.Headers != nil {
			if count, ok := msg.Headers["x-retry-count"].(int32); ok {
				retryCount = count
			}
		}

		logrus.Infof(
			"Worker %d processing task=%s for user=%d (retry: %d)",
			id,
			task.TaskType,
			task.UserID,
			retryCount,
		)

		// Transaction 1: Mark as PROCESSING (commit immediately)
		if err := utils.WithTransaction(db, func(tx *sql.Tx) error {
			logrus.Infof("Worker %d: Marking task %d as PROCESSING", id, task.TaskID)
			return repo.MarkProcessing(tx, task.TaskID)
		}); err != nil {
			logrus.WithError(err).Error("Failed to mark task as processing")
			if err := msg.Nack(false, true); err != nil {
				logrus.WithError(err).Warn("Failed to nack message for requeue")
			}
			continue
		}

		taskErr := handleTask(task, id)

		// Transaction 2: Mark as SUCCESS or FAILED
		if err := utils.WithTransaction(db, func(tx *sql.Tx) error {
			if taskErr != nil {
				logrus.WithError(taskErr).Error("task failed")
				return repo.MarkFailed(tx, task.TaskID, taskErr.Error())
			}
			return repo.MarkSuccess(tx, task.TaskID, "result.txt")
		}); err != nil {
			logrus.WithError(err).Error("Failed to update task status")

			// Check retry logic
			if retryCount >= 3 {
				if err := utils.WithTransaction(db, func(tx *sql.Tx) error {
					return repo.MarkFailed(tx, task.TaskID, "max retries reached")
				}); err != nil {
					logrus.WithError(err).Error("Failed to mark task as failed after max retries")
				}
				if err := msg.Nack(false, false); err != nil {
					logrus.WithError(err).Warn("Failed to nack message after max retries")
				}
				continue
			}

			logrus.Infof("Worker %d: Task failed, requeuing (retry %d/3)", id, retryCount+1)

			if err := utils.RepublishWithRetry(ch, &msg, retryCount+1); err != nil {
				logrus.WithError(err).Error("Failed to republish message")
				if err := msg.Nack(false, false); err != nil {
					logrus.WithError(err).Warn("Failed to nack message after republish error")
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				logrus.WithError(err).Warn("Failed to ack message after republish")
			}
			continue
		}

		if err := msg.Ack(false); err != nil {
			logrus.WithError(err).Warn("Failed to ack message")
		}
	}
}
