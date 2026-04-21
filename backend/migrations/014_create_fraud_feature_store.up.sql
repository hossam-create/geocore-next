-- Fraud Engine: Feature Store + Audit Tables
-- Phase 1 — AI Fraud Engine

-- User risk profiles (existing table, add new columns)
ALTER TABLE user_risk_profiles ADD COLUMN IF NOT EXISTS tx_count_last_1h INT DEFAULT 0;
ALTER TABLE user_risk_profiles ADD COLUMN IF NOT EXISTS tx_count_last_24h INT DEFAULT 0;
ALTER TABLE user_risk_profiles ADD COLUMN IF NOT EXISTS withdraw_count_24h INT DEFAULT 0;
ALTER TABLE user_risk_profiles ADD COLUMN IF NOT EXISTS failed_logins_24h INT DEFAULT 0;
ALTER TABLE user_risk_profiles ADD COLUMN IF NOT EXISTS geo_mismatch_count INT DEFAULT 0;
ALTER TABLE user_risk_profiles ADD COLUMN IF NOT EXISTS device_count_7d INT DEFAULT 0;
ALTER TABLE user_risk_profiles ADD COLUMN IF NOT EXISTS wallet_drift_24h NUMERIC(14,2) DEFAULT 0;
ALTER TABLE user_risk_profiles ADD COLUMN IF NOT EXISTS last_login_ip VARCHAR(45);
ALTER TABLE user_risk_profiles ADD COLUMN IF NOT EXISTS last_login_country VARCHAR(3);

-- Fraud decision audit log
CREATE TABLE IF NOT EXISTS fraud_decision_audit (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    event_type VARCHAR(50) NOT NULL,
    decision VARCHAR(20) NOT NULL CHECK (decision IN ('ALLOW', 'CHALLENGE', 'BLOCK')),
    risk_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    risk_level VARCHAR(20) NOT NULL,
    signals JSONB DEFAULT '[]',
    request_id VARCHAR(100),
    trace_id VARCHAR(64),
    scoring_ms INT DEFAULT 0,
    feature_hit BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fraud_audit_user ON fraud_decision_audit(user_id);
CREATE INDEX idx_fraud_audit_decision ON fraud_decision_audit(decision);
CREATE INDEX idx_fraud_audit_created ON fraud_decision_audit(created_at);
CREATE INDEX idx_fraud_audit_trace ON fraud_decision_audit(trace_id);
