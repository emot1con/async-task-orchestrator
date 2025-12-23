package middleware

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
