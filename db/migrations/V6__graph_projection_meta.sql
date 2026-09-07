-- EBC-X EV1-005: Graph Projection Metadata (PostgreSQL side)
-- Tracks Graph Projection state, Reconciliation results, Shadow Rebuild progress
-- Neo4j is Projection; PostgreSQL is Truth Source (D-GATE-05, D-GATE-06)

-- =====================================================================
-- graph.projection_state — per-event projection tracking
-- =====================================================================
CREATE TABLE IF NOT EXISTS graph.projection_state (
    event_id          UUID         NOT NULL,
    tenant_id         UUID         NOT NULL,
    projection_status VARCHAR(20)  NOT NULL,  -- pending | projected | failed | dlq
    projected_at      TIMESTAMPTZ,
    projection_lag_ms BIGINT,
    retry_count       INT          NOT NULL DEFAULT 0,
    error_message     TEXT,
    projected_node_ids UUID[]      NOT NULL DEFAULT '{}',  -- nodes created/updated
    projected_edge_ids UUID[]      NOT NULL DEFAULT '{}',  -- edges created/updated
    graph_instance    VARCHAR(10)  NOT NULL DEFAULT 'A',   -- A or B (Shadow Rebuild)
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, graph_instance)
);

-- RLS on projection_state
ALTER TABLE graph.projection_state ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS projection_state_tenant_isolation ON graph.projection_state;
CREATE POLICY projection_state_tenant_isolation ON graph.projection_state
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT, UPDATE ON graph.projection_state TO ebcx_runtime;

-- =====================================================================
-- graph.reconciliation_results — Reconciliation Evidence (append-only)
-- =====================================================================
CREATE TABLE IF NOT EXISTS graph.reconciliation_results (
    reconciliation_id  UUID         NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           UUID         NOT NULL,
    started_at          TIMESTAMPTZ  NOT NULL,
    completed_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    graph_instance_a    VARCHAR(10)  NOT NULL,  -- 'A'
    graph_instance_b    VARCHAR(10)  NOT NULL,  -- 'B' (for Shadow Rebuild) or NULL
    node_count_a        BIGINT       NOT NULL,
    node_count_b        BIGINT       NOT NULL,
    edge_count_a        BIGINT       NOT NULL,
    edge_count_b        BIGINT       NOT NULL,
    checksum_a          TEXT         NOT NULL,  -- SHA256 of sorted node+edge IDs
    checksum_b          TEXT         NOT NULL,
    evidence_ref_match  BOOLEAN      NOT NULL,
    tenant_isolation_match BOOLEAN   NOT NULL,
    version_match       BOOLEAN      NOT NULL,
    node_count_match    BOOLEAN      NOT NULL,
    edge_count_match    BOOLEAN      NOT NULL,
    checksum_match      BOOLEAN      NOT NULL,
    overall_pass        BOOLEAN      NOT NULL,
    differences         JSONB        NOT NULL DEFAULT '{}',  -- detailed diff
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (reconciliation_id)
);

ALTER TABLE graph.reconciliation_results ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS reconciliation_tenant_isolation ON graph.reconciliation_results;
CREATE POLICY reconciliation_tenant_isolation ON graph.reconciliation_results
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT ON graph.reconciliation_results TO ebcx_runtime;

-- =====================================================================
-- graph.shadow_rebuild — Shadow Rebuild job tracking (TASK-H04)
-- =====================================================================
CREATE TABLE IF NOT EXISTS graph.shadow_rebuild (
    rebuild_id         UUID         NOT NULL DEFAULT gen_random_uuid(),
    tenant_id          UUID         NOT NULL,
    started_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    completed_at       TIMESTAMPTZ,
    source_instance    VARCHAR(10)  NOT NULL,  -- 'A' (current serving)
    target_instance    VARCHAR(10)  NOT NULL,  -- 'B' (shadow build)
    status             VARCHAR(20)  NOT NULL,  -- building | reconciling | cutover | completed | failed
    events_replayed    BIGINT       NOT NULL DEFAULT 0,
    nodes_built        BIGINT       NOT NULL DEFAULT 0,
    edges_built        BIGINT       NOT NULL DEFAULT 0,
    reconciliation_id UUID,                  -- FK to reconciliation_results
    cutover_at         TIMESTAMPTZ,           -- atomic cutover timestamp
    rto_ms             BIGINT,                -- actual RTO measured
    availability_window_ms BIGINT  NOT NULL DEFAULT 0,  -- must be 0
    error_message      TEXT,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (rebuild_id)
);

ALTER TABLE graph.shadow_rebuild ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS shadow_rebuild_tenant_isolation ON graph.shadow_rebuild;
CREATE POLICY shadow_rebuild_tenant_isolation ON graph.shadow_rebuild
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT, UPDATE ON graph.shadow_rebuild TO ebcx_runtime;

-- =====================================================================
-- graph.projection_metrics — 6 key metrics (D-GATE-05)
-- =====================================================================
CREATE TABLE IF NOT EXISTS graph.projection_metrics (
    metric_id          BIGSERIAL    NOT NULL,
    tenant_id          UUID         NOT NULL,
    metric_name        VARCHAR(50)  NOT NULL,  -- event_lag/projection_lag/consumer_lag/rebuild_duration/failed_projection_count/dlq_count
    metric_value       DOUBLE PRECISION NOT NULL,
    mode               VARCHAR(20)  NOT NULL,  -- normal/degraded/recovery/rebuild
    recorded_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (metric_id)
);

ALTER TABLE graph.projection_metrics ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS projection_metrics_tenant_isolation ON graph.projection_metrics;
CREATE POLICY projection_metrics_tenant_isolation ON graph.projection_metrics
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT ON graph.projection_metrics TO ebcx_runtime;

-- Index for metrics query
CREATE INDEX IF NOT EXISTS projection_metrics_name_time_idx
    ON graph.projection_metrics (metric_name, recorded_at DESC);