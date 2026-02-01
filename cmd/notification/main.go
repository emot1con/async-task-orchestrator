package main

import (
	"task_handler/internal/config"
	"task_handler/internal/consumer/notification_consumer"
	"task_handler/internal/db"
	"task_handler/internal/domain/user"
	"task_handler/internal/queue"

	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	cfg := config.Load()

	DB := db.Init(&cfg.DB)
	defer func() {
		if err := DB.Close(); err != nil {
			logrus.WithError(err).Fatal("Failed to close database connection")
		}
	}()

	repo := user.NewUserRepository()

	conn := queue.SetupRabbitMQ(&cfg.RabbitMQ)
	defer func() {
		if err := conn.Close(); err != nil {
			logrus.Fatalf("Failed to close RabbitMQ connection")
		}
	}()

	consumerChannel, err := queue.CreateChannel(conn)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to open RabbitMQ channel")
	}

	if _, err := queue.DeclareQueue(consumerChannel, "notification_queue"); err != nil {
		logrus.WithError(err).Fatal("Failed to declare RabbitMQ queue")
	}

	if err := consumerChannel.Close(); err != nil {
		logrus.WithError(err).Fatal("Failed to close RabbitMQ channel")
	}

	for i := 0; i < 3; i++ {
		go notification_consumer.StartWorker(conn, DB, repo, i+1, cfg)
	}

	select {}
}
