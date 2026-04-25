package cloudinit

import (
	"fmt"
	"net/netip"
	"strings"
)

// RenderNetworkConfigV2 returns cloud-init v2 network-config YAML for static IPv4 on first boot.
func RenderNetworkConfigV2(mac string, addr, gateway netip.Addr, prefixBits int) (string, error) {
	if !addr.Is4() || !gateway.Is4() {
		return "", fmt.Errorf("only IPv4 is supported for network-config")
	}
	if prefixBits < 0 || prefixBits > 32 {
		return "", fmt.Errorf("invalid prefix length %d", prefixBits)
	}
	m := strings.ToLower(strings.TrimSpace(mac))
	if m == "" {
		return "", fmt.Errorf("mac is required")
	}
	// Match virtio NIC by MAC (stable across distros).
	return fmt.Sprintf(`version: 2
ethernets:
  casper0:
    match:
      macaddress: '%s'
    dhcp4: false
    addresses:
      - %s/%d
    gateway4: %s
    nameservers:
      addresses: [8.8.8.8, 8.8.4.4]
`, m, addr.String(), prefixBits, gateway.String()), nil
}
