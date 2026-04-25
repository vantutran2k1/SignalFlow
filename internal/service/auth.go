package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo  domain.UserRepository
	jwtSecret string
	tokenTTL  time.Duration
}

func NewAuthService(userRepo domain.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		tokenTTL:  24 * time.Hour,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, name string) (*domain.User, error) {
	if email == "" || password == "" {
		return nil, fmt.Errorf("%w: email and password required", ErrInvalidInput)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(s.tokenTTL).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
