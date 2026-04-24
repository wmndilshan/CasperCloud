package service

import (
	"context"
	"errors"
	"time"

	"caspercloud/internal/auth"
	"caspercloud/internal/repository"

	"github.com/google/uuid"
)

type AuthService struct {
	repo *repository.Repository
	jwt  *auth.JWTManager
}

func NewAuthService(repo *repository.Repository, jwt *auth.JWTManager) *AuthService {
	return &AuthService{repo: repo, jwt: jwt}
}

type AuthResponse struct {
	Token            string            `json:"token"`
	User             repository.User   `json:"user"`
	ActiveProjectID  *uuid.UUID        `json:"active_project_id,omitempty"`
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
	token, err := s.jwt.Generate(user.ID, user.Email, nil, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: *user}, nil
}

type LoginInput struct {
	Email     string
	Password  string
	ProjectID *uuid.UUID
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}
	if err := auth.CheckPassword(user.PasswordHash, in.Password); err != nil {
		return nil, errors.New("invalid credentials")
	}
	var scoped *uuid.UUID
	if in.ProjectID != nil && *in.ProjectID != uuid.Nil {
		ok, err := s.repo.UserHasProjectAccess(ctx, user.ID, *in.ProjectID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("user is not a member of the requested project")
		}
		scoped = in.ProjectID
	}
	token, err := s.jwt.Generate(user.ID, user.Email, scoped, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	resp := &AuthResponse{Token: token, User: *user}
	if scoped != nil {
		resp.ActiveProjectID = scoped
	}
	return resp, nil
}

func (s *AuthService) SwitchProject(ctx context.Context, userID, projectID uuid.UUID) (*AuthResponse, error) {
	ok, err := s.repo.UserHasProjectAccess(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("user is not a member of the requested project")
	}
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	token, err := s.jwt.Generate(user.ID, user.Email, &projectID, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: *user, ActiveProjectID: &projectID}, nil
}
