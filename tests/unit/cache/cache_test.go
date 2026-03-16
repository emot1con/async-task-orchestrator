package cache_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"task_handler/internal/cache"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRedis spins up an in-memory Redis server and returns a client connected to it.
func setupRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return client, mr
}

// ---- Key Builder Tests (pure, no I/O) ----

func TestTaskKey(t *testing.T) {
	assert.Equal(t, "task:1", cache.TaskKey(1))
	assert.Equal(t, "task:42", cache.TaskKey(42))
	assert.Equal(t, "task:0", cache.TaskKey(0))
}

func TestUserTasksKey(t *testing.T) {
	key := cache.UserTasksKey(7, 15, 0)
	assert.Equal(t, "tasks:user:7:limit:15,offset:0", key)

	key2 := cache.UserTasksKey(1, 10, 20)
	assert.Equal(t, "tasks:user:1:limit:10,offset:20", key2)
}

func TestCacheTTLConstants(t *testing.T) {
	assert.Equal(t, 5*time.Minute, cache.TasksCacheTTL)
	assert.Equal(t, 5*time.Second, cache.TaskCacheTTL)
}

// ---- TaskCache.SetTask / GetTask ----

func TestTaskCache_SetAndGetTask(t *testing.T) {
	client, _ := setupRedis(t)
	tc := cache.NewTaskCache(client)
	ctx := context.Background()

	type Payload struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	data := Payload{ID: 1, Name: "test"}

	err := tc.SetTask(ctx, "task:1", data)
	require.NoError(t, err)

	raw, err := tc.GetTask(ctx, "task:1")
	require.NoError(t, err)
	require.NotNil(t, raw)

	var got Payload
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, data, got)
}

func TestTaskCache_GetTask_Miss(t *testing.T) {
	client, _ := setupRedis(t)
	tc := cache.NewTaskCache(client)
	ctx := context.Background()

	raw, err := tc.GetTask(ctx, "task:nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, raw, "cache miss should return nil, nil")
}

// ---- TaskCache.SetTasks / GetTasks ----

func TestTaskCache_SetAndGetTasks(t *testing.T) {
	client, _ := setupRedis(t)
	tc := cache.NewTaskCache(client)
	ctx := context.Background()

	type Item struct{ ID int }
	items := []Item{{1}, {2}, {3}}

	err := tc.SetTasks(ctx, "tasks:user:1:limit:15,offset:0", items)
	require.NoError(t, err)

	raw, err := tc.GetTasks(ctx, "tasks:user:1:limit:15,offset:0")
	require.NoError(t, err)
	require.NotNil(t, raw)

	var got []Item
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, items, got)
}

func TestTaskCache_GetTasks_Miss(t *testing.T) {
	client, _ := setupRedis(t)
	tc := cache.NewTaskCache(client)
	ctx := context.Background()

	raw, err := tc.GetTasks(ctx, "tasks:user:999:limit:15,offset:0")
	assert.NoError(t, err)
	assert.Nil(t, raw)
}

// ---- DeleteCacheByPattern ----

func TestTaskCache_DeleteCacheByPattern(t *testing.T) {
	client, _ := setupRedis(t)
	tc := cache.NewTaskCache(client)
	ctx := context.Background()

	// Set multiple keys
	_ = tc.SetTasks(ctx, "tasks:user:5:limit:15,offset:0", []int{1})
	_ = tc.SetTasks(ctx, "tasks:user:5:limit:10,offset:0", []int{2})
	_ = tc.SetTask(ctx, "task:99", map[string]int{"id": 99})

	// Delete only user:5 pattern
	err := tc.DeleteCacheByPattern(ctx, "tasks:user:5*")
	require.NoError(t, err)

	r1, _ := tc.GetTasks(ctx, "tasks:user:5:limit:15,offset:0")
	r2, _ := tc.GetTasks(ctx, "tasks:user:5:limit:10,offset:0")
	r3, _ := tc.GetTask(ctx, "task:99")

	assert.Nil(t, r1, "user 5 tasks should be deleted")
	assert.Nil(t, r2, "user 5 tasks should be deleted")
	assert.NotNil(t, r3, "task:99 should still exist")
}

func TestTaskCache_DeleteCacheByPattern_NoKeys(t *testing.T) {
	client, _ := setupRedis(t)
	tc := cache.NewTaskCache(client)
	ctx := context.Background()

	// No keys match → should be a no-op, no error
	err := tc.DeleteCacheByPattern(ctx, "tasks:user:999*")
	assert.NoError(t, err)
}

// ---- DeleteTaskCache ----

func TestTaskCache_DeleteTaskCache(t *testing.T) {
	client, _ := setupRedis(t)
	tc := cache.NewTaskCache(client)
	ctx := context.Background()

	// Seed a task key and a user tasks key
	_ = tc.SetTask(ctx, cache.TaskKey(10), map[string]int{"id": 10})
	_ = tc.SetTasks(ctx, cache.UserTasksKey(3, 15, 0), []int{10})

	err := tc.DeleteTaskCache(ctx, 10, 3)
	require.NoError(t, err)

	r1, _ := tc.GetTask(ctx, cache.TaskKey(10))
	r2, _ := tc.GetTasks(ctx, cache.UserTasksKey(3, 15, 0))
	assert.Nil(t, r1, "single task cache should be deleted")
	assert.Nil(t, r2, "user tasks cache should be deleted")
}
