# 🚀 Quick Start: Prometheus & Grafana

## 📦 **File Structure**
```
task_handler/
├── docker-compose.yaml           # ✅ Updated (Prometheus + Grafana)
├── MONITORING.md                 # 📚 Full documentation
├── monitoring/
│   ├── prometheus/
│   │   └── prometheus.yml        # Prometheus config (scrape targets)
│   └── grafana/
│       ├── provisioning/
│       │   ├── datasources/
│       │   │   └── datasource.yml   # Auto-connect to Prometheus
│       │   └── dashboards/
│       │       └── dashboard.yml    # Auto-load dashboards
│       └── dashboards/
│           └── task-handler-dashboard.json  # Pre-configured dashboard
├── internal/
│   ├── observability/
│   │   └── metrics.go            # ✅ Custom metrics definitions
│   └── middleware/
│       └── prometheus.go         # ✅ HTTP metrics middleware
├── cmd/
│   ├── api/
│   │   └── main.go               # ✅ Expose /metrics endpoint
│   └── worker/
│       └── main.go               # ✅ Expose /metrics endpoint (port 8088)
```

---

## ⚡ **Quick Commands**

### **1. Start Everything**
```bash
# Build and start all containers
docker-compose up --build -d

# Check all containers running
docker ps

# Check logs
docker-compose logs -f prometheus
docker-compose logs -f grafana
```

### **2. Access Services**
```bash
# Prometheus UI
open http://localhost:9090

# Grafana Login
open http://localhost:3000
# Login: admin / admin

# Grafana Dashboard (Direct Link)
open http://localhost:3000/d/task-handler-dashboard/task-handler-monitoring

# API Metrics
curl http://localhost:8087/metrics

# Worker Metrics
curl http://localhost:8088/metrics
```

### **3. Generate Test Traffic**
```bash
# Register & Login
curl -X POST http://localhost:8087/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "password123"}'

TOKEN=$(curl -X POST http://localhost:8087/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "password123"}' \
  | jq -r '.access_token')

# Create 100 tasks
for i in {1..100}; do
  curl -X POST http://localhost:8087/api/v1/tasks \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"task_type": "email"}'
  sleep 0.1
done
```

### **4. Stop Everything**
```bash
docker-compose down
```

---

## 📊 **Monitoring URLs**

| Service | URL | Credentials | Purpose |
|---------|-----|-------------|---------|
| **Grafana Login** | http://localhost:3000 | admin / admin | Login page |
| **Dashboard (Direct)** | http://localhost:3000/d/task-handler-dashboard/task-handler-monitoring | admin / admin | Task Handler dashboard |
| **Prometheus** | http://localhost:9090 | - | Query metrics |
| **API Metrics** | http://localhost:8087/metrics | - | Raw API metrics |
| **Worker Metrics** | http://localhost:8088/metrics | - | Raw Worker metrics |
| **Prometheus Targets** | http://localhost:9090/targets | - | Check scrape status |

---

## 📈 **Key Metrics to Watch**

### **HTTP Performance**
```promql
# Request rate
rate(http_requests_total[5m])

# p95 response time
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate
sum(rate(http_requests_total{status=~"5.."}[5m]))
```

### **Task Processing**
```promql
# Task creation rate
rate(tasks_created_total[5m])

# Task success rate
sum(rate(tasks_processed_total{status="success"}[5m])) / sum(rate(tasks_processed_total[5m])) * 100

# Task processing time (p95)
histogram_quantile(0.95, rate(task_processing_duration_seconds_bucket[5m]))
```

### **System Health**
```promql
# Goroutines
go_goroutines

# Memory usage (MB)
go_memstats_alloc_bytes / 1024 / 1024
```

---

## 🔍 **Grafana Dashboard**

### **Pre-configured Panels**

1. **HTTP Request Rate** (req/s) - Real-time traffic
2. **Total Tasks Created** - Cumulative count
3. **Tasks Processed (Success)** - Success count
4. **Tasks Failed (Total)** - Failure count
5. **HTTP Request Rate by Endpoint** - Traffic breakdown
6. **HTTP Request Duration (p95, p99)** - Latency
7. **Task Creation Rate by Type** - Task distribution
8. **Task Processing Duration (p95)** - Worker performance
9. **Go Goroutines** - System health

### **Dashboard Location**
- URL: http://localhost:3000
- Path: Home → Dashboards → Task Handler Monitoring
- UID: `task-handler-dashboard`

---

## 🛠️ **Troubleshooting**

### **Prometheus can't scrape targets**
```bash
# Check if metrics endpoints are accessible
curl http://localhost:8087/metrics
curl http://localhost:8088/metrics

# Check Prometheus targets status
open http://localhost:9090/targets

# Should show:
# - task-handler-api (UP)
# - task-handler-worker (UP)
```

### **Grafana shows "No data"**
1. Check time range (top-right) → Set to "Last 1 hour"
2. Generate traffic to application
3. Wait 15-30 seconds for scrape
4. Refresh dashboard (Ctrl+R or Cmd+R)

### **Restart specific service**
```bash
docker-compose restart prometheus
docker-compose restart grafana
```

---

## 🎯 **What You Get**

### **Observability**
✅ Real-time application monitoring
✅ HTTP request tracking (rate, latency, errors)
✅ Task processing metrics (created, processed, failed)
✅ System metrics (goroutines, memory, GC)

### **Visualization**
✅ Pre-configured Grafana dashboard
✅ Auto-provisioned datasource (no manual setup)
✅ Beautiful graphs and stats
✅ Time-series analysis

### **Performance Insights**
✅ Identify slow endpoints (p95, p99 latency)
✅ Detect traffic spikes
✅ Monitor task processing times
✅ Track error rates

### **Operational Benefits**
✅ Proactive issue detection
✅ Root cause analysis
✅ Capacity planning data
✅ Historical trend analysis

---

## 📝 **Next Steps**

### **Immediate**
1. ✅ Run `docker-compose up -d`
2. ✅ Access Grafana at http://localhost:3000
3. ✅ Generate test traffic
4. ✅ Watch metrics in dashboard

### **Optional Enhancements**
- [ ] Setup alerting (Alertmanager)
- [ ] Add more business metrics
- [ ] Create custom dashboards
- [ ] Implement distributed tracing
- [ ] Add database metrics exporter
- [ ] Add Redis metrics exporter
- [ ] Add RabbitMQ metrics exporter

---

## 📚 **Documentation**

- **Full Guide**: Read `MONITORING.md` for detailed explanations
- **PromQL Tutorial**: https://prometheus.io/docs/prometheus/latest/querying/basics/
- **Grafana Docs**: https://grafana.com/docs/grafana/latest/

---

**You're all set! Happy Monitoring! 🎉📊**
