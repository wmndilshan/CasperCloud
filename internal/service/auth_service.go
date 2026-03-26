package service

import (
	"context"
	"errors"
	"time"

	"caspercloud/internal/auth"
	"caspercloud/internal/repository"
)

type AuthService struct {
	repo *repository.Repository
	jwt  *auth.JWTManager
}

func NewAuthService(repo *repository.Repository, jwt *auth.JWTManager) *AuthService {
	return &AuthService{repo: repo, jwt: jwt}
}

type AuthResponse struct {
	Token string          `json:"token"`
	User  repository.User `json:"user"`
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*AuthResponse, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.CreateUser(ctx, email, hash)
	if err != nil {
		return nil, err
	}
	token, err := s.jwt.Generate(user.ID, user.Email, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: *user}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}
	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		return nil, errors.New("invalid credentials")
	}
	token, err := s.jwt.Generate(user.ID, user.Email, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: *user}, nil
}
