-- Floating public IPs: pool rows (unallocated), per-project allocation, optional bind to a running instance (active NAT).

CREATE TABLE IF NOT EXISTS floating_ips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    public_ip INET NOT NULL UNIQUE,
    instance_id UUID REFERENCES instances(id) ON DELETE RESTRICT,
    private_ip INET,
    status TEXT NOT NULL CHECK (status IN ('unallocated', 'allocated', 'active')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_floating_ips_project_id ON floating_ips (project_id);
CREATE INDEX IF NOT EXISTS idx_floating_ips_instance_id ON floating_ips (instance_id) WHERE instance_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_floating_ips_status ON floating_ips (status);

DROP TRIGGER IF EXISTS trg_floating_ips_updated_at ON floating_ips;
CREATE TRIGGER trg_floating_ips_updated_at
    BEFORE UPDATE ON floating_ips
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE floating_ips IS 'External address pool; allocated rows belong to a project; active rows are DNAT/SNAT to instance private_ip.';

-- RFC 5737 TEST-NET-3 documentation addresses (replace in production via admin SQL or future pool sync).
INSERT INTO floating_ips (project_id, public_ip, instance_id, private_ip, status)
SELECT NULL, ('203.0.113.' || (gs.n + 1))::inet, NULL, NULL, 'unallocated'
FROM generate_series(1, 20) AS gs(n)
ON CONFLICT (public_ip) DO NOTHING;
