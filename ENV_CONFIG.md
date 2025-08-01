# Environment Configuration Guide

## 📋 Complete .env Configuration

Copy `.env.example` to `.env` and configure your environment:

```bash
cp .env.example .env
```

### 🔧 Server Configuration

```env
SERVER_PORT=8080          # Server port (legacy)
PORT=8080                 # Server port (preferred)
HOST=localhost            # Server host
```

### 🗄️ Database Configuration

```env
MONGODB_URI=mongodb://localhost:27017    # MongoDB connection string
DB_NAME=archive_db                       # Database name (legacy)
DATABASE_NAME=archive_db                 # Database name (preferred)
```

### 📁 File Upload Configuration

```env
ALLOWED_TYPES=application/pdf            # Allowed MIME types (comma separated)
MAX_UPLOAD_SIZE=52428800                # Max file size in bytes (50MB)
```

### 📋 Logging Configuration

```env
LOG_DIR=logs                            # Log directory
LOG_FILE_FORMAT=2006-01-02.log         # Log file naming format
LOG_RETENTION_DAYS=7                   # Days to keep log files
LOG_LEVEL=info                         # Log level (debug, info, warn, error)
```

### 🔐 JWT Authentication Configuration

```env
JWT_PRIVATE_KEY_PATH=./keys/private.pem # RSA private key path
JWT_PUBLIC_KEY_PATH=./keys/public.pem   # RSA public key path
ACCESS_TOKEN_TTL=15m                    # Access token lifetime
REFRESH_TOKEN_TTL=168h                  # Refresh token lifetime (7 days)
JWT_ISSUER=archiven-api                 # JWT issuer name
```

### 🍪 Cookie Configuration

```env
COOKIE_SECURE=false                     # Set true for HTTPS only
COOKIE_SAME_SITE=lax                    # SameSite policy (strict, lax, none)
```

### 🌐 CORS Configuration

```env
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080,https://yourdomain.com
```

## 🔍 Configuration Validation

Run the configuration test to verify all settings:

```bash
go run cmd/test_config.go
```

Expected output:

```
🔧 Configuration Loaded Successfully!
=====================================
🌐 Server Port: 8080
🗄️  MongoDB URI: mongodb://localhost:27017
📂 Database Name: archive_db
📁 Upload Max Size: 52428800 bytes (50.00 MB)
📄 Allowed Types: [application/pdf]
📋 Log Level: info
📁 Log Directory: logs

🔐 JWT Issuer: archiven-api
🔑 Private Key Path: ./keys/private.pem
🔑 Public Key Path: ./keys/public.pem
⏰ Access Token TTL: 15m0s
⏰ Refresh Token TTL: 168h0m0s

✅ All configurations loaded successfully!
🚀 Ready to start the server!
```

## 🚀 Production Configuration

For production, update these values:

```env
# Production Server
PORT=8080
HOST=0.0.0.0

# Production Database
MONGODB_URI=mongodb://username:password@production-host:27017/archive_db

# Security Settings
COOKIE_SECURE=true
COOKIE_SAME_SITE=strict
ALLOWED_ORIGINS=https://yourdomain.com

# File Upload (adjust as needed)
MAX_UPLOAD_SIZE=104857600  # 100MB for production

# Logging
LOG_LEVEL=warn
LOG_RETENTION_DAYS=30
```

## 🛡️ Security Notes

1. **Never commit `.env` file to version control**
2. **Use strong JWT secrets in production**
3. **Set COOKIE_SECURE=true for HTTPS**
4. **Limit ALLOWED_ORIGINS to your domain only**
5. **Use environment-specific MongoDB credentials**

## 🔧 Environment Variables Priority

The application loads configuration in this order:

1. Environment variables (highest priority)
2. `.env` file
3. Default values in code (lowest priority)

This allows you to override any setting via environment variables for containerized deployments.
