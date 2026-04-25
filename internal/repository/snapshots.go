package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrSnapshotStateConflict means the instance is not in a state that allows this snapshot operation.
var ErrSnapshotStateConflict = errors.New("instance state does not allow this snapshot operation")

// ErrSnapshotNameConflict is returned when an instance already has a snapshot with the same name.
var ErrSnapshotNameConflict = errors.New("snapshot name already exists for this instance")

func isSnapshotUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Snapshot row for QCOW2 internal snapshots.
type Snapshot struct {
	ID               uuid.UUID `json:"id"`
	ProjectID        uuid.UUID `json:"project_id"`
	InstanceID       uuid.UUID `json:"instance_id"`
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	DomainWasRunning bool      `json:"domain_was_running"`
	CreatedAt        time.Time `json:"created_at"`
}

const (
	SnapshotStatusCreating  = "creating"
	SnapshotStatusAvailable = "available"
	SnapshotStatusError     = "error"
)

// LibvirtSnapshotName returns the libvirt snapshot name (stable UUID string).
func (s Snapshot) LibvirtSnapshotName() string {
	return s.ID.String()
}

func (r *Repository) UpdateSnapshotStatus(ctx context.Context, projectID, snapshotID uuid.UUID, status string) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE snapshots SET status = $3
		WHERE id = $2 AND project_id = $1`, projectID, snapshotID, status)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListSnapshotsForInstance(ctx context.Context, projectID, instanceID uuid.UUID) ([]Snapshot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, instance_id, name, status, domain_was_running, created_at
		FROM snapshots
		WHERE project_id = $1 AND instance_id = $2
		ORDER BY created_at DESC`, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Snapshot
	for rows.Next() {
		var s Snapshot
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.InstanceID, &s.Name, &s.Status, &s.DomainWasRunning, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// BeginSnapshotCreate inserts the snapshot row and sets instance state to snapshotting in one transaction.
// priorState is "running" or "stopped" for worker restore.
func (r *Repository) BeginSnapshotCreate(ctx context.Context, projectID, instanceID, snapshotID uuid.UUID, name string) (priorState string, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var state string
	err = tx.QueryRow(ctx, `
		SELECT state FROM instances WHERE project_id = $1 AND id = $2 FOR UPDATE`,
		projectID, instanceID).Scan(&state)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if state != "running" && state != "stopped" {
		return "", ErrSnapshotStateConflict
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO snapshots (id, project_id, instance_id, name, status, domain_was_running)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		snapshotID, projectID, instanceID, name, SnapshotStatusCreating, state == "running"); err != nil {
		if isSnapshotUniqueViolation(err) {
			return "", ErrSnapshotNameConflict
		}
		return "", err
	}
	cmd, err := tx.Exec(ctx, `
		UPDATE instances SET state = 'snapshotting', updated_at = now()
		WHERE project_id = $1 AND id = $2 AND state = $3`, projectID, instanceID, state)
	if err != nil {
		return "", err
	}
	if cmd.RowsAffected() == 0 {
		return "", ErrSnapshotStateConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return state, nil
}

// BeginSnapshotRevert sets instance to snapshotting if it is running or stopped (for revert job).
func (r *Repository) BeginSnapshotRevert(ctx context.Context, projectID, instanceID uuid.UUID) (priorState string, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var state string
	err = tx.QueryRow(ctx, `
		SELECT state FROM instances WHERE project_id = $1 AND id = $2 FOR UPDATE`,
		projectID, instanceID).Scan(&state)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if state != "running" && state != "stopped" {
		return "", ErrSnapshotStateConflict
	}
	cmd, err := tx.Exec(ctx, `
		UPDATE instances SET state = 'snapshotting', updated_at = now()
		WHERE project_id = $1 AND id = $2 AND state = $3`, projectID, instanceID, state)
	if err != nil {
		return "", err
	}
	if cmd.RowsAffected() == 0 {
		return "", ErrSnapshotStateConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return state, nil
}

func (r *Repository) GetSnapshot(ctx context.Context, projectID, instanceID, snapshotID uuid.UUID) (*Snapshot, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, instance_id, name, status, domain_was_running, created_at
		FROM snapshots
		WHERE project_id = $1 AND instance_id = $2 AND id = $3`, projectID, instanceID, snapshotID)
	var s Snapshot
	if err := row.Scan(&s.ID, &s.ProjectID, &s.InstanceID, &s.Name, &s.Status, &s.DomainWasRunning, &s.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}
