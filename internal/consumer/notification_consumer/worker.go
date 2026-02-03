package notification_consumer

import (
	"database/sql"
	"encoding/json"
	"task_handler/internal/config"
	"task_handler/internal/domain/user"
	"task_handler/internal/events"
	"task_handler/internal/notification"
	"task_handler/internal/queue"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func StartWorker(conn *amqp.Connection, DB *sql.DB, repo user.UserRepositoryInterface, workerID int, cfg *config.Config) {
	ch, err := conn.Channel()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "notification_consumer",
			"error":     err.Error(),
		}).Fatal("Failed to setup RabbitMQ channel")
	}

	msgs, err := queue.Consume(ch, events.NotificationQueueName)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"worker_id": workerID,
			"service":   "notification_consumer",
			"queue":     events.NotificationQueueName,
			"error":     err.Error(),
		}).Fatal("Failed to consume messages from queue")
	}

	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "notification_consumer",
		"queue":     events.NotificationQueueName,
	}).Info("Worker started successfully")

	emailSender := notification.NewEmailSender()
	notifHandler := NewNotificationHandler(emailSender, repo, DB)

	for msg := range msgs {
		handleNotificationEvent(msg, ch, notifHandler)
	}
}

func handleNotificationEvent(msg amqp.Delivery, ch *amqp.Channel, notifHandler *NotificationHandler) {
	var base events.BaseEvent
	if err := json.Unmarshal(msg.Body, &base); err != nil {
		logrus.WithFields(logrus.Fields{
			"service": "notification_consumer",
			"error":   err.Error(),
		}).Error("Failed to unmarshal message to BaseEvent")
		if err := msg.Nack(false, false); err != nil {
			logrus.WithFields(logrus.Fields{
				"service": "notification_consumer",
				"error":   err.Error(),
			}).Error("Failed to nack message")
		}
		return
	}

	var err error
	switch base.EventType {
	case events.EventTypeTaskSuccedd:
		err = handleSucceddTask(msg, notifHandler)
	case events.EventTypeTaskFailed:
		err = handleFailedTask(msg, notifHandler)
	case events.EventTypeUserRegistered:
		err = handleUserRegistered(msg, notifHandler)
	}

	if err != nil {
		retryCount := int32(0)
		if msg.Headers != nil {
			if count, ok := msg.Headers["x-retry-count"].(int32); ok {
				retryCount = count
			}
		}

		if retryCount >= 3 {
			logrus.WithFields(logrus.Fields{
				"service":     "notification_consumer",
				"event_id":    base.EventID,
				"event_type":  base.EventType,
				"retry_count": retryCount,
			}).Error("Max retry reached, message will be dead-lettered")
			if err := msg.Nack(false, false); err != nil {
				logrus.WithFields(logrus.Fields{
					"service":    "notification_consumer",
					"event_id":   base.EventID,
					"event_type": base.EventType,
					"error":      err.Error(),
				}).Error("Failed to nack message after max retry")
			}
			return
		}

		logrus.WithFields(logrus.Fields{
			"service":     "notification_consumer",
			"event_id":    base.EventID,
			"event_type":  base.EventType,
			"retry_count": retryCount,
			"error":       err.Error(),
		}).Warn("Event handling failed, will retry")
		if err := queue.RepublishWithRetry(ch, &msg, retryCount+1); err != nil {
			logrus.WithFields(logrus.Fields{
				"service":    "notification_consumer",
				"event_id":   base.EventID,
				"event_type": base.EventType,
				"error":      err.Error(),
			}).Error("Failed to republish message for retry")
			if err := msg.Nack(false, false); err != nil {
				logrus.WithFields(logrus.Fields{
					"service":    "notification_consumer",
					"event_id":   base.EventID,
					"event_type": base.EventType,
					"error":      err.Error(),
				}).Error("Failed to nack message after republish failure")
			}
			return
		}

		if err := msg.Ack(false); err != nil {
			logrus.WithFields(logrus.Fields{
				"service":    "notification_consumer",
				"event_id":   base.EventID,
				"event_type": base.EventType,
				"error":      err.Error(),
			}).Error("Failed to ack message after republish")
			return
		}
	} else {
		logrus.WithFields(logrus.Fields{
			"service":    "notification_consumer",
			"event_id":   base.EventID,
			"event_type": base.EventType,
		}).Info("Event handled successfully")
		if err := msg.Ack(false); err != nil {
			logrus.WithFields(logrus.Fields{
				"service":    "notification_consumer",
				"event_id":   base.EventID,
				"event_type": base.EventType,
				"error":      err.Error(),
			}).Error("Failed to ack message after successful handling")
			return
		}
	}
}

func handleSucceddTask(msg amqp.Delivery, notifHandler *NotificationHandler) error {
	var succeddTask events.TaskSucceddEvent
	if err := json.Unmarshal(msg.Body, &succeddTask); err != nil {
		return err
	}

	if err := notifHandler.HandleTaskSuccedd(&succeddTask); err != nil {
		return err
	}
	return nil
}

func handleFailedTask(msg amqp.Delivery, notifHandler *NotificationHandler) error {
	var failedTask events.TaskFailedEvent
	if err := json.Unmarshal(msg.Body, &failedTask); err != nil {
		return err
	}

	if err := notifHandler.HandleTaskFailed(&failedTask); err != nil {
		return err
	}
	return nil
}

func handleUserRegistered(msg amqp.Delivery, notifHandler *NotificationHandler) error {
	var userRegistered events.UserRegisteredEvent
	if err := json.Unmarshal(msg.Body, &userRegistered); err != nil {
		return err
	}

	if err := notifHandler.HandleUserRegistered(&userRegistered); err != nil {
		return err
	}
	return nil
}
