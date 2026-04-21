-- Sprint 9: Live Auction Items + Bids (eBay Live style)

CREATE TABLE IF NOT EXISTS live_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES livestream_sessions(id),
    listing_id UUID REFERENCES listings(id),
    title VARCHAR(255) NOT NULL,
    image_url TEXT,
    start_price_cents BIGINT NOT NULL DEFAULT 0,
    buy_now_price_cents BIGINT,
    current_bid_cents BIGINT NOT NULL DEFAULT 0,
    min_increment_cents BIGINT NOT NULL DEFAULT 100,
    highest_bidder_id UUID,
    bid_count INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, active, sold, unsold, cancelled
    ends_at TIMESTAMPTZ,
    anti_snipe_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_live_items_session ON live_items(session_id);
CREATE INDEX idx_live_items_status ON live_items(status);

CREATE TABLE IF NOT EXISTS live_bids (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id UUID NOT NULL REFERENCES live_items(id),
    user_id UUID NOT NULL,
    bid_amount_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_live_bids_item ON live_bids(item_id);
CREATE INDEX idx_live_bids_user ON live_bids(user_id);
