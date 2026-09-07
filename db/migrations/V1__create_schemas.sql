-- EBC-X EV1-002: Create 7 PostgreSQL schemas
-- Design ref: design.md D15 Data Architecture
-- Schemas: business / evidence / outbox / audit / master_data / policy / tenant

CREATE SCHEMA IF NOT EXISTS business;
CREATE SCHEMA IF NOT EXISTS evidence;
CREATE SCHEMA IF NOT EXISTS outbox;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS master_data;
CREATE SCHEMA IF NOT EXISTS policy;
CREATE SCHEMA IF NOT EXISTS tenant;

COMMENT ON SCHEMA business     IS 'EBC-X business transaction data (Order/Contract/Invoice/Payment)';
COMMENT ON SCHEMA evidence     IS 'EBC-X Evidence Ledger — append-only System of Record (TASK-R04)';
COMMENT ON SCHEMA outbox       IS 'EBC-X Outbox pattern for reliable event delivery';
COMMENT ON SCHEMA audit        IS 'EBC-X append-only audit trail';
COMMENT ON SCHEMA master_data  IS 'EBC-X master data (Enterprise/Org/Person/Product/Material)';
COMMENT ON SCHEMA policy       IS 'EBC-X Policy Engine rules and decisions';
COMMENT ON SCHEMA tenant       IS 'EBC-X multi-tenant configuration and Country Pack routing';