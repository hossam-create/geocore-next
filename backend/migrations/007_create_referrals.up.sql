-- GeoCore Next — Referral / Affiliate Program
-- Each user gets a unique referral code; referrals are tracked and rewarded
-- on the referee's first completed order.

-- Add referral_code column to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_code VARCHAR(16) UNIQUE;

-- Back-fill existing users with a code derived from their ID
UPDATE users SET referral_code = UPPER(SUBSTRING(REPLACE(id::text, '-', ''), 1, 8)) WHERE referral_code IS NULL;

-- Make it non-nullable after back-fill
ALTER TABLE users ALTER COLUMN referral_code SET NOT NULL;

-- Referrals table
CREATE TABLE referrals (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    referrer_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    referee_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code            VARCHAR(16) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending | completed | expired
    reward_points   INT NOT NULL DEFAULT 100,
    reward_paid_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT referrals_unique_referee UNIQUE (referee_id),
    CONSTRAINT referrals_self_check     CHECK  (referrer_id <> referee_id)
);

CREATE INDEX idx_referrals_referrer ON referrals(referrer_id);
CREATE INDEX idx_referrals_code     ON referrals(code);
CREATE INDEX idx_referrals_status   ON referrals(status);
