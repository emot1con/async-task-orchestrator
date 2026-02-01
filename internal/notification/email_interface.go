package notification

// EmailSenderInterface defines the contract for email sending
type EmailSenderInterface interface {
	SendEmail(to, subject, body string) error
	SendTaskSucceddEmail(userEmail string, taskID int, taskType string, resultFile string) error
	SendTaskFailedEmail(userEmail string, taskID int, taskType string, errorMessage string, retryCount int) error
	SendUserRegisteredEmail(userEmail, username string) error
}

// GetEmailSender returns appropriate email sender based on environment
func GetEmailSender(useMock bool) EmailSenderInterface {
	if useMock {
		return NewMockEmailSender()
	}
	return NewEmailSender()
}
