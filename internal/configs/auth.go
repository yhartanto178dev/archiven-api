package configs

import (
	"os"
	"strconv"
	"time"
)

type AuthConfig struct {
	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	JWTIssuer         string
}

func LoadAuthConfig() *AuthConfig {
	accessTokenTTL := 15 * time.Minute // Default 15 minutes
	if ttl := os.Getenv("ACCESS_TOKEN_TTL"); ttl != "" {
		if duration, err := time.ParseDuration(ttl); err == nil {
			accessTokenTTL = duration
		}
	}

	refreshTokenTTL := 7 * 24 * time.Hour // Default 7 days
	if ttl := os.Getenv("REFRESH_TOKEN_TTL"); ttl != "" {
		if duration, err := time.ParseDuration(ttl); err == nil {
			refreshTokenTTL = duration
		}
	}

	return &AuthConfig{
		JWTPrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", "./keys/private.pem"),
		JWTPublicKeyPath:  getEnv("JWT_PUBLIC_KEY_PATH", "./keys/public.pem"),
		AccessTokenTTL:    accessTokenTTL,
		RefreshTokenTTL:   refreshTokenTTL,
		JWTIssuer:         getEnv("JWT_ISSUER", "archiven-api"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
