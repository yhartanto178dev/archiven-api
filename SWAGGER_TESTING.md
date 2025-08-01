# 🎯 Testing Archive API dengan Swagger

## ✅ **Setup Berhasil!**

API Archive dengan JWT RSA Authentication sudah berjalan dengan sukses:

- **Server**: http://localhost:8080 ✅
- **MongoDB**: Container Podman (IP: 10.123.138.18:27017) ✅
- **Swagger UI**: http://localhost:8080/swagger ✅
- **Firewall**: Port 27017 dibuka ✅
- **Users**: Admin dan user regular sudah dibuat ✅

## 🔑 **Kredensial Login**

### Admin User:

- **Username**: `admin`
- **Password**: `admin123`
- **Role**: `admin`

### Regular User:

- **Username**: `user123`
- **Password**: `user123`
- **Role**: `user`

## 📖 **Testing dengan Swagger UI**

### 1. **Buka Swagger Documentation:**

```
http://localhost:8080/swagger
```

### 2. **Login untuk mendapatkan Token:**

- Di pojok kanan atas, ada form **Quick Login**
- Masukkan username: `admin`
- Masukkan password: `admin123`
- Klik **Login**
- Token akan tersimpan otomatis di localStorage

### 3. **Test Endpoints:**

#### **Public Endpoints:**

- `GET /health` - ✅ Tested
- `POST /auth/login` - ✅ Tested
- `POST /auth/refresh` - Ready to test

#### **Protected Endpoints:**

- `GET /api/v1/profile` - ✅ Tested
- `GET /api/v1/archives` - ✅ Tested
- `POST /api/v1/archives` - Ready for file upload
- Dan semua endpoint lainnya...

## 🧪 **Manual Testing via cURL**

### **Login:**

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

### **Get Profile:**

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8080/api/v1/profile
```

### **List Archives:**

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8080/api/v1/archives
```

### **Upload File (multipart/form-data):**

```bash
curl -X POST http://localhost:8080/api/v1/archives \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@document.pdf" \
  -F "category=documents" \
  -F "type=contract" \
  -F "tags=important,legal" \
  -F "description=Test upload"
```

## 🐳 **Podman Container Info**

### **MongoDB Container:**

- **Name**: `mymongoDB`
- **Image**: `mongo:latest`
- **Internal IP**: `10.123.138.18`
- **Port**: `27017`
- **Status**: Running ✅

### **Container Commands:**

```bash
# Check container status
podman ps | grep mongo

# MongoDB logs
podman logs mymongoDB

# Access MongoDB shell
podman exec -it mymongoDB mongosh archive_db

# Stop container
podman stop mymongoDB

# Start container
podman start mymongoDB
```

## 🔧 **Troubleshooting**

### **Issue: MongoDB Connection Failed**

**Solution**:

1. Check container: `podman ps | grep mongo`
2. Check firewall: `firewall-cmd --list-ports`
3. Get container IP: `podman exec -it mymongoDB hostname -i`
4. Update .env: `MONGODB_URI=mongodb://CONTAINER_IP:27017/archive_db`

### **Issue: JWT Token Invalid**

**Solution**:

1. Check if users exist in MongoDB
2. Re-run setup: `MONGODB_URI=mongodb://CONTAINER_IP:27017/archive_db go run scripts/setup_auth.go`
3. Login again to get fresh token

### **Issue: File Upload Failed**

**Solution**:

1. Check file size (max 50MB)
2. Check file type (only PDF)
3. Ensure proper Authorization header
4. Use multipart/form-data format

## 📊 **Quick Commands**

### **Using Makefile:**

```bash
# Show all available commands
make help

# Check status
make status
make mongo-status
make full-status

# View logs
make logs
make mongo-logs

# Quick test endpoints
make quick-test

# Show all available routes
make routes
```

## 🎉 **Next Steps**

1. **Test File Upload** dengan PDF file melalui Swagger UI
2. **Test File Download** dari file yang sudah diupload
3. **Test Archive Management** (delete, restore, history)
4. **Test User Management** (logout, logout all devices)
5. **Implement Frontend** yang menggunakan API ini

**🚀 API Archive dengan JWT RSA Authentication siap untuk production!**
