package libvirt

import (
	"context"

	"github.com/google/uuid"
)

// VMConfig carries everything required to define and boot a new guest.
type VMConfig struct {
	InstanceID        uuid.UUID
	InstanceName      string
	Hostname          string // used for NoCloud meta-data when non-empty
	BaseImagePath     string // absolute path to backing qcow2
	UserData          string // full #cloud-config payload for user-data
	NetworkConfigYAML string // cloud-init v2 network-config (optional)
	MACAddress        string // virtio NIC MAC (must match network-config match.macaddress)
	BridgeName        string // libvirt bridge, e.g. virbr0
	MemoryMB          int
	VCPUs             int
}

// Adapter is the hypervisor surface used by the worker and state sync.
type Adapter interface {
	CreateVM(ctx context.Context, cfg VMConfig) error
	StartVM(ctx context.Context, instanceName string) error
	// StopVM is equivalent to GracefulShutdown (ACPI shutdown).
	StopVM(ctx context.Context, instanceName string) error
	GracefulShutdown(ctx context.Context, instanceName string) error
	HardPowerOff(ctx context.Context, instanceName string) error
	RebootVM(ctx context.Context, instanceName string) error
	DeleteVM(ctx context.Context, instanceName string, instanceID uuid.UUID) error
	// ListDomainRunning maps libvirt domain name -> true if the domain is running (or paused, etc.).
	ListDomainRunning(ctx context.Context) (map[string]bool, error)
	// GetInstanceStats returns stats for a running domain (by instance name, sanitized as for other calls).
	GetInstanceStats(ctx context.Context, instanceName string) (*InstanceMetrics, error)
	// NextVirtioDiskTarget returns the next free virtio data disk target (vdb..vdz) from current domain XML.
	NextVirtioDiskTarget(ctx context.Context, instanceName string) (string, error)
	// AttachVolume adds a virtio qcow2 disk (live+config when live is true).
	AttachVolume(ctx context.Context, instanceName, absPath, targetDev string, live bool) error
	// DetachVolume removes a virtio disk by matching XML (live+config when live is true).
	DetachVolume(ctx context.Context, instanceName, absPath, targetDev string, live bool) error
}
