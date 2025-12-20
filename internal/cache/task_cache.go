package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const TaskCacheTTL = 1 * time.Hour

type TaskCache struct {
	client *redis.Client
}

func NewTaskCache(client *redis.Client) *TaskCache {
	return &TaskCache{client: client}
}

// Get task from cache
func (c *TaskCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

// Set task to cache with TTL
func (c *TaskCache) Set(ctx context.Context, key string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, jsonData, TaskCacheTTL).Err()
}

// Build cache key for single task
func TaskKey(taskID int) string {
	return fmt.Sprintf("task:%d", taskID)
}

// Build cache key for user tasks
func UserTasksKey(userID int) string {
	return fmt.Sprintf("tasks:user:%d", userID)
}
