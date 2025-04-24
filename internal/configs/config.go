package configs

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerPort       int
	MongoURI         string
	DBName           string
	BucketName       string
	UploadDir        string
	Host             string
	AllowedTypes     []string
	MaxUploadSize    int64
	LogDir           string
	LogFileFormat    string
	LogRetentionDays int
	LogLevel         string
}

func Load() *Config {
	allowedTypes := strings.Split(getEnvString("ALLOWED_TYPES", "application/pdf"), ",")
	return &Config{
		ServerPort:       getEnvInt("SERVER_PORT", 8080),
		MongoURI:         getEnvString("MONGODB_URI", "mongodb://localhost:27017"),
		DBName:           getEnvString("DB_NAME", "archive_db"),
		Host:             getEnvString("HOST", "localhost"),
		AllowedTypes:     allowedTypes,
		MaxUploadSize:    int64(getEnvInt("MAX_UPLOAD_SIZE", 3145728)), // 3 MB
		LogDir:           getEnvString("LOG_DIR", "logs"),
		LogFileFormat:    getEnvString("LOG_FILE_FORMAT", "2006-01-02.log"),
		LogRetentionDays: getEnvInt("LOG_RETENTION_DAYS", 7),
		LogLevel:         getEnvString("LOG_LEVEL", "info"),
	}
}

func getEnvString(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
