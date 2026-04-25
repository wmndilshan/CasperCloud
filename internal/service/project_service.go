package service

import (
	"context"

	"caspercloud/internal/repository"
	"github.com/google/uuid"
)

type ProjectService struct {
	repo              *repository.Repository
	defaultNetCIDR    string
	defaultNetGateway string
	defaultBridge     string
}

func NewProjectService(repo *repository.Repository, defaultNetCIDR, defaultNetGateway, defaultBridge string) *ProjectService {
	return &ProjectService{
		repo:              repo,
		defaultNetCIDR:    defaultNetCIDR,
		defaultNetGateway: defaultNetGateway,
		defaultBridge:     defaultBridge,
	}
}

func (s *ProjectService) CreateProject(ctx context.Context, userID uuid.UUID, name string) (*repository.Project, error) {
	return s.repo.CreateProject(ctx, userID, name, s.defaultNetCIDR, s.defaultNetGateway, s.defaultBridge)
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

// ListNetworks returns all networks for a project.
func (s *ProjectService) ListNetworks(ctx context.Context, projectID uuid.UUID) ([]repository.Network, error) {
	return s.repo.ListNetworksForProject(ctx, projectID)
}
