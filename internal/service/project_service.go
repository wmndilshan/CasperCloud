package service

import (
	"context"

	"caspercloud/internal/repository"
	"github.com/google/uuid"
)

type ProjectService struct {
	repo *repository.Repository
}

func NewProjectService(repo *repository.Repository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) CreateProject(ctx context.Context, userID uuid.UUID, name string) (*repository.Project, error) {
	return s.repo.CreateProject(ctx, userID, name)
}

func (s *ProjectService) ListProjects(ctx context.Context, userID uuid.UUID) ([]repository.Project, error) {
	return s.repo.ListProjectsForUser(ctx, userID)
}

func (s *ProjectService) EnsureProjectAccess(ctx context.Context, userID, projectID uuid.UUID) error {
	hasAccess, err := s.repo.UserHasProjectAccess(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if !hasAccess {
		return repository.ErrNotFound
	}
	return nil
}
