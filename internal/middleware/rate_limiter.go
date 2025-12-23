package middleware

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

//go:embed rate_limiter.lua
var luaScript string

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	Capacity   int     // Maximum number of tokens (max requests)
	RefillRate float64 // Tokens refilled per second
}

// DefaultRateLimiterConfig returns default rate limiter settings
// 10 requests per second with burst capacity of 20
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		Capacity:   20,   // Can burst up to 20 requests
		RefillRate: 10.0, // Refills 10 tokens per second
	}
}

// RateLimiterMiddleware implements Token Bucket algorithm using Redis + Lua script
func RateLimiterMiddleware(redisClient *redis.Client, config *RateLimiterConfig) gin.HandlerFunc {
	// Load Lua script into Redis (SHA hash will be cached)
	ctx := context.Background()
	scriptSHA, err := redisClient.ScriptLoad(ctx, luaScript).Result()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load Lua script for rate limiter")
	}

	return func(c *gin.Context) {
		// Get user ID from JWT context
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized - user_id not found in context",
			})
			c.Abort()
			return
		}

		// Build rate limiter key
		key := UserRateLimiterKey(userID.(int))

		// Current timestamp
		now := time.Now().Unix()

		// Execute Lua script
		result, err := redisClient.EvalSha(ctx, scriptSHA, []string{key},
			config.Capacity,
			config.RefillRate,
			now,
		).Result()

		if err != nil {
			logrus.WithError(err).Error("Failed to execute rate limiter Lua script")
			// Fail open: allow request if Redis fails
			c.Next()
			return
		}

		// Check if request is allowed
		allowed := result.(int64)
		if allowed == 0 {
			// Rate limit exceeded
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     fmt.Sprintf("Maximum %d requests per second allowed", int(config.RefillRate)),
				"retry_after": fmt.Sprintf("%.1f seconds", 1.0/config.RefillRate),
			})
			c.Abort()
			return
		}

		// Request allowed, continue
		c.Next()
	}
}

// Build cache key for user rate limiting
func UserRateLimiterKey(userID int) string {
	return fmt.Sprintf("rate_limiter:user:%d", userID)
}
