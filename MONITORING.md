# 📊 Monitoring dengan Prometheus & Grafana

## 📖 **Penjelasan Dasar**

### **Apa itu Prometheus?** 🔍
Prometheus adalah **monitoring system** dan **time-series database** yang:
- **Mengumpulkan metrics** dari aplikasi kamu (CPU, memory, HTTP requests, dll)
- **Menyimpan data** dalam format time-series (data dengan timestamp)
- **Query data** menggunakan bahasa PromQL (Prometheus Query Language)
- **Scraping model**: Prometheus aktif "menarik" (pull) data dari aplikasi

**Analogi Sederhana:**
> Prometheus seperti **tukang catat** yang datang setiap 15 detik untuk mencatat semua aktivitas aplikasi kamu (berapa request masuk, berapa task dibuat, berapa lama response time, dll). Data ini disimpan dalam "buku catatan" yang bisa kamu query kapan saja.

### **Apa itu Grafana?** 📈
Grafana adalah **visualization tool** yang:
- **Menampilkan metrics** dalam bentuk grafik yang cantik
- **Membuat dashboard** untuk monitoring real-time
- **Setup alerting** (kirim notifikasi kalau ada masalah)
- **Support banyak data source** (Prometheus, MySQL, PostgreSQL, dll)

**Analogi Sederhana:**
> Grafana seperti **papan display** atau **layar monitor** yang menampilkan catatan dari Prometheus dalam bentuk grafik, gauge, dan table yang mudah dipahami. Bayangkan dashboard mobil yang menunjukkan kecepatan, RPM, suhu mesin - tapi untuk aplikasi.

---

## 🎯 **Mengapa Perlu Monitoring?**

Bayangkan aplikasi kamu sudah di production:

### **Tanpa Monitoring** ❌
```
User: "Kok aplikasi lambat?"
Kamu: "Emm... gak tahu kenapa 🤷"

User: "API error terus!"
Kamu: "Error apa ya? Kapan?" *buka server logs*

User: "Server down!"
Kamu: "Oh iya ya?" *baru tahu setelah 30 menit*
```

### **Dengan Monitoring** ✅
```
Dashboard Grafana menunjukkan:
- HTTP Request Rate naik 300% → Traffic spike
- Response Time naik dari 50ms ke 2s → Ada bottleneck
- Memory usage 90% → Memory leak
- Task Failed Rate 20% → Ada bug di worker

Alert Grafana:
📧 "Warning: API response time > 1s"
📧 "Critical: Memory usage > 90%"
📧 "Error: Task failed rate > 10%"

Kamu bisa:
✅ Lihat masalah secara real-time
✅ Identifikasi root cause dengan cepat
✅ Fix sebelum user complain
✅ Analyze trend untuk capacity planning
```

---

## 🔧 **Arsitektur Monitoring System**

```
┌─────────────────────────────────────────────────────────────┐
│                     Task Handler System                      │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐           ┌─────────────────┐            │
│  │   API Server │           │  Worker Service │            │
│  │   (port 8087)│           │   (port 8088)   │            │
│  │              │           │                 │            │
│  │ /metrics ◄───┼───────────┼──────► /metrics │            │
│  └──────▲───────┘           └────────▲────────┘            │
│         │                             │                     │
│         │    Expose Metrics           │                     │
│         │    (Pull Model)             │                     │
└─────────┼─────────────────────────────┼─────────────────────┘
          │                             │
          │                             │
          │    ┌────────────────────────┤
          │    │   Scrape every 10s     │
          │    │                        │
    ┌─────▼────▼─────────────┐         │
    │   Prometheus            │         │
    │   (port 9090)           │         │
    │                         │         │
    │ - Store metrics         │         │
    │ - Run queries (PromQL)  │         │
    │ - Evaluate alerts       │         │
    └────────────┬────────────┘         │
                 │                      │
                 │  Query metrics       │
                 │                      │
          ┌──────▼───────────┐          │
          │   Grafana        │          │
          │   (port 3000)    │          │
          │                  │          │
          │ - Visualize data │          │
          │ - Create dashboards        │
          │ - Send alerts    │          │
          └──────────────────┘          │
                                        │
         You (Developer) ──────────────┘
         Access via browser:
         http://localhost:3000
```

### **Flow Penjelasan:**
1. **API & Worker** expose `/metrics` endpoint dengan metrics data
2. **Prometheus** scrape (ambil) metrics dari API & Worker setiap 10 detik
3. **Prometheus** simpan data dalam time-series database
4. **Grafana** query Prometheus dan tampilkan dalam dashboard
5. **You** buka Grafana dashboard untuk monitoring

---

## 📊 **Metrics yang Di-Track**

### **1. HTTP Metrics** (dari API Server)
```go
http_requests_total{method="POST", endpoint="/api/v1/tasks", status="201"}
// Total HTTP requests by method, endpoint, and status code

http_request_duration_seconds{method="POST", endpoint="/api/v1/tasks"}
// Response time (latency) untuk setiap endpoint

http_requests_in_flight
// Berapa request yang sedang diproses saat ini
```

**Use Case:**
- Monitor API traffic
- Detect slow endpoints
- Track error rates (4xx, 5xx)

### **2. Task Metrics** (dari Worker)
```go
tasks_created_total{task_type="email"}
// Total tasks dibuat per task type

tasks_processed_total{task_type="email", status="success"}
// Total tasks selesai diproses (success/failed)

task_processing_duration_seconds{task_type="email"}
// Berapa lama worker memproses task

tasks_failed_total{task_type="email", error_type="timeout"}
// Total tasks yang failed by error type

tasks_queue_size
// Berapa task yang menunggu di queue
```

**Use Case:**
- Monitor worker performance
- Detect task processing bottlenecks
- Track failure patterns

### **3. System Metrics** (Go Runtime)
```go
go_goroutines
// Number of goroutines (detect goroutine leaks)

go_memstats_alloc_bytes
// Memory allocated (detect memory leaks)

go_gc_duration_seconds
// Garbage collection time
```

**Use Case:**
- Monitor application health
- Detect resource leaks
- Optimize performance

### **4. Database Metrics** (Custom)
```go
db_connections_open
// Open database connections

db_connections_in_use
// Active database connections

db_query_duration_seconds{query_type="SELECT"}
// Query execution time
```

### **5. Cache Metrics** (Redis - Custom)
```go
cache_hits_total{key_type="user"}
// Cache hits

cache_misses_total{key_type="user"}
// Cache misses
```

### **6. Queue Metrics** (RabbitMQ)
```go
queue_messages_published_total{queue_name="task_queue"}
// Messages published

queue_messages_consumed_total{queue_name="task_queue"}
// Messages consumed
```

---

## 🚀 **Cara Menjalankan**

### **1. Start Semua Services**
```bash
docker-compose up -d
```

Ini akan start:
- PostgreSQL (port 5432)
- Redis (port 6379)
- RabbitMQ (port 5672, management 15672)
- API Server (port 8087)
- Worker (port 8088 for metrics)
- **Prometheus** (port 9090)
- **Grafana** (port 3000)

### **2. Verify Services Running**
```bash
# Check containers
docker ps

# Should show:
# - prometheus
# - grafana
# - api
# - worker
# - postgres
# - redis
# - rabbitmq
```

### **3. Access Monitoring Tools**

#### **Prometheus UI** 🔍
```
URL: http://localhost:9090
```

**What you can do:**
- Check scrape targets: http://localhost:9090/targets
  - Should show `task-handler-api` and `task-handler-worker` as **UP**
- Run PromQL queries
- View metrics graphs

#### **Grafana Dashboard** 📊
```
URL: http://localhost:3000
Username: admin
Password: admin
```

**What you can do:**
- View pre-configured dashboard: "Task Handler Monitoring"
- Create custom dashboards
- Setup alerting

---

## 📈 **Menggunakan Grafana Dashboard**

### **Dashboard Overview**

Dashboard sudah dikonfigurasi dengan panels:

#### **Panel 1-4: Key Metrics (Stat Panels)**
```
┌──────────────┬──────────────┬──────────────┬──────────────┐
│ HTTP Request │ Total Tasks  │   Tasks      │    Tasks     │
│ Rate (req/s) │   Created    │  Processed   │    Failed    │
│    5.2       │    1,234     │    1,200     │      34      │
└──────────────┴──────────────┴──────────────┴──────────────┘
```

#### **Panel 5: HTTP Request Rate by Endpoint**
- Line graph showing request rate per endpoint
- Legends: `POST /api/v1/tasks (201)`, `GET /api/v1/tasks/:id (200)`

#### **Panel 6: HTTP Request Duration (p95, p99)**
- Line graph showing response time percentiles
- p95: 95% requests faster than this
- p99: 99% requests faster than this

#### **Panel 7: Task Creation Rate by Type**
- Line graph showing task creation rate per task type
- Legends: `email`, `report`, `notification`

#### **Panel 8: Task Processing Duration (p95)**
- Line graph showing how long workers process tasks
- Useful to detect slow tasks

#### **Panel 9: Go Goroutines**
- Line graph showing goroutine count for API & Worker
- Detect goroutine leaks

---

## 🔍 **PromQL Query Examples**

### **Basic Queries**

#### **1. Total HTTP Requests (last 5 minutes)**
```promql
sum(rate(http_requests_total[5m]))
```

#### **2. HTTP Request Rate by Status Code**
```promql
sum by (status) (rate(http_requests_total[5m]))
```

#### **3. HTTP 5xx Error Rate**
```promql
sum(rate(http_requests_total{status=~"5.."}[5m]))
```

#### **4. Average Response Time**
```promql
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])
```

#### **5. p95 Response Time**
```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

#### **6. p99 Response Time**
```promql
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
```

### **Task Queries**

#### **7. Task Creation Rate**
```promql
rate(tasks_created_total[5m])
```

#### **8. Task Success Rate (%)**
```promql
sum(rate(tasks_processed_total{status="success"}[5m])) / sum(rate(tasks_processed_total[5m])) * 100
```

#### **9. Task Failure Rate (%)**
```promql
sum(rate(tasks_processed_total{status="failed"}[5m])) / sum(rate(tasks_processed_total[5m])) * 100
```

#### **10. Average Task Processing Time**
```promql
rate(task_processing_duration_seconds_sum[5m]) / rate(task_processing_duration_seconds_count[5m])
```

#### **11. Slow Tasks (processing > 5s)**
```promql
histogram_quantile(0.95, rate(task_processing_duration_seconds_bucket[5m])) > 5
```

### **System Queries**

#### **12. Goroutine Count**
```promql
go_goroutines{job="task-handler-api"}
```

#### **13. Memory Usage (MB)**
```promql
go_memstats_alloc_bytes{job="task-handler-api"} / 1024 / 1024
```

#### **14. GC Duration**
```promql
rate(go_gc_duration_seconds_sum[5m])
```

---

## ⚡ **Testing Metrics**

### **1. Generate HTTP Traffic**
```bash
# Register user
curl -X POST http://localhost:8087/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "password123"}'

# Login
TOKEN=$(curl -X POST http://localhost:8087/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "password123"}' \
  | jq -r '.access_token')

# Create tasks (loop 100 times)
for i in {1..100}; do
  curl -X POST http://localhost:8087/api/v1/tasks \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"task_type": "email"}'
  sleep 0.1
done
```

### **2. Check Metrics Endpoint**
```bash
# API metrics
curl http://localhost:8087/metrics

# Worker metrics
curl http://localhost:8088/metrics
```

You should see output like:
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{endpoint="/api/v1/tasks",method="POST",status="201"} 100

# HELP task_processing_duration_seconds Duration of task processing in seconds
# TYPE task_processing_duration_seconds histogram
task_processing_duration_seconds_bucket{task_type="email",le="0.1"} 45
task_processing_duration_seconds_bucket{task_type="email",le="0.5"} 89
task_processing_duration_seconds_bucket{task_type="email",le="1"} 98
task_processing_duration_seconds_bucket{task_type="email",le="+Inf"} 100
```

### **3. View in Prometheus**
1. Open http://localhost:9090
2. Go to "Graph" tab
3. Enter query: `rate(http_requests_total[5m])`
4. Click "Execute"
5. Switch to "Graph" view

### **4. View in Grafana**
1. Open http://localhost:3000
2. Login (admin/admin)
3. Go to Dashboards → Task Handler Monitoring
4. You should see all panels updating with real data

---

## 🔔 **Setup Alerting (Optional)**

### **Contoh Alert Rules**

Create file `monitoring/prometheus/alerts/alerts.yml`:

```yaml
groups:
  - name: task_handler_alerts
    interval: 30s
    rules:
      # High error rate
      - alert: HighHTTPErrorRate
        expr: sum(rate(http_requests_total{status=~"5.."}[5m])) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High HTTP 5xx error rate"
          description: "Error rate is {{ $value }} errors/sec"
      
      # Slow API response
      - alert: SlowAPIResponse
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API response time is slow"
          description: "p95 latency is {{ $value }}s"
      
      # High task failure rate
      - alert: HighTaskFailureRate
        expr: sum(rate(tasks_processed_total{status="failed"}[5m])) / sum(rate(tasks_processed_total[5m])) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High task failure rate"
          description: "{{ $value | humanizePercentage }} tasks are failing"
      
      # Memory usage high
      - alert: HighMemoryUsage
        expr: go_memstats_alloc_bytes / 1024 / 1024 > 500
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Memory usage is {{ $value }}MB"
```

Update `prometheus.yml`:
```yaml
rule_files:
  - "alerts/*.yml"
```

---

## 🎓 **Tips & Best Practices**

### **1. Metric Naming Convention**
```
✅ Good:
http_requests_total          (clear, descriptive)
task_processing_duration_seconds  (includes unit)
cache_hits_total             (clear purpose)

❌ Bad:
requests                     (ambiguous)
duration                     (no unit)
hits                         (what type?)
```

### **2. Use Labels Wisely**
```
✅ Good:
http_requests_total{method="POST", endpoint="/api/v1/tasks", status="201"}
// Labels add dimensions for filtering

❌ Bad:
http_requests_post_api_tasks_201
// Too specific, can't aggregate
```

### **3. Choose Right Metric Type**

#### **Counter** - Always increasing
```go
http_requests_total       // Total requests (never decreases)
tasks_created_total       // Total tasks created
```

#### **Gauge** - Can go up/down
```go
http_requests_in_flight   // Current requests
tasks_queue_size          // Current queue size
memory_usage_bytes        // Current memory
```

#### **Histogram** - Distribution of values
```go
http_request_duration_seconds  // Response time distribution
task_processing_duration_seconds  // Processing time distribution
```

### **4. Scrape Interval**
```yaml
# Too frequent (expensive)
scrape_interval: 1s  ❌

# Good balance
scrape_interval: 10s  ✅

# Too slow (miss spikes)
scrape_interval: 5m  ❌
```

### **5. Retention Policy**
```yaml
# Keep data for reasonable time
storage:
  tsdb:
    retention:
      time: 15d      # 15 days
      size: 10GB     # Max 10GB
```

---

## 🐛 **Troubleshooting**

### **Problem: Prometheus can't scrape targets**
```
Error: "context deadline exceeded"
```

**Solution:**
1. Check if API & Worker expose `/metrics`
   ```bash
   curl http://localhost:8087/metrics
   curl http://localhost:8088/metrics
   ```

2. Check Prometheus targets: http://localhost:9090/targets
   - Should show "UP" status

3. Check Docker network:
   ```bash
   docker network inspect task_handler_backend
   ```

### **Problem: Grafana can't connect to Prometheus**
```
Error: "Bad Gateway"
```

**Solution:**
1. Check Prometheus URL in Grafana datasource: http://prometheus:9090
2. Restart Grafana:
   ```bash
   docker-compose restart grafana
   ```

### **Problem: Dashboard shows "No data"**
```
Panel shows empty graph
```

**Solution:**
1. Check time range (top-right corner) - set to "Last 1 hour"
2. Generate traffic to application
3. Wait 15-30 seconds for Prometheus to scrape
4. Refresh dashboard

### **Problem: Metrics not updating**
```
Metrics stuck at old values
```

**Solution:**
1. Check if application is running:
   ```bash
   docker ps
   ```
2. Check Prometheus scrape interval (should be 10s)
3. Force Prometheus to scrape:
   - Go to http://localhost:9090/targets
   - Wait for next scrape

---

## 📚 **Referensi Lebih Lanjut**

### **Official Documentation**
- **Prometheus**: https://prometheus.io/docs/
- **Grafana**: https://grafana.com/docs/
- **Prometheus Go Client**: https://github.com/prometheus/client_golang

### **Belajar PromQL**
- **Official Guide**: https://prometheus.io/docs/prometheus/latest/querying/basics/
- **PromQL Cheat Sheet**: https://promlabs.com/promql-cheat-sheet/

### **Dashboard Examples**
- **Grafana Dashboards**: https://grafana.com/grafana/dashboards/
- **Go Runtime Dashboard**: https://grafana.com/grafana/dashboards/10826

---

## 🎯 **Summary**

### **What We Built**
✅ Prometheus untuk collect & store metrics
✅ Grafana untuk visualize metrics
✅ Custom metrics untuk HTTP, Tasks, System
✅ Pre-configured dashboard
✅ Auto-provisioning datasource

### **What You Can Do Now**
✅ Monitor API traffic in real-time
✅ Track task processing performance
✅ Detect bottlenecks & errors
✅ Analyze trends & patterns
✅ Setup alerts for critical issues

### **Next Steps (Optional)**
- [ ] Setup Alertmanager untuk notifications (email, Slack)
- [ ] Add more custom metrics (business metrics)
- [ ] Create custom dashboards untuk specific use cases
- [ ] Setup long-term storage (Thanos, VictoriaMetrics)
- [ ] Implement distributed tracing (Jaeger, Tempo)

---

**Happy Monitoring! 🚀📊**
