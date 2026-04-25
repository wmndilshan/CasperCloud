package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// InstanceSyncRow is a minimal row for hypervisor ↔ Postgres reconciliation.
type InstanceSyncRow struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	State     string
}

// InstanceStateUpdate is one batched state write.
type InstanceStateUpdate struct {
	ProjectID  uuid.UUID
	InstanceID uuid.UUID
	NewState   string
}

// ListInstancesForHypervisorSync returns instances that may have a libvirt domain (excludes terminal delete flow).
func (r *Repository) ListInstancesForHypervisorSync(ctx context.Context) ([]InstanceSyncRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, name, state
		FROM instances
		WHERE state != 'deleting'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []InstanceSyncRow
	for rows.Next() {
		var row InstanceSyncRow
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.Name, &row.State); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

const stateSyncBatchSize = 128

// BatchUpdateInstanceStates applies updates in chunks of stateSyncBatchSize using pgx.Batch.
func (r *Repository) BatchUpdateInstanceStates(ctx context.Context, updates []InstanceStateUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	for i := 0; i < len(updates); i += stateSyncBatchSize {
		end := i + stateSyncBatchSize
		if end > len(updates) {
			end = len(updates)
		}
		batch := &pgx.Batch{}
		for _, u := range updates[i:end] {
			batch.Queue(`
				UPDATE instances
				SET state = $3, updated_at = now()
				WHERE project_id = $1 AND id = $2 AND state != $3`,
				u.ProjectID, u.InstanceID, u.NewState)
		}
		br := r.pool.SendBatch(ctx, batch)
		for range updates[i:end] {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return err
			}
		}
		if err := br.Close(); err != nil {
			return err
		}
	}
	return nil
}

// DeleteInstanceTx deletes an instance row inside an existing transaction (IP slots / allocations follow FK rules).
func (r *Repository) DeleteInstanceTx(ctx context.Context, tx pgx.Tx, projectID, instanceID uuid.UUID) error {
	cmd, err := tx.Exec(ctx, `
		DELETE FROM instances
		WHERE project_id = $1 AND id = $2`, projectID, instanceID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FinalizeInstanceDestroy marks the destroy task succeeded and deletes the instance in one transaction.
// CASCADE on ip_allocations and SET NULL on network_ip_slots.instance_id release the IP back to the pool.
func (r *Repository) FinalizeInstanceDestroy(ctx context.Context, projectID, taskID, instanceID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `
		UPDATE tasks
		SET status = 'succeeded', error = NULL, updated_at = now()
		WHERE project_id = $1 AND id = $2`, projectID, taskID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err := r.DeleteInstanceTx(ctx, tx, projectID, instanceID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
