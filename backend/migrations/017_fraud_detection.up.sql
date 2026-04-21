-- Fraud Detection & Prevention

CREATE TYPE fraud_severity AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE fraud_alert_status AS ENUM ('pending', 'investigating', 'confirmed', 'false_positive', 'resolved');
CREATE TYPE fraud_target_type AS ENUM ('user', 'order', 'transaction', 'listing', 'review');

CREATE TABLE IF NOT EXISTS fraud_alerts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type     fraud_target_type NOT NULL,
    target_id       UUID NOT NULL,
    alert_type      VARCHAR(100) NOT NULL,
    severity        fraud_severity NOT NULL DEFAULT 'medium',
    risk_score      NUMERIC(5,2) NOT NULL DEFAULT 0,
    detected_by     VARCHAR(100) NOT NULL DEFAULT 'rule_engine',
    confidence      NUMERIC(4,3) NOT NULL DEFAULT 0,
    indicators      JSONB NOT NULL DEFAULT '[]',
    raw_data        JSONB,
    status          fraud_alert_status NOT NULL DEFAULT 'pending',
    reviewed_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at     TIMESTAMPTZ,
    resolution      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fraud_alerts_target  ON fraud_alerts(target_type, target_id);
CREATE INDEX idx_fraud_alerts_status  ON fraud_alerts(status, severity);
CREATE INDEX idx_fraud_alerts_created ON fraud_alerts(created_at DESC);

CREATE TABLE IF NOT EXISTS fraud_rules (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(200) NOT NULL UNIQUE,
    description     TEXT,
    rule_type       VARCHAR(50) NOT NULL,
    conditions      JSONB NOT NULL DEFAULT '{}',
    severity        fraud_severity NOT NULL DEFAULT 'medium',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_risk_profiles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    risk_score      NUMERIC(5,2) NOT NULL DEFAULT 0,
    risk_level      VARCHAR(20) NOT NULL DEFAULT 'low',
    total_orders    INTEGER NOT NULL DEFAULT 0,
    total_spent     NUMERIC(14,2) NOT NULL DEFAULT 0,
    avg_order_value NUMERIC(12,2) NOT NULL DEFAULT 0,
    flags           JSONB NOT NULL DEFAULT '[]',
    last_assessed   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_risk_profiles_risk ON user_risk_profiles(risk_score DESC);

-- Seed default fraud rules
INSERT INTO fraud_rules (name, description, rule_type, conditions, severity) VALUES
('high_velocity_hourly', 'More than 5 transactions per hour', 'velocity', '{"period":"1h","max_count":5}', 'high'),
('high_amount_single', 'Single transaction over 5000', 'amount', '{"max_amount":5000}', 'high'),
('high_amount_daily', 'Daily total over 10000', 'amount', '{"period":"24h","max_total":10000}', 'critical'),
('new_account_high_value', 'High-value order within 24h of registration', 'behavior', '{"account_age_hours":24,"min_amount":1000}', 'medium'),
('country_mismatch', 'Billing and shipping country mismatch', 'location', '{"check":"country_mismatch"}', 'medium'),
('multiple_failed_payments', 'More than 3 failed payments in 1 hour', 'velocity', '{"period":"1h","max_failures":3}', 'high')
ON CONFLICT (name) DO NOTHING;
