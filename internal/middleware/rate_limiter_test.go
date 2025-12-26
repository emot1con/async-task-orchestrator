package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a Redis client for testing
// Make sure Redis is running on localhost:6379
func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use DB 1 for tests (not default DB 0)
	})

	// Test connection
	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}

	// Clean up test keys
	client.FlushDB(ctx)

	return client
}

// setupTestRouter creates a test Gin router with rate limiter
func setupTestRouter(redisClient *redis.Client, config *RateLimiterConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock JWT middleware - sets userID in context
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1) // Mock user ID
		c.Next()
	})

	router.Use(RateLimiterMiddleware(redisClient, config))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	return router
}

func TestRateLimiter_AllowRequestsUnderLimit(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	config := &RateLimiterConfig{
		Capacity:   5,
		RefillRate: 10.0, // 10 tokens per second
	}

	router := setupTestRouter(redisClient, config)

	// Should allow 5 requests (capacity)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}
}

func TestRateLimiter_DenyRequestsOverLimit(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	config := &RateLimiterConfig{
		Capacity:   3,
		RefillRate: 1.0, // 1 token per second
	}

	router := setupTestRouter(redisClient, config)

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request should be rate limited")
	assert.Contains(t, w.Body.String(), "Rate limit exceeded")
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	config := &RateLimiterConfig{
		Capacity:   2,
		RefillRate: 2.0, // 2 tokens per second = 1 token per 0.5 seconds
	}

	router := setupTestRouter(redisClient, config)

	// Use up all tokens
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Next request should be denied
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Wait for tokens to refill (1 second should give us 2 tokens: rate=2 tokens/sec)
	time.Sleep(1 * time.Second)

	// Request should succeed after refill
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Request should succeed after token refill")
}

func TestRateLimiter_DifferentUsers(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	config := &RateLimiterConfig{
		Capacity:   2,
		RefillRate: 1.0,
	}

	gin.SetMode(gin.TestMode)

	// User 1 router
	router1 := gin.New()
	router1.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Next()
	})
	router1.Use(RateLimiterMiddleware(redisClient, config))
	router1.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// User 2 router
	router2 := gin.New()
	router2.Use(func(c *gin.Context) {
		c.Set("userID", 2)
		c.Next()
	})
	router2.Use(RateLimiterMiddleware(redisClient, config))
	router2.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// User 1: Use all tokens
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router1.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// User 1: Should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router1.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// User 2: Should still be able to make requests
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	router2.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "User 2 should not be affected by User 1's rate limit")
}

func TestRateLimiter_NoUserIDInContext(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	config := DefaultRateLimiterConfig()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// No JWT middleware - userID not set
	router.Use(RateLimiterMiddleware(redisClient, config))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "user_id not found in context")
}

func TestRateLimiter_RedisFailure_FailOpen(t *testing.T) {
	// Use invalid Redis connection to simulate failure
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:9999", // Non-existent Redis
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	// config := DefaultRateLimiterConfig()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Next()
	})

	// This will fail to load Lua script, should panic/fatal
	// In production, this is caught during startup
	// For runtime Redis failures after script load, test below

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Note: Cannot fully test fail-open without complex Redis mock
	// The fail-open logic is in the EvalSha error handling
}

func TestUserRateLimiterKey(t *testing.T) {
	tests := []struct {
		name     string
		userID   int
		expected string
	}{
		{
			name:     "User ID 1",
			userID:   1,
			expected: "rate_limiter:user:1",
		},
		{
			name:     "User ID 100",
			userID:   100,
			expected: "rate_limiter:user:100",
		},
		{
			name:     "User ID 0",
			userID:   0,
			expected: "rate_limiter:user:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UserRateLimiterKey(tt.userID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultRateLimiterConfig(t *testing.T) {
	config := DefaultRateLimiterConfig()

	require.NotNil(t, config)
	assert.Equal(t, 20, config.Capacity)
	assert.Equal(t, 10.0, config.RefillRate)
}

func TestRateLimiter_BurstCapacity(t *testing.T) {
	redisClient := setupTestRedis(t)
	defer redisClient.Close()

	config := &RateLimiterConfig{
		Capacity:   10,  // Can burst 10 requests
		RefillRate: 1.0, // But only refills 1 per second
	}

	router := setupTestRouter(redisClient, config)

	// Should allow 10 burst requests immediately
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Burst request %d should succeed", i+1)
	}

	// 11th request should fail
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request beyond burst should fail")
}

// Benchmark rate limiter performance
func BenchmarkRateLimiter(b *testing.B) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	})
	defer redisClient.Close()

	ctx := context.Background()
	redisClient.FlushDB(ctx)

	config := &RateLimiterConfig{
		Capacity:   1000,
		RefillRate: 100.0,
	}

	router := setupTestRouter(redisClient, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
