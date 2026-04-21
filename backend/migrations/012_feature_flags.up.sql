-- GeoCore Next — Feature Flags

CREATE TABLE feature_flags (
    key            VARCHAR(255) PRIMARY KEY,
    enabled        BOOLEAN DEFAULT FALSE,
    rollout_pct    INTEGER DEFAULT 100 CHECK (rollout_pct BETWEEN 0 AND 100),
    allowed_groups TEXT[],
    description    TEXT,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);
