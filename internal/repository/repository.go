package repository

import (
	"context"
	"database/sql"
	"errors"
	"net/netip"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound               = errors.New("not found")
	ErrNoFloatingIPsAvailable = errors.New("no floating ips available in pool")
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at`, email, passwordHash)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE email = $1`, email)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE id = $1`, userID)
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CreateProject(ctx context.Context, ownerID uuid.UUID, name, defaultNetCIDR, defaultNetGateway, defaultBridge string) (*Project, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var p Project
	row := tx.QueryRow(ctx, `
		INSERT INTO projects (name, owner_id)
		VALUES ($1, $2)
		RETURNING id, name, owner_id, created_at`, name, ownerID)
	if err := row.Scan(&p.ID, &p.Name, &p.OwnerID, &p.CreatedAt); err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO project_memberships (project_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT DO NOTHING`, p.ID, ownerID)
	if err != nil {
		return nil, err
	}

	netID, err := r.InsertDefaultNetwork(ctx, tx, p.ID, defaultNetCIDR, defaultNetGateway, defaultBridge)
	if err != nil {
		return nil, err
	}
	pfx, err := netip.ParsePrefix(defaultNetCIDR)
	if err != nil {
		return nil, err
	}
	gw, err := netip.ParseAddr(defaultNetGateway)
	if err != nil {
		return nil, err
	}
	if err := SeedNetworkIPSlots(ctx, tx, netID, pfx, gw); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) ListProjectsForUser(ctx context.Context, userID uuid.UUID) ([]Project, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.name, p.owner_id, p.created_at
		FROM projects p
		JOIN project_memberships m ON m.project_id = p.id
		WHERE m.user_id = $1
		ORDER BY p.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]Project, 0)
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.OwnerID, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *Repository) UserHasProjectAccess(ctx context.Context, userID, projectID uuid.UUID) (bool, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM project_memberships
			WHERE project_id = $1 AND user_id = $2
		)`, projectID, userID)
	var hasAccess bool
	if err := row.Scan(&hasAccess); err != nil {
		return false, err
	}
	return hasAccess, nil
}

func (r *Repository) CreateImage(ctx context.Context, projectID uuid.UUID, name, sourceURL, description string) (*Image, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO images (project_id, name, source_url, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, project_id, name, source_url, description, created_at`, projectID, name, sourceURL, description)
	var img Image
	if err := row.Scan(&img.ID, &img.ProjectID, &img.Name, &img.SourceURL, &img.Description, &img.CreatedAt); err != nil {
		return nil, err
	}
	return &img, nil
}

func (r *Repository) GetImage(ctx context.Context, projectID, imageID uuid.UUID) (*Image, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, source_url, description, created_at
		FROM images
		WHERE project_id = $1 AND id = $2`, projectID, imageID)
	var img Image
	if err := row.Scan(&img.ID, &img.ProjectID, &img.Name, &img.SourceURL, &img.Description, &img.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &img, nil
}

func (r *Repository) ListImages(ctx context.Context, projectID uuid.UUID) ([]Image, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, name, source_url, description, created_at
		FROM images
		WHERE project_id = $1
		ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := make([]Image, 0)
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.ProjectID, &img.Name, &img.SourceURL, &img.Description, &img.CreatedAt); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

func (r *Repository) UpdateImage(ctx context.Context, projectID, imageID uuid.UUID, name, sourceURL, description string) (*Image, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE images
		SET name = $3,
		    source_url = $4,
		    description = $5
		WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, name, source_url, description, created_at`, projectID, imageID, name, sourceURL, description)
	var img Image
	if err := row.Scan(&img.ID, &img.ProjectID, &img.Name, &img.SourceURL, &img.Description, &img.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &img, nil
}

func (r *Repository) DeleteImage(ctx context.Context, projectID, imageID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `
		DELETE FROM images
		WHERE project_id = $1 AND id = $2`, projectID, imageID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) GetInstance(ctx context.Context, projectID, instanceID uuid.UUID) (*Instance, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, image_id, name, state, cloud_init_data, network_id, mac_address, bridge_name, ipv4_address::text, network_config_yaml, created_at, updated_at
		FROM instances
		WHERE project_id = $1 AND id = $2`, projectID, instanceID)
	var inst Instance
	var netID sql.NullString
	if err := row.Scan(&inst.ID, &inst.ProjectID, &inst.ImageID, &inst.Name, &inst.State, &inst.CloudInitData, &netID, &inst.MACAddress, &inst.BridgeName, &inst.IPv4Address, &inst.NetworkConfigYAML, &inst.CreatedAt, &inst.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if netID.Valid {
		u, err := uuid.Parse(netID.String)
		if err == nil {
			inst.NetworkID = &u
		}
	}
	return &inst, nil
}

func (r *Repository) ListInstances(ctx context.Context, projectID uuid.UUID) ([]Instance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, image_id, name, state, cloud_init_data, network_id, mac_address, bridge_name, ipv4_address::text, network_config_yaml, created_at, updated_at
		FROM instances
		WHERE project_id = $1
		ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	instances := make([]Instance, 0)
	for rows.Next() {
		var inst Instance
		var netID sql.NullString
		if err := rows.Scan(&inst.ID, &inst.ProjectID, &inst.ImageID, &inst.Name, &inst.State, &inst.CloudInitData, &netID, &inst.MACAddress, &inst.BridgeName, &inst.IPv4Address, &inst.NetworkConfigYAML, &inst.CreatedAt, &inst.UpdatedAt); err != nil {
			return nil, err
		}
		if netID.Valid {
			u, err := uuid.Parse(netID.String)
			if err == nil {
				inst.NetworkID = &u
			}
		}
		instances = append(instances, inst)
	}
	return instances, rows.Err()
}

func (r *Repository) UpdateInstanceState(ctx context.Context, projectID, instanceID uuid.UUID, state string) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE instances
		SET state = $3, updated_at = now()
		WHERE project_id = $1 AND id = $2`, projectID, instanceID, state)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteInstance(ctx context.Context, projectID, instanceID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `
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

func scanTaskRow(row pgx.Row) (*Task, error) {
	var t Task
	if err := row.Scan(&t.ID, &t.Type, &t.ProjectID, &t.InstanceID, &t.Status, &t.Error, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) CreateTask(ctx context.Context, taskType string, projectID, instanceID uuid.UUID, status string) (*Task, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO tasks (type, project_id, instance_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, type, project_id, instance_id, status, error, created_at, updated_at`,
		taskType, projectID, instanceID, status)
	return scanTaskRow(row)
}

// CreateTaskWithoutInstance inserts a task row with NULL instance_id (e.g. floating_ip.disassociate).
func (r *Repository) CreateTaskWithoutInstance(ctx context.Context, taskType string, projectID uuid.UUID, status string) (*Task, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO tasks (type, project_id, instance_id, status)
		VALUES ($1, $2, NULL, $3)
		RETURNING id, type, project_id, instance_id, status, error, created_at, updated_at`,
		taskType, projectID, status)
	return scanTaskRow(row)
}

func (r *Repository) UpdateTaskStatus(ctx context.Context, projectID, taskID uuid.UUID, status string, errMessage *string) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE tasks
		SET status = $3,
		    error = $4,
		    updated_at = now()
		WHERE id = $2 AND project_id = $1`, projectID, taskID, status, errMessage)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) GetTask(ctx context.Context, projectID, taskID uuid.UUID) (*Task, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, type, project_id, instance_id, status, error, created_at, updated_at
		FROM tasks
		WHERE id = $2 AND project_id = $1`, projectID, taskID)
	t, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

const staleRunningTaskMessage = "reconciled: task stuck in running for more than 30 minutes"

// ReconcileStaleRunningTasks marks tasks that have been running too long as failed and sets
// their instances to error. One round-trip: a CTE updates tasks, then instances in the same statement.
func (r *Repository) ReconcileStaleRunningTasks(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
WITH timed_out AS (
	UPDATE tasks
	SET status = 'failed',
	    error = $1,
	    updated_at = now()
	WHERE status = 'running'
	  AND updated_at < now() - interval '30 minutes'
	RETURNING project_id, instance_id
)
UPDATE instances AS i
SET state = 'error',
    updated_at = now()
FROM timed_out t
WHERE i.project_id = t.project_id
  AND i.id = t.instance_id`, staleRunningTaskMessage)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
