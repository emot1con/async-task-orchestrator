package main

import (
	"net/http"
	"task_handler/internal/config"
	"task_handler/internal/db"
	"task_handler/internal/observability"
	"task_handler/internal/queue"
	"task_handler/internal/task"
	"task_handler/internal/worker"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	cfg := config.Load()

	db := db.Init(&cfg.DB)
	defer func() {
		if err := db.Close(); err != nil {
			logrus.WithError(err).Fatal("Failed to close database connection")
		}
	}()

	conn := queue.SetupRabbitMQ(&cfg.RabbitMQ)
	defer func() {
		if err := conn.Close(); err != nil {
			logrus.Fatalf("Failed to close RabbitMQ connection")
		}
	}()

	repo := task.NewTaskRepository()

	consumerChannel, err := queue.CreateChannel(conn)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create RabbitMQ channel")
	}

	if _, err := queue.DeclareQueue(consumerChannel, "task_queue"); err != nil {
		logrus.WithError(err).Fatal("Failed to declare RabbitMQ queue")
	}

	if err := consumerChannel.Close(); err != nil {
		logrus.WithError(err).Fatal("Failed to close RabbitMQ channel")
	}

	// Initialize Prometheus metrics
	observability.InitMetrics()
	logrus.Info("Metrics initialized")

	// Start metrics HTTP server for Prometheus scraping
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logrus.Info("Worker metrics server started on :8088")
		if err := http.ListenAndServe(":8088", nil); err != nil {
			logrus.WithError(err).Fatal("Failed to start metrics server")
		}
	}()

	for i := 1; i <= 3; i++ {
		go worker.StartWorker(conn, db, repo, i)
	}

	select {}
}
