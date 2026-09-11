-- EBC-X EV1-011: Person Aggregate Root + Command Idempotency Extension
-- Design ref: design.md v1.0 §2.4.1 Data Model / §2.4.2 UNIQUE / §2.4.3 RLS / §2.4.4 Index / §2.4.5 command_idempotency extension
-- R1: PersonAggregate is Mutation Root
-- R2: Idempotency + Person + Evidence + Outbox same ACID transaction
-- R3: Neo4j never enters Person Mutation main transaction
-- R4: Update/AssignRole version-based CAS optimistic concurrency control
-- R5: Person must belong to existing Organization, orgId immutable after creation
-- R6: Role reference uniqueness: same Person roleId not duplicated

-- =====================================================================
-- business.persons - Person aggregate root table
-- =====================================================================
CREATE TABLE IF NOT EXISTS business.persons (
    person_id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id               UUID         NOT NULL,
    name                 VARCHAR(128) NOT NULL,
    employee_no          VARCHAR(64)  NOT NULL,
    roles                JSONB        NOT NULL DEFAULT '[]',
    version              BIGINT       NOT NULL DEFAULT 1,
    source_evidence_id   UUID,
    tenant_id            UUID         NOT NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_person_version CHECK (version >= 1),
    CONSTRAINT chk_person_name CHECK (length(btrim(name)) > 0 AND length(name) <= 128),
    CONSTRAINT chk_person_employee_no CHECK (length(btrim(employee_no)) > 0 AND length(employee_no) <= 64),
    CONSTRAINT chk_person_roles_array CHECK (jsonb_typeof(roles) = 'array'),

    CONSTRAINT fk_person_organization FOREIGN KEY (org_id)
        REFERENCES business.organizations(org_id)
);

COMMENT ON TABLE business.persons IS 'EBC-X Person Aggregate Root - Mutation Root (R1, design.md v1.0 §2.4.1)';

-- Unique index: same org + same employee_no unique (person uniqueness invariant)
CREATE UNIQUE INDEX IF NOT EXISTS idx_persons_org_employee_no
    ON business.persons (org_id, employee_no);

-- Index: org_id queries (organization membership lookup)
CREATE INDEX IF NOT EXISTS idx_persons_org
    ON business.persons (org_id);

-- Index: tenant_id queries (RLS isolation optimization)
CREATE INDEX IF NOT EXISTS idx_persons_tenant
    ON business.persons (tenant_id);

-- Index: org_id + tenant_id queries (composite lookup)
CREATE INDEX IF NOT EXISTS idx_persons_org_tenant
    ON business.persons (org_id, tenant_id);

-- Enable RLS for tenant isolation
ALTER TABLE business.persons ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS persons_tenant_isolation ON business.persons;
CREATE POLICY persons_tenant_isolation ON business.persons
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

-- Runtime Role: SELECT + INSERT + UPDATE (no DELETE, persons are long-lived)
GRANT SELECT, INSERT, UPDATE ON business.persons TO ebcx_runtime;

-- =====================================================================
-- Extend command_idempotency CHECK constraint (compatibility extension)
-- =====================================================================
ALTER TABLE business.command_idempotency DROP CONSTRAINT IF EXISTS chk_idem_cmd_type;
ALTER TABLE business.command_idempotency ADD CONSTRAINT chk_idem_cmd_type
    CHECK (command_type IN (
        'CreateEnterprise', 'UpdateEnterprise',
        'CreateOrganization', 'UpdateOrganization', 'MoveOrganization',
        'CreatePerson', 'UpdatePerson', 'AssignRole'
    ));