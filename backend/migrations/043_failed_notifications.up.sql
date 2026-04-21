-- Sprint 8.5: Failed notification log for fallback tracking
CREATE TABLE IF NOT EXISTS failed_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    channel VARCHAR(20) NOT NULL,          -- 'push', 'email', 'sms'
    event_type VARCHAR(100) NOT NULL,      -- e.g. 'offer_accepted', 'escrow_released'
    payload JSONB,                          -- original notification payload
    error_message TEXT,                     -- why delivery failed
    retry_count INT NOT NULL DEFAULT 0,
    last_retry_at TIMESTAMP,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_failed_notifications_user ON failed_notifications(user_id);
CREATE INDEX idx_failed_notifications_unresolved ON failed_notifications(resolved) WHERE resolved = FALSE;
CREATE INDEX idx_failed_notifications_channel ON failed_notifications(channel);
