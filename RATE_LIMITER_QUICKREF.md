# 🚀 Redis Rate Limiter - Quick Reference

## 📖 Konsep Dasar

**Token Bucket Algorithm**: Seperti ember berisi koin
- **Capacity** = Ukuran ember (max tokens/burst)
- **Refill Rate** = Koin masuk per detik
- Setiap request = Ambil 1 koin
- Koin habis = Request ditolak (429)

---

## 🎯 Preset Configurations

```go
middleware.StrictRateLimiter()        // 3 burst, 0.1/sec  (login)
middleware.ConservativeRateLimiter()  // 10 burst, 5/sec   (write)
middleware.ModerateRateLimiter()      // 20 burst, 10/sec  (default)
middleware.GenerousRateLimiter()      // 100 burst, 50/sec (read)
middleware.UnlimitedRateLimiter()     // 10k burst, 1000/sec (dev only)
middleware.CustomRateLimiter(15, 7.0) // Custom: 15 burst, 7/sec
```

---

## 💻 Cara Pakai

### 1. Default (Sudah Aktif di `/api/v1/*`)

```go
// File: internal/handler/handler.go
api := r.Group("/api/v1")
api.Use(middleware.AuthMiddleware(jwtSecret))
api.Use(middleware.RateLimiterMiddleware(redisClient, middleware.DefaultRateLimiterConfig()))
```

✅ **Sudah aktif dengan config default (20 burst, 10/sec)**

### 2. Custom per Endpoint

```go
api := r.Group("/api/v1")
api.Use(middleware.AuthMiddleware(jwtSecret))
{
    // Ketat untuk write
    api.POST("/tasks", 
        middleware.RateLimiterMiddleware(redisClient, middleware.ConservativeRateLimiter()),
        taskCtrl.CreateTask,
    )
    
    // Longgar untuk read
    api.GET("/tasks/:id", 
        middleware.RateLimiterMiddleware(redisClient, middleware.GenerousRateLimiter()),
        taskCtrl.GetTask,
    )
}
```

### 3. Anti Brute-Force Login

```go
authGroup := r.Group("/auth")
{
    authGroup.POST("/register", userCtrl.Register) // No limit
    
    authGroup.POST("/login", 
        middleware.RateLimiterMiddleware(redisClient, middleware.StrictRateLimiter()),
        userCtrl.Login, // 3 burst, 1 req per 10 seconds
    )
}
```

---

## 🧪 Testing

### Quick Test dengan cURL

```bash
# 1. Login
TOKEN=$(curl -s -X POST http://localhost:8087/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}' \
  | jq -r '.access_token')

# 2. Spam requests untuk trigger rate limit
for i in {1..30}; do
  curl -X POST http://localhost:8087/api/v1/tasks \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"task_type":"email"}'
  echo ""
done
```

### Test dengan Script

```bash
# Run test script
./test_rate_limiter.sh 50

# Output:
# Request 1: ✓ SUCCESS (HTTP 201)
# Request 2: ✓ SUCCESS (HTTP 201)
# ...
# Request 21: ✗ RATE LIMITED (HTTP 429)
# Request 22: ✗ RATE LIMITED (HTTP 429)
```

---

## 📊 Response Format

### ✅ Success (200/201)
```json
{
    "task_id": 123,
    "status": "PENDING"
}
```

### ❌ Rate Limited (429)
```json
{
    "error": "Rate limit exceeded",
    "message": "Maximum 10 requests per second allowed",
    "retry_after": "0.1 seconds"
}
```

---

## 🔍 Monitor di Redis

```bash
# Lihat semua rate limiter keys
redis-cli KEYS "rate_limiter:*"

# Check user tertentu
redis-cli HGETALL rate_limiter:user:1
# Output:
# 1) "tokens"       -> 15.7  (sisa tokens)
# 2) "last_refill"  -> 1703352000 (timestamp)

# Reset rate limit
redis-cli DEL rate_limiter:user:1
```

---

## ⚙️ Configuration Guide

| Use Case | Capacity | Refill Rate | Preset |
|----------|----------|-------------|--------|
| Login (anti brute-force) | 3 | 0.1/sec | `StrictRateLimiter()` |
| Write API (create/update) | 10 | 5/sec | `ConservativeRateLimiter()` |
| General API | 20 | 10/sec | `ModerateRateLimiter()` |
| Read API (get/list) | 100 | 50/sec | `GenerousRateLimiter()` |
| Internal/Admin | 10000 | 1000/sec | `UnlimitedRateLimiter()` |

---

## 🎯 Best Practices

✅ **DO:**
- Use `StrictRateLimiter()` for login/auth endpoints
- Use `ConservativeRateLimiter()` for write operations
- Use `GenerousRateLimiter()` for read operations
- Monitor Redis keys regularly
- Test rate limits before production

❌ **DON'T:**
- Don't use `UnlimitedRateLimiter()` in production
- Don't apply same limit to all endpoints
- Don't forget to test with real traffic patterns

---

## 🐛 Troubleshooting

### Rate limiter tidak bekerja?
```bash
# Check Redis connection
redis-cli PING

# Check if rate limiter keys exist
redis-cli KEYS "rate_limiter:*"

# Check middleware is applied
# Lihat logs saat request masuk
```

### Terlalu ketat?
- Increase `Capacity` (burst)
- Increase `RefillRate` (sustained)

### Terlalu longgar?
- Decrease `Capacity`
- Decrease `RefillRate`

---

## 📚 Files Created

- ✅ `internal/middleware/rate_limiter.go` - Main implementation
- ✅ `internal/middleware/rate_limiter.lua` - Lua script
- ✅ `internal/middleware/rate_limiter_presets.go` - Preset configs
- ✅ `internal/handler/rate_limiter_examples.go` - Usage examples
- ✅ `test_rate_limiter.sh` - Test script
- ✅ `RATE_LIMITER.md` - Full documentation
- ✅ `RATE_LIMITER_QUICKREF.md` - This file

---

## 🚀 Next Steps

1. **Test**: Run `./test_rate_limiter.sh 50`
2. **Customize**: Adjust configs per endpoint needs
3. **Monitor**: Check Redis keys with `redis-cli`
4. **Deploy**: Ready for production!

---

**🎉 Rate Limiter is Ready to Use!**

For detailed documentation, see: `RATE_LIMITER.md`
For code examples, see: `internal/handler/rate_limiter_examples.go`
