# JWT Authentication Implementation

## Summary
JWT authentication with refresh token support has been successfully implemented for the task handler API.

## Changes Made

### 1. JWT Utilities (`internal/auth/jwt.go`)
- **GenerateTokenPair**: Creates access token (15 min) and refresh token (7 days)
- **ValidateToken**: Validates and parses JWT tokens
- **RefreshAccessToken**: Generates new access token from refresh token
- **AuthMiddleware**: Gin middleware that validates JWT from Authorization header
- **GetUserIDFromContext**: Helper to extract userID from Gin context

Token structure:
```go
type Claims struct {
    UserID int       `json:"user_id"`
    Type   TokenType `json:"type"` // "access" or "refresh"
    jwt.RegisteredClaims
}
```

### 2. Auth Handlers (`internal/auth/handler.go`)
- **POST /auth/login**: Login endpoint (returns access + refresh tokens)
- **POST /auth/refresh**: Refresh token endpoint (returns new access token)

**Note**: Login handler currently uses placeholder authentication (userID=1). You need to:
1. Create a user repository to fetch users from database
2. Implement proper password hashing verification (bcrypt)
3. Add proper user validation

### 3. Task Handler Updates (`internal/task/handler.go`)
- Removed `user_id` from `CreateTaskRequest` (security improvement)
- Added JWT middleware to all `/api/v1` routes
- Updated CreateTask handler to extract userID from JWT context

### 4. Main API Setup (`cmd/api/main.go`)
- Added public auth routes: `/auth/login` and `/auth/refresh`
- Protected task routes with JWT middleware

## API Endpoints

### Public Endpoints (No Authentication Required)
```bash
POST /auth/login
POST /auth/refresh
```

### Protected Endpoints (Requires JWT)
```bash
POST /api/v1/tasks
GET  /api/v1/tasks/:id
GET  /api/v1/users/:user_id/tasks
```

## Testing Guide

### 1. Set JWT Secret Environment Variable
```bash
export JWT_SECRET="your-secret-key-here"
```

### 2. Login to Get Tokens
```bash
curl -X POST http://localhost:8087/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test",
    "password": "test123"
  }'
```

Response:
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 900
}
```

### 3. Create Task (with JWT)
```bash
curl -X POST http://localhost:8087/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{
    "task_type": "email"
  }'
```

Note: `user_id` is now automatically extracted from JWT token.

### 4. Get Task Status (with JWT)
```bash
curl -X GET http://localhost:8087/api/v1/tasks/1 \
  -H "Authorization: Bearer <access_token>"
```

### 5. Refresh Token
```bash
curl -X POST http://localhost:8087/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<refresh_token>"
  }'
```

Response:
```json
{
  "access_token": "eyJhbGc...",
  "expires_in": 900
}
```

## Error Responses

### Missing Authorization Header
```json
{
  "error": "Authorization header required"
}
```

### Invalid Token Format
```json
{
  "error": "Invalid authorization format. Use: Bearer <token>"
}
```

### Expired Token
```json
{
  "error": "Token expired"
}
```

### Invalid Token
```json
{
  "error": "Invalid token"
}
```

## Security Considerations

1. **JWT_SECRET**: Store in environment variable, never commit to git
2. **HTTPS**: Use HTTPS in production to protect tokens in transit
3. **Token Storage**: Client should store tokens securely (HttpOnly cookies or secure storage)
4. **Password Hashing**: Implement bcrypt for password verification (currently placeholder)
5. **Rate Limiting**: Already configured in nginx (5 req/s + burst 10)

## Token Lifetimes

- **Access Token**: 15 minutes
- **Refresh Token**: 7 days

## Next Steps (User Implementation Required)

1. **Create User Management**:
   - Create `internal/user` package
   - Implement user repository (CRUD operations)
   - Add user registration endpoint
   - Add password hashing (bcrypt)

2. **Update Login Handler**:
   ```go
   // Replace placeholder in internal/auth/handler.go
   user, err := userRepo.GetByUsername(req.Username)
   if err != nil {
       c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
       return
   }
   if !bcrypt.CheckPasswordHash(req.Password, user.PasswordHash) {
       c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
       return
   }
   userID := user.ID
   ```

3. **Database Migration for Users Table**:
   ```sql
   CREATE TABLE users (
       id SERIAL PRIMARY KEY,
       username VARCHAR(255) UNIQUE NOT NULL,
       password_hash VARCHAR(255) NOT NULL,
       email VARCHAR(255) UNIQUE NOT NULL,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
       updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );
   ```

## Configuration

Ensure `config.JWT.Secret` is loaded from environment variable in `internal/config/config.go`:
```go
type Config struct {
    // ... other fields
    JWT struct {
        Secret string
    }
}

// In Load() function:
config.JWT.Secret = os.Getenv("JWT_SECRET")
```
