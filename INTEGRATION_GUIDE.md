# Archive API Usage Examples with Authentication

## 🔐 Step 1: Login First

```bash
# Login untuk mendapatkan access token dan refresh token
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
      "role": "admin"
    }
  }
}
```

**Important:** Cookie `refresh_token` juga akan di-set otomatis!

## 📁 Step 2: Access Archive Endpoints

### Upload File (PROTECTED)

```bash
curl -X POST http://localhost:8080/api/v1/archives \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..." \
  -F "file=@document.pdf" \
  -F "category=reports" \
  -F "type=financial" \
  -F "description=Q4 Financial Report"
```

### List Archives (PROTECTED)

```bash
curl -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
     -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..." \
     "http://localhost:8080/api/v1/archives?page=1&limit=10"
```

### Download File (PROTECTED)

```bash
curl -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
     -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..." \
     "http://localhost:8080/api/v1/archives/FILE_ID/download" \
     -o downloaded_file.pdf
```

### Delete File (PROTECTED)

```bash
curl -X DELETE \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..." \
  "http://localhost:8080/api/v1/archives/FILE_ID"
```

## 🔄 Step 3: Token Refresh (When Needed)

Jika access token expired (15 menit), otomatis refresh:

```bash
curl -X POST http://localhost:8080/auth/refresh \
  -H "Cookie: refresh_token=eyJhbGciOiJSUzI1NiIs..."
```

## ❌ Step 4: Error Responses

### Tanpa Login:

```bash
curl http://localhost:8080/api/v1/archives
# Response: 401 Unauthorized
{
  "status": "error",
  "message": "Access token required"
}
```

### Token Expired:

```bash
curl -H "Authorization: Bearer EXPIRED_TOKEN" \
     http://localhost:8080/api/v1/archives
# Response: 401 Unauthorized
{
  "status": "error",
  "message": "Token expired"
}
```

## 🔧 Frontend Integration Example

### JavaScript/React Example:

```javascript
class ArchiveAPI {
  constructor() {
    this.baseURL = "http://localhost:8080";
    this.accessToken = localStorage.getItem("access_token");
  }

  // Login
  async login(username, password) {
    const response = await fetch(`${this.baseURL}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include", // Important untuk cookies
      body: JSON.stringify({ username, password }),
    });

    const data = await response.json();
    if (data.status === "success") {
      this.accessToken = data.data.access_token;
      localStorage.setItem("access_token", this.accessToken);
      return data.data.user;
    }
    throw new Error(data.message);
  }

  // Upload file dengan auth
  async uploadFile(file, metadata) {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("category", metadata.category);
    formData.append("type", metadata.type);
    formData.append("description", metadata.description);

    return this.authenticatedRequest("/api/v1/archives", {
      method: "POST",
      body: formData,
    });
  }

  // List archives dengan auth
  async listArchives(page = 1, limit = 10) {
    return this.authenticatedRequest(
      `/api/v1/archives?page=${page}&limit=${limit}`
    );
  }

  // Helper method untuk authenticated requests
  async authenticatedRequest(url, options = {}) {
    const response = await fetch(`${this.baseURL}${url}`, {
      ...options,
      credentials: "include",
      headers: {
        ...options.headers,
        Authorization: `Bearer ${this.accessToken}`,
      },
    });

    // Auto refresh token jika expired
    if (response.status === 401) {
      const refreshed = await this.refreshToken();
      if (refreshed) {
        // Retry request dengan token baru
        return fetch(`${this.baseURL}${url}`, {
          ...options,
          credentials: "include",
          headers: {
            ...options.headers,
            Authorization: `Bearer ${this.accessToken}`,
          },
        });
      } else {
        // Redirect ke login
        window.location.href = "/login";
      }
    }

    return response;
  }

  // Refresh token
  async refreshToken() {
    try {
      const response = await fetch(`${this.baseURL}/auth/refresh`, {
        method: "POST",
        credentials: "include",
      });

      const data = await response.json();
      if (data.status === "success") {
        this.accessToken = data.data.access_token;
        localStorage.setItem("access_token", this.accessToken);
        return true;
      }
    } catch (error) {
      console.error("Refresh token failed:", error);
    }
    return false;
  }
}

// Usage
const api = new ArchiveAPI();

// Login dulu
await api.login("admin", "admin123");

// Sekarang bisa akses archives
const archives = await api.listArchives();
const uploadResult = await api.uploadFile(file, {
  category: "reports",
  type: "financial",
  description: "Q4 Report",
});
```

## 🛡️ Security Features yang Aktif:

1. **JWT Authentication**: Semua `/api/v1/*` routes protected
2. **Cookie Security**: HTTP-only cookies untuk refresh tokens
3. **CORS Protection**: Only allowed origins can access
4. **Token Rotation**: New refresh token pada setiap refresh
5. **Role-based Access**: User data tersedia di context untuk authorization
6. **Request Logging**: Semua request tercatat dengan user info

## 📋 Summary:

✅ **Authentication & Archive sudah terintegrasi sempurna!**

- Login dulu → dapat access token
- Gunakan access token untuk semua archive operations
- Token expired → auto refresh dengan refresh token
- Logout → revoke semua tokens

Tidak ada yang perlu diubah - sistem sudah production-ready! 🚀
