CREATE TYPE livestream_status AS ENUM ('scheduled', 'live', 'ended', 'cancelled');

CREATE TABLE IF NOT EXISTS livestream_sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    auction_id      UUID REFERENCES auctions(id) ON DELETE CASCADE,
    host_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    status          livestream_status NOT NULL DEFAULT 'scheduled',
    room_name       VARCHAR(255) NOT NULL UNIQUE,
    viewer_count    INTEGER NOT NULL DEFAULT 0,
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,
    thumbnail_url   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_livestream_sessions_host_id    ON livestream_sessions(host_id);
CREATE INDEX idx_livestream_sessions_auction_id ON livestream_sessions(auction_id);
CREATE INDEX idx_livestream_sessions_status     ON livestream_sessions(status);
