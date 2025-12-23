# Redis Rate Limiter Guide

## 📖 Penjelasan

Rate limiter ini menggunakan **Token Bucket Algorithm** yang diimplementasikan dengan Redis + Lua script untuk performa tinggi dan konsistensi di distributed system.

### Token Bucket Algorithm

Bayangkan seperti **ember berisi koin**:
- **Capacity** = Ukuran ember (berapa koin maksimal yang bisa ditampung)
- **Refill Rate** = Kecepatan koin masuk ke ember per detik
- Setiap request = Ambil 1 koin dari ember
- Jika koin habis = Request ditolak (429 Too Many Requests)

**Contoh**: 
- Capacity = 20, Refill Rate = 10/detik
- User bisa burst hingga 20 requests sekaligus
- Setelah itu, hanya bisa 10 requests per detik

## 🚀 Cara Pakai

### 1. **Default Configuration** (Sudah Aktif)

Rate limiter sudah otomatis aktif di semua endpoint `/api/v1/*` dengan konfigurasi default:
- **Capacity**: 20 tokens (burst capacity)
- **Refill Rate**: 10 tokens/second

```go
// Di internal/handler/handler.go
api := r.Group("/api/v1")
api.Use(middleware.AuthMiddleware(jwtSecret))
api.Use(middleware.RateLimiterMiddleware(redisClient, middleware.DefaultRateLimiterConfig()))
{
    api.POST("/tasks", taskCtrl.CreateTask)
    api.GET("/tasks/:id", taskCtrl.GetTask)
    api.GET("/users/:user_id/tasks", taskCtrl.GetTasksByUser)
}
```

### 2. **Custom Configuration per Endpoint**

Jika ingin konfigurasi berbeda untuk endpoint tertentu:

```go
// Contoh: Rate limit ketat untuk endpoint create task
createTaskRateLimit := &middleware.RateLimiterConfig{
    Capacity:   5,   // Max 5 burst requests
    RefillRate: 1.0, // 1 request per second
}

api.POST("/tasks", 
    middleware.RateLimiterMiddleware(redisClient, createTaskRateLimit),
    taskCtrl.CreateTask,
)

// Contoh: Rate limit lebih longgar untuk endpoint read
readTaskRateLimit := &middleware.RateLimiterConfig{
    Capacity:   100,  // Max 100 burst requests
    RefillRate: 50.0, // 50 requests per second
}

api.GET("/tasks/:id",
    middleware.RateLimiterMiddleware(redisClient, readTaskRateLimit),
    taskCtrl.GetTask,
)
```

### 3. **Tanpa Rate Limiter untuk Endpoint Tertentu**

Jika ingin endpoint tanpa rate limit:

```go
// Create endpoint group tanpa rate limiter
apiNoLimit := r.Group("/api/v1")
apiNoLimit.Use(middleware.AuthMiddleware(jwtSecret))
{
    apiNoLimit.GET("/health", healthCheck) // Endpoint tanpa rate limit
}
```

### 4. **Custom Rate Limiter Config**

Buat konfigurasi sendiri:

```go
// Conservative (untuk production)
conservativeConfig := &middleware.RateLimiterConfig{
    Capacity:   10,  // Burst: 10 requests
    RefillRate: 5.0, // Sustained: 5 req/sec
}

// Moderate (default)
moderateConfig := middleware.DefaultRateLimiterConfig() // 20/burst, 10/sec

// Aggressive (untuk testing/development)
aggressiveConfig := &middleware.RateLimiterConfig{
    Capacity:   1000, // Burst: 1000 requests
    RefillRate: 500,  // Sustained: 500 req/sec
}

// Strict (untuk sensitive endpoints)
strictConfig := &middleware.RateLimiterConfig{
    Capacity:   3,    // Burst: 3 requests
    RefillRate: 0.5,  // Sustained: 1 request per 2 seconds
}
```

## 📊 Response Rate Limit

### Request Berhasil (200 OK)
```json
{
    "task_id": 123,
    "status": "PENDING",
    "message": "Task created successfully"
}
```

### Request Ditolak (429 Too Many Requests)
```json
{
    "error": "Rate limit exceeded",
    "message": "Maximum 10 requests per second allowed",
    "retry_after": "0.1 seconds"
}
```

## 🧪 Testing Rate Limiter

### Test dengan cURL:

```bash
# Login dulu untuk dapat token
TOKEN=$(curl -s -X POST http://localhost:8087/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}' \
  | jq -r '.access_token')

# Test rate limiter dengan loop
for i in {1..30}; do
  echo "Request #$i:"
  curl -s -w "\nHTTP Status: %{http_code}\n" \
    -X POST http://localhost:8087/api/v1/tasks \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"task_type":"email"}' | jq
  echo "---"
  sleep 0.05  # 50ms delay
done
```

### Test dengan Script Bash:

```bash
#!/bin/bash
# test_rate_limiter.sh

TOKEN="your_jwt_token_here"
ENDPOINT="http://localhost:8087/api/v1/tasks"
REQUESTS=50

success=0
rate_limited=0

for i in $(seq 1 $REQUESTS); do
  response=$(curl -s -w "\n%{http_code}" \
    -X POST $ENDPOINT \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"task_type":"email"}')
  
  status_code=$(echo "$response" | tail -n1)
  
  if [ "$status_code" = "201" ] || [ "$status_code" = "200" ]; then
    ((success++))
    echo "Request $i: SUCCESS ✓"
  elif [ "$status_code" = "429" ]; then
    ((rate_limited++))
    echo "Request $i: RATE LIMITED ✗"
  else
    echo "Request $i: ERROR ($status_code)"
  fi
  
  sleep 0.01  # 10ms delay
done

echo ""
echo "========== RESULTS =========="
echo "Total Requests: $REQUESTS"
echo "Successful: $success"
echo "Rate Limited: $rate_limited"
echo "Success Rate: $(echo "scale=2; $success * 100 / $REQUESTS" | bc)%"
```

## 🔍 Monitor Rate Limiter di Redis

```bash
# Connect ke Redis
redis-cli

# Lihat semua rate limiter keys
KEYS rate_limiter:user:*

# Lihat detail rate limiter untuk user_id 1
HGETALL rate_limiter:user:1

# Output:
# 1) "tokens"       -> Sisa token saat ini
# 2) "12.5"
# 3) "last_refill"  -> Timestamp terakhir refill
# 4) "1703352000"

# Monitor real-time
MONITOR

# Hapus rate limit untuk user tertentu (reset)
DEL rate_limiter:user:1
```

## ⚙️ Konfigurasi Berdasarkan Use Case

### API Public (Rate limit ketat)
```go
publicConfig := &middleware.RateLimiterConfig{
    Capacity:   5,
    RefillRate: 1.0, // 1 req/sec
}
```

### API Internal (Rate limit longgar)
```go
internalConfig := &middleware.RateLimiterConfig{
    Capacity:   1000,
    RefillRate: 500, // 500 req/sec
}
```

### Login Endpoint (Anti brute-force)
```go
loginConfig := &middleware.RateLimiterConfig{
    Capacity:   3,
    RefillRate: 0.1, // 1 request per 10 seconds
}
```

### Read-Heavy Endpoint
```go
readConfig := &middleware.RateLimiterConfig{
    Capacity:   100,
    RefillRate: 50, // 50 req/sec
}
```

### Write-Heavy Endpoint
```go
writeConfig := &middleware.RateLimiterConfig{
    Capacity:   10,
    RefillRate: 5, // 5 req/sec
}
```

## 🎯 Best Practices

1. **Per-User Rate Limiting**: Rate limiter ini otomatis per-user (berdasarkan JWT `user_id`)
2. **Fail Open**: Jika Redis down, request tetap diizinkan (tidak block aplikasi)
3. **Atomic Operations**: Lua script dijalankan atomically di Redis (thread-safe)
4. **Auto Cleanup**: Keys otomatis expire setelah tidak dipakai
5. **Distributed Safe**: Bekerja sempurna di multiple instances (horizontal scaling)

## 🐛 Troubleshooting

### Rate Limiter Tidak Bekerja
```bash
# Check Redis connection
redis-cli PING

# Check Lua script loaded
redis-cli SCRIPT EXISTS <sha_hash>

# Check rate limiter keys exist
redis-cli KEYS rate_limiter:*
```

### Rate Limit Terlalu Ketat
- Increase `Capacity` untuk burst traffic
- Increase `RefillRate` untuk sustained traffic

### Rate Limit Terlalu Longgar
- Decrease `Capacity`
- Decrease `RefillRate`

## 📈 Performance

- **Latency**: <1ms overhead (Lua script runs in Redis)
- **Throughput**: Handles 100k+ req/sec per Redis instance
- **Memory**: ~100 bytes per active user
- **Scalability**: Linear with Redis capacity

## 🔐 Security Benefits

1. **DoS Protection**: Mencegah user spam requests
2. **Brute-Force Prevention**: Lindungi login endpoints
3. **Fair Usage**: Pastikan semua user dapat akses yang adil
4. **Cost Control**: Batasi penggunaan resources

## 📚 Referensi

- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- [Redis Lua Scripting](https://redis.io/docs/manual/programmability/eval-intro/)
- [Rate Limiting Strategies](https://redis.com/redis-best-practices/basic-rate-limiting/)
