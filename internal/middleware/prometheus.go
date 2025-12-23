package middleware

import (
	"strconv"
	"time"

	"task_handler/internal/observability"

	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware adalah middleware untuk tracking HTTP metrics
func PrometheusMiddleware(metrics *observability.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Increment in-flight requests
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get request details
		method := c.Request.Method
		endpoint := c.FullPath() // e.g., /api/v1/tasks/:id
		if endpoint == "" {
			endpoint = c.Request.URL.Path // fallback ke path biasa jika tidak ada route pattern
		}
		status := strconv.Itoa(c.Writer.Status())

		// Record metrics
		metrics.HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	}
}
