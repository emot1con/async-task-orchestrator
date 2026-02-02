package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const TasksCacheTTL = 5 * time.Minute
const TaskCacheTTL = 5 * time.Second

type TaskCache struct {
	client *redis.Client
}

func NewTaskCache(client *redis.Client) *TaskCache {
	return &TaskCache{client: client}
}

// GetTask task from cache
func (c *TaskCache) GetTask(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

// SetTask task to cache with TTL
func (c *TaskCache) SetTask(ctx context.Context, key string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, jsonData, TaskCacheTTL).Err()
}

// Get task from cache
func (c *TaskCache) GetTasks(ctx context.Context, key string) ([]byte, error) {
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
func (c *TaskCache) SetTasks(ctx context.Context, key string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, jsonData, TasksCacheTTL).Err()
}

func (c *TaskCache) DeleteCacheByPattern(ctx context.Context, key string) error {
	var cursor uint64
	var keys []string

	for {
		var err error
		var scanKeys []string

		scanKeys, cursor, err = c.client.Scan(ctx, cursor, key, 100).Result()
		if err != nil {
			return err
		}
		keys = append(keys, scanKeys...)
		if cursor == 0 {
			break
		}
	}

	// Skip deletion if no keys found
	if len(keys) == 0 {
		return nil
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return err
	}

	return nil
}

// DeleteTaskCache deletes single task cache and all related user tasks cache
func (c *TaskCache) DeleteTaskCache(ctx context.Context, taskID, userID int) error {
	// Delete single task cache
	taskKey := TaskKey(taskID)
	if err := c.client.Del(ctx, taskKey).Err(); err != nil {
		return err
	}

	// Delete all user tasks cache (with pattern matching)
	userTasksPattern := fmt.Sprintf("tasks:user:%d*", userID)
	return c.DeleteCacheByPattern(ctx, userTasksPattern)
}

// Build cache key for single task
func TaskKey(taskID int) string {
	return fmt.Sprintf("task:%d", taskID)
}

// Build cache key for user tasks
func UserTasksKey(userID, limit, offset int) string {
	return fmt.Sprintf("tasks:user:%d:limit:%d,offset:%d", userID, limit, offset)
}
