package service

import (
	"context"
	"fmt"

	"caspercloud/internal/cloudinit"
	"caspercloud/internal/libvirt"
	"caspercloud/internal/queue"
	"caspercloud/internal/repository"
	"github.com/google/uuid"
)

const (
	InstanceStateCreating = "creating"
	InstanceStateRunning  = "running"
	InstanceStateStopped  = "stopped"
	InstanceStateDeleting = "deleting"
	InstanceStateError    = "error"
)

type InstanceService struct {
	repo        *repository.Repository
	queueClient *queue.Client
	libvirt     libvirt.Adapter
	defaultRAM  int
	defaultVCPU int
}

func NewInstanceService(repo *repository.Repository, queueClient *queue.Client, libvirtAdapter libvirt.Adapter, defaultRAM, defaultVCPU int) *InstanceService {
	return &InstanceService{
		repo:        repo,
		queueClient: queueClient,
		libvirt:     libvirtAdapter,
		defaultRAM:  defaultRAM,
		defaultVCPU: defaultVCPU,
	}
}

type CreateInstanceInput struct {
	Name         string
	ImageID      uuid.UUID
	Hostname     string
	Username     string
	SSHPublicKey string
	Packages     []string
	RunCommands  []string
}

type CreateInstanceResult struct {
	Instance *repository.Instance `json:"instance"`
	Task     *repository.Task     `json:"task"`
}

func (s *InstanceService) CreateAsync(ctx context.Context, projectID uuid.UUID, in CreateInstanceInput) (*CreateInstanceResult, error) {
	if _, err := s.repo.GetImage(ctx, projectID, in.ImageID); err != nil {
		return nil, err
	}
	userData, err := cloudinit.RenderUserData(cloudinit.Data{
		Hostname:     in.Hostname,
		Username:     in.Username,
		SSHPublicKey: in.SSHPublicKey,
		Packages:     in.Packages,
		RunCommands:  in.RunCommands,
	})
	if err != nil {
		return nil, err
	}
	instance, err := s.repo.CreateInstance(ctx, projectID, in.ImageID, in.Name, userData, InstanceStateCreating)
	if err != nil {
		return nil, err
	}
	task, err := s.repo.CreateTask(ctx, "instance.create", projectID, instance.ID, "pending")
	if err != nil {
		return nil, err
	}
	err = s.queueClient.PublishTask(ctx, queue.TaskMessage{
		TaskID:     task.ID.String(),
		Type:       task.Type,
		ProjectID:  projectID.String(),
		InstanceID: instance.ID.String(),
	})
	if err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, task.ID, "failed", &msg)
		_ = s.repo.UpdateInstanceState(ctx, projectID, instance.ID, InstanceStateError)
		return nil, err
	}
	return &CreateInstanceResult{Instance: instance, Task: task}, nil
}

func (s *InstanceService) Get(ctx context.Context, projectID, instanceID uuid.UUID) (*repository.Instance, error) {
	return s.repo.GetInstance(ctx, projectID, instanceID)
}

func (s *InstanceService) List(ctx context.Context, projectID uuid.UUID) ([]repository.Instance, error) {
	return s.repo.ListInstances(ctx, projectID)
}

func (s *InstanceService) Start(ctx context.Context, projectID, instanceID uuid.UUID) error {
	inst, err := s.repo.GetInstance(ctx, projectID, instanceID)
	if err != nil {
		return err
	}
	if err := s.libvirt.StartVM(ctx, inst.Name); err != nil {
		return fmt.Errorf("start vm: %w", err)
	}
	return s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateRunning)
}

func (s *InstanceService) Stop(ctx context.Context, projectID, instanceID uuid.UUID) error {
	inst, err := s.repo.GetInstance(ctx, projectID, instanceID)
	if err != nil {
		return err
	}
	if err := s.libvirt.StopVM(ctx, inst.Name); err != nil {
		return fmt.Errorf("stop vm: %w", err)
	}
	return s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateStopped)
}

func (s *InstanceService) Reboot(ctx context.Context, projectID, instanceID uuid.UUID) error {
	inst, err := s.repo.GetInstance(ctx, projectID, instanceID)
	if err != nil {
		return err
	}
	if err := s.libvirt.RebootVM(ctx, inst.Name); err != nil {
		return fmt.Errorf("reboot vm: %w", err)
	}
	return s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateRunning)
}

func (s *InstanceService) Delete(ctx context.Context, projectID, instanceID uuid.UUID) error {
	inst, err := s.repo.GetInstance(ctx, projectID, instanceID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateDeleting); err != nil {
		return err
	}
	if err := s.libvirt.DeleteVM(ctx, inst.Name); err != nil {
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		return fmt.Errorf("delete vm: %w", err)
	}
	return s.repo.DeleteInstance(ctx, projectID, instanceID)
}

func (s *InstanceService) HandleCreateTask(ctx context.Context, taskID, projectID, instanceID uuid.UUID) error {
	if err := s.repo.UpdateTaskStatus(ctx, taskID, "running", nil); err != nil {
		return err
	}
	inst, err := s.repo.GetInstanceByID(ctx, instanceID)
	if err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, taskID, "failed", &msg)
		return err
	}
	cfg := libvirt.VMConfig{
		InstanceName: inst.Name,
		ImageRef:     inst.ImageID.String(),
		DiskPath:     "",
		MemoryMB:     s.defaultRAM,
		VCPUs:        s.defaultVCPU,
		CloudInitISO: "",
	}
	if err := s.libvirt.CreateVM(ctx, cfg); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		_ = s.repo.UpdateTaskStatus(ctx, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateRunning); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.UpdateTaskStatus(ctx, taskID, "succeeded", nil); err != nil {
		return err
	}
	return nil
}
