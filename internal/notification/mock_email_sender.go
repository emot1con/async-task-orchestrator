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
	logrus.Info("Using MockEmailSender - emails will be logged, not sent")
	return &MockEmailSender{}
}

// SendEmail logs email instead of sending
func (m *MockEmailSender) SendEmail(to, subject, body string) error {
	logrus.Info("===============================================")
	logrus.Info("MOCK EMAIL SENT")
	logrus.Infof("To: %s", to)
	logrus.Infof("Subject: %s", subject)
	logrus.Info("Body: (HTML content - see below)")
	logrus.Info("-----------------------------------------------")
	// Log body as plain text for readability (strip HTML in real scenario)
	logrus.Info(body)
	logrus.Info("===============================================")
	return nil
}

// SendTaskSucceddEmail logs task completion email
func (m *MockEmailSender) SendTaskSucceddEmail(userEmail string, taskID int, taskType string, resultFile string) error {
	logrus.Infof("📧 MOCK: Sending task completed email to %s", userEmail)
	logrus.Infof("   Task ID: %d, Type: %s, Result: %s", taskID, taskType, resultFile)

	return m.SendEmail(
		userEmail,
		fmt.Sprintf("Task #%d Completed Successfully", taskID),
		fmt.Sprintf("Task #%d (%s) completed with result: %s", taskID, taskType, resultFile),
	)
}

// SendTaskFailedEmail logs task failure email
func (m *MockEmailSender) SendTaskFailedEmail(userEmail string, taskID int, taskType string, errorMessage string, retryCount int) error {
	logrus.Infof("📧 MOCK: Sending task failed email to %s", userEmail)
	logrus.Infof("   Task ID: %d, Type: %s, Error: %s, Retries: %d", taskID, taskType, errorMessage, retryCount)

	return m.SendEmail(
		userEmail,
		fmt.Sprintf("Task #%d Failed", taskID),
		fmt.Sprintf("Task #%d (%s) failed: %s (Retry count: %d)", taskID, taskType, errorMessage, retryCount),
	)
}

// SendUserRegisteredEmail logs welcome email
func (m *MockEmailSender) SendUserRegisteredEmail(userEmail, username string) error {
	logrus.Infof("📧 MOCK: Sending welcome email to %s", userEmail)
	logrus.Infof("   Username: %s", username)

	return m.SendEmail(
		userEmail,
		"Welcome to Task Handler!",
		fmt.Sprintf("Welcome %s! Your account has been created successfully.", username),
	)
}
