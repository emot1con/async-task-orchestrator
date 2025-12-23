package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics adalah struct yang berisi semua custom metrics aplikasi kita
type Metrics struct {
	// HTTP Metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// Task Metrics
	TasksCreatedTotal      *prometheus.CounterVec
	TasksProcessedTotal    *prometheus.CounterVec
	TaskProcessingDuration *prometheus.HistogramVec
	TasksQueueSize         prometheus.Gauge
	TasksFailedTotal       *prometheus.CounterVec

	// Database Metrics
	DBConnectionsOpen  prometheus.Gauge
	DBConnectionsInUse prometheus.Gauge
	DBQueryDuration    *prometheus.HistogramVec

	// Cache (Redis) Metrics
	CacheHitsTotal   *prometheus.CounterVec
	CacheMissesTotal *prometheus.CounterVec

	// Queue (RabbitMQ) Metrics
	QueueMessagesPublished *prometheus.CounterVec
	QueueMessagesConsumed  *prometheus.CounterVec
}

// NewMetrics membuat instance baru dari Metrics dengan semua metric terdaftar
func NewMetrics() *Metrics {
	return &Metrics{
		// HTTP Metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
			},
			[]string{"method", "endpoint"},
		),

		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),

		// Task Metrics
		TasksCreatedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tasks_created_total",
				Help: "Total number of tasks created",
			},
			[]string{"task_type"},
		),

		TasksProcessedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tasks_processed_total",
				Help: "Total number of tasks processed",
			},
			[]string{"task_type", "status"}, // status: success, failed
		),

		TaskProcessingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "task_processing_duration_seconds",
				Help:    "Duration of task processing in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60}, // Custom buckets untuk task processing
			},
			[]string{"task_type"},
		),

		TasksQueueSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tasks_queue_size",
				Help: "Current number of tasks in the queue",
			},
		),

		TasksFailedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tasks_failed_total",
				Help: "Total number of tasks that failed processing",
			},
			[]string{"task_type", "error_type"},
		),

		// Database Metrics
		DBConnectionsOpen: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_open",
				Help: "Number of open database connections",
			},
		),

		DBConnectionsInUse: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_in_use",
				Help: "Number of database connections currently in use",
			},
		),

		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Duration of database queries in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"query_type"}, // SELECT, INSERT, UPDATE, DELETE
		),

		// Cache Metrics
		CacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"key_type"},
		),

		CacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"key_type"},
		),

		// Queue Metrics
		QueueMessagesPublished: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_messages_published_total",
				Help: "Total number of messages published to the queue",
			},
			[]string{"queue_name"},
		),

		QueueMessagesConsumed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_messages_consumed_total",
				Help: "Total number of messages consumed from the queue",
			},
			[]string{"queue_name"},
		),
	}
}

// GlobalMetrics adalah instance global dari Metrics yang bisa digunakan di seluruh aplikasi
var GlobalMetrics *Metrics

// InitMetrics menginisialisasi global metrics
func InitMetrics() {
	GlobalMetrics = NewMetrics()
}
