package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/yhartanto178dev/archiven-api/internal/configs"
	"github.com/yhartanto178dev/archiven-api/internal/interfaces"

	"github.com/yhartanto178dev/archiven-api/internal/loggers"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(cfg.MongoURI).
		SetMaxPoolSize(100).
		SetMinPoolSize(5).
		SetMaxConnecting(20).
		SetWriteConcern(writeconcern.W1())

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// Echo setup
	e := echo.New()

	// Initialize logger
	// Inisialisasi logger produksi Zap
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	// Contoh penggunaan logger
	zapLogger.Info("Application started",
		zap.String("version", "1.0.0"),
		zap.Any("config", cfg),
	)

	// Inisialisasi custom file logger
	fileLogger, err := loggers.NewLogger(cfg)
	if err != nil {
		log.Fatalf("Gagal inisialisasi logger: %v", err)
	}
	defer func() {
		if err := fileLogger.Sync(); err != nil {
			log.Printf("Gagal sync log: %v", err)
		}
	}()

	// Create ticker for periodic log syncing
	syncTicker := time.NewTicker(1 * time.Second)
	defer syncTicker.Stop()

	go func() {
		for range syncTicker.C {
			if err := fileLogger.Sync(); err != nil {
				log.Printf("Error syncing logs: %v", err)
			}
		}
	}()

	fileLogger.Info("Aplikasi mulai berjalan",
		zap.String("version", "1.0.0"),
		zap.Any("config", cfg),
	)

	// Gabungkan kedua logger
	combinedLogger := zapLogger.With(zap.Namespace("file_logger"))
	//Initialize routes
	interfaces.RegisterRoutes(e, client, cfg, fileLogger)

	e.HTTPErrorHandler = interfaces.CreateErrorHandler(combinedLogger)
	// Tambahkan di middleware
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:     true,
		LogStatus:  true,
		LogMethod:  true,
		LogLatency: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			fileLogger.Info("request",
				zap.String("method", v.Method),
				zap.String("uri", v.URI),
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
			)
			// Force sync after each request log
			if err := fileLogger.Sync(); err != nil {
				log.Printf("Error syncing request log: %v", err)
			}
			return nil
		},
	}))

	// Start server
	fmt.Println("Server running on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}
