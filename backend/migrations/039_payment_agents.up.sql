-- Sprint 6: Agent Payment System (Fawry + M-Pesa + Western Union Agents model)

CREATE TABLE payment_agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    status          VARCHAR(20) DEFAULT 'pending'
                    CHECK (status IN ('pending','active','suspended','terminated')),
    country         VARCHAR(3) NOT NULL,
    currency        VARCHAR(3) NOT NULL,
    balance_limit   DECIMAL(15,2) NOT NULL DEFAULT 10000.00,
    current_balance DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    collateral_held DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    trust_score     INTEGER NOT NULL DEFAULT 50
                    CHECK (trust_score BETWEEN 0 AND 100),
    payment_methods JSONB DEFAULT '[]',
    approved_by     UUID REFERENCES users(id),
    approved_at     TIMESTAMPTZ,
    suspension_reason TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_pa_user ON payment_agents(user_id);
CREATE INDEX idx_pa_country_currency ON payment_agents(country, currency)
    WHERE status = 'active';

-- Deposit requests (user → agent → wallet)

CREATE TABLE deposit_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    agent_id    UUID NOT NULL REFERENCES payment_agents(id),
    amount      DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    currency    VARCHAR(3) NOT NULL,
    usd_amount  DECIMAL(15,2) NOT NULL,
    fx_rate     DECIMAL(10,6) NOT NULL,
    status      VARCHAR(20) DEFAULT 'pending'
                CHECK (status IN ('pending','paid','confirmed','rejected','expired')),
    proof_url   VARCHAR(500),
    proof_uploaded_at TIMESTAMPTZ,
    confirmed_by_agent_at TIMESTAMPTZ,
    rejection_reason TEXT,
    idempotency_key VARCHAR(255) UNIQUE,
    expires_at  TIMESTAMPTZ DEFAULT NOW() + INTERVAL '30 minutes',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dr_user ON deposit_requests(user_id, created_at DESC);
CREATE INDEX idx_dr_agent ON deposit_requests(agent_id, status);
CREATE INDEX idx_dr_expires ON deposit_requests(expires_at)
    WHERE status = 'pending';

-- Withdraw requests (wallet → agent → user)

CREATE TABLE withdraw_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    agent_id    UUID REFERENCES payment_agents(id),
    amount      DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    currency    VARCHAR(3) NOT NULL,
    usd_amount  DECIMAL(15,2) NOT NULL,
    fx_rate     DECIMAL(10,6) NOT NULL,
    recipient_details JSONB NOT NULL,
    status      VARCHAR(20) DEFAULT 'pending'
                CHECK (status IN ('pending','assigned','processing','completed','failed','cancelled')),
    assigned_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failure_reason TEXT,
    idempotency_key VARCHAR(255) UNIQUE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_wr_user ON withdraw_requests(user_id, created_at DESC);
CREATE INDEX idx_wr_agent ON withdraw_requests(agent_id, status);

-- Agent liquidity log (full audit trail)

CREATE TABLE agent_liquidity_log (
    id          BIGSERIAL PRIMARY KEY,
    agent_id    UUID NOT NULL REFERENCES payment_agents(id),
    event_type  VARCHAR(30) NOT NULL,
    amount      DECIMAL(15,2) NOT NULL,
    balance_before DECIMAL(15,2) NOT NULL,
    balance_after  DECIMAL(15,2) NOT NULL,
    reference_id UUID,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_all_agent ON agent_liquidity_log(agent_id, created_at DESC);

-- VIP Users table

CREATE TABLE vip_users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) UNIQUE,
    tier            VARCHAR(20) DEFAULT 'silver'
                    CHECK (tier IN ('silver','gold','platinum')),
    daily_limit     DECIMAL(15,2) NOT NULL,
    monthly_limit   DECIMAL(15,2) NOT NULL,
    transfer_fee_pct DECIMAL(5,4) DEFAULT 0.005,
    priority_matching BOOLEAN DEFAULT TRUE,
    fast_track_kyc  BOOLEAN DEFAULT FALSE,
    dedicated_agent_id UUID REFERENCES payment_agents(id),
    activated_at    TIMESTAMPTZ DEFAULT NOW(),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
