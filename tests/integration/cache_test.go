package integration

import (
	"context"
	"encoding/json"
	"task_handler/internal/cache"
	"task_handler/internal/domain/task"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_TaskCache_SetAndGet(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testEnv.Redis.FlushDB(ctx)

	c := cache.NewTaskCache(testEnv.Redis)
	key := cache.TaskKey(1)

	tk := &task.Task{TaskID: 1, UserID: 2, TaskType: "generate_report", Status: "PENDING"}

	err := c.SetTask(ctx, key, tk)
	require.NoError(t, err)

	data, err := c.GetTask(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, data)

	var result task.Task
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, 1, result.TaskID)
	assert.Equal(t, "generate_report", result.TaskType)
}

func TestIntegration_TaskCache_Miss(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testEnv.Redis.FlushDB(ctx)

	c := cache.NewTaskCache(testEnv.Redis)

	data, err := c.GetTask(ctx, "task:nonexistent:9999")
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestIntegration_TaskCache_SetAndGetTasks(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testEnv.Redis.FlushDB(ctx)

	c := cache.NewTaskCache(testEnv.Redis)
	key := cache.UserTasksKey(5, 10, 0)

	tasks := []*task.Task{
		{TaskID: 1, UserID: 5, TaskType: "send_email", Status: "PENDING"},
		{TaskID: 2, UserID: 5, TaskType: "resize_image", Status: "SUCCESS"},
	}

	err := c.SetTasks(ctx, key, tasks)
	require.NoError(t, err)

	data, err := c.GetTasks(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, data)

	var result []*task.Task
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Len(t, result, 2)
}

func TestIntegration_TaskCache_DeleteCacheByPattern(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testEnv.Redis.FlushDB(ctx)

	c := cache.NewTaskCache(testEnv.Redis)

	// Set multiple cache entries for user 7
	for _, key := range []string{
		"tasks:user:7:limit:10,offset:0",
		"tasks:user:7:limit:10,offset:10",
		"tasks:user:7:limit:5,offset:0",
	} {
		err := c.SetTasks(ctx, key, []*task.Task{})
		require.NoError(t, err)
	}

	// Set a key for a different user (should NOT be deleted)
	err := c.SetTasks(ctx, "tasks:user:8:limit:10,offset:0", []*task.Task{})
	require.NoError(t, err)

	// Delete all keys for user 7
	err = c.DeleteCacheByPattern(ctx, "tasks:user:7*")
	require.NoError(t, err)

	// User 7 cache should be gone
	for _, key := range []string{
		"tasks:user:7:limit:10,offset:0",
		"tasks:user:7:limit:10,offset:10",
		"tasks:user:7:limit:5,offset:0",
	} {
		data, err := c.GetTasks(ctx, key)
		require.NoError(t, err)
		assert.Nil(t, data, "key %s should be deleted", key)
	}

	// User 8 cache should still exist
	data, err := c.GetTasks(ctx, "tasks:user:8:limit:10,offset:0")
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestIntegration_TaskCache_TTL(t *testing.T) {
	if testEnv == nil {
		t.Skip("integration env not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	testEnv.Redis.FlushDB(ctx)

	c := cache.NewTaskCache(testEnv.Redis)
	key := cache.TaskKey(999)

	err := c.SetTask(ctx, key, &task.Task{TaskID: 999, Status: "PENDING"})
	require.NoError(t, err)

	// Should exist right after setting
	data, err := c.GetTask(ctx, key)
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Verify TTL is set (should be <= TaskCacheTTL)
	ttl, err := testEnv.Redis.TTL(ctx, key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl.Seconds(), float64(0), "TTL should be positive")
	assert.LessOrEqual(t, ttl.Seconds(), cache.TaskCacheTTL.Seconds())
}

func TestIntegration_CacheKeys(t *testing.T) {
	assert.Equal(t, "task:1", cache.TaskKey(1))
	assert.Equal(t, "task:42", cache.TaskKey(42))
	assert.Equal(t, "tasks:user:5:limit:10,offset:0", cache.UserTasksKey(5, 10, 0))
	assert.Equal(t, "tasks:user:3:limit:15,offset:15", cache.UserTasksKey(3, 15, 15))
}
