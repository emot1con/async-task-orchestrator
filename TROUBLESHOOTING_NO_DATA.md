# 🔧 Troubleshooting: "No Data" in Grafana Dashboard

## ❓ **Problem**
Dashboard muncul tapi semua panel menunjukkan **"No data"**

## 🔍 **Root Cause**
**SELinux permission denied** pada volume mounts untuk Prometheus dan Grafana.

### **Error Symptoms:**
- Prometheus logs: `permission denied` saat load config
- Grafana logs: `permission denied` saat read provisioning files
- Prometheus targets menunjukkan targets DOWN
- Queries return empty results

---

## ✅ **Solution**

### **Step 1: Add `:Z` Flag untuk SELinux**

Update `docker-compose.yaml` dengan menambahkan `:Z` pada volume mounts:

```yaml
  prometheus:
    volumes:
      - ./monitoring/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:Z  # ← Add :Z
      - prometheus_data:/prometheus

  grafana:
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/provisioning:/etc/grafana/provisioning:Z  # ← Add :Z
      - ./monitoring/grafana/dashboards:/var/lib/grafana/dashboards:Z  # ← Add :Z
```

### **Step 2: Recreate Containers**

```bash
# Stop and remove containers
docker-compose stop prometheus grafana
docker-compose rm -f prometheus grafana

# Start with new configuration
docker-compose up -d prometheus grafana
```

### **Step 3: Verify Prometheus**

```bash
# Wait 10 seconds for startup
sleep 10

# Check Prometheus logs (should show no errors)
docker-compose logs prometheus --tail=20

# Check targets status
curl -s http://localhost:9090/api/v1/targets | python3 -m json.tool | grep "health"

# Expected: "health": "up" for both task-handler-api and task-handler-worker
```

### **Step 4: Verify Data**

```bash
# Query tasks_created_total
curl -s 'http://localhost:9090/api/v1/query?query=tasks_created_total' | python3 -m json.tool

# Expected: Should return data with metric values
```

### **Step 5: Generate Traffic**

```bash
# Use the helper script
./scripts/generate-traffic.sh 50

# Or manually
TOKEN=$(curl -s -X POST http://localhost:8087/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}' \
  | jq -r '.access_token')

for i in {1..50}; do
  curl -X POST http://localhost:8087/api/v1/tasks \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"task_type":"email"}'
  sleep 0.1
done
```

### **Step 6: Refresh Dashboard**

1. Open Grafana: http://localhost:3000/d/task-handler-dashboard/task-handler-monitoring
2. **Wait 15-30 seconds** for Prometheus to scrape new data
3. **Refresh dashboard** (or change time range to "Last 5 minutes")
4. Data should now appear! 📊

---

## 🎯 **Verification Checklist**

Use this checklist to verify everything is working:

### ✅ **Prometheus**
```bash
# 1. Check Prometheus is running
docker ps | grep prometheus
# Expected: Container running

# 2. Check Prometheus can read config
docker-compose logs prometheus | grep "permission denied"
# Expected: No output (no errors)

# 3. Check targets are UP
curl -s http://localhost:9090/api/v1/targets | grep '"health":"up"'
# Expected: Should find 3 targets (prometheus, api, worker)

# 4. Check metrics data exists
curl -s 'http://localhost:9090/api/v1/query?query=up' | grep '"value"'
# Expected: Should return values
```

### ✅ **Grafana**
```bash
# 1. Check Grafana is running
docker ps | grep grafana
# Expected: Container running

# 2. Check Grafana can read provisioning
docker-compose logs grafana | grep "permission denied"
# Expected: No output (no errors)

# 3. Check datasource is configured
curl -s -u admin:admin http://localhost:3000/api/datasources | grep "Prometheus"
# Expected: Should find Prometheus datasource

# 4. Check dashboard exists
curl -s -u admin:admin http://localhost:3000/api/search?type=dash-db | grep "Task Handler"
# Expected: Should find "Task Handler Monitoring"
```

### ✅ **Metrics Data**
```bash
# 1. Check API metrics endpoint
curl -s http://localhost:8087/metrics | head -20
# Expected: Should return Prometheus format metrics

# 2. Check Worker metrics endpoint
curl -s http://localhost:8088/metrics | head -20
# Expected: Should return Prometheus format metrics

# 3. Query specific metrics
curl -s 'http://localhost:9090/api/v1/query?query=tasks_created_total' | python3 -m json.tool
# Expected: Should return metric with value

curl -s 'http://localhost:9090/api/v1/query?query=http_requests_total' | python3 -m json.tool
# Expected: Should return metric with value
```

---

## 🐛 **Common Issues**

### **Issue 1: "permission denied" in logs**
**Cause:** SELinux blocking volume access

**Fix:**
```bash
# Add :Z flag to volume mounts in docker-compose.yaml
volumes:
  - ./path:/container/path:Z
```

### **Issue 2: Targets show "DOWN" in Prometheus**
**Cause:** 
- Container networking issue
- API/Worker not exposing metrics endpoint

**Fix:**
```bash
# Check if containers can reach each other
docker exec prometheus wget -qO- http://api:8087/metrics | head -5
docker exec prometheus wget -qO- http://worker:8088/metrics | head -5

# Check if metrics endpoints are exposed
curl http://localhost:8087/metrics | head -10
curl http://localhost:8088/metrics | head -10
```

### **Issue 3: Dashboard shows "No data" but Prometheus has data**
**Cause:**
- Time range too narrow
- No traffic generated yet
- Grafana can't connect to Prometheus

**Fix:**
```bash
# 1. Generate traffic
./scripts/generate-traffic.sh 50

# 2. Wait for Prometheus scrape (15 seconds)
sleep 15

# 3. In dashboard, change time range to "Last 1 hour"
# 4. Refresh dashboard (Ctrl+R or Cmd+R)

# 5. Test Grafana → Prometheus connection
curl -s -u admin:admin http://localhost:3000/api/datasources/proxy/1/api/v1/query?query=up
```

### **Issue 4: Metrics values are zero**
**Cause:** No application traffic

**Fix:**
```bash
# Generate traffic using helper script
./scripts/generate-traffic.sh 100

# This will:
# - Register/login user
# - Create 100 tasks
# - Generate HTTP and task metrics
```

---

## 📊 **Expected Dashboard Data**

After generating traffic, you should see:

### **Panel 1-4: Stats**
- **HTTP Request Rate**: ~0.5 to 2 req/s (depends on traffic)
- **Total Tasks Created**: Increasing number (e.g., 92, 142, etc.)
- **Tasks Processed (Success)**: Should match tasks created (after worker processes)
- **Tasks Failed**: 0 (if no errors)

### **Panel 5: HTTP Request Rate by Endpoint**
- Lines showing request rate for each endpoint
- `/auth/login`, `/auth/register`, `/api/v1/tasks` should be visible

### **Panel 6: HTTP Request Duration**
- p95 and p99 latency lines
- Should be < 100ms for most endpoints

### **Panel 7-9: Task & System Metrics**
- Task creation rate
- Task processing duration
- Goroutine count

---

## 🚀 **Quick Fix Script**

Save this as `fix-monitoring.sh`:

```bash
#!/bin/bash

echo "🔧 Fixing monitoring setup..."

# Stop containers
docker-compose stop prometheus grafana

# Remove containers
docker-compose rm -f prometheus grafana

# Start with new config
docker-compose up -d prometheus grafana

# Wait for startup
echo "⏳ Waiting 15 seconds for services to start..."
sleep 15

# Verify
echo ""
echo "✅ Verification:"
echo ""

# Check Prometheus
if docker-compose logs prometheus | grep -q "Server is ready"; then
  echo "✅ Prometheus: OK"
else
  echo "❌ Prometheus: FAILED"
fi

# Check Grafana
if docker-compose logs grafana | grep -q "HTTP Server Listen"; then
  echo "✅ Grafana: OK"
else
  echo "❌ Grafana: FAILED"
fi

# Check targets
TARGETS_UP=$(curl -s http://localhost:9090/api/v1/targets 2>/dev/null | grep -o '"health":"up"' | wc -l)
echo "✅ Prometheus targets UP: $TARGETS_UP/3"

echo ""
echo "🎉 Fix complete!"
echo ""
echo "📊 Access dashboard at:"
echo "   http://localhost:3000/d/task-handler-dashboard/task-handler-monitoring"
echo ""
echo "💡 Generate traffic with:"
echo "   ./scripts/generate-traffic.sh 50"
```

---

## 📚 **Additional Resources**

- **SELinux and Docker**: https://docs.docker.com/storage/bind-mounts/#configure-the-selinux-label
- **Prometheus Configuration**: https://prometheus.io/docs/prometheus/latest/configuration/configuration/
- **Grafana Provisioning**: https://grafana.com/docs/grafana/latest/administration/provisioning/

---

**Fixed? Great! Enjoy your monitoring! 🎉📊**
