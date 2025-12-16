package queue

import (
	"fmt"
	"log"
	"task_handler/internal/config"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SetupRabbitMQ(rabbitMQCfg *config.RabbitMQConfig) *amqp.Connection {
	var conn *amqp.Connection
	var err error

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(rabbitMQCfg.URL)
		if err != nil {
			log.Printf("Failed to connect to RabbitMQ (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		break
	}

	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ after %d attempts: %v", maxRetries, err)
	}

	log.Println("RabbitMQ connection established successfully")
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
