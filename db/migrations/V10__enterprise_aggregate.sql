-- EBC-X EV1-009: Enterprise Aggregate Root + Command Idempotency
-- Design ref: design.md v1.1 §2.3.2 Data Model / §2.4.6.1 Command Idempotency
-- R1: EnterpriseAggregate is the sole Mutation Root
-- R2: Idempotency + Enterprise + Evidence + Outbox same ACID transaction
-- BLOCKER-01: UNIQUE(tenant_id, command_id) for command idempotency

-- =====================================================================
-- business.enterprises - Enterprise aggregate root table
-- =====================================================================
CREATE TABLE IF NOT EXISTS business.enterprises (
    enterprise_id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                 VARCHAR(256) NOT NULL,
    version              BIGINT      NOT NULL DEFAULT 1,
    source_evidence_id   UUID,
    tenant_id            UUID        NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_enterprise_name CHECK (length(btrim(name)) > 0 AND length(name) <= 256),
    CONSTRAINT chk_enterprise_version CHECK (version >= 1)
);

COMMENT ON TABLE business.enterprises IS 'EBC-X Enterprise Aggregate Root - sole Mutation Root (R1, design.md §2.2.2.1)';

-- Unique constraint: same tenant cannot have duplicate enterprise name
CREATE UNIQUE INDEX IF NOT EXISTS idx_enterprises_tenant_name
    ON business.enterprises (tenant_id, name);

-- Index for tenant isolation queries
CREATE INDEX IF NOT EXISTS idx_enterprises_tenant ON business.enterprises (tenant_id);

-- Enable RLS for tenant isolation
ALTER TABLE business.enterprises ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS enterprises_tenant_isolation ON business.enterprises;
CREATE POLICY enterprises_tenant_isolation ON business.enterprises
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

-- Runtime Role: SELECT + INSERT + UPDATE (no DELETE, enterprises are long-lived)
GRANT SELECT, INSERT, UPDATE ON business.enterprises TO ebcx_runtime;

-- =====================================================================
-- business.command_idempotency - Command idempotency record (BLOCKER-01)
-- =====================================================================
CREATE TABLE IF NOT EXISTS business.command_idempotency (
    command_id           UUID        NOT NULL,
    tenant_id            UUID        NOT NULL,
    aggregate_id         UUID,
    command_type         VARCHAR(64) NOT NULL,
    status               VARCHAR(16) NOT NULL DEFAULT 'pending',
    result_version       BIGINT,
    result_event_id      UUID,
    result_evidence_id   UUID,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (tenant_id, command_id),

    CONSTRAINT chk_idem_status CHECK (status IN ('pending', 'success', 'failed')),
    CONSTRAINT chk_idem_cmd_type CHECK (command_type IN ('CreateEnterprise', 'UpdateEnterprise'))
);

COMMENT ON TABLE business.command_idempotency IS 'EBC-X Command Idempotency Record - BLOCKER-01, UNIQUE(tenant_id, command_id)';

-- Index for status queries (e.g., find pending commands)
CREATE INDEX IF NOT EXISTS idx_idempotency_tenant_status
    ON business.command_idempotency (tenant_id, status);

-- Enable RLS for tenant isolation
ALTER TABLE business.command_idempotency ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS idempotency_tenant_isolation ON business.command_idempotency;
CREATE POLICY idempotency_tenant_isolation ON business.command_idempotency
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

-- Runtime Role: SELECT + INSERT + UPDATE + DELETE
-- (Idempotency Record is NOT append-only; failed records can be DELETEd for retry)
GRANT SELECT, INSERT, UPDATE, DELETE ON business.command_idempotency TO ebcx_runtime;

-- ============================================================================
-- Grant UPDATE on evidence.evidence_ledger to ebcx_runtime for SELECT ... FOR UPDATE
-- (TASK-AMEND-001 / design.md §2.4.3.1: Chain Predecessor 行锁)
-- WORM triggers (V4 evidence_no_update/evidence_no_delete) still block actual
-- UPDATE/DELETE operations; this grant enables row locking only, not mutation.
-- ============================================================================
GRANT UPDATE ON evidence.evidence_ledger TO ebcx_runtime;