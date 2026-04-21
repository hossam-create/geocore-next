-- ── Responsible Engagement Engine ────────────────────────────────────────────────
-- Session Momentum + Notification AI + Re-engagement + Timing

-- Session Momentum (real-time session tracking)
CREATE TABLE IF NOT EXISTS engagement_momentum (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID        NOT NULL,
    session_id      VARCHAR(50) NOT NULL UNIQUE,
    click_rate      NUMERIC(5,4) NOT NULL DEFAULT 0,
    bid_rate        NUMERIC(5,4) NOT NULL DEFAULT 0,
    time_on_item    NUMERIC(8,2) NOT NULL DEFAULT 0,
    scroll_velocity NUMERIC(8,2) NOT NULL DEFAULT 0,
    friction        NUMERIC(5,4) NOT NULL DEFAULT 0,
    momentum_score  NUMERIC(5,4) NOT NULL DEFAULT 0,
    feed_intensity  VARCHAR(20) NOT NULL DEFAULT 'balanced',
    views_count     INT         NOT NULL DEFAULT 0,
    clicks_count    INT         NOT NULL DEFAULT 0,
    bids_count      INT         NOT NULL DEFAULT 0,
    saves_count     INT         NOT NULL DEFAULT 0,
    purchases_count INT         NOT NULL DEFAULT 0,
    backs_count     INT         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Notification Events (audit trail for every notification)
CREATE TABLE IF NOT EXISTS engagement_notifications (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID        NOT NULL,
    event_type   VARCHAR(30) NOT NULL,
    channel      VARCHAR(20) NOT NULL DEFAULT 'push',
    score        NUMERIC(8,4) NOT NULL,
    reason       VARCHAR(200),
    opened       BOOLEAN     NOT NULL DEFAULT FALSE,
    acted        BOOLEAN     NOT NULL DEFAULT FALSE,
    opted_out    BOOLEAN     NOT NULL DEFAULT FALSE,
    sent_at      TIMESTAMPTZ,
    opened_at    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- User Engagement Profiles (per-user preferences + stats)
CREATE TABLE IF NOT EXISTS engagement_profiles (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                 UUID        NOT NULL UNIQUE,
    segment                 VARCHAR(20) NOT NULL DEFAULT 'active',
    last_active_at          TIMESTAMPTZ,
    notifications_today     INT         NOT NULL DEFAULT 0,
    notifications_this_week INT         NOT NULL DEFAULT 0,
    opt_out_all             BOOLEAN     NOT NULL DEFAULT FALSE,
    opt_out_push            BOOLEAN     NOT NULL DEFAULT FALSE,
    opt_out_email           BOOLEAN     NOT NULL DEFAULT FALSE,
    quiet_hours_start       INT         NOT NULL DEFAULT 22,
    quiet_hours_end         INT         NOT NULL DEFAULT 8,
    preferred_channels      VARCHAR(100) NOT NULL DEFAULT 'push,in_app',
    total_notifications_sent INT        NOT NULL DEFAULT 0,
    total_opened            INT         NOT NULL DEFAULT 0,
    total_acted             INT         NOT NULL DEFAULT 0,
    open_rate               NUMERIC(5,4) NOT NULL DEFAULT 0,
    act_rate                NUMERIC(5,4) NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Planned Touches (scheduled re-engagement actions)
CREATE TABLE IF NOT EXISTS engagement_planned_touches (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID        NOT NULL,
    segment      VARCHAR(20) NOT NULL,
    channel      VARCHAR(20) NOT NULL DEFAULT 'push',
    message_type VARCHAR(30) NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    sent_at      TIMESTAMPTZ,
    opened       BOOLEAN     NOT NULL DEFAULT FALSE,
    acted        BOOLEAN     NOT NULL DEFAULT FALSE,
    status       VARCHAR(20) NOT NULL DEFAULT 'planned',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- User Activity Hours (send-time optimization)
CREATE TABLE IF NOT EXISTS engagement_activity_hours (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID        NOT NULL,
    hour       INT         NOT NULL,
    day_of_week INT        NOT NULL DEFAULT -1,
    count      INT         NOT NULL DEFAULT 0,
    score      NUMERIC(5,4) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Engagement Config (global settings)
CREATE TABLE IF NOT EXISTS engagement_configs (
    id                           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    max_notifications_per_day     INT         NOT NULL DEFAULT 3,
    max_notifications_per_week    INT         NOT NULL DEFAULT 12,
    notification_score_threshold  NUMERIC(5,4) NOT NULL DEFAULT 0.3,
    quiet_hours_default_start     INT         NOT NULL DEFAULT 22,
    quiet_hours_default_end       INT         NOT NULL DEFAULT 8,
    exploration_percent           INT         NOT NULL DEFAULT 10,
    momentum_high_threshold       NUMERIC(5,4) NOT NULL DEFAULT 0.7,
    momentum_low_threshold        NUMERIC(5,4) NOT NULL DEFAULT 0.3,
    kill_switch_active            BOOLEAN     NOT NULL DEFAULT FALSE,
    is_active                     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_eng_momentum_user ON engagement_momentum(user_id);
CREATE INDEX IF NOT EXISTS idx_eng_notifications_user ON engagement_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_eng_notifications_type ON engagement_notifications(event_type);
CREATE INDEX IF NOT EXISTS idx_eng_notifications_sent ON engagement_notifications(sent_at);
CREATE INDEX IF NOT EXISTS idx_eng_profiles_segment ON engagement_profiles(segment);
CREATE INDEX IF NOT EXISTS idx_eng_touches_user ON engagement_planned_touches(user_id);
CREATE INDEX IF NOT EXISTS idx_eng_touches_scheduled ON engagement_planned_touches(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_eng_touches_status ON engagement_planned_touches(status);
CREATE INDEX IF NOT EXISTS idx_eng_activity_user ON engagement_activity_hours(user_id);
CREATE INDEX IF NOT EXISTS idx_eng_activity_hour ON engagement_activity_hours(hour);
