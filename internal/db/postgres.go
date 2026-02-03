package db

import (
	"database/sql"
	"fmt"
	"task_handler/internal/config"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"
)

func Init(DBCfg *config.DBConfig) *sql.DB {
	// Don't log full DSN (contains password)
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", DBCfg.Host, DBCfg.Port, DBCfg.User, DBCfg.Password, DBCfg.Name, DBCfg.SSLMode)

	var db *sql.DB
	var err error

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("pgx", dsn)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"service":     "database",
				"attempt":     i + 1,
				"max_retries": maxRetries,
				"host":        DBCfg.Host,
				"database":    DBCfg.Name,
				"error":       err.Error(),
			}).Warn("Failed to open database connection")
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		if err = db.Ping(); err != nil {
			logrus.WithFields(logrus.Fields{
				"service":     "database",
				"attempt":     i + 1,
				"max_retries": maxRetries,
				"host":        DBCfg.Host,
				"database":    DBCfg.Name,
				"error":       err.Error(),
			}).Warn("Failed to ping database")
			if err := db.Close(); err != nil {
				logrus.WithFields(logrus.Fields{
					"service": "database",
					"error":   err.Error(),
				}).Error("Failed to close database connection")
			}
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		// Connection successful
		break
	}

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"service":     "database",
			"max_retries": maxRetries,
			"host":        DBCfg.Host,
			"database":    DBCfg.Name,
			"error":       err.Error(),
		}).Fatal("Failed to connect to database after retries")
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	logrus.WithFields(logrus.Fields{
		"service":  "database",
		"host":     DBCfg.Host,
		"database": DBCfg.Name,
	}).Info("Database connection established successfully")
	return db
}
