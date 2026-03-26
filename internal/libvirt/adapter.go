package libvirt

import "context"

type VMConfig struct {
	InstanceName string
	ImageRef     string
	DiskPath     string
	MemoryMB     int
	VCPUs        int
	CloudInitISO string
}

type Adapter interface {
	CreateVM(ctx context.Context, cfg VMConfig) error
	StartVM(ctx context.Context, instanceName string) error
	StopVM(ctx context.Context, instanceName string) error
	RebootVM(ctx context.Context, instanceName string) error
	DeleteVM(ctx context.Context, instanceName string) error
}
