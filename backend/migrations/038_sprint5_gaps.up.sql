-- Sprint 5 Gaps: Trusted Agents + Delivery Confirmation + Order Evidence

-- Trusted agents for P2P currency matching
CREATE TABLE IF NOT EXISTS trusted_agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE,
    kyc_verified BOOLEAN NOT NULL DEFAULT false,
    deposit_guarantee NUMERIC(14,2) NOT NULL DEFAULT 0,
    max_daily_volume NUMERIC(14,2) NOT NULL DEFAULT 5000,
    is_active BOOLEAN NOT NULL DEFAULT true,
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    total_transactions INT NOT NULL DEFAULT 0,
    total_volume NUMERIC(14,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trusted_agents_user_id ON trusted_agents(user_id);
CREATE INDEX IF NOT EXISTS idx_trusted_agents_active ON trusted_agents(is_active) WHERE is_active = true;

-- Agent match requests (P2P currency matching, NOT auto-matched)
CREATE TABLE IF NOT EXISTS agent_match_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    agent_id UUID NOT NULL,
    amount NUMERIC(14,2) NOT NULL,
    from_currency VARCHAR(3) NOT NULL,
    to_currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewed_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_match_requests_user_id ON agent_match_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_match_requests_agent_id ON agent_match_requests(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_match_requests_status ON agent_match_requests(status);

-- Order evidence (delivery confirmation photos, tracking)
CREATE TABLE IF NOT EXISTS order_evidence (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL,
    submitted_by UUID NOT NULL,
    type VARCHAR(30) NOT NULL, -- delivery_photo, tracking_confirmation, receipt
    url TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_order_evidence_order_id ON order_evidence(order_id);
