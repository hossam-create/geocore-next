-- Crowdshipping / Traveler Delivery System

CREATE TYPE trip_status AS ENUM ('active', 'matched', 'in_transit', 'completed', 'cancelled');
CREATE TYPE delivery_request_status AS ENUM ('pending', 'matched', 'accepted', 'picked_up', 'in_transit', 'delivered', 'cancelled', 'disputed');

-- Traveler trips
CREATE TABLE IF NOT EXISTS trips (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    traveler_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    origin_country    VARCHAR(100) NOT NULL,
    origin_city       VARCHAR(100) NOT NULL,
    origin_address    TEXT,
    dest_country      VARCHAR(100) NOT NULL,
    dest_city         VARCHAR(100) NOT NULL,
    dest_address      TEXT,
    departure_date    TIMESTAMPTZ NOT NULL,
    arrival_date      TIMESTAMPTZ NOT NULL,
    available_weight  NUMERIC(10,2) DEFAULT 0,
    max_items         INTEGER DEFAULT 5,
    price_per_kg      NUMERIC(10,2) DEFAULT 0,
    base_price        NUMERIC(10,2) DEFAULT 0,
    currency          VARCHAR(10) NOT NULL DEFAULT 'AED',
    notes             TEXT,
    frequency         VARCHAR(20) NOT NULL DEFAULT 'one-time',
    status            trip_status NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trips_traveler    ON trips(traveler_id);
CREATE INDEX idx_trips_status      ON trips(status);
CREATE INDEX idx_trips_route       ON trips(origin_country, dest_country);
CREATE INDEX idx_trips_departure   ON trips(departure_date);

-- Buyer delivery requests
CREATE TABLE IF NOT EXISTS delivery_requests (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    buyer_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trip_id           UUID REFERENCES trips(id) ON DELETE SET NULL,
    traveler_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    item_name         VARCHAR(255) NOT NULL,
    item_description  TEXT,
    item_url          TEXT,
    item_price        NUMERIC(12,2) NOT NULL DEFAULT 0,
    item_weight       NUMERIC(10,2),
    pickup_country    VARCHAR(100) NOT NULL,
    pickup_city       VARCHAR(100) NOT NULL,
    delivery_country  VARCHAR(100) NOT NULL,
    delivery_city     VARCHAR(100) NOT NULL,
    reward            NUMERIC(10,2) NOT NULL DEFAULT 0,
    currency          VARCHAR(10) NOT NULL DEFAULT 'AED',
    deadline          TIMESTAMPTZ,
    status            delivery_request_status NOT NULL DEFAULT 'pending',
    match_score       NUMERIC(6,2),
    proof_image_url   TEXT,
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_delivery_requests_buyer     ON delivery_requests(buyer_id);
CREATE INDEX idx_delivery_requests_traveler  ON delivery_requests(traveler_id);
CREATE INDEX idx_delivery_requests_trip      ON delivery_requests(trip_id);
CREATE INDEX idx_delivery_requests_status    ON delivery_requests(status);
CREATE INDEX idx_delivery_requests_route     ON delivery_requests(pickup_country, delivery_country);
