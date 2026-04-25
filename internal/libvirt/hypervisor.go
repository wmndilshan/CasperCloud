package libvirt

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"caspercloud/internal/cloudinit"
	"caspercloud/internal/config"

	"github.com/google/uuid"
	libvirtgo "github.com/libvirt/libvirt-go"
)

// LibvirtAdapter provisions and controls VMs via libvirt using qemu:///system (or configured URI).
type LibvirtAdapter struct {
	uri           string
	instancesDir  string
	defaultBridge string
	qemuImg       string
	cloudLocalds  string
	genisoimage   string
}

func NewLibvirtAdapter(cfg *config.Config) *LibvirtAdapter {
	return &LibvirtAdapter{
		uri:           cfg.LibvirtURI,
		instancesDir:  cfg.VMInstancesDir,
		defaultBridge: cfg.LibvirtBridge,
		qemuImg:       cfg.QEMUImgPath,
		cloudLocalds:  cfg.CloudLocalDSPath,
		genisoimage:   cfg.GenisoimagePath,
	}
}

func (a *LibvirtAdapter) instanceDir(id uuid.UUID) string {
	return filepath.Join(a.instancesDir, id.String())
}

// CreateVM clones the base disk, builds a NoCloud seed ISO, defines the domain XML, and starts the guest.
func (a *LibvirtAdapter) CreateVM(ctx context.Context, cfg VMConfig) error {
	domainName := SanitizeLibvirtName(cfg.InstanceName)
	if domainName != cfg.InstanceName {
		log.Printf("caspercloud libvirt: sanitized domain name from %q to %q", cfg.InstanceName, domainName)
	}
	cfg.InstanceName = domainName

	conn, err := libvirtgo.NewConnect(a.uri)
	if err != nil {
		return fmt.Errorf("libvirt connect %q: %w", a.uri, err)
	}
	defer func() {
		if _, cErr := conn.Close(); cErr != nil {
			log.Printf("caspercloud libvirt: close connection: %v", cErr)
		}
	}()

	if err := a.removeExistingDomain(conn, cfg.InstanceName); err != nil {
		return err
	}

	instDir := a.instanceDir(cfg.InstanceID)
	if err := os.MkdirAll(instDir, 0o755); err != nil {
		return fmt.Errorf("create instance dir %q: %w", instDir, err)
	}

	diskPath := filepath.Join(instDir, "root.qcow2")
	seedPath := filepath.Join(instDir, "seed.iso")
	isoWork := filepath.Join(instDir, "nocloud-staging")

	log.Printf("caspercloud libvirt: cloning base image instance_id=%s dst=%s", cfg.InstanceID, diskPath)
	if err := CloneQCOW2(ctx, a.qemuImg, cfg.BaseImagePath, diskPath); err != nil {
		_ = os.RemoveAll(instDir)
		return fmt.Errorf("clone disk: %w", err)
	}

	hostname := strings.TrimSpace(cfg.Hostname)
	if hostname == "" {
		hostname = cfg.InstanceName
	}
	log.Printf("caspercloud libvirt: building NoCloud ISO instance_id=%s path=%s", cfg.InstanceID, seedPath)
	bridge := strings.TrimSpace(cfg.BridgeName)
	if bridge == "" {
		bridge = a.defaultBridge
	}
	mac := strings.TrimSpace(cfg.MACAddress)
	if mac == "" {
		return fmt.Errorf("vm config: mac address is required for provisioning")
	}
	if err := cloudinit.BuildNoCloudISO(ctx, isoWork, seedPath, cfg.UserData, cfg.InstanceID.String(), hostname, cfg.NetworkConfigYAML, a.cloudLocalds, a.genisoimage); err != nil {
		_ = os.RemoveAll(instDir)
		return fmt.Errorf("nocloud iso: %w", err)
	}

	memKiB := cfg.MemoryMB * 1024
	if memKiB <= 0 {
		return fmt.Errorf("invalid memory_mb %d", cfg.MemoryMB)
	}
	if cfg.VCPUs <= 0 {
		return fmt.Errorf("invalid vcpu count %d", cfg.VCPUs)
	}

	diskAbs, err := filepath.Abs(diskPath)
	if err != nil {
		_ = os.RemoveAll(instDir)
		return fmt.Errorf("abs disk path: %w", err)
	}
	seedAbs, err := filepath.Abs(seedPath)
	if err != nil {
		_ = os.RemoveAll(instDir)
		return fmt.Errorf("abs seed path: %w", err)
	}

	xml, err := renderDomainXML(domainXMLInput{
		Name:        cfg.InstanceName,
		UUID:        cfg.InstanceID.String(),
		MemoryKiB:   memKiB,
		VCPUs:       cfg.VCPUs,
		BridgeName:  bridge,
		MACAddress:  strings.ToLower(mac),
		DiskPath:    diskAbs,
		SeedISOPath: seedAbs,
	})
	if err != nil {
		_ = os.RemoveAll(instDir)
		return err
	}

	log.Printf("caspercloud libvirt: defining domain name=%s uuid=%s", cfg.InstanceName, cfg.InstanceID)
	dom, err := conn.DomainDefineXML(xml)
	if err != nil {
		_ = os.RemoveAll(instDir)
		return fmt.Errorf("domain define: %w", err)
	}
	defer dom.Free()

	if err := dom.Create(); err != nil {
		_ = dom.Destroy()
		_ = dom.Undefine()
		_ = os.RemoveAll(instDir)
		return fmt.Errorf("domain start: %w", err)
	}

	log.Printf("caspercloud libvirt: domain started name=%s", cfg.InstanceName)
	return nil
}

func (a *LibvirtAdapter) removeExistingDomain(conn *libvirtgo.Connect, name string) error {
	dom, err := conn.LookupDomainByName(name)
	if err != nil {
		return nil
	}
	defer dom.Free()
	active, err := dom.IsActive()
	if err != nil {
		return fmt.Errorf("domain is_active: %w", err)
	}
	if active {
		if err := dom.Destroy(); err != nil {
			return fmt.Errorf("destroy existing domain %q: %w", name, err)
		}
	}
	if err := dom.Undefine(); err != nil {
		return fmt.Errorf("undefine existing domain %q: %w", name, err)
	}
	log.Printf("caspercloud libvirt: removed pre-existing domain %q", name)
	return nil
}

func (a *LibvirtAdapter) StartVM(ctx context.Context, instanceName string) error {
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
	if active {
		return nil
	}
	if err := dom.Create(); err != nil {
		return fmt.Errorf("domain create: %w", err)
	}
	_ = ctx
	return nil
}

func (a *LibvirtAdapter) StopVM(ctx context.Context, instanceName string) error {
	return a.GracefulShutdown(ctx, instanceName)
}

func (a *LibvirtAdapter) RebootVM(ctx context.Context, instanceName string) error {
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
		return fmt.Errorf("domain reboot: domain %q is not running", instanceName)
	}
	if err := dom.Reboot(libvirtgo.DOMAIN_REBOOT_DEFAULT); err != nil {
		return fmt.Errorf("domain reboot: %w", err)
	}
	_ = ctx
	return nil
}

// DeleteVM destroys the domain (if running), undefines it, and deletes the instance disk directory.
func (a *LibvirtAdapter) DeleteVM(ctx context.Context, instanceName string, instanceID uuid.UUID) error {
	name := SanitizeLibvirtName(instanceName)
	conn, err := libvirtgo.NewConnect(a.uri)
	if err != nil {
		return fmt.Errorf("libvirt connect: %w", err)
	}
	defer func() { _, _ = conn.Close() }()

	dom, err := conn.LookupDomainByName(name)
	if err == nil {
		defer dom.Free()
		active, aerr := dom.IsActive()
		if aerr != nil {
			return fmt.Errorf("is_active: %w", aerr)
		}
		if active {
			if err := dom.Destroy(); err != nil {
				return fmt.Errorf("destroy: %w", err)
			}
		}
		if err := dom.Undefine(); err != nil {
			return fmt.Errorf("undefine: %w", err)
		}
	}

	instDir := a.instanceDir(instanceID)
	if err := os.RemoveAll(instDir); err != nil {
		return fmt.Errorf("remove instance dir %q: %w", instDir, err)
	}
	_ = ctx
	return nil
}

// SanitizeLibvirtName maps arbitrary instance names to a libvirt-safe domain name.
func SanitizeLibvirtName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "instance"
	}
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}
