# 📊 Metrics Reference

## 📖 **Penjelasan Jenis Metrics**

### **1. Counter** 🔢
- **Karakteristik**: Nilai **selalu naik**, tidak pernah turun
- **Reset**: Hanya reset ke 0 saat aplikasi restart
- **Use Case**: Menghitung events yang terjadi

**Contoh:**
```go
http_requests_total         // Total HTTP requests (bertambah terus)
tasks_created_total         // Total tasks dibuat
tasks_failed_total          // Total tasks gagal
```

**Query Examples:**
```promql
# Total requests sepanjang waktu
http_requests_total

# Request rate (per second)
rate(http_requests_total[5m])

# Total requests in last 1 hour
increase(http_requests_total[1h])
```

---

### **2. Gauge** 📏
- **Karakteristik**: Nilai bisa **naik dan turun**
- **Use Case**: Mengukur nilai "saat ini"

**Contoh:**
```go
http_requests_in_flight     // Request yang sedang diproses SEKARANG
tasks_queue_size            // Task di queue SEKARANG
go_goroutines               // Goroutines SEKARANG
go_memstats_alloc_bytes     // Memory usage SEKARANG
```

**Query Examples:**
```promql
# Current value
http_requests_in_flight

# Average in last 5 minutes
avg_over_time(http_requests_in_flight[5m])

# Max in last 1 hour
max_over_time(tasks_queue_size[1h])
```

---

### **3. Histogram** 📊
- **Karakteristik**: Distribusi nilai dalam **buckets**
- **Use Case**: Mengukur durasi, size, dll dengan percentiles
- **Output**: `_bucket`, `_sum`, `_count`

**Contoh:**
```go
http_request_duration_seconds     // Response time distribution
task_processing_duration_seconds  // Processing time distribution
```

**Buckets Example:**
```
http_request_duration_seconds_bucket{le="0.005"} 10   # 10 requests < 5ms
http_request_duration_seconds_bucket{le="0.01"}  25   # 25 requests < 10ms
http_request_duration_seconds_bucket{le="0.025"} 50   # 50 requests < 25ms
http_request_duration_seconds_bucket{le="0.05"}  80   # 80 requests < 50ms
http_request_duration_seconds_bucket{le="0.1"}   95   # 95 requests < 100ms
http_request_duration_seconds_bucket{le="+Inf"}  100  # All requests
http_request_duration_seconds_sum 5.234               # Total time
http_request_duration_seconds_count 100               # Total requests
```

**Query Examples:**
```promql
# Average duration
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])

# p50 (median) - 50% requests faster than this
histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))

# p95 - 95% requests faster than this
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# p99 - 99% requests faster than this (only 1% slower)
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
```

---

## 🎯 **Custom Metrics in Task Handler**

### **HTTP Metrics** (API Server)

#### **`http_requests_total`** (Counter)
```
Type: Counter
Labels: method, endpoint, status
Description: Total number of HTTP requests
```

**Example Values:**
```
http_requests_total{method="POST", endpoint="/api/v1/tasks", status="201"} 1234
http_requests_total{method="GET", endpoint="/api/v1/tasks/:id", status="200"} 567
http_requests_total{method="POST", endpoint="/auth/login", status="200"} 89
http_requests_total{method="POST", endpoint="/api/v1/tasks", status="500"} 12
```

**Use Case:**
- Monitor API traffic
- Track endpoint popularity
- Calculate error rate

**Queries:**
```promql
# Total requests per second
sum(rate(http_requests_total[5m]))

# Requests by endpoint
sum by (endpoint) (rate(http_requests_total[5m]))

# Error rate (5xx)
sum(rate(http_requests_total{status=~"5.."}[5m]))

# Success rate
sum(rate(http_requests_total{status=~"2.."}[5m])) / sum(rate(http_requests_total[5m])) * 100
```

---

#### **`http_request_duration_seconds`** (Histogram)
```
Type: Histogram
Labels: method, endpoint
Description: Duration of HTTP requests in seconds
Buckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
```

**Use Case:**
- Monitor API response time
- Identify slow endpoints
- SLA monitoring

**Queries:**
```promql
# Average response time
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])

# p95 latency (95% requests faster than this)
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# p99 latency
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# Slowest endpoint (p95)
topk(5, histogram_quantile(0.95, sum by (endpoint) (rate(http_request_duration_seconds_bucket[5m]))))
```

---

#### **`http_requests_in_flight`** (Gauge)
```
Type: Gauge
Labels: -
Description: Number of HTTP requests currently being processed
```

**Use Case:**
- Monitor current load
- Detect traffic spikes
- Identify stuck requests

**Queries:**
```promql
# Current in-flight requests
http_requests_in_flight

# Average concurrent requests
avg_over_time(http_requests_in_flight[5m])

# Peak concurrent requests
max_over_time(http_requests_in_flight[1h])
```

---

### **Task Metrics** (Worker)

#### **`tasks_created_total`** (Counter)
```
Type: Counter
Labels: task_type
Description: Total number of tasks created
```

**Example Values:**
```
tasks_created_total{task_type="email"} 5000
tasks_created_total{task_type="report"} 1200
tasks_created_total{task_type="notification"} 8900
```

**Queries:**
```promql
# Task creation rate
rate(tasks_created_total[5m])

# Total tasks created (last 1 hour)
increase(tasks_created_total[1h])

# Most created task type
topk(5, rate(tasks_created_total[5m]))
```

---

#### **`tasks_processed_total`** (Counter)
```
Type: Counter
Labels: task_type, status
Description: Total number of tasks processed
Status: success, failed
```

**Example Values:**
```
tasks_processed_total{task_type="email", status="success"} 4800
tasks_processed_total{task_type="email", status="failed"} 200
tasks_processed_total{task_type="report", status="success"} 1150
tasks_processed_total{task_type="report", status="failed"} 50
```

**Queries:**
```promql
# Task processing rate
rate(tasks_processed_total[5m])

# Success rate
sum(rate(tasks_processed_total{status="success"}[5m])) / sum(rate(tasks_processed_total[5m])) * 100

# Failure rate
sum(rate(tasks_processed_total{status="failed"}[5m])) / sum(rate(tasks_processed_total[5m])) * 100

# Tasks pending (created - processed)
sum(rate(tasks_created_total[5m])) - sum(rate(tasks_processed_total[5m]))
```

---

#### **`task_processing_duration_seconds`** (Histogram)
```
Type: Histogram
Labels: task_type
Description: Duration of task processing in seconds
Buckets: [0.1, 0.5, 1, 2, 5, 10, 30, 60]
```

**Use Case:**
- Monitor worker performance
- Identify slow task types
- SLA monitoring for background jobs

**Queries:**
```promql
# Average processing time
rate(task_processing_duration_seconds_sum[5m]) / rate(task_processing_duration_seconds_count[5m])

# p95 processing time
histogram_quantile(0.95, rate(task_processing_duration_seconds_bucket[5m]))

# Slowest task type (p95)
topk(5, histogram_quantile(0.95, sum by (task_type) (rate(task_processing_duration_seconds_bucket[5m]))))

# Tasks taking > 10 seconds
sum(rate(task_processing_duration_seconds_bucket{le="10"}[5m])) / sum(rate(task_processing_duration_seconds_count[5m])) * 100
```

---

#### **`tasks_queue_size`** (Gauge)
```
Type: Gauge
Labels: -
Description: Current number of tasks in the queue
```

**Use Case:**
- Monitor queue backlog
- Detect bottlenecks
- Capacity planning

**Queries:**
```promql
# Current queue size
tasks_queue_size

# Average queue size
avg_over_time(tasks_queue_size[5m])

# Max queue size (last 1 hour)
max_over_time(tasks_queue_size[1h])
```

---

#### **`tasks_failed_total`** (Counter)
```
Type: Counter
Labels: task_type, error_type
Description: Total number of tasks that failed processing
```

**Example Values:**
```
tasks_failed_total{task_type="email", error_type="timeout"} 50
tasks_failed_total{task_type="email", error_type="mark_processing_error"} 10
tasks_failed_total{task_type="report", error_type="task_execution_error"} 30
tasks_failed_total{task_type="email", error_type="max_retries"} 15
tasks_failed_total{task_type="report", error_type="republish_error"} 5
```

**Queries:**
```promql
# Failure rate
rate(tasks_failed_total[5m])

# Most common error type
topk(5, rate(tasks_failed_total[5m]))

# Failures by task type
sum by (task_type) (rate(tasks_failed_total[5m]))
```

---

### **Database Metrics**

#### **`db_connections_open`** (Gauge)
```
Type: Gauge
Labels: -
Description: Number of open database connections
```

**Queries:**
```promql
# Current open connections
db_connections_open

# Average open connections
avg_over_time(db_connections_open[5m])
```

---

#### **`db_connections_in_use`** (Gauge)
```
Type: Gauge
Labels: -
Description: Number of database connections currently in use
```

**Queries:**
```promql
# Current in-use connections
db_connections_in_use

# Connection pool usage
db_connections_in_use / db_connections_open * 100
```

---

#### **`db_query_duration_seconds`** (Histogram)
```
Type: Histogram
Labels: query_type
Description: Duration of database queries in seconds
Query Types: SELECT, INSERT, UPDATE, DELETE
Buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1]
```

**Queries:**
```promql
# Average query time
rate(db_query_duration_seconds_sum[5m]) / rate(db_query_duration_seconds_count[5m])

# p95 query time
histogram_quantile(0.95, rate(db_query_duration_seconds_bucket[5m]))

# Slow queries (> 100ms)
histogram_quantile(0.95, rate(db_query_duration_seconds_bucket[5m])) > 0.1
```

---

### **Cache Metrics** (Redis)

#### **`cache_hits_total`** (Counter)
```
Type: Counter
Labels: key_type
Description: Total number of cache hits
```

**Queries:**
```promql
# Cache hit rate
rate(cache_hits_total[5m])

# Cache hit ratio
sum(rate(cache_hits_total[5m])) / (sum(rate(cache_hits_total[5m])) + sum(rate(cache_misses_total[5m]))) * 100
```

---

#### **`cache_misses_total`** (Counter)
```
Type: Counter
Labels: key_type
Description: Total number of cache misses
```

**Queries:**
```promql
# Cache miss rate
rate(cache_misses_total[5m])

# Cache miss ratio
sum(rate(cache_misses_total[5m])) / (sum(rate(cache_hits_total[5m])) + sum(rate(cache_misses_total[5m]))) * 100
```

---

### **Queue Metrics** (RabbitMQ)

#### **`queue_messages_published_total`** (Counter)
```
Type: Counter
Labels: queue_name
Description: Total number of messages published to the queue
```

**Queries:**
```promql
# Publishing rate
rate(queue_messages_published_total[5m])
```

---

#### **`queue_messages_consumed_total`** (Counter)
```
Type: Counter
Labels: queue_name
Description: Total number of messages consumed from the queue
```

**Queries:**
```promql
# Consumption rate
rate(queue_messages_consumed_total[5m])

# Queue lag (published vs consumed)
sum(rate(queue_messages_published_total[5m])) - sum(rate(queue_messages_consumed_total[5m]))
```

---

### **Go Runtime Metrics** (Built-in)

#### **`go_goroutines`** (Gauge)
```
Description: Number of goroutines currently running
```

**Use Case:**
- Detect goroutine leaks
- Monitor concurrency

**Queries:**
```promql
go_goroutines

# Goroutine leak detection (increasing over time)
delta(go_goroutines[1h]) > 100
```

---

#### **`go_memstats_alloc_bytes`** (Gauge)
```
Description: Bytes of allocated heap objects
```

**Use Case:**
- Monitor memory usage
- Detect memory leaks

**Queries:**
```promql
# Memory in MB
go_memstats_alloc_bytes / 1024 / 1024

# Memory leak detection
delta(go_memstats_alloc_bytes[1h]) > 100000000
```

---

#### **`go_gc_duration_seconds`** (Summary)
```
Description: Garbage collection duration
```

**Use Case:**
- Monitor GC performance
- Detect GC pressure

**Queries:**
```promql
# GC time per second
rate(go_gc_duration_seconds_sum[5m])
```

---

## 📈 **Common Query Patterns**

### **Rate Calculation**
```promql
# Counter → Rate (per second)
rate(metric_total[5m])

# Counter → Total increase
increase(metric_total[1h])
```

### **Aggregation**
```promql
# Sum across all labels
sum(metric)

# Sum by specific label
sum by (label) (metric)

# Average
avg(metric)

# Max/Min
max(metric)
min(metric)
```

### **Filtering**
```promql
# Exact match
metric{label="value"}

# Regex match
metric{label=~"pattern"}

# Multiple values
metric{status=~"200|201|202"}

# Not equal
metric{status!="500"}
```

### **Percentiles (Histogram)**
```promql
# p50 (median)
histogram_quantile(0.50, rate(metric_bucket[5m]))

# p95
histogram_quantile(0.95, rate(metric_bucket[5m]))

# p99
histogram_quantile(0.99, rate(metric_bucket[5m]))
```

---

## 🎯 **Real-World Examples**

### **Example 1: API Health Check**
```promql
# Is API responding?
up{job="task-handler-api"}

# Request rate
sum(rate(http_requests_total[5m]))

# Error rate
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100

# Slow responses (p95 > 1s)
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
```

### **Example 2: Worker Performance**
```promql
# Task throughput
sum(rate(tasks_processed_total[5m]))

# Success rate
sum(rate(tasks_processed_total{status="success"}[5m])) / sum(rate(tasks_processed_total[5m])) * 100

# Processing time (p95)
histogram_quantile(0.95, rate(task_processing_duration_seconds_bucket[5m]))

# Queue backlog
tasks_queue_size
```

### **Example 3: Resource Monitoring**
```promql
# Memory usage (MB)
go_memstats_alloc_bytes / 1024 / 1024

# Goroutines
go_goroutines

# DB connections usage
db_connections_in_use / db_connections_open * 100
```

---

**Happy Querying! 🚀📊**
