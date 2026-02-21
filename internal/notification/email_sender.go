package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"time"

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
	Timeout      time.Duration
}

// ResendConfig holds Resend API configuration
type ResendConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
	Timeout   time.Duration
}

// EmailSender handles email sending operations
type EmailSender struct {
	config          *EmailConfig
	fallbackConfigs []*EmailConfig // Alternative SMTP servers
	resendConfig    *ResendConfig  // Resend API config
}

// NewEmailSender creates a new email sender with config from environment
func NewEmailSender() *EmailSender {
	primaryConfig := &EmailConfig{
		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("SMTP_FROM_EMAIL", "noreply@taskhandler.com"),
		FromName:     getEnv("SMTP_FROM_NAME", "Task Handler"),
		Timeout:      10 * time.Second,
	}

	// Resend API configuration (modern, developer-friendly)
	resendConfig := &ResendConfig{
		APIKey:    getEnv("RESEND_API_KEY", ""),
		FromEmail: getEnv("RESEND_FROM_EMAIL", "noreply@taskhandler.com"),
		FromName:  getEnv("SMTP_FROM_NAME", "Task Handler"),
		Timeout:   10 * time.Second,
	}

	// Fallback SMTP servers (tried if primary fails)
	fallbackConfigs := []*EmailConfig{
		// Fallback 1: Try port 465 (SSL) if 587 fails
		{
			SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     "465",
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("SMTP_FROM_EMAIL", "noreply@taskhandler.com"),
			FromName:     getEnv("SMTP_FROM_NAME", "Task Handler"),
			Timeout:      10 * time.Second,
		},
		// Fallback 2: Resend API (if configured)
		// Note: This is a placeholder, actual Resend logic is in sendViaResend()

		// Fallback 3: Alternative SMTP (e.g., SendGrid, Mailgun)
		{
			SMTPHost:     getEnv("SMTP_FALLBACK_HOST", "smtp.sendgrid.net"),
			SMTPPort:     getEnv("SMTP_FALLBACK_PORT", "587"),
			SMTPUsername: getEnv("SMTP_FALLBACK_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_FALLBACK_PASSWORD", ""),
			FromEmail:    getEnv("SMTP_FALLBACK_FROM_EMAIL", "noreply@taskhandler.com"),
			FromName:     getEnv("SMTP_FROM_NAME", "Task Handler"),
			Timeout:      10 * time.Second,
		},
	}

	return &EmailSender{
		config:          primaryConfig,
		fallbackConfigs: fallbackConfigs,
		resendConfig:    resendConfig,
	}
}

// SendEmail sends an email using SMTP with fallback support
func (e *EmailSender) SendEmail(to, subject, body string) error {
	// Try primary SMTP
	err := e.sendEmailWithConfig(e.config, to, subject, body)
	if err == nil {
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"service":     "email_sender",
		"smtp_host":   e.config.SMTPHost,
		"smtp_port":   e.config.SMTPPort,
		"error":       err.Error(),
		"retry_count": 0,
	}).Warn("Primary SMTP failed, trying fallbacks...")

	// Try Resend API first (if configured) - Modern and reliable
	if e.resendConfig.APIKey != "" {
		logrus.WithFields(logrus.Fields{
			"service":     "email_sender",
			"provider":    "resend",
			"retry_count": 1,
		}).Info("Trying Resend API...")

		err = e.sendViaResend(to, subject, body)
		if err == nil {
			logrus.WithFields(logrus.Fields{
				"service":  "email_sender",
				"provider": "resend",
			}).Info("Email sent successfully via Resend API")
			return nil
		}

		logrus.WithFields(logrus.Fields{
			"service":     "email_sender",
			"provider":    "resend",
			"error":       err.Error(),
			"retry_count": 1,
		}).Warn("Resend API failed")
	}

	// Try fallback SMTP servers
	startIndex := 2 // Resend was retry 1
	for i, fallbackConfig := range e.fallbackConfigs {
		// Skip if fallback not configured
		if fallbackConfig.SMTPUsername == "" || fallbackConfig.SMTPPassword == "" {
			continue
		}

		retryCount := startIndex + i
		logrus.WithFields(logrus.Fields{
			"service":     "email_sender",
			"smtp_host":   fallbackConfig.SMTPHost,
			"smtp_port":   fallbackConfig.SMTPPort,
			"retry_count": retryCount,
		}).Info("Trying fallback SMTP...")

		err = e.sendEmailWithConfig(fallbackConfig, to, subject, body)
		if err == nil {
			logrus.WithFields(logrus.Fields{
				"service":     "email_sender",
				"smtp_host":   fallbackConfig.SMTPHost,
				"smtp_port":   fallbackConfig.SMTPPort,
				"retry_count": retryCount,
			}).Info("Email sent successfully via fallback")
			return nil
		}

		logrus.WithFields(logrus.Fields{
			"service":     "email_sender",
			"smtp_host":   fallbackConfig.SMTPHost,
			"smtp_port":   fallbackConfig.SMTPPort,
			"error":       err.Error(),
			"retry_count": retryCount,
		}).Warn("Fallback SMTP failed")
	}

	// All attempts failed
	return fmt.Errorf("all email providers failed: %w", err)
}

// sendEmailWithConfig sends email using specific SMTP configuration
func (e *EmailSender) sendEmailWithConfig(config *EmailConfig, to, subject, body string) error {
	// Skip if SMTP not configured
	if config.SMTPUsername == "" || config.SMTPPassword == "" {
		return fmt.Errorf("SMTP credentials not configured")
	}

	// SMTP authentication
	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)

	// Build email message
	from := fmt.Sprintf("%s <%s>", config.FromName, config.FromEmail)
	msg := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", from, to, subject, body))

	// Send email with timeout
	addr := fmt.Sprintf("%s:%s", config.SMTPHost, config.SMTPPort)

	// Create channel for result
	errChan := make(chan error, 1)

	go func() {
		err := smtp.SendMail(addr, auth, config.FromEmail, []string{to}, msg)
		errChan <- err
	}()

	// Wait for result or timeout
	select {
	case err := <-errChan:
		if err != nil {
			return err
		}
		logrus.WithFields(logrus.Fields{
			"service":   "email_sender",
			"subject":   subject,
			"smtp_host": config.SMTPHost,
			"smtp_port": config.SMTPPort,
		}).Info("Email sent successfully")
		return nil
	case <-time.After(config.Timeout):
		return fmt.Errorf("SMTP connection timeout after %v", config.Timeout)
	}
}

// sendViaResend sends email using Resend API
func (e *EmailSender) sendViaResend(to, subject, body string) error {
	if e.resendConfig.APIKey == "" {
		return fmt.Errorf("Resend API key not configured")
	}

	// Resend API endpoint
	apiURL := "https://api.resend.com/emails"

	// Build request payload
	from := fmt.Sprintf("%s <%s>", e.resendConfig.FromName, e.resendConfig.FromEmail)
	payload := map[string]interface{}{
		"from":    from,
		"to":      []string{to},
		"subject": subject,
		"html":    body,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.resendConfig.APIKey))
	req.Header.Set("Content-Type", "application/json")

	// Send request with timeout
	client := &http.Client{
		Timeout: e.resendConfig.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errorResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResp)
		return fmt.Errorf("Resend API error (status %d): %v", resp.StatusCode, errorResp)
	}

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
