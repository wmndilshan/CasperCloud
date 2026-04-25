-- Block volumes: sparse qcow2 files, attachable to instances (survive instance deletion when detached).

CREATE TABLE IF NOT EXISTS volumes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    size_gb INT NOT NULL CHECK (size_gb > 0 AND size_gb <= 4096),
    status TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'in-use', 'error')),
    instance_id UUID REFERENCES instances(id) ON DELETE RESTRICT,
    target_dev TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, name)
);

CREATE INDEX IF NOT EXISTS idx_volumes_project_id ON volumes (project_id);
CREATE INDEX IF NOT EXISTS idx_volumes_instance_id ON volumes (instance_id);

COMMENT ON TABLE volumes IS 'Persistent QCOW2 volumes; instance_id set only while attached (ON DELETE RESTRICT prevents destroying an instance until volumes are detached).';

DROP TRIGGER IF EXISTS trg_volumes_updated_at ON volumes;
CREATE TRIGGER trg_volumes_updated_at
BEFORE UPDATE ON volumes
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
