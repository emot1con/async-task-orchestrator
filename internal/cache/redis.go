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
		logrus.Fatalf("Invalid Redis DB number: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: redisCfg.RedisPassword,
		DB:       port,
	})

	// Test connection
	ctx := context.Background()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		logrus.Fatalf("Failed to connect to Redis: %v", err)
	}

	return rdb
}
