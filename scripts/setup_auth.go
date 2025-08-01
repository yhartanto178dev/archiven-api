package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"github.com/yhartanto178dev/archiven-api/internal/auth/domain"
	authInfra "github.com/yhartanto178dev/archiven-api/internal/auth/infrastructure"
	"github.com/yhartanto178dev/archiven-api/internal/configs"
)

func main() {
	// Load configuration
	cfg := configs.LoadConfig()

	fmt.Println("🔗 Connecting to MongoDB...")
	fmt.Printf("   URI: %s\n", cfg.MongoURI)
	fmt.Printf("   Database: %s\n", cfg.DatabaseName)

	// Connect to MongoDB with retry
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Force IPv4 connection
	clientOptions := options.Client().ApplyURI(cfg.MongoURI)
	clientOptions.SetDirect(true)
	clientOptions.SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("❌ Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Test connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("❌ MongoDB ping failed: %v", err)
	}
	fmt.Println("✅ MongoDB connection successful")

	db := client.Database(cfg.DatabaseName)
	userRepo := authInfra.NewUserRepository(db)

	// Create admin user
	fmt.Println("👤 Creating admin user...")
	adminPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	adminUser := &domain.User{
		Username: "admin",
		Email:    "admin@archiven.com",
		Password: string(adminPassword),
		Role:     "admin",
	}

	if err := userRepo.Create(adminUser); err != nil {
		log.Printf("⚠️  Failed to create admin user (might already exist): %v", err)
	} else {
		fmt.Println("✅ Admin user created successfully")
		fmt.Println("   Username: admin")
		fmt.Println("   Password: admin123")
	}

	// Create regular user
	fmt.Println("👤 Creating regular user...")
	userPassword, _ := bcrypt.GenerateFromPassword([]byte("user123"), bcrypt.DefaultCost)
	regularUser := &domain.User{
		Username: "user123",
		Email:    "user@archiven.com",
		Password: string(userPassword),
		Role:     "user",
	}

	if err := userRepo.Create(regularUser); err != nil {
		log.Printf("⚠️  Failed to create regular user (might already exist): %v", err)
	} else {
		fmt.Println("✅ Regular user created successfully")
		fmt.Println("   Username: user123")
		fmt.Println("   Password: user123")
	}

	fmt.Println("🎉 Database setup completed!")
	fmt.Println("📖 You can now test the API with Swagger: http://localhost:8080/swagger")
}
