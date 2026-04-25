package libvirt

import (
	"context"
	"fmt"

	libvirtgo "github.com/libvirt/libvirt-go"
)

// ListDomainRunning reports whether each libvirt domain name (as returned by GetName) is in a "running" libvirt state
// (RUNNING, PAUSED, BLOCKED, PMSUSPENDED). Shutoff and crashed domains map to false.
func (a *LibvirtAdapter) ListDomainRunning(ctx context.Context) (map[string]bool, error) {
	_ = ctx
	conn, err := libvirtgo.NewConnect(a.uri)
	if err != nil {
		return nil, fmt.Errorf("libvirt connect: %w", err)
	}
	defer func() { _, _ = conn.Close() }()

	flags := libvirtgo.CONNECT_LIST_DOMAINS_ACTIVE | libvirtgo.CONNECT_LIST_DOMAINS_INACTIVE
	doms, err := conn.ListAllDomains(flags)
	if err != nil {
		return nil, fmt.Errorf("list all domains: %w", err)
	}

	out := make(map[string]bool, len(doms))
	for _, dom := range doms {
		if dom == nil {
			continue
		}
		name, err := dom.GetName()
		if err != nil {
			_ = dom.Free()
			continue
		}
		state, _, err := dom.GetState()
		if err != nil {
			_ = dom.Free()
			continue
		}
		running := domainStateIsRunning(state)
		out[name] = running
		_ = dom.Free()
	}
	return out, nil
}

func domainStateIsRunning(state int) bool {
	switch state {
	case libvirtgo.DOMAIN_RUNNING,
		libvirtgo.DOMAIN_BLOCKED,
		libvirtgo.DOMAIN_PAUSED,
		libvirtgo.DOMAIN_PMSUSPENDED:
		return true
	default:
		return false
	}
}
