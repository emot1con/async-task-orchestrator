//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"task_handler/internal/cache"
	"task_handler/internal/config"
	"task_handler/internal/db"
	"task_handler/internal/queue"

	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
)

// TestEnv holds all test dependencies
type TestEnv struct {
	DB          *sql.DB
	RedisClient *redis.Client
	RabbitConn  *amqp.Connection
	Config      *config.Config
}

// SetupTestEnv initializes test environment
func SetupTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	cfg := loadTestConfig()

	// Setup database
	database := db.Init(&cfg.DB)
	if database == nil {
		t.Fatal("Failed to connect to test database")
	}

	// Run schema migrations
	if err := runMigrations(database); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Setup Redis
	redisClient := cache.SetupRedis(&cfg.Redis)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	redisClient.FlushDB(ctx)

	// Setup RabbitMQ
	rabbitConn := queue.SetupRabbitMQ(&cfg.RabbitMQ)
	if rabbitConn == nil {
		t.Fatal("Failed to connect to RabbitMQ")
	}

	// Declare and purge queue
	ch, err := rabbitConn.Channel()
	if err != nil {
		t.Fatalf("Failed to open channel: %v", err)
	}
	_, err = ch.QueueDeclare("task_queue", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("Failed to declare queue: %v", err)
	}
	ch.QueuePurge("task_queue", false)
	ch.Close()

	return &TestEnv{
		DB:          database,
		RedisClient: redisClient,
		RabbitConn:  rabbitConn,
		Config:      cfg,
	}
}

// Cleanup cleans up test environment
func (env *TestEnv) Cleanup(t *testing.T) {
	t.Helper()

	if env.DB != nil {
		env.DB.Exec("TRUNCATE TABLE tasks CASCADE")
		env.DB.Exec("TRUNCATE TABLE users CASCADE")
		env.DB.Close()
	}

	if env.RedisClient != nil {
		env.RedisClient.FlushDB(context.Background())
		env.RedisClient.Close()
	}

	if env.RabbitConn != nil {
		if ch, err := env.RabbitConn.Channel(); err == nil {
			ch.QueuePurge("task_queue", false)
			ch.Close()
		}
		env.RabbitConn.Close()
	}
}

// loadTestConfig loads test configuration with defaults
func loadTestConfig() *config.Config {
	return &config.Config{
		AppName: "integration-test",
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
			Secret: getEnv("JWT_SECRET", "test-secret-key-for-integration"),
		},
	}
}

// runMigrations creates database schema
func runMigrations(database *sql.DB) error {
	_, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS users (
id SERIAL PRIMARY KEY,
username VARCHAR(255) UNIQUE NOT NULL,
password VARCHAR(255) NOT NULL,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	_, err = database.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
id SERIAL PRIMARY KEY,
user_id INTEGER NOT NULL REFERENCES users(id),
task_type VARCHAR(50) NOT NULL,
status VARCHAR(20) DEFAULT 'PENDING',
result_file TEXT,
error_message TEXT,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
	`)
	if err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
