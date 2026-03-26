package libvirt

import (
	"context"
	"fmt"
	"os/exec"
)

type VirshAdapter struct {
	uri string
}

func NewVirshAdapter(uri string) *VirshAdapter {
	return &VirshAdapter{uri: uri}
}

func (v *VirshAdapter) CreateVM(ctx context.Context, cfg VMConfig) error {
	xmlPath := fmt.Sprintf("/tmp/%s.xml", cfg.InstanceName)
	cmd := exec.CommandContext(ctx, "virsh", "-c", v.uri, "define", xmlPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("virsh define failed: %w: %s", err, string(out))
	}
	return v.StartVM(ctx, cfg.InstanceName)
}

func (v *VirshAdapter) StartVM(ctx context.Context, instanceName string) error {
	cmd := exec.CommandContext(ctx, "virsh", "-c", v.uri, "start", instanceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("virsh start failed: %w: %s", err, string(out))
	}
	return nil
}

func (v *VirshAdapter) StopVM(ctx context.Context, instanceName string) error {
	cmd := exec.CommandContext(ctx, "virsh", "-c", v.uri, "shutdown", instanceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("virsh shutdown failed: %w: %s", err, string(out))
	}
	return nil
}

func (v *VirshAdapter) RebootVM(ctx context.Context, instanceName string) error {
	cmd := exec.CommandContext(ctx, "virsh", "-c", v.uri, "reboot", instanceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("virsh reboot failed: %w: %s", err, string(out))
	}
	return nil
}

func (v *VirshAdapter) DeleteVM(ctx context.Context, instanceName string) error {
	destroyCmd := exec.CommandContext(ctx, "virsh", "-c", v.uri, "destroy", instanceName)
	_, _ = destroyCmd.CombinedOutput()

	undefineCmd := exec.CommandContext(ctx, "virsh", "-c", v.uri, "undefine", instanceName, "--remove-all-storage")
	if out, err := undefineCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("virsh undefine failed: %w: %s", err, string(out))
	}
	return nil
}
