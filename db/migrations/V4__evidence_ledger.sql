-- EBC-X EV1-003: Evidence Ledger — append-only System of Record
-- Design ref: design.md D05 + D-GATE-01 (four-layer defense) + TASK-H01 (Hash Chain) + TASK-H02 (REVOKE)
--
-- Four-layer defense:
--   Layer 1: DB Permission REVOKE (V2__create_roles.sql — Runtime Role has no UPDATE/DELETE/TRUNCATE)
--   Layer 2: Trigger RAISE EXCEPTION (this file — defense-in-depth for privileged paths)
--   Layer 3: Application Guard (Go internal/evidence package)
--   Layer 4: Audit + Legal Hold + Hash Chain verification

-- Evidence Ledger table (append-only)
CREATE TABLE evidence.evidence_ledger (
    evidence_id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id                VARCHAR(128) NOT NULL,
    sequence_no             BIGINT      NOT NULL,
    previous_evidence_hash  CHAR(64)    NOT NULL,
    evidence_hash           CHAR(64)    NOT NULL,
    evidence_type           VARCHAR(16) NOT NULL,
    payload                 JSONB       NOT NULL,
    source_event_id         VARCHAR(128) NOT NULL,
    transaction_id          VARCHAR(128) NOT NULL,
    tenant_id               UUID        NOT NULL,
    provenance              JSONB       NOT NULL DEFAULT '{}',
    correlation_id          VARCHAR(128),
    causation_id            VARCHAR(128),
    created_by              VARCHAR(128) NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    version                 INT         NOT NULL DEFAULT 1,
    legal_hold              BOOLEAN     NOT NULL DEFAULT false,

    CONSTRAINT chk_evidence_type CHECK (evidence_type IN ('mandatory', 'decision', 'execution')),
    CONSTRAINT chk_hash_length CHECK (length(evidence_hash) = 64 AND length(previous_evidence_hash) = 64),
    CONSTRAINT chk_sequence_positive CHECK (sequence_no >= 0)
);

COMMENT ON TABLE evidence.evidence_ledger IS 'EBC-X Evidence Ledger — append-only, Hash Chain, System of Record (TASK-H01/H02, D-GATE-01)';

-- Unique constraint: one sequence per chain
CREATE UNIQUE INDEX idx_evidence_chain_seq ON evidence.evidence_ledger (chain_id, sequence_no);

-- Index for tenant isolation (RLS)
CREATE INDEX idx_evidence_tenant ON evidence.evidence_ledger (tenant_id);
CREATE INDEX idx_evidence_transaction ON evidence.evidence_ledger (transaction_id);
CREATE INDEX idx_evidence_created ON evidence.evidence_ledger (created_at);

-- Enable RLS for tenant isolation
ALTER TABLE evidence.evidence_ledger ENABLE ROW LEVEL SECURITY;

-- RLS policy: runtime role sees only rows matching session variable app.tenant_id
CREATE POLICY evidence_tenant_isolation ON evidence.evidence_ledger
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

-- ============================================================================
-- Layer 2: Trigger defense — RAISE EXCEPTION on UPDATE/DELETE (TASK-H02)
-- Even if a privileged role bypasses Layer 1 (REVOKE), triggers block mutation.
-- ============================================================================
CREATE OR REPLACE FUNCTION evidence.block_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'Evidence Ledger is append-only (TASK-H02). UPDATE/DELETE prohibited. Use new version or tombstone.';
END;
$$;

CREATE TRIGGER evidence_no_update
    BEFORE UPDATE ON evidence.evidence_ledger
    FOR EACH ROW EXECUTE FUNCTION evidence.block_mutation();

CREATE TRIGGER evidence_no_delete
    BEFORE DELETE ON evidence.evidence_ledger
    FOR EACH ROW EXECUTE FUNCTION evidence.block_mutation();

-- ============================================================================
-- Audit table for tamper detection (Layer 4)
-- ============================================================================
CREATE TABLE audit.evidence_audit (
    audit_id        BIGSERIAL   PRIMARY KEY,
    audit_type      VARCHAR(32) NOT NULL,
    evidence_id     UUID,
    chain_id        VARCHAR(128),
    detail          JSONB,
    audited_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE audit.evidence_audit IS 'EBC-X Evidence audit trail — tamper detection and chain verification logs';