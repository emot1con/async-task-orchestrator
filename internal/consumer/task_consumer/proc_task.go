package task_consumer

import (
	"fmt"
	"task_handler/internal/domain/task"
	"time"

	"github.com/sirupsen/logrus"
)

func handleTask(payload *task.TaskPayload, workerID int) error {
	switch payload.TaskType {
	case "send_email":
		return processSendEmail(payload, workerID)
	case "generate_report":
		return processGenerateReport(payload, workerID)
	case "resize_image":
		return processResizeImage(payload, workerID)
	case "cleanup_temp":
		return processCleanupTemp(payload, workerID)
	default:
		return fmt.Errorf("unknown task type: %s", payload.TaskType)
	}
}

func processSendEmail(payload *task.TaskPayload, workerID int) error {
	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "task_processor",
		"task_id":   payload.TaskID,
		"task_type": "send_email",
		"user_id":   payload.UserID,
	}).Info("Processing send email task")

	time.Sleep(500 * time.Millisecond) // simulasi kirim email

	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "task_processor",
		"task_id":   payload.TaskID,
		"task_type": "send_email",
	}).Info("Email sent successfully")
	return nil
}

func processGenerateReport(payload *task.TaskPayload, workerID int) error {
	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "task_processor",
		"task_id":   payload.TaskID,
		"task_type": "generate_report",
		"user_id":   payload.UserID,
	}).Info("Processing generate report task")

	time.Sleep(5 * time.Second) // simulasi query + processing berat

	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "task_processor",
		"task_id":   payload.TaskID,
		"task_type": "generate_report",
	}).Info("Report generated successfully")
	return nil
}

func processResizeImage(payload *task.TaskPayload, workerID int) error {
	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "task_processor",
		"task_id":   payload.TaskID,
		"task_type": "resize_image",
		"user_id":   payload.UserID,
	}).Info("Processing resize image task")

	time.Sleep(2 * time.Second) // simulasi CPU-bound task

	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "task_processor",
		"task_id":   payload.TaskID,
		"task_type": "resize_image",
	}).Info("Image resized successfully")
	return nil
}

func processCleanupTemp(payload *task.TaskPayload, workerID int) error {
	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "task_processor",
		"task_id":   payload.TaskID,
		"task_type": "cleanup_temp",
		"user_id":   payload.UserID,
	}).Info("Processing cleanup temp task")

	time.Sleep(1 * time.Second) // simulasi IO cleanup

	logrus.WithFields(logrus.Fields{
		"worker_id": workerID,
		"service":   "task_processor",
		"task_id":   payload.TaskID,
		"task_type": "cleanup_temp",
	}).Info("Temp cleanup completed successfully")
	return nil
}
