package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"caspercloud/internal/libvirt"
	"caspercloud/internal/repository"
	"caspercloud/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	VolumeStatusAvailable = "available"
	VolumeStatusInUse     = "in-use"
	VolumeStatusError     = "error"
)

var (
	ErrInstanceBusySnapshotting    = errors.New("instance is busy with a snapshot operation")
	ErrInstanceNotRunning          = errors.New("instance must be running to hot-attach a volume")
	ErrVolumeNotAvailable          = errors.New("volume is not available for attach")
	ErrVolumeNotAttachedToInstance = errors.New("volume is not attached to this instance")
	ErrInstanceHasAttachedVolumes  = errors.New("detach all volumes before deleting or destroying the instance")
	ErrVolumeNameConflict          = errors.New("volume name already exists in this project")
)

// VolumeService manages persistent volumes and libvirt attach/detach.
type VolumeService struct {
	repo    *repository.Repository
	libvirt libvirt.Adapter
	qemuImg string
	volDir  string
}

func NewVolumeService(repo *repository.Repository, lv libvirt.Adapter, qemuImgPath, volumesDir string) *VolumeService {
	return &VolumeService{repo: repo, libvirt: lv, qemuImg: qemuImgPath, volDir: volumesDir}
}

func (s *VolumeService) CreateVolume(ctx context.Context, projectID uuid.UUID, name string, sizeGB int) (*repository.Volume, error) {
	id := uuid.New()
	vol, err := s.repo.CreateVolume(ctx, projectID, id, name, sizeGB, VolumeStatusAvailable)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrVolumeNameConflict
		}
		return nil, err
	}
	path := storage.VolumeQCOW2Path(s.volDir, id)
	if err := storage.CreateSparseQCOW2(ctx, s.qemuImg, path, sizeGB); err != nil {
		_ = s.repo.SetVolumeError(ctx, projectID, id)
		_ = os.Remove(path)
		return nil, err
	}
	return vol, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (s *VolumeService) ListVolumes(ctx context.Context, projectID uuid.UUID) ([]repository.Volume, error) {
	return s.repo.ListVolumes(ctx, projectID)
}

func (s *VolumeService) GetVolume(ctx context.Context, projectID, volumeID uuid.UUID) (*repository.Volume, error) {
	return s.repo.GetVolume(ctx, projectID, volumeID)
}

func (s *VolumeService) DeleteVolume(ctx context.Context, projectID, volumeID uuid.UUID) error {
	if err := s.repo.DeleteVolume(ctx, projectID, volumeID); err != nil {
		return err
	}
	path := storage.VolumeQCOW2Path(s.volDir, volumeID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("caspercloud: remove volume file %q: %v", path, err)
	}
	return nil
}

// AttachVolume hot-attaches a volume to a running instance.
func (s *VolumeService) AttachVolume(ctx context.Context, projectID, instanceID, volumeID uuid.UUID) error {
	inst, err := s.repo.GetInstance(ctx, projectID, instanceID)
	if err != nil {
		return err
	}
	if inst.State == InstanceStateSnapshotting {
		return ErrInstanceBusySnapshotting
	}
	if inst.State != InstanceStateRunning {
		return ErrInstanceNotRunning
	}
	vol, err := s.repo.GetVolume(ctx, projectID, volumeID)
	if err != nil {
		return err
	}
	if vol.Status != VolumeStatusAvailable || vol.InstanceID != nil {
		return ErrVolumeNotAvailable
	}

	target, err := s.libvirt.NextVirtioDiskTarget(ctx, inst.Name)
	if err != nil {
		return fmt.Errorf("next disk target: %w", err)
	}
	path := storage.VolumeQCOW2Path(s.volDir, volumeID)

	if err := s.libvirt.AttachVolume(ctx, inst.Name, path, target, true); err != nil {
		return fmt.Errorf("libvirt attach: %w", err)
	}
	if err := s.repo.AttachVolumeRecord(ctx, projectID, volumeID, instanceID, target); err != nil {
		_ = s.libvirt.DetachVolume(ctx, inst.Name, path, target, true)
		return fmt.Errorf("record attach: %w", err)
	}
	return nil
}

// DetachVolume removes a volume from a running or stopped instance domain configuration.
func (s *VolumeService) DetachVolume(ctx context.Context, projectID, instanceID, volumeID uuid.UUID) error {
	inst, err := s.repo.GetInstance(ctx, projectID, instanceID)
	if err != nil {
		return err
	}
	vol, err := s.repo.GetVolume(ctx, projectID, volumeID)
	if err != nil {
		return err
	}
	if vol.InstanceID == nil || *vol.InstanceID != instanceID || vol.TargetDev == nil {
		return ErrVolumeNotAttachedToInstance
	}
	if inst.State == InstanceStateSnapshotting {
		return ErrInstanceBusySnapshotting
	}
	path := storage.VolumeQCOW2Path(s.volDir, volumeID)
	live := inst.State == InstanceStateRunning
	if err := s.libvirt.DetachVolume(ctx, inst.Name, path, *vol.TargetDev, live); err != nil {
		return fmt.Errorf("libvirt detach: %w", err)
	}
	if err := s.repo.DetachVolumeRecord(ctx, projectID, instanceID, volumeID); err != nil {
		return err
	}
	return nil
}

// DetachAllFromInstanceForDestroy detaches every volume from the instance before the domain is undefined.
func (s *VolumeService) DetachAllFromInstanceForDestroy(ctx context.Context, projectID, instanceID uuid.UUID, instanceName string, domainLive bool) error {
	vols, err := s.repo.ListVolumesAttachedToInstance(ctx, projectID, instanceID)
	if err != nil {
		return err
	}
	for _, v := range vols {
		if v.TargetDev == nil {
			continue
		}
		path := storage.VolumeQCOW2Path(s.volDir, v.ID)
		if err := s.libvirt.DetachVolume(ctx, instanceName, path, *v.TargetDev, domainLive); err != nil {
			log.Printf("caspercloud: detach volume %s during destroy: %v", v.ID, err)
		}
		if err := s.repo.DetachVolumeRecord(ctx, projectID, instanceID, v.ID); err != nil {
			return err
		}
	}
	return nil
}
