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
		logrus.Fatalf("Failed to setup rabbitmq chanel on notification service: %v", err)
	}

	msgs, err := queue.Consume(ch, events.NotificationQueueName)
	if err != nil {
		logrus.Fatalf("Failed to consume notification message on notification service: %v", err)
	}

	logrus.Infof("Worker %d notification started", workerID)

	emailSender := notification.NewEmailSender()
	notifHandler := NewNotificationHandler(emailSender, repo, DB)

	for msg := range msgs {
		handleNotificationEvent(msg, ch, notifHandler)
	}
}

func handleNotificationEvent(msg amqp.Delivery, ch *amqp.Channel, notifHandler *NotificationHandler) {
	var base events.BaseEvent
	if err := json.Unmarshal(msg.Body, &base); err != nil {
		logrus.Errorf("Failed to unmarshal msg to base event on notification consumer: %v", err)
		if err := msg.Nack(false, false); err != nil {
			logrus.Errorf("Failed nack msg on notification service")
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
			logrus.Errorf("Max retry reached for event: %s", base.EventType)
			if err := msg.Nack(false, false); err != nil {
				logrus.Errorf("Failed to nack event on notification service: %v", err)
			}
			return
		}

		logrus.Errorf("Error handling event %s: %v. Retrying (%d)", base.EventType, err, retryCount+1)
		if err := queue.RepublishWithRetry(ch, &msg, retryCount+1); err != nil {
			logrus.Errorf("Failed to requeue msg on notification service")
			if err := msg.Nack(false, false); err != nil {
				logrus.Errorf("Failed to nack event on notification service and republish: %v", err)
			}
			return
		}

		if err := msg.Ack(false); err != nil {
			logrus.Errorf("Failed to ack event on notification service: %v", err)
			return
		}
	} else {
		logrus.Infof("Successfully handled event: %s", base.EventType)
		if err := msg.Ack(false); err != nil {
			logrus.Errorf("Failed to ack event on notification service: %v", err)
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
