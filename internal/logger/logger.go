package logger

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// serviceHook adds service name to every log entry automatically
type serviceHook struct {
	service string
}

func (h *serviceHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *serviceHook) Fire(entry *logrus.Entry) error {
	// Only set if not already present (allow override)
	if _, exists := entry.Data["service"]; !exists {
		entry.Data["service"] = h.service
	}
	return nil
}

func InitLogger() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetOutput(os.Stdout)

	// Detect service name from environment variable or default
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "unknown"
	}

	logrus.AddHook(&serviceHook{service: serviceName})
}
