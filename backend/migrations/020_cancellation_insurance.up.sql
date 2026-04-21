-- Cancellation Insurance Engine
-- Optional opt-in insurance at checkout that reduces/waives cancellation fees.

-- ── Order Insurance ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS order_insurances (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id                UUID        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    user_id                 UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    price_cents             BIGINT      NOT NULL DEFAULT 0,
    coverage_type           VARCHAR(20) NOT NULL DEFAULT 'basic',  -- basic | plus | premium
    max_fee_covered_pct     NUMERIC(5,2) NOT NULL DEFAULT 100,    -- 100 = full waive, 98 = cap at 2%
    is_active               BOOLEAN     NOT NULL DEFAULT TRUE,
    is_used                 BOOLEAN     NOT NULL DEFAULT FALSE,    -- true once cancellation applied
    first_order_free        BOOLEAN     NOT NULL DEFAULT FALSE,    -- true if this was a free first-order insurance
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(order_id)
);

-- ── User Insurance Usage (anti-abuse) ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_insurance_usage (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    month               DATE        NOT NULL,                     -- e.g. 2026-04-01
    cancellations_used  INT         NOT NULL DEFAULT 0,           -- how many times insurance was used this month
    insurance_purchased INT         NOT NULL DEFAULT 0,           -- how many insurances bought this month
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, month)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_order_insurance_user ON order_insurances(user_id);
CREATE INDEX IF NOT EXISTS idx_order_insurance_active ON order_insurances(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_insurance_usage_user_month ON user_insurance_usage(user_id, month);
