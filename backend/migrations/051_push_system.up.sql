-- Push Notification System: device registry + delivery logs

-- Enhanced device registry (extends existing push_tokens concept)
CREATE TABLE IF NOT EXISTS user_devices (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_token    TEXT NOT NULL,
    platform        VARCHAR(20) NOT NULL CHECK (platform IN ('ios','android','web')),
    app_version     VARCHAR(20),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE(device_token)
);

CREATE INDEX idx_user_devices_user_id     ON user_devices(user_id);
CREATE INDEX idx_user_devices_is_active   ON user_devices(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_user_devices_deleted_at  ON user_devices(deleted_at) WHERE deleted_at IS NOT NULL;

-- Push delivery log for observability
CREATE TABLE IF NOT EXISTS push_logs (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id           UUID NOT NULL,
    device_token      TEXT NOT NULL,
    platform          VARCHAR(20),
    notification_type VARCHAR(50) NOT NULL,
    priority          VARCHAR(20) NOT NULL CHECK (priority IN ('high','medium','low')),
    title             VARCHAR(255),
    body              TEXT,
    data              JSONB,
    status            VARCHAR(20) NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','sent','failed','delivered','bounced')),
    provider_msg_id   VARCHAR(200),
    error_reason      TEXT,
    attempts          INT NOT NULL DEFAULT 1,
    idempotency_key   VARCHAR(200) UNIQUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_push_logs_user_id      ON push_logs(user_id);
CREATE INDEX idx_push_logs_status       ON push_logs(status);
CREATE INDEX idx_push_logs_priority     ON push_logs(priority);
CREATE INDEX idx_push_logs_notif_type   ON push_logs(notification_type);
CREATE INDEX idx_push_logs_created_at   ON push_logs(created_at);
CREATE INDEX idx_push_logs_device_token ON push_logs(device_token);
