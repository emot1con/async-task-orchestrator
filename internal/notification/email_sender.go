package notification

import (
	"fmt"
	"net/smtp"
	"os"

	"github.com/sirupsen/logrus"
)

// EmailConfig holds SMTP configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

// EmailSender handles email sending operations
type EmailSender struct {
	config *EmailConfig
}

// NewEmailSender creates a new email sender with config from environment
func NewEmailSender() *EmailSender {
	return &EmailSender{
		config: &EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     getEnv("SMTP_PORT", "587"),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("SMTP_FROM_EMAIL", "noreply@taskhandler.com"),
			FromName:     getEnv("SMTP_FROM_NAME", "Task Handler"),
		},
	}
}

// SendEmail sends an email using SMTP
func (e *EmailSender) SendEmail(to, subject, body string) error {
	// SMTP authentication
	auth := smtp.PlainAuth("", e.config.SMTPUsername, e.config.SMTPPassword, e.config.SMTPHost)

	// Build email message
	from := fmt.Sprintf("%s <%s>", e.config.FromName, e.config.FromEmail)
	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", from, to, subject, body))

	// Send email
	addr := fmt.Sprintf("%s:%s", e.config.SMTPHost, e.config.SMTPPort)
	err := smtp.SendMail(addr, auth, e.config.FromEmail, []string{to}, msg)

	if err != nil {
		logrus.WithError(err).Errorf("Failed to send email to %s", to)
		return err
	}

	logrus.Infof("Email sent successfully to %s", to)
	return nil
}

// SendTaskSucceddEmail sends notification for completed task
func (e *EmailSender) SendTaskSucceddEmail(userEmail string, taskID int, taskType string, resultFile string) error {
	subject := fmt.Sprintf("Task #%d Completed Successfully", taskID)

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
			<div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 5px;">
				<h2 style="color: #4CAF50;">✅ Task Completed Successfully</h2>
				<p>Your task has been completed successfully!</p>
				
				<div style="background-color: #f9f9f9; padding: 15px; border-radius: 5px; margin: 20px 0;">
					<p><strong>Task ID:</strong> #%d</p>
					<p><strong>Task Type:</strong> %s</p>
					<p><strong>Status:</strong> <span style="color: #4CAF50; font-weight: bold;">SUCCESS</span></p>
					<p><strong>Result File:</strong> %s</p>
				</div>
				
				<p>You can download the result file or view the details in your dashboard.</p>
				
				<div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #777;">
					<p>This is an automated notification from Task Handler.</p>
					<p>If you have any questions, please contact support.</p>
				</div>
			</div>
		</body>
		</html>
	`, taskID, taskType, resultFile)

	return e.SendEmail(userEmail, subject, body)
}

// SendTaskFailedEmail sends notification for failed task
func (e *EmailSender) SendTaskFailedEmail(userEmail string, taskID int, taskType string, errorMessage string, retryCount int) error {
	subject := fmt.Sprintf("Task #%d Failed", taskID)

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
			<div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 5px;">
				<h2 style="color: #f44336;">❌ Task Failed</h2>
				<p>Unfortunately, your task encountered an error and could not be completed.</p>
				
				<div style="background-color: #fff3f3; padding: 15px; border-radius: 5px; margin: 20px 0; border-left: 4px solid #f44336;">
					<p><strong>Task ID:</strong> #%d</p>
					<p><strong>Task Type:</strong> %s</p>
					<p><strong>Status:</strong> <span style="color: #f44336; font-weight: bold;">FAILED</span></p>
					<p><strong>Error:</strong> %s</p>
					<p><strong>Retry Count:</strong> %d</p>
				</div>
				
				<p>Our team has been notified and will investigate the issue. You may try creating a new task or contact support for assistance.</p>
				
				<div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #777;">
					<p>This is an automated notification from Task Handler.</p>
					<p>If you need immediate assistance, please contact support.</p>
				</div>
			</div>
		</body>
		</html>
	`, taskID, taskType, errorMessage, retryCount)

	return e.SendEmail(userEmail, subject, body)
}

// SendUserRegisteredEmail sends welcome email to newly registered user
func (e *EmailSender) SendUserRegisteredEmail(userEmail, username string) error {
	subject := "Welcome to Task Handler!"

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
			<div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 5px;">
				<h2 style="color: #2196F3;">🎉 Welcome to Task Handler!</h2>
				<p>Hi <strong>%s</strong>,</p>
				<p>Thank you for registering with Task Handler. Your account has been successfully created!</p>
				
				<div style="background-color: #f0f8ff; padding: 15px; border-radius: 5px; margin: 20px 0;">
					<p><strong>Username:</strong> %s</p>
					<p><strong>Email:</strong> %s</p>
				</div>
				
				<p>You can now start creating and managing tasks. Our system will notify you whenever a task is completed or if any issues occur.</p>
				
				<h3 style="color: #2196F3;">Getting Started:</h3>
				<ul style="line-height: 2;">
					<li>Login to your account</li>
					<li>Create your first task</li>
					<li>Monitor task progress in real-time</li>
					<li>Receive notifications on task completion</li>
				</ul>
				
				<div style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #777;">
					<p>This is an automated notification from Task Handler.</p>
					<p>If you have any questions, please contact support.</p>
				</div>
			</div>
		</body>
		</html>
	`, username, username, userEmail)

	return e.SendEmail(userEmail, subject, body)
}

// Helper function to get environment variable with default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
