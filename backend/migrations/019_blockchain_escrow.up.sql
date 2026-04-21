-- Blockchain Escrow & Smart Contracts

CREATE TYPE escrow_status AS ENUM ('pending', 'funded', 'released', 'refunded', 'disputed');

CREATE TABLE IF NOT EXISTS escrow_contracts (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id          UUID NOT NULL,
    buyer_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    seller_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount            NUMERIC(14,2) NOT NULL,
    currency          VARCHAR(10) NOT NULL DEFAULT 'AED',
    chain             VARCHAR(50) NOT NULL DEFAULT 'ethereum',
    contract_address  VARCHAR(100),
    tx_hash_fund      VARCHAR(100),
    tx_hash_release   VARCHAR(100),
    status            escrow_status NOT NULL DEFAULT 'pending',
    funded_at         TIMESTAMPTZ,
    released_at       TIMESTAMPTZ,
    expires_at        TIMESTAMPTZ,
    metadata          JSONB DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_escrow_contracts_order  ON escrow_contracts(order_id);
CREATE INDEX idx_escrow_contracts_buyer  ON escrow_contracts(buyer_id);
CREATE INDEX idx_escrow_contracts_seller ON escrow_contracts(seller_id);
CREATE INDEX idx_escrow_contracts_status ON escrow_contracts(status);
