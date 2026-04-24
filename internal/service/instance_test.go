package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"caspercloud/internal/cloudinit"
	"caspercloud/internal/libvirt"
	"caspercloud/internal/mocks/libvirtmock"
	"caspercloud/internal/queue"
	"caspercloud/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// noopLibvirt satisfies libvirt.Adapter for flows that do not touch the hypervisor.
type noopLibvirt struct{}

func (noopLibvirt) CreateVM(context.Context, libvirt.VMConfig) error { return nil }
func (noopLibvirt) StartVM(context.Context, string) error            { return nil }
func (noopLibvirt) StopVM(context.Context, string) error             { return nil }
func (noopLibvirt) RebootVM(context.Context, string) error           { return nil }
func (noopLibvirt) DeleteVM(context.Context, string, uuid.UUID) error {
	return nil
}

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
}

func (f *fakeInstanceStore) GetImage(ctx context.Context, projectID, imageID uuid.UUID) (*repository.Image, error) {
	if f.getImageErr != nil {
		return nil, f.getImageErr
	}
	return f.image, nil
}

func (f *fakeInstanceStore) CreateInstance(ctx context.Context, projectID, imageID uuid.UUID, name, cloudInitData, initialState string) (*repository.Instance, error) {
	if f.createInstErr != nil {
		return nil, f.createInstErr
	}
	if f.instance != nil {
		return f.instance, nil
	}
	return &repository.Instance{
		ID:        uuid.New(),
		ProjectID: projectID,
		ImageID:   imageID,
		Name:      name,
		State:     initialState,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (f *fakeInstanceStore) CreateTask(ctx context.Context, taskType string, projectID, instanceID uuid.UUID, status string) (*repository.Task, error) {
	if f.createTaskErr != nil {
		return nil, f.createTaskErr
	}
	if f.task != nil {
		return f.task, nil
	}
	return &repository.Task{
		ID:         uuid.New(),
		Type:       taskType,
		ProjectID:  projectID,
		InstanceID: instanceID,
		Status:     status,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

func (f *fakeInstanceStore) UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status string, errMessage *string) error {
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

func (f *fakeInstanceStore) GetInstanceByID(ctx context.Context, instanceID uuid.UUID) (*repository.Instance, error) {
	if f.instance != nil && f.instance.ID == instanceID {
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

func TestInstanceService_CreateAsync(t *testing.T) {
	projectID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	imageID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	instanceID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	taskID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	userData, err := cloudinit.RenderUserData(cloudinit.Data{
		Hostname:     "h",
		Username:     "u",
		SSHPublicKey: "ssh-rsa AAAAB3",
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		store          *fakeInstanceStore
		publisher      *recordingPublisher
		wantPubErr     bool
		wantPubCount   int
		wantLastTask   string
		wantLastInst   string
		checkPublished func(t *testing.T, msgs []queue.TaskMessage)
	}{
		{
			name: "publishes task message after db writes",
			store: &fakeInstanceStore{
				image: &repository.Image{ID: imageID, ProjectID: projectID, Name: "img", SourceURL: "/tmp/base.qcow2"},
				instance: &repository.Instance{
					ID:            instanceID,
					ProjectID:     projectID,
					ImageID:       imageID,
					Name:          "vm-a",
					State:         InstanceStateCreating,
					CloudInitData: userData,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				},
				task: &repository.Task{
					ID:         taskID,
					Type:       "instance.create",
					ProjectID:  projectID,
					InstanceID: instanceID,
					Status:     "pending",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
			},
			publisher:    &recordingPublisher{},
			wantPubCount: 1,
			checkPublished: func(t *testing.T, msgs []queue.TaskMessage) {
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
				image: &repository.Image{ID: imageID, ProjectID: projectID, Name: "img", SourceURL: "/tmp/x"},
				instance: &repository.Instance{
					ID:            instanceID,
					ProjectID:     projectID,
					ImageID:       imageID,
					Name:          "vm-b",
					State:         InstanceStateCreating,
					CloudInitData: userData,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				},
				task: &repository.Task{
					ID:         taskID,
					Type:       "instance.create",
					ProjectID:  projectID,
					InstanceID: instanceID,
					Status:     "pending",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
			},
			publisher:    &recordingPublisher{err: errors.New("amqp down")},
			wantPubErr:   true,
			wantLastTask: "failed",
			wantLastInst: InstanceStateError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewInstanceService(tt.store, tt.publisher, noopLibvirt{}, 2048, 2)
			ctx := context.Background()
			res, err := svc.CreateAsync(ctx, projectID, CreateInstanceInput{
				Name:         tt.store.instance.Name,
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
			if res.Instance.ID != instanceID || res.Task.ID != taskID {
				t.Fatalf("ids instance=%s task=%s", res.Instance.ID, res.Task.ID)
			}
			if len(tt.publisher.msgs) != tt.wantPubCount {
				t.Fatalf("publish count: got %d want %d", len(tt.publisher.msgs), tt.wantPubCount)
			}
			if tt.checkPublished != nil {
				tt.checkPublished(t, tt.publisher.msgs)
			}
		})
	}
}

func TestInstanceService_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	projectID := uuid.New()
	instanceID := uuid.New()
	inst := &repository.Instance{
		ID:        instanceID,
		ProjectID: projectID,
		ImageID:   uuid.New(),
		Name:      "vm-lib",
		State:     InstanceStateStopped,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store := &fakeInstanceStore{instance: inst}
	lv := libvirtmock.NewMockAdapter(ctrl)
	lv.EXPECT().StartVM(gomock.Any(), "vm-lib").Return(nil)

	svc := NewInstanceService(store, &recordingPublisher{}, lv, 2048, 2)
	if err := svc.Start(context.Background(), projectID, instanceID); err != nil {
		t.Fatal(err)
	}
	if store.lastInstState != InstanceStateRunning {
		t.Fatalf("state=%q", store.lastInstState)
	}
}
