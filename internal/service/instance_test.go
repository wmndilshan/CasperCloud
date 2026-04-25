package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"caspercloud/internal/cloudinit"
	"caspercloud/internal/libvirt"
	"caspercloud/internal/queue"
	"caspercloud/internal/repository"

	"github.com/google/uuid"
)

// noopLibvirt satisfies libvirt.Adapter for flows that do not touch the hypervisor.
type noopLibvirt struct{}

func (noopLibvirt) CreateVM(context.Context, libvirt.VMConfig) error { return nil }
func (noopLibvirt) StartVM(context.Context, string) error            { return nil }
func (noopLibvirt) StopVM(context.Context, string) error             { return nil }
func (noopLibvirt) GracefulShutdown(context.Context, string) error   { return nil }
func (noopLibvirt) HardPowerOff(context.Context, string) error       { return nil }
func (noopLibvirt) RebootVM(context.Context, string) error           { return nil }
func (noopLibvirt) DeleteVM(context.Context, string, uuid.UUID) error {
	return nil
}
func (noopLibvirt) ListDomainRunning(context.Context) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (noopLibvirt) GetInstanceStats(context.Context, string) (*libvirt.InstanceMetrics, error) {
	return &libvirt.InstanceMetrics{}, nil
}

func (noopLibvirt) NextVirtioDiskTarget(context.Context, string) (string, error) { return "vdb", nil }

func (noopLibvirt) AttachVolume(context.Context, string, string, string, bool) error { return nil }

func (noopLibvirt) DetachVolume(context.Context, string, string, string, bool) error { return nil }

func (noopLibvirt) CreateInternalSnapshot(context.Context, string, string, string, bool) error {
	return nil
}

func (noopLibvirt) RevertToInternalSnapshot(context.Context, string, string, bool, bool) error {
	return nil
}

func (noopLibvirt) DeleteInternalSnapshot(context.Context, string, string) error { return nil }

type recordingPublisher struct {
	msgs []queue.TaskMessage
	err  error
}

func (r *recordingPublisher) PublishTask(ctx context.Context, msg queue.TaskMessage) error {
	if r.err != nil {
		return r.err
	}
	r.msgs = append(r.msgs, msg)
	return nil
}

type fakeInstanceStore struct {
	image          *repository.Image
	getImageErr    error
	instance       *repository.Instance
	createInstErr  error
	task           *repository.Task
	createTaskErr  error
	lastInstState  string
	lastTaskStatus string
	defaultNetwork *repository.Network
	createParams   *repository.CreateInstanceParams
}

func (f *fakeInstanceStore) GetImage(ctx context.Context, projectID, imageID uuid.UUID) (*repository.Image, error) {
	if f.getImageErr != nil {
		return nil, f.getImageErr
	}
	return f.image, nil
}

func (f *fakeInstanceStore) GetNetworkForProject(ctx context.Context, projectID, networkID uuid.UUID) (*repository.Network, error) {
	if f.defaultNetwork != nil && networkID == f.defaultNetwork.ID && projectID == f.defaultNetwork.ProjectID {
		return f.defaultNetwork, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeInstanceStore) GetDefaultNetworkID(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error) {
	if f.defaultNetwork == nil {
		return uuid.Nil, repository.ErrNotFound
	}
	return f.defaultNetwork.ID, nil
}

func (f *fakeInstanceStore) CreateInstanceWithIPAM(ctx context.Context, p repository.CreateInstanceParams) (*repository.Instance, error) {
	if f.createInstErr != nil {
		return nil, f.createInstErr
	}
	f.createParams = &p
	ip := "192.168.122.50"
	nid := p.NetworkID
	mac := p.MAC
	return &repository.Instance{
		ID:                p.InstanceID,
		ProjectID:         p.ProjectID,
		ImageID:           p.ImageID,
		Name:              p.Name,
		State:             p.InitialState,
		CloudInitData:     p.UserData,
		NetworkID:         &nid,
		MACAddress:        &mac,
		BridgeName:        p.BridgeName,
		IPv4Address:       &ip,
		NetworkConfigYAML: "version: 2\n",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}, nil
}

func (f *fakeInstanceStore) CreateTask(ctx context.Context, taskType string, projectID, instanceID uuid.UUID, status string) (*repository.Task, error) {
	if f.createTaskErr != nil {
		return nil, f.createTaskErr
	}
	if f.task != nil {
		return f.task, nil
	}
	inst := instanceID
	return &repository.Task{
		ID:         uuid.New(),
		Type:       taskType,
		ProjectID:  projectID,
		InstanceID: &inst,
		Status:     status,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

func (f *fakeInstanceStore) UpdateTaskStatus(ctx context.Context, projectID, taskID uuid.UUID, status string, errMessage *string) error {
	_ = projectID
	_ = taskID
	f.lastTaskStatus = status
	return nil
}

func (f *fakeInstanceStore) UpdateInstanceState(ctx context.Context, projectID, instanceID uuid.UUID, state string) error {
	f.lastInstState = state
	return nil
}

func (f *fakeInstanceStore) GetInstance(ctx context.Context, projectID, instanceID uuid.UUID) (*repository.Instance, error) {
	if f.instance != nil {
		return f.instance, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeInstanceStore) ListInstances(ctx context.Context, projectID uuid.UUID) ([]repository.Instance, error) {
	return nil, nil
}

func (f *fakeInstanceStore) DeleteInstance(ctx context.Context, projectID, instanceID uuid.UUID) error {
	return nil
}

func (f *fakeInstanceStore) ListInstancesForHypervisorSync(ctx context.Context) ([]repository.InstanceSyncRow, error) {
	return nil, nil
}

func (f *fakeInstanceStore) BatchUpdateInstanceStates(ctx context.Context, updates []repository.InstanceStateUpdate) error {
	return nil
}

func (f *fakeInstanceStore) FinalizeInstanceDestroy(ctx context.Context, projectID, taskID, instanceID uuid.UUID) error {
	return nil
}

func (f *fakeInstanceStore) CountVolumesAttachedToInstance(ctx context.Context, projectID, instanceID uuid.UUID) (int64, error) {
	return 0, nil
}

func (f *fakeInstanceStore) BeginSnapshotCreate(ctx context.Context, projectID, instanceID, snapshotID uuid.UUID, name string) (string, error) {
	return InstanceStateRunning, nil
}

func (f *fakeInstanceStore) BeginSnapshotRevert(ctx context.Context, projectID, instanceID uuid.UUID) (string, error) {
	return InstanceStateRunning, nil
}

func (f *fakeInstanceStore) UpdateSnapshotStatus(ctx context.Context, projectID, snapshotID uuid.UUID, status string) error {
	return nil
}

func (f *fakeInstanceStore) ListSnapshotsForInstance(ctx context.Context, projectID, instanceID uuid.UUID) ([]repository.Snapshot, error) {
	return nil, nil
}

func (f *fakeInstanceStore) GetSnapshot(ctx context.Context, projectID, instanceID, snapshotID uuid.UUID) (*repository.Snapshot, error) {
	return nil, repository.ErrNotFound
}

func (f *fakeInstanceStore) ListActiveFloatingIPNATBindingsByInstance(ctx context.Context, projectID, instanceID uuid.UUID) ([]repository.FloatingIPNATBinding, error) {
	return nil, nil
}

func (f *fakeInstanceStore) ClearFloatingIPBindingsForInstance(ctx context.Context, projectID, instanceID uuid.UUID) error {
	return nil
}

func TestInstanceService_CreateAsync(t *testing.T) {
	projectID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	imageID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	netID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	taskID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	userData, err := cloudinit.RenderUserData(cloudinit.Data{
		Hostname:     "h",
		Username:     "u",
		SSHPublicKey: "ssh-rsa AAAAB3",
	})
	if err != nil {
		t.Fatal(err)
	}

	defaultNet := &repository.Network{
		ID:         netID,
		ProjectID:  projectID,
		Name:       "default",
		CIDR:       "192.168.122.0/24",
		Gateway:    "192.168.122.1",
		BridgeName: "virbr0",
		IsDefault:  true,
	}

	tests := []struct {
		name           string
		store          *fakeInstanceStore
		publisher      *recordingPublisher
		wantPubErr     bool
		wantPubCount   int
		wantLastTask   string
		wantLastInst   string
		checkPublished func(t *testing.T, instanceID uuid.UUID, msgs []queue.TaskMessage)
	}{
		{
			name: "publishes task message after db writes",
			store: &fakeInstanceStore{
				image:          &repository.Image{ID: imageID, ProjectID: projectID, Name: "img", SourceURL: "/tmp/base.qcow2"},
				defaultNetwork: defaultNet,
				task: func() *repository.Task {
					z := uuid.Nil
					return &repository.Task{
						ID:         taskID,
						Type:       "instance.create",
						ProjectID:  projectID,
						InstanceID: &z,
						Status:     "pending",
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}
				}(),
			},
			publisher:    &recordingPublisher{},
			wantPubCount: 1,
			checkPublished: func(t *testing.T, instanceID uuid.UUID, msgs []queue.TaskMessage) {
				t.Helper()
				if len(msgs) != 1 {
					t.Fatalf("got %d messages", len(msgs))
				}
				m := msgs[0]
				if m.TaskID != taskID.String() || m.Type != "instance.create" {
					t.Fatalf("unexpected payload: %+v", m)
				}
				if m.ProjectID != projectID.String() || m.InstanceID != instanceID.String() {
					t.Fatalf("unexpected ids: %+v", m)
				}
			},
		},
		{
			name: "marks task and instance failed when publish fails",
			store: &fakeInstanceStore{
				image:          &repository.Image{ID: imageID, ProjectID: projectID, Name: "img", SourceURL: "/tmp/x"},
				defaultNetwork: defaultNet,
				task: func() *repository.Task {
					z := uuid.Nil
					return &repository.Task{
						ID:         taskID,
						Type:       "instance.create",
						ProjectID:  projectID,
						InstanceID: &z,
						Status:     "pending",
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}
				}(),
			},
			publisher:    &recordingPublisher{err: errors.New("amqp down")},
			wantPubErr:   true,
			wantLastTask: "failed",
			wantLastInst: InstanceStateError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewInstanceService(tt.store, tt.publisher, noopLibvirt{}, 2048, 2, "virbr0")
			ctx := context.Background()
			res, err := svc.CreateAsync(ctx, projectID, CreateInstanceInput{
				Name:         "vm-a",
				ImageID:      imageID,
				Hostname:     "h",
				Username:     "u",
				SSHPublicKey: "ssh-rsa AAAAB3",
			})
			if tt.wantPubErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.store.lastTaskStatus != tt.wantLastTask || tt.store.lastInstState != tt.wantLastInst {
					t.Fatalf("state: task=%q inst=%q", tt.store.lastTaskStatus, tt.store.lastInstState)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if res.Instance == nil || res.Task == nil {
				t.Fatal("nil result")
			}
			if res.Task.ID != taskID {
				t.Fatalf("task id: got %s want %s", res.Task.ID, taskID)
			}
			if res.Instance.CloudInitData != userData {
				t.Fatal("userdata mismatch")
			}
			if len(tt.publisher.msgs) != tt.wantPubCount {
				t.Fatalf("publish count: got %d want %d", len(tt.publisher.msgs), tt.wantPubCount)
			}
			if tt.checkPublished != nil {
				tt.checkPublished(t, res.Instance.ID, tt.publisher.msgs)
			}
		})
	}
}

func TestInstanceService_RequestInstanceAction_StartEnqueues(t *testing.T) {
	projectID := uuid.New()
	instanceID := uuid.New()
	taskID := uuid.New()
	inst := &repository.Instance{
		ID:        instanceID,
		ProjectID: projectID,
		ImageID:   uuid.New(),
		Name:      "vm-lib",
		State:     InstanceStateStopped,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	instID := instanceID
	store := &fakeInstanceStore{
		instance: inst,
		task: &repository.Task{
			ID:         taskID,
			Type:       TaskTypeInstanceStart,
			ProjectID:  projectID,
			InstanceID: &instID,
			Status:     "pending",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}
	pub := &recordingPublisher{}
	svc := NewInstanceService(store, pub, noopLibvirt{}, 2048, 2, "virbr0")
	res, err := svc.RequestInstanceAction(context.Background(), projectID, instanceID, "start")
	if err != nil {
		t.Fatal(err)
	}
	if res.Task.ID != taskID {
		t.Fatalf("task id got %s want %s", res.Task.ID, taskID)
	}
	if len(pub.msgs) != 1 || pub.msgs[0].Type != TaskTypeInstanceStart {
		t.Fatalf("published: %+v", pub.msgs)
	}
}
