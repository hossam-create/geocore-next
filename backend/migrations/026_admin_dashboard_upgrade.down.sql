-- 026 down: reverse admin dashboard upgrade
DROP TABLE IF EXISTS listing_extra_purchases;
DROP TABLE IF EXISTS listing_extras;
DROP TABLE IF EXISTS user_custom_fields;
DROP TABLE IF EXISTS discount_codes;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS payment_gateways;
DROP TABLE IF EXISTS email_templates;
DROP TABLE IF EXISTS static_pages;
DROP TABLE IF EXISTS announcements;
DROP TABLE IF EXISTS geo_regions;
ALTER TABLE users DROP COLUMN IF EXISTS group_id;
DROP TABLE IF EXISTS user_groups;
ALTER TABLE categories DROP COLUMN IF EXISTS settings;
