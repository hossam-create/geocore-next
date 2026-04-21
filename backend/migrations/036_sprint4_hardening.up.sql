-- Sprint 4 Hardening: Auto-accept settings + boost constraints

-- Auto-accept user settings (opt-in, per-user)
CREATE TABLE IF NOT EXISTS user_auto_accept_settings (
    user_id UUID PRIMARY KEY,
    auto_accept_enabled BOOLEAN NOT NULL DEFAULT false,
    max_auto_accept_amount_cents BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique active boost per listing (partial unique index)
CREATE UNIQUE INDEX IF NOT EXISTS idx_listing_boosts_unique_active
    ON listing_boosts (listing_id) WHERE expires_at > now();
