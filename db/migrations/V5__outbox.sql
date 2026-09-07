-- EBC-X EV1-004: Outbox table — reliable event delivery
-- Design ref: design.md D07 Outbox+EventBus, TASK-H03 Normal/Retry separation
--
-- Principle: Outbox is a reliable propagation mechanism, NOT a truth source.
-- PostgreSQL Evidence/Transaction Truth is the root.

CREATE TABLE outbox.events (
    event_id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type    VARCHAR(64) NOT NULL,
    aggregate_id      VARCHAR(128) NOT NULL,
    event_type        VARCHAR(64) NOT NULL,
    payload           JSONB       NOT NULL,
    tenant_id         UUID        NOT NULL,
    correlation_id    VARCHAR(128),
    causation_id      VARCHAR(128),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at      TIMESTAMPTZ,
    retry_count       INT         NOT NULL DEFAULT 0,
    status            VARCHAR(16) NOT NULL DEFAULT 'pending',
    next_retry_at     TIMESTAMPTZ,

    CONSTRAINT chk_outbox_status CHECK (status IN ('pending','published','failed','dlq'))
);

COMMENT ON TABLE outbox.events IS 'EBC-X Outbox — reliable event delivery, same-transaction as business commit (design.md D07)';

CREATE INDEX idx_outbox_pending ON outbox.events (created_at) WHERE status = 'pending';
CREATE INDEX idx_outbox_tenant ON outbox.events (tenant_id);
CREATE INDEX idx_outbox_retry ON outbox.events (next_retry_at) WHERE status = 'pending' AND retry_count > 0;

ALTER TABLE outbox.events ENABLE ROW LEVEL SECURITY;

CREATE POLICY outbox_tenant_isolation ON outbox.events
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA outbox TO ebcx_runtime;
REVOKE UPDATE, DELETE, TRUNCATE ON ALL TABLES IN SCHEMA outbox FROM ebcx_runtime;
ALTER DEFAULT PRIVILEGES IN SCHEMA outbox GRANT SELECT, INSERT ON TABLES TO ebcx_runtime;
ALTER DEFAULT PRIVILEGES IN SCHEMA outbox REVOKE UPDATE, DELETE, TRUNCATE ON TABLES FROM ebcx_runtime;