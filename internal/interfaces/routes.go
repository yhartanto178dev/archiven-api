package interfaces

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/yhartanto178dev/archiven-api/internal/archive/application"
	"github.com/yhartanto178dev/archiven-api/internal/archive/infrastructure"
	"github.com/yhartanto178dev/archiven-api/internal/configs"
	"go.uber.org/zap"

	"go.mongodb.org/mongo-driver/mongo"
)

func RegisterRoutes(e *echo.Echo, client *mongo.Client, cfg *configs.Config, logger *zap.Logger) {
	// Initialize Repository
	repo, err := infrastructure.NewArchiveRepository(client, cfg.DBName)
	if err != nil {
		e.Logger.Fatal("Failed to initialize archive repository:", err)
	}

	// Initialize service
	service := application.NewArchiveService(repo)

	fileValidator := NewFileValidator(
		3*1024*1024, // 3MB
		[]string{"application/pdf"},
		".pdf",
		logger,
	)

	// Initialize handlers
	handler := NewArchiveHandler(service, fileValidator, logger)
	startCleanupTask(service, 1*time.Hour, logger)
	// Register routes
	// Routes
	e.POST("/archives", handler.Upload)
	e.GET("/archives", handler.List)
	e.GET("/download/:id", handler.Download)
	e.GET("/archives/list", handler.GetByIDs)
	e.DELETE("/archives/:id", handler.DeleteArchive)
	e.DELETE("/archives/:id/permanent", handler.DeleteArchive)
	e.POST("/archives/:id/restore", handler.RestoreArchive)
}
