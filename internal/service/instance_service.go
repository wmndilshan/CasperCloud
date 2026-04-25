package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"caspercloud/internal/cloudinit"
	"caspercloud/internal/instancemetrics"
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

const (
	TaskTypeInstanceCreate  = "instance.create"
	TaskTypeInstanceStart   = "instance.start"
	TaskTypeInstanceStop    = "instance.stop"
	TaskTypeInstanceReboot  = "instance.reboot"
	TaskTypeInstanceDestroy = "instance.destroy"
)

var (
	ErrInvalidInstanceAction   = errors.New("invalid instance action")
	ErrInvalidStateTransition  = errors.New("invalid instance state for this action")
	ErrStatsFetcherNotConfigured = errors.New("instance stats require REDIS_URL (worker must publish metrics to the same Redis)")
	ErrStatsNotInCache         = errors.New("no metrics yet for this instance")
	ErrStatsStale              = errors.New("instance metrics are stale")
)

const maxStatsPayloadAge = 45 * time.Second

// StatsFetcher loads raw JSON for instance metrics (typically Redis, written by the worker).
type StatsFetcher interface {
	FetchInstanceStatsPayload(ctx context.Context, instanceID uuid.UUID) ([]byte, error)
}

// volumeDetachCoordinator detaches block volumes before an instance domain is destroyed.
type volumeDetachCoordinator interface {
	DetachAllFromInstanceForDestroy(ctx context.Context, projectID, instanceID uuid.UUID, instanceName string, domainLive bool) error
}

type InstanceService struct {
	repo           InstanceStore
	publisher      queue.TaskPublisher
	libvirt        libvirt.Adapter
	statsFetcher   StatsFetcher
	volumes        volumeDetachCoordinator
	defaultRAM     int
	defaultVCPU    int
	defaultBridge  string
}

type InstanceServiceOption func(*InstanceService)

// WithStatsFetcher configures Redis-backed (or other) stats retrieval for GET .../stats.
func WithStatsFetcher(f StatsFetcher) InstanceServiceOption {
	return func(s *InstanceService) {
		s.statsFetcher = f
	}
}

// WithVolumes wires volume detach during instance destroy (worker + API should set this when VolumeService exists).
func WithVolumes(v volumeDetachCoordinator) InstanceServiceOption {
	return func(s *InstanceService) {
		s.volumes = v
	}
}

func NewInstanceService(repo InstanceStore, publisher queue.TaskPublisher, libvirtAdapter libvirt.Adapter, defaultRAM, defaultVCPU int, defaultBridge string, opts ...InstanceServiceOption) *InstanceService {
	s := &InstanceService{
		repo:          repo,
		publisher:     publisher,
		libvirt:       libvirtAdapter,
		defaultRAM:    defaultRAM,
		defaultVCPU:   defaultVCPU,
		defaultBridge: defaultBridge,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

type CreateInstanceInput struct {
	Name         string
	ImageID      uuid.UUID
	Hostname     string
	Username     string
	SSHPublicKey string
	Packages     []string
	RunCommands  []string
	NetworkID    *uuid.UUID
}

type CreateInstanceResult struct {
	Instance *repository.Instance `json:"instance"`
	Task     *repository.Task     `json:"task"`
}

// InstanceActionResult is returned when a lifecycle action is accepted asynchronously.
type InstanceActionResult struct {
	Task *repository.Task `json:"task"`
}

func macForInstance(id uuid.UUID) string {
	sum := sha256.Sum256(id[:])
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", sum[0], sum[1], sum[2])
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

	var net *repository.Network
	if in.NetworkID != nil && *in.NetworkID != uuid.Nil {
		net, err = s.repo.GetNetworkForProject(ctx, projectID, *in.NetworkID)
		if err != nil {
			return nil, err
		}
	} else {
		nid, err := s.repo.GetDefaultNetworkID(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("default network: %w", err)
		}
		net, err = s.repo.GetNetworkForProject(ctx, projectID, nid)
		if err != nil {
			return nil, err
		}
	}

	instID := uuid.New()
	mac := macForInstance(instID)

	instance, err := s.repo.CreateInstanceWithIPAM(ctx, repository.CreateInstanceParams{
		InstanceID:     instID,
		ProjectID:      projectID,
		ImageID:      in.ImageID,
		Name:         in.Name,
		UserData:     userData,
		InitialState: InstanceStateCreating,
		NetworkID:    net.ID,
		NetworkCIDR:  net.CIDR,
		NetworkGateway: net.Gateway,
		BridgeName:   net.BridgeName,
		MAC:          mac,
	})
	if err != nil {
		return nil, err
	}

	task, err := s.repo.CreateTask(ctx, TaskTypeInstanceCreate, projectID, instance.ID, "pending")
	if err != nil {
		return nil, err
	}
	err = s.publisher.PublishTask(ctx, queue.TaskMessage{
		TaskID:     task.ID.String(),
		Type:       task.Type,
		ProjectID:  projectID.String(),
		InstanceID: instance.ID.String(),
	})
	if err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, task.ID, "failed", &msg)
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

// GetInstanceStats returns the latest metrics snapshot for an instance (Redis, populated by the worker).
func (s *InstanceService) GetInstanceStats(ctx context.Context, projectID, instanceID uuid.UUID) (*instancemetrics.Payload, error) {
	if s.statsFetcher == nil {
		return nil, ErrStatsFetcherNotConfigured
	}
	if _, err := s.repo.GetInstance(ctx, projectID, instanceID); err != nil {
		return nil, err
	}
	raw, err := s.statsFetcher.FetchInstanceStatsPayload(ctx, instanceID)
	if err != nil {
		if errors.Is(err, instancemetrics.ErrNoPayload) {
			return nil, ErrStatsNotInCache
		}
		return nil, err
	}
	var p instancemetrics.Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("decode instance metrics: %w", err)
	}
	if p.ProjectID != projectID {
		return nil, repository.ErrNotFound
	}
	if p.InstanceID != instanceID {
		return nil, repository.ErrNotFound
	}
	if time.Since(p.CollectedAt) > maxStatsPayloadAge {
		return nil, ErrStatsStale
	}
	return &p, nil
}

func taskTypeForAction(action string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "start":
		return TaskTypeInstanceStart, nil
	case "stop":
		return TaskTypeInstanceStop, nil
	case "reboot":
		return TaskTypeInstanceReboot, nil
	case "destroy":
		return TaskTypeInstanceDestroy, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidInstanceAction, action)
	}
}

func actionAllowed(state, action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "start":
		return state == InstanceStateStopped || state == InstanceStateError
	case "stop":
		return state == InstanceStateRunning
	case "reboot":
		return state == InstanceStateRunning
	case "destroy":
		return state != InstanceStateCreating && state != InstanceStateDeleting
	default:
		return false
	}
}

// RequestInstanceAction validates the current instance state, enqueues a worker task, and publishes to RabbitMQ.
func (s *InstanceService) RequestInstanceAction(ctx context.Context, projectID, instanceID uuid.UUID, action string) (*InstanceActionResult, error) {
	taskType, err := taskTypeForAction(action)
	if err != nil {
		return nil, err
	}
	inst, err := s.repo.GetInstance(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if !actionAllowed(inst.State, action) {
		return nil, fmt.Errorf("%w (state=%s action=%s)", ErrInvalidStateTransition, inst.State, action)
	}
	if taskType == TaskTypeInstanceDestroy {
		n, err := s.repo.CountVolumesAttachedToInstance(ctx, projectID, instanceID)
		if err != nil {
			return nil, err
		}
		if n > 0 {
			return nil, ErrInstanceHasAttachedVolumes
		}
	}
	task, err := s.repo.CreateTask(ctx, taskType, projectID, instanceID, "pending")
	if err != nil {
		return nil, err
	}
	pubErr := s.publisher.PublishTask(ctx, queue.TaskMessage{
		TaskID:     task.ID.String(),
		Type:       task.Type,
		ProjectID:  projectID.String(),
		InstanceID: instanceID.String(),
	})
	if pubErr != nil {
		msg := pubErr.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, task.ID, "failed", &msg)
		return nil, pubErr
	}
	return &InstanceActionResult{Task: task}, nil
}

// SyncHypervisorWithDB reconciles instances.state with qemu domain activity (batched DB updates).
func (s *InstanceService) SyncHypervisorWithDB(ctx context.Context) error {
	libMap, err := s.libvirt.ListDomainRunning(ctx)
	if err != nil {
		return err
	}
	rows, err := s.repo.ListInstancesForHypervisorSync(ctx)
	if err != nil {
		return err
	}
	var updates []repository.InstanceStateUpdate
	for _, row := range rows {
		if row.State == InstanceStateCreating {
			continue
		}
		domName := libvirt.SanitizeLibvirtName(row.Name)
		libRunning := libMap[domName]
		dbRunning := row.State == InstanceStateRunning
		if libRunning == dbRunning {
			continue
		}
		newState := InstanceStateStopped
		if libRunning {
			newState = InstanceStateRunning
		}
		updates = append(updates, repository.InstanceStateUpdate{
			ProjectID:  row.ProjectID,
			InstanceID: row.ID,
			NewState:   newState,
		})
	}
	return s.repo.BatchUpdateInstanceStates(ctx, updates)
}

func (s *InstanceService) HandleStartTask(ctx context.Context, taskID, projectID, instanceID uuid.UUID) error {
	return s.handleSimpleLifecycle(ctx, taskID, projectID, instanceID, func(ctx context.Context, inst *repository.Instance) error {
		return s.libvirt.StartVM(ctx, inst.Name)
	}, InstanceStateRunning)
}

func (s *InstanceService) HandleStopTask(ctx context.Context, taskID, projectID, instanceID uuid.UUID) error {
	return s.handleSimpleLifecycle(ctx, taskID, projectID, instanceID, func(ctx context.Context, inst *repository.Instance) error {
		return s.libvirt.GracefulShutdown(ctx, inst.Name)
	}, InstanceStateStopped)
}

func (s *InstanceService) HandleRebootTask(ctx context.Context, taskID, projectID, instanceID uuid.UUID) error {
	return s.handleSimpleLifecycle(ctx, taskID, projectID, instanceID, func(ctx context.Context, inst *repository.Instance) error {
		return s.libvirt.RebootVM(ctx, inst.Name)
	}, InstanceStateRunning)
}

func (s *InstanceService) handleSimpleLifecycle(ctx context.Context, taskID, projectID, instanceID uuid.UUID, fn func(context.Context, *repository.Instance) error, wantState string) error {
	log.Printf("caspercloud worker: lifecycle task start task_id=%s project_id=%s instance_id=%s want_state=%s", taskID, projectID, instanceID, wantState)
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, "running", nil); err != nil {
		return err
	}
	inst, err := s.repo.GetInstance(ctx, projectID, instanceID)
	if err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	if err := fn(ctx, inst); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.UpdateInstanceState(ctx, projectID, instanceID, wantState); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, "succeeded", nil); err != nil {
		return err
	}
	log.Printf("caspercloud worker: lifecycle task done task_id=%s instance_id=%s", taskID, instanceID)
	return nil
}

func (s *InstanceService) HandleDestroyTask(ctx context.Context, taskID, projectID, instanceID uuid.UUID) error {
	log.Printf("caspercloud worker: instance.destroy task start task_id=%s project_id=%s instance_id=%s", taskID, projectID, instanceID)
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, "running", nil); err != nil {
		return err
	}
	inst, err := s.repo.GetInstance(ctx, projectID, instanceID)
	if err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	domainLive := inst.State == InstanceStateRunning

	if err := s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateDeleting); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	if s.volumes != nil {
		if err := s.volumes.DetachAllFromInstanceForDestroy(ctx, projectID, instanceID, inst.Name, domainLive); err != nil {
			msg := err.Error()
			_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
			_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
			return err
		}
	}
	if err := s.libvirt.DeleteVM(ctx, inst.Name, inst.ID); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.FinalizeInstanceDestroy(ctx, projectID, taskID, instanceID); err != nil {
		log.Printf("caspercloud worker: CRITICAL finalize destroy failed task_id=%s instance_id=%s err=%v", taskID, instanceID, err)
		return err
	}
	log.Printf("caspercloud worker: instance.destroy task done task_id=%s instance_id=%s", taskID, instanceID)
	return nil
}

func (s *InstanceService) HandleCreateTask(ctx context.Context, taskID, projectID, instanceID uuid.UUID) error {
	log.Printf("caspercloud worker: instance.create task start task_id=%s project_id=%s instance_id=%s", taskID, projectID, instanceID)
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, "running", nil); err != nil {
		log.Printf("caspercloud worker: update task running failed task_id=%s err=%v", taskID, err)
		return err
	}
	inst, err := s.repo.GetInstance(ctx, projectID, instanceID)
	if err != nil {
		msg := err.Error()
		log.Printf("caspercloud worker: get instance failed task_id=%s instance_id=%s err=%v", taskID, instanceID, err)
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	img, err := s.repo.GetImage(ctx, projectID, inst.ImageID)
	if err != nil {
		msg := err.Error()
		log.Printf("caspercloud worker: get image failed task_id=%s image_id=%s err=%v", taskID, inst.ImageID, err)
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	basePath, err := libvirt.ImagePathFromSource(img.SourceURL)
	if err != nil {
		msg := err.Error()
		log.Printf("caspercloud worker: resolve base image failed task_id=%s source=%q err=%v", taskID, img.SourceURL, err)
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	hostname := hostnameFromUserData(inst.CloudInitData)
	bridge := inst.BridgeName
	if bridge == "" {
		bridge = s.defaultBridge
	}
	mac := ""
	if inst.MACAddress != nil {
		mac = *inst.MACAddress
	}
	cfg := libvirt.VMConfig{
		InstanceID:          inst.ID,
		InstanceName:        inst.Name,
		Hostname:            hostname,
		BaseImagePath:       basePath,
		UserData:            inst.CloudInitData,
		NetworkConfigYAML:   inst.NetworkConfigYAML,
		MACAddress:          mac,
		BridgeName:          bridge,
		MemoryMB:            s.defaultRAM,
		VCPUs:               s.defaultVCPU,
	}
	log.Printf("caspercloud worker: provisioning vm task_id=%s instance=%q base_image=%q mem_mb=%d vcpu=%d", taskID, inst.Name, basePath, cfg.MemoryMB, cfg.VCPUs)
	if err := s.libvirt.CreateVM(ctx, cfg); err != nil {
		msg := err.Error()
		log.Printf("caspercloud worker: libvirt CreateVM failed task_id=%s instance_id=%s err=%v", taskID, instanceID, err)
		_ = s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateError)
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.UpdateInstanceState(ctx, projectID, instanceID, InstanceStateRunning); err != nil {
		msg := err.Error()
		log.Printf("caspercloud worker: update instance running failed task_id=%s err=%v", taskID, err)
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, "succeeded", nil); err != nil {
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
