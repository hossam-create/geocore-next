-- Listing views tracking table

CREATE TABLE IF NOT EXISTS listing_views (
    id          BIGSERIAL PRIMARY KEY,
    listing_id  UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    viewer_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_hash     VARCHAR(64),
    user_agent  VARCHAR(500),
    viewed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lv_listing ON listing_views(listing_id);
CREATE INDEX idx_lv_date    ON listing_views(viewed_at);

-- Materialized view for daily aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS listing_daily_views AS
SELECT
    listing_id,
    DATE(viewed_at) AS view_date,
    COUNT(*) AS total_views,
    COUNT(DISTINCT COALESCE(viewer_id::text, ip_hash)) AS unique_views
FROM listing_views
GROUP BY listing_id, DATE(viewed_at);

CREATE INDEX idx_ldv_listing ON listing_daily_views(listing_id);
