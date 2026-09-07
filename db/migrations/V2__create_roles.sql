-- EBC-X EV1-002: Role separation — Migration Role vs Runtime Role
-- Design ref: design.md D-GATE-01 (four-layer defense), TASK-H02 (REVOKE produces ERROR)
--
-- Principle: Migration/Admin Role ≠ Application Runtime Role
-- Runtime Role: INSERT + SELECT only (no UPDATE/DELETE/TRUNCATE on evidence)
-- Migration Role: DDL + DML (only active during migration window)

-- Runtime Role (application runtime — least privilege)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ebcx_runtime') THEN
        CREATE ROLE ebcx_runtime NOLOGIN;
    END IF;
END $$;

-- Migration Role (DDL + DML — only active during schema migration)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ebcx_migration') THEN
        CREATE ROLE ebcx_migration NOLOGIN;
    END IF;
END $$;

-- Grant Runtime Role: SELECT + INSERT on all schemas (UPDATE/DELETE/TRUNCATE NOT granted)
GRANT USAGE ON SCHEMA business, evidence, outbox, audit, master_data, policy, tenant TO ebcx_runtime;
GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA business, evidence, outbox, audit, master_data, policy, tenant TO ebcx_runtime;

-- Explicitly REVOKE UPDATE, DELETE, TRUNCATE from Runtime Role (TASK-H02: REVOKE produces ERROR)
REVOKE UPDATE, DELETE, TRUNCATE ON ALL TABLES IN SCHEMA evidence FROM ebcx_runtime;

-- Grant Migration Role: full DDL + DML (migration window only)
GRANT ALL ON SCHEMA business, evidence, outbox, audit, master_data, policy, tenant TO ebcx_migration;
GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE ON ALL TABLES IN SCHEMA business, evidence, outbox, audit, master_data, policy, tenant TO ebcx_migration;

-- Default privileges: future tables inherit Runtime Role restrictions
ALTER DEFAULT PRIVILEGES IN SCHEMA evidence GRANT SELECT, INSERT ON TABLES TO ebcx_runtime;
ALTER DEFAULT PRIVILEGES IN SCHEMA evidence REVOKE UPDATE, DELETE, TRUNCATE ON TABLES FROM ebcx_runtime;

COMMENT ON ROLE ebcx_runtime   IS 'EBC-X application runtime role — INSERT+SELECT only, no UPDATE/DELETE/TRUNCATE on evidence (TASK-H02)';
COMMENT ON ROLE ebcx_migration IS 'EBC-X migration role — full DDL+DML, active only during migration window';