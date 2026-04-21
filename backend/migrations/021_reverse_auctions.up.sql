-- Reverse Auctions: buyer posts request, sellers make offers

CREATE TABLE IF NOT EXISTS reverse_auction_requests (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    buyer_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    category_id     UUID REFERENCES categories(id) ON DELETE SET NULL,
    max_budget      NUMERIC(15,2),
    deadline        TIMESTAMPTZ NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed','fulfilled','expired')),
    images          JSONB DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS reverse_auction_offers (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id      UUID NOT NULL REFERENCES reverse_auction_requests(id) ON DELETE CASCADE,
    seller_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    price           NUMERIC(15,2) NOT NULL,
    description     TEXT,
    delivery_days   INTEGER,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','rejected','withdrawn')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(request_id, seller_id)
);

CREATE INDEX idx_rar_status ON reverse_auction_requests(status);
CREATE INDEX idx_rar_buyer  ON reverse_auction_requests(buyer_id);
CREATE INDEX idx_rao_request ON reverse_auction_offers(request_id);
CREATE INDEX idx_rao_seller  ON reverse_auction_offers(seller_id);
