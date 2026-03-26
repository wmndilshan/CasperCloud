package service

import (
	"context"

	"caspercloud/internal/repository"
	"github.com/google/uuid"
)

type ImageService struct {
	repo *repository.Repository
}

func NewImageService(repo *repository.Repository) *ImageService {
	return &ImageService{repo: repo}
}

func (s *ImageService) Create(ctx context.Context, projectID uuid.UUID, name, sourceURL, description string) (*repository.Image, error) {
	return s.repo.CreateImage(ctx, projectID, name, sourceURL, description)
}

func (s *ImageService) Get(ctx context.Context, projectID, imageID uuid.UUID) (*repository.Image, error) {
	return s.repo.GetImage(ctx, projectID, imageID)
}

func (s *ImageService) List(ctx context.Context, projectID uuid.UUID) ([]repository.Image, error) {
	return s.repo.ListImages(ctx, projectID)
}

func (s *ImageService) Update(ctx context.Context, projectID, imageID uuid.UUID, name, sourceURL, description string) (*repository.Image, error) {
	return s.repo.UpdateImage(ctx, projectID, imageID, name, sourceURL, description)
}

func (s *ImageService) Delete(ctx context.Context, projectID, imageID uuid.UUID) error {
	return s.repo.DeleteImage(ctx, projectID, imageID)
}
