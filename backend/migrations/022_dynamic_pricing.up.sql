-- Dynamic Insurance Pricing AI
-- Per-user insurance pricing with ML model support + A/B testing

-- ── Pricing Model Config ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS pricing_model_configs (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    version               VARCHAR(20) NOT NULL,
    strategy              VARCHAR(20) NOT NULL DEFAULT 'rules',  -- static | rules | ai
    min_price_percent     NUMERIC(5,2) NOT NULL DEFAULT 1,
    max_price_percent     NUMERIC(5,2) NOT NULL DEFAULT 4,
    base_price_percent    NUMERIC(5,2) NOT NULL DEFAULT 1.5,
    static_price_percent  NUMERIC(5,2) NOT NULL DEFAULT 2,
    confidence_threshold  NUMERIC(5,2) NOT NULL DEFAULT 0.7,
    model_json            TEXT,
    is_active             BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Pricing Events (training + tracking) ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS pricing_events (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID        NOT NULL,
    order_id        UUID        NOT NULL,
    strategy        VARCHAR(20) NOT NULL,  -- static | rules | ai
    price_cents     BIGINT      NOT NULL,
    buy_probability NUMERIC(5,4),
    confidence      NUMERIC(5,4),
    did_buy         BOOLEAN     NOT NULL DEFAULT FALSE,
    did_cancel      BOOLEAN     NOT NULL DEFAULT FALSE,
    claim_filed     BOOLEAN     NOT NULL DEFAULT FALSE,
    ab_variant      VARCHAR(20) NOT NULL DEFAULT 'control',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Pricing A/B Assignments ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS pricing_ab_assignments (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL,
    experiment  VARCHAR(50) NOT NULL,
    variant     VARCHAR(20) NOT NULL,  -- static | rules | ai
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, experiment)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_pricing_events_user ON pricing_events(user_id);
CREATE INDEX IF NOT EXISTS idx_pricing_events_strategy ON pricing_events(strategy);
CREATE INDEX IF NOT EXISTS idx_pricing_events_created ON pricing_events(created_at);
CREATE INDEX IF NOT EXISTS idx_pricing_ab_user_exp ON pricing_ab_assignments(user_id, experiment);

-- ── Multi-Armed Bandit Tables ──────────────────────────────────────────────────

-- Bandit Arms: price points per segment with Thompson Sampling priors
CREATE TABLE IF NOT EXISTS bandit_arms (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    segment       VARCHAR(50) NOT NULL,
    price_percent NUMERIC(5,2) NOT NULL,
    impressions   BIGINT      NOT NULL DEFAULT 0,
    conversions   BIGINT      NOT NULL DEFAULT 0,
    total_reward  NUMERIC(12,2) NOT NULL DEFAULT 0,
    alpha         NUMERIC(8,2) NOT NULL DEFAULT 1,  -- Beta dist α (successes + 1)
    beta          NUMERIC(8,2) NOT NULL DEFAULT 1,  -- Beta dist β (failures + 1)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(segment, price_percent)
);

-- Bandit Events: impressions + outcomes for learning
CREATE TABLE IF NOT EXISTS bandit_events (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID        NOT NULL,
    order_id      UUID        NOT NULL,
    segment       VARCHAR(50) NOT NULL,
    arm_id        UUID        NOT NULL,
    price_percent NUMERIC(5,2) NOT NULL,
    price_cents   BIGINT      NOT NULL,
    did_buy       BOOLEAN     NOT NULL DEFAULT FALSE,
    reward        NUMERIC(12,2) NOT NULL DEFAULT 0,
    claim_cost    NUMERIC(12,2) NOT NULL DEFAULT 0,
    algorithm     VARCHAR(20) NOT NULL DEFAULT 'thompson',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Bandit Config: algorithm settings + kill switch
CREATE TABLE IF NOT EXISTS bandit_configs (
    id                            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    algorithm                     VARCHAR(20) NOT NULL DEFAULT 'thompson',
    epsilon                       NUMERIC(5,2) NOT NULL DEFAULT 0.2,
    min_price_percent             NUMERIC(5,2) NOT NULL DEFAULT 1,
    max_price_percent             NUMERIC(5,2) NOT NULL DEFAULT 4,
    conversion_drop_threshold     NUMERIC(5,2) NOT NULL DEFAULT 0.10,
    session_cooldown_minutes      INT         NOT NULL DEFAULT 5,
    min_impressions_before_exploit INT        NOT NULL DEFAULT 100,
    is_active                     BOOLEAN     NOT NULL DEFAULT TRUE,
    kill_switch_active            BOOLEAN     NOT NULL DEFAULT FALSE,
    fallback_price_percent        NUMERIC(5,2) NOT NULL DEFAULT 2,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Bandit indexes
CREATE INDEX IF NOT EXISTS idx_bandit_arms_segment ON bandit_arms(segment);
CREATE INDEX IF NOT EXISTS idx_bandit_events_user ON bandit_events(user_id);
CREATE INDEX IF NOT EXISTS idx_bandit_events_segment ON bandit_events(segment);
CREATE INDEX IF NOT EXISTS idx_bandit_events_arm ON bandit_events(arm_id);
CREATE INDEX IF NOT EXISTS idx_bandit_events_created ON bandit_events(created_at);

-- ── Reinforcement Learning Tables ────────────────────────────────────────────────

-- RL Transitions: (s, a, r, s') for offline training
CREATE TABLE IF NOT EXISTS rl_transitions (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID        NOT NULL,
    order_id         UUID        NOT NULL,
    session_id       VARCHAR(50) NOT NULL,

    -- State
    state_key        VARCHAR(100) NOT NULL,
    state_json       TEXT,

    -- Action
    action_index     INT         NOT NULL,
    price_percent    NUMERIC(5,2) NOT NULL,
    ux_variant       VARCHAR(30) NOT NULL,
    price_cents      BIGINT      NOT NULL,

    -- Reward components
    reward_revenue   NUMERIC(12,2) NOT NULL DEFAULT 0,
    reward_claim_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    reward_churn     NUMERIC(12,2) NOT NULL DEFAULT 0,
    reward_total     NUMERIC(12,2) NOT NULL DEFAULT 0,

    -- Outcome
    did_buy          BOOLEAN     NOT NULL DEFAULT FALSE,
    did_claim        BOOLEAN     NOT NULL DEFAULT FALSE,
    did_churn        BOOLEAN     NOT NULL DEFAULT FALSE,

    -- Next state
    next_state_key   VARCHAR(100) NOT NULL DEFAULT '',
    next_state_json  TEXT,

    -- Episode
    is_terminal      BOOLEAN     NOT NULL DEFAULT FALSE,
    episode_id       VARCHAR(50),

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RL Sessions: sequential interactions per user+order
CREATE TABLE IF NOT EXISTS rl_sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID        NOT NULL,
    order_id        UUID        NOT NULL,
    episode_id      VARCHAR(50) NOT NULL,
    current_step    INT         NOT NULL DEFAULT 0,
    previous_offers TEXT,
    refusal_count   INT         NOT NULL DEFAULT 0,
    last_action_idx INT         NOT NULL DEFAULT -1,
    total_reward    NUMERIC(12,2) NOT NULL DEFAULT 0,
    is_complete     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, order_id)
);

-- RL Config: algorithm settings + rollout phase + kill switch
CREATE TABLE IF NOT EXISTS rl_configs (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    algorithm                  VARCHAR(20) NOT NULL DEFAULT 'q_learning',
    learning_rate              NUMERIC(5,4) NOT NULL DEFAULT 0.1,
    discount_factor            NUMERIC(5,4) NOT NULL DEFAULT 0.95,
    epsilon                    NUMERIC(5,4) NOT NULL DEFAULT 0.1,
    epsilon_decay              NUMERIC(5,4) NOT NULL DEFAULT 0.995,
    min_epsilon                NUMERIC(5,4) NOT NULL DEFAULT 0.05,
    churn_penalty              NUMERIC(8,2) NOT NULL DEFAULT 5.0,
    min_price_percent          NUMERIC(5,2) NOT NULL DEFAULT 1,
    max_price_percent          NUMERIC(5,2) NOT NULL DEFAULT 4,
    conversion_drop_threshold  NUMERIC(5,2) NOT NULL DEFAULT 0.08,
    session_cooldown_minutes   INT         NOT NULL DEFAULT 5,
    max_session_steps          INT         NOT NULL DEFAULT 3,
    rollout_phase              VARCHAR(20) NOT NULL DEFAULT 'shadow',
    kill_switch_active         BOOLEAN     NOT NULL DEFAULT FALSE,
    fallback_price_percent     NUMERIC(5,2) NOT NULL DEFAULT 2,
    is_active                  BOOLEAN     NOT NULL DEFAULT TRUE,
    q_table_json               TEXT,
    policy_json                TEXT,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RL indexes
CREATE INDEX IF NOT EXISTS idx_rl_transitions_user ON rl_transitions(user_id);
CREATE INDEX IF NOT EXISTS idx_rl_transitions_session ON rl_transitions(session_id);
CREATE INDEX IF NOT EXISTS idx_rl_transitions_state ON rl_transitions(state_key);
CREATE INDEX IF NOT EXISTS idx_rl_transitions_episode ON rl_transitions(episode_id);
CREATE INDEX IF NOT EXISTS idx_rl_sessions_user ON rl_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_rl_sessions_episode ON rl_sessions(episode_id);

-- ── Hybrid Pricing Engine Tables ────────────────────────────────────────────────

-- Hybrid Config: orchestrates RL + Bandit + Rules
CREATE TABLE IF NOT EXISTS hybrid_configs (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rl_confidence_threshold    NUMERIC(5,2) NOT NULL DEFAULT 0.6,
    blend_weight_rl            NUMERIC(5,2) NOT NULL DEFAULT 0.7,
    enable_soft_blend          BOOLEAN     NOT NULL DEFAULT FALSE,
    min_price_percent          NUMERIC(5,2) NOT NULL DEFAULT 1,
    max_price_percent          NUMERIC(5,2) NOT NULL DEFAULT 4,
    emergency_mode_active      BOOLEAN     NOT NULL DEFAULT FALSE,
    emergency_price_percent    NUMERIC(5,2) NOT NULL DEFAULT 2,
    conversion_drop_threshold  NUMERIC(5,2) NOT NULL DEFAULT 0.08,
    session_cooldown_minutes   INT         NOT NULL DEFAULT 5,
    max_session_steps          INT         NOT NULL DEFAULT 3,
    anomaly_detection_enabled  BOOLEAN     NOT NULL DEFAULT TRUE,
    rollout_percent            INT         NOT NULL DEFAULT 5,
    is_active                  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hybrid Events: full observability (source, confidence, guardrails)
CREATE TABLE IF NOT EXISTS hybrid_events (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID        NOT NULL,
    order_id        UUID        NOT NULL,
    source          VARCHAR(20) NOT NULL,  -- rules/rl/bandit/blend/emergency/session/shadow
    price_cents     BIGINT      NOT NULL,
    price_percent   NUMERIC(5,2) NOT NULL,
    confidence      NUMERIC(5,4) NOT NULL,
    is_exploration  BOOLEAN     NOT NULL DEFAULT FALSE,
    is_shadow       BOOLEAN     NOT NULL DEFAULT FALSE,
    ux_variant      VARCHAR(30) NOT NULL DEFAULT 'standard',
    guardrails_json TEXT,
    did_buy         BOOLEAN     NOT NULL DEFAULT FALSE,
    reward          NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hybrid indexes
CREATE INDEX IF NOT EXISTS idx_hybrid_events_user ON hybrid_events(user_id);
CREATE INDEX IF NOT EXISTS idx_hybrid_events_source ON hybrid_events(source);
CREATE INDEX IF NOT EXISTS idx_hybrid_events_created ON hybrid_events(created_at);

-- ── Cross-System RL Coordinator Tables ──────────────────────────────────────────

-- Cross Config: multi-objective reward weights + consistency rules
CREATE TABLE IF NOT EXISTS cross_configs (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    weight_gmv                 NUMERIC(5,2) NOT NULL DEFAULT 0.4,
    weight_ctr                 NUMERIC(5,2) NOT NULL DEFAULT 0.2,
    weight_claim_cost          NUMERIC(5,2) NOT NULL DEFAULT 0.2,
    weight_churn               NUMERIC(5,2) NOT NULL DEFAULT 0.2,
    learning_rate              NUMERIC(5,4) NOT NULL DEFAULT 0.1,
    discount_factor            NUMERIC(5,4) NOT NULL DEFAULT 0.95,
    epsilon                    NUMERIC(5,4) NOT NULL DEFAULT 0.1,
    confidence_threshold       NUMERIC(5,2) NOT NULL DEFAULT 0.6,
    min_price_percent          NUMERIC(5,2) NOT NULL DEFAULT 1,
    max_price_percent          NUMERIC(5,2) NOT NULL DEFAULT 4,
    max_boost_with_high_price  INT         NOT NULL DEFAULT 30,
    high_price_threshold       NUMERIC(5,2) NOT NULL DEFAULT 3.0,
    emergency_mode_active      BOOLEAN     NOT NULL DEFAULT FALSE,
    conversion_drop_threshold  NUMERIC(5,2) NOT NULL DEFAULT 0.08,
    session_cooldown_minutes   INT         NOT NULL DEFAULT 5,
    max_session_steps          INT         NOT NULL DEFAULT 3,
    anomaly_detection_enabled  BOOLEAN     NOT NULL DEFAULT TRUE,
    rollout_percent            INT         NOT NULL DEFAULT 5,
    fallback_price_percent     NUMERIC(5,2) NOT NULL DEFAULT 2,
    fallback_boost_score       INT         NOT NULL DEFAULT 50,
    fallback_rec_strategy      VARCHAR(20) NOT NULL DEFAULT 'popular',
    is_active                  BOOLEAN     NOT NULL DEFAULT TRUE,
    q_table_json               TEXT,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cross Transitions: (s, a_bundle, r, s') for offline training
CREATE TABLE IF NOT EXISTS cross_transitions (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID        NOT NULL,
    order_id         UUID        NOT NULL,
    session_id       VARCHAR(50) NOT NULL,
    state_key        VARCHAR(120) NOT NULL,
    state_json       TEXT,
    price_cents      BIGINT      NOT NULL,
    price_percent    NUMERIC(5,2) NOT NULL,
    boost_score      INT         NOT NULL,
    rec_ids_json     TEXT,
    rec_strategy     VARCHAR(20) NOT NULL,
    reward_gmv       NUMERIC(12,2) NOT NULL DEFAULT 0,
    reward_ctr       NUMERIC(12,2) NOT NULL DEFAULT 0,
    reward_claim_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    reward_churn     NUMERIC(12,2) NOT NULL DEFAULT 0,
    reward_total     NUMERIC(12,2) NOT NULL DEFAULT 0,
    did_buy          BOOLEAN     NOT NULL DEFAULT FALSE,
    did_click        BOOLEAN     NOT NULL DEFAULT FALSE,
    did_claim        BOOLEAN     NOT NULL DEFAULT FALSE,
    did_churn        BOOLEAN     NOT NULL DEFAULT FALSE,
    next_state_key   VARCHAR(120) NOT NULL DEFAULT '',
    is_terminal      BOOLEAN     NOT NULL DEFAULT FALSE,
    episode_id       VARCHAR(50),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cross Events: full observability
CREATE TABLE IF NOT EXISTS cross_events (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID        NOT NULL,
    order_id        UUID        NOT NULL,
    source_pricing  VARCHAR(20) NOT NULL DEFAULT 'rules',
    source_ranking  VARCHAR(20) NOT NULL DEFAULT 'heuristic',
    source_recs     VARCHAR(20) NOT NULL DEFAULT 'popular',
    price_cents     BIGINT      NOT NULL,
    price_percent   NUMERIC(5,2) NOT NULL,
    boost_score     INT         NOT NULL,
    rec_ids_json    TEXT,
    confidence      NUMERIC(5,4) NOT NULL,
    is_shadow       BOOLEAN     NOT NULL DEFAULT FALSE,
    guardrails_json TEXT,
    did_buy         BOOLEAN     NOT NULL DEFAULT FALSE,
    did_click       BOOLEAN     NOT NULL DEFAULT FALSE,
    reward          NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cross indexes
CREATE INDEX IF NOT EXISTS idx_cross_transitions_user ON cross_transitions(user_id);
CREATE INDEX IF NOT EXISTS idx_cross_transitions_state ON cross_transitions(state_key);
CREATE INDEX IF NOT EXISTS idx_cross_transitions_episode ON cross_transitions(episode_id);
CREATE INDEX IF NOT EXISTS idx_cross_events_user ON cross_events(user_id);
CREATE INDEX IF NOT EXISTS idx_cross_events_created ON cross_events(created_at);

-- ── Feature Store + Embeddings + Retrieval Tables ────────────────────────────────

-- Enable pgvector extension (for similarity search)
CREATE EXTENSION IF NOT EXISTS vector;

-- User Features (online feature store)
CREATE TABLE IF NOT EXISTS feature_users (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id            UUID        NOT NULL UNIQUE,
    trust_score        NUMERIC(5,2) NOT NULL DEFAULT 50,
    avg_order_value    NUMERIC(12,2) NOT NULL DEFAULT 0,
    cancel_rate        NUMERIC(5,4) NOT NULL DEFAULT 0,
    insurance_buy_rate NUMERIC(5,4) NOT NULL DEFAULT 0,
    total_orders       BIGINT      NOT NULL DEFAULT 0,
    total_spent        NUMERIC(12,2) NOT NULL DEFAULT 0,
    account_age_days   NUMERIC(8,1) NOT NULL DEFAULT 0,
    abuse_flags        INT         NOT NULL DEFAULT 0,
    last_order_at      TIMESTAMPTZ,
    segment            VARCHAR(20) NOT NULL DEFAULT 'regular',
    embedding_id       UUID,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Item Features (online feature store)
CREATE TABLE IF NOT EXISTS feature_items (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id          UUID        NOT NULL UNIQUE,
    price_cents      BIGINT      NOT NULL DEFAULT 0,
    category_path    VARCHAR(200) NOT NULL DEFAULT '',
    view_count       BIGINT      NOT NULL DEFAULT 0,
    purchase_count   BIGINT      NOT NULL DEFAULT 0,
    avg_rating       NUMERIC(3,2) NOT NULL DEFAULT 0,
    claim_rate       NUMERIC(5,4) NOT NULL DEFAULT 0,
    delivery_risk    NUMERIC(5,4) NOT NULL DEFAULT 0,
    popularity_score NUMERIC(8,4) NOT NULL DEFAULT 0,
    is_trending      BOOLEAN     NOT NULL DEFAULT FALSE,
    embedding_id     UUID,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Session Features (real-time session context)
CREATE TABLE IF NOT EXISTS feature_sessions (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id        UUID        NOT NULL,
    session_id     VARCHAR(50) NOT NULL UNIQUE,
    device         VARCHAR(20) NOT NULL DEFAULT 'desktop',
    geo            VARCHAR(5)  NOT NULL DEFAULT '',
    session_step   INT         NOT NULL DEFAULT 0,
    refusal_count  INT         NOT NULL DEFAULT 0,
    items_viewed   INT         NOT NULL DEFAULT 0,
    items_clicked  INT         NOT NULL DEFAULT 0,
    demand_score   NUMERIC(5,4) NOT NULL DEFAULT 0,
    urgency_score  NUMERIC(5,4) NOT NULL DEFAULT 0,
    last_action_at TIMESTAMPTZ,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Embedding Vectors (for similarity search)
CREATE TABLE IF NOT EXISTS embedding_vectors (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type  VARCHAR(20) NOT NULL,
    entity_id    UUID        NOT NULL,
    vector       TEXT        NOT NULL,  -- JSON array of float32 (upgrade to pgvector later)
    version      INT         NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(entity_type, entity_id)
);

-- Embedding Events (real-time update log)
CREATE TABLE IF NOT EXISTS embedding_events (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type   VARCHAR(20) NOT NULL,
    entity_id     UUID        NOT NULL,
    event_type    VARCHAR(30) NOT NULL,
    delta_json    TEXT,
    trust_weight  NUMERIC(5,4) NOT NULL DEFAULT 1.0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Pipeline Latency Logs (performance monitoring)
CREATE TABLE IF NOT EXISTS pipeline_latency_logs (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID        NOT NULL,
    feature_latency_ms  BIGINT      NOT NULL DEFAULT 0,
    embedding_latency_ms BIGINT     NOT NULL DEFAULT 0,
    retrieval_latency_ms BIGINT     NOT NULL DEFAULT 0,
    total_latency_ms    BIGINT      NOT NULL DEFAULT 0,
    candidate_count     INT         NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Feature store indexes
CREATE INDEX IF NOT EXISTS idx_feature_users_segment ON feature_users(segment);
CREATE INDEX IF NOT EXISTS idx_feature_items_category ON feature_items(category_path);
CREATE INDEX IF NOT EXISTS idx_feature_items_trending ON feature_items(is_trending);
CREATE INDEX IF NOT EXISTS idx_feature_items_popularity ON feature_items(popularity_score DESC);
CREATE INDEX IF NOT EXISTS idx_feature_sessions_user ON feature_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_embedding_vectors_type ON embedding_vectors(entity_type);
CREATE INDEX IF NOT EXISTS idx_embedding_vectors_entity ON embedding_vectors(entity_id);
CREATE INDEX IF NOT EXISTS idx_embedding_events_entity ON embedding_events(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_embedding_events_type ON embedding_events(event_type);
CREATE INDEX IF NOT EXISTS idx_pipeline_latency_user ON pipeline_latency_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_latency_created ON pipeline_latency_logs(created_at);
