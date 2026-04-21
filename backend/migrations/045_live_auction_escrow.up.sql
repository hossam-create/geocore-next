-- Sprint 9.5: Production-grade Live Auction — Escrow binding + state machine

-- Add extension_count for anti-sniping cap
ALTER TABLE live_items ADD COLUMN IF NOT EXISTS extension_count INT NOT NULL DEFAULT 0;

-- Add settling + payment_failed statuses (via check constraint update)
-- PostgreSQL allows any string in varchar(20), statuses enforced in Go code.
