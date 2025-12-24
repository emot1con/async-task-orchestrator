#!/bin/bash

# Script untuk testing Rate Limiter di Auth Endpoint (Nginx)
# Usage: ./test_auth_rate_limiter.sh

BASE_URL="http://localhost"  # Akses via Nginx (port 80)
REQUESTS=${1:-100}  # Default 10 requests

echo "🔐 Testing Auth Rate Limiter (Nginx)"
echo "===================================="
echo "Target: $BASE_URL/auth/login"
echo "Config: 1 req/sec per IP (burst=10)"
echo "Requests: $REQUESTS"
echo ""

# Counter untuk hasil
success=0
rate_limited=0
errors=0

start_time=$(date +%s)

echo "🔥 Sending $REQUESTS login requests rapidly..."
echo ""

for i in $(seq 1 $REQUESTS); do
  # Send login request
  response=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{
      "email": "user@example.com",
      "password": "password123"
    }')
  
  # Extract status code
  status_code=$(echo "$response" | tail -n1)
  body=$(echo "$response" | head -n -1)
  
  # Get timestamp untuk tracking
  timestamp=$(date +%H:%M:%S.%3N)
  
  # Count results
  if [ "$status_code" = "200" ]; then
    ((success++))
    echo -e "[$timestamp] Request $i: \033[0;32m✓ SUCCESS\033[0m (HTTP $status_code)"
  elif [ "$status_code" = "503" ]; then
    ((rate_limited++))
    echo -e "[$timestamp] Request $i: \033[0;33m✗ RATE LIMITED (Nginx)\033[0m (HTTP $status_code)"
  elif [ "$status_code" = "401" ] || [ "$status_code" = "400" ]; then
    ((success++))
    echo -e "[$timestamp] Request $i: \033[0;32m✓ REACHED APP\033[0m (HTTP $status_code - Auth failed, but not rate limited)"
  else
    ((errors++))
    echo -e "[$timestamp] Request $i: \033[0;31m✗ ERROR\033[0m (HTTP $status_code)"
    # echo "   Response: $body" | head -c 200
  fi
  
  # NO DELAY - kirim secepat mungkin untuk trigger rate limit
done

end_time=$(date +%s)
duration=$((end_time - start_time))

echo ""
echo "=========================================="
echo "📊 TEST RESULTS"
echo "=========================================="
echo "Total Requests:     $REQUESTS"
echo "Successful:         $success (✓)"
echo "Rate Limited:       $rate_limited (503 Nginx)"
echo "Errors:             $errors (✗)"
echo "Duration:           ${duration}s"
echo ""

if [ $duration -gt 0 ]; then
  echo "Throughput:         $(echo "scale=2; $REQUESTS / $duration" | bc) req/sec"
fi

echo "Success Rate:       $(echo "scale=2; $success * 100 / $REQUESTS" | bc)%"
echo "Rate Limit Rate:    $(echo "scale=2; $rate_limited * 100 / $REQUESTS" | bc)%"
echo "=========================================="
echo ""

# Interpretation
echo "📋 ANALYSIS:"
echo ""

if [ $rate_limited -gt 0 ]; then
  echo "✅ Rate limiter is WORKING!"
  echo ""
  echo "💡 Current Configuration (Nginx):"
  echo "   - Zone: auth_api_limit"
  echo "   - Rate: 1 req/sec per IP"
  echo "   - Burst: 10 requests"
  echo "   - Expected behavior:"
  echo "     • First 10 requests: PASS (burst capacity)"
  echo "     • Request 11+: RATE LIMITED (503)"
  echo ""
  echo "🔐 Security Status: PROTECTED against brute-force!"
else
  echo "⚠️  No rate limiting detected!"
  echo ""
  echo "💡 Possible reasons:"
  echo "   - Requests sent too slowly (< 1 req/sec)"
  echo "   - Rate limiter not configured"
  echo "   - All requests within burst capacity (10)"
  echo ""
  echo "💡 To trigger rate limit:"
  echo "   Run: ./test_auth_rate_limiter.sh 15"
  echo "   (15 requests will exceed burst of 10)"
fi

echo ""
echo "🔍 Nginx Configuration Check:"
echo "To verify nginx config, run:"
echo "   sudo grep -A 2 'auth_api_limit' /etc/nginx/nginx.conf"
echo "   sudo cat /etc/nginx/conf.d/project-a.conf | grep -A 3 '/auth'"
echo ""

echo "📈 Real-time Monitoring:"
echo "Watch nginx access logs:"
echo "   sudo tail -f /var/log/nginx/access.log | grep '/auth/login'"
echo ""

echo "🧪 Test Different Scenarios:"
echo "   Fast (immediate): ./test_auth_rate_limiter.sh 15"
echo "   Slow (1 req/sec): for i in {1..10}; do curl -X POST $BASE_URL/auth/login -H 'Content-Type: application/json' -d '{\"email\":\"test@test.com\",\"password\":\"123\"}'; sleep 1; done"
echo ""
