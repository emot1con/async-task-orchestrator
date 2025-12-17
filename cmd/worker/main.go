package main

import (
	"task_handler/internal/config"
	"task_handler/internal/db"
	"task_handler/internal/queue"
	"task_handler/internal/task"
	"task_handler/internal/worker"

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
	if err := consumerChannel.Close(); err != nil {
		logrus.WithError(err).Fatal("Failed to close RabbitMQ channel")
	}

	if _, err := queue.DeclareQueue(consumerChannel, "task_queue"); err != nil {
		logrus.WithError(err).Fatal("Failed to declare RabbitMQ queue")
	}

	for i := 1; i <= 3; i++ {
		go worker.StartWorker(conn, db, repo, i)
	}

	select {}
}
