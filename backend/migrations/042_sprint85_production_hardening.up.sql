-- Sprint 8.5: Production Hardening — Admin Controls, Audit Log

-- User Freezes
CREATE TABLE user_freezes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) UNIQUE,
    reason     TEXT NOT NULL,
    frozen_by  UUID NOT NULL REFERENCES users(id),
    is_frozen  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Wallet Adjustments (audit trail)
CREATE TABLE wallet_adjustments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    amount_cents    BIGINT NOT NULL,
    reason          TEXT NOT NULL,
    adjusted_by     UUID NOT NULL REFERENCES users(id),
    previous_balance DECIMAL(15,2) NOT NULL,
    new_balance      DECIMAL(15,2) NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_wa_user ON wallet_adjustments(user_id);

-- Transaction Overrides (high-risk admin actions)
CREATE TABLE transaction_overrides (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id  UUID NOT NULL,
    override_type   VARCHAR(50) NOT NULL,
    reason          TEXT NOT NULL,
    overridden_by   UUID NOT NULL REFERENCES users(id),
    previous_status VARCHAR(50) NOT NULL,
    new_status      VARCHAR(50) NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_to_transaction ON transaction_overrides(transaction_id);

-- Admin Audit Log
CREATE TABLE admin_audit_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action     VARCHAR(100) NOT NULL,
    actor_id   UUID NOT NULL REFERENCES users(id),
    target_id  UUID NOT NULL,
    details    TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_action ON admin_audit_log(action);
CREATE INDEX idx_audit_actor ON admin_audit_log(actor_id);
CREATE INDEX idx_audit_target ON admin_audit_log(target_id);
