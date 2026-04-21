-- 047: Unify admin audit log tables on canonical `admin_audit_log` name.
--
-- Prior state was inconsistent and **writes were silently failing**:
--   * Migration 028 created `admin_logs` with BIGSERIAL id.
--   * GORM struct `admin.AdminLog` expects `admin_logs` with UUID id.
--   * GORM struct `admin.AuditLogEntry` + `freeze.AuditLogEntry` expect
--     `admin_audit_log` with a different column set (actor_id vs admin_id).
--
-- This migration consolidates on a single canonical table:
--   Table name : `admin_audit_log`  (matches prompt spec)
--   PK         : UUID                (matches GORM struct in admin/model.go)
--   Columns    : superset covering all three prior shapes.

-- 1. Ensure uuid extension is available (no-op if already loaded).
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 2. Create the canonical table (fresh installs land here).
CREATE TABLE IF NOT EXISTS admin_audit_log (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_id    UUID         REFERENCES users(id) ON DELETE SET NULL,
    action      VARCHAR(100) NOT NULL,
    target_type VARCHAR(50)  NOT NULL DEFAULT '',
    target_id   VARCHAR(128) NOT NULL DEFAULT '',
    details     JSONB        DEFAULT '{}',
    ip_address  VARCHAR(45),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 3. Backfill missing columns on existing GORM-AutoMigrated tables.
ALTER TABLE admin_audit_log
    ADD COLUMN IF NOT EXISTS admin_id    UUID         REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS target_type VARCHAR(50)  DEFAULT '',
    ADD COLUMN IF NOT EXISTS target_id   VARCHAR(128) DEFAULT '',
    ADD COLUMN IF NOT EXISTS ip_address  VARCHAR(45),
    ADD COLUMN IF NOT EXISTS details     JSONB        DEFAULT '{}';

-- 4. If the legacy `admin_logs` table (from migration 028) exists, migrate
--    its rows into `admin_audit_log` and drop the legacy table. BIGSERIAL
--    ids are discarded; new UUIDs are generated on insert.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_tables
        WHERE schemaname = current_schema() AND tablename = 'admin_logs'
    ) THEN
        INSERT INTO admin_audit_log (admin_id, action, target_type, target_id, details, ip_address, created_at)
        SELECT admin_id, action, target_type, target_id, details, ip_address, created_at
        FROM admin_logs
        ON CONFLICT DO NOTHING;

        DROP TABLE admin_logs;
    END IF;
END $$;

-- 5. Indexes (idempotent).
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_admin_id   ON admin_audit_log (admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_action     ON admin_audit_log (action);
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_created_at ON admin_audit_log (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_target     ON admin_audit_log (target_type, target_id);
