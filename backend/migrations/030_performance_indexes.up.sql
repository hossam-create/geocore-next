-- 030: Performance indexes for high-traffic queries
-- All CONCURRENTLY to avoid table locks in production

-- Listings search (most used queries)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_listings_status_cat
    ON listings(status, category_id)
    WHERE deleted_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_listings_price_active
    ON listings(price)
    WHERE status = 'active' AND deleted_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_listings_created_active
    ON listings(created_at DESC)
    WHERE status = 'active';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_listings_seller
    ON listings(seller_id, status)
    WHERE deleted_at IS NULL;

-- Users (login & lookup)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email
    ON users(email);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_created
    ON users(created_at DESC);

-- Auctions (active auctions queried frequently)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_auctions_active
    ON auctions(end_time)
    WHERE status = 'active';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_auctions_seller
    ON auctions(seller_id, status);

-- Orders
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_buyer
    ON orders(buyer_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_seller
    ON orders(seller_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_status
    ON orders(status);

-- Wallet transactions
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_wallet_tx_wallet
    ON wallet_transactions(wallet_id, created_at DESC);
