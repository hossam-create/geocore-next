-- Sprint 8: Growth Engine — Liquidity Bootstrap, Referrals, Funnels, Retention

-- Ghost Listings (admin-controlled supply seeding)
CREATE TABLE ghost_listings (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title                VARCHAR(255) NOT NULL,
    description          TEXT,
    price                DECIMAL(12,2) NOT NULL,
    currency             VARCHAR(10) NOT NULL DEFAULT 'USD',
    category             VARCHAR(100),
    origin_country       VARCHAR(100) NOT NULL,
    dest_country         VARCHAR(100) NOT NULL,
    is_platform_assisted BOOLEAN NOT NULL DEFAULT TRUE,
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    created_by           UUID REFERENCES users(id),
    created_at           TIMESTAMPTZ DEFAULT NOW()
);

-- Platform Travelers (internal high-rep travelers for early matching)
CREATE TABLE platform_travelers (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) UNIQUE,
    name           VARCHAR(255) NOT NULL,
    reputation     INTEGER NOT NULL DEFAULT 80,
    origin_country VARCHAR(100) NOT NULL,
    dest_country   VARCHAR(100) NOT NULL,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    is_internal    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

-- Referrals
CREATE TABLE referrals (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_id   UUID NOT NULL REFERENCES users(id),
    referee_id    UUID NOT NULL REFERENCES users(id) UNIQUE,
    code          VARCHAR(16) NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    reward_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_referrals_referrer ON referrals(referrer_id);
CREATE INDEX idx_referrals_code ON referrals(code);

-- Traveler Invites
CREATE TABLE traveler_invites (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inviter_id     UUID NOT NULL REFERENCES users(id),
    invitee_email  VARCHAR(255) NOT NULL,
    code           VARCHAR(16) NOT NULL UNIQUE,
    status         VARCHAR(20) NOT NULL DEFAULT 'sent',
    registered_id  UUID REFERENCES users(id),
    reward_claimed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ti_inviter ON traveler_invites(inviter_id);

-- Funnel Events
CREATE TABLE funnel_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id),
    funnel     VARCHAR(30) NOT NULL,
    step       VARCHAR(50) NOT NULL,
    step_order INTEGER NOT NULL,
    metadata   TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_funnel_user ON funnel_events(user_id, funnel);
CREATE INDEX idx_funnel_type ON funnel_events(funnel, step);

-- Stale Listings (price drop suggestions)
CREATE TABLE stale_listings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id      UUID NOT NULL UNIQUE,
    original_price  DECIMAL(12,2) NOT NULL,
    suggested_price DECIMAL(12,2) NOT NULL,
    discount_pct   DECIMAL(5,2) NOT NULL,
    days_stale      INTEGER NOT NULL,
    notified_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Retention Events (notification throttle tracking)
CREATE TABLE retention_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id),
    type       VARCHAR(30) NOT NULL,
    entity_id  UUID,
    sent_at    TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_retention_user ON retention_events(user_id, sent_at);
