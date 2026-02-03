package queue

import (
	"context"
	"fmt"
	"task_handler/internal/config"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func SetupRabbitMQ(rabbitMQCfg *config.RabbitMQConfig) *amqp.Connection {
	var conn *amqp.Connection
	var err error

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(rabbitMQCfg.URL) // URL contains credentials, don't log it
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"service":     "rabbitmq",
				"attempt":     i + 1,
				"max_retries": maxRetries,
				"error":       err.Error(),
			}).Warn("Failed to connect to RabbitMQ")
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		break
	}

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"service":     "rabbitmq",
			"max_retries": maxRetries,
			"error":       err.Error(),
		}).Fatal("Failed to connect to RabbitMQ after retries")
	}

	logrus.WithFields(logrus.Fields{
		"service": "rabbitmq",
	}).Info("RabbitMQ connection established successfully")
	return conn
}

func CreateChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	return ch, nil
}

func DeclareQueue(ch *amqp.Channel, queueName string) (amqp.Queue, error) {
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return amqp.Queue{}, fmt.Errorf("failed to declare queue: %w", err)
	}

	return q, nil
}

func Publish(ch *amqp.Channel, queueName string, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return ch.PublishWithContext(
		ctx,
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func Consume(ch *amqp.Channel, queueName string) (<-chan amqp.Delivery, error) {
	return ch.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
}

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
