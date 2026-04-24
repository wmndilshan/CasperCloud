package service

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	if err := s.libvirt.DeleteVM(ctx, inst.Name, inst.ID); err != nil {
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		return fmt.Errorf("delete vm: %w", err)
	}
	return s.repo.DeleteInstance(ctx, projectID, instanceID)
}

func (s *InstanceService) HandleCreateTask(ctx context.Context, taskID, projectID, instanceID uuid.UUID) error {
	log.Printf("caspercloud worker: instance.create task start task_id=%s project_id=%s instance_id=%s", taskID, projectID, instanceID)
	if err := s.repo.UpdateTaskStatus(ctx, taskID, "running", nil); err != nil {
		log.Printf("caspercloud worker: update task running failed task_id=%s err=%v", taskID, err)
		return err
	}
	inst, err := s.repo.GetInstanceByID(ctx, instanceID)
	if err != nil {
		msg := err.Error()
		log.Printf("caspercloud worker: get instance failed task_id=%s instance_id=%s err=%v", taskID, instanceID, err)
		_ = s.repo.UpdateTaskStatus(ctx, taskID, "failed", &msg)
		return err
	}
	if inst.ProjectID != projectID {
		msg := "task project_id does not match instance project"
		log.Printf("caspercloud worker: project mismatch task_id=%s task_project=%s instance_project=%s", taskID, projectID, inst.ProjectID)
		_ = s.repo.UpdateInstanceState(ctx, inst.ProjectID, instanceID, InstanceStateError)
		_ = s.repo.UpdateTaskStatus(ctx, taskID, "failed", &msg)
		return fmt.Errorf("%s", msg)
	}
	img, err := s.repo.GetImage(ctx, projectID, inst.ImageID)
	if err != nil {
		msg := err.Error()
		log.Printf("caspercloud worker: get image failed task_id=%s image_id=%s err=%v", taskID, inst.ImageID, err)
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		_ = s.repo.UpdateTaskStatus(ctx, taskID, "failed", &msg)
		return err
	}
	basePath, err := libvirt.ImagePathFromSource(img.SourceURL)
	if err != nil {
		msg := err.Error()
		log.Printf("caspercloud worker: resolve base image failed task_id=%s source=%q err=%v", taskID, img.SourceURL, err)
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		_ = s.repo.UpdateTaskStatus(ctx, taskID, "failed", &msg)
		return err
	}
	hostname := hostnameFromUserData(inst.CloudInitData)
	cfg := libvirt.VMConfig{
		InstanceID:    inst.ID,
		InstanceName:  inst.Name,
		Hostname:      hostname,
		BaseImagePath: basePath,
		UserData:      inst.CloudInitData,
		MemoryMB:      s.defaultRAM,
		VCPUs:         s.defaultVCPU,
	}
	log.Printf("caspercloud worker: provisioning vm task_id=%s instance=%q base_image=%q mem_mb=%d vcpu=%d", taskID, inst.Name, basePath, cfg.MemoryMB, cfg.VCPUs)
	if err := s.libvirt.CreateVM(ctx, cfg); err != nil {
		msg := err.Error()
		log.Printf("caspercloud worker: libvirt CreateVM failed task_id=%s instance_id=%s err=%v", taskID, instanceID, err)
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		_ = s.repo.UpdateTaskStatus(ctx, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateRunning); err != nil {
		msg := err.Error()
		log.Printf("caspercloud worker: update instance running failed task_id=%s err=%v", taskID, err)
		_ = s.repo.UpdateTaskStatus(ctx, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.UpdateTaskStatus(ctx, taskID, "succeeded", nil); err != nil {
		log.Printf("caspercloud worker: mark task succeeded failed task_id=%s err=%v", taskID, err)
		return err
	}
	log.Printf("caspercloud worker: instance.create task done task_id=%s instance_id=%s state=running", taskID, instanceID)
	return nil
}

func hostnameFromUserData(userData string) string {
	for _, line := range strings.Split(userData, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "hostname:") {
			return strings.TrimSpace(strings.TrimPrefix(s, "hostname:"))
		}
	}
	return ""
}
