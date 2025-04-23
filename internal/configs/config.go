package configs

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerPort    int
	MongoURI      string
	DBName        string
	BucketName    string
	UploadDir     string
	Host          string
	AllowedTypes  []string
	MaxUploadSize int64
}

func Load() *Config {
	allowedTypes := strings.Split(getEnvString("ALLOWED_TYPES", "application/pdf"), ",")
	return &Config{
		ServerPort:    getEnvInt("SERVER_PORT", 8080),
		MongoURI:      getEnvString("MONGODB_URI", "mongodb://localhost:27017"),
		DBName:        getEnvString("DB_NAME", "archive_db"),
		Host:          getEnvString("HOST", "localhost"),
		AllowedTypes:  allowedTypes,
		MaxUploadSize: int64(getEnvInt("MAX_UPLOAD_SIZE", 3145728)), // 10 MB
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
