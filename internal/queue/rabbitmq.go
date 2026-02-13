package queue

import (
	"context"
	"fmt"
	"sync"
	"task_handler/internal/config"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// RabbitMQManager manages RabbitMQ connection with auto-reconnect
type RabbitMQManager struct {
	config    *config.RabbitMQConfig
	conn      *amqp.Connection
	mu        sync.RWMutex
	isClosing bool
}

// NewRabbitMQManager creates a new RabbitMQ manager with auto-reconnect
func NewRabbitMQManager(rabbitMQCfg *config.RabbitMQConfig) *RabbitMQManager {
	manager := &RabbitMQManager{
		config: rabbitMQCfg,
	}

	// Initial connection
	manager.connect()

	// Start connection monitor
	go manager.monitorConnection()

	return manager
}

// SetupRabbitMQ creates a new RabbitMQ manager and returns it
func SetupRabbitMQ(rabbitMQCfg *config.RabbitMQConfig) *RabbitMQManager {
	return NewRabbitMQManager(rabbitMQCfg)
}

// connect establishes connection to RabbitMQ with retry
func (m *RabbitMQManager) connect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error
	maxRetries := 10
	baseDelay := time.Second

	for i := 0; i < maxRetries; i++ {
		if m.isClosing {
			return
		}

		m.conn, err = amqp.Dial(m.config.URL)
		if err == nil {
			logrus.WithFields(logrus.Fields{
				"service": "rabbitmq",
			}).Info("RabbitMQ connection established successfully")
			return
		}

		delay := baseDelay * time.Duration(i+1)
		logrus.WithFields(logrus.Fields{
			"service":     "rabbitmq",
			"attempt":     i + 1,
			"max_retries": maxRetries,
			"retry_in":    delay.String(),
			"error":       err.Error(),
		}).Warn("Failed to connect to RabbitMQ, retrying...")

		time.Sleep(delay)
	}

	logrus.WithFields(logrus.Fields{
		"service":     "rabbitmq",
		"max_retries": maxRetries,
		"error":       err.Error(),
	}).Fatal("Failed to connect to RabbitMQ after all retries")
}

// monitorConnection monitors connection health and reconnects if needed
func (m *RabbitMQManager) monitorConnection() {
	for {
		if m.isClosing {
			return
		}

		m.mu.RLock()
		conn := m.conn
		m.mu.RUnlock()

		if conn == nil {
			time.Sleep(time.Second)
			continue
		}

		// Listen for connection close
		closeNotify := conn.NotifyClose(make(chan *amqp.Error))
		closeErr := <-closeNotify

		if m.isClosing {
			return
		}

		if closeErr != nil {
			logrus.WithFields(logrus.Fields{
				"service": "rabbitmq",
				"error":   closeErr.Error(),
			}).Warn("RabbitMQ connection closed, attempting to reconnect...")
		} else {
			logrus.WithFields(logrus.Fields{
				"service": "rabbitmq",
			}).Warn("RabbitMQ connection lost, attempting to reconnect...")
		}

		// Reconnect
		m.connect()
	}
}

// GetConnection returns the current connection
// It waits until connection is ready and verified before returning
func (m *RabbitMQManager) GetConnection() *amqp.Connection {
	for {
		m.mu.RLock()
		conn := m.conn
		m.mu.RUnlock()

		// Check if connection is valid and not closed
		if conn != nil && !conn.IsClosed() {
			// Additional check: try to create a temporary channel to verify connection is truly ready
			testCh, err := conn.Channel()
			if err == nil {
				testCh.Close() // Close test channel immediately
				return conn    // Connection is ready
			}

			// Connection object exists but not fully ready yet
			logrus.WithFields(logrus.Fields{
				"service": "rabbitmq",
				"error":   err.Error(),
			}).Debug("Connection exists but not ready yet, waiting...")
		}

		// Connection not ready, wait a bit before retry
		time.Sleep(500 * time.Millisecond)
	}
}

// Close closes the RabbitMQ connection
func (m *RabbitMQManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.isClosing = true

	if m.conn != nil && !m.conn.IsClosed() {
		return m.conn.Close()
	}

	return nil
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

// PublishWithRetry publishes a message with automatic retry on transient failures
func PublishWithRetry(conn *amqp.Connection, queueName string, body []byte, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		ch, err := CreateChannel(conn)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				time.Sleep(time.Second * time.Duration(attempt+1))
				continue
			}
			break
		}

		err = Publish(ch, queueName, body)
		ch.Close()

		if err == nil {
			return nil
		}

		lastErr = err
		if attempt < maxRetries {
			logrus.WithFields(logrus.Fields{
				"service":     "rabbitmq",
				"queue":       queueName,
				"attempt":     attempt + 1,
				"max_retries": maxRetries,
				"error":       err.Error(),
			}).Warn("Failed to publish message, retrying...")
			time.Sleep(time.Second * time.Duration(attempt+1))
		}
	}

	return fmt.Errorf("failed to publish after %d attempts: %w", maxRetries+1, lastErr)
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
