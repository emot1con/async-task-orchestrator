package middleware_test

import (
	"fmt"
	"task_handler/internal/middleware"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Config Factory Tests ----

func TestDefaultRateLimiterConfig(t *testing.T) {
	cfg := middleware.DefaultRateLimiterConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, 20, cfg.Capacity)
	assert.Equal(t, 10.0, cfg.RefillRate)
}

func TestStrictRateLimiter(t *testing.T) {
	cfg := middleware.StrictRateLimiter()

	require.NotNil(t, cfg)
	assert.Equal(t, 3, cfg.Capacity)
	assert.Equal(t, 0.1, cfg.RefillRate)
}

func TestConservativeRateLimiter(t *testing.T) {
	cfg := middleware.ConservativeRateLimiter()

	require.NotNil(t, cfg)
	assert.Equal(t, 10, cfg.Capacity)
	assert.Equal(t, 5.0, cfg.RefillRate)
}

func TestModerateRateLimiter(t *testing.T) {
	cfg := middleware.ModerateRateLimiter()
	def := middleware.DefaultRateLimiterConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, def.Capacity, cfg.Capacity)
	assert.Equal(t, def.RefillRate, cfg.RefillRate)
}

func TestGenerousRateLimiter(t *testing.T) {
	cfg := middleware.GenerousRateLimiter()

	require.NotNil(t, cfg)
	assert.Equal(t, 100, cfg.Capacity)
	assert.Equal(t, 50.0, cfg.RefillRate)
}

func TestUnlimitedRateLimiter(t *testing.T) {
	cfg := middleware.UnlimitedRateLimiter()

	require.NotNil(t, cfg)
	assert.Equal(t, 10000, cfg.Capacity)
	assert.Equal(t, 1000.0, cfg.RefillRate)
}

func TestCustomRateLimiter(t *testing.T) {
	cfg := middleware.CustomRateLimiter(42, 7.5)

	require.NotNil(t, cfg)
	assert.Equal(t, 42, cfg.Capacity)
	assert.Equal(t, 7.5, cfg.RefillRate)
}

func TestCustomRateLimiter_ZeroValues(t *testing.T) {
	cfg := middleware.CustomRateLimiter(0, 0)

	require.NotNil(t, cfg)
	assert.Equal(t, 0, cfg.Capacity)
	assert.Equal(t, 0.0, cfg.RefillRate)
}

// ---- UserRateLimiterKey Tests ----

func TestUserRateLimiterKey(t *testing.T) {
	cases := []struct {
		userID   int
		expected string
	}{
		{1, "rate_limiter:user:1"},
		{42, "rate_limiter:user:42"},
		{0, "rate_limiter:user:0"},
		{999999, "rate_limiter:user:999999"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("userID=%d", tc.userID), func(t *testing.T) {
			key := middleware.UserRateLimiterKey(tc.userID)
			assert.Equal(t, tc.expected, key)
		})
	}
}

// ---- Struct Field Tests ----

func TestRateLimiterConfig_Fields(t *testing.T) {
	cfg := &middleware.RateLimiterConfig{
		Capacity:   5,
		RefillRate: 2.5,
	}
	assert.Equal(t, 5, cfg.Capacity)
	assert.Equal(t, 2.5, cfg.RefillRate)
}
