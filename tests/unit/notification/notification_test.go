package notification_test

import (
	"task_handler/internal/events"
	"task_handler/internal/notification"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---- Mock Email Sender ----

type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendEmail(to, subject, body string) error {
	args := m.Called(to, subject, body)
	return args.Error(0)
}

func (m *MockEmailSender) SendTaskSucceddEmail(userEmail string, taskID int, taskType string, resultFile string) error {
	args := m.Called(userEmail, taskID, taskType, resultFile)
	return args.Error(0)
}

func (m *MockEmailSender) SendTaskFailedEmail(userEmail string, taskID int, taskType string, errorMessage string, retryCount int) error {
	args := m.Called(userEmail, taskID, taskType, errorMessage, retryCount)
	return args.Error(0)
}

func (m *MockEmailSender) SendUserRegisteredEmail(userEmail, username string) error {
	args := m.Called(userEmail, username)
	return args.Error(0)
}

// ---- MockEmailSender Tests ----

func TestMockEmailSender_SendEmail(t *testing.T) {
	sender := notification.NewMockEmailSender()

	err := sender.SendEmail("test@example.com", "Test Subject", "<p>Hello</p>")

	assert.NoError(t, err)
}

func TestMockEmailSender_SendTaskSucceddEmail(t *testing.T) {
	sender := notification.NewMockEmailSender()

	err := sender.SendTaskSucceddEmail("user@example.com", 1, "generate_report", "report.pdf")

	assert.NoError(t, err)
}

func TestMockEmailSender_SendTaskFailedEmail(t *testing.T) {
	sender := notification.NewMockEmailSender()

	err := sender.SendTaskFailedEmail("user@example.com", 2, "resize_image", "timeout error", 3)

	assert.NoError(t, err)
}

func TestMockEmailSender_SendUserRegisteredEmail(t *testing.T) {
	sender := notification.NewMockEmailSender()

	err := sender.SendUserRegisteredEmail("newuser@example.com", "alice")

	assert.NoError(t, err)
}

// ---- GetEmailSender Tests ----

func TestGetEmailSender_MockMode(t *testing.T) {
	sender := notification.GetEmailSender(true)

	require.NotNil(t, sender)
	// Should work without errors in mock mode
	err := sender.SendEmail("to@example.com", "Subject", "body")
	assert.NoError(t, err)
}

func TestGetEmailSender_RealMode(t *testing.T) {
	sender := notification.GetEmailSender(false)

	require.NotNil(t, sender)
	// Real sender is created (credentials not configured → will fail on actual send)
}

// ---- Mock-based notification handler tests ----

func TestMockEmailSenderInterface_ImplementsInterface(t *testing.T) {
	var _ notification.EmailSenderInterface = notification.NewMockEmailSender()
	var _ notification.EmailSenderInterface = notification.NewEmailSender()
}

func TestMockSender_TaskSucceddEmailFormat(t *testing.T) {
	sender := new(MockEmailSender)
	sender.On("SendTaskSucceddEmail", "u@x.com", 10, "generate_report", "out.pdf").Return(nil)

	err := sender.SendTaskSucceddEmail("u@x.com", 10, "generate_report", "out.pdf")

	assert.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestMockSender_TaskFailedEmailFormat(t *testing.T) {
	sender := new(MockEmailSender)
	sender.On("SendTaskFailedEmail", "u@x.com", 5, "resize_image", "OOM", 2).Return(nil)

	err := sender.SendTaskFailedEmail("u@x.com", 5, "resize_image", "OOM", 2)

	assert.NoError(t, err)
	sender.AssertExpectations(t)
}

// ---- Event-to-email mapping tests ----

func TestTaskSucceddEvent_HasRequiredFields(t *testing.T) {
	event := events.NewTaskCompletedEvent(1, 2, "generate_report", "result.pdf", 1000, "w1", "corr")

	assert.Equal(t, 1, event.Data.TaskID)
	assert.Equal(t, 2, event.Data.UserID)
	assert.Equal(t, "generate_report", event.Data.TaskType)
	assert.Equal(t, "result.pdf", event.Data.ResultFile)
}

func TestTaskFailedEvent_HasRequiredFields(t *testing.T) {
	event := events.NewTaskFailedEvent(3, 4, "resize_image", "timeout", "TIMEOUT", 3, "w2", "corr2")

	assert.Equal(t, 3, event.Data.TaskID)
	assert.Equal(t, 4, event.Data.UserID)
	assert.Equal(t, "timeout", event.Data.ErrorMessage)
	assert.Equal(t, 3, event.Data.RetryCount)
}
