package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"task_handler/internal/config"
	"task_handler/internal/db"
	"task_handler/internal/handler"
	"task_handler/internal/logger"
	"task_handler/internal/queue"

	"github.com/sirupsen/logrus"
)

func main() {
	logger.InitLogger()

	config := config.Load()
	psqlDB := db.Init(&config.DB)

	defer func() {
		if err := psqlDB.Close(); err != nil {
			logrus.WithError(err).Fatal("Failed to close database connection")
		}
	}()

	rdb := db.SetupRedis(&config.Redis)
	defer func() {
		if err := rdb.Close(); err != nil {
			logrus.WithError(err).Fatal("Failed to close redis connection")
		}
	}()

	manager := queue.SetupRabbitMQ(&config.RabbitMQ)
	defer func() {
		if err := manager.Close(); err != nil {
			logrus.WithError(err).Fatal("Failed to close RabbitMQ connection")
		}
	}()

	conn := manager.GetConnection()
	r := handler.SetupHandler(psqlDB, conn, rdb, config)

	srv := &http.Server{
		Addr:    ":8087",
		Handler: r,
	}

	go func() {
		logrus.WithFields(logrus.Fields{
			"service": "api",
			"port":    "8087",
		}).Info("Starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.WithFields(logrus.Fields{
		"service": "api",
	}).Info("Shutting down server gracefully")
}
