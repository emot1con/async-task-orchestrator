package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"task_handler/internal/task"
	"task_handler/internal/utils"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func republishWithRetry(ch *amqp.Channel, msg *amqp.Delivery, retryCount int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create new headers with incremented retry count
	headers := amqp.Table{}
	if msg.Headers != nil {
		headers = msg.Headers
	}
	headers["x-retry-count"] = retryCount

	return ch.PublishWithContext(
		ctx,
		"",             // exchange
		msg.RoutingKey, // routing key (queue name)
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: msg.ContentType,
			Body:        msg.Body,
			Headers:     headers,
		},
	)
}

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
		var payload task.TaskPayload
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			logrus.Error("invalid payload")
			msg.Nack(false, false)
			continue
		}

		retryCount := int32(0)
		if msg.Headers != nil {
			if count, ok := msg.Headers["x-retry-count"].(int32); ok {
				retryCount = count
			}
		}

		if err := utils.WithTransaction(db, func(tx *sql.Tx) error {
			logrus.Infof(
				"Worker %d processing task=%s for user=%d (retry: %d)",
				id,
				payload.TaskType,
				payload.UserID,
				retryCount,
			)

			if err := repo.MarkProcessing(tx, payload.ID); err != nil {
				return err
			}

			if err := handleTask(&payload, id); err != nil {
				logrus.WithError(err).Error("task failed")
				repo.MarkFailed(tx, payload.ID, err.Error())
				return err
			}
			return repo.MarkSuccess(tx, payload.ID, "result.txt")
		}); err != nil {
			if retryCount >= 3 {
				if err := utils.WithTransaction(db, func(tx *sql.Tx) error {
					return repo.MarkFailed(tx, payload.ID, "max retries reached")
				}); err != nil {
					logrus.WithError(err).Error("Failed to mark task as failed after max retries")
				}
				msg.Nack(false, false)
				continue
			}

			logrus.Infof("Worker %d: Task failed, requeuing (retry %d/3)", id, retryCount+1)

			if err := republishWithRetry(ch, &msg, retryCount+1); err != nil {
				logrus.WithError(err).Error("Failed to republish message")
				msg.Nack(false, false)
				continue
			}

			msg.Ack(false)
			continue
		}
		msg.Ack(false)
	}
}
