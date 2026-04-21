-- Fraud Feedback Loop: closes the learning cycle
-- Sprint 1 — AI Fraud → Learning System

CREATE TABLE IF NOT EXISTS fraud_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    decision VARCHAR(20) NOT NULL CHECK (decision IN ('ALLOW', 'BLOCK', 'CHALLENGE')),
    outcome VARCHAR(20) NOT NULL CHECK (outcome IN ('LEGIT', 'FRAUD')),
    notes TEXT,
    reviewed_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fraud_feedback_event_id ON fraud_feedback(event_id);
CREATE INDEX idx_fraud_feedback_user_id ON fraud_feedback(user_id);
CREATE INDEX idx_fraud_feedback_outcome ON fraud_feedback(outcome);
CREATE INDEX idx_fraud_feedback_created ON fraud_feedback(created_at);
