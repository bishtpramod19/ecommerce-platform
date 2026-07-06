package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/model"
	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/ports"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo      ports.UserRepository
	jwtSecret string
	jwtExpiry time.Duration
}

func NewUserService(repo ports.UserRepository, jwtSecret string, jwtExpiryHours int) *UserService {
	return &UserService{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtExpiry: time.Duration(jwtExpiryHours) * time.Hour,
	}
}

func (s *UserService) Register(ctx context.Context, req *model.RegisterRequest) (*model.AuthResponse, error) {
	existingUser, err := s.repo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, fmt.Errorf("email already registered")
	}
	if err != nil && err.Error() != "user not found" {
		return nil, fmt.Errorf("error checking email: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         "customer",
		IsActive:     true,
	}

	createdUser, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	accessToken, err := s.generateAccessToken(createdUser)
	if err != nil {
		return nil, fmt.Errorf("error generating token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(createdUser)
	if err != nil {
		return nil, fmt.Errorf("error generating refresh token: %w", err)
	}

	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *createdUser,
	}, nil
}

func (s *UserService) Login(ctx context.Context, req *model.LoginRequest) (*model.AuthResponse, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("error generating token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("error generating refresh token: %w", err)
	}

	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %w", err)
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id int64, req *model.UpdateUserRequest) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	user.FirstName = req.FirstName
	user.LastName = req.LastName

	updatedUser, err := s.repo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("error updating user: %w", err)
	}

	return updatedUser, nil
}

func (s *UserService) generateAccessToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"type":    "access",
		"exp":     time.Now().Add(s.jwtExpiry).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *UserService) generateRefreshToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"type":    "refresh",
		"exp":     time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}