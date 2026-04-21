DROP TABLE IF EXISTS negotiation_messages;
DROP TABLE IF EXISTS negotiation_threads;
ALTER TABLE listings DROP COLUMN IF EXISTS listing_type;
ALTER TABLE listings DROP COLUMN IF EXISTS trade_config;
ALTER TABLE listings DROP COLUMN IF EXISTS price_cents;
