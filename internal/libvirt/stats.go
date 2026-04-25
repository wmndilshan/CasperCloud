package libvirt

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	libvirtgo "github.com/libvirt/libvirt-go"
)

// InstanceMetrics holds libvirt-reported stats for an active domain.
type InstanceMetrics struct {
	DomainState    int
	MaxMemoryKiB   uint64
	MemoryKiB      uint64
	VirtCPUCount   uint
	CPUTimeNs      uint64
	DiskReadBytes  int64
	DiskWriteBytes int64
	DiskReadReq    int64
	DiskWriteReq   int64
	NetRxBytes     int64
	NetTxBytes     int64
	NetRxPackets   int64
	NetTxPackets   int64
	NetIface       string // host tap/vnet device used for stats, if any
}

var vnetTargetRE = regexp.MustCompile(`<target dev='(vnet[0-9]+)'`)

// GetInstanceStats collects CPU/memory from DomainGetInfo, disk I/O from DomainBlockStats (root disk vda),
// and network I/O from DomainInterfaceStats using the live interface target from domain XML.
func (a *LibvirtAdapter) GetInstanceStats(ctx context.Context, instanceName string) (*InstanceMetrics, error) {
	_ = ctx
	name := SanitizeLibvirtName(instanceName)
	conn, err := libvirtgo.NewConnect(a.uri)
	if err != nil {
		return nil, fmt.Errorf("libvirt connect: %w", err)
	}
	defer func() { _, _ = conn.Close() }()

	dom, err := conn.LookupDomainByName(name)
	if err != nil {
		return nil, fmt.Errorf("lookup domain %q: %w", instanceName, err)
	}
	defer dom.Free()

	active, err := dom.IsActive()
	if err != nil {
		return nil, fmt.Errorf("is_active: %w", err)
	}
	if !active {
		return nil, fmt.Errorf("domain %q is not running", name)
	}

	info, err := dom.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("domain get info: %w", err)
	}
	out := &InstanceMetrics{
		DomainState:    int(info.State),
		MaxMemoryKiB:   info.MaxMem,
		MemoryKiB:      info.Memory,
		VirtCPUCount:   info.NrVirtCpu,
		CPUTimeNs:      info.CpuTime,
	}

	if blk, err := dom.BlockStats("vda"); err == nil && blk != nil {
		if blk.RdBytesSet {
			out.DiskReadBytes = blk.RdBytes
		}
		if blk.WrBytesSet {
			out.DiskWriteBytes = blk.WrBytes
		}
		if blk.RdReqSet {
			out.DiskReadReq = blk.RdReq
		}
		if blk.WrReqSet {
			out.DiskWriteReq = blk.WrReq
		}
	}

	xml, err := dom.GetXMLDesc(0)
	if err == nil {
		if m := vnetTargetRE.FindStringSubmatch(xml); len(m) == 2 {
			iface := m[1]
			out.NetIface = iface
			if net, err := dom.InterfaceStats(iface); err == nil && net != nil {
				if net.RxBytesSet {
					out.NetRxBytes = net.RxBytes
				}
				if net.TxBytesSet {
					out.NetTxBytes = net.TxBytes
				}
				if net.RxPacketsSet {
					out.NetRxPackets = net.RxPackets
				}
				if net.TxPacketsSet {
					out.NetTxPackets = net.TxPackets
				}
			}
		}
	}

	return out, nil
}

// DomainStateString returns a short state label for libvirt DOMAIN_* constants.
func DomainStateString(state int) string {
	switch state {
	case libvirtgo.DOMAIN_RUNNING:
		return "running"
	case libvirtgo.DOMAIN_BLOCKED:
		return "blocked"
	case libvirtgo.DOMAIN_PAUSED:
		return "paused"
	case libvirtgo.DOMAIN_SHUTDOWN:
		return "shutdown"
	case libvirtgo.DOMAIN_SHUTOFF:
		return "shutoff"
	case libvirtgo.DOMAIN_CRASHED:
		return "crashed"
	case libvirtgo.DOMAIN_PMSUSPENDED:
		return "pmsuspended"
	case libvirtgo.DOMAIN_NOSTATE:
		return "nostate"
	default:
		return "unknown"
	}
}

// BlockDeviceTarget returns the primary virtio disk target used in domain XML (root disk).
func BlockDeviceTarget() string { return "vda" }

// ExtractVNetFromXML is exported for tests; returns the first vnet target dev from live XML.
func ExtractVNetFromXML(xml string) string {
	m := vnetTargetRE.FindStringSubmatch(xml)
	if len(m) != 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}
