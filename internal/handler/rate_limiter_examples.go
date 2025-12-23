package handler

/*
CONTOH PENGGUNAAN RATE LIMITER
================================

1. BASIC USAGE (Default untuk semua endpoint di group)
-------------------------------------------------------
api := r.Group("/api/v1")
api.Use(middleware.AuthMiddleware(jwtSecret))
api.Use(middleware.RateLimiterMiddleware(redisClient, middleware.DefaultRateLimiterConfig()))
{
    api.POST("/tasks", taskCtrl.CreateTask)
    api.GET("/tasks/:id", taskCtrl.GetTask)
}


2. DIFFERENT RATE LIMITS PER ENDPOINT
--------------------------------------
api := r.Group("/api/v1")
api.Use(middleware.AuthMiddleware(jwtSecret))
{
    // Strict rate limit for create (5 burst, 1 req/sec)
    api.POST("/tasks",
        middleware.RateLimiterMiddleware(redisClient, middleware.StrictRateLimiter()),
        taskCtrl.CreateTask,
    )

    // Generous rate limit for read (100 burst, 50 req/sec)
    api.GET("/tasks/:id",
        middleware.RateLimiterMiddleware(redisClient, middleware.GenerousRateLimiter()),
        taskCtrl.GetTask,
    )

    // Conservative for list (10 burst, 5 req/sec)
    api.GET("/users/:user_id/tasks",
        middleware.RateLimiterMiddleware(redisClient, middleware.ConservativeRateLimiter()),
        taskCtrl.GetTasksByUser,
    )
}


3. CUSTOM RATE LIMITS
----------------------
api := r.Group("/api/v1")
api.Use(middleware.AuthMiddleware(jwtSecret))
{
    // Custom: 15 burst, 7 requests per second
    customConfig := middleware.CustomRateLimiter(15, 7.0)
    api.POST("/tasks",
        middleware.RateLimiterMiddleware(redisClient, customConfig),
        taskCtrl.CreateTask,
    )
}


4. MIXED: SOME WITH RATE LIMIT, SOME WITHOUT
----------------------------------------------
// Protected with rate limit
apiLimited := r.Group("/api/v1")
apiLimited.Use(middleware.AuthMiddleware(jwtSecret))
apiLimited.Use(middleware.RateLimiterMiddleware(redisClient, middleware.DefaultRateLimiterConfig()))
{
    apiLimited.POST("/tasks", taskCtrl.CreateTask)
}

// Protected but NO rate limit (for internal/admin)
apiUnlimited := r.Group("/api/v1/admin")
apiUnlimited.Use(middleware.AuthMiddleware(jwtSecret))
{
    apiUnlimited.GET("/stats", adminCtrl.GetStats) // No rate limit
}


5. RATE LIMIT ON AUTH ENDPOINTS (Anti Brute-Force)
---------------------------------------------------
authGroup := r.Group("/auth")
{
    // No rate limit for register
    authGroup.POST("/register", userCtrl.Register)

    // Strict rate limit for login (anti brute-force)
    authGroup.POST("/login",
        middleware.RateLimiterMiddleware(redisClient, middleware.StrictRateLimiter()),
        userCtrl.Login,
    )

    // Moderate rate limit for refresh token
    authGroup.POST("/refresh",
        middleware.RateLimiterMiddleware(redisClient, middleware.ConservativeRateLimiter()),
        userCtrl.RefreshToken,
    )
}


6. PRESET CONFIGURATIONS
-------------------------
// Strict: 3 burst, 0.1 req/sec (1 req per 10 seconds)
middleware.StrictRateLimiter()

// Conservative: 10 burst, 5 req/sec
middleware.ConservativeRateLimiter()

// Moderate: 20 burst, 10 req/sec (DEFAULT)
middleware.ModerateRateLimiter()
middleware.DefaultRateLimiterConfig()  // Same as above

// Generous: 100 burst, 50 req/sec
middleware.GenerousRateLimiter()

// Unlimited: 10000 burst, 1000 req/sec (development only!)
middleware.UnlimitedRateLimiter()

// Custom: Your own values
middleware.CustomRateLimiter(capacity, refillRate)


7. PRODUCTION EXAMPLE
----------------------
func SetupHandler(db *sql.DB, conn *amqp091.Connection, redisClient *redis.Client, cfg *config.Config) *gin.Engine {
    r := gin.Default()

    // Initialize controllers...

    // Auth endpoints with anti-brute-force
    authGroup := r.Group("/auth")
    {
        authGroup.POST("/register", userCtrl.Register)
        authGroup.POST("/login",
            middleware.RateLimiterMiddleware(redisClient, middleware.StrictRateLimiter()),
            userCtrl.Login,
        )
        authGroup.POST("/refresh", userCtrl.RefreshToken)
    }

    // Public API with conservative limits
    publicAPI := r.Group("/api/v1/public")
    publicAPI.Use(middleware.RateLimiterMiddleware(redisClient, middleware.ConservativeRateLimiter()))
    {
        publicAPI.GET("/health", healthCheck)
    }

    // Protected API with moderate limits
    api := r.Group("/api/v1")
    api.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
    api.Use(middleware.RateLimiterMiddleware(redisClient, middleware.ModerateRateLimiter()))
    {
        // Write operations - more strict
        api.POST("/tasks",
            middleware.RateLimiterMiddleware(redisClient, middleware.ConservativeRateLimiter()),
            taskCtrl.CreateTask,
        )

        // Read operations - more generous
        api.GET("/tasks/:id", taskCtrl.GetTask)
        api.GET("/users/:user_id/tasks", taskCtrl.GetTasksByUser)
    }

    return r
}


8. UNDERSTANDING THE RESPONSE
------------------------------
// Success (200/201)
{
    "task_id": 123,
    "status": "PENDING"
}

// Rate Limited (429)
{
    "error": "Rate limit exceeded",
    "message": "Maximum 10 requests per second allowed",
    "retry_after": "0.1 seconds"
}


9. TESTING
----------
# Quick test with curl
TOKEN="your_jwt_token"
for i in {1..30}; do
  curl -X POST http://localhost:8087/api/v1/tasks \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"task_type":"email"}'
  echo ""
done

# Or use the test script
./test_rate_limiter.sh 50


10. MONITORING IN REDIS
------------------------
# Check rate limiter keys
redis-cli KEYS "rate_limiter:*"

# Check specific user's rate limiter
redis-cli HGETALL rate_limiter:user:1

# Output:
# 1) "tokens"       -> Current available tokens
# 2) "15.7"
# 3) "last_refill"  -> Last refill timestamp
# 4) "1703352000"

# Reset rate limit for user
redis-cli DEL rate_limiter:user:1

*/
