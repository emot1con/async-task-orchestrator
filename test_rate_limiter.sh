#!/bin/bash

# Script untuk testing Rate Limiter
# Usage: ./test_rate_limiter.sh [number_of_requests]

REQUESTS=${1:-1000}  # Default 50 requests jika tidak ada argument
BASE_URL="http://localhost"

echo "🚀 Testing Rate Limiter"
echo "======================="
echo ""

# Step 1: Login untuk mendapatkan token
echo "📝 Step 1: Login..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test1",
    "password": "test123"
  }')

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
  echo "❌ Login failed!"
  echo "Response: $LOGIN_RESPONSE"
  echo ""
  echo "💡 Tip: Pastikan user sudah register dengan:"
  echo "   curl -X POST $BASE_URL/auth/register \\"
  echo "     -H 'Content-Type: application/json' \\"
  echo "     -d '{\"username\":\"test1\",\"password\":\"test123\",\"name\":\"Test User\"}'"
  exit 1
fi

echo "✅ Login successful!"
echo "Token: ${TOKEN:0:50}..."
echo ""

# Step 2: Test rate limiter
echo "🔥 Step 2: Sending $REQUESTS requests to test rate limiter..."
echo ""

success=0
rate_limited=0
errors=0

start_time=$(date +%s)

for i in $(seq 1 $REQUESTS); do
  # Send request
  response=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/v1/tasks" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"task_type":"username"}')
  
  # Extract status code
  status_code=$(echo "$response" | tail -n1)
  body=$(echo "$response" | head -n -1)
  
  # Count results
  if [ "$status_code" = "201" ] || [ "$status_code" = "200" ]; then
    ((success++))
    echo -e "Request $i: \033[0;32m✓ SUCCESS\033[0m (HTTP $status_code)"
  elif [ "$status_code" = "429" ]; then
    ((rate_limited++))
    retry_after=$(echo "$body" | jq -r '.retry_after // "N/A"')
    echo -e "Request $i: \033[0;33m✗ RATE LIMITED\033[0m (HTTP $status_code) - Retry after: $retry_after"
  else
    ((errors++))
    echo -e "Request $i: \033[0;31m✗ ERROR\033[0m (HTTP $status_code)"
    echo "   Response: $body"
  fi
  
  # Small delay between requests (10ms)
  sleep 0.01
done

end_time=$(date +%s)
duration=$((end_time - start_time))

echo ""
echo "=========================================="
echo "📊 TEST RESULTS"
echo "=========================================="
echo "Total Requests:     $REQUESTS"
echo "Successful:         $success (✓)"
echo "Rate Limited:       $rate_limited (429)"
echo "Errors:             $errors (✗)"
echo "Duration:           ${duration}s"
echo ""
echo "Success Rate:       $(echo "scale=2; $success * 100 / $REQUESTS" | bc)%"
echo "Rate Limit Rate:    $(echo "scale=2; $rate_limited * 100 / $REQUESTS" | bc)%"
echo "Throughput:         $(echo "scale=2; $REQUESTS / $duration" | bc) req/sec"
echo "=========================================="
echo ""

# Interpretation
if [ $rate_limited -gt 0 ]; then
  echo "✅ Rate limiter is WORKING!"
  echo "💡 Configuration:"
  echo "   - Default: 20 tokens capacity, 10 tokens/sec refill rate"
  echo "   - Burst: Can handle 20 requests instantly"
  echo "   - Sustained: Max 10 requests per second"
else
  echo "⚠️  No rate limiting detected!"
  echo "💡 Possible reasons:"
  echo "   - Requests too slow (< 10 req/sec)"
  echo "   - Rate limiter not enabled"
  echo "   - Try increasing request count: ./test_rate_limiter.sh 100"
fi
echo ""

# Redis inspection
echo "🔍 Redis Inspection:"
echo "To see rate limiter data in Redis, run:"
echo "   redis-cli KEYS 'rate_limiter:*'"
echo "   redis-cli HGETALL rate_limiter:user:1"
echo ""
