package integration

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

// TestDB holds shared DB/Redis connections for all integration tests.
type TestDB struct {
	DB    *sql.DB
	Redis *redis.Client
}

var testEnv *TestDB

// getEnvOrDefault reads an environment variable, returning the default if unset.
func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// TestMain sets up shared infrastructure and tears it down after all tests.
func TestMain(m *testing.M) {
	db, err := setupPostgres()
	if err != nil {
		fmt.Printf("SKIP: could not connect to Postgres: %v\n", err)
		os.Exit(0)
	}

	rdb, err := setupRedis()
	if err != nil {
		fmt.Printf("SKIP: could not connect to Redis: %v\n", err)
		db.Close()
		os.Exit(0)
	}

	testEnv = &TestDB{DB: db, Redis: rdb}

	if err := runMigrations(db); err != nil {
		fmt.Printf("SKIP: could not run migrations: %v\n", err)
		os.Exit(0)
	}

	code := m.Run()

	db.Close()
	rdb.Close()
	os.Exit(code)
}

func setupPostgres() (*sql.DB, error) {
	dsn := getEnvOrDefault(
		"TEST_DATABASE_URL",
		"postgres://postgres:postgres@localhost:5432/task_handler_test?sslmode=disable",
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Retry a few times in case the DB is not ready yet
	for i := 0; i < 5; i++ {
		if err = db.Ping(); err == nil {
			return db, nil
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return nil, fmt.Errorf("postgres not reachable: %w", err)
}

func setupRedis() (*redis.Client, error) {
	addr := getEnvOrDefault("TEST_REDIS_ADDR", "localhost:6379")
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 15}) // DB 15 for tests

	ctx, cancel := withTimeout()
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis not reachable: %w", err)
	}
	return client, nil
}

// runMigrations creates the tables needed for testing.
func runMigrations(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id         SERIAL PRIMARY KEY,
			username   VARCHAR(100) UNIQUE NOT NULL,
			password   TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			task_id       SERIAL PRIMARY KEY,
			user_id       INTEGER NOT NULL REFERENCES users(id),
			task_type     VARCHAR(50) NOT NULL,
			status        VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			result_file   TEXT,
			error_message TEXT,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

// cleanupTables removes all test data between tests.
func cleanupTables(t *testing.T) {
	t.Helper()
	_, err := testEnv.DB.Exec(`TRUNCATE TABLE tasks, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	ctx, cancel := withTimeout()
	defer cancel()
	testEnv.Redis.FlushDB(ctx)
}
