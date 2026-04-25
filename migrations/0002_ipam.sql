-- IPAM: per-project networks, pre-seeded assignable IPs (SKIP LOCKED), and allocation audit rows.

CREATE TABLE IF NOT EXISTS networks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    cidr CIDR NOT NULL,
    gateway INET NOT NULL,
    bridge_name TEXT NOT NULL DEFAULT 'virbr0',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, name)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_networks_one_default_per_project
    ON networks (project_id)
    WHERE is_default;

-- One row per assignable host address; instance_id set when claimed (concurrent-safe via SKIP LOCKED).
-- FK to instances is DEFERRABLE so the same transaction can INSERT the instance then attach the slot before COMMIT.
CREATE TABLE IF NOT EXISTS network_ip_slots (
    id BIGSERIAL PRIMARY KEY,
    network_id UUID NOT NULL REFERENCES networks(id) ON DELETE CASCADE,
    ip_address INET NOT NULL,
    instance_id UUID REFERENCES instances(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (network_id, ip_address)
);

CREATE INDEX IF NOT EXISTS idx_network_ip_slots_available
    ON network_ip_slots (network_id)
    WHERE instance_id IS NULL;

CREATE TABLE IF NOT EXISTS ip_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    network_id UUID NOT NULL REFERENCES networks(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    instance_id UUID NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    ip_address INET NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (network_id, ip_address),
    UNIQUE (instance_id)
);

ALTER TABLE instances
    ADD COLUMN IF NOT EXISTS network_id UUID REFERENCES networks(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS mac_address TEXT,
    ADD COLUMN IF NOT EXISTS ipv4_address INET,
    ADD COLUMN IF NOT EXISTS network_config_yaml TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bridge_name TEXT NOT NULL DEFAULT '';

INSERT INTO networks (project_id, name, cidr, gateway, is_default, bridge_name)
SELECT p.id, 'default', '192.168.122.0/24'::cidr, '192.168.122.1'::inet, true, 'virbr0'
FROM projects p
WHERE NOT EXISTS (SELECT 1 FROM networks n WHERE n.project_id = p.id);

CREATE INDEX IF NOT EXISTS idx_instances_network_id ON instances (network_id);

COMMENT ON TABLE network_ip_slots IS 'Pre-seeded pool; claim with UPDATE ... FROM (SELECT ... FOR UPDATE SKIP LOCKED) to avoid duplicate IPs under concurrency.';
COMMENT ON TABLE ip_allocations IS 'Authoritative record of which instance owns which IP within a network.';
