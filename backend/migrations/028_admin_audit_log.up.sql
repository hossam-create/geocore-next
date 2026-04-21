-- 028: Admin Audit Log — tracks every admin action for compliance and rollback
-- Table is referenced by logAudit() in backend/internal/admin/settings/handler.go

CREATE TABLE IF NOT EXISTS admin_logs (
    id            BIGSERIAL PRIMARY KEY,
    admin_id      UUID          REFERENCES users(id) ON DELETE SET NULL,
    action        VARCHAR(100)  NOT NULL,
    target_type   VARCHAR(100)  NOT NULL,
    target_id     VARCHAR(255)  NOT NULL,
    details       JSONB         DEFAULT '{}',
    ip_address    VARCHAR(45),
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_logs_admin_id    ON admin_logs (admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_logs_action      ON admin_logs (action);
CREATE INDEX IF NOT EXISTS idx_admin_logs_created_at  ON admin_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_logs_target      ON admin_logs (target_type, target_id);
