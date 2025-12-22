# Controller Layer Architecture

## Overview
Implementasi MVC pattern dengan menambahkan Controller layer untuk memisahkan logic dari handler/routing.

## Architecture Structure

```
┌─────────────┐
│   Handler   │  ← Routing setup, minimal logic
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Controller  │  ← Business logic, request/response handling
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Service    │  ← Business rules, transactions
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Repository  │  ← Database operations
└─────────────┘
```

## Changes Made

### 1. Auth Controller (`internal/auth/controller.go`)
**Menggabungkan** semua auth-related operations dalam satu controller:
- `Register()` - User registration
- `Login()` - User authentication & JWT generation
- `RefreshToken()` - Refresh access token

```go
type AuthController struct {
    userService user.UserServiceInterface
    jwtSecret   string
}

// Methods:
// - Register(c *gin.Context)
// - Login(c *gin.Context)
// - RefreshToken(c *gin.Context)
```

### 2. Task Controller (`internal/task/controller.go`)
Task operations dengan JWT authentication:
- `CreateTask()` - Create new task (userID from JWT)
- `GetTask()` - Get task by ID
- `GetTasksByUser()` - Get all tasks for a user

```go
type TaskController struct {
    service TaskServiceInterface
}

// Methods:
// - CreateTask(c *gin.Context)
// - GetTask(c *gin.Context)
// - GetTasksByUser(c *gin.Context)
```

### 3. Updated Handlers

#### Auth Handler (`internal/auth/handler.go`)
Simplified - hanya setup routing:
```go
func SetupRoutes(r *gin.Engine, userService user.UserServiceInterface, jwtSecret string) {
    controller := NewAuthController(userService, jwtSecret)
    
    authGroup := r.Group("/auth")
    {
        authGroup.POST("/register", controller.Register)
        authGroup.POST("/login", controller.Login)
        authGroup.POST("/refresh", controller.RefreshToken)
    }
}
```

#### Task Handler (`internal/task/handler.go`)
Simplified - hanya setup routing:
```go
func SetupRoutes(r *gin.Engine, db *sql.DB, conn *amqp.Connection, rdb *redis.Client, cfg *config.Config) {
    repo := NewTaskRepository()
    service := NewTaskService(repo, db, conn, rdb)
    controller := NewTaskController(service)
    
    api := r.Group("/api/v1")
    api.Use(auth.AuthMiddleware(cfg.JWT.Secret))
    {
        api.POST("/tasks", controller.CreateTask)
        api.GET("/tasks/:id", controller.GetTask)
        api.GET("/users/:user_id/tasks", controller.GetTasksByUser)
    }
}
```

### 4. Removed Files
- ❌ `internal/user/handler.go` - Functionality moved to `auth/controller.go`

## Benefits

### 1. **Separation of Concerns**
- **Handler**: Hanya routing setup
- **Controller**: Request/response handling, validation
- **Service**: Business logic
- **Repository**: Database operations

### 2. **Better Organization**
- Auth operations (register, login, refresh) dalam satu controller
- Task operations dalam satu controller
- Mudah maintain dan extend

### 3. **Testability**
- Controller methods mudah di-unit test
- Mock service layer untuk testing
- Tidak perlu mock Gin context di service layer

### 4. **Reusability**
- Controller methods bisa dipanggil dari berbagai handler/route
- Logic terpusat, tidak tersebar

### 5. **Scalability**
- Mudah menambahkan endpoint baru
- Mudah menambahkan controller baru
- Clear structure untuk team development

## File Structure

```
internal/
├── auth/
│   ├── controller.go    # NEW: Register, Login, RefreshToken
│   ├── handler.go       # UPDATED: Only routing setup
│   └── jwt.go           # JWT utilities
├── task/
│   ├── controller.go    # NEW: CreateTask, GetTask, GetTasksByUser
│   ├── handler.go       # UPDATED: Only routing setup
│   ├── service.go       # Business logic
│   ├── repository.go    # Database operations
│   └── model.go         # Task struct
└── user/
    ├── service.go       # User business logic
    ├── repository.go    # User database operations
    └── model.go         # User struct
```

## API Endpoints (Unchanged)

### Auth Endpoints (Public)
```
POST /auth/register   → AuthController.Register
POST /auth/login      → AuthController.Login
POST /auth/refresh    → AuthController.RefreshToken
```

### Task Endpoints (Protected with JWT)
```
POST /api/v1/tasks                  → TaskController.CreateTask
GET  /api/v1/tasks/:id              → TaskController.GetTask
GET  /api/v1/users/:user_id/tasks   → TaskController.GetTasksByUser
```

## Example Usage

### Creating a New Endpoint

**Before (Old Pattern):**
```go
// In handler.go - mixed logic
func CreateTaskHandler(service TaskServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        // validation
        // business logic
        // response handling
    }
}
```

**After (Controller Pattern):**
```go
// In controller.go - clear separation
func (tc *TaskController) CreateTask(c *gin.Context) {
    // validation
    // business logic
    // response handling
}

// In handler.go - just routing
api.POST("/tasks", controller.CreateTask)
```

## Testing Example

### Controller Unit Test
```go
func TestTaskController_CreateTask(t *testing.T) {
    // Mock service
    mockService := &MockTaskService{}
    controller := NewTaskController(mockService)
    
    // Test request
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Set("userID", 1)
    
    // Execute
    controller.CreateTask(c)
    
    // Assert
    assert.Equal(t, 201, w.Code)
}
```

## Migration Guide

### Old Code
```go
// main.go
authGroup.POST("/register", user.RegisterHandler(userService))
authGroup.POST("/login", auth.LoginHandler(cfg.JWT.Secret, userService))
```

### New Code
```go
// main.go
auth.SetupRoutes(r, userService, config.JWT.Secret)
```

All auth endpoints now managed by AuthController internally.

## Best Practices

1. **Controller Naming**: `{Domain}Controller` (e.g., `TaskController`, `AuthController`)
2. **Method Naming**: Use action verbs (e.g., `CreateTask`, `Login`, `RefreshToken`)
3. **Keep Controllers Thin**: Delegate business logic to services
4. **Error Handling**: Return appropriate HTTP status codes
5. **Validation**: Validate input in controller before calling service
6. **Response Format**: Use consistent JSON response structure

## Future Improvements

- [ ] Add middleware at controller level for specific operations
- [ ] Implement request/response DTOs (Data Transfer Objects)
- [ ] Add controller-level caching
- [ ] Implement request validation middleware
- [ ] Add OpenAPI/Swagger documentation generation
- [ ] Implement rate limiting per controller method
- [ ] Add audit logging at controller level
