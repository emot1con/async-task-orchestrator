package db

import (
	"database/sql"
	"fmt"
	"log"
	"task_handler/internal/config"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Init(DBCfg *config.DBConfig) *sql.DB {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", DBCfg.Host, DBCfg.Port, DBCfg.User, DBCfg.Password, DBCfg.Name, DBCfg.SSLMode)

	var db *sql.DB
	var err error

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("pgx", dsn)
		if err != nil {
			log.Printf("Failed to open database connection (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		if err = db.Ping(); err != nil {
			log.Printf("Failed to ping database (attempt %d/%d): %v", i+1, maxRetries, err)
			if err := db.Close(); err != nil {
				log.Printf("Failed to close database connection: %v", err)
			}
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		// Connection successful
		break
	}

	if err != nil {
		log.Fatalf("Failed to connect to database after %d attempts: %v", maxRetries, err)
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	log.Println("Database connection established successfully")
	return db
}
