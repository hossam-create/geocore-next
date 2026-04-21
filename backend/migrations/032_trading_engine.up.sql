-- Trading Engine: listing modes, negotiation threads, negotiation messages
-- Extends listings with listing_type, trade_config, price_cents

-- ── Listing columns ──────────────────────────────────────────────────────────────
ALTER TABLE listings ADD COLUMN IF NOT EXISTS listing_type VARCHAR(20) NOT NULL DEFAULT 'buy_now';
ALTER TABLE listings ADD COLUMN IF NOT EXISTS trade_config JSONB NOT NULL DEFAULT '{}';
ALTER TABLE listings ADD COLUMN IF NOT EXISTS price_cents BIGINT NOT NULL DEFAULT 0;

-- Backfill price_cents from existing price column
UPDATE listings SET price_cents = ROUND(COALESCE(price, 0) * 100) WHERE price_cents = 0 AND price IS NOT NULL;

-- ── Negotiation threads ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS negotiation_threads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id UUID NOT NULL REFERENCES listings(id),
    buyer_id UUID NOT NULL,
    seller_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    agreed_cents BIGINT NOT NULL DEFAULT 0,
    agreed_price DECIMAL(15,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    payment_retry_allowed BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_negotiation_threads_listing_id ON negotiation_threads(listing_id);
CREATE INDEX IF NOT EXISTS idx_negotiation_threads_buyer_id ON negotiation_threads(buyer_id);
CREATE INDEX IF NOT EXISTS idx_negotiation_threads_seller_id ON negotiation_threads(seller_id);
CREATE INDEX IF NOT EXISTS idx_negotiation_threads_status ON negotiation_threads(status);
CREATE INDEX IF NOT EXISTS idx_negotiation_threads_deleted_at ON negotiation_threads(deleted_at);

-- ── Negotiation messages ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS negotiation_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    thread_id UUID NOT NULL REFERENCES negotiation_threads(id),
    sender_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL,
    price_cents BIGINT NOT NULL,
    price DECIMAL(15,2) NOT NULL,
    delivery_fee_cents BIGINT NOT NULL DEFAULT 0,
    delivery_fee DECIMAL(15,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    breakdown JSONB NOT NULL DEFAULT '{}',
    note TEXT,
    auto_decision BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_negotiation_messages_thread_id ON negotiation_messages(thread_id);
CREATE INDEX IF NOT EXISTS idx_negotiation_messages_sender_id ON negotiation_messages(sender_id);
