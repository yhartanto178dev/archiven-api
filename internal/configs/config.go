package configs

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port             string
	ServerPort       int
	MongoURI         string
	DatabaseName     string
	DBName           string
	BucketName       string
	UploadDir        string
	Host             string
	AllowedTypes     []string
	AllowedOrigins   []string
	MaxUploadSize    int64
	LogDir           string
	LogFileFormat    string
	LogRetentionDays int
	LogLevel         string
}

func Load() *Config {
	allowedTypes := strings.Split(getEnvString("ALLOWED_TYPES", "application/pdf"), ",")
	allowedOrigins := strings.Split(getEnvString("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080"), ",")
	port := getEnvString("PORT", "8080")
	return &Config{
		Port:             port,
		ServerPort:       getEnvInt("SERVER_PORT", 8080),
		MongoURI:         getEnvString("MONGODB_URI", "mongodb://localhost:27017"),
		DatabaseName:     getEnvString("DATABASE_NAME", "archive_db"),
		DBName:           getEnvString("DB_NAME", "archive_db"),
		Host:             getEnvString("HOST", "localhost"),
		AllowedTypes:     allowedTypes,
		AllowedOrigins:   allowedOrigins,
		MaxUploadSize:    int64(getEnvInt("MAX_UPLOAD_SIZE", 3145728)), // 3 MB
		LogDir:           getEnvString("LOG_DIR", "logs"),
		LogFileFormat:    getEnvString("LOG_FILE_FORMAT", "2006-01-02.log"),
		LogRetentionDays: getEnvInt("LOG_RETENTION_DAYS", 7),
		LogLevel:         getEnvString("LOG_LEVEL", "info"),
	}
}

// LoadConfig is an alias for Load to maintain compatibility
func LoadConfig() *Config {
	return Load()
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
