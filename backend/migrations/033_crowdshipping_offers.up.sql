-- Sprint 2: Crowdshipping Offer System + Tracking

CREATE TABLE IF NOT EXISTS traveler_offers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    delivery_request_id UUID NOT NULL REFERENCES delivery_requests(id),
    buyer_id UUID NOT NULL,
    traveler_id UUID NOT NULL,
    price_cents BIGINT NOT NULL,
    price DECIMAL(15,2) NOT NULL,
    delivery_fee_cents BIGINT NOT NULL DEFAULT 0,
    delivery_fee DECIMAL(15,2) NOT NULL DEFAULT 0,
    platform_fee_cents BIGINT NOT NULL DEFAULT 0,
    traveler_earnings_cents BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    payment_retry_allowed BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ NOT NULL,
    note TEXT,
    counter_to_id UUID REFERENCES traveler_offers(id),
    order_id UUID REFERENCES orders(id),
    idempotency_key VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_traveler_offers_delivery_request ON traveler_offers(delivery_request_id);
CREATE INDEX idx_traveler_offers_buyer ON traveler_offers(buyer_id);
CREATE INDEX idx_traveler_offers_traveler ON traveler_offers(traveler_id);
CREATE INDEX idx_traveler_offers_status ON traveler_offers(status);
CREATE INDEX idx_traveler_offers_counter_to ON traveler_offers(counter_to_id);
CREATE INDEX idx_traveler_offers_order ON traveler_offers(order_id);
CREATE INDEX idx_traveler_offers_idempotency ON traveler_offers(idempotency_key);
CREATE INDEX idx_traveler_offers_expires ON traveler_offers(expires_at) WHERE status IN ('pending', 'countered');

CREATE TABLE IF NOT EXISTS tracking_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id),
    traveler_id UUID NOT NULL,
    status VARCHAR(30) NOT NULL,
    location VARCHAR(255),
    note TEXT,
    proof_image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tracking_events_order ON tracking_events(order_id);
CREATE INDEX idx_tracking_events_traveler ON tracking_events(traveler_id);
CREATE INDEX idx_tracking_events_status ON tracking_events(status);
