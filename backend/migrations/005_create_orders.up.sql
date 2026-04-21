-- GeoCore Next - Orders Schema
-- Tracks purchases from fixed-price listings and auction wins

CREATE TYPE order_status AS ENUM (
    'pending',        -- Order created, awaiting seller confirmation
    'confirmed',      -- Seller confirmed, preparing for shipment
    'processing',     -- Order being processed
    'shipped',        -- Order shipped, in transit
    'delivered',      -- Delivered to buyer, awaiting confirmation
    'completed',      -- Buyer confirmed delivery, escrow released
    'cancelled',      -- Order cancelled (before shipment)
    'disputed',       -- Dispute opened
    'refunded'        -- Refunded to buyer
);

-- Orders table
CREATE TABLE orders (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    buyer_id            UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    seller_id           UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    
    -- Payment tracking
    payment_intent_id   VARCHAR(255),           -- Stripe payment intent ID
    payment_id          UUID,                   -- Internal payment record reference
    
    -- Order status
    status              order_status NOT NULL DEFAULT 'pending',
    status_history      JSONB DEFAULT '[]',     -- Array of {status, at, by, note}
    
    -- Pricing
    subtotal            DECIMAL(15,2) NOT NULL, -- Sum of line items
    platform_fee        DECIMAL(15,2) NOT NULL DEFAULT 0, -- Platform commission
    payment_fee         DECIMAL(15,2) NOT NULL DEFAULT 0, -- Stripe fees
    total               DECIMAL(15,2) NOT NULL, -- Final amount charged
    currency            VARCHAR(3) NOT NULL DEFAULT 'AED',
    
    -- Shipping
    shipping_address    JSONB,                  -- {name, line1, line2, city, country, phone}
    tracking_number     VARCHAR(100),
    carrier             VARCHAR(50),
    shipped_at          TIMESTAMPTZ,
    delivered_at        TIMESTAMPTZ,
    
    -- Metadata
    notes               TEXT,                   -- Buyer/seller notes
    dispute_reason      TEXT,                   -- If disputed
    dispute_evidence    TEXT,                   -- Dispute details
    
    -- Timestamps
    confirmed_at        TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    cancelled_at        TIMESTAMPTZ,
    cancelled_reason    TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_buyer ON orders(buyer_id, created_at DESC);
CREATE INDEX idx_orders_seller ON orders(seller_id, created_at DESC);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_payment_intent ON orders(payment_intent_id);

-- Order items (line items)
CREATE TABLE order_items (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id        UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Source tracking
    listing_id      UUID REFERENCES listings(id) ON DELETE SET NULL,
    auction_id      UUID REFERENCES auctions(id) ON DELETE SET NULL,
    
    -- Snapshot at time of purchase
    title           VARCHAR(200) NOT NULL,      -- Listing title snapshot
    quantity        INT NOT NULL DEFAULT 1,
    unit_price      DECIMAL(15,2) NOT NULL,    -- Price per unit at purchase
    total_price     DECIMAL(15,2) NOT NULL,    -- unit_price * quantity
    
    -- Optional attributes snapshot
    condition       VARCHAR(50),                -- Item condition snapshot
    attributes      JSONB DEFAULT '{}',         -- Listing attributes snapshot
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_listing ON order_items(listing_id);
CREATE INDEX idx_order_items_auction ON order_items(auction_id);

-- Trigger to update orders.updated_at
CREATE OR REPLACE FUNCTION update_orders_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_orders_updated_at();
