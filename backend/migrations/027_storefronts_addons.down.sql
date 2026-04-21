-- 027 down: Revert storefronts admin column + addons table
ALTER TABLE storefronts DROP COLUMN IF EXISTS is_featured;
DROP TABLE IF EXISTS addons;
