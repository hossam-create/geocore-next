-- ── Growth Engine + Messaging + Experiments ────────────────────────────────────────

-- User State (real-time session brain)
CREATE TABLE IF NOT EXISTS growth_user_states (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID        NOT NULL UNIQUE,
    last_active_at      TIMESTAMPTZ,
    session_duration_sec NUMERIC(12,2) NOT NULL DEFAULT 0,
    actions_last_5m     INT         NOT NULL DEFAULT 0,
    actions_last_1h     INT         NOT NULL DEFAULT 0,
    bids_count          INT         NOT NULL DEFAULT 0,
    purchases_count     INT         NOT NULL DEFAULT 0,
    views_count         INT         NOT NULL DEFAULT 0,
    saves_count         INT         NOT NULL DEFAULT 0,
    losses_count        INT         NOT NULL DEFAULT 0,
    drop_off_risk_score NUMERIC(5,4) NOT NULL DEFAULT 0,
    engagement_score    NUMERIC(8,2) NOT NULL DEFAULT 50,
    dopamine_score      NUMERIC(8,2) NOT NULL DEFAULT 50,
    segment             VARCHAR(20) NOT NULL DEFAULT 'active',
    preferred_channel   VARCHAR(20) NOT NULL DEFAULT 'push',
    experiment_group    VARCHAR(20) NOT NULL DEFAULT 'control',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Action Events (raw event log)
CREATE TABLE IF NOT EXISTS growth_action_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL,
    action      VARCHAR(30) NOT NULL,
    item_id     UUID,
    session_id  VARCHAR(50),
    metadata    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Dopamine Events
CREATE TABLE IF NOT EXISTS growth_dopamine_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL,
    event_type  VARCHAR(30) NOT NULL,
    delta       NUMERIC(8,2) NOT NULL,
    old_score   NUMERIC(8,2) NOT NULL,
    new_score   NUMERIC(8,2) NOT NULL,
    item_id     UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Re-engagement Logs
CREATE TABLE IF NOT EXISTS growth_reengagement_logs (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID        NOT NULL,
    segment      VARCHAR(20) NOT NULL,
    action_type  VARCHAR(30) NOT NULL,
    channel      VARCHAR(20) NOT NULL,
    success      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Decision Logs
CREATE TABLE IF NOT EXISTS growth_decision_logs (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL,
    action      VARCHAR(30) NOT NULL,
    confidence  NUMERIC(5,4) NOT NULL,
    reason      VARCHAR(200),
    sources     TEXT,
    outcome     VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at  BIGINT
);

-- ── Messaging ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS messaging_messages (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID        NOT NULL,
    type         VARCHAR(30) NOT NULL,
    title        VARCHAR(200) NOT NULL,
    body         TEXT        NOT NULL,
    priority     VARCHAR(10) NOT NULL DEFAULT 'normal',
    channel      VARCHAR(20) NOT NULL DEFAULT 'push',
    metadata     TEXT,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    sent_at      TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    opened_at    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS messaging_cooldowns (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID        NOT NULL,
    msg_type     VARCHAR(30) NOT NULL,
    last_sent_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS messaging_user_prefs (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id           UUID        NOT NULL UNIQUE,
    opt_out_push      BOOLEAN     NOT NULL DEFAULT FALSE,
    opt_out_email     BOOLEAN     NOT NULL DEFAULT FALSE,
    opt_out_all       BOOLEAN     NOT NULL DEFAULT FALSE,
    quiet_hours_start INT         NOT NULL DEFAULT 22,
    quiet_hours_end   INT         NOT NULL DEFAULT 8,
    max_per_hour      INT         NOT NULL DEFAULT 3,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Experiments ─────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS experiments (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name          VARCHAR(100) NOT NULL UNIQUE,
    variants      TEXT         NOT NULL,
    traffic_split TEXT         NOT NULL,
    metric        VARCHAR(30)  NOT NULL DEFAULT 'ctr',
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    started_at    TIMESTAMPTZ,
    ended_at      TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS experiment_assignments (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    experiment_id  UUID NOT NULL,
    user_id        UUID NOT NULL,
    variant        VARCHAR(20) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS experiment_events (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    experiment_id  UUID        NOT NULL,
    user_id        UUID        NOT NULL,
    variant        VARCHAR(20) NOT NULL,
    event_type     VARCHAR(30) NOT NULL,
    value          NUMERIC(12,4) NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS experiment_bandit_arms (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    experiment_id  UUID        NOT NULL,
    arm_name       VARCHAR(50) NOT NULL,
    alpha          NUMERIC(12,4) NOT NULL DEFAULT 1,
    beta           NUMERIC(12,4) NOT NULL DEFAULT 1,
    total_pulls    INT         NOT NULL DEFAULT 0,
    total_reward   NUMERIC(12,4) NOT NULL DEFAULT 0,
    avg_reward     NUMERIC(8,4) NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS experiment_bandit_pulls (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    experiment_id  UUID        NOT NULL,
    arm_id         UUID        NOT NULL,
    user_id        UUID        NOT NULL,
    reward         NUMERIC(8,4) NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Indexes ─────────────────────────────────────────────────────────────────────────

CREATE INDEX IF NOT EXISTS idx_growth_state_user ON growth_user_states(user_id);
CREATE INDEX IF NOT EXISTS idx_growth_state_segment ON growth_user_states(segment);
CREATE INDEX IF NOT EXISTS idx_growth_events_user ON growth_action_events(user_id);
CREATE INDEX IF NOT EXISTS idx_growth_events_action ON growth_action_events(action);
CREATE INDEX IF NOT EXISTS idx_growth_dopamine_user ON growth_dopamine_events(user_id);
CREATE INDEX IF NOT EXISTS idx_growth_dopamine_type ON growth_dopamine_events(event_type);
CREATE INDEX IF NOT EXISTS idx_growth_reengage_user ON growth_reengagement_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_growth_reengage_segment ON growth_reengagement_logs(segment);
CREATE INDEX IF NOT EXISTS idx_growth_decision_user ON growth_decision_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_msg_messages_user ON messaging_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_msg_messages_type ON messaging_messages(type);
CREATE INDEX IF NOT EXISTS idx_msg_messages_sent ON messaging_messages(sent_at);
CREATE INDEX IF NOT EXISTS idx_msg_cooldowns_user ON messaging_cooldowns(user_id, msg_type);
CREATE INDEX IF NOT EXISTS idx_msg_prefs_user ON messaging_user_prefs(user_id);
CREATE INDEX IF NOT EXISTS idx_exp_assign_user ON experiment_assignments(experiment_id, user_id);
CREATE INDEX IF NOT EXISTS idx_exp_events_exp ON experiment_events(experiment_id);
CREATE INDEX IF NOT EXISTS idx_exp_events_type ON experiment_events(event_type);
CREATE INDEX IF NOT EXISTS idx_exp_bandit_arms_exp ON experiment_bandit_arms(experiment_id);
CREATE INDEX IF NOT EXISTS idx_exp_bandit_pulls_exp ON experiment_bandit_pulls(experiment_id);
