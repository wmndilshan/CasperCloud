package service

import (
	"context"

	"caspercloud/internal/repository"

	"github.com/google/uuid"
)

// InstanceStore is the persistence surface required by InstanceService.
type InstanceStore interface {
	GetImage(ctx context.Context, projectID, imageID uuid.UUID) (*repository.Image, error)
	CreateInstanceWithIPAM(ctx context.Context, p repository.CreateInstanceParams) (*repository.Instance, error)
	CreateTask(ctx context.Context, taskType string, projectID, instanceID uuid.UUID, status string) (*repository.Task, error)
	UpdateTaskStatus(ctx context.Context, projectID, taskID uuid.UUID, status string, errMessage *string) error
	UpdateInstanceState(ctx context.Context, projectID, instanceID uuid.UUID, state string) error
	GetInstance(ctx context.Context, projectID, instanceID uuid.UUID) (*repository.Instance, error)
	ListInstances(ctx context.Context, projectID uuid.UUID) ([]repository.Instance, error)
	DeleteInstance(ctx context.Context, projectID, instanceID uuid.UUID) error
	GetNetworkForProject(ctx context.Context, projectID, networkID uuid.UUID) (*repository.Network, error)
	GetDefaultNetworkID(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error)
	ListInstancesForHypervisorSync(ctx context.Context) ([]repository.InstanceSyncRow, error)
	BatchUpdateInstanceStates(ctx context.Context, updates []repository.InstanceStateUpdate) error
	FinalizeInstanceDestroy(ctx context.Context, projectID, taskID, instanceID uuid.UUID) error
	CountVolumesAttachedToInstance(ctx context.Context, projectID, instanceID uuid.UUID) (int64, error)

	BeginSnapshotCreate(ctx context.Context, projectID, instanceID, snapshotID uuid.UUID, name string) (priorState string, err error)
	BeginSnapshotRevert(ctx context.Context, projectID, instanceID uuid.UUID) (priorState string, err error)
	UpdateSnapshotStatus(ctx context.Context, projectID, snapshotID uuid.UUID, status string) error
	ListSnapshotsForInstance(ctx context.Context, projectID, instanceID uuid.UUID) ([]repository.Snapshot, error)
	GetSnapshot(ctx context.Context, projectID, instanceID, snapshotID uuid.UUID) (*repository.Snapshot, error)

	ListActiveFloatingIPNATBindingsByInstance(ctx context.Context, projectID, instanceID uuid.UUID) ([]repository.FloatingIPNATBinding, error)
	ClearFloatingIPBindingsForInstance(ctx context.Context, projectID, instanceID uuid.UUID) error
}

var _ InstanceStore = (*repository.Repository)(nil)
