-- Down: rename back to admin_logs so older deployments that still expect that
-- name keep working. The schema remains the richer one (no column drops).
ALTER TABLE IF EXISTS admin_audit_log RENAME TO admin_logs;
ALTER INDEX IF EXISTS idx_admin_audit_log_admin_id   RENAME TO idx_admin_logs_admin_id;
ALTER INDEX IF EXISTS idx_admin_audit_log_action     RENAME TO idx_admin_logs_action;
ALTER INDEX IF EXISTS idx_admin_audit_log_created_at RENAME TO idx_admin_logs_created_at;
ALTER INDEX IF EXISTS idx_admin_audit_log_target     RENAME TO idx_admin_logs_target;
