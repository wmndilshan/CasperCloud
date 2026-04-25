package libvirt

import (
	"context"
	"fmt"
	"strings"

	libvirtgo "github.com/libvirt/libvirt-go"
)

// CreateInternalSnapshot creates a full-system internal QCOW2 snapshot (no DISK_ONLY / external chain).
// For a running guest, DOMAIN_SNAPSHOT_CREATE_LIVE reduces pause time while QEMU commits the snapshot.
func (a *LibvirtAdapter) CreateInternalSnapshot(ctx context.Context, instanceName, snapLibvirtName, description string, domainRunning bool) error {
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

	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = "caspercloud snapshot"
	}
	xml := fmt.Sprintf(
		`<domainsnapshot><name>%s</name><description>%s</description></domainsnapshot>`,
		escapeSnapshotXMLText(snapLibvirtName),
		escapeSnapshotXMLText(desc),
	)

	var flags libvirtgo.DomainSnapshotCreateFlags
	if domainRunning {
		flags |= libvirtgo.DOMAIN_SNAPSHOT_CREATE_LIVE
	}

	snap, err := dom.CreateSnapshotXML(xml, flags)
	if err != nil {
		return fmt.Errorf("snapshot-create: %w", err)
	}
	defer snap.Free()
	_ = ctx
	return nil
}

func escapeSnapshotXMLText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// RevertToInternalSnapshot reverts the domain to a named internal snapshot.
// If domainWasRunningAtSnapshot is true, libvirt is asked to leave the guest running after revert;
// if the domain is still inactive afterward, it is explicitly started.
func (a *LibvirtAdapter) RevertToInternalSnapshot(ctx context.Context, instanceName, snapLibvirtName string, domainWasRunningAtSnapshot, domainCurrentlyRunning bool) error {
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

	snap, err := dom.SnapshotLookupByName(snapLibvirtName, 0)
	if err != nil {
		return fmt.Errorf("snapshot lookup %q: %w", snapLibvirtName, err)
	}
	defer snap.Free()

	var flags libvirtgo.DomainSnapshotRevertFlags
	if domainWasRunningAtSnapshot {
		flags |= libvirtgo.DOMAIN_SNAPSHOT_REVERT_RUNNING
	}
	if domainCurrentlyRunning && !domainWasRunningAtSnapshot {
		flags |= libvirtgo.DOMAIN_SNAPSHOT_REVERT_FORCE
	}

	if err := snap.RevertToSnapshot(flags); err != nil {
		return fmt.Errorf("snapshot-revert: %w", err)
	}

	if domainWasRunningAtSnapshot || domainCurrentlyRunning {
		active, aerr := dom.IsActive()
		if aerr != nil {
			return fmt.Errorf("is_active after revert: %w", aerr)
		}
		if !active {
			if err := dom.Create(); err != nil {
				return fmt.Errorf("start domain after revert: %w", err)
			}
		}
	}
	_ = ctx
	return nil
}

// DeleteInternalSnapshot removes a libvirt snapshot by name.
func (a *LibvirtAdapter) DeleteInternalSnapshot(ctx context.Context, instanceName, snapLibvirtName string) error {
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

	snap, err := dom.SnapshotLookupByName(snapLibvirtName, 0)
	if err != nil {
		return fmt.Errorf("snapshot lookup %q: %w", snapLibvirtName, err)
	}
	defer snap.Free()

	if err := snap.Delete(libvirtgo.DomainSnapshotDeleteFlags(0)); err != nil {
		return fmt.Errorf("snapshot-delete: %w", err)
	}
	_ = ctx
	return nil
}
