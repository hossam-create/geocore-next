-- Sprint 9.6: Live Auction Hardening
-- Settlement timeout recovery + retry tracking

ALTER TABLE live_items ADD COLUMN IF NOT EXISTS settling_started_at TIMESTAMPTZ;
ALTER TABLE live_items ADD COLUMN IF NOT EXISTS settle_retries INT NOT NULL DEFAULT 0;
