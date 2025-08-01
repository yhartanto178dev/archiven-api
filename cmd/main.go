package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	archiveApp "github.com/yhartanto178dev/archiven-api/internal/archive/application"
	archiveInfra "github.com/yhartanto178dev/archiven-api/internal/archive/infrastructure"
	archiveHandlers "github.com/yhartanto178dev/archiven-api/internal/interfaces"

	authApp "github.com/yhartanto178dev/archiven-api/internal/auth/application"
	authInfra "github.com/yhartanto178dev/archiven-api/internal/auth/infrastructure"
	authHandlers "github.com/yhartanto178dev/archiven-api/internal/auth/interfaces"

	"github.com/yhartanto178dev/archiven-api/internal/configs"
	"github.com/yhartanto178dev/archiven-api/internal/loggers"
	"go.uber.org/zap"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Error loading .env file, using system environment variables")
	}

	// Load configuration
	cfg := configs.LoadConfig()
	authCfg := configs.LoadAuthConfig()

	// Initialize logger
	fileLogger, err := loggers.NewLogger(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer fileLogger.Sync()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(cfg.MongoURI).
		SetMaxPoolSize(100).
		SetMinPoolSize(5).
		SetMaxConnecting(20)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		fileLogger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	defer client.Disconnect(ctx)

	// Test MongoDB connection
	if err := client.Ping(ctx, nil); err != nil {
		fileLogger.Fatal("Failed to ping MongoDB", zap.Error(err))
	}
	fileLogger.Info("Connected to MongoDB successfully")

	db := client.Database(cfg.DatabaseName)

	// Initialize repositories
	userRepo := authInfra.NewUserRepository(db)
	refreshTokenRepo := authInfra.NewRefreshTokenRepository(db)
	archiveRepo, err := archiveInfra.NewArchiveRepository(client, cfg.DatabaseName)
	if err != nil {
		fileLogger.Fatal("Failed to initialize archive repository", zap.Error(err))
	}

	// Initialize JWT service
	jwtService, err := authInfra.NewJWTService(authInfra.JWTConfig{
		PrivateKeyPath:   authCfg.JWTPrivateKeyPath,
		PublicKeyPath:    authCfg.JWTPublicKeyPath,
		AccessTokenTTL:   authCfg.AccessTokenTTL,
		RefreshTokenTTL:  authCfg.RefreshTokenTTL,
		Issuer:           authCfg.JWTIssuer,
		RefreshTokenRepo: refreshTokenRepo,
	})
	if err != nil {
		fileLogger.Fatal("Failed to initialize JWT service", zap.Error(err))
	}

	// Initialize services
	authService := authApp.NewAuthService(userRepo, refreshTokenRepo, jwtService)
	archiveService := archiveApp.NewArchiveService(archiveRepo)

	// Initialize file validator
	fileValidator := archiveHandlers.NewFileValidator(
		50*1024*1024, // 50MB max size
		[]string{"application/pdf"},
		".pdf",
		fileLogger,
	)

	// Initialize handlers
	authHandler := authHandlers.NewAuthHandler(authService)
	authMiddleware := authHandlers.NewAuthMiddleware(authService)
	archiveHandler := archiveHandlers.NewArchiveHandler(archiveService, fileValidator, fileLogger)

	// Initialize Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

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
			if err := fileLogger.Sync(); err != nil {
				log.Printf("Error syncing request log: %v", err)
			}
			return nil
		},
	}))

	e.Use(middleware.Recover())

	// Auth routes (public)
	authGroup := e.Group("/auth")
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.RefreshToken)
	authGroup.POST("/logout", authHandler.Logout)

	// Protected routes
	api := e.Group("/api/v1")
	api.Use(authMiddleware.JWTAuth())

	// Auth protected routes
	api.GET("/profile", authHandler.GetProfile)
	api.POST("/logout-all", authHandler.LogoutAll)

	// Archive routes
	archives := api.Group("/archives")
	archives.POST("", archiveHandler.Upload)
	archives.GET("", archiveHandler.List)
	archives.GET("/:id/download", archiveHandler.Download)
	archives.DELETE("/:id", archiveHandler.DeleteArchive)
	archives.DELETE("/:id/permanent", archiveHandler.DeleteArchive)
	archives.POST("/:id/restore", archiveHandler.RestoreArchive)
	archives.GET("/:id/history", archiveHandler.GetHistory)
	archives.GET("/category/:category", archiveHandler.GetByCategory)
	archives.GET("/tags", archiveHandler.GetByTags)
	archives.POST("/bulk", archiveHandler.GetByIDs)

	// Swagger documentation routes
	e.Static("/docs", "docs")
	e.File("/swagger.yaml", "swagger.yaml")
	e.GET("/swagger", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/swagger.html")
	})

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
		})
	})

	// Start server
	fileLogger.Info("Starting server", zap.String("port", cfg.Port))

	// Graceful shutdown
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			fileLogger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fileLogger.Info("Shutting down server...")
	if err := e.Shutdown(ctx); err != nil {
		fileLogger.Fatal("Failed to shutdown server", zap.Error(err))
	}
}
