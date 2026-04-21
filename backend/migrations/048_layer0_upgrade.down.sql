-- 048: Rollback Layer 0 Upgrade
ALTER TABLE admin_settings DROP COLUMN IF EXISTS is_secret;
ALTER TABLE feature_flags DROP COLUMN IF EXISTS category;
DROP TABLE IF EXISTS trust_flags;
ALTER TABLE admin_audit_log DROP COLUMN IF EXISTS old_value;
ALTER TABLE admin_audit_log DROP COLUMN IF EXISTS new_value;
ALTER TABLE admin_audit_log DROP COLUMN IF EXISTS user_agent;
