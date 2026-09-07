-- EBC-X EV1-007: HTKIS-AF Security Base — Audit Log + RLS Policies
-- Design ref: design.md D13 Security Architecture
-- Audit append-only (TASK-R04), RLS tenant isolation, JWT/KMS at application layer
--
-- HTKIS-AF is the SOLE security base. No module may build its own security.

CREATE SCHEMA IF NOT EXISTS security;
COMMENT ON SCHEMA security IS 'EBC-X HTKIS-AF Security Base — KMS, JWT revocation, audit (D13)';

-- =====================================================================
-- audit.audit_log — append-only audit trail (D13, TASK-R04)
-- =====================================================================
CREATE TABLE IF NOT EXISTS audit.audit_log (
    audit_id        UUID         NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       UUID         NOT NULL,
    user_id         VARCHAR(128),
    action          VARCHAR(64)  NOT NULL,
    resource        VARCHAR(128) NOT NULL,
    resource_id     VARCHAR(128),
    decision        VARCHAR(20)  NOT NULL,  -- allow | deny | error
    reason          TEXT,
    ip_address      VARCHAR(45),
    user_agent      TEXT,
    metadata        JSONB        NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (audit_id),

    CONSTRAINT chk_audit_decision CHECK (decision IN ('allow', 'deny', 'error'))
);

COMMENT ON TABLE audit.audit_log IS 'EBC-X Audit Log — append-only, tenant-isolated (D13, TASK-R04)';

-- Index for tenant-scoped queries
CREATE INDEX idx_audit_tenant_time ON audit.audit_log (tenant_id, created_at DESC);
CREATE INDEX idx_audit_tenant_action ON audit.audit_log (tenant_id, action);
CREATE INDEX idx_audit_tenant_resource ON audit.audit_log (tenant_id, resource, resource_id);

-- RLS for tenant isolation
ALTER TABLE audit.audit_log ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS audit_log_tenant_isolation ON audit.audit_log;
CREATE POLICY audit_log_tenant_isolation ON audit.audit_log
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT ON audit.audit_log TO ebcx_runtime;

-- =====================================================================
-- Append-only enforcement: Block UPDATE and DELETE (defense-in-depth)
-- =====================================================================
CREATE OR REPLACE FUNCTION audit.block_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'Audit log is append-only (TASK-R04, EV1-007). UPDATE/DELETE prohibited.';
END;
$$;

DROP TRIGGER IF EXISTS audit_no_update ON audit.audit_log;
CREATE TRIGGER audit_no_update
    BEFORE UPDATE ON audit.audit_log
    FOR EACH ROW EXECUTE FUNCTION audit.block_mutation();

DROP TRIGGER IF EXISTS audit_no_delete ON audit.audit_log;
CREATE TRIGGER audit_no_delete
    BEFORE DELETE ON audit.audit_log
    FOR EACH ROW EXECUTE FUNCTION audit.block_mutation();

-- =====================================================================
-- security.kms_keys — Key Management Service metadata
-- =====================================================================
CREATE TABLE IF NOT EXISTS security.kms_keys (
    key_id          VARCHAR(128) NOT NULL,
    tenant_id       UUID,
    key_fingerprint CHAR(64)     NOT NULL,  -- SHA-256 of key (not the key itself)
    algorithm       VARCHAR(20)  NOT NULL DEFAULT 'AES-256-GCM',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ  NOT NULL,
    rotated_at      TIMESTAMPTZ,
    is_active       BOOLEAN      NOT NULL DEFAULT true,
    PRIMARY KEY (key_id),

    CONSTRAINT chk_kms_algorithm CHECK (algorithm IN ('AES-256-GCM', 'AES-256-CBC'))
);

COMMENT ON TABLE security.kms_keys IS 'EBC-X KMS key metadata — key rotation tracking (D13)';

CREATE INDEX idx_kms_active ON security.kms_keys (is_active) WHERE is_active = true;
CREATE INDEX idx_kms_tenant ON security.kms_keys (tenant_id);

GRANT SELECT, INSERT, UPDATE ON security.kms_keys TO ebcx_runtime;

-- =====================================================================
-- security.revoked_tokens — JWT revocation list
-- =====================================================================
CREATE TABLE IF NOT EXISTS security.revoked_tokens (
    token_jti       VARCHAR(128) NOT NULL,
    tenant_id       UUID         NOT NULL,
    revoked_by      VARCHAR(128) NOT NULL,
    revoked_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ  NOT NULL,
    reason          TEXT,
    PRIMARY KEY (token_jti)
);

COMMENT ON TABLE security.revoked_tokens IS 'EBC-X JWT revocation list (D13)';

ALTER TABLE security.revoked_tokens ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS revoked_tokens_tenant_isolation ON security.revoked_tokens;
CREATE POLICY revoked_tokens_tenant_isolation ON security.revoked_tokens
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT ON security.revoked_tokens TO ebcx_runtime;

-- Auto-cleanup of expired revocations
CREATE INDEX idx_revoked_expires ON security.revoked_tokens (expires_at);