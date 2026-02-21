package main

import (
	"task_handler/internal/config"
	"task_handler/internal/consumer/notification_consumer"
	"task_handler/internal/db"
	"task_handler/internal/domain/user"
	"task_handler/internal/logger"
	"task_handler/internal/queue"

	"github.com/sirupsen/logrus"
)

func main() {
	logger.InitLogger()

	cfg := config.Load()

	psqlDB := db.Init(&cfg.DB)
	defer func() {
		if err := psqlDB.Close(); err != nil {
			logrus.WithError(err).Fatal("Failed to closes database connection")
		}
	}()

	repo := user.NewUserRepository()

	manager := queue.SetupRabbitMQ(&cfg.RabbitMQ)
	defer func() {
		if err := manager.Close(); err != nil {
			logrus.WithError(err).Error("Failed to close RabbitMQ connection")
		}
	}()

	conn := manager.GetConnection()
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
		go notification_consumer.StartWorker(manager, psqlDB, repo, i+1, cfg)
	}

	select {}
}
