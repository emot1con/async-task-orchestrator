# Async Task Orchestrator

![CI](https://github.com/emot1con/task_handler/workflows/CI/badge.svg)
![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Coverage](https://img.shields.io/codecov/c/github/emot1con/task_handler)](https://codecov.io/gh/emot1con/task_handler)

**Production-ready asynchronous task processing system with Go, RabbitMQ, Redis rate limiting, and JWT auth**

A scalable, distributed task orchestration platform built with Go, featuring comprehensive security (JWT authentication, rate limiting), robust infrastructure (Docker, PostgreSQL, Redis, RabbitMQ), and complete CI/CD pipeline with unit and integration tests.

## Features

### Core Functionality
- **Asynchronous Task Processing** - Non-blocking task execution with RabbitMQ
- **Distributed Workers** - Scalable worker pool for parallel task processing
- **Task Status Tracking** - Real-time task status monitoring (PENDING, PROCESSING, SUCCESS, FAILED)
- **Multiple Task Types** - Support for `send_email`, `generate_report`, `resize_image`, `cleanup_temp`

### Security & Performance
- **JWT Authentication** - Secure access/refresh token implementation (HS256)
  - Access tokens: 15 minutes
  - Refresh tokens: 7 days
- **Ownership Authorization** - Users can only access their own resources (403 Forbidden on violation)
- **Rate Limiting** - Redis-based Token Bucket algorithm with atomic Lua scripts
  - IP-based rate limiting for auth endpoints
  - User-based rate limiting for API endpoints
  - Configurable burst capacity and refill rate
- **Redis Caching** - Fast task/user data access with connection pooling

### Testing & Quality
- **Comprehensive Test Coverage** - Unit tests + Integration tests
- **CI/CD Pipeline** - Automated testing, building, and security scanning
- **Separate Test Jobs** - Unit tests (fast) vs Integration tests (with services)
- **Code Coverage** - Codecov integration with coverage reports

### Infrastructure
- **Docker Compose** - Complete containerized setup (API, Worker, PostgreSQL, Redis, RabbitMQ)
- **PostgreSQL** - Reliable persistent storage with migrations
- **Health Checks** - Container health monitoring
- **Structured Logging** - Comprehensive application logging with logrus

## Table of Contents
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Documentation](#api-documentation)
- [Rate Limiting](#rate-limiting)
- [Development](#development)
- [Testing](#testing)
- [Project Structure](#project-structure)

## Architecture

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Client    │─────▶│  Nginx      │─────▶│   API       │
│             │◀─────│(Rate Limit) │◀─────│  (Gin)      │
└─────────────┘      └─────────────┘      └──────┬──────┘
                                                  │
                     ┌────────────────────────────┼────────────────┐
                     │                            │                │
                     ▼                            ▼                ▼
              ┌─────────────┐            ┌─────────────┐   ┌──────────┐
              │  PostgreSQL │            │   Redis     │   │ RabbitMQ │
              │  (Storage)  │            │(Cache/Rate) │   │ (Queue)  │
              └─────────────┘            └─────────────┘   └────┬─────┘
                                                                 │
                                                                 ▼
                                                          ┌─────────────┐
                                                          │   Worker    │
                                                          │   Pool      │
                                                          └─────────────┘
```

### Components

1. **API Server** (`cmd/api/main.go`)
   - RESTful API with Gin framework
   - JWT middleware for authentication
   - Rate limiter middleware (Redis Token Bucket)
   - Task creation and status endpoints

2. **Worker** (`cmd/worker/main.go`)
   - RabbitMQ consumer
   - Task processing engine
   - Status update publisher

3. **PostgreSQL**
   - Users table (authentication)
   - Tasks table (task tracking with timestamps)

4. **Redis**
   - Rate limiting state (Token Bucket per user)
   - Session caching (optional)

5. **RabbitMQ**
   - Task queue (task.created)
   - Async communication between API and workers

## Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Make (optional, for convenience commands)

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/async-task-orchestrator.git
cd async-task-orchestrator
```

### 2. Setup Environment Variables

```bash
cp .env.example .env
# Edit .env with your configuration (default values work for development)
```

### 3. Start All Services

```bash
# Using Make
make up_build

# Or using Docker Compose directly
docker-compose up -d --build
```

### 4. Verify Services

```bash
# Check all containers are running
docker-compose ps

# Expected output:
# NAME       STATE    PORTS
# api        Up       0.0.0.0:8087->8087/tcp
# worker     Up       
# postgres   Up       0.0.0.0:5432->5432/tcp
# redis      Up       0.0.0.0:6379->6379/tcp
# rabbitmq   Up       0.0.0.0:5672->5672/tcp, 0.0.0.0:15672->15672/tcp
```

### 5. Test the API

```bash
# Register a user
curl -X POST http://localhost:8087/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'

# Login to get JWT token
curl -X POST http://localhost:8087/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'

# Create a task (replace <TOKEN> with access_token from login)
curl -X POST http://localhost:8087/api/v1/tasks \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "task_type": "IMAGE_RESIZE"
  }'

# Get task status
curl -X GET http://localhost:8087/api/v1/tasks/1 \
  -H "Authorization: Bearer <TOKEN>"
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_NAME` | Application name | `event-driven-task` |
| `APP_ENV` | Environment (local/staging/production) | `local` |
| `APP_PORT` | API server port | `8080` |
| `DB_HOST` | PostgreSQL host | `postgres` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | `postgres` |
| `DB_NAME` | Database name | `task_db` |
| `REDIS_HOST` | Redis host | `redis` |
| `REDIS_PORT` | Redis port | `6379` |
| `RABBITMQ_URL` | RabbitMQ connection URL | `amqp://guest:guest@rabbitmq:5672/` |
| `JWT_SECRET` | JWT signing secret | `supersecret` |

**Security Note**: Change `JWT_SECRET` in production!

## API Documentation

### Authentication Endpoints

#### Register User
```http
POST /auth/register
Content-Type: application/json

{
  "username": "johndoe",
  "password": "securepass123"
}

Response: 201 Created
{
  "user_id": 1,
  "message": "User registered successfully"
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "username": "johndoe",
  "password": "securepass123"
}

Response: 200 OK
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

#### Refresh Token
```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}

Response: 200 OK
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

### Task Endpoints (Protected)

All task endpoints require `Authorization: Bearer <access_token>` header.

#### Create Task
```http
POST /api/v1/tasks
Authorization: Bearer <token>
Content-Type: application/json

{
  "task_type": "IMAGE_RESIZE"
}

Response: 201 Created
{
  "task_id": 1,
  "status": "PENDING",
  "message": "Task created successfully"
}
```

**Available Task Types:**
- `IMAGE_RESIZE`
- `VIDEO_PROCESS`
- `DATA_EXPORT`
- `REPORT_GENERATE`

#### Get Task by ID
```http
GET /api/v1/tasks/:id
Authorization: Bearer <token>

Response: 200 OK
{
  "id": 1,
  "user_id": 1,
  "task_type": "IMAGE_RESIZE",
  "status": "COMPLETED",
  "result_file": "result/image_123.jpg",
  "error_message": null,
  "created_at": "2024-12-25T10:30:00Z",
  "updated_at": "2024-12-25T10:30:45Z"
}
```

**Authorization**: Users can only view their own tasks (403 Forbidden if accessing others' tasks)

#### Get User's Tasks
```http
GET /api/v1/users/:user_id/tasks
Authorization: Bearer <token>

Response: 200 OK
{
  "tasks": [
    {
      "id": 1,
      "user_id": 1,
      "task_type": "IMAGE_RESIZE",
      "status": "COMPLETED",
      "created_at": "2024-12-25T10:30:00Z"
    },
    {
      "id": 2,
      "user_id": 1,
      "task_type": "DATA_EXPORT",
      "status": "PROCESSING",
      "created_at": "2024-12-25T10:35:00Z"
    }
  ]
}
```

**Authorization**: Users can only list their own tasks

### Task Status Flow

```
PENDING → PROCESSING → COMPLETED
                    ↘ FAILED
```

## Rate Limiting

This project implements defense-in-depth rate limiting strategy:

### 1. Nginx Layer (IP-based)
- Auth endpoints (`/auth/*`): 1 request/second, burst=3
- API endpoints (`/api/v1/*`): 300 requests/second, burst=10
- Protects against DDoS and brute-force attacks

### 2. Application Layer (User-based)
- Token Bucket Algorithm implemented with Redis + Lua
- Per-user rate limiting with configurable presets
- Atomic operations for distributed safety

#### Rate Limit Presets

| Preset | Capacity | Refill Rate | Use Case |
|--------|----------|-------------|----------|
| Strict | 3 | 0.1/sec | Anti-abuse |
| Conservative | 10 | 5/sec | Production |
| Moderate | 20 | 10/sec | Default |
| Generous | 100 | 50/sec | Heavy users |

#### Rate Limit Response

When rate limit is exceeded:
```json
HTTP 429 Too Many Requests
{
  "error": "Rate limit exceeded"
}
```

See [RATE_LIMITER.md](RATE_LIMITER.md) for detailed documentation.

## Development

### Local Development (without Docker)

1. Start infrastructure services:
```bash
docker-compose up postgres redis rabbitmq -d
```

2. Run migrations:
```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/task_db?sslmode=disable" up
```

3. Run API server:
```bash
go run cmd/api/main.go
```

4. Run worker (in another terminal):
```bash
go run cmd/worker/main.go
```

### Available Make Commands

```bash
make up_build    # Build and start all services
make app         # Rebuild and restart API + Worker
make api         # Rebuild and restart API only
make worker      # Rebuild and restart Worker only
```

### Adding New Task Types

1. Add task type constant in `internal/task/model.go`
2. Implement processing logic in `internal/worker/proc.go`
3. Update API validation in `internal/task/controller.go`

### Database Migrations

Create new migration:
```bash
migrate create -ext sql -dir migrations -seq add_new_feature
```

This creates:
- `XXX_add_new_feature.up.sql` - forward migration
- `XXX_add_new_feature.down.sql` - rollback migration

## Testing

The project includes comprehensive test coverage with both **unit tests** and **integration tests**:

### Test Structure

#### Unit Tests
- **Rate Limiter** - Token Bucket algorithm with Redis
- **JWT Authentication** - Token generation, validation, refresh
- **Authorization** - Ownership-based access control
- **Task Business Logic** - Task creation and validation

#### Integration Tests
- **Auth Flow** - Full registration, login, token refresh cycle
- **Task CRUD** - Complete task lifecycle with real dependencies
- **Cache Testing** - Redis caching behavior verification
- **Ownership & Security** - Cross-user access prevention
- **All Task Types** - `send_email`, `generate_report`, `resize_image`, `cleanup_temp`

### Running Tests Locally

```bash
# Run unit tests only
make test
# or
go test ./... -v

# Run integration tests (requires running services)
docker-compose up -d postgres redis rabbitmq

# Create test database
docker exec postgres psql -U postgres -c "CREATE DATABASE task_db_test;"

# Run integration tests
go test -v -tags=integration ./tests/integration/...

# Run all tests with coverage
make test-coverage
```

### CI/CD Pipeline

The project uses **GitHub Actions** with separate jobs for different test types:

```
┌────────────────┐     ┌──────────────────────┐     ┌──────────┐
│  Unit Test     │     │  Integration Test    │     │   Lint   │
│                │     │  • PostgreSQL        │     │          │
│  (Fast, no     │     │  • Redis             │     │ golangci │
│   services)    │     │  • RabbitMQ          │     │          │
└────────┬───────┘     └──────────┬───────────┘     └────┬─────┘
         │                        │                       │
         └────────────────────────┴───────────────────────┘
                                  │
                           ┌──────▼──────┐
                           │    Build    │
                           │   Docker    │
                           └──────┬──────┘
                                  │
                           ┌──────▼──────┐
                           │  Security   │
                           │   gosec     │
                           └─────────────┘
```

**On every push/PR:**
- ✅ Unit tests (no external dependencies)
- ✅ Integration tests (with PostgreSQL, Redis, RabbitMQ)
- ✅ Linting with golangci-lint
- ✅ Docker image builds (API + Worker)
- ✅ Security scanning with gosec
- ✅ Code coverage uploaded to Codecov

### Test Coverage

- **Unit Tests**: Core business logic and utilities
- **Integration Tests**: Full API + Database + Cache + Queue interactions
- Coverage reports available in GitHub Actions artifacts

### Manual E2E Testing

```bash
# Test auth endpoint rate limiting
./test_auth_rate_limiter.sh

# Test ownership authorization
./test_ownership.sh
```

### Manual Testing

```bash
# Test task creation and processing
curl -X POST http://localhost:8087/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"task_type": "IMAGE_RESIZE"}'

# Watch task status change from PENDING → PROCESSING → COMPLETED
watch -n 1 'curl -s http://localhost:8087/api/v1/tasks/1 -H "Authorization: Bearer $TOKEN" | jq .status'
```

### RabbitMQ Management UI

Access at http://localhost:15672
- Username: `guest`
- Password: `guest`

Monitor:
- Queue depth
- Message rates
- Consumer status

## Project Structure

```
task_handler/
├── cmd/
│   ├── api/           # API server entry point
│   │   └── main.go
│   └── worker/        # Worker service entry point
│       └── main.go
├── internal/
│   ├── auth/          # JWT authentication utilities
│   ├── cache/         # Redis cache client
│   ├── config/        # Configuration management
│   ├── db/            # PostgreSQL client
│   ├── handler/       # HTTP route handlers
│   ├── logger/        # Structured logging (logrus)
│   ├── middleware/    # JWT & Rate limiter middleware
│   ├── observability/ # Metrics & tracing
│   ├── queue/         # RabbitMQ client
│   ├── task/          # Task domain (model, repo, controller)
│   │   ├── model.go
│   │   ├── repository.go
│   │   └── controller.go
│   ├── user/          # User domain (auth, registration)
│   │   ├── model.go
│   │   ├── repository.go
│   │   ├── service.go
│   │   └── controller.go
│   └── worker/        # Task processing logic
│       └── proc.go
├── tests/
│   └── integration/   # Integration test suite
│       ├── setup_test.go      # Test environment setup
│       ├── auth_test.go       # Auth flow tests
│       ├── task_test.go       # Task CRUD tests
│       └── cache_test.go      # Cache behavior tests
├── migrations/        # Database migrations
│   ├── 001_create_tasks.up.sql
│   └── 002_add_task_column.up.sql
├── docker/            # Dockerfiles
│   ├── api.Dockerfile
│   └── worker.Dockerfile
├── .github/
│   └── workflows/
│       └── ci.yml     # CI/CD pipeline (unit + integration tests)
├── reports/           # Task processing results
├── docker-compose.yaml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### Key Files

- **`cmd/api/main.go`** - API server bootstrap
- **`cmd/worker/main.go`** - Worker service bootstrap
- **`internal/middleware/rate_limiter.go`** - Token Bucket implementation
- **`internal/middleware/rate_limiter.lua`** - Atomic Redis operations
- **`internal/task/controller.go`** - Task HTTP handlers with ownership checks
- **`internal/user/controller.go`** - Auth endpoints (register, login, refresh)
- **`internal/worker/proc.go`** - Task processing logic (send_email, generate_report, etc.)
- **`tests/integration/`** - Complete integration test suite
- **`migrations/`** - SQL schema definitions
- **`.github/workflows/ci.yml`** - Automated CI/CD with unit + integration tests

## Security Features

**Authentication**
- JWT with HS256 signing
- Access tokens (15 min) + Refresh tokens (7 days)
- Token rotation on refresh

**Authorization**
- Ownership-based access control
- Users can only access their own resources
- 403 Forbidden for unauthorized access

**Rate Limiting**
- Multi-layer protection (Nginx + App)
- IP-based for auth endpoints
- User-based for API endpoints
- Token Bucket algorithm with Redis

**Input Validation**
- Request body validation
- Username format checks
- SQL injection prevention (prepared statements)

**Secure Defaults**
- HTTPS ready (configure reverse proxy)
- Secure password hashing (bcrypt)
- SQL injection protection
- XSS prevention

## Production Deployment

### Recommended Setup

1. Use environment-specific configs:
```bash
APP_ENV=production
JWT_SECRET=<generate-strong-secret>
```

2. Enable HTTPS:
   - Use Nginx/Traefik as reverse proxy
   - Configure SSL/TLS certificates (Let's Encrypt)

3. Scale workers:
```bash
docker-compose up -d --scale worker=3
```

4. Monitor services:
   - Add health check endpoints
   - Configure monitoring system
   - Set up log aggregation

5. Database backups:
```bash
docker exec postgres pg_dump -U postgres task_db > backup.sql
```

### Performance Tuning

- Redis: Enable persistence (RDB/AOF) for rate limit state
- PostgreSQL: Add indexes on `user_id` and `status` columns
- RabbitMQ: Adjust prefetch count for workers
- Nginx: Tune worker_processes and connections

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [RabbitMQ](https://www.rabbitmq.com/)
- [Redis](https://redis.io/)
- [PostgreSQL](https://www.postgresql.org/)
- [golang-jwt](https://github.com/golang-jwt/jwt)
