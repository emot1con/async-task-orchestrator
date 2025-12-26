//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"task_handler/internal/cache"
	"task_handler/internal/config"
	"task_handler/internal/db"
	"task_handler/internal/queue"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

// TestDependencies holds all integration test dependencies
type TestDependencies struct {
	DB          *sql.DB
	RedisClient *redis.Client
	RabbitConn  *amqp.Connection
	Config      *config.Config
}

// SetupTestEnvironment initializes all dependencies for integration tests
func SetupTestEnvironment(t *testing.T) *TestDependencies {
	t.Helper()

	// Load configuration from environment
	cfg := loadTestConfig()

	// Setup PostgreSQL
	database := setupTestDB(t, cfg)

	// Setup Redis
	redisClient := setupTestRedis(t, cfg)

	// Setup RabbitMQ
	rabbitConn := setupTestRabbitMQ(t, cfg)

	return &TestDependencies{
		DB:          database,
		RedisClient: redisClient,
		RabbitConn:  rabbitConn,
		Config:      cfg,
	}
}

// Cleanup tears down all test dependencies
func (td *TestDependencies) Cleanup(t *testing.T) {
	t.Helper()

	if td.DB != nil {
		cleanupTestDB(t, td.DB)
		td.DB.Close()
	}

	if td.RedisClient != nil {
		td.RedisClient.FlushDB(context.Background())
		td.RedisClient.Close()
	}

	if td.RabbitConn != nil {
		td.RabbitConn.Close()
	}
}

// loadTestConfig loads configuration for integration tests
func loadTestConfig() *config.Config {
	return &config.Config{
		AppName: "test-integration",
		AppEnv:  "test",
		AppPort: getEnv("APP_PORT", "8081"),
		DB: config.DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "task_db_test"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: config.RedisConfig{
			Host:          getEnv("REDIS_HOST", "localhost"),
			Port:          getEnv("REDIS_PORT", "6379"),
			RedisPassword: getEnv("REDIS_PASSWORD", ""),
			RedisDB:       getEnv("REDIS_DB", "0"),
		},
		RabbitMQ: config.RabbitMQConfig{
			URL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		},
		JWT: config.JWTConfig{
			Secret: getEnv("JWT_SECRET", "test-integration-secret"),
		},
	}
}

// setupTestDB initializes test database with schema
func setupTestDB(t *testing.T, cfg *config.Config) *sql.DB {
	t.Helper()

	database := db.Init(&cfg.DB)
	if database == nil {
		t.Fatal("Failed to setup test database")
	}

	// Run migrations
	if err := runMigrations(database); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return database
}

// cleanupTestDB removes all test data
func cleanupTestDB(t *testing.T, database *sql.DB) {
	t.Helper()

	// Truncate all tables
	tables := []string{"tasks", "users"}
	for _, table := range tables {
		_, err := database.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: Failed to truncate %s: %v", table, err)
		}
	}
}

// runMigrations applies database migrations
func runMigrations(database *sql.DB) error {
	// Create users table
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(createUsersTable); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create tasks table
	createTasksTable := `
	CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		task_type VARCHAR(50) NOT NULL,
		status VARCHAR(20) DEFAULT 'PENDING',
		result_file TEXT,
		error_message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(createTasksTable); err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	return nil
}

// setupTestRedis initializes test Redis client
func setupTestRedis(t *testing.T, cfg *config.Config) *redis.Client {
	t.Helper()

	client := cache.SetupRedis(&cfg.Redis)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to test Redis: %v", err)
	}

	// Flush test database
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("Failed to flush test Redis: %v", err)
	}

	return client
}

// setupTestRabbitMQ initializes test RabbitMQ connection
func setupTestRabbitMQ(t *testing.T, cfg *config.Config) *amqp.Connection {
	t.Helper()

	conn := queue.SetupRabbitMQ(&cfg.RabbitMQ)
	if conn == nil {
		t.Fatal("Failed to connect to test RabbitMQ")
	}

	// Declare and purge test queue
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	// Declare queue (idempotent)
	_, err = ch.QueueDeclare(
		"task_queue", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		t.Fatalf("Failed to declare queue: %v", err)
	}

	// Purge existing messages
	_, err = ch.QueuePurge("task_queue", false)
	if err != nil {
		log.Printf("Warning: Failed to purge queue: %v", err)
	}

	return conn
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// WaitForCondition polls until condition is true or timeout
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for: %s", message)
}
