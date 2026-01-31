package utils

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func RepublishWithRetry(ch *amqp.Channel, msg *amqp.Delivery, retryCount int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create new headers with incremented retry count
	headers := amqp.Table{}
	if msg.Headers != nil {
		headers = msg.Headers
	}
	headers["x-retry-count"] = retryCount

	return ch.PublishWithContext(
		ctx,
		"",             // exchange
		msg.RoutingKey, // routing key (queue name)
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: msg.ContentType,
			Body:        msg.Body,
			Headers:     headers,
		},
	)
}
