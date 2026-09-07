-- EBC-X EV1-002: Tenant foundation table
-- Multi-tenant: shared schema + RLS row-level isolation (design.md D14)

CREATE TABLE IF NOT EXISTS tenant.tenants (
    tenant_id       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_code     VARCHAR(64) NOT NULL UNIQUE,
    tenant_name     VARCHAR(256) NOT NULL,
    country_pack    VARCHAR(32) NOT NULL DEFAULT 'china',
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_country_pack CHECK (country_pack IN ('china','eu','us','japan','asean')),
    CONSTRAINT chk_status CHECK (status IN ('active','suspended','archived'))
);

COMMENT ON TABLE tenant.tenants IS 'EBC-X tenant registry with Country Pack routing (design.md D14)';

-- RLS: enable row-level security on tenant-owned tables
ALTER TABLE tenant.tenants ENABLE ROW LEVEL SECURITY;