-- P2P Currency Exchange

CREATE TYPE exchange_request_status AS ENUM ('open', 'matched', 'escrow', 'completed', 'cancelled', 'disputed');

CREATE TABLE IF NOT EXISTS exchange_requests (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_currency    VARCHAR(10) NOT NULL,
    to_currency      VARCHAR(10) NOT NULL,
    from_amount      NUMERIC(14,2) NOT NULL,
    to_amount        NUMERIC(14,2) NOT NULL,
    desired_rate     NUMERIC(12,6) NOT NULL,
    use_escrow       BOOLEAN NOT NULL DEFAULT false,
    notes            TEXT,
    status           exchange_request_status NOT NULL DEFAULT 'open',
    matched_user_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    matched_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exchange_requests_user       ON exchange_requests(user_id);
CREATE INDEX idx_exchange_requests_status     ON exchange_requests(status);
CREATE INDEX idx_exchange_requests_currencies ON exchange_requests(from_currency, to_currency);

CREATE TABLE IF NOT EXISTS exchange_messages (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id       UUID NOT NULL REFERENCES exchange_requests(id) ON DELETE CASCADE,
    sender_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body             TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exchange_messages_request ON exchange_messages(request_id, created_at);
