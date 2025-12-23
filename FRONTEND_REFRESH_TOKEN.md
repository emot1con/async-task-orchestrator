# Frontend Implementation: Refresh Token Logic

## Overview
Backend menyediakan endpoint, **Frontend yang mengatur logika refresh**.

## Backend Responsibilities (Already Done ✅)

```
POST /auth/login    → Generate tokens
POST /auth/refresh  → Validate & return new tokens
GET  /api/v1/*      → Validate access token (middleware)
```

Backend **TIDAK** perlu tahu:
- ❌ Kapan access token expired
- ❌ Kapan trigger refresh
- ❌ Token lifecycle management

## Frontend Responsibilities

```javascript
✅ Store tokens securely
✅ Detect 401 (token expired)
✅ Call /auth/refresh automatically
✅ Retry original request with new token
✅ Handle refresh failure → redirect to login
✅ Update both tokens after refresh (rotation)
```

---

## 🔄 **Token Rotation Flow**

### Backend Response (with rotation)
```json
// POST /auth/refresh
{
  "access_token": "eyJ... (NEW 15 min)",
  "refresh_token": "eyJ... (NEW 7 days)",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Important:** Old refresh token is now **INVALID** after rotation!

---

## 💻 **Frontend Implementation Examples**

### **1. Vanilla JavaScript / Fetch API**

```javascript
class AuthClient {
    constructor() {
        this.accessToken = null;
        this.refreshToken = localStorage.getItem('refresh_token');
        this.isRefreshing = false;
        this.failedQueue = [];
    }

    // Process queue after token refresh
    processQueue(error, token = null) {
        this.failedQueue.forEach(prom => {
            if (error) {
                prom.reject(error);
            } else {
                prom.resolve(token);
            }
        });
        this.failedQueue = [];
    }

    // Login
    async login(username, password) {
        const response = await fetch('/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        if (!response.ok) {
            throw new Error('Login failed');
        }

        const data = await response.json();
        this.accessToken = data.access_token;
        this.refreshToken = data.refresh_token;
        
        // Store refresh token securely
        localStorage.setItem('refresh_token', data.refresh_token);
        
        return data;
    }

    // Refresh tokens
    async refreshTokens() {
        try {
            const response = await fetch('/auth/refresh', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ 
                    refresh_token: this.refreshToken 
                })
            });

            if (!response.ok) {
                throw new Error('Refresh failed');
            }

            const data = await response.json();
            
            // ✅ Update BOTH tokens (rotation)
            this.accessToken = data.access_token;
            this.refreshToken = data.refresh_token;
            localStorage.setItem('refresh_token', data.refresh_token);
            
            return data.access_token;
        } catch (error) {
            // Refresh failed, clear tokens and redirect to login
            this.logout();
            window.location.href = '/login';
            throw error;
        }
    }

    // Main request method with auto-refresh
    async request(url, options = {}) {
        // Add access token to request
        options.headers = {
            ...options.headers,
            'Authorization': `Bearer ${this.accessToken}`
        };

        let response = await fetch(url, options);

        // If 401, try to refresh
        if (response.status === 401) {
            if (this.isRefreshing) {
                // Already refreshing, queue this request
                return new Promise((resolve, reject) => {
                    this.failedQueue.push({ resolve, reject });
                }).then(token => {
                    options.headers['Authorization'] = `Bearer ${token}`;
                    return fetch(url, options);
                });
            }

            this.isRefreshing = true;

            try {
                const newToken = await this.refreshTokens();
                this.isRefreshing = false;
                
                // Process queued requests
                this.processQueue(null, newToken);
                
                // Retry original request with new token
                options.headers['Authorization'] = `Bearer ${newToken}`;
                response = await fetch(url, options);
            } catch (error) {
                this.isRefreshing = false;
                this.processQueue(error, null);
                throw error;
            }
        }

        return response;
    }

    // Logout
    logout() {
        this.accessToken = null;
        this.refreshToken = null;
        localStorage.removeItem('refresh_token');
    }
}

// Usage
const auth = new AuthClient();

// Login
await auth.login('user', 'password');

// Make requests (auto-refresh on 401)
const response = await auth.request('/api/v1/tasks');
const data = await response.json();

// Logout
auth.logout();
```

---

### **2. Axios Interceptor**

```javascript
import axios from 'axios';

const api = axios.create({
    baseURL: 'http://localhost:8087'
});

let isRefreshing = false;
let failedQueue = [];

const processQueue = (error, token = null) => {
    failedQueue.forEach(prom => {
        if (error) {
            prom.reject(error);
        } else {
            prom.resolve(token);
        }
    });
    failedQueue = [];
};

// Request interceptor - Add access token
api.interceptors.request.use(
    config => {
        const accessToken = localStorage.getItem('access_token');
        if (accessToken) {
            config.headers['Authorization'] = `Bearer ${accessToken}`;
        }
        return config;
    },
    error => Promise.reject(error)
);

// Response interceptor - Handle 401 and refresh
api.interceptors.response.use(
    response => response,
    async error => {
        const originalRequest = error.config;

        // If 401 and not already retried
        if (error.response?.status === 401 && !originalRequest._retry) {
            if (isRefreshing) {
                // Queue this request
                return new Promise((resolve, reject) => {
                    failedQueue.push({ resolve, reject });
                }).then(token => {
                    originalRequest.headers['Authorization'] = `Bearer ${token}`;
                    return api(originalRequest);
                }).catch(err => Promise.reject(err));
            }

            originalRequest._retry = true;
            isRefreshing = true;

            const refreshToken = localStorage.getItem('refresh_token');

            if (!refreshToken) {
                isRefreshing = false;
                window.location.href = '/login';
                return Promise.reject(error);
            }

            try {
                // Call refresh endpoint
                const response = await axios.post('/auth/refresh', {
                    refresh_token: refreshToken
                });

                const { access_token, refresh_token: newRefreshToken } = response.data;

                // ✅ Update BOTH tokens (rotation)
                localStorage.setItem('access_token', access_token);
                localStorage.setItem('refresh_token', newRefreshToken);

                // Update authorization header
                api.defaults.headers.common['Authorization'] = `Bearer ${access_token}`;
                originalRequest.headers['Authorization'] = `Bearer ${access_token}`;

                // Process queued requests
                processQueue(null, access_token);

                isRefreshing = false;

                // Retry original request
                return api(originalRequest);
            } catch (refreshError) {
                processQueue(refreshError, null);
                isRefreshing = false;

                // Refresh failed, logout
                localStorage.removeItem('access_token');
                localStorage.removeItem('refresh_token');
                window.location.href = '/login';

                return Promise.reject(refreshError);
            }
        }

        return Promise.reject(error);
    }
);

export default api;

// Usage
import api from './api';

// Login
const { data } = await api.post('/auth/login', { username, password });
localStorage.setItem('access_token', data.access_token);
localStorage.setItem('refresh_token', data.refresh_token);

// Make requests (auto-refresh on 401)
const tasks = await api.get('/api/v1/tasks');
```

---

### **3. React Hook (with Axios)**

```jsx
import { useState, useEffect, useCallback } from 'react';
import axios from 'axios';

const api = axios.create({
    baseURL: 'http://localhost:8087'
});

export const useAuth = () => {
    const [accessToken, setAccessToken] = useState(
        localStorage.getItem('access_token')
    );
    const [refreshToken, setRefreshToken] = useState(
        localStorage.getItem('refresh_token')
    );
    const [isRefreshing, setIsRefreshing] = useState(false);

    // Setup axios interceptors
    useEffect(() => {
        const requestInterceptor = api.interceptors.request.use(
            config => {
                if (accessToken) {
                    config.headers['Authorization'] = `Bearer ${accessToken}`;
                }
                return config;
            }
        );

        const responseInterceptor = api.interceptors.response.use(
            response => response,
            async error => {
                const originalRequest = error.config;

                if (error.response?.status === 401 && !originalRequest._retry) {
                    originalRequest._retry = true;

                    if (!isRefreshing && refreshToken) {
                        setIsRefreshing(true);

                        try {
                            const { data } = await axios.post('/auth/refresh', {
                                refresh_token: refreshToken
                            });

                            // ✅ Update BOTH tokens
                            setAccessToken(data.access_token);
                            setRefreshToken(data.refresh_token);
                            localStorage.setItem('access_token', data.access_token);
                            localStorage.setItem('refresh_token', data.refresh_token);

                            originalRequest.headers['Authorization'] = 
                                `Bearer ${data.access_token}`;

                            setIsRefreshing(false);
                            return api(originalRequest);
                        } catch (refreshError) {
                            setIsRefreshing(false);
                            logout();
                            return Promise.reject(refreshError);
                        }
                    }
                }

                return Promise.reject(error);
            }
        );

        return () => {
            api.interceptors.request.eject(requestInterceptor);
            api.interceptors.response.eject(responseInterceptor);
        };
    }, [accessToken, refreshToken, isRefreshing]);

    const login = useCallback(async (username, password) => {
        const { data } = await api.post('/auth/login', { username, password });
        
        setAccessToken(data.access_token);
        setRefreshToken(data.refresh_token);
        localStorage.setItem('access_token', data.access_token);
        localStorage.setItem('refresh_token', data.refresh_token);
        
        return data;
    }, []);

    const logout = useCallback(() => {
        setAccessToken(null);
        setRefreshToken(null);
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        window.location.href = '/login';
    }, []);

    return {
        accessToken,
        refreshToken,
        isAuthenticated: !!accessToken,
        login,
        logout,
        api
    };
};

// Usage in component
function App() {
    const { login, logout, isAuthenticated, api } = useAuth();

    const handleLogin = async () => {
        await login('user', 'password');
    };

    const fetchTasks = async () => {
        const { data } = await api.get('/api/v1/tasks');
        console.log(data);
    };

    return (
        <div>
            {isAuthenticated ? (
                <>
                    <button onClick={fetchTasks}>Get Tasks</button>
                    <button onClick={logout}>Logout</button>
                </>
            ) : (
                <button onClick={handleLogin}>Login</button>
            )}
        </div>
    );
}
```

---

## 🔒 **Security Best Practices**

### 1. **Token Storage**

```javascript
// ❌ BAD - XSS vulnerable
localStorage.setItem('access_token', token);
localStorage.setItem('refresh_token', token);

// ✅ BETTER - Access token in memory only
let accessToken = null; // In memory
localStorage.setItem('refresh_token', token); // Only refresh token

// ✅ BEST - HttpOnly cookies (set by server)
// Set-Cookie: refresh_token=...; HttpOnly; Secure; SameSite=Strict
// Frontend doesn't need to handle storage
```

### 2. **Request Queuing**

Prevent multiple refresh calls when multiple requests fail:

```javascript
if (isRefreshing) {
    // Queue request instead of refreshing again
    return new Promise((resolve, reject) => {
        failedQueue.push({ resolve, reject });
    });
}
```

### 3. **Refresh Token Rotation**

```javascript
// ✅ Always update BOTH tokens after refresh
const { access_token, refresh_token } = response.data;
localStorage.setItem('access_token', access_token);
localStorage.setItem('refresh_token', refresh_token); // NEW token
```

---

## 📊 **Testing Checklist**

```bash
# 1. Login works
✅ Store both tokens

# 2. API calls work with valid token
✅ Include Authorization header

# 3. Token expiration detected
✅ Backend returns 401 after 15 minutes

# 4. Auto-refresh works
✅ Frontend calls /auth/refresh
✅ Original request retried with new token
✅ Both tokens updated

# 5. Refresh failure handled
✅ Clear tokens
✅ Redirect to login

# 6. Multiple simultaneous requests
✅ Only one refresh call
✅ Other requests queued
✅ All requests resume after refresh
```

---

## 🎯 **Summary**

| Responsibility | Backend | Frontend |
|----------------|---------|----------|
| Provide endpoints | ✅ | ❌ |
| Validate tokens | ✅ | ❌ |
| Generate tokens | ✅ | ❌ |
| Store tokens | ❌ | ✅ |
| Detect expiration | ❌ | ✅ |
| Trigger refresh | ❌ | ✅ |
| Retry requests | ❌ | ✅ |
| Handle rotation | ❌ | ✅ |

**Backend:** "Here are the tools (endpoints)"
**Frontend:** "I'll manage the lifecycle"

This is the **correct and standard approach**! 🎉
