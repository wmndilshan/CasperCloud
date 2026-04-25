package repository

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrNoIPsAvailable is returned when the network pool is exhausted.
var ErrNoIPsAvailable = errors.New("no free ip addresses in network")

// Network describes a project L3 network attached to a libvirt bridge.
type Network struct {
	ID         uuid.UUID `json:"id"`
	ProjectID  uuid.UUID `json:"project_id"`
	Name       string    `json:"name"`
	CIDR       string    `json:"cidr"`
	Gateway    string    `json:"gateway"`
	BridgeName string    `json:"bridge_name"`
	IsDefault  bool      `json:"is_default"`
}

// InsertDefaultNetwork creates the default network row for a project (call inside a transaction).
func (r *Repository) InsertDefaultNetwork(ctx context.Context, tx pgx.Tx, projectID uuid.UUID, cidr, gateway, bridge string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO networks (project_id, name, cidr, gateway, bridge_name, is_default)
		VALUES ($1, 'default', $2::cidr, $3::inet, $4, true)
		RETURNING id`, projectID, cidr, gateway, bridge).Scan(&id)
	return id, err
}

// SeedNetworkIPSlots inserts one row per usable IPv4 host in the prefix (excludes network, broadcast, gateway).
func SeedNetworkIPSlots(ctx context.Context, tx pgx.Tx, networkID uuid.UUID, prefix netip.Prefix, gateway netip.Addr) error {
	if !prefix.IsValid() || !prefix.Addr().Is4() {
		return fmt.Errorf("only valid IPv4 prefixes are supported")
	}
	if !gateway.Is4() {
		return fmt.Errorf("gateway must be IPv4")
	}
	hosts, err := usableIPv4Hosts(prefix, gateway)
	if err != nil {
		return err
	}
	if len(hosts) == 0 {
		return fmt.Errorf("no usable host addresses in prefix %s", prefix)
	}
	batch := &pgx.Batch{}
	for _, ip := range hosts {
		batch.Queue(`INSERT INTO network_ip_slots (network_id, ip_address) VALUES ($1, $2::inet) ON CONFLICT DO NOTHING`, networkID, ip.String())
	}
	br := tx.SendBatch(ctx, batch)
	for range hosts {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return err
		}
	}
	return br.Close()
}

func usableIPv4Hosts(prefix netip.Prefix, gateway netip.Addr) ([]netip.Addr, error) {
	bits := prefix.Bits()
	if bits < 8 || bits > 30 {
		return nil, fmt.Errorf("prefix /%d not supported for automatic slotting (use /8..30)", bits)
	}
	networkU32 := ipv4ToUint32(prefix.Masked().Addr())
	hostCount := uint32(1<<(32-bits)) - 2 // exclude network and broadcast
	if hostCount < 1 {
		return nil, fmt.Errorf("prefix too small")
	}
	gwU := ipv4ToUint32(gateway)
	var out []netip.Addr
	for i := uint32(1); i <= hostCount; i++ {
		ipU := networkU32 + i
		if ipU == gwU {
			continue
		}
		out = append(out, uint32ToIPv4(ipU))
	}
	return out, nil
}

func ipv4ToUint32(a netip.Addr) uint32 {
	b := a.As4()
	return binary.BigEndian.Uint32(b[:])
}

func uint32ToIPv4(u uint32) netip.Addr {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], u)
	return netip.AddrFrom4(b)
}

// GetNetworkForProject returns a network owned by the project (or ErrNotFound).
func (r *Repository) GetNetworkForProject(ctx context.Context, projectID, networkID uuid.UUID) (*Network, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, cidr::text, gateway::text, bridge_name, is_default
		FROM networks
		WHERE id = $1 AND project_id = $2`, networkID, projectID)
	var n Network
	var cidr, gw string
	if err := row.Scan(&n.ID, &n.ProjectID, &n.Name, &cidr, &gw, &n.BridgeName, &n.IsDefault); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	n.CIDR = cidr
	n.Gateway = gw
	return &n, nil
}

// GetDefaultNetworkID returns the default network id for a project.
func (r *Repository) GetDefaultNetworkID(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM networks WHERE project_id = $1 AND is_default = true LIMIT 1`, projectID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	return id, nil
}

// ListNetworksForProject returns all networks for a project (default first).
func (r *Repository) ListNetworksForProject(ctx context.Context, projectID uuid.UUID) ([]Network, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, name, cidr::text, gateway::text, bridge_name, is_default
		FROM networks
		WHERE project_id = $1
		ORDER BY is_default DESC, name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Network
	for rows.Next() {
		var n Network
		var cidr, gw string
		if err := rows.Scan(&n.ID, &n.ProjectID, &n.Name, &cidr, &gw, &n.BridgeName, &n.IsDefault); err != nil {
			return nil, err
		}
		n.CIDR = cidr
		n.Gateway = gw
		out = append(out, n)
	}
	return out, rows.Err()
}

// ClaimNextIPSlot reserves the next free IP using SELECT FOR UPDATE SKIP LOCKED.
func ClaimNextIPSlot(ctx context.Context, tx pgx.Tx, networkID, instanceID uuid.UUID) (netip.Addr, error) {
	var ipStr string
	err := tx.QueryRow(ctx, `
WITH c AS (
	SELECT id, ip_address
	FROM network_ip_slots
	WHERE network_id = $1 AND instance_id IS NULL
	ORDER BY ip_address
	FOR UPDATE SKIP LOCKED
	LIMIT 1
)
UPDATE network_ip_slots s
SET instance_id = $2
FROM c
WHERE s.id = c.id
RETURNING s.ip_address::text`, networkID, instanceID).Scan(&ipStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return netip.Addr{}, ErrNoIPsAvailable
		}
		return netip.Addr{}, err
	}
	return netip.ParseAddr(ipStr)
}
