# 📖 Archive API Documentation

## 🚀 Swagger UI

Dokumentasi API lengkap tersedia melalui Swagger UI yang interaktif.

### 🌐 Akses Dokumentasi

Setelah menjalankan server, akses dokumentasi melalui:

```
http://localhost:8080/swagger
```

Atau langsung ke:

```
http://localhost:8080/docs/swagger.html
```

### 🔐 Testing dengan Swagger UI

1. **Login untuk Testing:**

   - Di pojok kanan atas Swagger UI terdapat form "Quick Login"
   - Username default: `admin`
   - Password default: `admin123`
   - Klik "Login" untuk mendapatkan access token

2. **Testing Endpoints:**
   - Setelah login, token akan tersimpan otomatis
   - Semua protected endpoints akan menggunakan token tersebut
   - Klik "Try it out" pada endpoint yang ingin ditest
   - Isi parameter yang diperlukan
   - Klik "Execute"

### 📋 Struktur Dokumentasi

#### 🏷️ **Tags/Kategori:**

- **Health** - Health check endpoint
- **Authentication** - Login, refresh token, logout
- **User Profile** - Profil user dan logout dari semua device
- **Archives** - Manajemen file archive

#### 🔑 **Authentication:**

API menggunakan **JWT dengan RSA signature**:

- Access token berlaku 15 menit
- Refresh token berlaku 7 hari
- Token juga disimpan sebagai HTTP-only cookies
- Header Authorization: `Bearer <access_token>`

#### 📤 **File Upload:**

- **Format yang didukung:** PDF saja
- **Ukuran maksimal:** 50MB
- **Storage:** MongoDB GridFS
- **Fields required:** file, category, type
- **Optional:** tags (max 5), description

### 🛠️ **Endpoint Overview:**

#### **Public Endpoints:**

- `GET /health` - Health check
- `POST /auth/login` - User login
- `POST /auth/refresh` - Refresh access token

#### **Protected Endpoints:**

- `GET /api/v1/profile` - User profile
- `POST /api/v1/logout-all` - Logout dari semua device
- `POST /api/v1/archives` - Upload file
- `GET /api/v1/archives` - List archives
- `GET /api/v1/archives/{id}/download` - Download file
- `DELETE /api/v1/archives/{id}` - Soft delete
- `DELETE /api/v1/archives/{id}/permanent` - Hard delete
- `POST /api/v1/archives/{id}/restore` - Restore deleted file
- `GET /api/v1/archives/{id}/history` - File history
- `GET /api/v1/archives/category/{category}` - Filter by category
- `GET /api/v1/archives/tags?tags=tag1,tag2` - Filter by tags
- `POST /api/v1/archives/bulk` - Get multiple archives by IDs

### 📝 **Request/Response Examples:**

#### **Login Request:**

```json
{
  "username": "admin",
  "password": "admin123",
  "device_id": "optional-device-id"
}
```

#### **Login Response:**

```json
{
  "status": "success",
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "token_type": "Bearer",
    "user": {
      "id": "60d5ecb54b24a12d8c5f1234",
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin",
      "is_active": true
    }
  }
}
```

#### **Upload File Request (multipart/form-data):**

```
file: [PDF file]
category: "documents"
type: "contract"
tags: ["important", "legal"]
description: "Contract document for project X"
```

#### **Archive Response:**

```json
{
  "status": "success",
  "data": {
    "id": "60d5ecb54b24a12d8c5f1234",
    "name": "contract.pdf",
    "size": 1048576,
    "size_mb": "1.00 MB",
    "category": "documents",
    "type": "contract",
    "tags": ["important", "legal"],
    "description": "Contract document for project X",
    "owner_id": "60d5ecb54b24a12d8c5f5678",
    "version": 1,
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

### 🔧 **Error Responses:**

Semua error menggunakan format standar:

```json
{
  "status": "error",
  "message": "Error description",
  "error_code": "OPTIONAL_ERROR_CODE"
}
```

**Common HTTP Status Codes:**

- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `413` - Payload Too Large
- `500` - Internal Server Error

### 🎯 **Quick Start Testing:**

1. **Start the server:**

   ```bash
   go run cmd/main.go
   ```

2. **Setup initial user:**

   ```bash
   go run scripts/setup_auth.go
   ```

3. **Open Swagger UI:**

   ```
   http://localhost:8080/swagger
   ```

4. **Login in Swagger UI:**

   - Use Quick Login form (admin/admin123)
   - Or use the `/auth/login` endpoint

5. **Test file upload:**

   - Use `/api/v1/archives` POST endpoint
   - Upload a PDF file with required fields

6. **Test other endpoints:**
   - List archives: GET `/api/v1/archives`
   - Download file: GET `/api/v1/archives/{id}/download`
   - View history: GET `/api/v1/archives/{id}/history`

### 📁 **File Structure:**

```
archiven-api/
├── swagger.yaml          # OpenAPI 3.0 specification
├── docs/
│   └── swagger.html      # Swagger UI HTML
└── cmd/main.go           # Server dengan Swagger routes
```

### 🎨 **Customization:**

- **Theme:** Swagger UI menggunakan theme default dengan warna kustom
- **Authentication:** Quick login form untuk testing
- **Auto-token:** Token tersimpan otomatis di localStorage
- **Interactive:** Semua endpoint bisa ditest langsung dari UI

**🎉 Dokumentasi API lengkap dan siap digunakan!**
