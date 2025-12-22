# Quick Reference: Controller Pattern

## Adding New Endpoint

### 1. Add Method to Controller

**Example: Add UpdateTask endpoint**

```go
// internal/task/controller.go
func (tc *TaskController) UpdateTask(c *gin.Context) {
    // Get task ID from URL
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
        return
    }
    
    // Parse request body
    var req struct {
        Status string `json:"status" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Get userID from JWT
    userID, err := auth.GetUserIDFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return
    }
    
    // Call service
    if err := tc.service.UpdateTask(id, userID, req.Status); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Task updated successfully"})
}
```

### 2. Register Route in Handler

```go
// internal/task/handler.go
func SetupRoutes(r *gin.Engine, db *sql.DB, conn *amqp.Connection, rdb *redis.Client, cfg *config.Config) {
    // ... existing code ...
    controller := NewTaskController(service)
    
    api := r.Group("/api/v1")
    api.Use(auth.AuthMiddleware(cfg.JWT.Secret))
    {
        api.POST("/tasks", controller.CreateTask)
        api.GET("/tasks/:id", controller.GetTask)
        api.PUT("/tasks/:id", controller.UpdateTask)  // NEW
        api.GET("/users/:user_id/tasks", controller.GetTasksByUser)
    }
}
```

### 3. Add Service Method

```go
// internal/task/service.go
func (s *TaskService) UpdateTask(id, userID int, status string) error {
    // Business logic here
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    if err := s.repo.Update(tx, id, userID, status); err != nil {
        return err
    }
    
    return tx.Commit()
}
```

## Controller Methods Template

```go
// CREATE operation
func (c *Controller) Create(ctx *gin.Context) {
    var req CreateRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    result, err := c.service.Create(req)
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    ctx.JSON(201, result)
}

// READ operation
func (c *Controller) GetByID(ctx *gin.Context) {
    id, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        ctx.JSON(400, gin.H{"error": "Invalid ID"})
        return
    }
    
    result, err := c.service.GetByID(id)
    if err != nil {
        ctx.JSON(404, gin.H{"error": "Not found"})
        return
    }
    
    ctx.JSON(200, result)
}

// UPDATE operation
func (c *Controller) Update(ctx *gin.Context) {
    id, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        ctx.JSON(400, gin.H{"error": "Invalid ID"})
        return
    }
    
    var req UpdateRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    if err := c.service.Update(id, req); err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    ctx.JSON(200, gin.H{"message": "Updated successfully"})
}

// DELETE operation
func (c *Controller) Delete(ctx *gin.Context) {
    id, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        ctx.JSON(400, gin.H{"error": "Invalid ID"})
        return
    }
    
    if err := c.service.Delete(id); err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    ctx.JSON(200, gin.H{"message": "Deleted successfully"})
}
```

## Common Patterns

### Extract UserID from JWT
```go
userID, err := auth.GetUserIDFromContext(c)
if err != nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
    return
}
```

### Parse URL Parameter
```go
id, err := strconv.Atoi(c.Param("id"))
if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
    return
}
```

### Parse Request Body
```go
var req struct {
    Field1 string `json:"field1" binding:"required"`
    Field2 int    `json:"field2" binding:"required,min=1"`
}
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
}
```

### Parse Query Parameters
```go
page := c.DefaultQuery("page", "1")
limit := c.DefaultQuery("limit", "10")
```

## HTTP Status Codes

| Code | Usage | Example |
|------|-------|---------|
| 200  | Success | GET request successful |
| 201  | Created | Resource created successfully |
| 400  | Bad Request | Invalid input data |
| 401  | Unauthorized | Missing or invalid JWT |
| 403  | Forbidden | Valid JWT but no permission |
| 404  | Not Found | Resource doesn't exist |
| 409  | Conflict | Duplicate username, etc |
| 500  | Server Error | Database error, etc |

## Error Response Format

```go
// Simple error
c.JSON(400, gin.H{"error": "Invalid input"})

// Detailed error
c.JSON(400, gin.H{
    "error": "Validation failed",
    "details": []string{
        "username must be at least 3 characters",
        "password is required",
    },
})

// With error code
c.JSON(409, gin.H{
    "error": "Username already exists",
    "code": "DUPLICATE_USERNAME",
})
```

## Success Response Format

```go
// Simple success
c.JSON(200, gin.H{"message": "Success"})

// With data
c.JSON(200, gin.H{
    "data": result,
    "message": "Success",
})

// List with metadata
c.JSON(200, gin.H{
    "data": items,
    "total": len(items),
    "page": 1,
    "limit": 10,
})
```

## Testing Controller

```go
func TestTaskController_CreateTask(t *testing.T) {
    // Setup
    mockService := &MockTaskService{}
    controller := NewTaskController(mockService)
    
    // Create test context
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    
    // Set JWT userID
    c.Set(auth.UserIDKey, 1)
    
    // Create request body
    body := `{"task_type":"email"}`
    c.Request = httptest.NewRequest("POST", "/tasks", strings.NewReader(body))
    c.Request.Header.Set("Content-Type", "application/json")
    
    // Execute
    controller.CreateTask(c)
    
    // Assert
    assert.Equal(t, 201, w.Code)
    assert.Contains(t, w.Body.String(), "task_id")
}
```

## Current Endpoints Summary

### Auth Endpoints (Public)
```bash
POST /auth/register   # Register new user
POST /auth/login      # Login and get JWT tokens
POST /auth/refresh    # Refresh access token
```

### Task Endpoints (JWT Required)
```bash
POST /api/v1/tasks                 # Create task
GET  /api/v1/tasks/:id             # Get task by ID
GET  /api/v1/users/:user_id/tasks  # Get user's tasks
```

## Testing Checklist

- [ ] Controller method parses request correctly
- [ ] Validation works for required fields
- [ ] JWT userID extraction works
- [ ] Service method is called with correct parameters
- [ ] Success response has correct status code
- [ ] Error responses have correct status codes
- [ ] Error messages are clear and helpful
