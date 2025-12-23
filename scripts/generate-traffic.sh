#!/bin/bash

# 🚀 Generate Test Traffic for Monitoring
# This script creates tasks to generate metrics data

echo "🚀 Generating test traffic..."
echo ""

# Check if user exists, if not register
echo "📝 Checking authentication..."
TOKEN=$(curl -s -X POST http://localhost:8087/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}' \
  2>/dev/null | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "👤 Registering new user..."
  curl -s -X POST http://localhost:8087/auth/register \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser","password":"password123"}' > /dev/null
  
  echo "🔐 Logging in..."
  TOKEN=$(curl -s -X POST http://localhost:8087/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser","password":"password123"}' \
    2>/dev/null | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
fi

if [ -z "$TOKEN" ]; then
  echo "❌ Failed to authenticate"
  exit 1
fi

echo "✅ Authenticated successfully"
echo ""

# Create tasks
NUM_TASKS=${1:-50}
echo "📊 Creating $NUM_TASKS tasks..."
for i in $(seq 1 $NUM_TASKS); do
  RESPONSE=$(curl -s -X POST http://localhost:8087/api/v1/tasks \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"task_type":"email"}' 2>/dev/null)
  
  if echo "$RESPONSE" | grep -q "task_id"; then
    echo -n "✓"
  else
    echo -n "✗"
  fi
  
  sleep 0.1
done

echo ""
echo ""
echo "✅ Traffic generation complete!"
echo ""
echo "📊 View metrics at:"
echo "  - Grafana: http://localhost:3000/d/task-handler-dashboard/task-handler-monitoring"
echo "  - Prometheus: http://localhost:9090"
echo ""
echo "💡 Wait 10-15 seconds for metrics to update in dashboard"
