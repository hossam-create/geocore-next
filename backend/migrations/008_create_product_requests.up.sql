-- GeoCore Next — Product Request System
-- Buyers signal demand for products they can't find; sellers respond with matching listings.

CREATE TYPE request_status AS ENUM ('open', 'fulfilled', 'expired', 'cancelled');

CREATE TABLE product_requests (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(200) NOT NULL,
    description TEXT,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    budget      DECIMAL(15,2),
    currency    VARCHAR(3) NOT NULL DEFAULT 'AED',
    status      request_status NOT NULL DEFAULT 'open',
    fulfilled_by UUID REFERENCES listings(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ DEFAULT NOW() + INTERVAL '30 days'
);

CREATE INDEX idx_product_requests_user   ON product_requests(user_id);
CREATE INDEX idx_product_requests_status ON product_requests(status);
CREATE INDEX idx_product_requests_cat    ON product_requests(category_id);

-- Seller responses to product requests
CREATE TABLE product_request_responses (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id  UUID NOT NULL REFERENCES product_requests(id) ON DELETE CASCADE,
    seller_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    listing_id  UUID REFERENCES listings(id) ON DELETE SET NULL,
    message     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT request_responses_unique UNIQUE (request_id, seller_id)
);

CREATE INDEX idx_request_responses_request ON product_request_responses(request_id);

-- Trigger to update product_requests.updated_at
CREATE OR REPLACE FUNCTION update_product_requests_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_product_requests_updated_at
    BEFORE UPDATE ON product_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_product_requests_updated_at();
