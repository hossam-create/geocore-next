-- Sprint 7: Payment Matching + Agent Economy + Financial Safety

-- P2P Money Matching Results
CREATE TABLE payment_match_results (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deposit_id  UUID NOT NULL REFERENCES deposit_requests(id),
    withdraw_id UUID NOT NULL REFERENCES withdraw_requests(id),
    amount_cents BIGINT NOT NULL,
    amount      DECIMAL(15,2) NOT NULL,
    rate        DECIMAL(10,6) NOT NULL DEFAULT 1.0,
    status      VARCHAR(20) NOT NULL DEFAULT 'settled',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_pmr_deposit ON payment_match_results(deposit_id);
CREATE INDEX idx_pmr_withdraw ON payment_match_results(withdraw_id);

-- Agent Reputation Scores
CREATE TABLE payment_agent_scores (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id     UUID NOT NULL REFERENCES payment_agents(id) UNIQUE,
    success_rate DECIMAL(5,2) NOT NULL DEFAULT 100.00,
    dispute_rate DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    volume       DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    score        INTEGER NOT NULL DEFAULT 50 CHECK (score BETWEEN 0 AND 100),
    total_tx     INTEGER NOT NULL DEFAULT 0,
    success_tx   INTEGER NOT NULL DEFAULT 0,
    dispute_tx   INTEGER NOT NULL DEFAULT 0,
    fraud_flags  INTEGER NOT NULL DEFAULT 0,
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

-- Payment Disputes
CREATE TABLE payment_disputes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id),
    agent_id     UUID NOT NULL REFERENCES payment_agents(id),
    deposit_id   UUID REFERENCES deposit_requests(id),
    withdraw_id  UUID REFERENCES withdraw_requests(id),
    amount_cents BIGINT NOT NULL,
    amount       DECIMAL(15,2) NOT NULL,
    reason       VARCHAR(50) NOT NULL,
    proof_image  VARCHAR(500),
    status       VARCHAR(20) NOT NULL DEFAULT 'open',
    resolution   VARCHAR(50),
    resolved_by  UUID REFERENCES users(id),
    resolved_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_pd_user ON payment_disputes(user_id, created_at DESC);
CREATE INDEX idx_pd_agent ON payment_disputes(agent_id, status);
CREATE INDEX idx_pd_status ON payment_disputes(status);
