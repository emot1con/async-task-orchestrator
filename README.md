# Async Task Orchestrator

![CI](https://github.com/emot1con/task_handler/workflows/CI/badge.svg)
![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Coverage](https://img.shields.io/codecov/c/github/emot1con/task_handler)](https://codecov.io/gh/emot1con/task_handler)

**Production-ready asynchronous task processing system with Go, RabbitMQ auto-reconnect, Redis rate limiting, JWT auth, and ELK Stack observability**

A scalable, distributed task orchestration platform built with Go, featuring comprehensive security (JWT authentication, rate limiting), robust infrastructure (Docker, PostgreSQL, Redis, RabbitMQ with auto-reconnect), centralized logging with ELK Stack (Elasticsearch, Logstash, Kibana, Filebeat), and complete CI/CD pipeline with unit and integration tests.

## Features

### Core Functionality
- **Asynchronous Task Processing** - Non-blocking task execution with RabbitMQ
- **Distributed Workers** - Scalable worker pool for parallel task processing
  - **Task Worker** - Processes task-related messages (3 concurrent workers)
  - **Notification Worker** - Handles notification delivery (3 concurrent workers)
- **Task Status Tracking** - Real-time task status monitoring (PENDING, PROCESSING, SUCCESS, FAILED)
- **Multiple Task Types** - Support for `send_email`, `generate_report`, `resize_image`, `cleanup_temp`
- **RabbitMQ Auto-Reconnect** - Resilient connection management with automatic recovery
  - Connection monitoring with exponential backoff retry (1s to 10s over 10 attempts)
  - Worker restart loops with graceful degradation
  - Thread-safe connection pooling with `sync.RWMutex`
  - 3-layer connection validation (exists, not closed, can create channel)

### Observability & Monitoring
- **ELK Stack Integration** - Centralized logging and real-time analysis
  - **Elasticsearch 8.11.0** - High-performance log storage and search engine
  - **Logstash 8.11.0** - Log processing pipeline with JSON parsing and fingerprinting
  - **Kibana 8.11.0** - Real-time log visualization and analytics dashboard (http://localhost:5601)
  - **Filebeat 8.11.0** - Docker container log shipping with autodiscovery
- **Structured Logging** - JSON-formatted logs with automatic service identification
- **Log Deduplication** - SHA256 fingerprinting prevents duplicate log entries
- **Selective Log Collection** - Docker labels (`logging.enabled=true`) for filtered collection

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
- **Docker Compose** - Complete containerized setup with 11 services:
  - Application: API, Task Worker, Notification Worker
  - Data Storage: PostgreSQL 15, Redis
  - Message Queue: RabbitMQ with Management UI
  - ELK Stack: Elasticsearch, Logstash, Kibana, Filebeat
  - Database Migration: Automated schema management
- **PostgreSQL** - Reliable persistent storage with migrations
- **Health Checks** - Container health monitoring for Elasticsearch and PostgreSQL
- **Structured Logging** - Comprehensive application logging with logrus and ELK Stack
- **Service Labels** - Docker labels for selective log collection (`logging.enabled=true`)

## Table of Contents
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Documentation](#api-documentation)
- [Rate Limiting](#rate-limiting)
- [Observability & Logging](#observability--logging)
- [RabbitMQ Auto-Reconnect](#rabbitmq-auto-reconnect)
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
                                    ┌────────────────────────────┼─────────────┐
                                    │                            │             │
                                    ▼                            ▼             ▼
                             ┌─────────────┐            ┌──────────────┐  ┌──────────────┐
                             │    Task     │            │Notification  │  │  RabbitMQ    │
                             │   Worker    │            │   Worker     │  │  Manager     │
                             │ (3 workers) │            │ (3 workers)  │  │(Auto-Recon.) │
                             └──────┬──────┘            └──────┬───────┘  └──────────────┘
                                    │                          │
                                    └──────────┬───────────────┘
                                               │
                                               ▼
                                    ┌─────────────────────┐
                                    │     ELK Stack       │
                                    ├─────────────────────┤
                                    │  Filebeat (Shipper) │
                                    │         ↓           │
                                    │ Logstash (Process)  │
                                    │         ↓           │
                                    │Elasticsearch (Store)│
                                    │         ↓           │
                                    │  Kibana (Visualize) │
                                    └─────────────────────┘
```

### Components

1. **API Server** (`cmd/api/main.go`)
   - RESTful API with Gin framework
   - JWT middleware for authentication
   - Rate limiter middleware (Redis Token Bucket)
   - Task creation and status endpoints
   - RabbitMQManager integration for resilient messaging

2. **Task Worker** (`cmd/task/main.go`)
   - RabbitMQ consumer for `task_queue`
   - Task processing engine (generate_report, resize_image, send_email, cleanup_temp)
   - Status update publisher to notification queue
   - Auto-reconnect with worker restart loop

3. **Notification Worker** (`cmd/notification/main.go`)
   - RabbitMQ consumer for `notification_queue`
   - Email notification delivery
   - Event-driven notification processing
   - Auto-reconnect with worker restart loop

4. **PostgreSQL**
   - Users table (authentication data)
   - Tasks table (task tracking with timestamps)
   - Health checks and automated migrations

5. **Redis**
   - Rate limiting state (Token Bucket per user/IP)
   - Task caching for performance optimization
   - Connection pooling

6. **RabbitMQ with Auto-Reconnect**
   - Task queue (`task_queue`)
   - Notification queue (`notification_queue`)
   - RabbitMQManager pattern with connection monitoring
   - Exponential backoff retry strategy (1s to 10s)
   - Thread-safe connection pooling with `sync.RWMutex`

7. **ELK Stack**
   - **Filebeat**: Collects logs from Docker containers with autodiscovery (label-based filtering)
   - **Logstash**: Processes JSON logs, deduplicates with SHA256 fingerprinting
   - **Elasticsearch**: Stores logs with index pattern `app-logs-YYYY.MM.dd`
   - **Kibana**: Web UI for log visualization at http://localhost:5601

## Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Make (optional, for convenience commands)

**System Resources:**
- Minimum 4GB RAM (ELK Stack requires ~2GB)
- 10GB disk space for Docker volumes

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

This will start **11 services**: API, Task Worker, Notification Worker, PostgreSQL, Redis, RabbitMQ, Elasticsearch, Logstash, Kibana, Filebeat, and Migrate.

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

# Expected output (11 services):
# NAME                  STATE    PORTS
# api                   Up       0.0.0.0:8087->8087/tcp
# task-worker           Up       
# notification-worker   Up       
# postgres              Up       0.0.0.0:5432->5432/tcp (healthy)
# redis                 Up       0.0.0.0:6379->6379/tcp
# rabbitmq              Up       0.0.0.0:5672->5672/tcp, 0.0.0.0:15672->15672/tcp
# elasticsearch         Up       0.0.0.0:9200->9200/tcp, 0.0.0.0:9300->9300/tcp (healthy)
# logstash              Up       0.0.0.0:5044->5044/tcp, 0.0.0.0:9600->9600/tcp
# kibana                Up       0.0.0.0:5601->5601/tcp
# filebeat              Up
# migrate               Exited (0)

# Access UIs
# - API: http://localhost:8087
# - RabbitMQ Management: http://localhost:15672 (guest/guest)
# - Kibana (Logs): http://localhost:5601
# - Elasticsearch: http://localhost:9200
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
| `SERVICE_NAME` | Service identifier for logging | `api` / `task-worker` / `notification-worker` |
| `GIN_MODE` | Gin framework mode | `release` / `debug` |

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

## Observability & Logging

### ELK Stack Architecture

The system uses ELK Stack for centralized logging and real-time monitoring:

```
Docker Containers (api, task-worker, notification-worker)
    │
    │ JSON logs to stdout
    ├─────────────────────────────────────────────────┐
    │                                                 │
    ▼                                                 ▼
┌─────────────┐                              ┌─────────────┐
│  Filebeat   │◄─────(autodiscovery)─────────│   Docker    │
│  (Shipper)  │      label: logging.enabled  │   Daemon    │
└──────┬──────┘                              └─────────────┘
       │
       │ Ship logs (port 5044)
       ▼
┌─────────────┐
│  Logstash   │
│ (Processor) │
│             │
│ • Parse JSON│
│ • Fingerprint (SHA256)
│ • Deduplicate
└──────┬──────┘
       │
       │ Index logs
       ▼
┌──────────────┐
│Elasticsearch │
│   (Storage)  │
│              │
│ Index: app-logs-YYYY.MM.dd
└──────┬───────┘
       │
       │ Query API
       ▼
┌─────────────┐
│   Kibana    │
│(Visualize)  │
│             │
│ http://localhost:5601
└─────────────┘
```

### Key Features

1. **Automatic Service Tagging**
   - Each service automatically tagged via `SERVICE_NAME` environment variable
   - Services: `api`, `task-worker`, `notification-worker`

2. **Log Deduplication**
   - SHA256 fingerprint created from: timestamp + message + service + worker_id + task_id
   - Elasticsearch uses fingerprint as `document_id` (upsert instead of insert)
   - Prevents duplicate logs even with multiple Filebeat restarts

3. **Selective Collection**
   - Only containers with `logging.enabled=true` label are monitored
   - Reduces noise from infrastructure containers

4. **Structured Logging**
   ```json
   {
     "level": "info",
     "service": "task-worker",
     "worker_id": 1,
     "task_id": 1234,
     "task_type": "generate_report",
     "message": "Processing task",
     "timestamp": "2026-02-13T10:30:00Z"
   }
   ```

### Accessing Logs

**Kibana Dashboard**
```bash
# Open Kibana in browser
open http://localhost:5601

# Create index pattern: app-logs-*
# Set time field: @timestamp
```

**Elasticsearch Query**
```bash
# Get recent logs
curl "http://localhost:9200/app-logs-*/_search?size=10&sort=@timestamp:desc"

# Search by service
curl -X POST "http://localhost:9200/app-logs-*/_search" -H 'Content-Type: application/json' -d'
{
  "query": {
    "match": { "app.service": "task-worker" }
  }
}'

# Search by task_id
curl -X POST "http://localhost:9200/app-logs-*/_search" -H 'Content-Type: application/json' -d'
{
  "query": {
    "match": { "app.task_id": 1234 }
  }
}'
```

### Configuration Files

- **Filebeat**: `internal/logger/filebeat/filebeat.yml`
  - Docker autodiscovery with label filtering
  - Registry persistence for restart resilience

- **Logstash**: `internal/logger/logstash/pipeline/logstash.conf`
  - JSON parsing
  - SHA256 fingerprinting
  - Field cleanup

- **Logger Hook**: `internal/logger/logger.go`
  - Automatic service field injection
  - Structured logging with logrus

## RabbitMQ Auto-Reconnect

### Overview

The system implements a robust auto-reconnect mechanism for RabbitMQ to handle broker failures, restarts, or network issues without manual intervention.

### Architecture

```
┌──────────────────────────────────────────────────┐
│              RabbitMQManager                     │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │  Connection Pool (sync.RWMutex)            │ │
│  │  • Thread-safe access                      │ │
│  │  • Single connection per service           │ │
│  └────────────┬───────────────────────────────┘ │
│               │                                  │
│  ┌────────────▼───────────────────────────────┐ │
│  │  Monitor Goroutine                         │ │
│  │  • Listen to NotifyClose()                 │ │
│  │  • Trigger reconnect on failure            │ │
│  └────────────┬───────────────────────────────┘ │
│               │                                  │
│  ┌────────────▼───────────────────────────────┐ │
│  │  Connect with Exponential Backoff          │ │
│  │  • 10 attempts                             │ │
│  │  • 1s → 2s → 3s → ... → 10s               │ │
│  └────────────┬───────────────────────────────┘ │
│               │                                  │
│  ┌────────────▼───────────────────────────────┐ │
│  │  GetConnection() with Validation           │ │
│  │  • Check connection exists                 │ │
│  │  • Check !IsClosed()                       │ │
│  │  • Verify can create channel               │ │
│  └────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
                        │
                        ▼
        ┌───────────────────────────────┐
        │    Workers (Restart Loop)     │
        │                               │
        │  for {                        │
        │    conn = manager.GetConn()   │
        │    runWorker(conn)            │
        │    // on error, wait 5s       │
        │    sleep(5s)                  │
        │  }                            │
        └───────────────────────────────┘
```

### Key Components

#### 1. **RabbitMQManager** (`internal/queue/rabbitmq.go`)

**Thread-Safe Connection Pool:**
```go
type RabbitMQManager struct {
    config    *config.RabbitMQConfig
    conn      *amqp.Connection
    mu        sync.RWMutex  // Protects concurrent access
    isClosing bool
}
```

**Connection Monitoring:**
```go
func (m *RabbitMQManager) monitorConnection() {
    for {
        closeNotify := conn.NotifyClose(make(chan *amqp.Error))
        closeErr := <-closeNotify  // Blocks until connection closes
        
        if !m.isClosing {
            log.Warn("Connection closed, reconnecting...")
            m.connect()  // Trigger reconnect
        }
    }
}
```

**Exponential Backoff Retry:**
```go
func (m *RabbitMQManager) connect() {
    for i := 0; i < 10; i++ {
        conn, err := amqp.Dial(m.config.URL)
        if err == nil {
            m.conn = conn
            return
        }
        
        delay := time.Second * time.Duration(i+1)  // 1s, 2s, 3s...
        time.Sleep(delay)
    }
}
```

**Connection Validation (3-Layer):**
```go
func (m *RabbitMQManager) GetConnection() *amqp.Connection {
    for {
        conn := m.conn
        
        // Layer 1: Check exists and not closed
        if conn != nil && !conn.IsClosed() {
            // Layer 2: Verify can create channel (AMQP handshake complete)
            testCh, err := conn.Channel()
            if err == nil {
                testCh.Close()  // Layer 3: Cleanup
                return conn     // Connection verified ready!
            }
        }
        
        time.Sleep(500 * time.Millisecond)  // Retry
    }
}
```

#### 2. **Worker Restart Loop** (`internal/consumer/*/worker.go`)

**Infinite Restart Pattern:**
```go
func StartWorker(manager *RabbitMQManager, workerID int) {
    for {
        // Get fresh connection each iteration
        conn := manager.GetConnection()
        
        // Run worker (blocks until error)
        err := runWorker(conn, workerID)
        
        if err != nil {
            log.WithError(err).Error("Worker stopped, restarting in 5s...")
            time.Sleep(5 * time.Second)
        }
    }
}

func runWorker(conn *amqp.Connection, workerID int) error {
    // Create isolated channel
    ch, err := conn.Channel()
    if err != nil {
        return err
    }
    defer ch.Close()
    
    // Declare queue (ensures queue exists after reconnect)
    queue, err := ch.QueueDeclare("task_queue", true, false, false, false, nil)
    
    // Monitor both messages and channel close
    msgs, _ := ch.Consume(queue.Name, "", false, false, false, false, nil)
    closeNotify := ch.NotifyClose(make(chan *amqp.Error))
    
    for {
        select {
        case msg := <-msgs:
            processMessage(msg)
        case closeErr := <-closeNotify:
            return fmt.Errorf("channel closed: %v", closeErr)
        }
    }
}
```

### Usage in Services

**API Service** (`cmd/api/main.go`):
```go
manager := queue.SetupRabbitMQ(&config.RabbitMQ)
defer manager.Close()

// Handler uses manager to get fresh connection when publishing
handler := handler.SetupHandler(db, manager, redis, config)
```

**Task Service** (`internal/domain/task/service.go`):
```go
func (s *TaskService) CreateTask(task *Task) error {
    // Get fresh connection from manager
    conn := s.manager.GetConnection()
    ch, err := queue.CreateChannel(conn)
    defer ch.Close()
    
    // Publish with confidence (connection is validated)
    return queue.Publish(ch, "task_queue", eventJSON)
}
```

**Worker Services** (`cmd/task/main.go`, `cmd/notification/main.go`):
```go
manager := queue.SetupRabbitMQ(&config.RabbitMQ)
defer manager.Close()

// Start 3 workers with restart loops
for i := 1; i <= 3; i++ {
    go worker.StartWorker(manager, i)
}
```

### Failure Scenarios

| Scenario | Behavior | Recovery Time |
|----------|----------|---------------|
| RabbitMQ restart | Manager detects close → retries with backoff | 1-10 seconds |
| Network partition | Manager retries 10 times with exponential backoff | Up to 55 seconds |
| Worker crash | Worker restart loop catches error → wait 5s → restart | 5 seconds |
| Connection stale | GetConnection() validates → waits until ready | 500ms per retry |
| Broker overload | Exponential backoff prevents thundering herd | Gradual recovery |

### Testing Auto-Reconnect

```bash
# Test RabbitMQ restart recovery
docker stop rabbitmq
sleep 10
docker logs task-worker --tail 20  # Should show retry logs

docker start rabbitmq
sleep 15
docker logs task-worker --tail 20  # Should show "Worker started successfully"

# Test task creation after reconnect
curl -X POST "http://localhost:8087/task-handler/api/v1/tasks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"task_type": "generate_report"}'
# Should return 201 Created (not 504 error)
```

### Benefits

✅ **Zero Downtime** - Workers automatically reconnect without manual intervention  
✅ **No Lost Messages** - Durable queues + manual ACK ensures message persistence  
✅ **Thread-Safe** - `sync.RWMutex` protects concurrent connection access  
✅ **Production-Ready** - Handles all failure scenarios gracefully  
✅ **Gradual Recovery** - Exponential backoff prevents thundering herd  

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
│   ├── api/                    # API server entry point
│   │   └── main.go
│   ├── task/                   # Task worker service
│   │   └── main.go
│   └── notification/           # Notification worker service
│       └── main.go
├── internal/
│   ├── auth/                   # JWT authentication utilities
│   │   ├── jwt.go
│   │   └── password.go
│   ├── cache/                  # Redis cache client
│   │   └── task_cache.go
│   ├── config/                 # Configuration management
│   │   └── config.go
│   ├── consumer/               # RabbitMQ consumers
│   │   ├── task_consumer/      # Task queue consumer
│   │   │   ├── worker.go       # Worker with restart loop
│   │   │   └── proc_task.go    # Task processing logic
│   │   └── notification_consumer/  # Notification queue consumer
│   │       ├── worker.go
│   │       └── proc_notification.go
│   ├── db/                     # Database clients
│   │   ├── postgres.go
│   │   └── redis.go
│   ├── domain/                 # Domain-driven design structure
│   │   ├── task/               # Task domain
│   │   │   ├── model.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   └── controller.go
│   │   └── user/               # User domain (auth)
│   │       ├── model.go
│   │       ├── repository.go
│   │       ├── service.go
│   │       └── controller.go
│   ├── events/                 # Event definitions
│   │   ├── base.go
│   │   ├── builder.go
│   │   ├── constants.go
│   │   ├── task_event.go
│   │   └── user_event.go
│   ├── handler/                # HTTP route handlers
│   │   └── handler.go
│   ├── logger/                 # Logging infrastructure
│   │   ├── logger.go           # Logrus with service hook
│   │   ├── filebeat/
│   │   │   └── filebeat.yml    # Filebeat autodiscovery config
│   │   └── logstash/
│   │       ├── config/
│   │       │   └── logstash.yml
│   │       └── pipeline/
│   │           └── logstash.conf  # Log processing pipeline
│   ├── middleware/             # HTTP middlewares
│   │   ├── jwt.go
│   │   ├── logger.go
│   │   ├── rate_limiter.go
│   │   └── rate_limiter.lua    # Atomic Redis operations
│   ├── notification/           # Notification interfaces
│   │   ├── email_interface.go
│   │   ├── email_sender.go
│   │   └── mock_email_sender.go
│   ├── queue/                  # RabbitMQ client
│   │   └── rabbitmq.go         # RabbitMQManager with auto-reconnect
│   ├── router/                 # Route definitions
│   │   └── router.go
│   └── utils/                  # Utility functions
│       └── tx.go               # Transaction helpers
│   └── utils/                  # Utility functions
│       └── tx.go               # Transaction helpers
├── tests/
│   └── integration/            # Integration test suite
│       ├── setup_test.go       # Test environment setup
│       ├── auth_test.go        # Auth flow tests
│       ├── task_test.go        # Task CRUD tests
│       └── cache_test.go       # Cache behavior tests
├── migrations/                 # Database migrations
│   ├── 001_create_tasks.up.sql
│   ├── 001_create_tasks.down.sql
│   ├── 002_add_task_column.up.sql
│   ├── 002_add_task_column.down.sql
│   ├── 003_change_id_type.up.sql
│   ├── 004_create_users.up.sql
│   └── 005_rename_id_to_task_id.up.sql
├── docker/                     # Dockerfiles
│   ├── api.Dockerfile
│   ├── task-worker.Dockerfile
│   ├── notification-worker.Dockerfile
│   ├── migrate.Dockerfile
│   ├── filebeat.Dockerfile
│   └── logstash.Dockerfile
├── .github/
│   └── workflows/
│       └── ci.yml              # CI/CD pipeline
├── logs/                       # Application logs
├── docker-compose.yaml         # Full stack orchestration
├── go.mod
├── go.sum
├── Makefile                    # Development commands
└── README.md
```

### Key Files

- **`cmd/api/main.go`** - API server bootstrap with RabbitMQManager
- **`cmd/task/main.go`** - Task worker service with 3 concurrent workers
- **`cmd/notification/main.go`** - Notification worker service with 3 concurrent workers
- **`internal/queue/rabbitmq.go`** - RabbitMQManager with auto-reconnect, monitoring, exponential backoff
- **`internal/consumer/task_consumer/worker.go`** - Worker restart loop with connection refresh
- **`internal/consumer/notification_consumer/worker.go`** - Notification worker with restart loop
- **`internal/middleware/rate_limiter.go`** - Token Bucket implementation
- **`internal/middleware/rate_limiter.lua`** - Atomic Redis operations
- **`internal/domain/task/controller.go`** - Task HTTP handlers with ownership checks
- **`internal/domain/user/controller.go`** - Auth endpoints (register, login, refresh)
- **`internal/logger/logger.go`** - Logrus with automatic service field injection
- **`internal/logger/filebeat/filebeat.yml`** - Filebeat autodiscovery configuration
- **`internal/logger/logstash/pipeline/logstash.conf`** - Log processing with fingerprinting
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
docker-compose up -d --scale task-worker=5 --scale notification-worker=5
```

4. Monitor services:
   - Use Kibana for centralized log monitoring (http://localhost:5601)
   - RabbitMQ Management UI for queue monitoring (http://localhost:15672)
   - Add Prometheus/Grafana for metrics (optional)

5. Database backups:
```bash
docker exec postgres pg_dump -U postgres task_db > backup.sql
```

### Performance Tuning

- **Redis**: Enable persistence (RDB/AOF) for rate limit state
- **PostgreSQL**: Add indexes on `user_id`, `status`, and `task_type` columns
- **RabbitMQ**: Adjust prefetch count (QoS) for workers, currently set to 1
- **Elasticsearch**: Increase heap size for high-volume logging (`ES_JAVA_OPTS=-Xms1g -Xmx1g`)
- **Logstash**: Tune batch size and workers for log processing throughput
- **Workers**: Scale horizontally based on queue depth

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
- [Elastic Stack](https://www.elastic.co/) - ELK Stack for observability
- [Logrus](https://github.com/sirupsen/logrus) - Structured logging
