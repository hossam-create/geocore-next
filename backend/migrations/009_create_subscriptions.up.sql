-- GeoCore Next — Subscription / Plans System

CREATE TYPE subscription_status AS ENUM (
    'active',
    'cancelled',
    'past_due',
    'trialing',
    'incomplete',
    'unpaid'
);

-- Seed plans
CREATE TABLE plans (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                VARCHAR(50)  NOT NULL UNIQUE,   -- Free, Basic, Pro, Enterprise
    display_name        VARCHAR(100) NOT NULL,
    price_monthly       DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency            VARCHAR(3)   NOT NULL DEFAULT 'AED',
    stripe_price_id     VARCHAR(128),                   -- Stripe Price object ID
    listing_limit       INT NOT NULL DEFAULT 5,         -- max active listings (0 = unlimited)
    features            JSONB NOT NULL DEFAULT '[]',
    is_active           BOOLEAN NOT NULL DEFAULT true,
    sort_order          INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO plans (name, display_name, price_monthly, listing_limit, features, sort_order) VALUES
('free',       'Free',       0,    5,   '["5 active listings","Basic analytics","Standard support"]', 0),
('basic',      'Basic',      49,   25,  '["25 active listings","Priority listing","Email support","Promoted badge"]', 1),
('pro',        'Pro',        149,  100, '["100 active listings","Advanced analytics","Priority support","Featured placement","Bulk upload"]', 2),
('enterprise', 'Enterprise', 499,  0,   '["Unlimited listings","Dedicated account manager","API access","Custom integrations","SLA guarantee"]', 3);

-- User subscriptions
CREATE TABLE subscriptions (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id                 UUID NOT NULL REFERENCES plans(id),
    status                  subscription_status NOT NULL DEFAULT 'active',
    stripe_subscription_id  VARCHAR(128) UNIQUE,
    stripe_customer_id      VARCHAR(128),
    current_period_start    TIMESTAMPTZ,
    current_period_end      TIMESTAMPTZ,
    cancel_at_period_end    BOOLEAN NOT NULL DEFAULT false,
    cancelled_at            TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT subscriptions_unique_user UNIQUE (user_id)  -- one active sub per user
);

CREATE INDEX idx_subscriptions_user          ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_stripe_sub_id ON subscriptions(stripe_subscription_id);
CREATE INDEX idx_subscriptions_status        ON subscriptions(status);
