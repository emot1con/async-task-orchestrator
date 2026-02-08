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

		key := UserRateLimiterKey(userID.(int))
		now := time.Now().Unix()
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

// RateLimiterPresets provides common rate limiting configurations

// StrictRateLimiter - For sensitive endpoints (login, password reset)
// Burst: 3 requests, Sustained: 1 request per 10 seconds
func StrictRateLimiter() *RateLimiterConfig {
	return &RateLimiterConfig{
		Capacity:   3,
		RefillRate: 0.1, // 1 request per 10 seconds
	}
}

// ConservativeRateLimiter - For production API endpoints
// Burst: 10 requests, Sustained: 5 requests per second
func ConservativeRateLimiter() *RateLimiterConfig {
	return &RateLimiterConfig{
		Capacity:   10,
		RefillRate: 5.0,
	}
}

// ModerateRateLimiter - Default configuration (alias for DefaultRateLimiterConfig)
// Burst: 20 requests, Sustained: 10 requests per second
func ModerateRateLimiter() *RateLimiterConfig {
	return DefaultRateLimiterConfig()
}

// GenerousRateLimiter - For read-heavy endpoints
// Burst: 100 requests, Sustained: 50 requests per second
func GenerousRateLimiter() *RateLimiterConfig {
	return &RateLimiterConfig{
		Capacity:   100,
		RefillRate: 50.0,
	}
}

// UnlimitedRateLimiter - For internal/admin endpoints (development only)
// Burst: 10000 requests, Sustained: 1000 requests per second
func UnlimitedRateLimiter() *RateLimiterConfig {
	return &RateLimiterConfig{
		Capacity:   10000,
		RefillRate: 1000.0,
	}
}

// CustomRateLimiter - Create your own configuration
// Example: CustomRateLimiter(5, 2.0) = 5 burst, 2 req/sec
func CustomRateLimiter(capacity int, refillRate float64) *RateLimiterConfig {
	return &RateLimiterConfig{
		Capacity:   capacity,
		RefillRate: refillRate,
	}
}
