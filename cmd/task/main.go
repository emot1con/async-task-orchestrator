package main

import (
	"task_handler/internal/cache"
	"task_handler/internal/config"
	"task_handler/internal/consumer/task_consumer"
	"task_handler/internal/db"
	"task_handler/internal/domain/task"
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
			logrus.WithError(err).Fatal("Failed to close database connection")
		}
	}()

	// Setup Redis for cache
	redisClient := db.SetupRedis(&cfg.Redis)
	defer func() {
		if err := redisClient.Close(); err != nil {
			logrus.WithError(err).Error("Failed to close Redis connection")
		}
	}()

	taskCache := cache.NewTaskCache(redisClient)

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

	for i := 1; i <= 3; i++ {
		go task_consumer.StartWorker(conn, psqlDB, repo, taskCache, i)
	}

	select {}
}
