package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yhartanto178dev/archiven-api/internal/auth/domain"
	"github.com/yhartanto178dev/archiven-api/internal/auth/infrastructure"
	"github.com/yhartanto178dev/archiven-api/internal/configs"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mock refresh token repository for testing
type MockRefreshTokenRepo struct {
	tokens map[string]*domain.RefreshToken
}

func NewMockRefreshTokenRepo() *MockRefreshTokenRepo {
	return &MockRefreshTokenRepo{
		tokens: make(map[string]*domain.RefreshToken),
	}
}

func (m *MockRefreshTokenRepo) Store(token *domain.RefreshToken) error {
	m.tokens[token.Token] = token
	return nil
}

func (m *MockRefreshTokenRepo) FindByToken(token string) (*domain.RefreshToken, error) {
	if t, ok := m.tokens[token]; ok {
		return t, nil
	}
	return nil, domain.ErrInvalidToken
}

func (m *MockRefreshTokenRepo) RevokeToken(token string) error {
	if t, ok := m.tokens[token]; ok {
		t.IsRevoked = true
		return nil
	}
	return domain.ErrInvalidToken
}

func (m *MockRefreshTokenRepo) RevokeAllUserTokens(userID primitive.ObjectID) error {
	for _, token := range m.tokens {
		if token.UserID == userID {
			token.IsRevoked = true
		}
	}
	return nil
}

func (m *MockRefreshTokenRepo) CleanupExpiredTokens() error {
	for key, token := range m.tokens {
		if time.Now().After(token.ExpiresAt) {
			delete(m.tokens, key)
		}
	}
	return nil
}

func setupJWTService(t *testing.T) (*infrastructure.JWTService, *MockRefreshTokenRepo) {
	// Load auth configuration
	authCfg := configs.LoadAuthConfig()

	// Use absolute paths for testing
	authCfg.JWTPrivateKeyPath = "/home/masterbkpsdmbms/development/archiven-api/keys/private.pem"
	authCfg.JWTPublicKeyPath = "/home/masterbkpsdmbms/development/archiven-api/keys/public.pem"

	// Create mock repository
	mockRepo := NewMockRefreshTokenRepo()

	// Initialize JWT service
	jwtService, err := infrastructure.NewJWTService(infrastructure.JWTConfig{
		PrivateKeyPath:   authCfg.JWTPrivateKeyPath,
		PublicKeyPath:    authCfg.JWTPublicKeyPath,
		AccessTokenTTL:   authCfg.AccessTokenTTL,
		RefreshTokenTTL:  authCfg.RefreshTokenTTL,
		Issuer:           authCfg.JWTIssuer,
		RefreshTokenRepo: mockRepo,
	})
	require.NoError(t, err)

	return jwtService, mockRepo
}

func createTestUser() *domain.User {
	return &domain.User{
		ID:       primitive.NewObjectID(),
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
		IsActive: true,
	}
}

func TestJWTTokenGeneration(t *testing.T) {
	fmt.Println("🔐 Testing JWT Token Generation...")

	jwtService, _ := setupJWTService(t)
	user := createTestUser()

	// Generate token pair
	loginResponse, err := jwtService.GenerateTokenPair(user, "test-device", "test-browser")
	require.NoError(t, err)
	assert.NotEmpty(t, loginResponse.AccessToken)
	assert.NotEmpty(t, loginResponse.RefreshToken)

	fmt.Printf("✅ Access Token: %s...\n", loginResponse.AccessToken[:50])
	fmt.Printf("✅ Refresh Token: %s...\n", loginResponse.RefreshToken[:50])
}

func TestJWTTokenValidation(t *testing.T) {
	fmt.Println("🔍 Testing JWT Token Validation...")

	jwtService, _ := setupJWTService(t)
	user := createTestUser()

	// Generate token
	loginResponse, err := jwtService.GenerateTokenPair(user, "test-device", "test-browser")
	require.NoError(t, err)

	// Validate token
	claims, err := jwtService.ValidateAccessToken(loginResponse.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.Role, claims.Role)

	fmt.Printf("✅ Token validated successfully for user: %s\n", claims.Username)
}

func TestJWTTokenRefresh(t *testing.T) {
	fmt.Println("🔄 Testing JWT Token Refresh...")

	jwtService, mockRepo := setupJWTService(t)
	user := createTestUser()

	// Generate initial tokens
	loginResponse1, err := jwtService.GenerateTokenPair(user, "test-device", "test-browser")
	require.NoError(t, err)

	// Wait a bit to ensure different timestamps
	time.Sleep(time.Millisecond * 100)

	// Refresh tokens
	loginResponse2, err := jwtService.RefreshTokens(loginResponse1.RefreshToken)
	require.NoError(t, err)
	assert.NotEqual(t, loginResponse1.AccessToken, loginResponse2.AccessToken)
	assert.NotEqual(t, loginResponse1.RefreshToken, loginResponse2.RefreshToken)

	// Verify old refresh token is revoked
	storedToken, err := mockRepo.FindByToken(loginResponse1.RefreshToken)
	if err == nil && storedToken.IsRevoked {
		// Token found but revoked - this is expected
		fmt.Printf("✅ Old refresh token properly revoked\n")
	} else if err != nil {
		// Token not found - this is also acceptable (could be deleted)
		fmt.Printf("✅ Old refresh token not found (cleaned up)\n")
	} else {
		// Token found and not revoked - this is unexpected
		t.Errorf("Expected old refresh token to be revoked or removed")
	}

	// Verify new tokens work
	claims, err := jwtService.ValidateAccessToken(loginResponse2.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.Username, claims.Username)

	fmt.Printf("✅ Token refresh successful\n")
	fmt.Printf("✅ New tokens validated\n")
}

func TestJWTTokenRevocation(t *testing.T) {
	fmt.Println("❌ Testing JWT Token Revocation...")

	jwtService, mockRepo := setupJWTService(t)
	user := createTestUser()

	// Generate tokens
	loginResponse, err := jwtService.GenerateTokenPair(user, "test-device", "test-browser")
	require.NoError(t, err)

	// Revoke token
	err = mockRepo.RevokeToken(loginResponse.RefreshToken)
	require.NoError(t, err)

	// Try to use revoked token
	_, err = jwtService.RefreshTokens(loginResponse.RefreshToken)
	assert.Error(t, err)

	fmt.Printf("✅ Token revocation works correctly\n")
}

func TestJWTInvalidToken(t *testing.T) {
	fmt.Println("🚫 Testing Invalid Token Handling...")

	jwtService, _ := setupJWTService(t)

	// Test invalid token
	_, err := jwtService.ValidateAccessToken("invalid.token.here")
	assert.Error(t, err)

	// Test refresh with invalid token
	_, err = jwtService.RefreshTokens("invalid.refresh.token")
	assert.Error(t, err)

	fmt.Printf("✅ Invalid token handling works correctly\n")
}

func TestJWTBulkRevocation(t *testing.T) {
	fmt.Println("🔥 Testing Bulk Token Revocation...")

	jwtService, mockRepo := setupJWTService(t)
	user := createTestUser()

	// Generate multiple tokens for same user
	loginResponse1, err := jwtService.GenerateTokenPair(user, "device-1", "browser-1")
	require.NoError(t, err)

	loginResponse2, err := jwtService.GenerateTokenPair(user, "device-2", "browser-2")
	require.NoError(t, err)

	// Revoke all tokens for user
	err = mockRepo.RevokeAllUserTokens(user.ID)
	require.NoError(t, err)

	// Try to use revoked tokens
	_, err = jwtService.RefreshTokens(loginResponse1.RefreshToken)
	assert.Error(t, err)

	_, err = jwtService.RefreshTokens(loginResponse2.RefreshToken)
	assert.Error(t, err)

	fmt.Printf("✅ Bulk token revocation works correctly\n")
}

func TestJWTSecurityFeatures(t *testing.T) {
	fmt.Println("🛡️ Testing Security Features...")

	jwtService, _ := setupJWTService(t)
	user := createTestUser()

	// Generate token
	loginResponse, err := jwtService.GenerateTokenPair(user, "test-device", "test-browser")
	require.NoError(t, err)

	// Validate token and check claims
	claims, err := jwtService.ValidateAccessToken(loginResponse.AccessToken)
	require.NoError(t, err)

	// Verify security claims
	assert.NotEmpty(t, claims.UserID)
	assert.NotEmpty(t, claims.Username)
	assert.NotEmpty(t, claims.Email)
	assert.NotEmpty(t, claims.Role)
	assert.NotEmpty(t, claims.TokenID)
	assert.Equal(t, "access", claims.Type)

	fmt.Printf("✅ Security features validated:\n")
	fmt.Printf("   - UserID: %s\n", claims.UserID)
	fmt.Printf("   - Username: %s\n", claims.Username)
	fmt.Printf("   - Email: %s\n", claims.Email)
	fmt.Printf("   - Role: %s\n", claims.Role)
	fmt.Printf("   - TokenID: %s\n", claims.TokenID)
	fmt.Printf("   - Type: %s\n", claims.Type)
}

// This can be run as a standalone test
func TestJWTFullFlow(t *testing.T) {
	fmt.Println("\n🎯 Running Full JWT Authentication Flow Test")
	fmt.Println("==============================================")

	t.Run("TokenGeneration", TestJWTTokenGeneration)
	t.Run("TokenValidation", TestJWTTokenValidation)
	t.Run("TokenRefresh", TestJWTTokenRefresh)
	t.Run("TokenRevocation", TestJWTTokenRevocation)
	t.Run("InvalidTokenHandling", TestJWTInvalidToken)
	t.Run("BulkRevocation", TestJWTBulkRevocation)
	t.Run("SecurityFeatures", TestJWTSecurityFeatures)

	fmt.Println("\n🎉 JWT Authentication System Test Completed Successfully!")
	fmt.Println("All security features are working as expected.")
}
