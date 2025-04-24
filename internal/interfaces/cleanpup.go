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
			if err := service.CleanupTempFiles(ctx); err != nil {
				logger.Error("Failed to cleanup temp files",
					zap.Error(err),
				)
			} else {
				logger.Info("Successfully cleaned up expired temporary files")
			}
			cancel()
		}
	}()
}
