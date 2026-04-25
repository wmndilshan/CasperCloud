package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrVolumeInUse = errors.New("volume is attached to an instance")

// Volume is a persistent block volume in a project.
type Volume struct {
	ID         uuid.UUID  `json:"id"`
	ProjectID  uuid.UUID  `json:"project_id"`
	Name       string     `json:"name"`
	SizeGB     int        `json:"size_gb"`
	Status     string     `json:"status"`
	InstanceID *uuid.UUID `json:"instance_id,omitempty"`
	TargetDev  *string    `json:"target_dev,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func scanVolume(row pgx.Row) (*Volume, error) {
	var v Volume
	var instID sql.NullString
	var tgt sql.NullString
	if err := row.Scan(&v.ID, &v.ProjectID, &v.Name, &v.SizeGB, &v.Status, &instID, &tgt, &v.CreatedAt, &v.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if instID.Valid {
		u, err := uuid.Parse(instID.String)
		if err == nil {
			v.InstanceID = &u
		}
	}
	if tgt.Valid {
		s := tgt.String
		v.TargetDev = &s
	}
	return &v, nil
}

func (r *Repository) CreateVolume(ctx context.Context, projectID uuid.UUID, id uuid.UUID, name string, sizeGB int, status string) (*Volume, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO volumes (id, project_id, name, size_gb, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, project_id, name, size_gb, status, instance_id, target_dev, created_at, updated_at`,
		id, projectID, name, sizeGB, status)
	return scanVolume(row)
}

func (r *Repository) GetVolume(ctx context.Context, projectID, volumeID uuid.UUID) (*Volume, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, size_gb, status, instance_id, target_dev, created_at, updated_at
		FROM volumes WHERE project_id = $1 AND id = $2`, projectID, volumeID)
	return scanVolume(row)
}

func (r *Repository) ListVolumes(ctx context.Context, projectID uuid.UUID) ([]Volume, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, name, size_gb, status, instance_id, target_dev, created_at, updated_at
		FROM volumes WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Volume
	for rows.Next() {
		var v Volume
		var instID sql.NullString
		var tgt sql.NullString
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.Name, &v.SizeGB, &v.Status, &instID, &tgt, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		if instID.Valid {
			u, err := uuid.Parse(instID.String)
			if err == nil {
				v.InstanceID = &u
			}
		}
		if tgt.Valid {
			s := tgt.String
			v.TargetDev = &s
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteVolume removes a volume row only when it is not attached.
func (r *Repository) DeleteVolume(ctx context.Context, projectID, volumeID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `
		DELETE FROM volumes
		WHERE project_id = $1 AND id = $2 AND instance_id IS NULL`, projectID, volumeID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		var attached bool
		_ = r.pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM volumes WHERE project_id = $1 AND id = $2 AND instance_id IS NOT NULL)`,
			projectID, volumeID).Scan(&attached)
		if attached {
			return ErrVolumeInUse
		}
		return ErrNotFound
	}
	return nil
}

// CountVolumesAttachedToInstance returns how many volumes reference the instance.
func (r *Repository) CountVolumesAttachedToInstance(ctx context.Context, projectID, instanceID uuid.UUID) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM volumes
		WHERE project_id = $1 AND instance_id = $2`, projectID, instanceID).Scan(&n)
	return n, err
}

// ListVolumesAttachedToInstance returns volumes currently attached to the instance.
func (r *Repository) ListVolumesAttachedToInstance(ctx context.Context, projectID, instanceID uuid.UUID) ([]Volume, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, name, size_gb, status, instance_id, target_dev, created_at, updated_at
		FROM volumes WHERE project_id = $1 AND instance_id = $2`, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Volume
	for rows.Next() {
		var v Volume
		var instID sql.NullString
		var tgt sql.NullString
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.Name, &v.SizeGB, &v.Status, &instID, &tgt, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		if instID.Valid {
			u, err := uuid.Parse(instID.String)
			if err == nil {
				v.InstanceID = &u
			}
		}
		if tgt.Valid {
			s := tgt.String
			v.TargetDev = &s
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// AttachVolumeRecord sets in-use state and target after libvirt attach succeeds.
func (r *Repository) AttachVolumeRecord(ctx context.Context, projectID, volumeID, instanceID uuid.UUID, targetDev string) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE volumes
		SET instance_id = $3, target_dev = $4, status = 'in-use', updated_at = now()
		WHERE project_id = $1 AND id = $2 AND status = 'available' AND instance_id IS NULL`,
		projectID, volumeID, instanceID, targetDev)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DetachVolumeRecord clears attachment for a volume bound to the given instance.
func (r *Repository) DetachVolumeRecord(ctx context.Context, projectID, instanceID, volumeID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE volumes
		SET instance_id = NULL, target_dev = NULL, status = 'available', updated_at = now()
		WHERE project_id = $1 AND id = $2 AND instance_id = $3`,
		projectID, volumeID, instanceID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetVolumeError marks provisioning or operational failure.
func (r *Repository) SetVolumeError(ctx context.Context, projectID, volumeID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE volumes SET status = 'error', updated_at = now()
		WHERE project_id = $1 AND id = $2`, projectID, volumeID)
	return err
}
