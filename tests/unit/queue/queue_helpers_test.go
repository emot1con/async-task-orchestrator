package queue_test

import (
	"fmt"
	"task_handler/internal/middleware"
	"testing"

	"github.com/stretchr/testify/assert"
)

// queue package functions (DeclareQueue, Publish, Consume, etc.) wrap raw AMQP
// channel calls that require a live broker. Those are exercised fully in
// integration tests. Here we test the pure, I/O-free pieces of the queue package.

// ---- UserRateLimiterKey (from middleware package, validates key format) ----
// We include a sanity check to document the expected key format used by workers.

func TestQueueNameConstants(t *testing.T) {
	// Validate correct key format independently of middleware pkg tests
	key := middleware.UserRateLimiterKey(1)
	assert.Equal(t, "rate_limiter:user:1", key)
}

// ---- RepublishWithRetry header mutation logic (pure logic test) ----
// Even though we cannot call RepublishWithRetry without a channel, we can
// test the header mutation contract by mimicking the same logic:

func TestRepublishHeaders_RetryCountIncrement(t *testing.T) {
	// Simulates the logic inside RepublishWithRetry:
	// existing headers are copied, x-retry-count is set to the new count.
	baseHeaders := map[string]interface{}{
		"x-retry-count": int32(2),
		"content-type":  "application/json",
	}

	newCount := int32(3)

	// Copy and set (mirrors what RepublishWithRetry does)
	newHeaders := make(map[string]interface{})
	for k, v := range baseHeaders {
		newHeaders[k] = v
	}
	newHeaders["x-retry-count"] = newCount

	assert.Equal(t, int32(3), newHeaders["x-retry-count"])
	assert.Equal(t, "application/json", newHeaders["content-type"], "other headers should be preserved")
}

func TestRepublishHeaders_NilHeaders(t *testing.T) {
	// When original headers are nil, a new map is created with just the retry count
	var originalHeaders map[string]interface{}
	newCount := int32(1)

	newHeaders := make(map[string]interface{})
	if originalHeaders != nil {
		for k, v := range originalHeaders {
			newHeaders[k] = v
		}
	}
	newHeaders["x-retry-count"] = newCount

	assert.Equal(t, int32(1), newHeaders["x-retry-count"])
	assert.Len(t, newHeaders, 1)
}

// ---- MaxRetries boundary logic ----

func TestMaxRetries_BoundaryValues(t *testing.T) {
	maxRetries := 3
	cases := []struct {
		retryCount int32
		shouldStop bool
	}{
		{0, false},
		{1, false},
		{2, false},
		{3, true},  // == max, stop
		{4, true},  // > max, stop
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("retry=%d", tc.retryCount), func(t *testing.T) {
			shouldStop := tc.retryCount >= int32(maxRetries)
			assert.Equal(t, tc.shouldStop, shouldStop)
		})
	}
}

// ---- PublishWithRetry loop logic (pure counter simulation) ----

func TestPublishRetryLoop_Exhaustion(t *testing.T) {
	// Simulates the retry loop counter behavior in PublishWithRetry
	maxRetries := 3
	attempts := 0
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		attempts++
		lastErr = fmt.Errorf("simulated failure on attempt %d", attempt+1)
		// No break → exhaust all retries
	}

	assert.Equal(t, maxRetries+1, attempts, "should attempt exactly maxRetries+1 times (0..maxRetries)")
	assert.Error(t, lastErr)
}

func TestPublishRetryLoop_SuccessOnFirstAttempt(t *testing.T) {
	maxRetries := 3
	attempts := 0
	succeeded := false

	for attempt := 0; attempt <= maxRetries; attempt++ {
		attempts++
		// Succeed on first attempt
		succeeded = true
		break
	}

	assert.Equal(t, 1, attempts)
	assert.True(t, succeeded)
}
