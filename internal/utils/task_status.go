package utils

import (
	"fmt"
	"task_handler/internal/events"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func CheckTaskStatus(status, eventID string, msg amqp.Delivery, updatedAt time.Time) error {
	if status != events.StatusPending {
		if status == events.StatusSuccess {
			msg.Ack(false) // acknowledge duplicate
			return fmt.Errorf("Task %s already completed, skipping", eventID)
		}
		if status == events.StatusFailed {
			msg.Ack(false)
			return fmt.Errorf("Task %s already failed, skipping", eventID)

		}
		if status == events.StatusProcessing {
			processingTime := time.Since(updatedAt)
			if processingTime < 10*time.Minute {
				msg.Nack(false, true) // requeue untuk cek lagi nanti
				return fmt.Errorf("Task %s being processed by another worker", eventID)

			}
		}
	}
	return nil
}
