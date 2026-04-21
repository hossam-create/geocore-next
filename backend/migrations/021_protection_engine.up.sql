-- Travel Guarantee + Protection Engine + A/B Testing
-- Extends cancellation insurance into a full protection system.

-- ── Order Protection (superset of insurance) ────────────────────────────────────
CREATE TABLE IF NOT EXISTS order_protections (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id            UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    price_cents         BIGINT      NOT NULL DEFAULT 0,
    has_cancellation    BOOLEAN     NOT NULL DEFAULT FALSE,
    has_delay           BOOLEAN     NOT NULL DEFAULT FALSE,
    has_full            BOOLEAN     NOT NULL DEFAULT FALSE,
    coverage_percent    NUMERIC(5,2) NOT NULL DEFAULT 100,
    risk_factor         NUMERIC(5,4) NOT NULL DEFAULT 0,
    urgency_factor      NUMERIC(5,4) NOT NULL DEFAULT 0,
    is_used             BOOLEAN     NOT NULL DEFAULT FALSE,
    first_order_free    BOOLEAN     NOT NULL DEFAULT FALSE,
    ab_variant          VARCHAR(10) NOT NULL DEFAULT 'control',  -- control | opt_out | social_proof
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(order_id)
);

-- ── Guarantee Claims ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS guarantee_claims (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id        UUID        NOT NULL REFERENCES orders(id),
    user_id         UUID        NOT NULL REFERENCES users(id),
    traveler_id     UUID        NOT NULL REFERENCES users(id),
    type            VARCHAR(20) NOT NULL,          -- no_show | delay | mismatch
    evidence_json   JSONB       NOT NULL DEFAULT '{}',
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending | auto_approved | approved | rejected
    refund_cents    BIGINT      NOT NULL DEFAULT 0,
    compensation_cents BIGINT   NOT NULL DEFAULT 0,  -- extra compensation to buyer
    traveler_penalty BOOLEAN    NOT NULL DEFAULT FALSE,
    auto_evaluated  BOOLEAN     NOT NULL DEFAULT FALSE,
    reviewer_id     UUID,                               -- admin who reviewed (if manual)
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── A/B Test Variants ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS ab_variant_assignments (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    experiment  VARCHAR(50) NOT NULL,           -- e.g. 'protection_default_on'
    variant     VARCHAR(20) NOT NULL,           -- control | opt_out | social_proof
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, experiment)
);

-- ── A/B Test Events ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS ab_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL,
    experiment  VARCHAR(50) NOT NULL,
    variant     VARCHAR(20) NOT NULL,
    event_type  VARCHAR(50) NOT NULL,           -- checkout_viewed | protection_added | order_placed | cancelled | claim_filed
    order_id    UUID,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Protection Metrics (materialized for admin dashboard) ────────────────────────
CREATE TABLE IF NOT EXISTS protection_daily_metrics (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    date                DATE        NOT NULL,
    total_orders        INT         NOT NULL DEFAULT 0,
    protection_attached INT         NOT NULL DEFAULT 0,
    attach_rate         NUMERIC(5,4) NOT NULL DEFAULT 0,
    revenue_cents       BIGINT      NOT NULL DEFAULT 0,
    claims_filed        INT         NOT NULL DEFAULT 0,
    claims_approved     INT         NOT NULL DEFAULT 0,
    payouts_cents       BIGINT      NOT NULL DEFAULT 0,
    net_revenue_cents   BIGINT      NOT NULL DEFAULT 0,
    avg_risk_factor     NUMERIC(5,4) NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(date)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_protection_user ON order_protections(user_id);
CREATE INDEX IF NOT EXISTS idx_protection_active ON order_protections(is_used) WHERE is_used = FALSE;
CREATE INDEX IF NOT EXISTS idx_claims_order ON guarantee_claims(order_id);
CREATE INDEX IF NOT EXISTS idx_claims_user ON guarantee_claims(user_id);
CREATE INDEX IF NOT EXISTS idx_claims_status ON guarantee_claims(status);
CREATE INDEX IF NOT EXISTS idx_ab_variant_user_exp ON ab_variant_assignments(user_id, experiment);
CREATE INDEX IF NOT EXISTS idx_ab_events_exp ON ab_events(experiment, created_at);
CREATE INDEX IF NOT EXISTS idx_metrics_date ON protection_daily_metrics(date);
