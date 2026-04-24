package service

import (
	"context"

	"caspercloud/internal/repository"

	"github.com/google/uuid"
)

// InstanceStore is the persistence surface required by InstanceService.
type InstanceStore interface {
	GetImage(ctx context.Context, projectID, imageID uuid.UUID) (*repository.Image, error)
	CreateInstance(ctx context.Context, projectID, imageID uuid.UUID, name, cloudInitData, initialState string) (*repository.Instance, error)
	CreateTask(ctx context.Context, taskType string, projectID, instanceID uuid.UUID, status string) (*repository.Task, error)
	UpdateTaskStatus(ctx context.Context, projectID, taskID uuid.UUID, status string, errMessage *string) error
	UpdateInstanceState(ctx context.Context, projectID, instanceID uuid.UUID, state string) error
	GetInstance(ctx context.Context, projectID, instanceID uuid.UUID) (*repository.Instance, error)
	ListInstances(ctx context.Context, projectID uuid.UUID) ([]repository.Instance, error)
	DeleteInstance(ctx context.Context, projectID, instanceID uuid.UUID) error
}

var _ InstanceStore = (*repository.Repository)(nil)
