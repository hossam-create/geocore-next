-- Sprint 3: Liquidity & Auto-Match Engine

-- Add auto-offer tracking fields to traveler_offers
ALTER TABLE traveler_offers ADD COLUMN IF NOT EXISTS is_auto_generated BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE traveler_offers ADD COLUMN IF NOT EXISTS match_score NUMERIC(6,2) DEFAULT 0;

-- Broadcast log for cooldown tracking
CREATE TABLE IF NOT EXISTS broadcast_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    traveler_id UUID NOT NULL,
    request_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_broadcast_logs_traveler_request ON broadcast_logs(traveler_id, request_id);
CREATE INDEX IF NOT EXISTS idx_broadcast_logs_created_at ON broadcast_logs(created_at);

-- Add delivery locked status support (already in enum from Sprint 2 improvement)
