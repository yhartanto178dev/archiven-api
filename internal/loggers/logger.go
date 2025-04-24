package loggers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yhartanto178dev/archiven-api/internal/configs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger(cfg *configs.Config) (*zap.Logger, error) {
	// 1. Pastikan direktori log ada
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("gagal membuat direktori log: %w", err)
	}

	// 2. Setup log rotation
	logFile := filepath.Join(cfg.LogDir, time.Now().Format(cfg.LogFileFormat))

	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // MB
		MaxBackups: cfg.LogRetentionDays,
		MaxAge:     cfg.LogRetentionDays,
		Compress:   true,
		LocalTime:  true,
	}

	// 3. Test penulisan log
	if _, err := lumberjackLogger.Write([]byte("Initial log entry\n")); err != nil {
		return nil, fmt.Errorf("gagal menulis log awal: %w", err)
	}

	// 4. Konfigurasi encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "timestamp"

	// 5. Set level log
	level := zap.InfoLevel
	switch cfg.LogLevel {
	case "debug":
		level = zap.DebugLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	}

	// 6. Membuat konfigurasi logger
	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       true,
		Encoding:          "json",
		OutputPaths:       []string{logFile},
		ErrorOutputPaths:  []string{logFile},
		EncoderConfig:     encoderConfig,
		DisableStacktrace: false,
		Sampling:          nil, // Disable sampling to ensure all logs are written
	}

	// 7. Membuat logger
	logger, err := config.Build(
		zap.WithCaller(true),
		zap.AddCallerSkip(1),
	)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat logger: %w", err)
	}

	// Force syncing after each write
	logger = logger.WithOptions(zap.Development())

	// 8. Test logging
	logger.Info("Logger berhasil diinisialisasi",
		zap.String("log_file", logFile),
		zap.String("log_dir", cfg.LogDir),
	)

	return logger, nil
}
