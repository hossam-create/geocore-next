-- Sprint 4: Monetization, Urgency, Watchlist, Deal Closer

-- Listing Boosts (paid visibility)
CREATE TABLE IF NOT EXISTS listing_boosts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    seller_id UUID NOT NULL,
    boost_type VARCHAR(20) NOT NULL CHECK (boost_type IN ('basic', 'premium')),
    boost_score INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_listing_boosts_listing ON listing_boosts(listing_id);
CREATE INDEX IF NOT EXISTS idx_listing_boosts_expires ON listing_boosts(expires_at);
CREATE INDEX IF NOT EXISTS idx_listing_boosts_seller ON listing_boosts(seller_id);

-- Watchlists (retention loop)
CREATE TABLE IF NOT EXISTS watchlists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, listing_id)
);

CREATE INDEX IF NOT EXISTS idx_watchlists_user ON watchlists(user_id);
CREATE INDEX IF NOT EXISTS idx_watchlists_listing ON watchlists(listing_id);
