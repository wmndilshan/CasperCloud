package repository

import (
	"context"

	"github.com/google/uuid"
)

// RunningInstanceRow is a minimal row for the metrics poller.
type RunningInstanceRow struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
}

// ListRunningInstancesForMetrics returns instances the control plane considers running (libvirt may still disagree; state sync fixes drift).
func (r *Repository) ListRunningInstancesForMetrics(ctx context.Context) ([]RunningInstanceRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, name
		FROM instances
		WHERE state = 'running'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RunningInstanceRow
	for rows.Next() {
		var row RunningInstanceRow
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.Name); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
