package worker

import (
	"fmt"
	"task_handler/internal/task"
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
	logrus.Infof("Worker %d sending email to user=%d", workerID, payload.UserID)

	time.Sleep(500 * time.Millisecond) // simulasi kirim email

	logrus.Infof("Worker %d email sent to user=%d", workerID, payload.UserID)
	return nil
}

func processGenerateReport(payload *task.TaskPayload, workerID int) error {
	logrus.Infof("Worker %d generating report for user=%d", workerID, payload.UserID)

	time.Sleep(5 * time.Second) // simulasi query + processing berat

	logrus.Infof("Worker %d report generated for user=%d", workerID, payload.UserID)
	return nil
}

func processResizeImage(payload *task.TaskPayload, workerID int) error {
	logrus.Infof("Worker %d resizing image for user=%d", workerID, payload.UserID)

	time.Sleep(2 * time.Second) // simulasi CPU-bound task

	logrus.Infof("Worker %d image resized for user=%d", workerID, payload.UserID)
	return nil
}

func processCleanupTemp(payload *task.TaskPayload, workerID int) error {
	logrus.Infof("Worker %d cleaning temp files for user=%d", workerID, payload.UserID)

	time.Sleep(1 * time.Second) // simulasi IO cleanup

	logrus.Infof("Worker %d temp cleanup done for user=%d", workerID, payload.UserID)
	return nil
}
