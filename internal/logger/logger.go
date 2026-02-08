package logger

import (
	"time"

	"github.com/sirupsen/logrus"
)

func InitLogger() {
	// if err := os.MkdirAll(logPath, 0755); err != nil {
	// 	return err
	// }

	// // Buka atau buat file log
	// logFile, err := os.OpenFile(logPath+"/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	// if err != nil {
	// 	return err
	// }

	// log = logrus.New()

	// multiWriter := io.MultiWriter(os.Stdout, logFile)
	// log.SetOutput(multiWriter)

	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	logrus.SetLevel(logrus.InfoLevel)
}
