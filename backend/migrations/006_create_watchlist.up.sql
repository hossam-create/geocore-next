-- GeoCore Next - Watchlist table

CREATE TABLE watchlist_items (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    listing_id  UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, listing_id)
);

CREATE INDEX idx_watchlist_user_created ON watchlist_items(user_id, created_at DESC);
CREATE INDEX idx_watchlist_listing ON watchlist_items(listing_id);
