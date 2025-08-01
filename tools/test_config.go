package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/yhartanto178dev/archiven-api/internal/configs"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Load configurations
	cfg := configs.LoadConfig()
	authCfg := configs.LoadAuthConfig()

	fmt.Println("🔧 Configuration Loaded Successfully!")
	fmt.Println("=====================================")

	// Main Config
	fmt.Printf("🌐 Server Port: %s\n", cfg.Port)
	fmt.Printf("🗄️  MongoDB URI: %s\n", cfg.MongoURI)
	fmt.Printf("📂 Database Name: %s\n", cfg.DatabaseName)
	fmt.Printf("📁 Upload Max Size: %d bytes (%.2f MB)\n", cfg.MaxUploadSize, float64(cfg.MaxUploadSize)/1024/1024)
	fmt.Printf("📄 Allowed Types: %v\n", cfg.AllowedTypes)
	fmt.Printf("🌍 Allowed Origins: %v\n", cfg.AllowedOrigins)
	fmt.Printf("📋 Log Level: %s\n", cfg.LogLevel)
	fmt.Printf("📁 Log Directory: %s\n", cfg.LogDir)

	fmt.Println()

	// Auth Config
	fmt.Printf("🔐 JWT Issuer: %s\n", authCfg.JWTIssuer)
	fmt.Printf("🔑 Private Key Path: %s\n", authCfg.JWTPrivateKeyPath)
	fmt.Printf("🔑 Public Key Path: %s\n", authCfg.JWTPublicKeyPath)
	fmt.Printf("⏰ Access Token TTL: %v\n", authCfg.AccessTokenTTL)
	fmt.Printf("⏰ Refresh Token TTL: %v\n", authCfg.RefreshTokenTTL)

	fmt.Println()
	fmt.Println("✅ All configurations loaded successfully!")
	fmt.Println("🚀 Ready to start the server!")
}
