// MongoDB initialization script
// This script will run when MongoDB container starts for the first time

// Switch to archiven database
db = db.getSiblingDB('archiven');

// Create collections with indexes
db.users.createIndex({ "username": 1 }, { unique: true });
db.users.createIndex({ "email": 1 }, { unique: true });

db.refresh_tokens.createIndex({ "token": 1 }, { unique: true });
db.refresh_tokens.createIndex({ "user_id": 1 });
db.refresh_tokens.createIndex({ "expires_at": 1 }, { expireAfterSeconds: 0 });

db.archives.createIndex({ "name": "text", "description": "text" });
db.archives.createIndex({ "owner_id": 1 });
db.archives.createIndex({ "category": 1 });
db.archives.createIndex({ "tags": 1 });
db.archives.createIndex({ "created_at": 1 });
db.archives.createIndex({ "deleted_at": 1 });

// Create GridFS indexes
db.fs.files.createIndex({ "filename": 1 });
db.fs.files.createIndex({ "uploadDate": 1 });
db.fs.chunks.createIndex({ "files_id": 1, "n": 1 }, { unique: true });

print("Database initialization completed!");
print("Collections and indexes created successfully.");
