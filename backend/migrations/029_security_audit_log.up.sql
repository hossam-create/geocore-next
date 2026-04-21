-- 029: Security Audit Log — tracks auth events, payment attempts, suspicious activity
-- Separate from admin_logs which tracks admin dashboard actions

CREATE TABLE IF NOT EXISTS security_audit_log (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     UUID         REFERENCES users(id) ON DELETE SET NULL,
    event_type  VARCHAR(50)  NOT NULL,
    -- login_success | login_failed | password_change | password_reset_request |
    -- password_reset_complete | account_created | account_deleted | social_login |
    -- payment_attempt | kyc_submitted | session_revoked | rate_limited
    ip_address  INET         NOT NULL,
    user_agent  TEXT,
    details     JSONB        DEFAULT '{}',
    risk_score  INTEGER      DEFAULT 0 CHECK (risk_score BETWEEN 0 AND 100),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sal_user       ON security_audit_log (user_id);
CREATE INDEX IF NOT EXISTS idx_sal_event      ON security_audit_log (event_type);
CREATE INDEX IF NOT EXISTS idx_sal_ip         ON security_audit_log (ip_address);
CREATE INDEX IF NOT EXISTS idx_sal_created    ON security_audit_log (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sal_risk       ON security_audit_log (risk_score) WHERE risk_score > 50;
