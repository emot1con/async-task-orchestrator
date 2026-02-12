package task_consumer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"task_handler/internal/cache"
	"task_handler/internal/domain/task"
	"task_handler/internal/events"
	"task_handler/internal/queue"
	"task_handler/internal/utils"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func StartWorker(manager *queue.RabbitMQManager, db *sql.DB, repo task.TaskRepositoryInterface, taskCache *cache.TaskCache, workerID int) {
	for {
		conn := manager.GetConnection()
		if err := runWorker(conn, db, repo, taskCache, workerID); err != nil {
			logrus.WithFields(logrus.Fields{
				"worker_id": workerID,
				"service":   "task_consumer",
				"error":     err.Error(),
			}).Error("Worker stopped, restarting in 5 seconds...")
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

func runWorker(conn *amqp.Connection, db *sql.DB, repo task.TaskRepositoryInterface, taskCache *cache.TaskCache, workerID int) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}
	defer ch.Close()

	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := queue.Consume(ch, events.TaskQueueName)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "task_consumer",
		"queue":     events.TaskQueueName,
	}).Info("Worker started successfully")

	// Listen for channel close to detect connection loss
	closeNotify := ch.NotifyClose(make(chan *amqp.Error))

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}
			processMessage(ch, db, repo, taskCache, workerID, msg)

		case closeErr := <-closeNotify:
			if closeErr != nil {
				return fmt.Errorf("channel closed: %v", closeErr)
			}
			return fmt.Errorf("channel closed unexpectedly")
		}
	}
}

func processMessage(ch *amqp.Channel, db *sql.DB, repo task.TaskRepositoryInterface, taskCache *cache.TaskCache, workerID int, msg amqp.Delivery) {
	// Parse and validate event
	taskPayload, event, retryCount, err := parseAndValidateMessage(db, repo, msg)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "task_consumer",
			"error":     err.Error(),
		}).Error("Message validation failed")
		return // message already acked/nacked inside
	}

	logrus.WithFields(logrus.Fields{
		"worker_id":   workerID,
		"service":     "task_consumer",
		"task_id":     taskPayload.TaskID,
		"user_id":     taskPayload.UserID,
		"task_type":   taskPayload.TaskType,
		"retry_count": retryCount,
	}).Info("Processing task")

	// Mark as PROCESSING
	if err := markAsProcessing(db, repo, taskPayload.TaskID, workerID); err != nil {
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "task_consumer",
			"task_id":   taskPayload.TaskID,
			"error":     err.Error(),
		}).Error("Failed to mark task as processing")
		msg.Nack(false, true)
		return
	}

	// Execute task
	taskErr := handleTask(taskPayload, workerID)

	// Update final status and handle retry if needed
	handleTaskResult(ch, db, repo, taskCache, msg, taskPayload, event, taskErr, retryCount, workerID)
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
		logrus.WithFields(logrus.Fields{
			"service": "task_consumer",
			"task_id": taskPayload.TaskID,
			"error":   err.Error(),
		}).Error("Failed to get task status from database")
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
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "task_consumer",
			"task_id":   taskID,
			"status":    "PROCESSING",
		}).Info("Marking task as PROCESSING")
		return repo.MarkProcessing(tx, taskID)
	})
}

func handleTaskResult(ch *amqp.Channel, db *sql.DB, repo task.TaskRepositoryInterface, taskCache *cache.TaskCache, msg amqp.Delivery, taskPayload *task.TaskPayload, originalEvent *events.TaskCreatedEvent, taskErr error, retryCount int32, workerID int) {
	err := utils.WithTransaction(db, func(tx *sql.Tx) error {
		if taskErr != nil {
			logrus.WithFields(logrus.Fields{
				"worker_id": workerID,
				"service":   "task_consumer",
				"task_id":   taskPayload.TaskID,
				"user_id":   taskPayload.UserID,
				"error":     taskErr.Error(),
			}).Error("Task execution failed")
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
			logrus.WithFields(logrus.Fields{
				"worker_id": workerID,
				"service":   "task_consumer",
				"task_id":   taskPayload.TaskID,
				"error":     err.Error(),
			}).Error("Failed to get task details after success")
			msg.Nack(false, true)
			return err
		}
		return publishTaskCompletedEvent(ch, taskPayload, originalEvent.Metadata.CorrelationID, "result.txt", taskData.UpdatedAt, workerID)
	})
	if err == nil {
		// Invalidate cache (consistent with task service pattern)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if cacheErr := taskCache.DeleteTaskCache(ctx, taskPayload.TaskID, taskPayload.UserID); cacheErr != nil {
			logrus.WithFields(logrus.Fields{
				"worker_id": workerID,
				"service":   "task_consumer",
				"task_id":   taskPayload.TaskID,
				"user_id":   taskPayload.UserID,
				"error":     cacheErr.Error(),
			}).Warn("Failed to invalidate task cache after update")
		}

		if ackErr := msg.Ack(false); ackErr != nil {
			logrus.WithFields(logrus.Fields{
				"worker_id": workerID,
				"service":   "task_consumer",
				"task_id":   taskPayload.TaskID,
				"error":     ackErr.Error(),
			}).Warn("Failed to ack message")
		}
		return
	}

	// If update failed, handle retry logic
	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "task_consumer",
		"task_id":   taskPayload.TaskID,
		"error":     err.Error(),
	}).Error("Failed to update task status")

	if retryCount >= events.MaxRetries {
		markAsMaxRetriesReached(db, repo, taskPayload.TaskID, taskPayload.UserID, workerID)
		msg.Nack(false, false)
		return
	}

	// Retry: republish message with incremented retry count
	logrus.WithFields(logrus.Fields{
		"worker_id":   workerID,
		"service":     "task_consumer",
		"task_id":     taskPayload.TaskID,
		"retry_count": retryCount + 1,
		"max_retries": events.MaxRetries,
	}).Info("Task failed, requeuing for retry")

	if err := queue.RepublishWithRetry(ch, &msg, retryCount+1); err != nil {
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "task_consumer",
			"task_id":   taskPayload.TaskID,
			"error":     err.Error(),
		}).Error("Failed to republish message for retry")
		msg.Nack(false, false)
		return
	}

	if ackErr := msg.Ack(false); ackErr != nil {
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "task_consumer",
			"task_id":   taskPayload.TaskID,
			"error":     ackErr.Error(),
		}).Warn("Failed to ack message after republish")
	}
}

func markAsMaxRetriesReached(db *sql.DB, repo task.TaskRepositoryInterface, taskID, userID, workerID int) {
	if err := utils.WithTransaction(db, func(tx *sql.Tx) error {
		return repo.MarkFailed(tx, taskID, "max retries reached")
	}); err != nil {
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "task_consumer",
			"task_id":   taskID,
			"user_id":   userID,
			"error":     err.Error(),
		}).Error("Failed to mark task as failed after max retries")
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
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "task_consumer",
			"task_id":   taskPayload.TaskID,
			"event":     "task.completed",
			"error":     err.Error(),
		}).Error("Failed to marshal task.completed event")
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := queue.Publish(ch, events.NotificationQueueName, eventJSON); err != nil {
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "task_consumer",
			"task_id":   taskPayload.TaskID,
			"event":     "task.completed",
			"queue":     events.NotificationQueueName,
			"error":     err.Error(),
		}).Error("Failed to publish task.completed event")
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"worker_id":       workerID,
		"service":         "task_consumer",
		"task_id":         taskPayload.TaskID,
		"event":           "task.completed",
		"queue":           events.NotificationQueueName,
		"processing_time": processingTime,
	}).Info("Published task.completed event")

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
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "task_consumer",
			"task_id":   taskPayload.TaskID,
			"event":     "task.failed",
			"error":     err.Error(),
		}).Error("Failed to marshal task.failed event")
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := queue.Publish(ch, events.NotificationQueueName, eventJSON); err != nil {
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "task_consumer",
			"task_id":   taskPayload.TaskID,
			"event":     "task.failed",
			"queue":     events.NotificationQueueName,
			"error":     err.Error(),
		}).Error("Failed to publish task.failed event")
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"worker_id":   workerID,
		"service":     "task_consumer",
		"task_id":     taskPayload.TaskID,
		"event":       "task.failed",
		"queue":       events.NotificationQueueName,
		"retry_count": retryCount,
	}).Info("Published task.failed event")

	return nil
}
