// MongoDB commands to create collections and indexes

// Create users collection
db.users.createIndex({ "username": 1 }, { unique: true })
db.users.createIndex({ "email": 1 }, { unique: true })
db.users.createIndex({ "created_at": 1 })

// Create refresh_tokens collection
db.refresh_tokens.createIndex({ "token": 1 }, { unique: true })
db.refresh_tokens.createIndex({ "user_id": 1 })
db.refresh_tokens.createIndex({ "expires_at": 1 }, { expireAfterSeconds: 0 })
db.refresh_tokens.createIndex({ "created_at": 1 })

// Create initial admin user (password: admin123)
db.users.insertOne({
  "username": "admin",
  "email": "admin@archiven.com",
  "password": "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // bcrypt hash of "password"
  "role": "admin",
  "is_active": true,
  "created_at": new Date(),
  "updated_at": new Date()
})

// Create test user (password: user123)
db.users.insertOne({
  "username": "user123",
  "email": "user@archiven.com",
  "password": "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // bcrypt hash of "password"
  "role": "user",
  "is_active": true,
  "created_at": new Date(),
  "updated_at": new Date()
})
