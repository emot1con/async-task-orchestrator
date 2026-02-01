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
	logrus.Infof("Handling task succeeded notification for task %d", event.Data.TaskID)

	// Get user email from database
	user, err := h.userRepo.GetByID(h.db, event.Data.UserID)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to get user %d for notification", event.Data.UserID)
		return err
	}

	// For now, use username as email (in production, you should have email field in users table)
	userEmail := user.Username + "@example.com" // TODO: Add email field to users table

	// Send email notification
	err = h.emailSender.SendTaskSucceddEmail(
		userEmail,
		event.Data.TaskID,
		event.Data.TaskType,
		event.Data.ResultFile,
	)

	if err != nil {
		logrus.WithError(err).Errorf("Failed to send task completed email to user %d", event.Data.UserID)
		return err
	}

	logrus.Infof("Successfully sent task completed notification to user %d", event.Data.UserID)
	return nil
}

// HandleTaskFailed handles task failure notification
func (h *NotificationHandler) HandleTaskFailed(event *events.TaskFailedEvent) error {
	logrus.Infof("Handling task failed notification for task %d", event.Data.TaskID)

	// Get user email from database
	user, err := h.userRepo.GetByID(h.db, event.Data.UserID)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to get user %d for notification", event.Data.UserID)
		return err
	}

	// For now, use username as email (in production, you should have email field in users table)
	userEmail := user.Username + "@example.com" // TODO: Add email field to users table

	// Send email notification
	err = h.emailSender.SendTaskFailedEmail(
		userEmail,
		event.Data.TaskID,
		event.Data.TaskType,
		event.Data.ErrorMessage,
		event.Data.RetryCount,
	)

	if err != nil {
		logrus.WithError(err).Errorf("Failed to send task failed email to user %d", event.Data.UserID)
		return err
	}

	logrus.Infof("Successfully sent task failed notification to user %d", event.Data.UserID)
	return nil
}

// HandleUserRegistered handles user registration welcome email
func (h *NotificationHandler) HandleUserRegistered(event *events.UserRegisteredEvent) error {
	logrus.Infof("Handling user registered notification for user %d", event.Data.UserID)

	// For now, use username as email
	userEmail := event.Data.Username + "@gmail.com" // TODO: Use actual email from event

	// Send welcome email
	err := h.emailSender.SendUserRegisteredEmail(
		userEmail,
		event.Data.Username,
	)

	if err != nil {
		logrus.WithError(err).Errorf("Failed to send welcome email to user %s", event.Data.Username)
		return err
	}

	logrus.Infof("Successfully sent welcome email to user %s", event.Data.Username)
	return nil
}
