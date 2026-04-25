package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/netip"

	"caspercloud/internal/cloudinit"

	"github.com/google/uuid"
)

// CreateInstanceParams carries everything needed to create an instance with static networking.
type CreateInstanceParams struct {
	InstanceID     uuid.UUID
	ProjectID      uuid.UUID
	ImageID        uuid.UUID
	Name           string
	UserData       string
	InitialState   string
	NetworkID      uuid.UUID
	NetworkCIDR    string
	NetworkGateway string
	BridgeName     string
	MAC            string
}

// CreateInstanceWithIPAM creates an instance row, ensures IP slots, claims an address (SKIP LOCKED),
// writes network-config YAML, and records ip_allocations — all in one transaction.
func (r *Repository) CreateInstanceWithIPAM(ctx context.Context, p CreateInstanceParams) (*Instance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var slotCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*)::int FROM network_ip_slots WHERE network_id = $1`, p.NetworkID).Scan(&slotCount); err != nil {
		return nil, err
	}
	if slotCount == 0 {
		pfx, err := netip.ParsePrefix(p.NetworkCIDR)
		if err != nil {
			return nil, fmt.Errorf("parse network cidr: %w", err)
		}
		gw, err := netip.ParseAddr(p.NetworkGateway)
		if err != nil {
			return nil, fmt.Errorf("parse gateway: %w", err)
		}
		if err := SeedNetworkIPSlots(ctx, tx, p.NetworkID, pfx, gw); err != nil {
			return nil, err
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO instances (id, project_id, image_id, name, state, cloud_init_data, network_id, mac_address, bridge_name, ipv4_address, network_config_yaml)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULL, '')`,
		p.InstanceID, p.ProjectID, p.ImageID, p.Name, p.InitialState, p.UserData, p.NetworkID, p.MAC, p.BridgeName)
	if err != nil {
		return nil, err
	}

	ip, err := ClaimNextIPSlot(ctx, tx, p.NetworkID, p.InstanceID)
	if err != nil {
		return nil, err
	}

	gw, err := netip.ParseAddr(p.NetworkGateway)
	if err != nil {
		return nil, err
	}
	pfx, err := netip.ParsePrefix(p.NetworkCIDR)
	if err != nil {
		return nil, err
	}
	netYAML, err := cloudinit.RenderNetworkConfigV2(p.MAC, ip, gw, pfx.Bits())
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE instances SET ipv4_address = $1::inet, network_config_yaml = $2 WHERE id = $3 AND project_id = $4`,
		ip.String(), netYAML, p.InstanceID, p.ProjectID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO ip_allocations (network_id, project_id, instance_id, ip_address)
		VALUES ($1, $2, $3, $4::inet)`, p.NetworkID, p.ProjectID, p.InstanceID, ip.String()); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
		SELECT id, project_id, image_id, name, state, cloud_init_data, network_id, mac_address, bridge_name, ipv4_address::text, network_config_yaml, created_at, updated_at
		FROM instances WHERE id = $1 AND project_id = $2`, p.InstanceID, p.ProjectID)
	var inst Instance
	var netID sql.NullString
	var macPtr *string
	var ipv4 *string
	if err := row.Scan(&inst.ID, &inst.ProjectID, &inst.ImageID, &inst.Name, &inst.State, &inst.CloudInitData, &netID, &macPtr, &inst.BridgeName, &ipv4, &inst.NetworkConfigYAML, &inst.CreatedAt, &inst.UpdatedAt); err != nil {
		return nil, err
	}
	if netID.Valid {
		u, err := uuid.Parse(netID.String)
		if err == nil {
			inst.NetworkID = &u
		}
	}
	inst.MACAddress = macPtr
	inst.IPv4Address = ipv4

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &inst, nil
}
