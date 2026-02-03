package notification_consumer

import (
	"database/sql"
	"task_handler/internal/domain/user"
	"task_handler/internal/events"
	"task_handler/internal/notification"

	"github.com/sirupsen/logrus"
)

// NotificationHandler handles different notification events
type NotificationHandler struct {
	emailSender notification.EmailSenderInterface
	userRepo    user.UserRepositoryInterface
	db          *sql.DB
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(emailSender notification.EmailSenderInterface, userRepo user.UserRepositoryInterface, db *sql.DB) *NotificationHandler {
	return &NotificationHandler{
		emailSender: emailSender,
		userRepo:    userRepo,
		db:          db,
	}
}

// HandleTaskSuccedd handles task completion notification
func (h *NotificationHandler) HandleTaskSuccedd(event *events.TaskSucceddEvent) error {
	logrus.WithFields(logrus.Fields{
		"service":   "notification_handler",
		"event":     "task.completed",
		"task_id":   event.Data.TaskID,
		"user_id":   event.Data.UserID,
		"task_type": event.Data.TaskType,
	}).Info("Handling task completed notification")

	// For now, use hardcoded email (in production, you should have email field in users table)
	userEmail := "taskhandleremot1con@gmail.com" // TODO: Add email field to users table

	// Send email notification
	err := h.emailSender.SendTaskSucceddEmail(
		userEmail,
		event.Data.TaskID,
		event.Data.TaskType,
		event.Data.ResultFile,
	)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"service": "notification_handler",
			"event":   "task.completed",
			"task_id": event.Data.TaskID,
			"user_id": event.Data.UserID,
			"error":   err.Error(),
		}).Error("Failed to send task completed email")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"service": "notification_handler",
		"event":   "task.completed",
		"task_id": event.Data.TaskID,
		"user_id": event.Data.UserID,
	}).Info("Successfully sent task completed notification")
	return nil
}

// HandleTaskFailed handles task failure notification
func (h *NotificationHandler) HandleTaskFailed(event *events.TaskFailedEvent) error {
	logrus.WithFields(logrus.Fields{
		"service":     "notification_handler",
		"event":       "task.failed",
		"task_id":     event.Data.TaskID,
		"user_id":     event.Data.UserID,
		"task_type":   event.Data.TaskType,
		"retry_count": event.Data.RetryCount,
	}).Info("Handling task failed notification")

	// For now, use hardcoded email (in production, you should have email field in users table)
	userEmail := "taskhandleremot1con@gmail.com" // TODO: Add email field to users table

	// Send email notification
	err := h.emailSender.SendTaskFailedEmail(
		userEmail,
		event.Data.TaskID,
		event.Data.TaskType,
		event.Data.ErrorMessage,
		event.Data.RetryCount,
	)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"service": "notification_handler",
			"event":   "task.failed",
			"task_id": event.Data.TaskID,
			"user_id": event.Data.UserID,
			"error":   err.Error(),
		}).Error("Failed to send task failed email")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"service": "notification_handler",
		"event":   "task.failed",
		"task_id": event.Data.TaskID,
		"user_id": event.Data.UserID,
	}).Info("Successfully sent task failed notification")
	return nil
}

// HandleUserRegistered handles user registration welcome email
func (h *NotificationHandler) HandleUserRegistered(event *events.UserRegisteredEvent) error {
	logrus.WithFields(logrus.Fields{
		"service":  "notification_handler",
		"event":    "user.registered",
		"user_id":  event.Data.UserID,
		"username": event.Data.Username,
	}).Info("Handling user registered notification")

	// For now, use username as email
	userEmail := event.Data.Username + "@gmail.com" // TODO: Use actual email from event

	// Send welcome email
	err := h.emailSender.SendUserRegisteredEmail(
		userEmail,
		event.Data.Username,
	)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"service":  "notification_handler",
			"event":    "user.registered",
			"user_id":  event.Data.UserID,
			"username": event.Data.Username,
			"error":    err.Error(),
		}).Error("Failed to send welcome email")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"service":  "notification_handler",
		"event":    "user.registered",
		"user_id":  event.Data.UserID,
		"username": event.Data.Username,
	}).Info("Successfully sent welcome email")
	return nil
}
