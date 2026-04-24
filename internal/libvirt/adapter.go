package libvirt

import (
	"context"

	"github.com/google/uuid"
)

// VMConfig carries everything required to define and boot a new guest.
type VMConfig struct {
	InstanceID    uuid.UUID
	InstanceName  string
	Hostname      string // used for NoCloud meta-data when non-empty
	BaseImagePath string // absolute path to backing qcow2
	UserData      string // full #cloud-config payload for user-data
	MemoryMB      int
	VCPUs         int
}

type Adapter interface {
	CreateVM(ctx context.Context, cfg VMConfig) error
	StartVM(ctx context.Context, instanceName string) error
	StopVM(ctx context.Context, instanceName string) error
	RebootVM(ctx context.Context, instanceName string) error
	DeleteVM(ctx context.Context, instanceName string, instanceID uuid.UUID) error
}
