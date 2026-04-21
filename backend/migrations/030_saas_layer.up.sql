-- 030_saas_layer.up.sql
-- SaaS multi-tenant transformation layer
-- Adds: tenants, api_keys, usage_events, invoices

-- ── Tenants ──────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS tenants (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    plan        VARCHAR(50)  NOT NULL DEFAULT 'starter'
                             CHECK (plan IN ('starter', 'pro', 'enterprise')),
    status      VARCHAR(50)  NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active', 'suspended', 'cancelled')),
    email       VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug   ON tenants(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

-- ── API Keys ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS api_keys (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    key_hash     VARCHAR(64)  NOT NULL UNIQUE,          -- SHA-256 of raw key
    key_prefix   VARCHAR(12)  NOT NULL,                 -- first 10 chars for display
    role         VARCHAR(50)  NOT NULL DEFAULT 'dev'
                              CHECK (role IN ('owner', 'dev', 'readonly')),
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash   ON api_keys(key_hash);

-- ── Usage Events ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS usage_events (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_type VARCHAR(50)  NOT NULL
               CHECK (event_type IN (
                   'requests', 'kafka_events', 'aiops_incidents',
                   'chaos_runs', 'storage_gb_hour'
               )),
    quantity   BIGINT       NOT NULL DEFAULT 1,
    metadata   JSONB,
    ts         TIMESTAMPTZ  NOT NULL DEFAULT now()
) PARTITION BY RANGE (ts);

-- Monthly partitions (current + next)
CREATE TABLE IF NOT EXISTS usage_events_default PARTITION OF usage_events DEFAULT;

CREATE INDEX IF NOT EXISTS idx_usage_tenant_type_ts
    ON usage_events(tenant_id, event_type, ts DESC);

-- ── Invoices ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS invoices (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ  NOT NULL,
    period_end   TIMESTAMPTZ  NOT NULL,
    amount_cents INTEGER      NOT NULL DEFAULT 0,
    items        JSONB,                              -- LineItem array
    status       VARCHAR(50)  NOT NULL DEFAULT 'draft'
                              CHECK (status IN ('draft', 'finalized', 'paid', 'overdue')),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_invoices_tenant        ON invoices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_invoices_period_status ON invoices(tenant_id, period_start, status);
