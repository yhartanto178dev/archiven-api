package application

import (
	"context"
	"fmt"

	"github.com/yhartanto178dev/archiven-api/internal/auth/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo         domain.UserRepository
	refreshTokenRepo domain.RefreshTokenRepository
	jwtService       JWTServiceInterface
}

type JWTServiceInterface interface {
	GenerateTokenPair(user *domain.User, deviceID, userAgent string) (*domain.LoginResponse, error)
	ValidateAccessToken(token string) (*domain.Claims, error)
	RefreshTokens(refreshToken string) (*domain.LoginResponse, error)
	RevokeToken(refreshToken string) error
	RevokeAllUserTokens(userID primitive.ObjectID) error
}

func NewAuthService(
	userRepo domain.UserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtService JWTServiceInterface,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
	}
}

func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest, userAgent string) (*domain.LoginResponse, error) {
	// Find user by username or email
	var user *domain.User
	var err error

	if isEmail(req.Username) {
		user, err = s.userRepo.FindByEmail(req.Username)
	} else {
		user, err = s.userRepo.FindByUsername(req.Username)
	}

	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	// Check if user is active
	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidPassword
	}

	// Generate token pair
	return s.jwtService.GenerateTokenPair(user, req.DeviceID, userAgent)
}

func (s *AuthService) RefreshToken(ctx context.Context, req domain.RefreshRequest) (*domain.LoginResponse, error) {
	return s.jwtService.RefreshTokens(req.RefreshToken)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.jwtService.RevokeToken(refreshToken)
}

func (s *AuthService) LogoutAll(ctx context.Context, userID primitive.ObjectID) error {
	return s.jwtService.RevokeAllUserTokens(userID)
}

func (s *AuthService) ValidateToken(ctx context.Context, token string) (*domain.Claims, error) {
	return s.jwtService.ValidateAccessToken(token)
}

func (s *AuthService) GetUser(ctx context.Context, userID primitive.ObjectID) (*domain.User, error) {
	return s.userRepo.FindByID(userID)
}

func (s *AuthService) CreateUser(ctx context.Context, user *domain.User, password string) error {
	// Check if user already exists
	if _, err := s.userRepo.FindByUsername(user.Username); err == nil {
		return domain.ErrUserAlreadyExists
	}

	if _, err := s.userRepo.FindByEmail(user.Email); err == nil {
		return domain.ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	user.Password = string(hashedPassword)
	return s.userRepo.Create(user)
}

func isEmail(str string) bool {
	// Simple email validation
	return len(str) > 0 && str[0] != '@' && len(str) > 3 &&
		len(str) < 254 && containsAtSign(str)
}

func containsAtSign(str string) bool {
	for _, char := range str {
		if char == '@' {
			return true
		}
	}
	return false
}
