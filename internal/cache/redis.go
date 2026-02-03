package cache

import (
	"context"
	"fmt"
	"strconv"
	"task_handler/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

func SetupRedis(redisCfg *config.RedisConfig) *redis.Client {
	addr := fmt.Sprintf("%s:%s", redisCfg.Host, redisCfg.Port)

	port, err := strconv.Atoi(redisCfg.RedisDB)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"service": "redis",
			"error":   err.Error(),
		}).Fatal("Invalid Redis DB number")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: redisCfg.RedisPassword, // Don't log password
		DB:       port,
	})

	// Test connection
	ctx := context.Background()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"service": "redis",
			"host":    redisCfg.Host,
			"port":    redisCfg.Port,
			"db":      port,
			"error":   err.Error(),
		}).Fatal("Failed to connect to Redis")
	}

	logrus.WithFields(logrus.Fields{
		"service": "redis",
		"host":    redisCfg.Host,
		"port":    redisCfg.Port,
		"db":      port,
	}).Info("Redis connection established successfully")

	return rdb
}
