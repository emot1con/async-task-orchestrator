# User Management Implementation

## Summary
User repository dan authentication system sudah terintegrasi dengan JWT authentication.

## Files Created

### 1. Model (`internal/user/model.go`)
```go
type User struct {
    ID        int
    Username  string
    Password  string    // bcrypt hashed, never exposed in JSON
    CreatedAt time.Time
}
```

### 2. Repository (`internal/user/repository.go`)
Interface methods:
- `Create(tx *sql.Tx, user *User) (int, error)` - Create new user
- `GetByID(db *sql.DB, id int) (*User, error)` - Get user by ID
- `GetByUsername(db *sql.DB, username string) (*User, error)` - Get user by username
- `UpdatePassword(tx *sql.Tx, id int, hashedPassword string) error` - Update password

### 3. Service (`internal/user/service.go`)
Business logic dengan bcrypt password hashing:
- `CreateUser(username, password string) (int, error)` - Hash password & create user
- `ValidateCredentials(username, password string) (*User, error)` - Validate login
- `GetUserByID(id int) (*User, error)` - Get user by ID
- `GetUserByUsername(username string) (*User, error)` - Get user by username

### 4. Handler (`internal/user/handler.go`)
- `POST /auth/register` - User registration endpoint

### 5. Migration Files
- `migrations/003_create_users.up.sql` - Create users table
- `migrations/003_create_users.down.sql` - Drop users table

## Database Schema

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password TEXT NOT NULL,  -- bcrypt hashed
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
```

## Integration

### Updated Files:
1. **`internal/auth/handler.go`** - LoginHandler sekarang menggunakan UserService (real authentication)
2. **`cmd/api/main.go`** - Setup user service dan auth routes

## API Endpoints

### Public Endpoints

#### 1. Register User
```bash
POST /auth/register
Content-Type: application/json

{
  "username": "johndoe",
  "password": "password123"
}
```

**Response (Success - 201):**
```json
{
  "message": "User created successfully",
  "user_id": 1
}
```

**Response (Username Exists - 409):**
```json
{
  "error": "Username already exists"
}
```

**Validation:**
- Username: 3-50 characters, required
- Password: minimum 6 characters, required

#### 2. Login
```bash
POST /auth/login
Content-Type: application/json

{
  "username": "johndoe",
  "password": "password123"
}
```

**Response (Success - 200):**
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 900
}
```

**Response (Invalid Credentials - 401):**
```json
{
  "error": "Invalid credentials"
}
```

#### 3. Refresh Token
```bash
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGc..."
}
```

**Response (Success - 200):**
```json
{
  "access_token": "eyJhbGc...",
  "expires_in": 900
}
```

## Testing Flow

### 1. Run Migration
```bash
# Apply migration to create users table
psql -U your_user -d your_database -f migrations/003_create_users.up.sql
```

Or if you're using golang-migrate:
```bash
migrate -path migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up
```

### 2. Register New User
```bash
curl -X POST http://localhost:8087/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "test123456"
  }'
```

### 3. Login with Created User
```bash
curl -X POST http://localhost:8087/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "test123456"
  }'
```

Save the `access_token` from response.

### 4. Create Task with JWT
```bash
curl -X POST http://localhost:8087/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{
    "task_type": "email"
  }'
```

The `user_id` is automatically extracted from JWT token.

### 5. Get Task Status
```bash
curl -X GET http://localhost:8087/api/v1/tasks/1 \
  -H "Authorization: Bearer <access_token>"
```

## Security Features

### Password Hashing
- Uses `bcrypt` with default cost (10)
- Passwords never stored in plain text
- Password never exposed in JSON responses (`json:"-"` tag)

### JWT Integration
- Login validates credentials via `userService.ValidateCredentials()`
- Returns JWT tokens only on successful authentication
- UserID from authenticated user is used in token generation

### Validation
- Username: 3-50 characters, unique
- Password: minimum 6 characters (enforce stronger requirements in production)
- Database unique constraint on username

## Error Handling

### Common Errors:

1. **Username Already Exists (409)**
   ```json
   {"error": "Username already exists"}
   ```

2. **Invalid Credentials (401)**
   ```json
   {"error": "Invalid credentials"}
   ```

3. **Validation Error (400)**
   ```json
   {"error": "Key: 'RegisterRequest.Password' Error:Field validation for 'Password' failed on the 'min' tag"}
   ```

4. **Token Expired (401)**
   ```json
   {"error": "Token expired"}
   ```

## Production Recommendations

1. **Password Requirements**: Enforce stronger password policy
   ```go
   binding:"required,min=8,max=72" // bcrypt max is 72 bytes
   ```

2. **Rate Limiting**: Already configured in nginx (5 req/s)

3. **HTTPS**: Always use HTTPS in production

4. **Password Reset**: Implement forgot password functionality

5. **Email Verification**: Add email verification on registration

6. **Account Lockout**: Implement account lockout after failed login attempts

7. **Audit Logging**: Log authentication attempts

8. **Session Management**: Consider storing refresh tokens in database for revocation

## Additional Features to Consider

- [ ] Email verification on registration
- [ ] Password reset functionality  
- [ ] Account lockout after failed attempts
- [ ] User profile update endpoint
- [ ] Change password endpoint
- [ ] List all users (admin only)
- [ ] Delete user endpoint (admin only)
- [ ] Role-based access control (RBAC)

## Files Structure
```
internal/
├── user/
│   ├── model.go       # User struct
│   ├── repository.go  # Database operations
│   ├── service.go     # Business logic + bcrypt
│   └── handler.go     # HTTP handlers (register)
├── auth/
│   ├── jwt.go         # JWT utilities
│   └── handler.go     # Login & refresh handlers (updated)
└── task/
    └── handler.go     # Task handlers (updated)

migrations/
├── 003_create_users.up.sql
└── 003_create_users.down.sql
```
