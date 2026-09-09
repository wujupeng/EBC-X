-- EBC-X EV1-010: Organization Aggregate Root + Command Idempotency Extension
-- Design ref: design.md v1.0 §2.4.1 Data Model / §2.4.2 UNIQUE NULLS NOT DISTINCT / §2.4.3 RLS / §2.4.4 Index / §2.4.5 command_idempotency extension
-- R1.1: OrganizationAggregate is Business State Mutation Root
-- R1.2: OrganizationTreeCoordinator is Tree Structural State Mutation Authority
-- R1.3: descendant level change does not go through Aggregate Mutation, no version increment
-- R1.4: MoveOrganization produces 2 Evidence records
-- R5: Organization tree level <= 5
-- R6: Enterprise ownership immutable

-- =====================================================================
-- business.organizations - Organization aggregate root table
-- =====================================================================
CREATE TABLE IF NOT EXISTS business.organizations (
    org_id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id        UUID         NOT NULL,
    parent_id            UUID,
    name                 VARCHAR(256) NOT NULL,
    code                 VARCHAR(64)  NOT NULL,
    level                INTEGER      NOT NULL,
    version              BIGINT       NOT NULL DEFAULT 1,
    source_evidence_id   UUID,
    tenant_id            UUID         NOT NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_org_level CHECK (level BETWEEN 1 AND 5),
    CONSTRAINT chk_org_version CHECK (version >= 1),
    CONSTRAINT chk_org_name CHECK (length(btrim(name)) > 0 AND length(name) <= 256),
    CONSTRAINT chk_org_code CHECK (length(btrim(code)) > 0 AND length(code) <= 64),

    CONSTRAINT fk_org_parent FOREIGN KEY (parent_id)
        REFERENCES business.organizations(org_id),
    CONSTRAINT fk_org_enterprise FOREIGN KEY (enterprise_id)
        REFERENCES business.enterprises(enterprise_id)
);

COMMENT ON TABLE business.organizations IS 'EBC-X Organization Aggregate Root - Business State Mutation Root (R1.1, design.md v1.0 §2.4.1)';

-- Unique constraint: same enterprise + same parent + code unique (NULLS NOT DISTINCT for root orgs)
CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_enterprise_parent_code
    ON business.organizations (enterprise_id, parent_id, code)
    NULLS NOT DISTINCT;

-- Index: enterprise_id + parent_id queries (children lookup)
CREATE INDEX IF NOT EXISTS idx_organizations_enterprise_parent
    ON business.organizations (enterprise_id, parent_id);

-- Index: level queries (level statistics/validation)
CREATE INDEX IF NOT EXISTS idx_organizations_level
    ON business.organizations (level);

-- Index: tenant_id queries (RLS isolation optimization)
CREATE INDEX IF NOT EXISTS idx_organizations_tenant
    ON business.organizations (tenant_id);

-- Index: enterprise_id queries (org tree rebuild / Enterprise ownership validation)
CREATE INDEX IF NOT EXISTS idx_organizations_enterprise
    ON business.organizations (enterprise_id);

-- Enable RLS for tenant isolation
ALTER TABLE business.organizations ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS organizations_tenant_isolation ON business.organizations;
CREATE POLICY organizations_tenant_isolation ON business.organizations
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

-- Runtime Role: SELECT + INSERT + UPDATE (no DELETE, organizations are long-lived)
GRANT SELECT, INSERT, UPDATE ON business.organizations TO ebcx_runtime;

-- =====================================================================
-- Extend command_idempotency CHECK constraint (compatibility extension)
-- =====================================================================
ALTER TABLE business.command_idempotency DROP CONSTRAINT IF EXISTS chk_idem_cmd_type;
ALTER TABLE business.command_idempotency ADD CONSTRAINT chk_idem_cmd_type
    CHECK (command_type IN (
        'CreateEnterprise', 'UpdateEnterprise',
        'CreateOrganization', 'UpdateOrganization', 'MoveOrganization'
    ));