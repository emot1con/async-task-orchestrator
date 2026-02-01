package task_consumer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"task_handler/internal/domain/task"
	"task_handler/internal/events"
	"task_handler/internal/queue"
	"task_handler/internal/utils"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func StartWorker(conn *amqp.Connection, db *sql.DB, repo task.TaskRepositoryInterface, workerID int) {
	ch, err := conn.Channel()
	if err != nil {
		logrus.Fatalf("Worker %d failed to open channel: %v", workerID, err)
	}
	defer ch.Close()

	if err := ch.Qos(1, 0, false); err != nil {
		logrus.Fatalf("Worker %d failed to set QoS: %v", workerID, err)
	}

	msgs, err := queue.Consume(ch, "task_queue")
	if err != nil {
		logrus.Fatalf("Worker %d failed to start consuming messages: %v", workerID, err)
		return
	}

	logrus.Infof("Worker %d started", workerID)

	for msg := range msgs {
		processMessage(ch, db, repo, workerID, msg)
	}
}

func processMessage(ch *amqp.Channel, db *sql.DB, repo task.TaskRepositoryInterface, workerID int, msg amqp.Delivery) {
	// Parse and validate event
	taskPayload, event, retryCount, err := parseAndValidateMessage(db, repo, msg)
	if err != nil {
		logrus.WithError(err).Error("Message validation failed")
		return // message already acked/nacked inside
	}

	logrus.Infof("Worker %d processing task=%s for user=%d (retry: %d)", workerID, taskPayload.TaskType, taskPayload.UserID, retryCount)

	// Mark as PROCESSING
	if err := markAsProcessing(db, repo, taskPayload.TaskID, workerID); err != nil {
		logrus.WithError(err).Error("Failed to mark task as processing")
		msg.Nack(false, true)
		return
	}

	// Execute task
	taskErr := handleTask(taskPayload, workerID)

	// Update final status and handle retry if needed
	handleTaskResult(ch, db, repo, msg, taskPayload, event, taskErr, retryCount, workerID)
}

// parseAndValidateMessage parses event and validates task status
func parseAndValidateMessage(db *sql.DB, repo task.TaskRepositoryInterface, msg amqp.Delivery) (*task.TaskPayload, *events.TaskCreatedEvent, int32, error) {
	// Parse event
	var event events.TaskCreatedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		logrus.Error("invalid payload")
		msg.Nack(false, false)
		return nil, nil, 0, fmt.Errorf("invalid JSON payload")
	}

	taskPayload := &task.TaskPayload{
		TaskID:   event.Data.TaskID,
		UserID:   event.Data.UserID,
		TaskType: event.Data.TaskType,
	}

	retryCount := getRetryCount(msg)

	// Validate task status
	currentTask, err := repo.GetByID(db, taskPayload.TaskID)
	if err != nil {
		logrus.WithError(err).Error("Failed to get task status")
		msg.Nack(false, true)
		return nil, nil, 0, fmt.Errorf("failed to get task status")
	}

	if err := checkTaskStatus(currentTask.Status, event.EventID, msg, currentTask.UpdatedAt); err != nil {
		return nil, nil, 0, err
	}

	return taskPayload, &event, retryCount, nil
}

func getRetryCount(msg amqp.Delivery) int32 {
	if msg.Headers != nil {
		if count, ok := msg.Headers["x-retry-count"].(int32); ok {
			return count
		}
	}
	return 0
}

func markAsProcessing(db *sql.DB, repo task.TaskRepositoryInterface, taskID int, workerID int) error {
	return utils.WithTransaction(db, func(tx *sql.Tx) error {
		logrus.Infof("Worker %d: Marking task %d as PROCESSING", workerID, taskID)
		return repo.MarkProcessing(tx, taskID)
	})
}

func handleTaskResult(ch *amqp.Channel, db *sql.DB, repo task.TaskRepositoryInterface, msg amqp.Delivery, taskPayload *task.TaskPayload, originalEvent *events.TaskCreatedEvent, taskErr error, retryCount int32, workerID int) {
	err := utils.WithTransaction(db, func(tx *sql.Tx) error {
		if taskErr != nil {
			logrus.WithError(taskErr).Error("task failed")
			if updateErr := repo.MarkFailed(tx, taskPayload.TaskID, taskErr.Error()); updateErr != nil {
				return updateErr
			}
			return publishTaskFailedEvent(ch, taskPayload, originalEvent.Metadata.CorrelationID, taskErr.Error(), int(retryCount), workerID)
		}
		if updateErr := repo.MarkSuccess(tx, taskPayload.TaskID, "result.txt"); updateErr != nil {
			return updateErr
		}

		taskData, err := repo.GetByID(db, taskPayload.TaskID)
		if err != nil {
			logrus.WithError(err).Error("Failed to get task details")
			msg.Nack(false, true)
			return err
		}
		return publishTaskCompletedEvent(ch, taskPayload, originalEvent.Metadata.CorrelationID, "result.txt", taskData.UpdatedAt, workerID)
	})
	if err == nil {
		if ackErr := msg.Ack(false); ackErr != nil {
			logrus.WithError(ackErr).Warn("Failed to ack message")
		}
		return
	}

	// If update failed, handle retry logic
	logrus.WithError(err).Error("Failed to update task status")

	if retryCount >= events.MaxRetries {
		markAsMaxRetriesReached(db, repo, taskPayload.TaskID)
		msg.Nack(false, false)
		return
	}

	// Retry: republish message with incremented retry count
	logrus.Infof("Worker %d: Task failed, requeuing (retry %d/%d)", workerID, retryCount+1, events.MaxRetries)

	if err := queue.RepublishWithRetry(ch, &msg, retryCount+1); err != nil {
		logrus.WithError(err).Error("Failed to republish message")
		msg.Nack(false, false)
		return
	}

	if ackErr := msg.Ack(false); ackErr != nil {
		logrus.WithError(ackErr).Warn("Failed to ack message after republish")
	}
}

func markAsMaxRetriesReached(db *sql.DB, repo task.TaskRepositoryInterface, taskID int) {
	if err := utils.WithTransaction(db, func(tx *sql.Tx) error {
		return repo.MarkFailed(tx, taskID, "max retries reached")
	}); err != nil {
		logrus.WithError(err).Error("Failed to mark task as failed after max retries")
	}
}

func checkTaskStatus(status, eventID string, msg amqp.Delivery, updatedAt time.Time) error {
	if status != events.StatusPending {
		if status == events.StatusSuccess {
			msg.Ack(false) // acknowledge duplicate
			return fmt.Errorf("Task %s already completed, skipping", eventID)
		}
		if status == events.StatusFailed {
			msg.Ack(false)
			return fmt.Errorf("Task %s already failed, skipping", eventID)

		}
		if status == events.StatusProcessing {
			processingTime := time.Since(updatedAt)
			if processingTime < 10*time.Minute {
				msg.Nack(false, true) // requeue to check later
				return fmt.Errorf("Task %s being processed by another worker", eventID)

			}
		}
	}
	return nil
}

// publishTaskCompletedEvent publishes task.completed event to notification queue
func publishTaskCompletedEvent(ch *amqp.Channel, taskPayload *task.TaskPayload, correlationID, resultFile string, updatedAt time.Time, workerID int) error {
	processingTime := time.Since(updatedAt).Milliseconds()

	event := events.NewTaskCompletedEvent(
		taskPayload.TaskID,
		taskPayload.UserID,
		taskPayload.TaskType,
		resultFile,
		processingTime,
		fmt.Sprintf("worker-%d", workerID),
		correlationID,
	)

	eventJSON, err := json.Marshal(event)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal task.completed event")
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := queue.Publish(ch, "notification_queue", eventJSON); err != nil {
		logrus.WithError(err).Error("Failed to publish task.completed event")
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logrus.Infof("Published task.completed event for task_id=%d to notification_queue", taskPayload.TaskID)
	return nil
}

// publishTaskFailedEvent publishes task.failed event to notification queue
func publishTaskFailedEvent(ch *amqp.Channel, taskPayload *task.TaskPayload, correlationID, errorMsg string, retryCount, workerID int) error {
	event := events.NewTaskFailedEvent(
		taskPayload.TaskID,
		taskPayload.UserID,
		taskPayload.TaskType,
		errorMsg,
		"TASK_EXECUTION_ERROR",
		retryCount,
		fmt.Sprintf("worker-%d", workerID),
		correlationID,
	)

	eventJSON, err := json.Marshal(event)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal task.failed event")
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := queue.Publish(ch, "notification_queue", eventJSON); err != nil {
		logrus.WithError(err).Error("Failed to publish task.failed event")
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logrus.Infof("Published task.failed event for task_id=%d to notification_queue", taskPayload.TaskID)
	return nil
}
