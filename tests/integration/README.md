# Integration Tests

This directory contains integration tests for the Async Task Orchestrator system. Integration tests verify end-to-end functionality with real dependencies (PostgreSQL, Redis, RabbitMQ).

## 📋 Overview

### Test Coverage

| Test File | Description | Tests |
|-----------|-------------|-------|
| **setup_test.go** | Test infrastructure & helpers | Setup/teardown, fixtures |
| **api_test.go** | HTTP API integration tests | Auth flow, CRUD operations, rate limiting, caching |
| **worker_test.go** | Worker processing tests | Task execution, retries, concurrency |

## 🎯 Test Scenarios

### API Integration Tests (`api_test.go`)

1. **TestAPIIntegration_FullUserFlow**
   - User registration
   - Login and JWT generation
   - Task creation with authentication
   - Task retrieval with authorization
   - Multi-user isolation

2. **TestAPIIntegration_TokenRefresh**
   - Refresh token flow
   - Token rotation
   - Invalid token handling

3. **TestAPIIntegration_RateLimiting**
   - Rate limit enforcement
   - Per-user rate limiting
   - Token bucket algorithm

4. **TestAPIIntegration_CacheHit**
   - Redis caching behavior
   - Cache hit vs cache miss
   - Performance comparison

### Worker Integration Tests (`worker_test.go`)

1. **TestWorkerIntegration_TaskProcessing**
   - End-to-end task execution
   - RabbitMQ message consumption
   - Database state updates

2. **TestWorkerIntegration_TaskRetry**
   - Retry mechanism
   - Max retry handling
   - Failure conditions

3. **TestWorkerIntegration_ConcurrentProcessing**
   - Multiple workers
   - Concurrent task execution
   - Load distribution

4. **TestWorkerIntegration_TaskStateTransitions**
   - State machine: PENDING → PROCESSING → COMPLETED/FAILED
   - Timestamp tracking
   - Data consistency

## 🚀 Running Integration Tests

### Prerequisites

Integration tests require running services:
- PostgreSQL (port 5432)
- Redis (port 6379)
- RabbitMQ (port 5672)

### Option 1: Using Docker Compose (Recommended)

```bash
# Start all services
docker-compose up -d postgres redis rabbitmq

# Run integration tests
make test-integration

# Or with Go directly
go test -v -tags=integration ./tests/integration/...

# Cleanup
docker-compose down
```

### Option 2: Local Services

If you have services installed locally:

```bash
# Export environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=task_db_test
export DB_SSLMODE=disable
export REDIS_HOST=localhost
export REDIS_PORT=6379
export RABBITMQ_URL=amqp://guest:guest@localhost:5672/
export JWT_SECRET=test-integration-secret

# Run tests
make test-integration
```

## 📊 Test Output Example

```
=== RUN   TestAPIIntegration_FullUserFlow
=== RUN   TestAPIIntegration_FullUserFlow/Register
    api_test.go:68: ✅ User registered successfully: testuser_1703600000
=== RUN   TestAPIIntegration_FullUserFlow/Login
    api_test.go:95: ✅ User logged in successfully, token: eyJhbGciOiJIUzI1NiIs...
=== RUN   TestAPIIntegration_FullUserFlow/CreateTask
    api_test.go:124: ✅ Task created successfully: ID=1
=== RUN   TestAPIIntegration_FullUserFlow/GetTask_OwnTask
    api_test.go:146: ✅ Retrieved task successfully
=== RUN   TestAPIIntegration_FullUserFlow/GetUserTasks
    api_test.go:175: ✅ Retrieved 1 tasks for user
=== RUN   TestAPIIntegration_FullUserFlow/Unauthorized_NoToken
    api_test.go:191: ✅ Properly rejected unauthorized request
=== RUN   TestAPIIntegration_FullUserFlow/Unauthorized_InvalidToken
    api_test.go:203: ✅ Properly rejected invalid token
=== RUN   TestAPIIntegration_FullUserFlow/Forbidden_OtherUserTask
    api_test.go:251: ✅ Properly rejected unauthorized access
--- PASS: TestAPIIntegration_FullUserFlow (2.34s)
PASS
ok      task_handler/tests/integration  3.567s
```

## 🏗️ Test Architecture

### Build Tags

Integration tests use Go build tags to prevent running with unit tests:

```go
//go:build integration
// +build integration
```

This ensures:
- ✅ Unit tests run fast without external dependencies
- ✅ Integration tests only run when explicitly requested
- ✅ CI/CD can run different test suites separately

### Test Structure

```go
func TestAPIIntegration_Feature(t *testing.T) {
    // 1. Setup: Initialize test environment
    deps := SetupTestEnvironment(t)
    defer deps.Cleanup(t)  // Ensure cleanup happens

    // 2. Arrange: Prepare test data
    // ...

    // 3. Act: Execute operation
    // ...

    // 4. Assert: Verify results
    assert.Equal(t, expected, actual)
    
    // 5. Log: Provide visibility
    t.Logf("✅ Test passed with result: %v", result)
}
```

### Test Helpers

**SetupTestEnvironment(t)**
- Creates isolated test database
- Initializes Redis client
- Connects to RabbitMQ
- Runs migrations
- Returns `*TestDependencies`

**Cleanup(t)**
- Truncates database tables
- Flushes Redis cache
- Purges RabbitMQ queues
- Closes all connections

**WaitForCondition(t, condition, timeout, message)**
- Polls until condition is true
- Useful for async operations
- Fails test on timeout

## 🎓 Best Practices

### ✅ DO

1. **Use unique test data**
   ```go
   testUsername := fmt.Sprintf("testuser_%d", time.Now().Unix())
   ```
   Prevents conflicts in concurrent test runs

2. **Always cleanup**
   ```go
   defer deps.Cleanup(t)
   ```
   Ensures test isolation

3. **Test real scenarios**
   - Full user workflows
   - Error conditions
   - Edge cases

4. **Verify side effects**
   - Database changes
   - Cache updates
   - Queue messages

5. **Use descriptive subtests**
   ```go
   t.Run("CreateTask_Success", func(t *testing.T) { ... })
   ```

### ❌ DON'T

1. **Don't share state between tests**
   - Each test should be independent
   - No global variables

2. **Don't hardcode IDs**
   - Use generated/returned IDs
   - Adapt to test environment

3. **Don't skip cleanup**
   - Always use `defer deps.Cleanup(t)`
   - Handle cleanup errors gracefully

4. **Don't ignore timing**
   - Use `WaitForCondition` for async ops
   - Don't rely on fixed sleep durations

## 🔍 Debugging Integration Tests

### View Docker Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f postgres
docker-compose logs -f redis
docker-compose logs -f rabbitmq
```

### Check Service Health

```bash
# PostgreSQL
docker-compose exec postgres pg_isready

# Redis
docker-compose exec redis redis-cli ping

# RabbitMQ
curl http://localhost:15672/api/health/checks/alarms
```

### Run Single Test

```bash
# Run specific test
go test -v -tags=integration ./tests/integration -run TestAPIIntegration_FullUserFlow

# With debug output
go test -v -tags=integration ./tests/integration -run TestAPIIntegration_FullUserFlow -test.v
```

### Database Inspection

```bash
# Connect to test database
docker-compose exec postgres psql -U postgres -d task_db_test

# Check tables
\dt

# Query data
SELECT * FROM tasks;
SELECT * FROM users;
```

## 📈 Coverage

Integration tests complement unit tests:

| Test Type | Coverage | Speed | Dependencies |
|-----------|----------|-------|--------------|
| **Unit Tests** | Logic, algorithms | Fast (< 3s) | None |
| **Integration Tests** | End-to-end flows | Slower (10-30s) | Docker services |

Combined coverage target: **70-80%**

## 🚦 CI/CD Integration

Integration tests run in GitHub Actions:

```yaml
integration-test:
  runs-on: ubuntu-latest
  services:
    postgres: ...
    redis: ...
    rabbitmq: ...
  
  steps:
    - name: Run integration tests
      run: make test-integration-ci
```

## 📚 Resources

- [Testing in Go](https://go.dev/doc/tutorial/add-a-test)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Docker Compose for Tests](https://docs.docker.com/compose/)
- [Build Tags](https://pkg.go.dev/cmd/go#hdr-Build_constraints)

## 🤝 Contributing

When adding new integration tests:

1. Follow existing structure
2. Use build tags
3. Add cleanup logic
4. Document test purpose
5. Update this README

---

**Need help?** Check the [main TESTING.md](../../TESTING.md) or open an issue.
