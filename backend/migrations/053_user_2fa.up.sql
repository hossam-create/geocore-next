-- ════════════════════════════════════════════════════════════════════════════
-- 053: Two-Factor Authentication (TOTP) support
-- Creates user_2fa table for storing encrypted TOTP secrets and backup codes
-- ════════════════════════════════════════════════════════════════════════════

-- 1. Create user_2fa table
CREATE TABLE IF NOT EXISTS user_2fa (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    encrypted_secret TEXT       NOT NULL,                         -- XChaCha20-Poly1305 encrypted TOTP secret
    backup_codes_hashed TEXT,                                     -- JSON array of {hash, used} objects
    enabled         BOOLEAN     NOT NULL DEFAULT FALSE,
    verified        BOOLEAN     NOT NULL DEFAULT FALSE,           -- true after first successful TOTP verify
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Unique index — one 2FA config per user
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_2fa_user_id ON user_2fa (user_id);

-- 3. Index for fast enabled lookup (used on every login)
CREATE INDEX IF NOT EXISTS idx_user_2fa_enabled ON user_2fa (user_id, enabled) WHERE enabled = TRUE;

-- 4. Documentation
COMMENT ON TABLE user_2fa IS 'Stores TOTP 2FA configuration per user. Secrets are encrypted at rest with XChaCha20-Poly1305.';
COMMENT ON COLUMN user_2fa.encrypted_secret IS 'AES-256 encrypted TOTP base32 secret (decrypted at runtime only)';
COMMENT ON COLUMN user_2fa.backup_codes_hashed IS 'JSON array of bcrypt-hashed backup codes with usage tracking';
COMMENT ON COLUMN user_2fa.verified IS 'Must be true before 2FA enforcement kicks in — prevents lockout from incomplete setup';
