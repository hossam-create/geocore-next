DROP TABLE IF EXISTS broadcast_logs;
ALTER TABLE traveler_offers DROP COLUMN IF EXISTS is_auto_generated;
ALTER TABLE traveler_offers DROP COLUMN IF EXISTS match_score;
