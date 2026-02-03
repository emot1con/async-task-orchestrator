package notification

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// MockEmailSender is a mock implementation for development/testing
// It logs email content instead of actually sending emails
type MockEmailSender struct{}

// NewMockEmailSender creates a new mock email sender
func NewMockEmailSender() *MockEmailSender {
	logrus.WithFields(logrus.Fields{
		"service": "email_sender",
		"mode":    "mock",
	}).Info("Using MockEmailSender - emails will be logged, not sent")
	return &MockEmailSender{}
}

// SendEmail logs email instead of sending (without PII)
func (m *MockEmailSender) SendEmail(to, subject, body string) error {
	logrus.WithFields(logrus.Fields{
		"service": "email_sender",
		"mode":    "mock",
		"subject": subject,
	}).Info("Mock email sent") // Don't log email address or body (may contain PII)
	return nil
}

// SendTaskSucceddEmail logs task completion email
func (m *MockEmailSender) SendTaskSucceddEmail(userEmail string, taskID int, taskType string, resultFile string) error {
	logrus.WithFields(logrus.Fields{
		"service":     "email_sender",
		"mode":        "mock",
		"email_type":  "task_completed",
		"task_id":     taskID,
		"task_type":   taskType,
		"result_file": resultFile,
	}).Info("Mock: Sending task completed email") // Don't log email address (PII)

	return m.SendEmail(
		userEmail,
		fmt.Sprintf("Task #%d Completed Successfully", taskID),
		fmt.Sprintf("Task #%d (%s) completed with result: %s", taskID, taskType, resultFile),
	)
}

// SendTaskFailedEmail logs task failure email
func (m *MockEmailSender) SendTaskFailedEmail(userEmail string, taskID int, taskType string, errorMessage string, retryCount int) error {
	logrus.WithFields(logrus.Fields{
		"service":     "email_sender",
		"mode":        "mock",
		"email_type":  "task_failed",
		"task_id":     taskID,
		"task_type":   taskType,
		"retry_count": retryCount,
	}).Info("Mock: Sending task failed email") // Don't log email address or error details (may contain sensitive info)

	return m.SendEmail(
		userEmail,
		fmt.Sprintf("Task #%d Failed", taskID),
		fmt.Sprintf("Task #%d (%s) failed: %s (Retry count: %d)", taskID, taskType, errorMessage, retryCount),
	)
}

// SendUserRegisteredEmail logs welcome email
func (m *MockEmailSender) SendUserRegisteredEmail(userEmail, username string) error {
	logrus.WithFields(logrus.Fields{
		"service":    "email_sender",
		"mode":       "mock",
		"email_type": "user_registered",
	}).Info("Mock: Sending welcome email") // Don't log email address or username (PII)

	return m.SendEmail(
		userEmail,
		"Welcome to Task Handler!",
		fmt.Sprintf("Welcome %s! Your account has been created successfully.", username),
	)
}
