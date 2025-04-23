package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/yhartanto178dev/archiven-api/internal/archive/application"
	"github.com/yhartanto178dev/archiven-api/internal/archive/infrastructure"
	"github.com/yhartanto178dev/archiven-api/internal/configs"
	interfaces "github.com/yhartanto178dev/archiven-api/internal/interface"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {

	// Load environment variables
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, using system environment variables")
	}

	cfg := configs.Load()
	// MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// Initialize repository
	repo, err := infrastructure.NewArchiveRepository(client, cfg.DBName)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize service
	service := application.NewArchiveService(repo)

	// Initialize handlers
	handler := interfaces.NewArchiveHandler(service, cfg)

	// Echo setup
	e := echo.New()

	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Tambahkan di middleware
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:     true,
		LogStatus:  true,
		LogMethod:  true,
		LogLatency: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info("request",
				zap.String("method", v.Method),
				zap.String("uri", v.URI),
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
			)
			return nil
		},
	}))

	// Routes
	e.POST("/archives", handler.Upload)
	e.GET("/archives", handler.List)
	e.GET("/download/:id", handler.Download)
	e.GET("/archives/list", handler.GetByIDs)
	// Start server
	fmt.Println("Server running on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}
