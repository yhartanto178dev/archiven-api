# JWT RSA Authentication System

## 🔐 Complete Authentication System Implementation

This JWT RSA authentication system provides enterprise-grade security with access tokens, refresh tokens, and HTTP-only cookies.

## 🚀 Quick Start

### 1. Setup Database (with MongoDB)

```bash
# Run the setup script to create initial users
go run scripts/setup_auth.go
```

### 2. Test the JWT System

```bash
# Test the JWT authentication without MongoDB
go test -v ./cmd -run TestJWTFullFlow
```

### 3. Start the Server

```bash
# Start the server with authentication
go run cmd/main.go
```

## 📝 API Usage Examples

### Authentication Endpoints

#### Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123",
    "device_id": "web-browser-001"
  }'
```

**Response:**

```json
{
  "status": "success",
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIs...",
    "expires_in": 900,
    "token_type": "Bearer",
    "user": {
      "id": "...",
      "username": "admin",
      "email": "admin@archiven.com",
      "role": "admin",
      "is_active": true,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  }
}
```

#### Refresh Token

```bash
# Using cookie (automatic)
curl -X POST http://localhost:8080/auth/refresh \
  -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..."

# Or using request body
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "eyJhbGciOiJSUzI1NiIs..."}'
```

#### Logout

```bash
curl -X POST http://localhost:8080/auth/logout \
  -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..."
```

### Protected Endpoints

#### Get Profile

```bash
# Using Authorization header
curl -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
     -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..." \
     http://localhost:8080/api/v1/profile

# Or using access token cookie
curl -H "Cookie: access_token=eyJhbGciOiJSUzI1NiIs...; refresh_token=eyJhbGciOiJSUzI1NiIs..." \
     http://localhost:8080/api/v1/profile
```

#### Logout from All Devices

```bash
curl -X POST http://localhost:8080/api/v1/logout-all \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..."
```

### Archive Endpoints (Protected)

#### Upload File

```bash
curl -X POST http://localhost:8080/api/v1/archives \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..." \
  -F "file=@document.pdf" \
  -F "category=reports" \
  -F "type=financial" \
  -F "description=Q4 Financial Report"
```

#### List Archives

```bash
curl -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
     -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..." \
     "http://localhost:8080/api/v1/archives?page=1&limit=10"
```

## 🔧 Configuration

### Environment Variables

```env
# JWT Configuration
JWT_PRIVATE_KEY_PATH=./keys/private.pem
JWT_PUBLIC_KEY_PATH=./keys/public.pem
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
JWT_ISSUER=archiven-api

# Database
MONGODB_URI=mongodb://localhost:27017
DATABASE_NAME=archive_db

# Server
PORT=8080
```

### Default Users

- **Admin User**: `admin` / `admin123`
- **Regular User**: `user123` / `user123`

## 🛡️ Security Features

### Token Security

- **RSA-256 Signing**: More secure than HMAC
- **Short-lived Access Tokens**: 15 minutes (configurable)
- **Long-lived Refresh Tokens**: 7 days (configurable)
- **Token Rotation**: New refresh token on each refresh
- **Automatic Key Generation**: RSA keys auto-generated if missing

### Cookie Security

- **HTTP-Only**: Prevents XSS attacks
- **Secure Flag**: HTTPS only in production
- **SameSite**: CSRF protection
- **Automatic Management**: Set/clear cookies automatically

### Additional Security

- **Password Hashing**: bcrypt with secure defaults
- **Device Tracking**: Multiple device support
- **Token Revocation**: Individual or bulk token revocation
- **Role-Based Access**: Middleware for role checking

## 🏗️ Architecture

### Components

1. **Domain Layer** (`internal/auth/domain/`)

   - User models and interfaces
   - Error definitions
   - Business logic contracts

2. **Infrastructure Layer** (`internal/auth/infrastructure/`)

   - JWT service implementation
   - Database repositories
   - External service integrations

3. **Application Layer** (`internal/auth/application/`)

   - Business logic services
   - Use case implementations
   - Service orchestration

4. **Interface Layer** (`internal/auth/interfaces/`)
   - HTTP handlers
   - Middleware implementations
   - Request/response formatting

### Token Flow

```
1. User Login → Generate Access & Refresh Tokens
2. Set HTTP-Only Cookies → Store Refresh Token
3. API Requests → Validate Access Token
4. Token Expired → Use Refresh Token
5. Generate New Tokens → Rotate Refresh Token
6. Logout → Revoke Tokens & Clear Cookies
```

## 🧪 Testing

The system includes comprehensive tests:

```bash
# Test JWT functionality
go run cmd/test_jwt.go

# Test with real database (requires MongoDB)
go run cmd/setup_auth.go
```

## 🚦 Error Handling

The system provides detailed error responses:

```json
{
  "status": "error",
  "message": "Token expired"
}
```

**Common Error Types:**

- `Invalid credentials`
- `User not found`
- `User account is inactive`
- `Access token required`
- `Invalid token`
- `Token expired`
- `Token revoked`
- `Insufficient permissions`

## 🔄 Token Refresh Best Practices

1. **Automatic Refresh**: Implement client-side automatic token refresh
2. **Silent Refresh**: Use refresh tokens in background
3. **Fallback to Login**: Redirect to login if refresh fails
4. **Secure Storage**: Store tokens in HTTP-only cookies when possible

## 📱 Frontend Integration

### JavaScript Example

```javascript
// Login function
async function login(username, password) {
  const response = await fetch("/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include", // Include cookies
    body: JSON.stringify({ username, password }),
  });

  const data = await response.json();
  if (data.status === "success") {
    // Store access token (optional, can use cookie)
    localStorage.setItem("access_token", data.data.access_token);
    return data.data.user;
  }
}

// API request with automatic token refresh
async function apiRequest(url, options = {}) {
  const token = localStorage.getItem("access_token");

  const response = await fetch(url, {
    ...options,
    credentials: "include",
    headers: {
      ...options.headers,
      Authorization: `Bearer ${token}`,
    },
  });

  if (response.status === 401) {
    // Try to refresh token
    const refreshResponse = await fetch("/auth/refresh", {
      method: "POST",
      credentials: "include",
    });

    if (refreshResponse.ok) {
      const refreshData = await refreshResponse.json();
      localStorage.setItem("access_token", refreshData.data.access_token);

      // Retry original request
      return fetch(url, {
        ...options,
        credentials: "include",
        headers: {
          ...options.headers,
          Authorization: `Bearer ${refreshData.data.access_token}`,
        },
      });
    } else {
      // Redirect to login
      window.location.href = "/login";
    }
  }

  return response;
}
```

This authentication system provides enterprise-grade security with modern best practices for web applications.
