package libvirt

import (
	"context"
	"fmt"

	libvirtgo "github.com/libvirt/libvirt-go"
)

// GracefulShutdown requests an ACPI shutdown (virDomainShutdown).
func (a *LibvirtAdapter) GracefulShutdown(ctx context.Context, instanceName string) error {
	_ = ctx
	return a.shutdownDomain(instanceName, false)
}

// HardPowerOff force-powers off a running domain (virDomainDestroy).
func (a *LibvirtAdapter) HardPowerOff(ctx context.Context, instanceName string) error {
	_ = ctx
	return a.shutdownDomain(instanceName, true)
}

func (a *LibvirtAdapter) shutdownDomain(instanceName string, destroy bool) error {
	conn, err := libvirtgo.NewConnect(a.uri)
	if err != nil {
		return fmt.Errorf("libvirt connect: %w", err)
	}
	defer func() { _, _ = conn.Close() }()

	dom, err := conn.LookupDomainByName(SanitizeLibvirtName(instanceName))
	if err != nil {
		return fmt.Errorf("lookup domain %q: %w", instanceName, err)
	}
	defer dom.Free()

	active, err := dom.IsActive()
	if err != nil {
		return fmt.Errorf("is_active: %w", err)
	}
	if !active {
		return nil
	}
	if destroy {
		if err := dom.Destroy(); err != nil {
			return fmt.Errorf("domain destroy: %w", err)
		}
		return nil
	}
	if err := dom.Shutdown(); err != nil {
		return fmt.Errorf("domain shutdown: %w", err)
	}
	return nil
}
