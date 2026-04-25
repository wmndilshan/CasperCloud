package libvirt

import (
	"context"
	"fmt"
	"strings"

	libvirtgo "github.com/libvirt/libvirt-go"
)

func xmlEscapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func virtioDiskXML(absPath, targetDev string) string {
	return fmt.Sprintf(
		`<disk type='file' device='disk'><driver name='qemu' type='qcow2' discard='unmap'/><source file='%s'/><target dev='%s' bus='virtio'/></disk>`,
		xmlEscapeAttr(absPath), xmlEscapeAttr(targetDev),
	)
}

// NextVirtioDiskTarget inspects the live+persistent domain XML and returns the next free vd* letter (vdb..vdz).
func (a *LibvirtAdapter) NextVirtioDiskTarget(ctx context.Context, instanceName string) (string, error) {
	_ = ctx
	name := SanitizeLibvirtName(instanceName)
	conn, err := libvirtgo.NewConnect(a.uri)
	if err != nil {
		return "", fmt.Errorf("libvirt connect: %w", err)
	}
	defer func() { _, _ = conn.Close() }()

	dom, err := conn.LookupDomainByName(name)
	if err != nil {
		return "", fmt.Errorf("lookup domain %q: %w", instanceName, err)
	}
	defer dom.Free()

	xml, err := dom.GetXMLDesc(0)
	if err != nil {
		return "", fmt.Errorf("get domain xml: %w", err)
	}
	return NextVirtioDataDiskTarget(xml)
}

// AttachVolume hot-plugs (when live=true) a virtio qcow2 disk and persists it in the domain config.
func (a *LibvirtAdapter) AttachVolume(ctx context.Context, instanceName, absPath, targetDev string, live bool) error {
	_ = ctx
	name := SanitizeLibvirtName(instanceName)
	conn, err := libvirtgo.NewConnect(a.uri)
	if err != nil {
		return fmt.Errorf("libvirt connect: %w", err)
	}
	defer func() { _, _ = conn.Close() }()

	dom, err := conn.LookupDomainByName(name)
	if err != nil {
		return fmt.Errorf("lookup domain %q: %w", instanceName, err)
	}
	defer dom.Free()

	xml := virtioDiskXML(absPath, targetDev)
	flags := libvirtgo.DomainDeviceModifyFlags(libvirtgo.DOMAIN_AFFECT_CONFIG)
	if live {
		flags |= libvirtgo.DomainDeviceModifyFlags(libvirtgo.DOMAIN_AFFECT_LIVE)
	}
	if err := dom.AttachDeviceFlags(xml, flags); err != nil {
		return fmt.Errorf("attach device %s: %w", targetDev, err)
	}
	return nil
}

// DetachVolume removes a virtio disk by target dev from live and/or persistent domain configuration.
func (a *LibvirtAdapter) DetachVolume(ctx context.Context, instanceName, absPath, targetDev string, live bool) error {
	_ = ctx
	name := SanitizeLibvirtName(instanceName)
	conn, err := libvirtgo.NewConnect(a.uri)
	if err != nil {
		return fmt.Errorf("libvirt connect: %w", err)
	}
	defer func() { _, _ = conn.Close() }()

	dom, err := conn.LookupDomainByName(name)
	if err != nil {
		return fmt.Errorf("lookup domain %q: %w", instanceName, err)
	}
	defer dom.Free()

	xml := virtioDiskXML(absPath, targetDev)
	flags := libvirtgo.DomainDeviceModifyFlags(libvirtgo.DOMAIN_AFFECT_CONFIG)
	if live {
		flags |= libvirtgo.DomainDeviceModifyFlags(libvirtgo.DOMAIN_AFFECT_LIVE)
	}
	if err := dom.DetachDeviceFlags(xml, flags); err != nil {
		return fmt.Errorf("detach device %s: %w", targetDev, err)
	}
	return nil
}
