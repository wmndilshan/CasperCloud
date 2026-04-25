package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type rowScanner interface {
	Scan(dest ...any) error
}

// ErrFloatingIPAssociatePreconditions is returned when the floating IP or instance cannot be bound atomically.
var ErrFloatingIPAssociatePreconditions = errors.New("floating ip cannot be associated (check ownership, state=allocated, instance running with ipv4)")

// FloatingIPNATBinding is a public→private pair used for host NAT teardown.
type FloatingIPNATBinding struct {
	PublicIP  string
	PrivateIP string
}

func scanFloatingIPRow(row rowScanner) (*FloatingIP, error) {
	var f FloatingIP
	var projID *uuid.UUID
	var instID *uuid.UUID
	var priv *string
	if err := row.Scan(&f.ID, &projID, &f.PublicIP, &instID, &priv, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}
	f.ProjectID = projID
	f.InstanceID = instID
	f.PrivateIP = priv
	return &f, nil
}

// AllocateFloatingIPToProject claims one unallocated address for the project (SKIP LOCKED).
func (r *Repository) AllocateFloatingIPToProject(ctx context.Context, projectID uuid.UUID) (*FloatingIP, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE floating_ips f
		SET project_id = $1,
		    status = 'allocated',
		    updated_at = now()
		FROM (
			SELECT id FROM floating_ips
			WHERE status = 'unallocated'
			ORDER BY public_ip
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		) pick
		WHERE f.id = pick.id
		RETURNING f.id, f.project_id, f.public_ip::text, f.instance_id, f.private_ip::text, f.status, f.created_at, f.updated_at`,
		projectID)
	f, err := scanFloatingIPRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoFloatingIPsAvailable
		}
		return nil, err
	}
	return f, nil
}

// ListFloatingIPsForProject returns all floating IPs owned by the project.
func (r *Repository) ListFloatingIPsForProject(ctx context.Context, projectID uuid.UUID) ([]FloatingIP, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, public_ip::text, instance_id, private_ip::text, status, created_at, updated_at
		FROM floating_ips
		WHERE project_id = $1
		ORDER BY public_ip`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FloatingIP
	for rows.Next() {
		f, err := scanFloatingIPRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

// GetFloatingIPForProject returns a floating IP row owned by the project.
func (r *Repository) GetFloatingIPForProject(ctx context.Context, projectID, floatingIPID uuid.UUID) (*FloatingIP, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, public_ip::text, instance_id, private_ip::text, status, created_at, updated_at
		FROM floating_ips
		WHERE id = $1 AND project_id = $2`, floatingIPID, projectID)
	f, err := scanFloatingIPRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// TryAssociateFloatingIP binds an allocated floating IP to a running instance and its IPv4 in one statement.
func (r *Repository) TryAssociateFloatingIP(ctx context.Context, projectID, floatingIPID, instanceID uuid.UUID) (*FloatingIP, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE floating_ips f
		SET instance_id = i.id,
		    private_ip = i.ipv4_address,
		    status = 'active',
		    updated_at = now()
		FROM instances i
		WHERE f.id = $2
		  AND f.project_id = $1
		  AND f.status = 'allocated'
		  AND f.instance_id IS NULL
		  AND i.project_id = $1
		  AND i.id = $3
		  AND i.state = 'running'
		  AND i.ipv4_address IS NOT NULL
		RETURNING f.id, f.project_id, f.public_ip::text, f.instance_id, f.private_ip::text, f.status, f.created_at, f.updated_at`,
		projectID, floatingIPID, instanceID)
	f, err := scanFloatingIPRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFloatingIPAssociatePreconditions
		}
		return nil, err
	}
	return f, nil
}

// TakeDisassociateFloatingIP locks an active floating IP, clears the bind, and returns the NAT pair for iptables removal.
func (r *Repository) TakeDisassociateFloatingIP(ctx context.Context, projectID, floatingIPID uuid.UUID) (publicIP, privateIP string, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx)

	var pub, priv *string
	err = tx.QueryRow(ctx, `
		SELECT public_ip::text, private_ip::text
		FROM floating_ips
		WHERE id = $1 AND project_id = $2 AND status = 'active'
		FOR UPDATE`,
		floatingIPID, projectID).Scan(&pub, &priv)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrNotFound
		}
		return "", "", err
	}
	if pub == nil || priv == nil || *pub == "" || *priv == "" {
		return "", "", fmt.Errorf("floating ip %s missing public or private address", floatingIPID)
	}
	cmd, err := tx.Exec(ctx, `
		UPDATE floating_ips
		SET instance_id = NULL,
		    private_ip = NULL,
		    status = 'allocated',
		    updated_at = now()
		WHERE id = $1 AND project_id = $2`, floatingIPID, projectID)
	if err != nil {
		return "", "", err
	}
	if cmd.RowsAffected() == 0 {
		return "", "", ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return *pub, *priv, nil
}

// ListActiveFloatingIPNATBindingsByInstance returns active public/private pairs for an instance (worker destroy path).
func (r *Repository) ListActiveFloatingIPNATBindingsByInstance(ctx context.Context, projectID, instanceID uuid.UUID) ([]FloatingIPNATBinding, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT public_ip::text, private_ip::text
		FROM floating_ips
		WHERE project_id = $1 AND instance_id = $2 AND status = 'active' AND private_ip IS NOT NULL`,
		projectID, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FloatingIPNATBinding
	for rows.Next() {
		var b FloatingIPNATBinding
		if err := rows.Scan(&b.PublicIP, &b.PrivateIP); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ListFloatingIPsActiveForReconcile returns bindings that should have ingress NAT rules applied.
// Excludes instances in deleting (worker tears down NAT before the row is cleared).
func (r *Repository) ListFloatingIPsActiveForReconcile(ctx context.Context) ([]FloatingIPNATBinding, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT f.public_ip::text, f.private_ip::text
		FROM floating_ips f
		INNER JOIN instances i ON i.id = f.instance_id AND i.project_id = f.project_id
		WHERE f.status = 'active'
		  AND f.private_ip IS NOT NULL
		  AND i.state = 'running'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FloatingIPNATBinding
	for rows.Next() {
		var b FloatingIPNATBinding
		if err := rows.Scan(&b.PublicIP, &b.PrivateIP); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// RollbackFloatingIPAssociation reverts an active bind to allocated (used when enqueue fails after TryAssociate).
func (r *Repository) RollbackFloatingIPAssociation(ctx context.Context, projectID, floatingIPID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE floating_ips
		SET instance_id = NULL,
		    private_ip = NULL,
		    status = 'allocated',
		    updated_at = now()
		WHERE id = $2 AND project_id = $1 AND status = 'active'`, projectID, floatingIPID)
	return err
}

// ClearFloatingIPBindingsForInstance clears active bindings after host NAT has been removed (instance destroy path).
func (r *Repository) ClearFloatingIPBindingsForInstance(ctx context.Context, projectID, instanceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE floating_ips
		SET instance_id = NULL,
		    private_ip = NULL,
		    status = CASE WHEN status = 'active' THEN 'allocated' ELSE status END,
		    updated_at = now()
		WHERE project_id = $1 AND instance_id = $2`, projectID, instanceID)
	return err
}
