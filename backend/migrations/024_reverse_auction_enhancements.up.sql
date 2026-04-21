-- Reverse auction enhancements: counter-offers, expiration, responded_at

ALTER TABLE reverse_auction_offers
    ADD COLUMN IF NOT EXISTS counter_price NUMERIC(15,2),
    ADD COLUMN IF NOT EXISTS message TEXT,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS responded_at TIMESTAMPTZ;

-- Expand status check to include 'countered'
ALTER TABLE reverse_auction_offers DROP CONSTRAINT IF EXISTS reverse_auction_offers_status_check;
ALTER TABLE reverse_auction_offers ADD CONSTRAINT reverse_auction_offers_status_check
    CHECK (status IN ('pending','accepted','rejected','withdrawn','countered','expired'));
