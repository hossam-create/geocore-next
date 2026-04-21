-- Sprint 5: Trust + Reputation + Anti-Fraud Engine

-- User reputation table (per role)
CREATE TABLE IF NOT EXISTS user_reputations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    role VARCHAR(20) NOT NULL,
    score NUMERIC(5,2) NOT NULL DEFAULT 50,
    total_orders INT NOT NULL DEFAULT 0,
    completed_orders INT NOT NULL DEFAULT 0,
    cancelled_orders INT NOT NULL DEFAULT 0,
    dispute_count INT NOT NULL DEFAULT 0,
    avg_rating NUMERIC(3,2) NOT NULL DEFAULT 3.00,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, role)
);

CREATE INDEX IF NOT EXISTS idx_user_reputations_user_id ON user_reputations(user_id);
CREATE INDEX IF NOT EXISTS idx_user_reputations_score ON user_reputations(score);

-- Penalty audit log
CREATE TABLE IF NOT EXISTS penalty_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    role VARCHAR(20) NOT NULL,
    reason VARCHAR(50) NOT NULL,
    delta NUMERIC(6,2) NOT NULL,
    new_score NUMERIC(5,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_penalty_logs_user_id ON penalty_logs(user_id);

-- Add traveler_id to disputes if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='disputes' AND column_name='traveler_id') THEN
        ALTER TABLE disputes ADD COLUMN traveler_id UUID;
    END IF;
END $$;
