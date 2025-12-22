# Separation of Concerns: Controller vs Service

## Question
**"Apakah pengenalan credential di controller bukan di service layer?"**

## Answer: ✅ Struktur Saat Ini SUDAH BENAR

Credential validation **SUDAH** di service layer, controller hanya **memanggil** service method.

## Current Implementation (CORRECT)

### Controller Layer (user/controller.go)
```go
func (ac *UserController) Login(c *gin.Context) {
    var req struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }
    
    // ✅ Controller hanya MEMANGGIL service
    // Controller TIDAK tahu detail tentang bcrypt, hashing, dll
    authenticatedUser, err := ac.userService.ValidateCredentials(req.Username, req.Password)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        return
    }
    
    // Generate JWT tokens (this could also be moved to service)
    tokens, err := auth.GenerateTokenPair(authenticatedUser.ID, ac.jwtSecret)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
        return
    }
    
    c.JSON(http.StatusOK, tokens)
}
```

**Controller Responsibility:**
- ✅ Parse HTTP request
- ✅ Validate request format (JSON binding)
- ✅ Call service method
- ✅ Format HTTP response
- ❌ NO business logic
- ❌ NO credential validation
- ❌ NO bcrypt operations

### Service Layer (user/service.go)
```go
func (s *UserService) ValidateCredentials(username, password string) (*User, error) {
    // ✅ Service melakukan ACTUAL credential validation
    
    // 1. Get user from database
    user, err := s.repo.GetByUsername(s.db, username)
    if err != nil {
        return nil, errors.New("invalid credentials")
    }
    
    // 2. Compare password with bcrypt hash
    // ✅ Business logic di service
    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "username": username,
        }).Warn("Invalid password attempt")
        return nil, errors.New("invalid credentials")
    }
    
    // 3. Return authenticated user
    return user, nil
}
```

**Service Responsibility:**
- ✅ Business logic (credential validation)
- ✅ Bcrypt password comparison
- ✅ Get user from repository
- ✅ Security logging
- ✅ Return domain objects
- ❌ NO HTTP handling
- ❌ NO JSON parsing

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    CONTROLLER LAYER                         │
│  • Parse HTTP request (JSON)                                │
│  • Validate request format                                  │
│  • Call: service.ValidateCredentials(username, password)    │
│  • Format HTTP response                                     │
│  • NO business logic                                        │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           │ service.ValidateCredentials()
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                     SERVICE LAYER                           │
│  ✅ CREDENTIAL VALIDATION HAPPENS HERE:                     │
│     • Get user from repository                              │
│     • bcrypt.CompareHashAndPassword()                       │
│     • Business rules & validation                           │
│     • Security logging                                      │
│     • Return authenticated user                             │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           │ repo.GetByUsername()
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   REPOSITORY LAYER                          │
│  • SQL query: SELECT * FROM users WHERE username = ?        │
│  • Return user with hashed password                         │
│  • NO business logic                                        │
└─────────────────────────────────────────────────────────────┘
```

## Why This is CORRECT

### 1. Controller Only Orchestrates
```go
// ❌ WRONG - Controller doing business logic
func (c *Controller) Login(ctx *gin.Context) {
    user := getUserFromDB(username)
    if bcrypt.CompareHashAndPassword(user.Password, password) != nil {
        return error
    }
    // BAD: Business logic in controller
}

// ✅ CORRECT - Controller delegates to service
func (c *Controller) Login(ctx *gin.Context) {
    user, err := c.service.ValidateCredentials(username, password)
    if err != nil {
        return error
    }
    // GOOD: Controller only orchestrates
}
```

### 2. Service Handles Business Logic
```go
// ✅ Service knows about:
// - Password hashing (bcrypt)
// - Business rules (account lockout, password policy)
// - Security (logging failed attempts)
// - Domain logic

func (s *Service) ValidateCredentials(username, password string) (*User, error) {
    user, err := s.repo.GetByUsername(s.db, username)
    if err != nil {
        return nil, errors.New("invalid credentials")
    }
    
    // Business logic in service
    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
    if err != nil {
        s.logFailedAttempt(username)
        s.checkAccountLockout(user)
        return nil, errors.New("invalid credentials")
    }
    
    return user, nil
}
```

### 3. Repository Only Database
```go
// ✅ Repository knows about:
// - SQL queries
// - Database connection
// - Data mapping

func (r *Repository) GetByUsername(db *sql.DB, username string) (*User, error) {
    query := `SELECT id, username, password, created_at FROM users WHERE username = $1`
    
    user := &User{}
    err := db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt)
    
    return user, err
}
```

## Common Misconception

### ❓ "Controller calls service method, so isn't controller doing validation?"

**NO!** Controller is only **orchestrating**, not **executing** the validation.

```go
// Controller
authenticatedUser, err := service.ValidateCredentials(username, password)
                                   ↑
                                   │
        "Call this method" ────────┘
        Controller doesn't know HOW it's validated
```

**Think of it like:**
- **Controller**: "Hey Service, can you validate these credentials?"
- **Service**: "Sure! Let me check the database and compare bcrypt hashes... Done! Here's the user."
- **Controller**: "Thanks! Here's the JWT token for the user."

## What if We Move Validation to Controller? (WRONG)

### ❌ Anti-Pattern: Business Logic in Controller
```go
func (c *UserController) Login(ctx *gin.Context) {
    var req LoginRequest
    ctx.ShouldBindJSON(&req)
    
    // ❌ BAD: Getting user directly in controller
    user, err := c.repo.GetByUsername(req.Username)
    
    // ❌ BAD: bcrypt in controller
    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
    if err != nil {
        return "Invalid credentials"
    }
    
    // Now what if we need to:
    // - Add account lockout?
    // - Add password expiry check?
    // - Add 2FA?
    // Controller becomes bloated with business logic!
}
```

**Problems:**
1. ❌ Controller becomes **bloated**
2. ❌ **Hard to test** (need to mock database in controller tests)
3. ❌ **Hard to reuse** logic (what if we add API v2?)
4. ❌ **Violates SRP** (Single Responsibility Principle)
5. ❌ **Hard to maintain** (business logic scattered)

## Best Practice Summary

| Layer | Responsibility | Example |
|-------|---------------|---------|
| **Controller** | HTTP handling | Parse JSON, call service, return JSON |
| **Service** | Business logic | Validate credentials, bcrypt, rules |
| **Repository** | Data access | SQL queries, database operations |

### Rules of Thumb:

1. **Controller should NOT import:**
   - ❌ `golang.org/x/crypto/bcrypt`
   - ❌ `database/sql` (except passing to service)
   - ❌ Business logic packages

2. **Service should NOT import:**
   - ❌ `github.com/gin-gonic/gin`
   - ❌ HTTP-related packages
   - ✅ CAN import: bcrypt, business logic packages

3. **Repository should NOT import:**
   - ❌ HTTP packages
   - ❌ Business logic
   - ✅ ONLY database-related packages

## Testing Benefits

### Controller Test (Simple)
```go
func TestUserController_Login(t *testing.T) {
    mockService := &MockUserService{
        ValidateCredentialsFunc: func(username, password string) (*User, error) {
            return &User{ID: 1}, nil
        },
    }
    
    controller := NewUserController(mockService, "secret")
    
    // Test HTTP handling only
    // No need to mock bcrypt, database, etc.
}
```

### Service Test (Business Logic)
```go
func TestUserService_ValidateCredentials(t *testing.T) {
    mockRepo := &MockRepository{
        GetByUsernameFunc: func(username string) (*User, error) {
            hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
            return &User{ID: 1, Password: string(hash)}, nil
        },
    }
    
    service := NewUserService(mockRepo, db)
    
    // Test business logic: bcrypt comparison, error handling, etc.
    user, err := service.ValidateCredentials("john", "password123")
    assert.NoError(t, err)
    assert.Equal(t, 1, user.ID)
    
    // Test invalid password
    user, err = service.ValidateCredentials("john", "wrongpassword")
    assert.Error(t, err)
}
```

## Conclusion

✅ **Your current implementation is CORRECT!**

- Controller: HTTP handling only
- Service: Credential validation with bcrypt
- Repository: Database queries

This is proper **separation of concerns** and follows **clean architecture** principles.

## Further Improvements (Optional)

You could make it even more explicit by renaming:
```go
// Before
service.ValidateCredentials(username, password)

// After (more explicit)
service.Login(username, password)
service.Authenticate(username, password)
```

But the current structure is already correct! 🎉
