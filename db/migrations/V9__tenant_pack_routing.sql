-- EBC-X EV1-008: Multi-Tenant Base — Tenant Registry + Country Pack Routing
-- Design ref: design.md D14 Multi-Tenant Architecture
-- Shared schema + RLS + Pack routing + core table no-fork + extension tables
--
-- 5 Country Pack extension points: CN / EU / US / JP / ASEAN

-- =====================================================================
-- tenant.tenants — tenant registry
-- =====================================================================
CREATE TABLE IF NOT EXISTS tenant.tenants (
    tenant_id       UUID         NOT NULL DEFAULT gen_random_uuid(),
    tenant_name     VARCHAR(256) NOT NULL,
    country_code    VARCHAR(10)  NOT NULL,
    is_active       BOOLEAN      NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id)
);

COMMENT ON TABLE tenant.tenants IS 'EBC-X Tenant Registry (D14)';

-- =====================================================================
-- tenant.country_packs — Country Pack definitions and per-tenant enablement
-- =====================================================================
CREATE TABLE IF NOT EXISTS tenant.country_packs (
    pack_id         VARCHAR(10)  NOT NULL,
    display_name    VARCHAR(128) NOT NULL,
    ext_table_suffix VARCHAR(20) NOT NULL,
    is_active       BOOLEAN      NOT NULL DEFAULT true,
    PRIMARY KEY (pack_id),

    CONSTRAINT chk_pack_id CHECK (pack_id IN ('CN', 'EU', 'US', 'JP', 'ASEAN'))
);

COMMENT ON TABLE tenant.country_packs IS 'EBC-X Country Pack definitions (D14) — 5 extension points';

-- Insert the 5 Country Packs
INSERT INTO tenant.country_packs (pack_id, display_name, ext_table_suffix) VALUES
    ('CN', 'China Pack', 'cn_ext'),
    ('EU', 'European Union Pack', 'eu_ext'),
    ('US', 'United States Pack', 'us_ext'),
    ('JP', 'Japan Pack', 'jp_ext'),
    ('ASEAN', 'ASEAN Pack', 'asean_ext')
ON CONFLICT (pack_id) DO NOTHING;

-- =====================================================================
-- tenant.tenant_packs — per-tenant Country Pack enablement
-- =====================================================================
CREATE TABLE IF NOT EXISTS tenant.tenant_packs (
    tenant_id       UUID         NOT NULL,
    pack_id         VARCHAR(10)  NOT NULL,
    enabled_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, pack_id),

    CONSTRAINT fk_tp_tenant FOREIGN KEY (tenant_id) REFERENCES tenant.tenants(tenant_id),
    CONSTRAINT fk_tp_pack FOREIGN KEY (pack_id) REFERENCES tenant.country_packs(pack_id)
);

COMMENT ON TABLE tenant.tenant_packs IS 'EBC-X Per-tenant Country Pack enablement (D14)';

-- =====================================================================
-- Extension table template: business.order_{pack}_ext
-- Example for CN pack — other packs follow same pattern
-- Core table (business.order) is NOT forked; localization goes in ext tables
-- =====================================================================
CREATE TABLE IF NOT EXISTS business.order_cn_ext (
    order_id        UUID         NOT NULL,
    tenant_id       UUID         NOT NULL,
    invoice_type    VARCHAR(20)  NOT NULL DEFAULT 'general',
    tax_reg_number  VARCHAR(64),
    fapiao_number   VARCHAR(64),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id)
);

ALTER TABLE business.order_cn_ext ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_cn_ext_tenant_isolation ON business.order_cn_ext;
CREATE POLICY order_cn_ext_tenant_isolation ON business.order_cn_ext
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT, UPDATE ON business.order_cn_ext TO ebcx_runtime;

-- EU extension
CREATE TABLE IF NOT EXISTS business.order_eu_ext (
    order_id        UUID         NOT NULL,
    tenant_id       UUID         NOT NULL,
    vat_number      VARCHAR(64),
    gdpr_consent_id VARCHAR(128),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id)
);

ALTER TABLE business.order_eu_ext ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_eu_ext_tenant_isolation ON business.order_eu_ext;
CREATE POLICY order_eu_ext_tenant_isolation ON business.order_eu_ext
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT, UPDATE ON business.order_eu_ext TO ebcx_runtime;

-- US extension
CREATE TABLE IF NOT EXISTS business.order_us_ext (
    order_id        UUID         NOT NULL,
    tenant_id       UUID         NOT NULL,
    ein_number      VARCHAR(64),
    state_code      VARCHAR(10),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id)
);

ALTER TABLE business.order_us_ext ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_us_ext_tenant_isolation ON business.order_us_ext;
CREATE POLICY order_us_ext_tenant_isolation ON business.order_us_ext
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT, UPDATE ON business.order_us_ext TO ebcx_runtime;

-- JP extension
CREATE TABLE IF NOT EXISTS business.order_jp_ext (
    order_id        UUID         NOT NULL,
    tenant_id       UUID         NOT NULL,
    consumption_tax_rate NUMERIC(5,4),
    invoice_number  VARCHAR(64),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id)
);

ALTER TABLE business.order_jp_ext ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_jp_ext_tenant_isolation ON business.order_jp_ext;
CREATE POLICY order_jp_ext_tenant_isolation ON business.order_jp_ext
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT, UPDATE ON business.order_jp_ext TO ebcx_runtime;

-- ASEAN extension
CREATE TABLE IF NOT EXISTS business.order_asean_ext (
    order_id        UUID         NOT NULL,
    tenant_id       UUID         NOT NULL,
    gst_rate        NUMERIC(5,4),
    country_of_origin VARCHAR(10),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id)
);

ALTER TABLE business.order_asean_ext ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS order_asean_ext_tenant_isolation ON business.order_asean_ext;
CREATE POLICY order_asean_ext_tenant_isolation ON business.order_asean_ext
    FOR ALL TO ebcx_runtime
    USING (tenant_id::text = current_setting('app.tenant_id', true))
    WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));

GRANT SELECT, INSERT, UPDATE ON business.order_asean_ext TO ebcx_runtime;