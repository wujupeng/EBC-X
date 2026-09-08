-- EBC-X EV1-006: Object Storage Artifact Metadata
-- Design ref: design.md D15 Data Architecture + D-GATE-03 (Three-Layer Truth Model)
-- Artifact Truth = Object Storage; Metadata tracks WORM + Evidence Ledger reference
--
-- Naming Convention: {tenantId}/{evidenceId}/{artifactType}/{version}/{filename}
-- WORM: Write Once Read Many — no update, no delete
-- Evidence First: every artifact MUST reference an evidence_id from evidence.evidence_ledger

-- =====================================================================
-- artifact schema + artifact.metadata table
-- =====================================================================
CREATE SCHEMA IF NOT EXISTS artifact;
GRANT USAGE ON SCHEMA artifact TO ebcx_runtime;

COMMENT ON SCHEMA artifact IS 'EBC-X Object Storage Artifact metadata — WORM, Evidence-linked (D-GATE-03)';

CREATE TABLE IF NOT EXISTS artifact.metadata (
    artifact_id     UUID         NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       UUID         NOT NULL,
    evidence_id     UUID         NOT NULL,
    artifact_type   VARCHAR(20)  NOT NULL,
    version         BIGINT       NOT NULL,
    filename        VARCHAR(512) NOT NULL,
    content_type    VARCHAR(128) NOT NULL,
    size_bytes      BIGINT       NOT NULL,
    checksum        CHAR(64)     NOT NULL,
    object_key      TEXT         NOT NULL,
    uploaded_by     VARCHAR(128) NOT NULL,
    uploaded_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    worm_protected  BOOLEAN      NOT NULL DEFAULT true,
    storage_backend VARCHAR(20)  NOT NULL DEFAULT 'local',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (artifact_id),

    CONSTRAINT chk_artifact_type CHECK (artifact_type IN (
        'pdf', 'image', 'cad', 'quality_report', 'contract',
        'invoice', 'evidence', 'log', 'custom'
    )),
    CONSTRAINT chk_version_positive CHECK (version >= 1),
    CONSTRAINT chk_size_nonnegative CHECK (size_bytes >= 0),
    CONSTRAINT chk_worm_immutable CHECK (worm_protected = true)
);

COMMENT ON TABLE artifact.metadata IS 'EBC-X Artifact metadata — WORM-protected, Evidence-linked (EV1-006, D-GATE-03)';

-- FK to Evidence Ledger (Evidence First: every artifact has an evidence record)
ALTER TABLE artifact.metadata
    ADD CONSTRAINT fk_artifact_evidence
    FOREIGN KEY (evidence_id) REFERENCES evidence.evidence_ledger(evidence_id);

-- Unique constraint: one artifact per object_key (WORM enforcement at DB level)
CREATE UNIQUE INDEX idx_artifact_object_key ON artifact.metadata (object_key);

-- Index for tenant isolation + Evidence lookup
CREATE INDEX idx_artifact_tenant ON artifact.metadata (tenant_id);
CREATE INDEX idx_artifact_evidence ON artifact.metadata (evidence_id);
CREATE INDEX idx_artifact_tenant_type ON artifact.metadata (tenant_id, artifact_type);

-- RLS for tenant isolation
ALTER TABLE artifact.metadata ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS artifact_tenant_isolation ON artifact.metadata;
CREATE POLICY artifact_tenant_isolation ON artifact.metadata
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT ON artifact.metadata TO ebcx_runtime;

-- =====================================================================
-- WORM enforcement: Block UPDATE and DELETE via triggers (defense-in-depth)
-- =====================================================================
CREATE OR REPLACE FUNCTION artifact.block_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'Artifact is WORM-protected (EV1-006). UPDATE/DELETE prohibited.';
END;
$$;

DROP TRIGGER IF EXISTS artifact_no_update ON artifact.metadata;
CREATE TRIGGER artifact_no_update
    BEFORE UPDATE ON artifact.metadata
    FOR EACH ROW EXECUTE FUNCTION artifact.block_mutation();

DROP TRIGGER IF EXISTS artifact_no_delete ON artifact.metadata;
CREATE TRIGGER artifact_no_delete
    BEFORE DELETE ON artifact.metadata
    FOR EACH ROW EXECUTE FUNCTION artifact.block_mutation();