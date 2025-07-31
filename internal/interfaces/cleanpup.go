package interfaces

import (
	"context"
	"time"

	"github.com/yhartanto178dev/archiven-api/internal/archive/application"
	"go.uber.org/zap"
)

func startCleanupTask(service *application.ArchiveService, interval time.Duration, logger *zap.Logger) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

			count, err := service.CleanupExpiredFiles(ctx)
			if err != nil {
				logger.Error("Failed to cleanup expired files",
					zap.Error(err),
					zap.String("task", "cleanup"),
					zap.Time("timestamp", time.Now()),
				)
			} else {
				logger.Info("Successfully cleaned up expired files",
					zap.Int64("files_removed", count),
					zap.String("task", "cleanup"),
					zap.Time("timestamp", time.Now()),
				)
			}

			// Cleanup temporary files separately
			tempCount, err := service.CleanupTempFiles(ctx)
			if err != nil {
				logger.Error("Failed to cleanup temporary files",
					zap.Error(err),
					zap.String("task", "temp_cleanup"),
					zap.Time("timestamp", time.Now()),
				)
			} else {
				logger.Info("Successfully cleaned up temporary files",
					zap.Int64("temp_files_removed", tempCount),
					zap.String("task", "temp_cleanup"),
					zap.Time("timestamp", time.Now()),
				)
			}

			cancel()
		}
	}()
}
