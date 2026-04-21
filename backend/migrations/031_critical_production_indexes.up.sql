-- Production-critical indexes for high-traffic paths
-- All CONCURRENTLY to avoid table locks

-- Orders: payment_intent_id lookup (webhook path)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_payment_intent
    ON orders(payment_intent_id)
    WHERE payment_intent_id IS NOT NULL;

-- Orders: status + created_at for dashboard filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_status_created
    ON orders(status, created_at DESC);

-- Payments: buyer_id for buyer payment history
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_payments_buyer
    ON payments(buyer_id, created_at DESC);

-- Payments: seller_id for seller payment history
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_payments_seller
    ON payments(seller_id, created_at DESC);

-- Payments: status for admin dashboard filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_payments_status
    ON payments(status)
    WHERE status IN ('pending', 'succeeded', 'failed');

-- Wallet: user_id lookup (most common query)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_wallets_user_id
    ON wallets(user_id);

-- Wallet balances: wallet_id + currency (GetBalance path)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_wallet_balances_wallet_currency
    ON wallet_balances(wallet_id, currency);

-- Wallet transactions: user-facing transaction list with pagination
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_wallet_tx_type_status
    ON wallet_transactions(type, status);

-- Escrow accounts: order_id lookup (webhook → escrow creation)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_escrow_order_id
    ON escrow_accounts(order_id)
    WHERE order_id IS NOT NULL;

-- Escrow accounts: seller_id for seller escrow view
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_escrow_seller
    ON escrow_accounts(seller_id, status);

-- Disputes: order_id lookup
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_disputes_order
    ON disputes(order_id);

-- Disputes: status for admin queue
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_disputes_status
    ON disputes(status, created_at DESC);

-- Users: role for admin user management
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_role
    ON users(role)
    WHERE role IN ('admin', 'super_admin', 'seller');

-- Reviews: listing_id for listing detail page
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reviews_listing
    ON reviews(listing_id, created_at DESC);

-- Idempotent requests: user_id + key lookup (wallet idempotency)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_idempotent_user_key
    ON idempotent_requests(user_id, idempotency_key)
    WHERE expires_at > NOW();
