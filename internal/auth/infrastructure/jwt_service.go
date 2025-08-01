package infrastructure

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yhartanto178dev/archiven-api/internal/auth/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type JWTService struct {
	privateKey       *rsa.PrivateKey
	publicKey        *rsa.PublicKey
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	issuer           string
	refreshTokenRepo domain.RefreshTokenRepository
}

type JWTConfig struct {
	PrivateKeyPath   string
	PublicKeyPath    string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	Issuer           string
	RefreshTokenRepo domain.RefreshTokenRepository
}

func NewJWTService(config JWTConfig) (*JWTService, error) {
	privateKey, err := loadPrivateKey(config.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %v", err)
	}

	publicKey, err := loadPublicKey(config.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %v", err)
	}

	return &JWTService{
		privateKey:       privateKey,
		publicKey:        publicKey,
		accessTokenTTL:   config.AccessTokenTTL,
		refreshTokenTTL:  config.RefreshTokenTTL,
		issuer:           config.Issuer,
		refreshTokenRepo: config.RefreshTokenRepo,
	}, nil
}

func (j *JWTService) GenerateTokenPair(user *domain.User, deviceID, userAgent string) (*domain.LoginResponse, error) {
	tokenID := primitive.NewObjectID().Hex()
	now := time.Now()

	// Generate access token
	accessClaims := jwt.MapClaims{
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
		"token_id": tokenID,
		"type":     "access",
		"iss":      j.issuer,
		"iat":      now.Unix(),
		"exp":      now.Add(j.accessTokenTTL).Unix(),
		"nbf":      now.Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(j.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %v", err)
	}

	// Generate refresh token
	refreshClaims := jwt.MapClaims{
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
		"token_id": tokenID,
		"type":     "refresh",
		"iss":      j.issuer,
		"iat":      now.Unix(),
		"exp":      now.Add(j.refreshTokenTTL).Unix(),
		"nbf":      now.Unix(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(j.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %v", err)
	}

	// Store refresh token in database
	refreshTokenRecord := &domain.RefreshToken{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		Token:     refreshTokenString,
		ExpiresAt: now.Add(j.refreshTokenTTL),
		CreatedAt: now,
		IsRevoked: false,
		DeviceID:  deviceID,
		UserAgent: userAgent,
	}

	if err := j.refreshTokenRepo.Store(refreshTokenRecord); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %v", err)
	}

	return &domain.LoginResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(j.accessTokenTTL.Seconds()),
		TokenType:    "Bearer",
		User:         *user,
	}, nil
}

func (j *JWTService) ValidateAccessToken(tokenString string) (*domain.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.publicKey, nil
	})

	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	if !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrInvalidToken
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		return nil, domain.ErrInvalidToken
	}

	return &domain.Claims{
		UserID:   claims["user_id"].(string),
		Username: claims["username"].(string),
		Email:    claims["email"].(string),
		Role:     claims["role"].(string),
		TokenID:  claims["token_id"].(string),
		Type:     tokenType,
	}, nil
}

func (j *JWTService) RefreshTokens(refreshTokenString string) (*domain.LoginResponse, error) {
	// Validate refresh token
	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.publicKey, nil
	})

	if err != nil || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrInvalidToken
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return nil, domain.ErrInvalidToken
	}

	// Check if token exists and is not revoked
	refreshTokenRecord, err := j.refreshTokenRepo.FindByToken(refreshTokenString)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	if refreshTokenRecord.IsRevoked {
		return nil, domain.ErrTokenRevoked
	}

	if time.Now().After(refreshTokenRecord.ExpiresAt) {
		return nil, domain.ErrTokenExpired
	}

	// Get user ID and generate new token pair
	userID, err := primitive.ObjectIDFromHex(claims["user_id"].(string))
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Extract user information from refresh token claims
	username, _ := claims["username"].(string)
	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)

	// Create user object from token claims
	user := &domain.User{
		ID:       userID,
		Username: username,
		Email:    email,
		Role:     role,
	}

	// Revoke old refresh token
	if err := j.refreshTokenRepo.RevokeToken(refreshTokenString); err != nil {
		return nil, fmt.Errorf("failed to revoke old token: %v", err)
	}

	// Generate new token pair
	return j.GenerateTokenPair(user, refreshTokenRecord.DeviceID, refreshTokenRecord.UserAgent)
}

func (j *JWTService) RevokeToken(refreshTokenString string) error {
	return j.refreshTokenRepo.RevokeToken(refreshTokenString)
}

func (j *JWTService) RevokeAllUserTokens(userID primitive.ObjectID) error {
	return j.refreshTokenRepo.RevokeAllUserTokens(userID)
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	// Try to load from file first
	if _, err := os.Stat(path); err == nil {
		keyData, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		block, _ := pem.Decode(keyData)
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block")
		}

		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}

	// Generate new key if file doesn't exist
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Save the generated key
	keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})

	if err := os.WriteFile(path, keyPEM, 0600); err != nil {
		return nil, err
	}

	return privateKey, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	// Try to load from file first
	if _, err := os.Stat(path); err == nil {
		keyData, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		block, _ := pem.Decode(keyData)
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block")
		}

		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		return pub.(*rsa.PublicKey), nil
	}

	// If public key doesn't exist, try to derive it from private key
	privateKeyPath := strings.Replace(path, "public.pem", "private.pem", 1)
	if privateKey, err := loadPrivateKey(privateKeyPath); err == nil {
		publicKey := &privateKey.PublicKey

		// Save the public key for future use
		pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			return nil, err
		}

		pubKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubKeyBytes,
		})

		if err := os.WriteFile(path, pubKeyPEM, 0644); err != nil {
			return nil, err
		}

		return publicKey, nil
	}

	return nil, fmt.Errorf("public key file not found and cannot derive from private key: %s", path)
}
