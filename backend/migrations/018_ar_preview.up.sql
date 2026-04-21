-- AR/VR 3D Model Preview for Listings

CREATE TABLE IF NOT EXISTS listing_3d_models (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id      UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    model_url       TEXT NOT NULL,
    poster_url      TEXT,
    format          VARCHAR(20) NOT NULL DEFAULT 'glb',
    file_size_bytes BIGINT DEFAULT 0,
    is_primary      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_listing_3d_models_listing ON listing_3d_models(listing_id);
