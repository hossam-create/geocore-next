-- Smart Cancellation Fee Engine (Buyer → Traveler)
-- Dynamic fees based on time since acceptance, fair compensation to traveler,
-- anti-abuse logic, and free cancellation tokens.

-- ── Cancellation Policy (per corridor or global default) ──────────────────────
CREATE TABLE IF NOT EXISTS cancellation_policies (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    corridor_key    VARCHAR(100) NOT NULL DEFAULT 'global',  -- e.g. 'global', 'EG-AE'
    grace_seconds   INT         NOT NULL DEFAULT 600,         -- 10 min free window
    tier1_seconds   INT         NOT NULL DEFAULT 3600,        -- 1 hour
    tier2_seconds   INT         NOT NULL DEFAULT 86400,       -- 24 hours
    fee_grace_pct   NUMERIC(5,2) NOT NULL DEFAULT 0,         -- 0% during grace
    fee_tier1_pct   NUMERIC(5,2) NOT NULL DEFAULT 5,         -- 5% within tier1
    fee_tier2_pct   NUMERIC(5,2) NOT NULL DEFAULT 10,        -- 10% within tier2
    fee_max_pct     NUMERIC(5,2) NOT NULL DEFAULT 15,        -- 15% after tier2
    traveler_split  NUMERIC(5,2) NOT NULL DEFAULT 70,        -- 70% of fee → traveler
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(corridor_key)
);

-- ── User Cancellation Stats (anti-abuse) ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_cancellation_stats (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_orders        INT         NOT NULL DEFAULT 0,
    total_cancellations INT         NOT NULL DEFAULT 0,
    cancel_rate         NUMERIC(5,4) NOT NULL DEFAULT 0,     -- 0.0 – 1.0
    abuse_multiplier    NUMERIC(5,2) NOT NULL DEFAULT 1.0,   -- 1.0 = normal, 1.5 = penalized
    last_cancel_at      TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

-- ── Free Cancellation Tokens ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_cancellation_tokens (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    remaining_tokens  INT         NOT NULL DEFAULT 2,         -- 2 free cancels/month
    period_start      DATE        NOT NULL DEFAULT DATE_TRUNC('month', NOW())::date,
    period_end        DATE        NOT NULL DEFAULT (DATE_TRUNC('month', NOW()) + INTERVAL '1 month')::date,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, period_start)
);

-- ── Cancellation Ledger (audit trail) ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS cancellation_ledger (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id                UUID        NOT NULL REFERENCES orders(id),
    user_id                 UUID        NOT NULL REFERENCES users(id),
    fee_cents               BIGINT      NOT NULL DEFAULT 0,
    traveler_compensation   BIGINT      NOT NULL DEFAULT 0,
    platform_fee            BIGINT      NOT NULL DEFAULT 0,
    fee_percent             NUMERIC(5,2) NOT NULL DEFAULT 0,
    abuse_multiplier        NUMERIC(5,2) NOT NULL DEFAULT 1.0,
    token_used              BOOLEAN     NOT NULL DEFAULT FALSE,
    seconds_since_accept    INT         NOT NULL DEFAULT 0,
    tier                    VARCHAR(20) NOT NULL DEFAULT 'grace',  -- grace/tier1/tier2/max
    reason                  TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_cancel_stats_user ON user_cancellation_stats(user_id);
CREATE INDEX IF NOT EXISTS idx_cancel_tokens_user_period ON user_cancellation_tokens(user_id, period_start);
CREATE INDEX IF NOT EXISTS idx_cancel_ledger_order ON cancellation_ledger(order_id);
CREATE INDEX IF NOT EXISTS idx_cancel_ledger_user ON cancellation_ledger(user_id);
CREATE INDEX IF NOT EXISTS idx_cancel_policies_active ON cancellation_policies(is_active) WHERE is_active = TRUE;

-- Seed default global policy
INSERT INTO cancellation_policies (corridor_key, grace_seconds, tier1_seconds, tier2_seconds,
    fee_grace_pct, fee_tier1_pct, fee_tier2_pct, fee_max_pct, traveler_split)
VALUES ('global', 600, 3600, 86400, 0, 5, 10, 15, 70)
ON CONFLICT (corridor_key) DO NOTHING;
