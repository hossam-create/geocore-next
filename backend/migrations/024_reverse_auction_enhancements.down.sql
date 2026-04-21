ALTER TABLE reverse_auction_offers
    DROP COLUMN IF EXISTS counter_price,
    DROP COLUMN IF EXISTS message,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS responded_at;

ALTER TABLE reverse_auction_offers DROP CONSTRAINT IF EXISTS reverse_auction_offers_status_check;
ALTER TABLE reverse_auction_offers ADD CONSTRAINT reverse_auction_offers_status_check
    CHECK (status IN ('pending','accepted','rejected','withdrawn'));
