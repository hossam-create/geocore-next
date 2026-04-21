-- 048: Layer 0 Upgrade — align settings engine with eBay-level admin blueprint.
--
-- Adds:
--   1. is_secret column on admin_settings
--   2. category column on feature_flags
--   3. trust_flags table
--   4. old_value / new_value / user_agent columns on admin_audit_log

-- ── 1. admin_settings: add is_secret ──────────────────────────────────────────
ALTER TABLE admin_settings
    ADD COLUMN IF NOT EXISTS is_secret BOOLEAN DEFAULT FALSE;

-- Mark known secret keys
UPDATE admin_settings SET is_secret = TRUE
WHERE key IN (
    'payments.stripe_sk', 'payments.stripe_webhook_secret',
    'payments.paymob_api_key', 'payments.paypal_client_secret',
    'payments.tabby_secret_key', 'payments.tamara_token', 'payments.tamara_notification_key',
    'email.resend_api_key', 'email.sendgrid_api_key', 'email.smtp_password',
    'oauth.google_client_secret', 'oauth.apple_private_key', 'oauth.facebook_app_secret',
    'storage.r2_access_key', 'storage.r2_secret_key',
    'storage.s3_access_key', 'storage.s3_secret_key',
    'sms.twilio_auth_token', 'sms.vonage_api_secret',
    'push.fcm_server_key', 'push.apns_private_key',
    'shipping.dhl_api_key',
    'aws.s3_access_key', 'aws.s3_secret_key'
);

-- ── 2. feature_flags: add category ────────────────────────────────────────────
ALTER TABLE feature_flags
    ADD COLUMN IF NOT EXISTS category VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_feature_flags_category ON feature_flags(category);

-- Backfill categories for existing seed flags
UPDATE feature_flags SET category = 'commerce'  WHERE key IN ('feature.dutch_auction','feature.reverse_auction','feature.storefronts','feature.wallet');
UPDATE feature_flags SET category = 'growth'    WHERE key IN ('feature.loyalty_program','feature.referral_program','feature.deals_promotions','feature.subscription_plans');
UPDATE feature_flags SET category = 'auctions'  WHERE key IN ('feature.live_streaming');
UPDATE feature_flags SET category = 'payments'  WHERE key IN ('feature.paypal','feature.crypto_payments','feature.bnpl');
UPDATE feature_flags SET category = 'future'    WHERE key IN ('feature.ai_chatbot','feature.ai_fraud_detection','feature.ai_pricing','feature.ar_preview','feature.crowdshipping','feature.blockchain','feature.plugin_marketplace','feature.p2p_exchange');

-- ── 3. trust_flags table ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS trust_flags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type VARCHAR(50)  NOT NULL,
    target_id   UUID         NOT NULL,
    flag_type   VARCHAR(100) NOT NULL,
    severity    VARCHAR(20)  NOT NULL,
    source      VARCHAR(50)  NOT NULL,
    status      VARCHAR(50)  DEFAULT 'open',
    notes       TEXT,
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_trust_flags_target     ON trust_flags(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_trust_flags_status     ON trust_flags(status);
CREATE INDEX IF NOT EXISTS idx_trust_flags_severity   ON trust_flags(severity);
CREATE INDEX IF NOT EXISTS idx_trust_flags_created_at ON trust_flags(created_at DESC);

-- ── 4. admin_audit_log: add separate old_value/new_value + user_agent ─────────
ALTER TABLE admin_audit_log
    ADD COLUMN IF NOT EXISTS old_value  JSONB,
    ADD COLUMN IF NOT EXISTS new_value  JSONB,
    ADD COLUMN IF NOT EXISTS user_agent TEXT;
