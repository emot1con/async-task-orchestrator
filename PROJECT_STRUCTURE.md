# Project Structure Overview

## Complete Architecture Diagram

```
cmd/api/main.go
    │
    ├── Setup Dependencies (DB, Redis, RabbitMQ)
    │
    ├── Setup User Service
    │   └── user.NewUserService(repo, db)
    │
    ├── Setup Auth Routes
    │   └── auth.SetupRoutes(r, userService, jwtSecret)
    │       └── AuthController
    │           ├── Register    → POST /auth/register
    │           ├── Login       → POST /auth/login
    │           └── RefreshToken → POST /auth/refresh
    │
    └── Setup Task Routes
        └── task.SetupRoutes(r, db, conn, rdb, config)
            └── TaskController (JWT Protected)
                ├── CreateTask      → POST /api/v1/tasks
                ├── GetTask         → GET  /api/v1/tasks/:id
                └── GetTasksByUser  → GET  /api/v1/users/:user_id/tasks
```

## Layer Responsibilities

### 1. Handler Layer (`handler.go`)
**Purpose**: Route registration and setup
```
┌─────────────────────────────────────┐
│         Handler Layer               │
│  • Register routes                  │
│  • Apply middleware                 │
│  • Initialize controllers           │
│  • No business logic                │
└─────────────────────────────────────┘
```

### 2. Controller Layer (`controller.go`)
**Purpose**: Request/Response handling
```
┌─────────────────────────────────────┐
│        Controller Layer             │
│  • Parse request body               │
│  • Validate input                   │
│  • Extract context data (JWT, etc)  │
│  • Call service methods             │
│  • Format response                  │
│  • Handle HTTP errors               │
└─────────────────────────────────────┘
```

### 3. Service Layer (`service.go`)
**Purpose**: Business logic
```
┌─────────────────────────────────────┐
│         Service Layer               │
│  • Business rules                   │
│  • Transaction management           │
│  • Data validation                  │
│  • Orchestrate repositories         │
│  • External service calls           │
└─────────────────────────────────────┘
```

### 4. Repository Layer (`repository.go`)
**Purpose**: Data access
```
┌─────────────────────────────────────┐
│       Repository Layer              │
│  • Database queries                 │
│  • CRUD operations                  │
│  • No business logic                │
│  • Return domain models             │
└─────────────────────────────────────┘
```

## Request Flow Example: Create Task

```
1. HTTP Request
   POST /api/v1/tasks
   Authorization: Bearer <jwt_token>
   Body: {"task_type": "email"}
   
2. Handler Layer (handler.go)
   ├── Route matched: POST /tasks
   ├── Apply JWT Middleware
   │   └── Extract userID from token
   │   └── Set in context: c.Set("userID", 1)
   └── Call controller: controller.CreateTask(c)
   
3. Controller Layer (controller.go)
   ├── Parse request body
   ├── Validate input: task_type required
   ├── Get userID from context
   ├── Create Task object
   └── Call service: service.CreateTask(task)
   
4. Service Layer (service.go)
   ├── Validate business rules
   ├── Begin transaction
   ├── Call repository: repo.Create(tx, task)
   ├── Publish to RabbitMQ
   ├── Commit transaction
   └── Return task ID
   
5. Repository Layer (repository.go)
   ├── Execute SQL INSERT
   ├── Return generated ID
   └── Handle database errors
   
6. Response Flow (reverse)
   Repository → Service → Controller → Handler → Client
   
7. HTTP Response
   201 Created
   {
     "task_id": 1,
     "status": "PENDING",
     "message": "Task created successfully"
   }
```

## Authentication Flow: Login

```
1. HTTP Request
   POST /auth/login
   Body: {"username": "john", "password": "secret"}
   
2. Handler Layer
   ├── Route: POST /auth/login
   └── Call: controller.Login(c)
   
3. Controller Layer (AuthController)
   ├── Parse credentials
   ├── Validate format
   └── Call: userService.ValidateCredentials(username, password)
   
4. Service Layer (UserService)
   ├── Get user by username
   ├── Compare bcrypt hash
   └── Return user if valid
   
5. Repository Layer (UserRepository)
   ├── SELECT * FROM users WHERE username = ?
   └── Return user with hashed password
   
6. Back to Controller
   ├── Generate JWT tokens
   │   ├── Access token (15 min)
   │   └── Refresh token (7 days)
   └── Return token pair
   
7. HTTP Response
   200 OK
   {
     "access_token": "eyJhbGc...",
     "refresh_token": "eyJhbGc...",
     "expires_in": 900
   }
```

## Directory Structure

```
task_handler/
├── cmd/
│   ├── api/
│   │   └── main.go              # Entry point, setup dependencies
│   └── worker/
│       └── main.go              # Worker process
│
├── internal/
│   ├── auth/
│   │   ├── controller.go        # Register, Login, RefreshToken
│   │   ├── handler.go           # Auth routes setup
│   │   └── jwt.go               # JWT utilities
│   │
│   ├── user/
│   │   ├── model.go             # User struct
│   │   ├── repository.go        # User DB operations
│   │   └── service.go           # User business logic + bcrypt
│   │
│   ├── task/
│   │   ├── controller.go        # CreateTask, GetTask, GetTasksByUser
│   │   ├── handler.go           # Task routes setup
│   │   ├── model.go             # Task struct
│   │   ├── repository.go        # Task DB operations
│   │   └── service.go           # Task business logic + RabbitMQ
│   │
│   ├── worker/
│   │   └── worker.go            # Task processing
│   │
│   ├── config/
│   │   └── config.go            # Configuration
│   │
│   ├── db/
│   │   └── postgres.go          # Database connection
│   │
│   ├── cache/
│   │   └── redis.go             # Redis connection
│   │
│   └── queue/
│       └── rabbitmq.go          # RabbitMQ connection
│
├── migrations/
│   ├── 001_create_tasks.up.sql
│   ├── 002_add_task_column.up.sql
│   └── 003_create_users.up.sql
│
├── docker/
│   ├── api.Dockerfile
│   └── worker.Dockerfile
│
├── docker-compose.yaml
├── go.mod
└── Makefile
```

## Middleware Flow

```
Request
  │
  ├─► gin.Default() middleware
  │    ├─► Logger
  │    └─► Recovery
  │
  ├─► JWT Middleware (for /api/v1/*)
  │    ├─► Validate token
  │    ├─► Extract userID
  │    └─► Set in context
  │
  └─► Controller method
       ├─► GetUserIDFromContext()
       ├─► Process request
       └─► Return response
```

## Dependency Injection Pattern

```go
// main.go - Dependency injection at startup
func main() {
    // Infrastructure
    db := db.Init(config)
    redis := cache.SetupRedis(config)
    rabbitmq := queue.SetupRabbitMQ(config)
    
    // Repository layer
    userRepo := user.NewUserRepository()
    taskRepo := task.NewTaskRepository()
    
    // Service layer (inject repo + infra)
    userService := user.NewUserService(userRepo, db)
    taskService := task.NewTaskService(taskRepo, db, rabbitmq, redis)
    
    // Controller layer (inject service)
    authController := auth.NewAuthController(userService, jwtSecret)
    taskController := task.NewTaskController(taskService)
    
    // Handler layer (inject controller)
    auth.SetupRoutes(router, userService, jwtSecret)
    task.SetupRoutes(router, db, rabbitmq, redis, config)
}
```

## Key Design Decisions

1. **Controller per Domain**: AuthController handles all auth operations, TaskController handles all task operations
2. **Handler for Routing**: Handlers only setup routes, no business logic
3. **Service for Business Logic**: All business rules in service layer
4. **Repository for Data**: All database operations isolated in repository
5. **JWT Middleware**: Applied at route group level, not per-endpoint
6. **Dependency Injection**: Dependencies passed through constructors, not globals

## Benefits Summary

| Layer      | Before (Handler Pattern) | After (Controller Pattern) |
|------------|-------------------------|---------------------------|
| Routing    | Mixed with logic        | ✅ Separated (handler.go) |
| Validation | In handler functions    | ✅ In controller methods  |
| Testing    | Complex (mock Gin)      | ✅ Easier (mock service)  |
| Reusability| Low (tied to routes)    | ✅ High (method-based)    |
| Maintenance| Scattered functions     | ✅ Organized in structs   |
